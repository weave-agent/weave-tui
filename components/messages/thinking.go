package messages

import (
	"strings"
	"unicode/utf8"

	"github.com/weave-agent/weave-tui/internal/palette"
	"github.com/weave-agent/weave-tui/internal/styles"

	uv "github.com/charmbracelet/ultraviolet"
)

// ThinkingBlock renders a thinking section, always shown.
type ThinkingBlock struct {
	content string
	styles  *styles.Styles
}

// NewThinkingBlock creates a new thinking block.
func NewThinkingBlock(content string) *ThinkingBlock {
	return &ThinkingBlock{
		content: content,
		styles:  styles.New(palette.DefaultTheme()),
	}
}

// SetStyles sets the style set used for rendering.
func (b *ThinkingBlock) SetStyles(s *styles.Styles) {
	b.styles = s
}

// Content returns the thinking content.
func (b *ThinkingBlock) Content() string {
	return b.content
}

// View renders the thinking block.
func (b *ThinkingBlock) View(width int) string {
	if width <= 0 {
		width = 80
	}

	lines := strings.Split(b.content, "\n")

	var bldr strings.Builder
	bldr.WriteString(b.styles.ThinkingMarkerRendered() + " Thinking…")
	bldr.WriteString("\n")

	contentStyle := b.styles.ForegroundDim().Width(width)

	for _, line := range lines {
		bldr.WriteString(contentStyle.Render(line))
		bldr.WriteString("\n")
	}

	return strings.TrimRight(bldr.String(), "\n")
}

// Draw renders the thinking block into a screen buffer region.
func (b *ThinkingBlock) Draw(scr uv.Screen, area uv.Rectangle) {
	drawView(scr, area, b.View(area.Dx()))
}

// Summary returns a short preview of the thinking content.
func (b *ThinkingBlock) Summary(maxLen int) string {
	first := strings.SplitN(b.content, "\n", 2)[0]
	if utf8.RuneCountInString(first) > maxLen {
		runes := []rune(first)
		return string(runes[:maxLen-3]) + "..."
	}

	if first == "" {
		return "(empty)"
	}

	return first
}

// LineCount returns the number of lines in the thinking content.
func (b *ThinkingBlock) LineCount() int {
	if b.content == "" {
		return 0
	}

	return len(strings.Split(b.content, "\n"))
}
