package bridge

import (
	"testing"

	"github.com/weave-agent/weave/sdk"

	tuievents "github.com/weave-agent/weave-tui/internal/events"
	"github.com/weave-agent/weave-tui/internal/palette"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentStateTracker_Initial(t *testing.T) {
	tracker := newAgentStateTracker()
	assert.Equal(t, palette.StateIdle, tracker.state)
}

func TestAgentStateTracker_TurnStartToStreaming(t *testing.T) {
	tracker := newAgentStateTracker()

	state, changed := tracker.update(tuievents.TurnStartMsg{})
	assert.True(t, changed)
	assert.Equal(t, palette.StateStreaming, state)
}

func TestAgentStateTracker_TurnEndToIdle(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.TurnStartMsg{}) // Idle → Streaming

	state, changed := tracker.update(tuievents.TurnEndMsg{})
	assert.True(t, changed)
	assert.Equal(t, palette.StateIdle, state)
}

func TestAgentStateTracker_MessageEndWithToolCalls(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.TurnStartMsg{}) // Streaming

	msg := tuievents.MessageEndMsg{
		Content:   "response",
		ToolCalls: []sdk.ToolCall{{ID: "t1", Name: "bash"}, {ID: "t2", Name: "read"}},
	}
	state, changed := tracker.update(msg)
	assert.True(t, changed)
	assert.Equal(t, palette.StateToolRunning, state)
	assert.Equal(t, 2, tracker.toolCount)
}

func TestAgentStateTracker_ToolResultDecrements(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.TurnStartMsg{})
	tracker.update(tuievents.MessageEndMsg{
		ToolCalls: []sdk.ToolCall{{ID: "t1", Name: "bash"}, {ID: "t2", Name: "read"}},
	})
	assert.Equal(t, palette.StateToolRunning, tracker.state)

	// First tool result: still ToolRunning (one remaining)
	state, changed := tracker.update(tuievents.ToolResultMsg{ToolID: "t1"})
	assert.False(t, changed) // still ToolRunning
	assert.Equal(t, palette.StateToolRunning, state)

	// Second tool result: back to Streaming
	state, changed = tracker.update(tuievents.ToolResultMsg{ToolID: "t2"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateStreaming, state)
}

func TestAgentStateTracker_AgentEndWithError(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.TurnStartMsg{})

	state, changed := tracker.update(tuievents.AgentEndMsg{Payload: "timeout error"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateError, state)
}

func TestAgentStateTracker_AgentEndNoError(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.TurnStartMsg{})

	state, changed := tracker.update(tuievents.AgentEndMsg{Payload: nil})
	assert.True(t, changed)
	assert.Equal(t, palette.StateIdle, state)
}

func TestAgentStateTracker_NoChangeOnSameState(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.TurnStartMsg{}) // Idle → Streaming

	// MessageStart while streaming: no state change
	_, changed := tracker.update(tuievents.MessageStartMsg{})
	assert.False(t, changed)

	// MessageUpdate: no state change
	_, changed = tracker.update(tuievents.MessageUpdateMsg{Content: "hello"})
	assert.False(t, changed)
}

func TestAgentStateTracker_FullCycle(t *testing.T) {
	tracker := newAgentStateTracker()

	// Turn starts: Idle → Streaming
	state, _ := tracker.update(tuievents.TurnStartMsg{})
	assert.Equal(t, palette.StateStreaming, state)

	// Tool calls in message: Streaming → ToolRunning
	state, _ = tracker.update(tuievents.MessageEndMsg{
		ToolCalls: []sdk.ToolCall{{ID: "t1", Name: "bash"}},
	})
	assert.Equal(t, palette.StateToolRunning, state)

	// Tool result: ToolRunning → Streaming (no more pending tools)
	state, _ = tracker.update(tuievents.ToolResultMsg{ToolID: "t1"})
	assert.Equal(t, palette.StateStreaming, state)

	// Turn ends: Streaming → Idle
	state, _ = tracker.update(tuievents.TurnEndMsg{})
	assert.Equal(t, palette.StateIdle, state)
}

func TestAgentStateTracker_ToolStartSetsToolRunning(t *testing.T) {
	tracker := newAgentStateTracker()

	state, changed := tracker.update(tuievents.ToolStartMsg{ToolID: "t1", Tool: "bash"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateToolRunning, state)
	assert.Equal(t, 1, tracker.toolCount)
}

func TestAgentStateTracker_ToolCompleteDecrements(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.ToolStartMsg{ToolID: "t1", Tool: "bash"})
	assert.Equal(t, 1, tracker.toolCount)

	state, changed := tracker.update(tuievents.ToolCompleteMsg{ToolID: "t1", Tool: "bash"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateStreaming, state)
	assert.Equal(t, 0, tracker.toolCount)
}

func TestAgentStateTracker_ToolErrorDecrements(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.ToolStartMsg{ToolID: "t1", Tool: "bash"})

	state, changed := tracker.update(tuievents.ToolErrorMsg{ToolID: "t1", Tool: "bash", Error: "failed"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateStreaming, state)
	assert.Equal(t, 0, tracker.toolCount)
}

func TestAgentStateTracker_ToolInterruptedDecrements(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.ToolStartMsg{ToolID: "t1", Tool: "bash"})

	state, changed := tracker.update(tuievents.ToolInterruptedMsg{ToolID: "t1", Tool: "bash"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateStreaming, state)
	assert.Equal(t, 0, tracker.toolCount)
}

func TestAgentStateTracker_ToolStartWhenAlreadyToolRunning(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.ToolStartMsg{ToolID: "t1", Tool: "bash"})
	assert.Equal(t, 1, tracker.toolCount)
	assert.Equal(t, palette.StateToolRunning, tracker.state)

	// Second tool start while already running: count increments but no state change
	state, changed := tracker.update(tuievents.ToolStartMsg{ToolID: "t2", Tool: "read"})
	assert.False(t, changed)
	assert.Equal(t, palette.StateToolRunning, state)
	assert.Equal(t, 2, tracker.toolCount)
}

func TestAgentStateTracker_ToolCompleteWhenToolCountZero(t *testing.T) {
	tracker := newAgentStateTracker()

	// Stray complete without prior start should not go negative
	state, changed := tracker.update(tuievents.ToolCompleteMsg{ToolID: "t1", Tool: "bash"})
	assert.False(t, changed)
	assert.Equal(t, palette.StateIdle, state)
	assert.Equal(t, 0, tracker.toolCount)
}

func TestAgentStateTracker_ToolStartThenMultipleResults(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.ToolStartMsg{ToolID: "t1", Tool: "bash"})
	tracker.update(tuievents.ToolStartMsg{ToolID: "t2", Tool: "read"})
	assert.Equal(t, 2, tracker.toolCount)
	assert.Equal(t, palette.StateToolRunning, tracker.state)

	// First complete: still ToolRunning
	state, changed := tracker.update(tuievents.ToolCompleteMsg{ToolID: "t1", Tool: "bash"})
	assert.False(t, changed)
	assert.Equal(t, palette.StateToolRunning, state)
	assert.Equal(t, 1, tracker.toolCount)

	// Second complete: back to Streaming
	state, changed = tracker.update(tuievents.ToolCompleteMsg{ToolID: "t2", Tool: "read"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateStreaming, state)
	assert.Equal(t, 0, tracker.toolCount)
}

func TestAgentStateTracker_ToolLifecycleMixedWithStreaming(t *testing.T) {
	tracker := newAgentStateTracker()
	tracker.update(tuievents.TurnStartMsg{}) // Idle → Streaming

	// Tool starts while streaming
	state, changed := tracker.update(tuievents.ToolStartMsg{ToolID: "t1", Tool: "bash"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateToolRunning, state)

	// Tool completes: back to Streaming
	state, changed = tracker.update(tuievents.ToolCompleteMsg{ToolID: "t1", Tool: "bash"})
	assert.True(t, changed)
	assert.Equal(t, palette.StateStreaming, state)
}

func TestTranslateEvent_AgentStateChangeNotFromBus(t *testing.T) {
	// tuievents.AgentStateChangeMsg is generated by the bridge's tracker, not from bus events.
	// translateEvent should return nil for it (it's not a bus topic).
	msg := translateEvent(sdk.NewEvent("agent.state.change", nil))
	assert.Nil(t, msg)
}

func TestBridge_EmitsStateChanges(t *testing.T) {
	sender := &collectingSender{}
	events := make(chan sdk.Event, 20)

	done := make(chan struct{})

	go func() {
		Bridge(sender, events)
		close(done)
	}()

	// Full agent lifecycle
	events <- sdk.NewEvent(topicTurnStart, 1) // Idle→Streaming

	events <- sdk.NewEvent(topicMsgStart, nil) // no change

	events <- sdk.NewEvent(topicMsgUpdate, "hi") // no change

	events <- sdk.NewEvent(topicMsgEnd, map[string]any{ // Streaming→ToolRunning
		"content":    "response",
		"tool_calls": []sdk.ToolCall{{ID: "t1", Name: "bash"}},
	})

	events <- sdk.NewEvent(topicToolResult, map[string]any{ // ToolRunning→Streaming
		"id":     "t1",
		"tool":   "bash",
		"result": sdk.ToolResult{Content: "output"},
	})

	events <- sdk.NewEvent(topicTurnEnd, nil) // Streaming→Idle

	events <- sdk.NewEvent(topicEnd, "error") // Idle→Error

	close(events)
	<-done

	// Count state change messages
	var stateMsgs []tuievents.AgentStateChangeMsg

	for _, msg := range sender.msgs {
		if sc, ok := msg.(tuievents.AgentStateChangeMsg); ok {
			stateMsgs = append(stateMsgs, sc)
		}
	}

	require.Len(t, stateMsgs, 5)
	assert.Equal(t, palette.StateStreaming, stateMsgs[0].State)
	assert.Equal(t, palette.StateToolRunning, stateMsgs[1].State)
	assert.Equal(t, palette.StateStreaming, stateMsgs[2].State)
	assert.Equal(t, palette.StateIdle, stateMsgs[3].State)
	assert.Equal(t, palette.StateError, stateMsgs[4].State)
}
