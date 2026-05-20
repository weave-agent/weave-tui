package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/weave-agent/weave-tui/palette"
	"github.com/weave-agent/weave/bus"
	"github.com/weave-agent/weave/sdk"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslateEvent_TurnStart(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicTurnStart, 3))
	ts, ok := msg.(TurnStartMsg)
	require.True(t, ok)
	assert.Equal(t, 3, ts.Turn)
}

func TestTranslateEvent_TurnEnd(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicTurnEnd, nil))
	_, ok := msg.(TurnEndMsg)
	require.True(t, ok)
}

func TestTranslateEvent_MessageStart(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicMsgStart, nil))
	_, ok := msg.(MessageStartMsg)
	require.True(t, ok)
}

func TestTranslateEvent_MessageUpdate(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicMsgUpdate, "hello "))
	mu, ok := msg.(MessageUpdateMsg)
	require.True(t, ok)
	assert.Equal(t, "hello ", mu.Content)
}

func TestTranslateEvent_MessageEnd(t *testing.T) {
	payload := map[string]any{
		"content":    "response text",
		"tool_calls": []sdk.ToolCall{{ID: "tc1", Name: "bash"}},
	}

	msg := translateEvent(sdk.NewEvent(topicMsgEnd, payload))
	me, ok := msg.(MessageEndMsg)
	require.True(t, ok)
	assert.Equal(t, "response text", me.Content)
	require.Len(t, me.ToolCalls, 1)
	assert.Equal(t, "tc1", me.ToolCalls[0].ID)
	assert.Equal(t, "bash", me.ToolCalls[0].Name)
}

func TestTranslateEvent_MessageEnd_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicMsgEnd, nil))
	me, ok := msg.(MessageEndMsg)
	require.True(t, ok)
	assert.Empty(t, me.Content)
	assert.Nil(t, me.ToolCalls)
}

func TestTranslateEvent_MessageEnd_WithThinking(t *testing.T) {
	payload := map[string]any{
		"content":    "response text",
		"thinking":   "I considered the alternatives...",
		"tool_calls": []sdk.ToolCall{},
	}

	msg := translateEvent(sdk.NewEvent(topicMsgEnd, payload))
	me, ok := msg.(MessageEndMsg)
	require.True(t, ok)
	assert.Equal(t, "response text", me.Content)
	assert.Equal(t, "I considered the alternatives...", me.Thinking)
}

func TestTranslateEvent_MessageEnd_WithoutThinking(t *testing.T) {
	payload := map[string]any{
		"content":    "response text",
		"tool_calls": []sdk.ToolCall{},
	}

	msg := translateEvent(sdk.NewEvent(topicMsgEnd, payload))
	me, ok := msg.(MessageEndMsg)
	require.True(t, ok)
	assert.Empty(t, me.Thinking)
}

func TestTranslateEvent_ToolResult(t *testing.T) {
	payload := map[string]any{
		"id":     "tc1",
		"tool":   "bash",
		"result": sdk.ToolResult{Content: "output", IsError: false},
	}

	msg := translateEvent(sdk.NewEvent(topicToolResult, payload))
	tr, ok := msg.(ToolResultMsg)
	require.True(t, ok)
	assert.Equal(t, "tc1", tr.ToolID)
	assert.Equal(t, "bash", tr.Tool)
	assert.Equal(t, "output", tr.Result.Content)
	assert.False(t, tr.Result.IsError)
}

func TestTranslateEvent_ToolResult_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicToolResult, nil))
	tr, ok := msg.(ToolResultMsg)
	require.True(t, ok)
	assert.Empty(t, tr.ToolID)
	assert.Empty(t, tr.Tool)
}

func TestTranslateEvent_ToolStart(t *testing.T) {
	payload := map[string]any{
		"id":    "tc1",
		"tool":  "bash",
		"input": "ls -la",
	}

	msg := translateEvent(sdk.NewEvent(topicToolStart, payload))
	ts, ok := msg.(ToolStartMsg)
	require.True(t, ok)
	assert.Equal(t, "tc1", ts.ToolID)
	assert.Equal(t, "bash", ts.Tool)
	assert.Equal(t, "ls -la", ts.Input)
}

func TestTranslateEvent_ToolStart_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicToolStart, nil))
	ts, ok := msg.(ToolStartMsg)
	require.True(t, ok)
	assert.Empty(t, ts.ToolID)
	assert.Empty(t, ts.Tool)
}

func TestTranslateEvent_ToolProgress(t *testing.T) {
	payload := map[string]any{
		"id":      "tc1",
		"tool":    "bash",
		"content": "line 1\nline 2",
	}

	msg := translateEvent(sdk.NewEvent(topicToolProgress, payload))
	tp, ok := msg.(ToolProgressMsg)
	require.True(t, ok)
	assert.Equal(t, "tc1", tp.ToolID)
	assert.Equal(t, "bash", tp.Tool)
	assert.Equal(t, "line 1\nline 2", tp.Content)
}

func TestTranslateEvent_ToolProgress_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicToolProgress, nil))
	tp, ok := msg.(ToolProgressMsg)
	require.True(t, ok)
	assert.Empty(t, tp.ToolID)
	assert.Empty(t, tp.Content)
}

func TestTranslateEvent_ToolProgress_NonMapPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicToolProgress, "not a map"))
	tp, ok := msg.(ToolProgressMsg)
	require.True(t, ok)
	assert.Empty(t, tp.ToolID)
	assert.Empty(t, tp.Content)
}

func TestTranslateEvent_ToolComplete(t *testing.T) {
	payload := map[string]any{
		"id":      "tc1",
		"tool":    "bash",
		"content": "output here",
	}

	msg := translateEvent(sdk.NewEvent(topicToolComplete, payload))
	tc, ok := msg.(ToolCompleteMsg)
	require.True(t, ok)
	assert.Equal(t, "tc1", tc.ToolID)
	assert.Equal(t, "bash", tc.Tool)
	assert.Equal(t, "output here", tc.Content)
}

func TestTranslateEvent_ToolComplete_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicToolComplete, nil))
	tc, ok := msg.(ToolCompleteMsg)
	require.True(t, ok)
	assert.Empty(t, tc.ToolID)
	assert.Empty(t, tc.Content)
}

func TestTranslateEvent_ToolError(t *testing.T) {
	payload := map[string]any{
		"id":    "tc1",
		"tool":  "bash",
		"error": "command not found",
	}

	msg := translateEvent(sdk.NewEvent(topicToolError, payload))
	te, ok := msg.(ToolErrorMsg)
	require.True(t, ok)
	assert.Equal(t, "tc1", te.ToolID)
	assert.Equal(t, "bash", te.Tool)
	assert.Equal(t, "command not found", te.Error)
}

func TestTranslateEvent_ToolError_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicToolError, nil))
	te, ok := msg.(ToolErrorMsg)
	require.True(t, ok)
	assert.Empty(t, te.ToolID)
	assert.Empty(t, te.Error)
}

func TestTranslateEvent_ToolInterrupted(t *testing.T) {
	payload := map[string]any{
		"id":   "tc1",
		"tool": "bash",
	}

	msg := translateEvent(sdk.NewEvent(topicToolInterrupted, payload))
	ti, ok := msg.(ToolInterruptedMsg)
	require.True(t, ok)
	assert.Equal(t, "tc1", ti.ToolID)
	assert.Equal(t, "bash", ti.Tool)
}

func TestTranslateEvent_ToolInterrupted_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicToolInterrupted, nil))
	ti, ok := msg.(ToolInterruptedMsg)
	require.True(t, ok)
	assert.Empty(t, ti.ToolID)
	assert.Empty(t, ti.Tool)
}

func TestTranslateEvent_AgentEnd(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicEnd, "stream error: timeout"))
	ae, ok := msg.(AgentEndMsg)
	require.True(t, ok)
	assert.Equal(t, "stream error: timeout", ae.Payload)
}

func TestTranslateEvent_AgentEnd_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicEnd, nil))
	ae, ok := msg.(AgentEndMsg)
	require.True(t, ok)
	assert.Nil(t, ae.Payload)
}

func TestTranslateEvent_UnknownTopic(t *testing.T) {
	msg := translateEvent(sdk.NewEvent("unknown.topic", "data"))
	assert.Nil(t, msg)
}

func TestTranslateEvent_SessionResume_StringPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicSessionResume, "sess-123"))
	sr, ok := msg.(SessionResumedMsg)
	require.True(t, ok)
	assert.Empty(t, sr.SessionID)
}

func TestTranslateEvent_SessionResume_PayloadStruct(t *testing.T) {
	payload := sdk.SessionResumePayload{SessionID: "sess-456", Messages: []sdk.Message{
		{Role: sdk.RoleUser, Content: "hello"},
	}}
	msg := translateEvent(sdk.NewEvent(topicSessionResume, payload))
	sr, ok := msg.(SessionResumedMsg)
	require.True(t, ok)
	assert.Equal(t, "sess-456", sr.SessionID)
	require.Len(t, sr.Messages, 1)
	assert.Equal(t, sdk.RoleUser, sr.Messages[0].Role)
	assert.Equal(t, "hello", sr.Messages[0].Content)
}

func TestTranslateEvent_SessionResume_UnknownPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicSessionResume, 42))
	sr, ok := msg.(SessionResumedMsg)
	require.True(t, ok)
	assert.Empty(t, sr.SessionID)
}

func TestTranslateEvent_MessageUpdate_NonStringPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicMsgUpdate, 42))
	mu, ok := msg.(MessageUpdateMsg)
	require.True(t, ok)
	assert.Empty(t, mu.Content)
}

func TestTranslateEvent_TurnStart_NonIntPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicTurnStart, "not an int"))
	ts, ok := msg.(TurnStartMsg)
	require.True(t, ok)
	assert.Equal(t, 0, ts.Turn)
}

func TestTranslateEvent_ExtOutdated(t *testing.T) {
	payload := sdk.OutdatedEvent{
		Extensions: []sdk.OutdatedInfo{
			{Name: "mcp", LocalHead: "abc123", RemoteHead: "def456"},
			{Name: "diff-viewer", LocalHead: "111", RemoteHead: "222"},
		},
	}

	msg := translateEvent(sdk.NewEvent(topicExtOutdated, payload))
	outdated, ok := msg.(OutdatedNotificationMsg)
	require.True(t, ok)
	require.Len(t, outdated.Extensions, 2)
	assert.Equal(t, "mcp", outdated.Extensions[0].Name)
	assert.Equal(t, "diff-viewer", outdated.Extensions[1].Name)
}

func TestTranslateEvent_ExtOutdated_NonEventPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicExtOutdated, "not an event"))
	outdated, ok := msg.(OutdatedNotificationMsg)
	require.True(t, ok)
	assert.Empty(t, outdated.Extensions)
}

func TestTranslateEvent_ExtOutdated_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicExtOutdated, nil))
	outdated, ok := msg.(OutdatedNotificationMsg)
	require.True(t, ok)
	assert.Empty(t, outdated.Extensions)
}

func TestBridge_ForwardsEventsAndShutdown(t *testing.T) {
	sender := &collectingSender{}

	events := make(chan sdk.Event, 5)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	events <- sdk.NewEvent(topicMsgStart, nil)

	events <- sdk.NewEvent(topicMsgUpdate, "hello")

	events <- sdk.NewEvent(topicTurnEnd, nil)

	close(events)

	<-done

	require.Len(t, sender.msgs, 6) // state change + msg start + update + state change + turn end + shutdown

	_, ok := sender.msgs[0].(AgentStateChangeMsg)
	assert.True(t, ok)

	_, ok = sender.msgs[1].(MessageStartMsg)
	assert.True(t, ok)

	mu, ok := sender.msgs[2].(MessageUpdateMsg)
	assert.True(t, ok)
	assert.Equal(t, "hello", mu.Content)

	_, ok = sender.msgs[3].(AgentStateChangeMsg)
	assert.True(t, ok)

	_, ok = sender.msgs[4].(TurnEndMsg)
	assert.True(t, ok)

	_, ok = sender.msgs[5].(ShutdownMsg)
	assert.True(t, ok)
}

func TestBridge_SkipsUnknownTopics(t *testing.T) {
	sender := &collectingSender{}

	events := make(chan sdk.Event, 5)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	events <- sdk.NewEvent("unknown.topic", "data")

	events <- sdk.NewEvent(topicMsgStart, nil)

	close(events)

	<-done

	require.Len(t, sender.msgs, 3) // unknown skipped, state change + msg start + shutdown

	_, ok := sender.msgs[0].(AgentStateChangeMsg)
	assert.True(t, ok)

	_, ok = sender.msgs[1].(MessageStartMsg)
	assert.True(t, ok)

	_, ok = sender.msgs[2].(ShutdownMsg)
	assert.True(t, ok)
}

func TestBridge_IntegrationWithRealBus(t *testing.T) {
	b := bus.New()

	events := make(chan sdk.Event, 256)

	b.OnAll(func(ev sdk.Event) error {
		select {
		case events <- ev:
		default:
		}

		return nil
	})

	sender := &collectingSender{}

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	b.Publish(sdk.NewEvent(topicTurnStart, 1))
	b.Publish(sdk.NewEvent(topicMsgStart, nil))
	b.Publish(sdk.NewEvent(topicMsgUpdate, "hi"))
	b.Publish(sdk.NewEvent(topicMsgEnd, map[string]any{"content": "hi", "tool_calls": []sdk.ToolCall{}}))
	b.Publish(sdk.NewEvent(topicTurnEnd, nil))
	b.Publish(sdk.NewEvent(topicEnd, nil))

	_ = b.Close()

	close(events)

	<-done

	require.Len(t, sender.msgs, 9) // 6 events + 2 state changes + shutdown

	assert.IsType(t, AgentStateChangeMsg{}, sender.msgs[0]) // Idle→Streaming
	assert.IsType(t, TurnStartMsg{}, sender.msgs[1])
	assert.IsType(t, MessageStartMsg{}, sender.msgs[2])

	mu, ok := sender.msgs[3].(MessageUpdateMsg)
	require.True(t, ok)
	assert.Equal(t, "hi", mu.Content)

	assert.IsType(t, MessageEndMsg{}, sender.msgs[4])
	assert.IsType(t, AgentStateChangeMsg{}, sender.msgs[5]) // Streaming→Idle
	assert.IsType(t, TurnEndMsg{}, sender.msgs[6])
	assert.IsType(t, AgentEndMsg{}, sender.msgs[7])
	assert.IsType(t, ShutdownMsg{}, sender.msgs[8])
}

func TestPublishPrompt(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, topicPrompt)

	cmd := PublishPrompt(b, "hello world")
	result := cmd()
	assert.Nil(t, result)

	evt := <-ch
	assert.Equal(t, topicPrompt, evt.Topic)
	assert.Equal(t, "hello world", evt.Payload)
}

func TestPublishFollowup(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, topicFollowup)

	cmd := PublishFollowup(b, "follow up text")
	result := cmd()
	assert.Nil(t, result)

	evt := <-ch
	assert.Equal(t, topicFollowup, evt.Topic)
	assert.Equal(t, "follow up text", evt.Payload)
}

func TestPublishSteer(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, topicSteer)

	cmd := PublishSteer(b, "steer text")
	result := cmd()
	assert.Nil(t, result)

	evt := <-ch
	assert.Equal(t, topicSteer, evt.Topic)
	assert.Equal(t, "steer text", evt.Payload)
}

func TestBridge_ToolLifecycleEvents(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 20)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	// Tool lifecycle: start → progress → complete
	events <- sdk.NewEvent(topicToolStart, map[string]any{
		"id":    "tc1",
		"tool":  "bash",
		"input": "ls",
	})

	events <- sdk.NewEvent(topicToolProgress, map[string]any{
		"id":      "tc1",
		"tool":    "bash",
		"content": "partial output",
	})

	events <- sdk.NewEvent(topicToolComplete, map[string]any{
		"id":      "tc1",
		"tool":    "bash",
		"content": "final output",
	})

	close(events)
	<-done

	// Should have: state change (Idle→ToolRunning) + ToolStart + ToolProgress + state change (ToolRunning→Streaming) + ToolComplete + Shutdown
	var toolStartFound, toolProgressFound, toolCompleteFound bool
	var stateChanges []AgentStateChangeMsg

	for _, msg := range sender.msgs {
		switch m := msg.(type) {
		case ToolStartMsg:
			toolStartFound = true
			assert.Equal(t, "tc1", m.ToolID)
			assert.Equal(t, "bash", m.Tool)
			assert.Equal(t, "ls", m.Input)
		case ToolProgressMsg:
			toolProgressFound = true
			assert.Equal(t, "partial output", m.Content)
		case ToolCompleteMsg:
			toolCompleteFound = true
			assert.Equal(t, "final output", m.Content)
		case AgentStateChangeMsg:
			stateChanges = append(stateChanges, m)
		}
	}

	assert.True(t, toolStartFound, "expected ToolStartMsg")
	assert.True(t, toolProgressFound, "expected ToolProgressMsg")
	assert.True(t, toolCompleteFound, "expected ToolCompleteMsg")
	require.Len(t, stateChanges, 2)
	assert.Equal(t, palette.StateToolRunning, stateChanges[0].State)
	assert.Equal(t, palette.StateStreaming, stateChanges[1].State)
}

func TestBridge_ToolErrorEvent(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 10)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	events <- sdk.NewEvent(topicToolStart, map[string]any{
		"id":   "tc1",
		"tool": "bash",
	})

	events <- sdk.NewEvent(topicToolError, map[string]any{
		"id":    "tc1",
		"tool":  "bash",
		"error": "command not found",
	})

	close(events)
	<-done

	var toolErrorFound bool
	var stateChanges []AgentStateChangeMsg

	for _, msg := range sender.msgs {
		switch m := msg.(type) {
		case ToolErrorMsg:
			toolErrorFound = true
			assert.Equal(t, "command not found", m.Error)
		case AgentStateChangeMsg:
			stateChanges = append(stateChanges, m)
		}
	}

	assert.True(t, toolErrorFound, "expected ToolErrorMsg")
	require.Len(t, stateChanges, 2)
	assert.Equal(t, palette.StateToolRunning, stateChanges[0].State)
	assert.Equal(t, palette.StateStreaming, stateChanges[1].State)
}

func TestBridge_ToolInterruptedEvent(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 10)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	events <- sdk.NewEvent(topicToolStart, map[string]any{
		"id":   "tc1",
		"tool": "bash",
	})

	events <- sdk.NewEvent(topicToolInterrupted, map[string]any{
		"id":   "tc1",
		"tool": "bash",
	})

	close(events)
	<-done

	var toolInterruptedFound bool
	var stateChanges []AgentStateChangeMsg

	for _, msg := range sender.msgs {
		switch m := msg.(type) {
		case ToolInterruptedMsg:
			toolInterruptedFound = true
			assert.Equal(t, "tc1", m.ToolID)
		case AgentStateChangeMsg:
			stateChanges = append(stateChanges, m)
		}
	}

	assert.True(t, toolInterruptedFound, "expected ToolInterruptedMsg")
	require.Len(t, stateChanges, 2)
	assert.Equal(t, palette.StateToolRunning, stateChanges[0].State)
	assert.Equal(t, palette.StateStreaming, stateChanges[1].State)
}

// collectingSender captures Send calls for testing.
type collectingSender struct {
	msgs []tea.Msg
}

func (c *collectingSender) Send(msg tea.Msg) {
	c.msgs = append(c.msgs, msg)
}

func TestBridge_DeltaBatching(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 10)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	// Send three deltas in rapid succession
	events <- sdk.NewEvent(topicMsgUpdate, "hello ")

	events <- sdk.NewEvent(topicMsgUpdate, "world ")

	events <- sdk.NewEvent(topicMsgUpdate, "test")

	close(events)

	<-done

	// The bridge should batch consecutive MessageUpdateMsg into one
	// (or at most a few) messages
	require.GreaterOrEqual(t, len(sender.msgs), 1, "expected at least 1 message, got %d", len(sender.msgs))

	// Find all MessageUpdateMsg
	var updates []string

	for _, msg := range sender.msgs {
		if mu, ok := msg.(MessageUpdateMsg); ok {
			updates = append(updates, mu.Content)
		}
	}

	// All content should be present (either in one batched msg or multiple)
	var combined strings.Builder
	for _, u := range updates {
		combined.WriteString(u)
	}

	assert.Equal(t, "hello world test", combined.String())

	// Last message should be ShutdownMsg
	_, ok := sender.msgs[len(sender.msgs)-1].(ShutdownMsg)
	assert.True(t, ok)
}

func TestBridge_DeltaBatchingMixedEvents(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 10)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	events <- sdk.NewEvent(topicMsgUpdate, "delta1")

	events <- sdk.NewEvent(topicMsgUpdate, "delta2")

	events <- sdk.NewEvent(topicTurnEnd, nil) // non-delta breaks the batch

	events <- sdk.NewEvent(topicMsgUpdate, "delta3")

	close(events)

	<-done

	// Should have: batched(delta1+delta2), TurnEnd, delta3, Shutdown
	require.GreaterOrEqual(t, len(sender.msgs), 3)

	// Last message is always ShutdownMsg
	_, ok := sender.msgs[len(sender.msgs)-1].(ShutdownMsg)
	assert.True(t, ok)

	// Verify combined content of all updates
	var combined strings.Builder

	for _, msg := range sender.msgs {
		if mu, ok := msg.(MessageUpdateMsg); ok {
			combined.WriteString(mu.Content)
		}
	}

	assert.Equal(t, "delta1delta2delta3", combined.String())

	// Verify TurnEndMsg is present
	hasTurnEnd := false

	for _, msg := range sender.msgs {
		if _, ok := msg.(TurnEndMsg); ok {
			hasTurnEnd = true
		}
	}

	assert.True(t, hasTurnEnd)
}

func TestPublishInterrupt(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, topicInterrupt)

	cmd := PublishInterrupt(b)
	result := cmd()
	assert.Nil(t, result)

	evt := <-ch
	assert.Equal(t, topicInterrupt, evt.Topic)
	assert.Equal(t, "user interrupt", evt.Payload)
}

func TestBridge_TokenRateTracking(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 20)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	// Start streaming
	events <- sdk.NewEvent(topicMsgStart, nil)

	// Send a delta with known content (enough runes for a measurable rate)
	events <- sdk.NewEvent(topicMsgUpdate, "hello world test data here")

	close(events)

	<-done

	// Should have: MessageStartMsg, MessageUpdateMsg(with rate), ShutdownMsg
	require.GreaterOrEqual(t, len(sender.msgs), 2, "expected at least 2 messages")

	// Find the MessageUpdateMsg
	var updateMsg MessageUpdateMsg

	found := false

	for _, msg := range sender.msgs {
		if mu, ok := msg.(MessageUpdateMsg); ok {
			updateMsg = mu
			found = true
		}
	}

	require.True(t, found, "expected a MessageUpdateMsg")
	assert.Equal(t, "hello world test data here", updateMsg.Content)
	assert.Greater(t, updateMsg.TokenRate, float64(0), "token rate should be > 0")
}

func TestBridge_TokenRateResetsOnMessageEnd(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 20)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	// First message: start, update, end
	events <- sdk.NewEvent(topicMsgStart, nil)

	events <- sdk.NewEvent(topicMsgUpdate, "first message")

	events <- sdk.NewEvent(topicMsgEnd, map[string]any{"content": "first message", "tool_calls": []sdk.ToolCall{}})

	// Second message: start, update
	events <- sdk.NewEvent(topicMsgStart, nil)

	time.Sleep(10 * time.Millisecond) // ensure non-zero elapsed time for rate calc

	events <- sdk.NewEvent(topicMsgUpdate, "second message")

	close(events)

	<-done

	// Find the two update messages and verify rates are independent
	var rates []float64

	for _, msg := range sender.msgs {
		if mu, ok := msg.(MessageUpdateMsg); ok {
			rates = append(rates, mu.TokenRate)
		}
	}

	require.Len(t, rates, 2)
	assert.Greater(t, rates[0], float64(0), "first message rate should be > 0")
	assert.Greater(t, rates[1], float64(0), "second message rate should be > 0 (independent)")
}

func TestTranslateEvent_Compacted(t *testing.T) {
	payload := map[string]any{
		"summarized":    5,
		"tokens_before": 10000,
		"tokens_after":  3000,
	}

	msg := translateEvent(sdk.NewEvent(topicCompacted, payload))
	c, ok := msg.(CompactedMsg)
	require.True(t, ok)
	assert.Equal(t, 5, c.Summarized)
	assert.Equal(t, 10000, c.TokensBefore)
	assert.Equal(t, 3000, c.TokensAfter)
	assert.Empty(t, c.Error)
}

func TestTranslateEvent_Compacted_WithError(t *testing.T) {
	payload := map[string]any{
		"error": "compaction stream: timeout",
	}

	msg := translateEvent(sdk.NewEvent(topicCompacted, payload))
	c, ok := msg.(CompactedMsg)
	require.True(t, ok)
	assert.Equal(t, "compaction stream: timeout", c.Error)
}

func TestTranslateEvent_Compacted_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicCompacted, nil))
	c, ok := msg.(CompactedMsg)
	require.True(t, ok)
	assert.Empty(t, c.Error)
	assert.Zero(t, c.Summarized)
}

func TestTranslateEvent_Compacted_NonMapPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicCompacted, "not a map"))
	c, ok := msg.(CompactedMsg)
	require.True(t, ok)
	assert.Empty(t, c.Error)
}

func TestTranslateEvent_Usage(t *testing.T) {
	payload := map[string]any{
		"input_tokens":          1000,
		"output_tokens":         500,
		"cache_creation_tokens": 50,
		"cache_read_tokens":     200,
		"context_tokens":        93800,
	}

	msg := translateEvent(sdk.NewEvent(topicUsage, payload))
	u, ok := msg.(TokenUsageMsg)
	require.True(t, ok)
	assert.Equal(t, 1000, u.InputTokens)
	assert.Equal(t, 500, u.OutputTokens)
	assert.Equal(t, 50, u.CacheCreationTokens)
	assert.Equal(t, 200, u.CacheReadTokens)
	assert.Equal(t, 93800, u.ContextTokens)
}

func TestTranslateEvent_Usage_NilPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicUsage, nil))
	u, ok := msg.(TokenUsageMsg)
	require.True(t, ok)
	assert.Zero(t, u.InputTokens)
	assert.Zero(t, u.OutputTokens)
}

func TestTranslateEvent_Usage_NonMapPayload(t *testing.T) {
	msg := translateEvent(sdk.NewEvent(topicUsage, "not a map"))
	u, ok := msg.(TokenUsageMsg)
	require.True(t, ok)
	assert.Zero(t, u.InputTokens)
	assert.Zero(t, u.OutputTokens)
}

func TestBridge_UsageEvent(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 5)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	events <- sdk.NewEvent(topicMsgStart, nil)

	events <- sdk.NewEvent(topicMsgUpdate, "hello")

	events <- sdk.NewEvent(topicUsage, map[string]any{
		"input_tokens":   100,
		"output_tokens":  50,
		"context_tokens": 1000,
	})

	events <- sdk.NewEvent(topicTurnEnd, nil)

	close(events)

	<-done

	// Find the TokenUsageMsg
	var found bool

	for _, msg := range sender.msgs {
		if u, ok := msg.(TokenUsageMsg); ok {
			found = true

			assert.Equal(t, 100, u.InputTokens)
			assert.Equal(t, 50, u.OutputTokens)
			assert.Equal(t, 1000, u.ContextTokens)
		}
	}

	assert.True(t, found, "expected TokenUsageMsg in sent messages")
}

func TestCalcTokenRate(t *testing.T) {
	tests := []struct {
		name       string
		runes      int
		elapsedSec float64
		wantRate   float64
	}{
		{name: "100 runes in 1s = 25 tok/s", runes: 100, elapsedSec: 1, wantRate: 25},
		{name: "400 runes in 1s = 100 tok/s", runes: 400, elapsedSec: 1, wantRate: 100},
		{name: "80 runes in 0.1s = 200 tok/s", runes: 80, elapsedSec: 0.1, wantRate: 200},
		{name: "zero runes", runes: 0, elapsedSec: 1, wantRate: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now().Add(-time.Duration(tt.elapsedSec * float64(time.Second)))
			rate := calcTokenRate(start, tt.runes)
			assert.InDelta(t, tt.wantRate, rate, 0.5)
		})
	}

	t.Run("zero time returns zero", func(t *testing.T) {
		rate := calcTokenRate(time.Time{}, 100)
		assert.InDelta(t, float64(0), rate, 0.001)
	})
}
