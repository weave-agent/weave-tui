package messages

import (
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/charmbracelet/glamour"
	glamouransi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"

	"github.com/weave-agent/weave-tui/internal/palette"

	"github.com/weave-agent/weave-tui/internal/xchroma"
)

const chromaFormatterName = "weave"

var registerFormatterOnce sync.Once

func registerFormatter() {
	registerFormatterOnce.Do(func() {
		formatters.Register(chromaFormatterName, xchroma.NewFormatter())
	})
}

// MarkdownRenderer wraps a glamour renderer for terminal-aware markdown rendering.
type MarkdownRenderer struct {
	mu       sync.Mutex
	renderer *glamour.TermRenderer
	width    int
	theme    *palette.Theme
}

// NewMarkdownRenderer creates a new markdown renderer with the given width.
func NewMarkdownRenderer(width int) *MarkdownRenderer {
	registerFormatter()

	r := &MarkdownRenderer{width: width, theme: palette.DefaultTheme()}
	r.rebuild()

	return r
}

// SetTheme updates the colors used for markdown rendering.
func (r *MarkdownRenderer) SetTheme(theme *palette.Theme) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if theme == nil {
		theme = palette.DefaultTheme()
	}

	t := *theme
	r.theme = &t
	r.rebuild()
}

// SetWidth updates the word-wrap width and rebuilds the renderer.
func (r *MarkdownRenderer) SetWidth(width int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.width != width {
		r.width = width
		r.rebuild()
	}
}

// Render converts markdown text to styled terminal output.
// If rendering fails, returns the plain text.
func (r *MarkdownRenderer) Render(text string) string {
	r.mu.Lock()
	renderer := r.renderer
	r.mu.Unlock()

	if renderer == nil {
		return text
	}

	out, err := renderer.Render(text)
	if err != nil {
		return text
	}

	return strings.TrimSpace(out)
}

func markdownStyle(theme *palette.Theme) glamouransi.StyleConfig {
	defaultTheme := palette.DefaultTheme()
	if theme == nil {
		theme = defaultTheme
	}

	style := styles.DarkStyleConfig
	fg := themeColor(theme.Foreground, defaultTheme.Foreground)
	fgDim := themeColor(theme.ForegroundDim, defaultTheme.ForegroundDim)
	fgBright := themeColor(theme.ForegroundBright, defaultTheme.ForegroundBright)
	background := themeColor(theme.Background, defaultTheme.Background)
	bg := themeColor(theme.BackgroundTint, defaultTheme.BackgroundTint)
	bg2 := themeColor(theme.BackgroundTint2, defaultTheme.BackgroundTint2)
	border := themeColor(theme.Border, defaultTheme.Border)
	accent := themeColor(theme.Accent, defaultTheme.Accent)

	style.Document.Color = &fg
	style.Paragraph.Color = &fg
	style.Text.Color = &fg
	style.Heading.Color = &accent
	style.H1.Color = &accent
	style.H2.Color = &accent
	style.H3.Color = &accent
	style.H4.Color = &fgBright
	style.H5.Color = &fgBright
	style.H6.Color = &fgBright
	style.Strong.Color = &fgBright
	style.Emph.Color = &fgDim
	style.HorizontalRule.Color = &border
	style.Item.Color = &accent
	style.Enumeration.Color = &accent
	style.Link.Color = &accent
	style.LinkText.Color = &accent
	style.BlockQuote.Color = &fgDim
	style.BlockQuote.BackgroundColor = &bg
	style.Code.Color = &fgBright
	style.Code.BackgroundColor = &bg
	style.CodeBlock.Color = &fg
	style.CodeBlock.BackgroundColor = &bg2
	style.Table.Color = &fg
	style.Table.BackgroundColor = &background
	style.Task.Color = &fg
	style.H2.Prefix = ""
	style.H3.Prefix = ""
	style.H4.Prefix = ""
	style.H5.Prefix = ""
	style.H6.Prefix = ""

	return style
}

func themeColor(value, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}

// rebuild recreates the glamour renderer with the current width.
func (r *MarkdownRenderer) rebuild() {
	opts := []glamour.TermRendererOption{
		glamour.WithStyles(markdownStyle(r.theme)),
		glamour.WithChromaFormatter(chromaFormatterName),
	}
	if r.width > 0 {
		opts = append(opts, glamour.WithWordWrap(r.width))
	}

	renderer, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		r.renderer = nil
		return
	}

	r.renderer = renderer
}
