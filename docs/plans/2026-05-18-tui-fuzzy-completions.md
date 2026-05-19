# TUI Fuzzy Completions

## Overview
Upgrade the TUI editor completion popup with fuzzy matching for slash commands and file paths.

Scope:
- Slash command completions use fuzzy search instead of prefix-only filtering.
- File completions use hybrid search: current directory for empty/1-character queries, recursive fuzzy search for 2+ character queries.
- Tab accepts the selected completion instead of cycling the list.
- No inline ghost text.

## Context
- **TUI extension**: `~/.weave/extensions/tui`
- **Completion model**: `components/completion.go` — currently filters items by case-insensitive prefix.
- **Path completion provider**: `components/path_completion.go` — currently reads one directory level and filters by prefix.
- **Editor key handling**: `components/editor.go` — currently Tab moves completion cursor down, Enter applies and submits.
- **Completion trigger wiring**: `model.go` — refreshes command/file completions after editor input.
- **Reference**: Crush (`/Users/andrey/Projects/crush/internal/ui/completions/`) — uses `github.com/sahilm/fuzzy`, recursive file loading, and tiered file ranking.

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Make small, focused changes
- Every task includes new/updated tests
- All tests must pass before starting next task
- Maintain backward compatibility except for intentional Tab completion behavior change

## Testing Strategy
- **Unit tests**: required for fuzzy filtering, path completion, and completion key behavior
- No E2E tests in this project — TUI components tested via unit tests
- Run extension tests from `~/.weave/extensions/tui`

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix

## Implementation Steps

### Task 1: Add fuzzy dependency
- [x] add `github.com/sahilm/fuzzy` to `~/.weave/extensions/tui/go.mod`
- [x] run `cd ~/.weave/extensions/tui && go mod tidy`
- [x] verify no unrelated module changes

### Task 2: Fuzzy-filter completion items
- [x] update `components/completion.go` so `CompletionModel.SetFilter()` uses fuzzy matching when filter is non-empty
- [x] preserve current behavior for empty filters: show all items in original order
- [x] match against completion labels
- [x] reset cursor to first filtered item after filter changes
- [x] keep popup rendering unchanged; no match highlighting in this pass
- [x] add/update tests for fuzzy command matching, empty filter behavior, no-match behavior, and cursor reset
- [x] run `cd ~/.weave/extensions/tui && go test ./components/...`

### Task 3: Implement hybrid path completion
- [x] update `components/path_completion.go` so empty/1-character queries use current directory completion
- [x] add recursive file collection for 2+ character queries
- [x] skip hidden files/directories to match current behavior
- [x] keep directory values with trailing `/`
- [x] use relative slash-separated paths in `CompletionItem.Value`
- [x] apply hard caps for recursive search depth and item count
- [x] ignore unreadable directories without surfacing UI errors
- [x] rank recursive fuzzy results with tiers: exact basename/stem, basename prefix, exact path segment, fallback fuzzy
- [x] add tests for current-directory short query, recursive fuzzy match, ranking preference, directory trailing slash, hidden path skip, and cap behavior
- [x] run `cd ~/.weave/extensions/tui && go test ./components/...`

### Task 4: Change Tab completion behavior
- [x] update `components/editor.go` so Tab applies the selected completion when completion is visible
- [x] keep Up/Down navigation behavior unchanged
- [x] keep Escape dismiss behavior unchanged
- [x] keep Enter behavior unchanged unless tests expose an existing inconsistency
- [x] update tests that expected Tab to move selection
- [x] add test verifying Tab fills selected completion and hides popup
- [x] run `cd ~/.weave/extensions/tui && go test ./components/...`

### Task 5: Verify command and file trigger integration
- [x] confirm slash command completion still opens for `/` at start of prompt
- [x] confirm file completion still opens for `@` after whitespace
- [x] confirm file-accepting slash commands still use path completions after command text
- [x] add/update model-level tests if fuzzy/hybrid behavior changes expected filtered results
- [x] run `cd ~/.weave/extensions/tui && go test ./...`

### Task 6: Acceptance verification
- [ ] run `cd ~/.weave/extensions/tui && go test ./...`
- [ ] run root tests if touched root-module files: `cd /Users/andrey/Projects/weave && make test`
- [ ] manual test: type `/hp` and verify `/help` appears
- [ ] manual test: type `/hp`, press Tab, verify `/help ` is inserted
- [ ] manual test: type `@` and verify current directory entries appear
- [ ] manual test: type a 2+ character fuzzy file query and verify recursive matches appear
- [ ] manual test: Up/Down navigates suggestions and Tab accepts selected item
- [ ] manual test: Esc dismisses popup

## Technical Details

**Completion filtering:**
- Empty filter: preserve original list order.
- Non-empty filter: use `github.com/sahilm/fuzzy` against `CompletionItem.Label`.
- Keep the existing `CompletionItem` shape and rendering.

**Path completion hybrid behavior:**
- Query length 0-1: read only the relevant current/nested parent directory, preserving existing shell-like path completion.
- Query length 2+: recursively walk from the base directory, bounded by fixed depth and item caps.
- Recursive values should be relative to base directory and use `/` separators.

**Tab behavior:**
- When completion is visible, Tab applies the selected item.
- Tab no longer cycles suggestions.
- Selection movement remains on Up/Down.

## Post-Completion

**Manual verification:**
- Verify no regressions in editor submission, history navigation, multiline input, or global keybindings.
- Verify large repositories do not noticeably freeze the editor for typical queries.
- Verify recursive file search does not include hidden directories like `.git`.

**Future enhancements (not in scope):**
- Inline ghost preview text.
- Highlight fuzzy match spans in popup rows.
- Async recursive file indexing/cache.
- Configurable completion depth/item limits.
