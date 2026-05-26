package events

import (
	"fmt"
	"time"

	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	"github.com/weave-agent/weave-tui/internal/contract"
	"github.com/weave-agent/weave-tui/internal/palette"
)

// SessionEntry holds minimal session metadata for the selector.
type SessionEntry struct {
	ID        string
	CWD       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ModelEntry describes a provider + model combination.
type ModelEntry struct {
	Provider string
	Model    string
}

// Display returns a human-readable label for the model entry.
func (e ModelEntry) Display() string {
	return fmt.Sprintf("%s/%s", e.Provider, e.Model)
}

// DisplayName returns the human-friendly name from the model registry,
// falling back to provider/model format.
func (e ModelEntry) DisplayName() string {
	if def, ok := sdkmodel.GetModelForProvider(e.Model, e.Provider); ok && def.DisplayName != "" {
		return def.DisplayName
	}

	return e.Display()
}

// ProviderEntry describes a provider with its API key status.
type ProviderEntry struct {
	Name   string
	HasKey bool
}

// Display returns a human-readable label showing provider name and key status.
func (e ProviderEntry) Display() string {
	if e.HasKey {
		return e.Name + "  key set"
	}

	return e.Name + "  no key"
}

// LoginProviderEntry describes a provider available for login.
type LoginProviderEntry struct {
	Name    string
	ID      string
	IsOAuth bool
	HasAuth bool
}

// LogoutProviderEntry describes a provider with configured auth.
type LogoutProviderEntry struct {
	Name string
	ID   string
}

type TurnStartMsg struct {
	Turn int
}

type TurnEndMsg struct{}

type MessageStartMsg struct{}

type MessageUpdateMsg struct {
	Content   string
	TokenRate float64
}

type MessageEndMsg struct {
	Content   string
	Thinking  string
	ToolCalls []sdk.ToolCall
}

type ToolResultMsg struct {
	ToolID string
	Tool   string
	Result sdk.ToolResult
}

// ToolStartMsg is sent when a tool begins executing.
type ToolStartMsg struct {
	ToolID string
	Tool   string
	Input  string
}

// ToolProgressMsg carries a live update from a running tool.
type ToolProgressMsg struct {
	ToolID  string
	Tool    string
	Content string
}

// ToolCompleteMsg is sent when a tool finishes successfully.
type ToolCompleteMsg struct {
	ToolID  string
	Tool    string
	Content string
}

// ToolErrorMsg is sent when a tool fails.
type ToolErrorMsg struct {
	ToolID  string
	Tool    string
	Error   string
	IsError bool
}

// ToolInterruptedMsg is sent when a tool is interrupted.
type ToolInterruptedMsg struct {
	ToolID string
	Tool   string
}

type AgentEndMsg struct {
	Payload any
}

type ShutdownMsg struct{}

// SessionListResultMsg carries the result of listing sessions.
type SessionListResultMsg struct {
	Sessions []SessionEntry
	Err      error
}

// SessionResumedMsg is sent when a session resume event arrives from the bus.
type SessionResumedMsg struct {
	SessionID string
	Messages  []sdk.Message
}

// ModelListResultMsg carries the result of listing available models.
type ModelListResultMsg struct {
	Models []ModelEntry
}

// ModelChangedMsg is sent when the user selects or cycles to a new model.
type ModelChangedMsg struct {
	Entry ModelEntry
}

// ModelChangeFailedMsg is sent when the loop fails to switch providers.
type ModelChangeFailedMsg struct {
	Provider string
	Error    string
}

// ThinkingLevelSetMsg is sent when the user sets the thinking level via /thinking command.
type ThinkingLevelSetMsg struct {
	Level sdkmodel.ThinkingLevel
}

// OutdatedNotificationMsg is sent when outdated extensions are detected at startup.
type OutdatedNotificationMsg struct {
	Extensions []sdk.OutdatedInfo
}

// NotifyMsg shows an informational UI notification.
type NotifyMsg struct {
	Message string
}

// NotifyTypedMsg shows a typed UI notification.
type NotifyTypedMsg struct {
	Message string
	Level   sdk.NotifyLevel
}

type CompactingMsg struct{}

// CompactedMsg is sent when the agent compacts the conversation context.
type CompactedMsg struct {
	Summarized   int
	TokensBefore int
	TokensAfter  int
	Error        string
}

// TokenUsageMsg is sent when the provider reports token usage for a turn.
type TokenUsageMsg struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	ContextTokens       int
}

// AgentStateChangeMsg is sent when the agent activity state changes.
type AgentStateChangeMsg struct {
	State palette.State
}

// ProviderListResultMsg carries the result of listing providers with key status.
type ProviderListResultMsg struct {
	Providers []ProviderEntry
}

// LoginListResultMsg carries the result of listing providers available for login.
type LoginListResultMsg struct {
	Providers []LoginProviderEntry
}

// LogoutListResultMsg carries the result of listing providers with configured auth.
type LogoutListResultMsg struct {
	Providers []LogoutProviderEntry
}

// LoginFlowResultMsg is sent when an asynchronous OAuth login flow completes.
type LoginFlowResultMsg struct {
	Provider   string
	Credential sdk.OAuthCredential
	Error      error
	Gen        int
}

// LoginAuthURLMsg is sent when the authorization URL has been generated.
type LoginAuthURLMsg struct {
	Provider string
	URL      string
	Handle   *sdk.AuthorizationFlowHandle
	Gen      int
}

// OverlayRequestKind identifies the type of cross-extension popup request.
type OverlayRequestKind int

const (
	RequestSelect OverlayRequestKind = iota
	RequestConfirm
	RequestInput
	RequestEditor
	RequestMultiSelect
)

// OverlayRequest is sent from the SDK UI implementation to the Bubble Tea model
// to trigger a popup overlay.
type OverlayRequest struct {
	Kind        OverlayRequestKind
	Title       string
	Message     string
	Items       []string
	Initial     string
	Defaults    []bool
	KeepContent bool
	Mask        rune
	Result      chan OverlayResponse
}

// OverlayResponse carries the popup result back to the blocking caller.
type OverlayResponse struct {
	Index     int
	Value     string
	Confirmed bool
	Selected  []int
	Err       error
}

type PopupPendingMsg struct{}

type ExtStatusMsg struct {
	Key  string
	Text string
}

// SlashCommandsUpdatedMsg is sent when commands are dynamically registered,
// so the editor can refresh its autocomplete list.
type SlashCommandsUpdatedMsg struct{}

// ThemeChangedMsg is sent when the active theme is switched.
type ThemeChangedMsg struct {
	Theme *palette.Theme
}

type PanelChangedMsg struct{}

type SetEditorTextMsg struct {
	Text string
}

type PasteToEditorMsg struct {
	Text string
}

type EditorTextRequestMsg struct {
	Response chan string
}

type SetFooterMsg struct {
	Component contract.TUIComponent
}

type SetHeaderMsg struct {
	Component contract.TUIComponent
}

type SetWorkingFramesMsg struct {
	Frames   []string
	Interval time.Duration
}
