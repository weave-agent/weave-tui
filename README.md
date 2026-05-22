# weave-tui

Terminal UI extension for [weave](https://github.com/weave-agent/weave) — an event-driven coding agent framework.

## Fork & Customize

1. Fork this repo
2. Edit the extension implementation
3. Install your fork: `weave install github.com/<you>/weave-tui --name tui`

The `--name tui` ensures your fork shadows the official extension.

## Install

```bash
weave install github.com/weave-agent/weave-tui@latest --name tui
```

## Use from Go extensions

Extensions can import the TUI extension API from a released module version:

```bash
go get github.com/weave-agent/weave-tui@latest
```

```go
import tui "github.com/weave-agent/weave-tui"
```

Releases are tagged with standard Go module versions, so downstream extensions can pin versions like `github.com/weave-agent/weave-tui v0.1.0` in `go.mod`.

## Public Go API

The root package is the only supported Go import path for downstream TUI extensions:

```go
import tui "github.com/weave-agent/weave-tui"
```

The root package intentionally exposes a small extension-author API:

- Runtime config: `TUIConfig`
- Extension registry: `RegisterTUIExtension`, `GetTUIExtension`, `ListTUIExtensions`, `TUIExtensionRegistered`, `ResetTUIExtensionRegistry`
- Extension contracts: `TUIExtension`, `TUIExtAPI`
- Panels: `PanelConfig`, `PanelPlacement`, `AsOverlay`, `AboveEditor`, `BelowEditor`, `TrayOnly`, `PanelDrawer`, `PanelTrayAPI`
- Rendering hooks: `RichToolRenderer`, `TUIComponent`
- Input and completion hooks: `KeyEvent`, `AutocompleteProvider`, `AutocompleteContext`, `AutocompleteSuggestion`
- Theme registration: `ThemeDef`

Rendering internals are private. Packages such as components, messages, overlays, attachments, palette, styles, and xchroma now live under `internal/` and are implementation details of this module. External extensions should use the root API plus the weave SDK interfaces for custom renderers, panels, header/footer components, autocomplete providers, and themes.

## Maintainer package map

- `.`: public API facade, TUI extension registration, and config aliases only
- `internal/contract`: canonical public API shapes aliased by the root package
- `internal/extensionregistry`: TUI extension factory registry used by the root facade
- `internal/app`: SDK extension lifecycle, terminal checks, Bubble Tea program ownership, and extension wiring
- `internal/bridge` and `internal/events`: bus-to-Bubble-Tea event translation and shared message structs
- `internal/model` and `internal/ui`: Bubble Tea model orchestration and SDK/TUI extension API implementation
- `internal/commands`, `internal/keybindings`, `internal/panels`, `internal/layout`: focused runtime state managers
- `internal/auth`, `internal/providers`, `internal/sessions`: provider, model, auth, and session helpers
- `internal/components`, `internal/components/*`, `internal/styles`, `internal/palette`, `internal/xchroma`: private rendering implementation

## Completion behavior

The editor supports completion popups for slash commands and file references:

- Slash command completions use fuzzy matching, so queries like `/hp` can match `/help`.
- File completions triggered with `@` show current-directory entries for empty or one-character queries.
- File queries with two or more characters use bounded recursive fuzzy search from the current working directory or typed directory.
- Recursive file results are ranked by relevance: exact filename matches appear first, followed by prefix matches, path segment matches, and then general fuzzy matches.
- Recursive search is bounded to a maximum depth of 4 directories and 2000 items for performance.
- Hidden files and directories are skipped.
- Use Up/Down to move through suggestions, Escape to dismiss, and Tab to accept the selected completion.
- Enter accepts the selected completion and immediately submits the message.

## Development

```bash
git clone git@github.com:weave-agent/weave-tui.git
cd weave-tui

# Add temporary replace for local SDK (don't commit this)
echo 'replace github.com/weave-agent/weave => /path/to/local/weave' >> go.mod

go test ./...
go doc .
```

## License

Same as the main weave project.
