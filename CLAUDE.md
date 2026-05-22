# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`weave-tui` is the terminal UI extension for the `weave` coding-agent framework. It is a Go module built on Bubble Tea/Bubbles v2, Lipgloss v2, and Ultraviolet for terminal layout/drawing.

The root package is a public facade for downstream extension authors. It registers the `tui` UI extension with the weave SDK, aliases the supported API from `internal/contract`, and delegates registry operations to `internal/extensionregistry`. Runtime implementation lives under `internal/`.

## Common Commands

- Run all tests: `go test ./...`
- Run tests for the root package only: `go test .`
- Run tests for one package: `go test ./internal/components/messages`
- Run a single test: `go test . -run TestName`
- Run a single subtest: `go test . -run 'TestName/SubtestName'`
- Inspect public root API: `go doc .`
- Inspect exported package list: `go list ./...`
- Check formatting before committing: `gofmt -w <files>`

For local development against a local weave SDK checkout, the README suggests adding a temporary `replace` directive in `go.mod`:

```go
replace github.com/weave-agent/weave => /path/to/local/weave
```

Do not commit that temporary replacement unless intentionally updating module wiring.

## Architecture

### Public boundary

The root import path, `github.com/weave-agent/weave-tui`, is the only supported Go import path for downstream TUI extensions. Keep the root package limited to:

- `TUIConfig`
- `RegisterTUIExtension`, `GetTUIExtension`, `ListTUIExtensions`, `TUIExtensionRegistered`, `ResetTUIExtensionRegistry`
- `TUIExtension`, `TUIExtAPI`
- `PanelConfig`, `PanelPlacement`, `AsOverlay`, `AboveEditor`, `BelowEditor`, `TrayOnly`, `PanelDrawer`, `PanelTrayAPI`
- `RichToolRenderer`, `TUIComponent`
- `KeyEvent`, `AutocompleteProvider`, `AutocompleteContext`, `AutocompleteSuggestion`
- `ThemeDef`

Do not expose Bubble Tea model internals, runtime lifecycle types, registries, rendering components, palette/style packages, or event message structs from the root package. Root may import internal packages to implement facade behavior. Internal packages must not import the root package; use `internal/contract` for shared public API shapes and `internal/events` for shared Bubble Tea messages.

### Runtime flow

- Root `tui.go` registers `internal/app.NewExtension` with the weave SDK as the `tui` UI extension.
- `internal/app` owns the SDK extension lifecycle. It checks for a terminal, creates the `tea.Program`, wires SDK UI extensions into `internal/ui.TUIImpl`, starts `internal/bridge`, closes the UI implementation, and publishes `agent.end` on shutdown.
- `internal/bridge` translates weave bus events into Bubble Tea messages from `internal/events`. It also batches streaming `agent.message_update` deltas and tracks coarse agent state so the UI can update spinner/accent/pulse state. The agent state tracker counts pending tool calls and transitions through idle, streaming, tool-running, and error states.
- `internal/events` owns message structs shared by bridge, model, and UI implementation.
- `internal/model` contains the Bubble Tea `Model`. It coordinates chat, editor, footer, overlays, provider/model state, command dispatch, keybindings, attachments, panels, and extension callbacks. It tracks in-flight tool panels and handles tool lifecycle messages.
- `internal/layout` computes terminal regions for header/main/pills/panel tray/panels/docked overlays/editor/footer using Ultraviolet's layout solver.
- `internal/model/landing.go` renders the pre-prompt boot/status screen showing model, provider, loaded extensions, and keybinding hints in a muted label/accent value layout.

### User input, commands, and keybindings

- `internal/components/editor.go` wraps `bubbles/v2/textarea` and emits `components.SubmitMsg` on Enter. The model decides whether submitted text is a slash command, initial prompt, or followup.
- `internal/commands` owns slash command registration and dispatch. Built-ins include `/new`, `/clear`, `/quit`, `/help`, `/compact`, `/name`, `/resume`, `/reload`, `/login`, and `/logout`; `/model`, `/providers`, and `/thinking` are registered during model construction.
- `internal/keybindings` maps terminal key strings to action names. Resolution priority is user config > extension registrations > built-in defaults. User keybindings are loaded from `keybindings.yaml` near the weave config or from `~/.weave/keybindings.yaml`.
- Escape key behavior: first press always publishes an `agent.interrupt` event (interrupting both streaming assistants and running tools); second press within the double-press window clears the editor.
- Completion logic is split between `internal/components/completion.go` and `internal/components/path_completion.go`. Slash command and file-reference completions are refreshed by the model when editor content or cursor position changes.

### Rendering components

Rendering internals are private implementation packages. Do not add external documentation encouraging downstream extensions to import them.

- `internal/components/chat.go`, `internal/components/messages/`, `internal/components/footer.go`, and `internal/components/spinner.go` render the main transcript, message types, footer status, and working indicator.
- `internal/components/messages/tool.go` renders `ToolPanel` chat items with a lifecycle state machine: pending, running (with spinner and live progress), success, error, and interrupted. Panels flash on state transitions and support expand/collapse.
- `internal/components/overlays/` contains reusable modal/dialog models for selectors, confirmation, input, editor, multiselect, OAuth login, and dialog stacking.
- `internal/components/attachments/` manages prompt attachments created from large pasted content or explicit attachment actions.
- `internal/styles` provides a structured design grammar that maps `internal/palette.Theme` tokens into product-specific render styles. It defines fixed glyph constants (user marker `❯`, assistant `◆`, thinking `∴`, tool pending `○`, success `✓`, error `×`, interrupted `■`) and reusable style helpers for role markers, text, accents, borders, selection rows, tabs, pills, tool states, overlays, and notification banners. Custom themes are treated as color-token changes only; glyphs, spacing, and layout grammar remain fixed in code. Components should use `styles.New(theme)` rather than calling `palette.DefaultTheme()` directly in render paths.
- `internal/palette` defines themes and agent activity colors. Agent state changes from the bridge adjust accent colors and editor border pulse behavior; tool-running state uses amber tones.
- `internal/xchroma` contains Chroma formatting support used by message/tool rendering.

### Focus and selection grammar

Focus states follow type-specific grammar defined in the style set:
- **List/completion rows**: accent background with bold foreground (`SelectedRow`)
- **Panel tray tabs**: bracketed when focused (`FocusedTab`), accent foreground when active but unfocused, muted when inactive
- **Editor/input**: accent border color (do not use row-style selection)
- **Footer/model status**: accent foreground only

### Extension APIs

- Root `tui_ext_api.go` aliases public TUI-specific extension interfaces from `internal/contract`, including `TUIExtension`, `TUIExtAPI`, `PanelDrawer`, rich tool renderers, custom components, raw key handlers, autocomplete providers, and theme definitions.
- Root `tui_ext_registry.go` delegates TUI-specific extension registration to `internal/extensionregistry`. Factories are registered under the SDK `ui_extensions` schema and instantiated from the current weave config.
- `internal/ui/ui_impl.go` implements both `sdk.UI` and `TUIExtAPI`. It exposes blocking overlays, notifications, command/keybinding registration, renderers, panels, theme APIs, editor access, header/footer replacement, autocomplete providers, and redraw requests.
- Panel support is split between `internal/panels/panel.go` for panel registration/state, `internal/panels/panel_tray.go` for tab-strip state, and model focus/layout code for routing input and drawing active panels.

### Provider, model, auth, and session state

- `internal/providers` selects the startup model using `WEAVE_PROVIDER`, persisted preferences, registered providers, and available model metadata. It also persists model/thinking choices through the SDK preference store.
- `internal/auth` and `internal/model/login_flow.go` support provider listing, API-key entry, OAuth login, logout, and auth status changes.
- `internal/sessions` resolves the session directory and supports `/resume` via `session.list`/`session.resume` bus events.
- Root `settings.go` aliases the TUI-specific config currently read by the extension (`theme`, `editor_max_lines`) from `internal/contract`.

### Package ownership map

- Root package: public API facade, registry facade, config alias, and SDK extension registration only
- `internal/contract`: canonical public API shapes used by internals and root aliases
- `internal/extensionregistry`: TUI extension factory registry
- `internal/app`: lifecycle, terminal checks, `tea.Program` ownership, UI extension wiring, shutdown signaling
- `internal/bridge`: bus event translation and agent activity tracking
- `internal/events`: Bubble Tea message structs shared by bridge, model, and UI implementation
- `internal/model`: Bubble Tea model, focus, landing screen, overlays, attachments, provider/model/auth/session orchestration
- `internal/ui`: SDK UI and TUI extension API implementation
- `internal/commands`: slash command registry, built-ins, reload handling
- `internal/keybindings`: binding registry, config loading, help dialog
- `internal/panels`: panel manager and tray state
- `internal/layout`: terminal region layout engine
- `internal/auth`, `internal/providers`, `internal/sessions`: feature-specific state helpers
- `internal/components`, `internal/components/*`, `internal/styles`, `internal/palette`, `internal/xchroma`: private rendering implementation

### Change guidance

- Put new runtime or rendering implementation under `internal/<domain>`.
- Put shared extension-author API shapes in `internal/contract` and expose them from root only when intentionally expanding the public API.
- Put cross-package Bubble Tea messages in `internal/events`.
- Do not create new top-level packages unless the public import surface is intentionally expanding.
- Do not import `github.com/weave-agent/weave-tui` from any `internal/...` package.
- Update README.md and this file when package ownership or the public boundary changes.

## Testing Notes

Most files have adjacent `_test.go` coverage. Prefer adding focused tests next to the package being changed:

- Root package tests should cover only public API facade behavior and registration behavior.
- Runtime lifecycle: `internal/app/*_test.go`.
- Bridge behavior and agent-state translation: `internal/bridge/*_test.go`.
- Bubble Tea model behavior: `internal/model/*_test.go`.
- Command and keybinding behavior: `internal/commands/*_test.go` and `internal/keybindings/*_test.go`.
- Editor, completion, chat, spinner, and path completion: `internal/components/*_test.go`.
- Message rendering and tool panels: `internal/components/messages/*_test.go`.
- Overlay behavior: `internal/components/overlays/*_test.go`.
- Theme and agent-state color behavior: `internal/palette/*_test.go` and `internal/styles/*_test.go`.
- Package visibility and public API surface: root `api_test.go`.

Use `go test ./...` as the final verification command for changes in this repository.
