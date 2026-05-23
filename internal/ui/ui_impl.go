package ui

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/weave-agent/weave/sdk"

	tuibridge "github.com/weave-agent/weave-tui/internal/bridge"
	tuicommands "github.com/weave-agent/weave-tui/internal/commands"
	"github.com/weave-agent/weave-tui/internal/components/messages"
	tuievents "github.com/weave-agent/weave-tui/internal/events"
	tuikeybindings "github.com/weave-agent/weave-tui/internal/keybindings"
	"github.com/weave-agent/weave-tui/internal/palette"
	"github.com/weave-agent/weave-tui/internal/panels"
)

// pendingCommand holds a command registered before the registry was set.
type pendingCommand struct {
	name    string
	handler func(args string) error
}

// pendingStatus holds a status update registered before the program was running.
type pendingStatus = extStatusMsg

// TUIImpl implements sdk.UI and TUIExtAPI by delegating to the TUI's internal
// registries and overlay components.
type TUIImpl struct {
	program   tuibridge.Sender
	commands  *tuicommands.CommandRegistry
	bindings  *tuikeybindings.BindingRegistry
	renderers map[string]sdk.ToolRenderer

	mu              sync.Mutex
	popupQ          []*overlayRequest
	pending         []pendingCommand
	pendingStatuses []pendingStatus
	done            chan struct{}
	closeOnce       sync.Once

	themeRegistry map[string]*palette.Theme
	activeTheme   string

	panelManager *panels.PanelManager
	width        int
	height       int

	// Task 9: deferred implementation fields
	richRenderers         map[string]RichToolRenderer
	inputHandlers         []func(KeyEvent)
	autocompleteProviders []AutocompleteProvider
	workingFrames         []string
	workingInterval       time.Duration
}

// NewTUIImpl creates a UI implementation backed by the given registries.
// The program is set later via SetProgram once the tea.Program is running.
func NewTUIImpl(commands *tuicommands.CommandRegistry, bindings *tuikeybindings.BindingRegistry) *TUIImpl {
	return &TUIImpl{
		commands:  commands,
		bindings:  bindings,
		renderers: make(map[string]sdk.ToolRenderer),
		done:      make(chan struct{}),
		themeRegistry: map[string]*palette.Theme{
			"default": palette.DefaultTheme(),
		},
		activeTheme:   "default",
		panelManager:  panels.NewPanelManager(),
		richRenderers: make(map[string]RichToolRenderer),
	}
}

// SetProgram sets the Bubble Tea program for sending overlay requests.
func (u *TUIImpl) SetProgram(p tuibridge.Sender) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.program = p
}

// SetSize updates the cached terminal dimensions.
func (u *TUIImpl) SetSize(width, height int) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.width = width
	u.height = height
}

// PanelManager returns the panel manager shared with the Bubble Tea model.
func (u *TUIImpl) PanelManager() *panels.PanelManager {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.panelManager
}

// SetRegistries sets the command and binding registries under lock.
// Any commands registered before the registry was available are flushed.
func (u *TUIImpl) SetRegistries(commands *tuicommands.CommandRegistry, bindings *tuikeybindings.BindingRegistry) {
	u.mu.Lock()
	pending := u.pending
	u.pending = nil
	u.commands = commands
	u.bindings = bindings
	u.mu.Unlock()

	for _, pc := range pending {
		u.registerCommand(commands, pc.name, pc.handler)
	}
}

// Close signals that the TUI is shutting down, unblocking any pending overlay calls.
func (u *TUIImpl) Close() {
	u.closeOnce.Do(func() {
		close(u.done)
	})
}

// Select shows a selection overlay and blocks until the user picks an item or cancels.
func (u *TUIImpl) Select(title string, items []string, opts ...sdk.SelectOption) (int, error) {
	config := sdk.SelectConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	req := &overlayRequest{
		Kind:        requestSelect,
		Title:       title,
		Items:       items,
		KeepContent: config.KeepContent,
		Result:      make(chan overlayResponse, 1),
	}
	if err := u.enqueue(req); err != nil {
		return -1, err
	}

	select {
	case resp := <-req.Result:
		return resp.Index, resp.Err
	case <-u.done:
		return -1, errors.New("tui shutting down")
	}
}

// Confirm shows a yes/no dialog and blocks until the user responds.
func (u *TUIImpl) Confirm(message string, opts ...sdk.ConfirmOption) (bool, error) {
	config := sdk.ConfirmConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	req := &overlayRequest{
		Kind:        requestConfirm,
		Message:     message,
		KeepContent: config.KeepContent,
		Result:      make(chan overlayResponse, 1),
	}
	if err := u.enqueue(req); err != nil {
		return false, err
	}

	select {
	case resp := <-req.Result:
		return resp.Confirmed, resp.Err
	case <-u.done:
		return false, errors.New("tui shutting down")
	}
}

// Input shows a single-line input modal and blocks until the user submits or cancels.
func (u *TUIImpl) Input(prompt string, opts ...sdk.InputOption) (string, error) {
	config := sdk.InputConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	req := &overlayRequest{
		Kind:        requestInput,
		Message:     prompt,
		KeepContent: config.KeepContent,
		Mask:        config.Mask,
		Result:      make(chan overlayResponse, 1),
	}
	if err := u.enqueue(req); err != nil {
		return "", err
	}

	select {
	case resp := <-req.Result:
		return resp.Value, resp.Err
	case <-u.done:
		return "", errors.New("tui shutting down")
	}
}

// SetStatus updates the footer's extension status area.
// If the program is not yet set, the update is buffered and flushed
// when the event loop starts (via DrainStatuses).
func (u *TUIImpl) SetStatus(key, text string) {
	u.mu.Lock()

	p := u.program

	if p == nil {
		u.pendingStatuses = append(u.pendingStatuses, pendingStatus{Key: key, Text: text})
		u.mu.Unlock()

		return
	}
	u.mu.Unlock()

	p.Send(extStatusMsg{Key: key, Text: text})
}

// DrainStatuses returns and clears pending status updates buffered before
// the program was available. Called from Model.Init to flush initial statuses.
func (u *TUIImpl) DrainStatuses() []pendingStatus {
	u.mu.Lock()
	defer u.mu.Unlock()

	statuses := u.pendingStatuses
	u.pendingStatuses = nil

	return statuses
}

// Notify shows a temporary notification in the chat area.
func (u *TUIImpl) Notify(message string) {
	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(tuievents.NotifyMsg{Message: message})
	}
}

// RegisterCommand adds a command to the slash command registry.
// If the registry is not yet set, the command is buffered and applied
// when SetRegistries is called.
func (u *TUIImpl) RegisterCommand(name string, handler func(args string) error) {
	u.mu.Lock()

	if u.commands == nil {
		u.pending = append(u.pending, pendingCommand{name: name, handler: handler})
		u.mu.Unlock()

		return
	}

	commands := u.commands
	u.mu.Unlock()

	u.registerCommand(commands, name, handler)
}

func (u *TUIImpl) registerCommand(commands *tuicommands.CommandRegistry, name string, handler func(args string) error) {
	displayName := strings.TrimPrefix(name, "/")

	commands.Register(name, "", false, func(args string) tuicommands.CommandResult {
		err := handler(args)
		if err != nil {
			return tuicommands.CommandResult{Notify: fmt.Sprintf("/%s: error: %v", displayName, err)}
		}

		if strings.HasPrefix(name, "/skill:") {
			return tuicommands.CommandResult{}
		}

		return tuicommands.CommandResult{Notify: "/" + displayName + ": ok"}
	})

	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(slashCommandsUpdatedMsg{})
	}
}

// RegisterRenderer stores a tool renderer for use by tool panels.
func (u *TUIImpl) RegisterRenderer(toolName string, renderer sdk.ToolRenderer) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.renderers[toolName] = renderer
}

// RegisterKeybinding delegates to the binding registry.
func (u *TUIImpl) RegisterKeybinding(kb sdk.Keybinding) {
	u.mu.Lock()
	bindings := u.bindings
	u.mu.Unlock()

	if bindings == nil {
		return
	}

	bindings.Register(tuikeybindings.BindingAction(kb.Name), kb.Keys, kb.Description)
}

// GetRenderer returns a registered tool renderer, if any.
func (u *TUIImpl) GetRenderer(toolName string) (sdk.ToolRenderer, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()

	r, ok := u.renderers[toolName]

	return r, ok
}

// MultiSelect shows a multi-selection overlay and blocks until the user responds.
func (u *TUIImpl) MultiSelect(title string, items []string, defaults []bool, opts ...sdk.SelectOption) ([]int, error) {
	config := sdk.SelectConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	req := &overlayRequest{
		Kind:        requestMultiSelect,
		Title:       title,
		Items:       items,
		Defaults:    defaults,
		KeepContent: config.KeepContent,
		Result:      make(chan overlayResponse, 1),
	}
	if err := u.enqueue(req); err != nil {
		return nil, err
	}

	select {
	case resp := <-req.Result:
		return resp.Selected, resp.Err
	case <-u.done:
		return nil, errors.New("tui shutting down")
	}
}

// Editor shows an editor overlay and blocks until the user responds.
func (u *TUIImpl) Editor(prompt, initial string, opts ...sdk.EditorOption) (string, error) {
	config := sdk.EditorConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	req := &overlayRequest{
		Kind:        requestEditor,
		Title:       prompt,
		Initial:     initial,
		KeepContent: config.KeepContent,
		Result:      make(chan overlayResponse, 1),
	}
	if err := u.enqueue(req); err != nil {
		return "", err
	}

	select {
	case resp := <-req.Result:
		return resp.Value, resp.Err
	case <-u.done:
		return "", errors.New("tui shutting down")
	}
}

// NotifyTyped shows a typed notification in the chat area.
func (u *TUIImpl) NotifyTyped(message string, level sdk.NotifyLevel) {
	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(tuievents.NotifyTypedMsg{Message: message, Level: level})
	}
}

// ShowError shows an error notification in the chat area.
func (u *TUIImpl) ShowError(message string) {
	u.NotifyTyped(message, sdk.NotifyError)
}

// SetWorking sets a working indicator in the UI.
func (u *TUIImpl) SetWorking(message string) {
	u.SetStatus("working", message)
}

// ClearWorking clears the working indicator.
func (u *TUIImpl) ClearWorking() {
	u.SetStatus("working", "")
}

// SetTheme sets the active UI theme.
func (u *TUIImpl) SetTheme(name string) error {
	u.mu.Lock()

	t, ok := u.themeRegistry[name]
	if !ok {
		u.mu.Unlock()
		return fmt.Errorf("unknown theme: %s", name)
	}

	u.activeTheme = name
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(themeChangedMsg{Theme: t})
	}

	return nil
}

// ListThemes returns available theme names.
func (u *TUIImpl) ListThemes() []string {
	u.mu.Lock()
	defer u.mu.Unlock()

	names := make([]string, 0, len(u.themeRegistry))
	for name := range u.themeRegistry {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// RegisterTheme implements TUIExtAPI.
func (u *TUIImpl) RegisterTheme(name string, theme ThemeDef) error {
	if name == "" {
		return errors.New("theme name cannot be empty")
	}

	if name == "default" {
		return errors.New("cannot override default theme")
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	u.themeRegistry[name] = toPaletteTheme(theme)

	return nil
}

// RegisterPaletteTheme registers or replaces an internal palette theme.
func (u *TUIImpl) RegisterPaletteTheme(name string, theme *palette.Theme) error {
	if name == "" {
		return errors.New("theme name cannot be empty")
	}

	if theme == nil {
		return errors.New("theme cannot be nil")
	}

	t := *theme

	u.mu.Lock()
	defer u.mu.Unlock()

	u.themeRegistry[name] = &t

	return nil
}

// Theme implements TUIExtAPI.
func (u *TUIImpl) Theme() sdk.ThemeInfo {
	u.mu.Lock()
	t := u.themeRegistry[u.activeTheme]
	name := u.activeTheme

	if t == nil {
		t = palette.DefaultTheme()
		name = "default"
	}

	info := sdk.ThemeInfo{
		Name:                  name,
		Accent:                t.Accent,
		AccentDim:             t.AccentDim,
		AccentBright:          t.AccentBright,
		Success:               t.Success,
		Error:                 t.Error,
		Warning:               t.Warning,
		Muted:                 t.Muted,
		MutedBright:           t.MutedBright,
		Border:                t.Border,
		BorderFocused:         t.BorderFocused,
		Foreground:            t.Foreground,
		ForegroundDim:         t.ForegroundDim,
		ForegroundBright:      t.ForegroundBright,
		Background:            t.Background,
		BackgroundTint:        t.BackgroundTint,
		BackgroundTint2:       t.BackgroundTint2,
		BackgroundTintPending: t.BackgroundTintPending,
		BackgroundTintSuccess: t.BackgroundTintSuccess,
		BackgroundTintError:   t.BackgroundTintError,
	}
	u.mu.Unlock()

	return info
}

// --- TUIExtAPI: Panels ---

// ShowPanel registers and shows a panel.
func (u *TUIImpl) ShowPanel(config PanelConfig, drawer PanelDrawer) {
	u.mu.Lock()
	if !u.panelManager.Register(config, drawer) {
		u.mu.Unlock()
		return
	}

	u.panelManager.Show(config.ID)
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(panelChangedMsg{})
	}
}

// HidePanel hides a panel.
func (u *TUIImpl) HidePanel(id string) {
	u.mu.Lock()
	u.panelManager.Hide(id)
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(panelChangedMsg{})
	}
}

// RemovePanel fully removes a panel.
func (u *TUIImpl) RemovePanel(id string) {
	u.mu.Lock()
	u.panelManager.Remove(id)
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(panelChangedMsg{})
	}
}

// PanelVisible returns whether a panel is currently visible.
func (u *TUIImpl) PanelVisible(id string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.panelManager.PanelVisible(id)
}

// RequestRedraw sends a message to the Bubble Tea program to trigger a
// redraw of the TUI. Safe to call when the program is not yet running.
func (u *TUIImpl) RequestRedraw() {
	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(panelChangedMsg{})
	}
}

// PanelTray returns the panel tray API.
func (u *TUIImpl) PanelTray() PanelTrayAPI {
	return u
}

// SetOrder implements PanelTrayAPI.
func (u *TUIImpl) SetOrder(ids []string) {
	u.mu.Lock()
	if u.panelManager == nil {
		u.mu.Unlock()
		return
	}

	u.panelManager.SetOrder(ids)
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(panelChangedMsg{})
	}
}

// GetOrder implements PanelTrayAPI.
func (u *TUIImpl) GetOrder() []string {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.panelManager == nil {
		return nil
	}

	return u.panelManager.GetOrder()
}

// --- TUIExtAPI: Read-only ---

// Size returns the terminal dimensions.
func (u *TUIImpl) Size() (int, int) {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.width, u.height
}

// --- TUIExtAPI: Editor ---

// EditorText returns the current editor content.
func (u *TUIImpl) EditorText() string {
	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p == nil {
		return ""
	}

	resp := make(chan string, 1)
	p.Send(editorTextRequestMsg{Response: resp})

	select {
	case text := <-resp:
		return text
	case <-u.done:
		return ""
	case <-time.After(5 * time.Second):
		return ""
	}
}

// SetEditorText replaces the editor content.
func (u *TUIImpl) SetEditorText(text string) {
	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(setEditorTextMsg{Text: text})
	}
}

// PasteToEditor inserts text at the cursor position.
func (u *TUIImpl) PasteToEditor(text string) {
	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(pasteToEditorMsg{Text: text})
	}
}

// --- TUIExtAPI: Rendering ---

// RegisterRichRenderer registers a theme-aware tool renderer.
func (u *TUIImpl) RegisterRichRenderer(tool string, renderer RichToolRenderer) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.richRenderers[tool] = renderer
}

// RegisterMessageRenderer registers a custom renderer for a message type.
func (u *TUIImpl) RegisterMessageRenderer(msgType string, renderer sdk.MessageRenderer) {
	messages.SetMessageRenderer(msgType, renderer)
}

// GetMessageRenderer returns a registered message renderer, if any.
func (u *TUIImpl) GetMessageRenderer(msgType string) (sdk.MessageRenderer, bool) {
	return messages.GetMessageRenderer(msgType)
}

// GetRichRenderer returns a registered rich tool renderer, if any.
func (u *TUIImpl) GetRichRenderer(toolName string) (RichToolRenderer, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()

	r, ok := u.richRenderers[toolName]

	return r, ok
}

// --- TUIExtAPI: Footer/Header ---

// SetFooter replaces the footer component.
func (u *TUIImpl) SetFooter(component TUIComponent) {
	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(setFooterMsg{Component: component})
	}
}

// SetHeader replaces the header component.
func (u *TUIImpl) SetHeader(component TUIComponent) {
	u.mu.Lock()
	p := u.program
	u.mu.Unlock()

	if p != nil {
		p.Send(setHeaderMsg{Component: component})
	}
}

// --- TUIExtAPI: Input (stubs for Task 9) ---

// OnTerminalInput registers a raw key event handler.
func (u *TUIImpl) OnTerminalInput(handler func(KeyEvent)) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.inputHandlers = append(u.inputHandlers, handler)
}

// AddAutocomplete registers an autocomplete provider.
func (u *TUIImpl) AddAutocomplete(provider AutocompleteProvider) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.autocompleteProviders = append(u.autocompleteProviders, provider)
}

// --- TUIExtAPI: Cosmetic ---

// SetWorkingFrames sets custom spinner animation frames.
func (u *TUIImpl) SetWorkingFrames(frames []string, interval time.Duration) {
	u.mu.Lock()
	p := u.program
	u.workingFrames = make([]string, len(frames))
	copy(u.workingFrames, frames)
	u.workingInterval = interval
	workingFrames := u.workingFrames
	u.mu.Unlock()

	if p != nil {
		p.Send(setWorkingFramesMsg{Frames: workingFrames, Interval: interval})
	}
}

// WorkingFrames returns the configured spinner frames and interval.
func (u *TUIImpl) WorkingFrames() ([]string, time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()

	frames := make([]string, len(u.workingFrames))
	copy(frames, u.workingFrames)

	return frames, u.workingInterval
}

// InputHandlers returns a snapshot of registered raw key handlers.
func (u *TUIImpl) InputHandlers() []func(KeyEvent) {
	u.mu.Lock()
	defer u.mu.Unlock()

	handlers := make([]func(KeyEvent), len(u.inputHandlers))
	copy(handlers, u.inputHandlers)

	return handlers
}

// AutocompleteProviders returns a snapshot of registered autocomplete providers.
func (u *TUIImpl) AutocompleteProviders() []AutocompleteProvider {
	u.mu.Lock()
	defer u.mu.Unlock()

	providers := make([]AutocompleteProvider, len(u.autocompleteProviders))
	copy(providers, u.autocompleteProviders)

	return providers
}

// enqueue adds a request to the popup queue and notifies the program.
// Returns an error if the program is not running.
func (u *TUIImpl) enqueue(req *overlayRequest) error {
	u.mu.Lock()

	if u.program == nil {
		u.mu.Unlock()
		return errors.New("tui not running")
	}

	select {
	case <-u.done:
		u.mu.Unlock()
		return errors.New("tui shutting down")
	default:
	}

	u.popupQ = append(u.popupQ, req)
	p := u.program
	u.mu.Unlock()

	p.Send(popupPendingMsg{})

	return nil
}

// EnqueuePopup adds a request to the popup queue and notifies the program.
func (u *TUIImpl) EnqueuePopup(req *overlayRequest) error {
	return u.enqueue(req)
}

// dequeue removes and returns the next popup request, or nil if empty.
func (u *TUIImpl) dequeue() *overlayRequest {
	u.mu.Lock()
	defer u.mu.Unlock()

	if len(u.popupQ) == 0 {
		return nil
	}

	req := u.popupQ[0]
	u.popupQ = u.popupQ[1:]

	return req
}

// DequeuePopup removes and returns the next popup request, or nil if empty.
func (u *TUIImpl) DequeuePopup() *overlayRequest {
	return u.dequeue()
}

// hasPendingPopups returns true if there are queued popup requests.
func (u *TUIImpl) hasPendingPopups() bool {
	u.mu.Lock()
	defer u.mu.Unlock()

	return len(u.popupQ) > 0
}

// HasPendingPopups returns true if there are queued popup requests.
func (u *TUIImpl) HasPendingPopups() bool {
	return u.hasPendingPopups()
}
