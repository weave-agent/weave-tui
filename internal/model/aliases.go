package model

import (
	"github.com/weave-agent/weave/sdk"

	tuicommands "github.com/weave-agent/weave-tui/internal/commands"
	"github.com/weave-agent/weave-tui/internal/contract"
	tuievents "github.com/weave-agent/weave-tui/internal/events"
	tuikeybindings "github.com/weave-agent/weave-tui/internal/keybindings"
	tuisessions "github.com/weave-agent/weave-tui/internal/sessions"
	tuiui "github.com/weave-agent/weave-tui/internal/ui"

	tea "charm.land/bubbletea/v2"
)

type TUIConfig = contract.TUIConfig
type PanelConfig = contract.PanelConfig
type PanelDrawer = contract.PanelDrawer
type TUIComponent = contract.TUIComponent
type KeyEvent = contract.KeyEvent
type AutocompleteProvider = contract.AutocompleteProvider
type AutocompleteContext = contract.AutocompleteContext
type AutocompleteSuggestion = contract.AutocompleteSuggestion
type ThemeDef = contract.ThemeDef
type TUIImpl = tuiui.TUIImpl
type CommandResult = tuicommands.CommandResult
type CommandRegistry = tuicommands.CommandRegistry
type Binding = tuikeybindings.Binding
type BindingAction = tuikeybindings.BindingAction
type BindingRegistry = tuikeybindings.BindingRegistry

const (
	AsOverlay   = contract.AsOverlay
	AboveEditor = contract.AboveEditor
	BelowEditor = contract.BelowEditor
	TrayOnly    = contract.TrayOnly
)

const (
	ActionExit               = tuikeybindings.ActionExit
	ActionInterrupt          = tuikeybindings.ActionInterrupt
	ActionModelSelect        = tuikeybindings.ActionModelSelect
	ActionModelCycle         = tuikeybindings.ActionModelCycle
	ActionCursorLineStart    = tuikeybindings.ActionCursorLineStart
	ActionCursorLineEnd      = tuikeybindings.ActionCursorLineEnd
	ActionCursorWordLeft     = tuikeybindings.ActionCursorWordLeft
	ActionCursorWordRight    = tuikeybindings.ActionCursorWordRight
	ActionEditorNewline      = tuikeybindings.ActionEditorNewline
	ActionScrollUp           = tuikeybindings.ActionScrollUp
	ActionScrollDown         = tuikeybindings.ActionScrollDown
	ActionScrollToBottom     = tuikeybindings.ActionScrollToBottom
	ActionDeleteWordBackward = tuikeybindings.ActionDeleteWordBackward
	ActionDeleteWordForward  = tuikeybindings.ActionDeleteWordForward
	ActionDeleteToLineStart  = tuikeybindings.ActionDeleteToLineStart
	ActionDeleteToLineEnd    = tuikeybindings.ActionDeleteToLineEnd
	ActionSuspend            = tuikeybindings.ActionSuspend
	ActionExternalEditor     = tuikeybindings.ActionExternalEditor
	ActionToggleToolOutput   = tuikeybindings.ActionToggleToolOutput
	ActionThinkingCycle      = tuikeybindings.ActionThinkingCycle
	ActionNewSession         = tuikeybindings.ActionNewSession
	ActionSandboxCycle       = tuikeybindings.ActionSandboxCycle
	ActionPanelPicker        = tuikeybindings.ActionPanelPicker
	ActionCopySelection      = tuikeybindings.ActionCopySelection
)

type overlayRequestKind = tuievents.OverlayRequestKind
type overlayRequest = tuievents.OverlayRequest
type overlayResponse = tuievents.OverlayResponse

const (
	requestSelect      = tuievents.RequestSelect
	requestConfirm     = tuievents.RequestConfirm
	requestInput       = tuievents.RequestInput
	requestEditor      = tuievents.RequestEditor
	requestMultiSelect = tuievents.RequestMultiSelect
)

type popupPendingMsg = tuievents.PopupPendingMsg
type extStatusMsg = tuievents.ExtStatusMsg
type slashCommandsUpdatedMsg = tuievents.SlashCommandsUpdatedMsg
type themeChangedMsg = tuievents.ThemeChangedMsg
type panelChangedMsg = tuievents.PanelChangedMsg
type setEditorTextMsg = tuievents.SetEditorTextMsg
type pasteToEditorMsg = tuievents.PasteToEditorMsg
type editorTextRequestMsg = tuievents.EditorTextRequestMsg
type setFooterMsg = tuievents.SetFooterMsg
type setHeaderMsg = tuievents.SetHeaderMsg
type setWorkingFramesMsg = tuievents.SetWorkingFramesMsg

func NewTUIImpl(commands *tuicommands.CommandRegistry, bindings *tuikeybindings.BindingRegistry) *tuiui.TUIImpl {
	return tuiui.NewTUIImpl(commands, bindings)
}

func NewCommandRegistry(bus sdk.Bus, sessionDir string) *tuicommands.CommandRegistry {
	return tuicommands.NewCommandRegistry(bus, sessionDir, tuicommands.RuntimeCommands{
		ListSessions: tuisessions.ListCmd,
		Login:        loginCmd,
		Logout:       logoutCmd,
	})
}

func NewBindingRegistry() *tuikeybindings.BindingRegistry {
	return tuikeybindings.NewBindingRegistry()
}

func keyString(msg tea.KeyPressMsg) string {
	return tuikeybindings.KeyString(msg)
}

// richRendererAdapter adapts a RichToolRenderer to sdk.ToolRenderer.
type richRendererAdapter struct {
	renderer  contract.RichToolRenderer
	themeFunc func() sdk.ThemeInfo
}

func (a *richRendererAdapter) Render(content string, width int) string {
	return a.renderer.Render(content, a.themeFunc(), width)
}
