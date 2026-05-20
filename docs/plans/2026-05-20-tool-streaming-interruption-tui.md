# Tool Streaming & Interruption — TUI Rendering and ESC Fix

## Overview
Fix ESC so it always interrupts running tools (not just streaming assistants). Add TUI rendering for tool lifecycle events: show spinner while running, live updates for progress, and final state for complete/error/interrupted.

## Context
- `model.go:handleEscape()` — only publishes interrupt for `await_agent` / `subagent_*`; falls through to `interruptStreaming()` which bails if no streaming assistant
- `bridge.go` — forwards bus events to Bubble Tea messages
- `model.go` — `activeToolName()` infers state from assistant messages
- Agent loop will publish `tool.start`, `tool.progress`, `tool.complete`, `tool.error`, `tool.interrupted` events

## Development Approach
- Regular approach
- Every task includes tests before moving to next

## Implementation Steps

### Task 1: Fix ESC to always publish interrupt
- [x] Modify `handleEscape()` to always call `PublishInterrupt(m.bus)` regardless of `activeTool`
- [x] Keep existing `activeTool` checks for extra behavior (editor clear, subagent abort)
- [x] Write/update tests for ESC during tool execution
- [x] Run extension tests — must pass

### Task 2: Add tool event messages and bridge forwarding
- [x] Add `ToolStartMsg`, `ToolProgressMsg`, `ToolCompleteMsg`, `ToolErrorMsg`, `ToolInterruptedMsg` types
- [x] Extend `Bridge` to translate new bus events into Tea messages
- [x] Write tests for bridge event translation
- [x] Run extension tests — must pass

### Task 3: Render tool lifecycle in chat
- [ ] Track in-flight tools in model state (map[string]ToolProgress)
- [ ] On `ToolStartMsg`: show tool name + spinner
- [ ] On `ToolProgressMsg`: update display with partial content
- [ ] On `ToolCompleteMsg`: show final result, stop spinner
- [ ] On `ToolErrorMsg`: show error state
- [ ] On `ToolInterruptedMsg`: show "(interrupted)", fade out
- [ ] Write tests for each state transition
- [ ] Run extension tests — must pass

### Task 4: Verify integration
- [ ] Run `go test ./...` in TUI extension dir
- [ ] Run `make lint` if available

## Technical Details

```go
// In handleEscape()
var cmds []tea.Cmd
cmds = append(cmds, PublishInterrupt(m.bus)) // always

if activeTool == "await_agent" || strings.HasPrefix(activeTool, "subagent_") {
    // extra subagent logic
} else {
    model, cmd := m.interruptStreaming()
    cmds = append(cmds, cmd)
}
```

## Post-Completion
- Depends on agent extension publishing lifecycle events
- Manual verification: run a long grep and press ESC — should interrupt and show "(interrupted)"
