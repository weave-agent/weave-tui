# Reorganize TUI Repository Around Public API Boundary

## Overview
- Reorganize `github.com/weave-agent/weave-tui` so root `package tui` exposes only the supported extension-author API.
- Move runtime implementation, Bubble Tea model internals, rendering components, palette/style implementation, and feature-specific code under `internal/` packages.
- Preserve the documented extension-author import path `github.com/weave-agent/weave-tui` for the minimal public API while intentionally hiding accidental public implementation packages.
- Keep observable TUI behavior unchanged; this is a structural refactoring, not a feature change.

## Context (from discovery)
- Files/components involved:
  - Root implementation: `tui.go`, `ui_impl.go`, `model.go`, `bridge.go`, `layout.go`, `commands.go`, `keybindings.go`, `panel.go`, `panel_tray.go`, `auth_selector.go`, `login_flow.go`, `models.go`, `providers.go`, `sessions.go`, `landing.go`, `overlays.go`, `attachments_panel.go`, `focus.go`, `settings.go`, `tui_ext_api.go`, `tui_ext_registry.go`
  - Public implementation packages to internalize: `components/`, `components/messages/`, `components/overlays/`, `components/attachments/`, `palette/`, `styles/`, `xchroma/`
  - Tests: extensive same-package tests beside the implementation, especially `model_test.go`, `ui_impl_test.go`, `integration_test.go`, and component package tests
- Related patterns found:
  - README documents downstream extensions importing only root `github.com/weave-agent/weave-tui`.
  - Root currently exposes many accidental internals: command registry, keybinding registry, bridge messages, layout engine, panel manager, landing model, provider/session/auth helpers, and UI implementation.
  - Component/rendering packages are public today only because they are top-level packages, not because they are documented as extension API.
- Dependencies identified:
  - Public API must still expose Bubble Tea/Ultraviolet types where existing extension APIs require them (`PanelDrawer`, `TUIComponent`).
  - Internal packages must not import root `github.com/weave-agent/weave-tui` to avoid cycles.
  - Bubble Tea message types should move to a dedicated internal package so bridge and model packages do not depend on each other.

## Development Approach
- **Testing approach**: Regular — move/reorg code first, then update/move tests and run tests after each task.
- Complete each task fully before moving to the next.
- Make small, focused changes within the one-shot reorg.
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task.
  - Tests are not optional - they are a required part of the checklist.
  - Write unit tests for new functions/methods.
  - Write unit tests for modified functions/methods.
  - Add new test cases for new code paths.
  - Update existing test cases if behavior changes.
  - Tests cover both success and error scenarios.
- **CRITICAL: all tests must pass before starting next task** - no exceptions.
- **CRITICAL: update this plan file when scope changes during implementation**.
- Run tests after each change.
- Maintain compatibility for the minimal root extension API.

## Testing Strategy
- **Unit tests**: required for every task (see Development Approach above).
- **Package visibility tests**:
  - Confirm `go list ./...` exposes only root plus `internal/...` packages.
  - Confirm root public API remains usable by an external-style test package where practical.
- **Integration tests**:
  - Move existing root implementation integration tests to the package that owns the behavior.
  - Keep root tests limited to public API and registration behavior.
- **E2E tests**: none identified in this repository. If UI-based e2e tests are discovered during implementation, update them in the same task as related UI changes.

## Progress Tracking
- Mark completed items with `[x]` immediately when done.
- Add newly discovered tasks with ➕ prefix.
- Document issues/blockers with ⚠️ prefix.
- Update plan if implementation deviates from original scope.
- Keep plan in sync with actual work done.

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): tasks achievable within this codebase - code changes, tests, documentation updates.
- **Post-Completion** (no checkboxes): items requiring external action - manual testing, changes in consuming projects, deployment configs, third-party verifications.
- **Checkbox placement**: Checkboxes belong only in Task sections (`### Task N:` or `### Iteration N:`). Do not put checkboxes in acceptance criteria, Overview, or Context — they cause extra loop iterations.

## Implementation Steps

### Task 1: Establish public contract package and root facade
- [x] create `internal/contract` with `TUIExtension`, `TUIExtAPI`, panel API types, renderer API types, autocomplete API types, key event type, and theme definition
- [x] update root `tui_ext_api.go` or replacement `api.go` to expose aliases for the minimal supported API only
- [x] keep `TUIConfig` in root or alias it from an internal config package if needed by registration schema
- [x] remove root exposure of implementation-only API from facade files
- [x] add/update root public API tests using `package tui_test` for registration and type usability
- [x] run `go test ./...` - must pass before next task

### Task 2: Move extension registry behind an internal boundary
- [ ] move TUI extension registry implementation to an internal package such as `internal/extensionregistry`
- [ ] update root registry functions to delegate to the internal registry while preserving `RegisterTUIExtension`, `GetTUIExtension`, `ListTUIExtensions`, `TUIExtensionRegistered`, and `ResetTUIExtensionRegistry`
- [ ] update internal callers to use the internal registry package instead of root package globals
- [ ] update registry tests to separate root facade behavior from internal registry behavior
- [ ] run `go test ./...` - must pass before next task

### Task 3: Internalize rendering support packages
- [ ] move `palette/` to `internal/palette/` and update imports
- [ ] move `styles/` to `internal/styles/` and update imports
- [ ] move `xchroma/` to `internal/xchroma/` and update imports
- [ ] move package tests with their implementation packages and update package import paths
- [ ] verify no non-internal package imports `internal/palette`, `internal/styles`, or `internal/xchroma` except root facade where explicitly necessary
- [ ] run `go test ./...` - must pass before next task

### Task 4: Internalize UI component packages
- [ ] move `components/` to `internal/components/` and update imports
- [ ] move `components/attachments/` to `internal/components/attachments/` and update imports
- [ ] move `components/messages/` to `internal/components/messages/` and update imports
- [ ] move `components/overlays/` to `internal/components/overlays/` and update imports
- [ ] move component tests with their implementation packages and update imports
- [ ] run `go test ./...` - must pass before next task

### Task 5: Create internal event message package
- [ ] create `internal/events` for Bubble Tea message structs currently exposed from `bridge.go` and related root files
- [ ] move agent, model, provider, session, tool, auth, compaction, token usage, notification, shutdown, and state-change message types into `internal/events`
- [ ] update bridge, model, UI implementation, and tests to use `internal/events`
- [ ] add/update tests for event translation and message handling after the package move
- [ ] run `go test ./...` - must pass before next task

### Task 6: Move bridge and layout implementation
- [ ] move bus-to-Bubble-Tea bridge code from root to `internal/bridge`
- [ ] move layout engine code from root to `internal/layout`
- [ ] update app/model callers to import `internal/bridge` and `internal/layout`
- [ ] move bridge and layout tests with their packages
- [ ] run `go test ./...` - must pass before next task

### Task 7: Move panels and panel tray implementation
- [ ] move panel manager and tray implementation to `internal/panels`
- [ ] keep only public panel API shapes in `internal/contract` and root aliases
- [ ] update UI/model callers to depend on `internal/panels` for state management and `internal/contract` for public drawer/config types
- [ ] move panel and tray tests with their package and update imports
- [ ] run `go test ./...` - must pass before next task

### Task 8: Move commands and keybindings implementation
- [ ] move slash command registry and built-ins to `internal/commands`
- [ ] move keybinding registry, keybinding loading, and keybinding help dialog to `internal/keybindings`
- [ ] update model/UI callers to use internal commands and keybindings packages
- [ ] move command and keybinding tests with their packages and update imports
- [ ] run `go test ./...` - must pass before next task

### Task 9: Move sessions, auth, and provider/model selection implementation
- [ ] move session listing/resume helpers to `internal/sessions`
- [ ] move login/logout provider selection and login flow helpers to `internal/auth`
- [ ] move provider/model selection helpers to `internal/providers`
- [ ] update command/model/UI callers to use the new internal packages
- [ ] move affected tests with their packages and update imports
- [ ] run `go test ./...` - must pass before next task

### Task 10: Move root Bubble Tea model and UI implementation
- [ ] move root Bubble Tea `Model`, focus state, landing screen, overlays orchestration, and attachments panel code to `internal/model`
- [ ] move `TUIImpl` SDK UI implementation to `internal/ui`
- [ ] update dependencies so `internal/model` and `internal/ui` communicate through internal types without importing root
- [ ] move model/UI implementation tests with their owning packages and update imports
- [ ] run `go test ./...` - must pass before next task

### Task 11: Move application lifecycle and root extension registration wiring
- [ ] move TUI runtime lifecycle from root `tui.go` into `internal/app`
- [ ] update root extension registration to construct the internal app implementation without exposing `TUI`, `NewTUI`, or `ErrNoTTY` as public API
- [ ] ensure `sdk.RegisterExtensionWithScopeAndWriter("tui", "ui", ...)` behavior is unchanged
- [ ] update TUI lifecycle tests to target internal app behavior or root registration behavior as appropriate
- [ ] run `go test ./...` - must pass before next task

### Task 12: Clean root package and verify public surface
- [ ] remove or internalize remaining root files that expose implementation-only symbols
- [ ] ensure root package contains only public API facade, registration facade, config, and extension registration
- [ ] run `go doc .` and verify accidental public internals are gone
- [ ] run `go list ./...` and verify top-level public implementation packages no longer exist
- [ ] update/add root public API tests for the final supported surface
- [ ] run `go test ./...` - must pass before next task

### Task 13: Update documentation
- [ ] update `README.md` to document root-only public API and note that rendering internals are private
- [ ] update `CLAUDE.md` if package layout guidance changes for future agents
- [ ] document the intended internal package ownership map if useful for maintainers
- [ ] run documentation-adjacent checks if available
- [ ] run `go test ./...` - must pass before next task

### Task 14: Verify acceptance criteria
- [ ] verify root `package tui` exposes only the minimal extension-author API
- [ ] verify implementation packages live under `internal/` feature/domain packages
- [ ] verify no internal package imports root `github.com/weave-agent/weave-tui`
- [ ] verify external-style root API tests compile and pass
- [ ] run full test suite with `go test ./...`
- [ ] run `go list ./...`
- [ ] run linter if available for this module

## Technical Details
- Target root public API:
  - `TUIConfig`
  - `RegisterTUIExtension`
  - `GetTUIExtension`
  - `ListTUIExtensions`
  - `TUIExtensionRegistered`
  - `ResetTUIExtensionRegistry`
  - `TUIExtension`
  - `TUIExtAPI`
  - `PanelConfig`
  - `PanelPlacement`, `AsOverlay`, `AboveEditor`, `BelowEditor`, `TrayOnly`
  - `PanelDrawer`
  - `PanelTrayAPI`
  - `RichToolRenderer`
  - `TUIComponent`
  - `KeyEvent`
  - `AutocompleteProvider`
  - `AutocompleteContext`
  - `AutocompleteSuggestion`
  - `ThemeDef`
- Target internal packages:
  - `internal/app`
  - `internal/auth`
  - `internal/bridge`
  - `internal/commands`
  - `internal/components`
  - `internal/components/attachments`
  - `internal/components/messages`
  - `internal/components/overlays`
  - `internal/contract`
  - `internal/events`
  - `internal/extensionregistry`
  - `internal/keybindings`
  - `internal/layout`
  - `internal/model`
  - `internal/palette`
  - `internal/panels`
  - `internal/providers`
  - `internal/sessions`
  - `internal/styles`
  - `internal/ui`
  - `internal/xchroma`
- Dependency rules:
  - Root may import internal packages to expose facade behavior.
  - Internal packages must not import root package.
  - `internal/contract` owns API shapes used by internals and root aliases.
  - `internal/events` owns Bubble Tea message structs shared by bridge, model, and UI implementation.
  - Avoid `utils`, `common`, or pass-through packages.
- Processing flow after reorg:
  - Root extension registration constructs `internal/app`.
  - `internal/app` wires `internal/ui`, `internal/model`, `internal/bridge`, and extension registry.
  - `internal/model` coordinates feature packages and rendering components.
  - `internal/ui` implements SDK UI and TUI extension API against internal state managers.

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- Launch the TUI manually and verify startup, prompt submission, streaming, tool panels, overlays, model selection, login/logout, sessions, keybindings, and shutdown.
- Verify custom TUI extensions using the root API still compile if a local fixture or downstream example is available.

**External system updates**:
- Downstream extensions importing `components`, `palette`, `styles`, or `xchroma` will need to migrate away from those internals.
- Release notes should call out the intentional public API boundary tightening.
