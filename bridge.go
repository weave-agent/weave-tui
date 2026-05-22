package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	tuievents "github.com/weave-agent/weave-tui/internal/events"
	"github.com/weave-agent/weave-tui/internal/palette"

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

// agentStateTracker tracks agent activity state from bus events.
// It lives in the Bridge goroutine and sends state changes to the program.
type agentStateTracker struct {
	state       palette.State
	toolCount   int                 // pending tool calls awaiting results
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

	if t.activeTools == nil {
		t.activeTools = make(map[string]struct{})
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
	t.activeTools = nil
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
	case tuievents.TurnStartMsg:
		t.state = palette.StateStreaming
		t.clearTools()
	case tuievents.MessageStartMsg:
		t.state = palette.StateStreaming
	case tuievents.ToolResultMsg:
		t.removeTool(msg.ToolID)
		t.maybeReturnToStreaming()
	case tuievents.MessageEndMsg:
		for _, tc := range msg.ToolCalls {
			t.addTool(tc.ID)
		}

		if t.toolCount > 0 {
			t.state = palette.StateToolRunning
		}
	case tuievents.ToolStartMsg:
		t.addTool(msg.ToolID)

		if t.toolCount > 0 {
			t.state = palette.StateToolRunning
		}
	case tuievents.ToolProgressMsg:
		t.addTool(msg.ToolID)

		if t.toolCount > 0 {
			t.state = palette.StateToolRunning
		}
	case tuievents.ToolCompleteMsg, tuievents.ToolErrorMsg, tuievents.ToolInterruptedMsg:
		var id string

		switch m := msg.(type) {
		case tuievents.ToolCompleteMsg:
			id = m.ToolID
		case tuievents.ToolErrorMsg:
			id = m.ToolID
		case tuievents.ToolInterruptedMsg:
			id = m.ToolID
		}

		t.removeTool(id)
		t.maybeReturnToStreaming()
	case tuievents.TurnEndMsg:
		t.state = palette.StateIdle
		t.clearTools()
	case tuievents.AgentEndMsg:
		if errStr, ok := msg.Payload.(string); ok && errStr != "" {
			t.state = palette.StateError
		} else {
			t.state = palette.StateIdle
		}

		t.clearTools()
	}

	return t.state, t.state != prev
}

// translateEvent converts a bus event into a tea.Msg.
// Returns nil for unknown topics.
func translateEvent(evt sdk.Event) tea.Msg {
	switch evt.Topic {
	case topicTurnStart:
		turn, _ := evt.Payload.(int)
		return tuievents.TurnStartMsg{Turn: turn}
	case topicTurnEnd:
		return tuievents.TurnEndMsg{}
	case topicMsgStart:
		return tuievents.MessageStartMsg{}
	case topicMsgUpdate:
		content, _ := evt.Payload.(string)
		return tuievents.MessageUpdateMsg{Content: content}
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
		return tuievents.AgentEndMsg{Payload: evt.Payload}
	case topicSessionResume:
		if p, ok := evt.Payload.(sdk.SessionResumePayload); ok {
			return tuievents.SessionResumedMsg{SessionID: p.SessionID, Messages: p.Messages}
		}

		return tuievents.SessionResumedMsg{}
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

func translateMsgEnd(payload any) tuievents.MessageEndMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.MessageEndMsg{}
	}

	content, _ := m["content"].(string)
	thinking, _ := m["thinking"].(string)

	var toolCalls []sdk.ToolCall

	if tc, ok := m["tool_calls"].([]sdk.ToolCall); ok {
		toolCalls = tc
	}

	return tuievents.MessageEndMsg{Content: content, Thinking: thinking, ToolCalls: toolCalls}
}

func translateToolResult(payload any) tuievents.ToolResultMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.ToolResultMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)

	result, ok := m["result"].(sdk.ToolResult)
	if !ok {
		result = sdk.ToolResult{}
	}

	return tuievents.ToolResultMsg{ToolID: id, Tool: tool, Result: result}
}

func translateToolStart(payload any) tuievents.ToolStartMsg {
	if tp, ok := payload.(sdk.ToolProgress); ok {
		return tuievents.ToolStartMsg{ToolID: tp.ToolCallID, Tool: tp.ToolName}
	}

	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.ToolStartMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)
	input, _ := m["input"].(string)

	return tuievents.ToolStartMsg{ToolID: id, Tool: tool, Input: input}
}

func translateToolProgress(payload any) tuievents.ToolProgressMsg {
	if tp, ok := payload.(sdk.ToolProgress); ok {
		return tuievents.ToolProgressMsg{ToolID: tp.ToolCallID, Tool: tp.ToolName, Content: tp.Content}
	}

	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.ToolProgressMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)
	content, _ := m["content"].(string)

	return tuievents.ToolProgressMsg{ToolID: id, Tool: tool, Content: content}
}

func translateToolComplete(payload any) tuievents.ToolCompleteMsg {
	if tp, ok := payload.(sdk.ToolProgress); ok {
		return tuievents.ToolCompleteMsg{ToolID: tp.ToolCallID, Tool: tp.ToolName, Content: tp.Content}
	}

	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.ToolCompleteMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)
	content, _ := m["content"].(string)

	return tuievents.ToolCompleteMsg{ToolID: id, Tool: tool, Content: content}
}

func translateToolError(payload any) tuievents.ToolErrorMsg {
	if tp, ok := payload.(sdk.ToolProgress); ok {
		return tuievents.ToolErrorMsg{ToolID: tp.ToolCallID, Tool: tp.ToolName, Error: tp.Content, IsError: tp.IsError}
	}

	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.ToolErrorMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)
	errStr, _ := m["error"].(string)
	isErr, _ := m["is_error"].(bool)

	return tuievents.ToolErrorMsg{ToolID: id, Tool: tool, Error: errStr, IsError: isErr}
}

func translateToolInterrupted(payload any) tuievents.ToolInterruptedMsg {
	if tp, ok := payload.(sdk.ToolProgress); ok {
		return tuievents.ToolInterruptedMsg{ToolID: tp.ToolCallID, Tool: tp.ToolName}
	}

	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.ToolInterruptedMsg{}
	}

	id, _ := m["id"].(string)
	tool, _ := m["tool"].(string)

	return tuievents.ToolInterruptedMsg{ToolID: id, Tool: tool}
}

func translateModelChangeFailed(payload any) tuievents.ModelChangeFailedMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.ModelChangeFailedMsg{}
	}

	provider, _ := m[keyProvider].(string)
	errStr, _ := m["error"].(string)

	return tuievents.ModelChangeFailedMsg{Provider: provider, Error: errStr}
}

func translateExtOutdated(payload any) tuievents.OutdatedNotificationMsg {
	evt, ok := payload.(sdk.OutdatedEvent)
	if !ok {
		return tuievents.OutdatedNotificationMsg{}
	}

	return tuievents.OutdatedNotificationMsg{Extensions: evt.Extensions}
}

func translateCompacted(payload any) tuievents.CompactedMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.CompactedMsg{}
	}

	if errStr, ok := m["error"].(string); ok {
		return tuievents.CompactedMsg{Error: errStr}
	}

	summarized, _ := m["summarized"].(int)
	tokensBefore, _ := m["tokens_before"].(int)
	tokensAfter, _ := m["tokens_after"].(int)

	return tuievents.CompactedMsg{
		Summarized:   summarized,
		TokensBefore: tokensBefore,
		TokensAfter:  tokensAfter,
	}
}

func translateUsage(payload any) tuievents.TokenUsageMsg {
	m, ok := payload.(map[string]any)
	if !ok {
		return tuievents.TokenUsageMsg{}
	}

	inputTokens, _ := m["input_tokens"].(int)
	outputTokens, _ := m["output_tokens"].(int)
	cacheCreationTokens, _ := m["cache_creation_tokens"].(int)
	cacheReadTokens, _ := m["cache_read_tokens"].(int)
	contextTokens, _ := m["context_tokens"].(int)

	return tuievents.TokenUsageMsg{
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		CacheCreationTokens: cacheCreationTokens,
		CacheReadTokens:     cacheReadTokens,
		ContextTokens:       contextTokens,
	}
}

// Bridge reads bus events and sends them as tea.Msg to the program.
// When multiple tuievents.MessageUpdateMsg deltas arrive in rapid succession, it batches
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
			sender.Send(tuievents.AgentStateChangeMsg{State: newState})
		}

		// Reset rate tracking at message boundaries
		switch msg.(type) {
		case tuievents.MessageStartMsg, tuievents.MessageEndMsg:
			streamStart = time.Time{}
			streamRunes = 0
		}

		// Batch consecutive tuievents.MessageUpdateMsg deltas.
		if mu, ok := msg.(tuievents.MessageUpdateMsg); ok { //nolint:nestif // batching requires nested select/drain
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
							sender.Send(tuievents.MessageUpdateMsg{
								Content:   batch.String(),
								TokenRate: calcTokenRate(streamStart, streamRunes),
							})
						}

						sender.Send(tuievents.ShutdownMsg{})

						return
					}

					nextMsg := translateEvent(next)
					if nextMu, ok := nextMsg.(tuievents.MessageUpdateMsg); ok {
						streamRunes += utf8.RuneCountInString(nextMu.Content)
						batch.WriteString(nextMu.Content)
					} else {
						// Non-delta message — flush the batch, then handle this message
						if batch.Len() > 0 {
							sender.Send(tuievents.MessageUpdateMsg{
								Content:   batch.String(),
								TokenRate: calcTokenRate(streamStart, streamRunes),
							})
							batch.Reset()
						}

						// Track state for non-delta messages found during drain
						if newState, changed := stateTracker.update(nextMsg); changed {
							sender.Send(tuievents.AgentStateChangeMsg{State: newState})
						}

						// Reset rate at message boundaries found during drain
						switch nextMsg.(type) {
						case tuievents.MessageStartMsg, tuievents.MessageEndMsg:
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
				sender.Send(tuievents.MessageUpdateMsg{
					Content:   batch.String(),
					TokenRate: calcTokenRate(streamStart, streamRunes),
				})
			}

			continue
		}

		sender.Send(msg)
	}

	sender.Send(tuievents.ShutdownMsg{})
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
func PublishModelChange(bus sdk.Bus, entry tuievents.ModelEntry) tea.Cmd {
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
		return tuievents.ModelListResultMsg{Models: listModels()}
	}
}

// listProvidersCmd returns a tea.Cmd that lists providers with key status.
func listProvidersCmd() tea.Cmd {
	return func() tea.Msg {
		return tuievents.ProviderListResultMsg{Providers: listProviders()}
	}
}

// loginCmd returns a tea.Cmd that lists providers available for login.
func loginCmd() tea.Cmd {
	return func() tea.Msg {
		return tuievents.LoginListResultMsg{Providers: buildLoginProviders()}
	}
}

// logoutCmd returns a tea.Cmd that lists providers with configured auth.
func logoutCmd() tea.Cmd {
	return func() tea.Msg {
		return tuievents.LogoutListResultMsg{Providers: buildLogoutProviders()}
	}
}
