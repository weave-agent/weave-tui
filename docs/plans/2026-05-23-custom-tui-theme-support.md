# Custom TUI Theme Support

## Overview
- Add custom theme support to the Weave TUI extension.
- Users can place full JSON theme files under `~/.weave/themes/*.json` and switch themes through `/theme`.
- The selected theme is applied at startup from `ui.theme`, previewed live while selecting, and persisted on confirm.
- The initial scope intentionally excludes CLI flags, theme installation commands, gallery management, and a dedicated keybinding.

## Context (from discovery)
- Files/components involved:
  - `internal/palette/theme.go` defines semantic color slots and the only built-in default theme.
  - `internal/ui/ui_impl.go` already owns a theme registry and implements `RegisterTheme`, `SetTheme`, `ListThemes`, and `Theme`.
  - `internal/contract/contract.go` exposes `TUIConfig.Theme` and `ThemeDef`.
  - `internal/model/model.go` registers slash commands, owns active `theme`/`styles`, and currently ignores `TUIConfig.Theme`.
  - `internal/components/overlays/selector.go` and `internal/components/overlays/stack.go` provide existing selector/dialog patterns.
  - `internal/styles/styles.go` maps `palette.Theme` to Lipgloss styles.
- Related patterns found:
  - Built-in commands are registered in `newModelWithConfig` with `CommandResult{Command: ...}` for Bubble Tea messages.
  - Selector dialogs are pushed onto `dialogStack` and completed through `handleDialogDone`-style message handling.
  - Existing preferences preserve nested `ui` fields, so persisting `ui.theme` should not overwrite unrelated UI settings.
- Dependencies identified:
  - Standard library JSON/filesystem support is enough for theme files.
  - Existing `sdk.PreferenceStore` should be used for persistence.
  - Existing `palette.Theme`, `styles.Styles`, `TUIImpl` registry, and selector overlay should be reused.

## Development Approach
- **Testing approach**: Regular (code first, then tests per task)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
  - tests are not optional - they are a required part of the checklist
  - write unit tests for new functions/methods
  - write unit tests for modified functions/methods
  - add new test cases for new code paths
  - update existing test cases if behavior changes
  - tests cover both success and error scenarios
- **CRITICAL: all tests must pass before starting next task** - no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- Run tests after each change
- Maintain backward compatibility

## Testing Strategy
- **Unit tests**: required for every task.
- **TUI/model tests**: cover `/theme` registration, selector opening, preview, cancel restore, confirm persistence, and startup application.
- **Catalog tests**: cover built-ins, user JSON loading, invalid names, invalid color values, unknown themes, and user override behavior.
- **Regression tests**: ensure existing `RegisterTheme`, `SetTheme`, and `ListThemes` behavior remains compatible.
- **No E2E tests**: no existing UI E2E test harness was identified.

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix
- Update plan if implementation deviates from original scope
- Keep plan in sync with actual work done

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): tasks achievable within this codebase - code changes, tests, documentation updates
- **Post-Completion** (no checkboxes): items requiring external action - manual testing, changes in consuming projects, deployment configs, third-party verifications
- **Checkbox placement**: Checkboxes belong only in Task sections (`### Task N:` or `### Iteration N:`). Do not put checkboxes in Success criteria, Overview, or Context — they cause extra loop iterations.

## Implementation Steps

### Task 1: Add JSON theme catalog
- [x] create an internal theme catalog package for loading built-in themes and `~/.weave/themes/*.json`
- [x] define a full JSON theme file schema matching `palette.Theme` semantic fields
- [x] validate theme names as plain filenames or safe identifiers and reject path traversal
- [x] validate color values accepted by Lipgloss, with `#RRGGBB` as the supported user-file format for v1
- [x] support user theme files overriding built-in themes with the same name
- [x] write tests for built-in loading, user loading, override ordering, and sorted listing
- [x] write tests for malformed JSON, missing required fields, invalid names, invalid colors, and unknown themes
- [x] run theme catalog tests - must pass before task 2

### Task 2: Wire catalog into TUI startup
- [x] load the theme catalog during TUI model construction in `internal/model/model.go`
- [x] register built-in and user themes into the existing `TUIImpl` theme registry
- [x] apply `TUIConfig.Theme` at startup when it resolves to a known theme
- [x] fall back safely to `default` when the configured theme is empty or unknown
- [x] ensure startup theme application updates `Model.theme`, `Model.styles`, editor styles, spinner theme, and UI theme registry consistently
- [x] write tests for startup applying `ui.theme`
- [x] write tests for unknown configured themes falling back without crashing
- [x] run startup/model tests - must pass before task 3

### Task 3: Add `/theme` selector command
- [x] register `/theme` in `newModelWithConfig` with description `Select TUI theme`
- [x] add a theme selector dialog or reuse the existing selector dialog with a distinct dialog ID
- [x] populate selector items from the theme catalog or UI theme registry in sorted order
- [x] initialize selector cursor to the currently active theme
- [x] show enough item metadata to distinguish built-in and user themes if the existing selector supports subtitles
- [x] write tests that `/theme` is registered and opens a selector command
- [x] write tests that selector items include built-in and user themes
- [x] run command/dialog tests - must pass before task 4

### Task 4: Implement live preview, cancel restore, and confirm persistence
- [x] preview the highlighted theme immediately as the selector cursor changes
- [x] store the previously active theme when `/theme` opens
- [x] restore the previous theme when the selector is canceled with Esc
- [x] persist confirmed selection to `ui.theme` through `sdk.PreferenceStore` without overwriting other `ui` settings
- [x] notify the user when a theme is applied or when persistence fails
- [x] write tests for live preview changing the active theme
- [x] write tests for cancel restoring the original theme
- [x] write tests for confirm persisting `ui.theme` and preserving sibling UI settings
- [x] run selector persistence tests - must pass before task 5

### Task 5: Make active theme usage consistent
- [x] audit runtime `palette.DefaultTheme()` calls in components that should respect the active theme
- [x] update editor, footer, chat/message rendering, selector/dialog overlays, and attachment rendering to use `m.theme`/`m.styles` where appropriate
- [x] keep `palette.DefaultTheme()` only for fallback construction, tests, and code paths without model-provided styles
- [x] ensure `ThemeChangedMsg` and direct theme application update all dependent components consistently
- [x] write tests for at least one representative overlay using custom theme styles
- [x] write tests for editor/footer/message components retaining custom theme after theme changes
- [x] run component/model tests - must pass before task 6

### Task 6: Preserve custom theme semantics during dynamic state changes
- [ ] review `palette.AccentForState` and agent state handling that overwrites accent colors
- [ ] decide whether state changes should tint from the active theme or use existing hardcoded state accents
- [ ] implement the minimal behavior that avoids permanently erasing the selected theme after returning to idle
- [ ] write tests for agent state changes with a custom theme
- [ ] write tests that returning to idle restores the active theme accent values
- [ ] run palette/model state tests - must pass before task 7

### Task 7: Verify acceptance criteria
- [ ] verify JSON themes under `~/.weave/themes/*.json` are loaded
- [ ] verify `/theme` opens a selector and previews themes live
- [ ] verify Esc cancels and restores the previous theme
- [ ] verify Enter confirms and persists `ui.theme`
- [ ] verify selected `ui.theme` applies on next startup
- [ ] run `go test ./...` in the TUI extension
- [ ] run `gofmt`/`goimports` or project formatting command for changed files
- [ ] run linter if configured for the TUI extension

### Task 8: Update documentation
- [ ] document the JSON theme file location and schema in existing TUI docs or README if present
- [ ] document `/theme` usage and v1 limitations
- [ ] add a minimal example custom theme JSON
- [ ] run documentation-related checks if present

## Technical Details
- Theme file location: `~/.weave/themes/*.json`.
- Theme file format: full JSON object, no partial inheritance for v1.
- Theme identity: filename without `.json` should be the canonical theme name unless a `name` field is included and matches it.
- Built-ins: include `default`, plus a small set such as `dracula`, `catppuccin-mocha`, `nord`, and `gruvbox-dark` if their palettes are added in this change.
- Runtime flow:
  1. Build catalog.
  2. Register built-ins and user themes into `TUIImpl`.
  3. Apply configured theme or `default`.
  4. `/theme` opens selector.
  5. Selector movement previews theme.
  6. Confirm persists `ui.theme`.
  7. Cancel restores previous theme.
- Security:
  - Reject theme names with path separators, `.` or `..`, and control characters.
  - Only read regular files from the configured themes directory.
  - Do not execute theme content.

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- Create `~/.weave/themes/my-theme.json`, run the TUI, and verify `/theme` shows it.
- Preview several themes and confirm that Esc restores the original colors.
- Confirm a theme, restart Weave, and verify it remains active.
- Check visual contrast in common terminal themes.

**External system updates**:
- None for v1 unless downstream extensions document theme recommendations.
