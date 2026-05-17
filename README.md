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
