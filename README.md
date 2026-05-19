# weave-tui

Terminal UI extension for [weave](https://github.com/weave-agent/weave) — an event-driven coding agent framework.

## Fork & Customize

1. Fork this repo
2. Edit the extension implementation
3. Install your fork: `weave install github.com/<you>/weave-tui --name tui`

The `--name tui` ensures your fork shadows the official extension.

## Install

```bash
weave install github.com/weave-agent/weave-tui --name tui
```

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
```

## License

Same as the main weave project.
