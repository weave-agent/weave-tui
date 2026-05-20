package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/weave-agent/weave-tui/palette"
	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	tea "charm.land/bubbletea/v2"
)

// Bus event topics (matching agent-loop topics).
const (
	topicPrompt    = "agent.prompt"
	topicSteer     = "agent.steer"
	topicFollowup  = "agent.followup"
	topicInterrupt = "agent.interrupt"

	topicTurnStart  = "agent.turn_start"
	topicTurnEnd    = "agent.turn_end"
	topicMsgStart   = "agent.message_start"
	topicMsgUpdate  = "agent.message_update"
	topicMsgEnd     = "agent.message_end"
	topicToolResult = "agent.tool_result"
	topicUsage      = "agent.usage"
	topicEnd        = "agent.end"

	topicSessionList       = "session.list"
	topicSessionResume     = "session.resume"
	topicModelChange       = "model.change"
	topicModelChangeFailed = "model.change_failed"
	topicThinkingChange    = "thinking.change"

	topicCompacted = "agent.compacted"

	topicExtOutdated = "extension.outdated"

	topicAuthLoginSuccess = "auth.login.success"
	topicAuthLogout       = "auth.logout"

	topicToolStart       = "tool.start"
	topicToolProgress    = "tool.progress"
	topicToolComplete    = "tool.complete"
	topicToolError       = "tool.error"
	topicToolInterrupted = "tool.interrupted"

	keyProvider = "provider"
	keyModel    = "model"
)

// Sender abstracts tea.Program.Send for testability.
type Sender interface {
	Send(msg tea.Msg)
}

// tea.Msg types for Bubble Tea.

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
	ToolID string
	Tool   string
	Error  string
}

// ToolInterruptedMsg is sent when a tool is interrupted (e.g., by ESC).
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
// The UI updates accent colors and editor pulse animation based on this.
type AgentStateChangeMsg struct {
	State palette.State
}

// agentStateTracker tracks agent activity state from bus events.
// It lives in the Bridge goroutine and sends state changes to the program.
type agentStateTracker struct {
	state      palette.State
	toolCount  int                 // pending tool calls awaiting results
	activeTools map[string]struct{} // tracks in-flight tool IDs to prevent double-count
}

func newAgentStateTracker() *agentStateTracker {
	return &agentStateTracker{
		state:       palette.StateIdle,
		activeTools: make(map[string]struct{}),
	}
}

// addTool adds a tool ID to the active set and updates the count.
func (t *agentStateTracker) addTool(id string) {
	if id == "" {
		return
	}
	if _, exists := t.activeTools[id]; !exists {
		t.activeTools[id] = struct{}{}
		t.toolCount = len(t.activeTools)
	}
}

// removeTool removes a tool ID from the active set and updates the count.
func (t *agentStateTracker) removeTool(id string) {
	if id == "" {
		return
	}
	if _, exists := t.activeTools[id]; exists {
		delete(t.activeTools, id)
		t.toolCount = len(t.activeTools)
	}
}

// clearTools removes all active tools and resets the count.
func (t *agentStateTracker) clearTools() {
	t.activeTools = make(map[string]struct{})
	t.toolCount = 0
}

// maybeReturnToStreaming transitions from ToolRunning to Streaming if no tools remain.
func (t *agentStateTracker) maybeReturnToStreaming() {
	if t.toolCount == 0 && t.state == palette.StateToolRunning {
		t.state = palette.StateStreaming
	}
}

// update computes the new state based on an incoming event message.
// Returns the new state and whether it changed.
func (t *agentStateTracker) update(msg tea.Msg) (palette.State, bool) {
	prev := t.state

	switch msg := msg.(type) {
	case TurnStartMsg:
		t.state = palette.StateStreaming
		t.clearTools()
	case MessageStartMsg:
		t.state = palette.StateStreaming
	case ToolResultMsg:
		t.removeTool(msg.ToolID)
		t.maybeReturnToStreaming()
	case MessageEndMsg:
		for _, tc := range msg.ToolCalls {
			t.addTool(tc.ID)
		}
		if t.toolCount > 0 {
			t.state = palette.StateToolRunning
		}
	case ToolStartMsg:
		t.addTool(msg.ToolID)
		if t.toolCount > 0 {
			t.state = palette.StateToolRunning
		}
	case ToolCompleteMsg, ToolErrorMsg, ToolInterruptedMsg:
		var id string
		switch m := msg.(type) {
		case ToolCompleteMsg:
			id = m.ToolID
		case ToolErrorMsg:
			id = m.ToolID
		case ToolInterruptedMsg:
			id = m.ToolID
		}
		t.removeTool(id)
		t.maybeReturnToStreaming()
	case TurnEndMsg:
		t.state = palette.StateIdle
		t.clearTools()
	case AgentEndMsg:
		if errStr, ok := msg.Payload.(string); ok && errStr != "" {
			t.state = palette.StateError
		} else {
			t.state = palette.StateIdle
		}

		t.clearTools()
	}

	return t.state, t.state != prev
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

// translateEvent converts a bus event into a tea.Msg.
// Returns nil for unknown topics.
func translateEvent(evt sdk.Event) tea.Msg {
	switch evt.Topic {
	case topicTurnStart:
		turn, _ := evt.Payload.(int)
		return TurnStartMsg{Turn: turn}
	case topicTurnEnd:
		return TurnEndMsg{}
	case topicMsgStart:
		return MessageStartMsg{}
	case topicMsgUpdate:
		content, _ := evt.Payload.(string)
		return MessageUpdateMsg{Content: content}
	case topicMsgEnd:
		return translateMsgEnd(evt.Payload)
	case topicToolResult:
		return translateToolResult(evt.Payload)
	case topicToolStart:
		return translateToolStart(evt.Payload)
	case topicToolProgress:
		return translateToolProgress(evt.Payload)
	case topicToolComplete:
		return translateToolComplete(evt.Payload)
	case topicToolError:
		return translateToolError(evt.Payload)
	case topicToolInterrupted:
		return translateToolInterrupted(evt.Payload)
	case topicEnd:
		return AgentEndMsg{Payload: evt.Payload}
	case topicSessionResume:
		if p, ok := evt.Payload.(sdk.SessionResumePayload); ok {
			return SessionResumedMsg{SessionID: p.SessionID, Messages: p.Messages}
		}

		return SessionResumedMsg{}
	case topicModelChangeFailed:
		return translateModelChangeFailed(evt.Payload)
	case topicExtOutdated:
		return translateExtOutdated(evt.Payload)
	case topicCompacted:
		return translateCompacted(evt.Payload)
	case topicUsage:
		return translateUsage(evt.Payload)
	default:
		return nil
	}
}

func translateMsgEnd(payload any) MessageEndMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return MessageEndMsg{}
	}

	content, _ := m["content"].(string)
	thinking, _ := m["thinking"].(string)

	var toolCalls []sdk.ToolCall

	if tc, ok := m["tool_calls"].([]sdk.ToolCall); ok {
		toolCalls = tc
	}

	return MessageEndMsg{Content: content, Thinking: thinking, ToolCalls: toolCalls}
}

func translateToolResult(payload any) ToolResultMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return ToolResultMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)

	result, ok := m["result"].(sdk.ToolResult)
	if !ok {
		result = sdk.ToolResult{}
	}

	return ToolResultMsg{ToolID: id, Tool: tool, Result: result}
}

func translateToolStart(payload any) ToolStartMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return ToolStartMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)
	input, _ := m["input"].(string)

	return ToolStartMsg{ToolID: id, Tool: tool, Input: input}
}

func translateToolProgress(payload any) ToolProgressMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return ToolProgressMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)
	content, _ := m["content"].(string)

	return ToolProgressMsg{ToolID: id, Tool: tool, Content: content}
}

func translateToolComplete(payload any) ToolCompleteMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return ToolCompleteMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)
	content, _ := m["content"].(string)

	return ToolCompleteMsg{ToolID: id, Tool: tool, Content: content}
}

func translateToolError(payload any) ToolErrorMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return ToolErrorMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)
	errStr, _ := m["error"].(string)

	return ToolErrorMsg{ToolID: id, Tool: tool, Error: errStr}
}

func translateToolInterrupted(payload any) ToolInterruptedMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return ToolInterruptedMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)

	return ToolInterruptedMsg{ToolID: id, Tool: tool}
}

func translateModelChangeFailed(payload any) ModelChangeFailedMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return ModelChangeFailedMsg{}
	}

	provider, _ := m[keyProvider].(string)
	errStr, _ := m["error"].(string)

	return ModelChangeFailedMsg{Provider: provider, Error: errStr}
}

func translateExtOutdated(payload any) OutdatedNotificationMsg {
	evt, ok := payload.(sdk.OutdatedEvent)
	if !ok {
		return OutdatedNotificationMsg{}
	}

	return OutdatedNotificationMsg{Extensions: evt.Extensions}
}

func translateCompacted(payload any) CompactedMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return CompactedMsg{}
	}

	if errStr, ok := m["error"].(string); ok {
		return CompactedMsg{Error: errStr}
	}

	summarized, _ := m["summarized"].(int)
	tokensBefore, _ := m["tokens_before"].(int)
	tokensAfter, _ := m["tokens_after"].(int)

	return CompactedMsg{
		Summarized:   summarized,
		TokensBefore: tokensBefore,
		TokensAfter:  tokensAfter,
	}
}

func translateUsage(payload any) TokenUsageMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return TokenUsageMsg{}
	}

	inputTokens, _ := m["input_tokens"].(int)
	outputTokens, _ := m["output_tokens"].(int)
	cacheCreationTokens, _ := m["cache_creation_tokens"].(int)
	cacheReadTokens, _ := m["cache_read_tokens"].(int)
	contextTokens, _ := m["context_tokens"].(int)

	return TokenUsageMsg{
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		CacheCreationTokens: cacheCreationTokens,
		CacheReadTokens:     cacheReadTokens,
		ContextTokens:       contextTokens,
	}
}

// Bridge reads bus events and sends them as tea.Msg to the program.
// When multiple MessageUpdateMsg deltas arrive in rapid succession, it batches
// them into a single concatenated message to reduce UI update pressure.
// Blocks until the event channel is closed.
func Bridge(sender Sender, events <-chan sdk.Event) {
	var (
		streamStart  time.Time
		streamRunes  int
		stateTracker = newAgentStateTracker()
	)

	for evt := range events {
		msg := translateEvent(evt)
		if msg == nil {
			continue
		}

		// Track agent state changes
		if newState, changed := stateTracker.update(msg); changed {
			sender.Send(AgentStateChangeMsg{State: newState})
		}

		// Reset rate tracking at message boundaries
		switch msg.(type) {
		case MessageStartMsg, MessageEndMsg:
			streamStart = time.Time{}
			streamRunes = 0
		}

		// Batch consecutive MessageUpdateMsg deltas
		if mu, ok := msg.(MessageUpdateMsg); ok { //nolint:nestif // batching requires nested select/drain
			// Track rate
			if streamStart.IsZero() {
				streamStart = time.Now()
			}

			streamRunes += utf8.RuneCountInString(mu.Content)

			var batch strings.Builder

			batch.WriteString(mu.Content)

			// Drain any queued deltas
			draining := true
			for draining {
				select {
				case next, ok := <-events:
					if !ok {
						// Channel closed while batching — flush and exit
						if batch.Len() > 0 {
							sender.Send(MessageUpdateMsg{
								Content:   batch.String(),
								TokenRate: calcTokenRate(streamStart, streamRunes),
							})
						}

						sender.Send(ShutdownMsg{})

						return
					}

					nextMsg := translateEvent(next)
					if nextMu, ok := nextMsg.(MessageUpdateMsg); ok {
						streamRunes += utf8.RuneCountInString(nextMu.Content)
						batch.WriteString(nextMu.Content)
					} else {
						// Non-delta message — flush the batch, then handle this message
						if batch.Len() > 0 {
							sender.Send(MessageUpdateMsg{
								Content:   batch.String(),
								TokenRate: calcTokenRate(streamStart, streamRunes),
							})
							batch.Reset()
						}

						// Track state for non-delta messages found during drain
						if newState, changed := stateTracker.update(nextMsg); changed {
							sender.Send(AgentStateChangeMsg{State: newState})
						}

						// Reset rate at message boundaries found during drain
						switch nextMsg.(type) {
						case MessageStartMsg, MessageEndMsg:
							streamStart = time.Time{}
							streamRunes = 0
						}

						if nextMsg != nil {
							sender.Send(nextMsg)
						}

						draining = false
					}
				default:
					draining = false
				}
			}

			if batch.Len() > 0 {
				sender.Send(MessageUpdateMsg{
					Content:   batch.String(),
					TokenRate: calcTokenRate(streamStart, streamRunes),
				})
			}

			continue
		}

		sender.Send(msg)
	}

	sender.Send(ShutdownMsg{})
}

// calcTokenRate estimates token rate from accumulated rune count and elapsed time.
// Uses the standard heuristic of 1 token ≈ 4 characters.
func calcTokenRate(start time.Time, totalRunes int) float64 {
	if start.IsZero() {
		return 0
	}

	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		return 0
	}

	return float64(totalRunes) / 4.0 / elapsed
}

// PublishPrompt returns a tea.Cmd that publishes an agent.prompt event.
func PublishPrompt(bus sdk.Bus, text string) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicPrompt, text))
		}

		return nil
	}
}

// PublishFollowup returns a tea.Cmd that publishes an agent.followup event.
func PublishFollowup(bus sdk.Bus, text string) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicFollowup, text))
		}

		return nil
	}
}

// PublishSteer returns a tea.Cmd that publishes an agent.steer event.
func PublishSteer(bus sdk.Bus, text string) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicSteer, text))
		}

		return nil
	}
}

// PublishInterrupt returns a tea.Cmd that publishes an agent.interrupt event.
func PublishInterrupt(bus sdk.Bus) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicInterrupt, "user interrupt"))
		}

		return nil
	}
}

// PublishSessionResume returns a tea.Cmd that publishes a session.resume event.
func PublishSessionResume(bus sdk.Bus, payload sdk.SessionResumePayload) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicSessionResume, payload))
		}

		return nil
	}
}

// PublishModelChange returns a tea.Cmd that publishes a model.change event.
func PublishModelChange(bus sdk.Bus, entry ModelEntry) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicModelChange, map[string]string{
				keyProvider: entry.Provider,
				keyModel:    entry.Model,
			}))
		}

		return nil
	}
}

// PublishThinkingChange returns a tea.Cmd that publishes a thinking.change event.
func PublishThinkingChange(bus sdk.Bus, level sdkmodel.ThinkingLevel) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicThinkingChange, map[string]string{
				"level": string(level),
			}))
		}

		return nil
	}
}

// PublishAuthLoginSuccess returns a tea.Cmd that publishes an auth.login.success event.
func PublishAuthLoginSuccess(bus sdk.Bus, provider string) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicAuthLoginSuccess, map[string]string{
				keyProvider: provider,
			}))
		}

		return nil
	}
}

// PublishAuthLogout returns a tea.Cmd that publishes an auth.logout event.
func PublishAuthLogout(bus sdk.Bus, provider string) tea.Cmd {
	return func() tea.Msg {
		if bus != nil {
			bus.Publish(sdk.NewEvent(topicAuthLogout, map[string]string{
				keyProvider: provider,
			}))
		}

		return nil
	}
}

// listModelsCmd returns a tea.Cmd that lists available models.
func listModelsCmd() tea.Cmd {
	return func() tea.Msg {
		return ModelListResultMsg{Models: listModels()}
	}
}

// listProvidersCmd returns a tea.Cmd that lists providers with key status.
func listProvidersCmd() tea.Cmd {
	return func() tea.Msg {
		return ProviderListResultMsg{Providers: listProviders()}
	}
}

// loginCmd returns a tea.Cmd that lists providers available for login.
func loginCmd() tea.Cmd {
	return func() tea.Msg {
		return LoginListResultMsg{Providers: buildLoginProviders()}
	}
}

// logoutCmd returns a tea.Cmd that lists providers with configured auth.
func logoutCmd() tea.Cmd {
	return func() tea.Msg {
		return LogoutListResultMsg{Providers: buildLogoutProviders()}
	}
}
