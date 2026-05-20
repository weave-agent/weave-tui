# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`weave-tui` is the terminal UI extension for the `weave` coding-agent framework. It is a Go module built on Bubble Tea/Bubbles v2, Lipgloss v2, and Ultraviolet for terminal layout/drawing.

The extension registers itself in `tui.go` as the `tui` UI extension. `TUI.Subscribe` creates the root Bubble Tea model, wires SDK UI extensions and TUI-specific extensions, starts the event bridge, and publishes `agent.end` on shutdown.

## Common Commands

- Run all tests: `go test ./...`
- Run tests for the root package only: `go test .`
- Run tests for one package: `go test ./components/messages`
- Run a single test: `go test . -run TestName`
- Run a single subtest: `go test . -run 'TestName/SubtestName'`
- Check formatting before committing: `gofmt -w <files>`

For local development against a local weave SDK checkout, the README suggests adding a temporary `replace` directive in `go.mod`:

```go
replace github.com/weave-agent/weave => /path/to/local/weave
```

Do not commit that temporary replacement unless intentionally updating module wiring.

## Architecture

### Runtime flow

- `tui.go` is the extension entry point. It registers the extension with the weave SDK, owns the `tea.Program`, and wires SDK UI extensions before the program starts accepting messages.
- `bridge.go` translates weave bus events into Bubble Tea messages. It also batches streaming `agent.message_update` deltas and tracks coarse agent state so the UI can update spinner/accent/pulse state. The `agentStateTracker` counts pending tool calls (`toolCount`) and transitions through `StateIdle`, `StateStreaming`, `StateToolRunning`, and `StateError`. It forwards `tool.start`, `tool.progress`, `tool.complete`, `tool.error`, and `tool.interrupted` events as Tea messages.
- `model.go` contains the root Bubble Tea `Model`. It coordinates chat, editor, footer, overlays, provider/model state, command dispatch, keybindings, attachments, panels, and extension callbacks. It tracks in-flight tool panels (`toolPanels` map) and pending tool call order (`pendingToolCalls`, `pendingToolOrder`), and handles tool lifecycle messages (`ToolStartMsg`, `ToolProgressMsg`, `ToolCompleteMsg`, `ToolErrorMsg`, `ToolInterruptedMsg`).
- `layout.go` computes the terminal regions for header/main/pills/panel tray/panels/docked overlays/editor/footer using Ultraviolet's layout solver.

### User input, commands, and keybindings

- `components/editor.go` wraps `bubbles/v2/textarea` and emits `components.SubmitMsg` on Enter. The root model decides whether the submitted text is a slash command, initial prompt, or followup.
- `commands.go` owns slash command registration and dispatch. Built-ins include `/new`, `/clear`, `/quit`, `/help`, `/compact`, `/name`, `/resume`, `/reload`, `/login`, and `/logout`; `/model`, `/providers`, and `/thinking` are registered during model construction.
- `keybindings.go` maps terminal key strings to action names. Resolution priority is user config > extension registrations > built-in defaults. User keybindings are loaded from `keybindings.yaml` near the weave config or from `~/.weave/keybindings.yaml`.
- Escape key behavior: first press always publishes an `agent.interrupt` event (interrupting both streaming assistants and running tools); second press within the double-press window clears the editor.
- Completion logic is split between `components/completion.go` and `components/path_completion.go`. Slash command and file-reference completions are refreshed by the root model when editor content or cursor position changes.

### Rendering components

- `components/chat.go`, `components/messages/`, `components/footer.go`, and `components/spinner.go` render the main transcript, message types, footer status, and working indicator.
- `components/messages/tool.go` renders `ToolPanel` chat items with a lifecycle state machine: pending, running (with spinner and live progress), success, error, and interrupted. Panels flash on state transitions and support expand/collapse.
- `components/overlays/` contains reusable modal/dialog models for selectors, confirmation, input, editor, multiselect, OAuth login, and dialog stacking.
- `components/attachments/` manages prompt attachments created from large pasted content or explicit attachment actions.
- `palette/` defines themes and agent activity colors. `palette.State` tracks `StateIdle`, `StateStreaming`, `StateToolRunning`, and `StateError`. Agent state changes from the bridge adjust accent colors and editor border pulse behavior; `StateToolRunning` uses amber tones.
- `xchroma/` contains Chroma formatting support used by message/tool rendering.

### Extension APIs

- `ui_impl.go` implements both `sdk.UI` and the local `TUIExtAPI`. It exposes blocking overlays, notifications, command/keybinding registration, renderers, panels, theme APIs, editor access, header/footer replacement, autocomplete providers, and redraw requests.
- `tui_ext_api.go` defines the public TUI-specific extension interfaces, including `TUIExtension`, `TUIExtAPI`, `PanelDrawer`, rich tool renderers, custom components, raw key handlers, autocomplete providers, and theme definitions.
- `tui_ext_registry.go` is the registry for TUI-specific extensions. Factories are registered under the SDK `ui_extensions` schema and instantiated from the current weave config.
- Panel support is split between `panel.go` for panel registration/state, `panel_tray.go` for tab-strip state, and root-model focus/layout code for routing input and drawing active panels.

### Provider, model, auth, and session state

- `models.go` selects the startup model using `WEAVE_PROVIDER`, persisted preferences, registered providers, and available model metadata. It also persists model/thinking choices through the SDK preference store.
- `providers.go`, `auth_selector.go`, and `login_flow.go` support provider listing, API-key entry, OAuth login, logout, and auth status changes.
- `sessions.go` resolves the session directory and supports `/resume` via `session.list`/`session.resume` bus events.
- `settings.go` defines TUI-specific config currently read by the extension (`theme`, `editor_max_lines`).

## Testing Notes

Most files have adjacent `_test.go` coverage. Prefer adding focused tests next to the package being changed:

- Root model/bridge/command behavior: root package tests such as `model_test.go`, `bridge_test.go`, `commands_test.go`, or feature-specific root tests.
- Editor, completion, chat, spinner, and path completion: `components/*_test.go`.
- Message rendering and tool panels: `components/messages/*_test.go`.
- Overlay behavior: `components/overlays/*_test.go`.
- Theme and agent-state color behavior: `palette/*_test.go`.

Use `go test ./...` as the final verification command for changes in this repository.
