# Structured TUI Design Language

## Overview
- Introduce a structured visual design language for the Weave TUI while preserving its existing graphite, terminal-native identity.
- Solve current inconsistencies where components independently define glyphs, borders, spacing, selection states, status labels, and theme usage.
- Establish a thin style-set layer between `palette.Theme` and rendering components so future custom themes can change colors only, while product grammar remains consistent.
- Apply the first pass to transcript messages, tool panels, status banners, landing screen, and focus/selection states.

## Context (from discovery)
- Files/components involved:
  - `palette/theme.go` — semantic color tokens.
  - `palette/state.go` — agent-state accent colors.
  - `components/messages/user.go` — user transcript rendering.
  - `components/messages/assistant.go` — assistant transcript rendering.
  - `components/messages/thinking.go` — thinking block rendering.
  - `components/messages/notification.go` — current notification transcript rendering.
  - `components/messages/tool.go` — tool panel rendering and tool state glyphs.
  - `components/chat.go` — transcript layout, spacing, and scroll indicators.
  - `components/spinner.go` — working spinner and status text.
  - `components/attachments/attachments.go` — attachment pills.
  - `components/completion.go` — completion popup selected-row behavior.
  - `components/overlays/selector.go` — selector overlay selected-row behavior.
  - `panel_tray.go` — tab focus behavior.
  - `landing.go` — current landing/splash screen.
  - `model.go` — layout drawing, pills/status area, overlays, footer, notifications.
- Related patterns found:
  - The TUI already uses semantic color slots and dynamic agent-state accents.
  - Components frequently call `palette.DefaultTheme()` directly, which blocks consistent active-theme propagation.
  - Message role glyphs and status markers exist but are not governed by one shared grammar.
  - Tool panels already use a lightweight horizontal-rule treatment and should stay minimal rather than becoming heavy cards.
- Dependencies identified:
  - Bubble Tea/Bubbles v2 rendering path.
  - Lipgloss v2 styles.
  - Ultraviolet screen buffers.
  - Existing adjacent Go render tests.

## Development Approach
- **Testing approach**: Regular — implement small focused changes, then add/update focused render tests in the same task.
- Complete each task fully before moving to the next.
- Make small, focused changes.
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task.
  - tests are not optional - they are a required part of the checklist
  - write unit tests for new functions/methods
  - write unit tests for modified functions/methods
  - add new test cases for new code paths
  - update existing test cases if behavior changes
  - tests cover both success and edge scenarios
- **CRITICAL: all tests must pass before starting next task** - no exceptions.
- **CRITICAL: update this plan file when scope changes during implementation**.
- Run tests after each change.
- Maintain backward compatibility for extension APIs unless intentionally changing UI rendering behavior.

## Testing Strategy
- **Unit/render tests**: required for every task.
- Add focused tests near changed packages:
  - `components/messages/*_test.go` for transcript/tool changes.
  - `components/*_test.go` for spinner/completion/chat/banner behavior.
  - root package tests for model, landing, panel tray, and notification routing.
- Use `go test ./...` as final verification.
- No browser/e2e test path is expected for this terminal UI.

## Progress Tracking
- Mark completed items with `[x]` immediately when done.
- Add newly discovered tasks with ➕ prefix.
- Document issues/blockers with ⚠️ prefix.
- Update plan if implementation deviates from original scope.
- Keep plan in sync with actual work done.

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code changes, tests, and documentation updates achievable within this repository.
- **Post-Completion** (no checkboxes): manual verification and external checks.
- **Checkbox placement**: Checkboxes belong only in Task sections. Do not put checkboxes in Success criteria, Overview, or Context.

## Implementation Steps

### Task 1: Add style-set foundation
- [x] add a small style package or root-level style module that creates `styles.New(theme)` from `palette.Theme`
- [x] define fixed design grammar constants for `❯`, `◆`, `∴`, `○`, `✓`, `×`, and `■`
- [x] add reusable style helpers for role markers, muted text, pills, selected rows, tool state colors, and overlay boxes
- [x] ensure custom themes are treated as color-token changes only; glyphs, spacing, border shapes, and layout grammar stay fixed in code
- [x] write tests for style-set defaults and glyph constants
- [x] write tests proving style helpers use the provided theme rather than `palette.DefaultTheme()`
- [x] run `go test ./...` - must pass before task 2

### Task 2: Migrate transcript role primitives
- [x] update `components/messages/user.go` so user messages use `❯` only on the first line
- [x] update user continuation lines to align under message content without repeating the marker
- [x] migrate assistant marker styling in `components/messages/assistant.go` to the style set while keeping `◆`
- [x] migrate thinking marker styling in `components/messages/thinking.go` to the style set while keeping `∴ Thinking…`
- [x] normalize human status copy touched by these components to sentence case and unicode ellipsis where ongoing
- [x] write/update user message tests for single-line, multi-line, and skill-invocation rendering
- [x] write/update assistant and thinking render tests for marker and active-theme behavior
- [x] run `go test ./...` - must pass before task 3

### Task 3: Migrate tool panel grammar
- [x] update `components/messages/tool.go` to use style-set tool glyphs: pending `○`, success `✓`, error `×`, interrupted `■`, running spinner unchanged
- [x] keep tool panels as minimal horizontal-rule blocks, not full cards
- [x] reduce tool header/body spacing to one blank line maximum
- [x] normalize tool status copy to human labels such as `Running…`, `Interrupted`, and `No output`
- [x] migrate tool border/body coloring to style-set helpers using the active theme
- [x] write/update tool panel tests for pending, running, success, error, interrupted, collapsed output, and spacing
- [x] write/update tests proving tool rendering uses the provided theme path where applicable
- [x] run `go test ./...` - must pass before task 4

### Task 4: Add status banner grammar for notifications
- [x] introduce a status banner/pill representation for UI notifications in the model layer
- [x] render notification banners in the existing pills/status area instead of appending ordinary UI notifications to the transcript by default
- [x] use marker grammar: info `i`, success `✓`, warning `!`, error `×`
- [x] implement persistence rules: success/info are ephemeral; warning/error persist until next user action or dismissal
- [x] keep transcript notification rendering available only for notifications that are semantically part of the conversation or required by extension APIs
- [x] write tests for banner rendering, marker/color selection, ephemeral behavior, and persistent warning/error behavior
- [x] write/update model tests for notification routing and dismissal-on-user-action behavior
- [x] run `go test ./...` - must pass before task 5

### Task 5: Redesign landing as boot/status screen
- [x] update `landing.go` to reduce splash/logo dominance and render a boot/status layout
- [x] render label/value rows for model, provider, and extensions using muted labels and accent values
- [x] keep shortcut hints quiet and functional using the same muted style grammar as the rest of the UI
- [x] ensure landing uses the active theme through the style set rather than pulling default colors directly
- [x] write/update landing tests for layout, labels, extension wrapping, and theme use
- [x] write tests for narrow/short terminal behavior if existing coverage does not already cover it
- [x] run `go test ./...` - must pass before task 6

### Task 6: Unify focus and selection states
- [ ] migrate `components/completion.go` selected rows to the shared selected-row style helper
- [ ] migrate `components/overlays/selector.go` selected rows to the same list-selection grammar
- [ ] keep panel tray tabs bracketed when focused and migrate colors to the style set
- [ ] keep editor/input focus expressed through accent border; do not replace it with row-style selection
- [ ] keep footer/model status as accent foreground only, not a selected control treatment
- [ ] write/update completion tests for selected row rendering and truncation
- [ ] write/update selector and panel tray tests for focus grammar
- [ ] run `go test ./...` - must pass before task 7

### Task 7: Remove direct default-theme usage from migrated render paths
- [ ] audit migrated components for remaining `palette.DefaultTheme()` calls inside render methods
- [ ] replace direct default-theme calls with active theme/style-set usage where the component is part of the structured design language
- [ ] keep `palette.DefaultTheme()` only at construction/default boundaries where an active theme is legitimately unavailable
- [ ] write/update tests that pass a non-default theme and verify visible render differences
- [ ] run `go test ./...` - must pass before task 8

### Task 8: Verify acceptance criteria
- [ ] verify user messages render with `❯` on the first line only
- [ ] verify assistant and thinking markers remain `◆` and `∴`
- [ ] verify tool panels remain horizontal-rule blocks with the new glyph/status grammar
- [ ] verify notifications render as banners/pills according to severity persistence rules
- [ ] verify landing reads as a boot/status screen rather than a splash screen
- [ ] verify list, tab, editor, and footer focus states follow their type-specific grammar
- [ ] run full test suite with `go test ./...`
- [ ] run `gofmt -w` on changed Go files

### Task 9: Update documentation if needed
- [ ] update README.md only if user-facing theme or UI behavior documentation is currently present and becomes inaccurate
- [ ] update project docs only if the new style-set package needs explanation for extension authors or future maintainers
- [ ] run `go test ./...` after documentation-adjacent changes if any Go files changed

## Technical Details
- `palette.Theme` remains the raw color-token structure.
- A new style set maps theme tokens into product-specific render grammar.
- Custom themes are planned as color-only token changes:
  - foregrounds
  - muted colors
  - surfaces
  - borders
  - accent colors
  - success/warning/error colors
- Glyphs and component grammar are fixed by product code:
  - user marker: `❯`
  - assistant marker: `◆`
  - thinking marker: `∴`
  - tool pending: `○`
  - tool success: `✓`
  - tool error: `×`
  - tool interrupted: `■`
- Focus grammar:
  - lists: accent background row
  - tabs: bracketed active tab
  - editor/input: accent border
  - footer/model status: accent foreground only
- Motion/state grammar:
  - spinner/status pill carries semantic activity state
  - editor border pulse carries subtle ambient activity
  - accent colors mean current agent state, not decoration
- Spacing grammar:
  - moderately breathable transcript
  - one-row separation between transcript items
  - no chat bubbles
  - no heavy inline tool cards
  - one blank line maximum between tool header and body

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- Run the TUI locally and visually inspect a normal chat turn, multi-line user prompt, thinking block, assistant response, tool execution, warning/error notification, completion popup, selector overlay, panel tray, and landing screen.
- Verify the UI feels like a cohesive graphite/status instrument panel: calm idle state, cyan streaming state, amber tool-running state, red error state.
- Verify warning/error banners do not disappear too quickly and success/info banners do not clutter the interface.

**Future work**:
- Add color-only custom theme configuration after the style-set migration is stable.
- Consider a follow-up pass to document theme token meanings for theme authors.
