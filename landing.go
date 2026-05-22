package tui

import (
	"fmt"
	"strings"

	"github.com/weave-agent/weave-tui/internal/palette"
	"github.com/weave-agent/weave-tui/internal/styles"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

const (
	landingLabelWidth   = 12
	landingContentWidth = 56
)

// LandingModel renders a landing screen shown before the first prompt.
// It displays a minimal title, current model/provider info, loaded extensions,
// and keybinding hints in a boot/status layout.
type LandingModel struct {
	model      string
	provider   string
	extensions []string
	width      int
	height     int
	styles     *styles.Styles
}

// NewLandingModel creates a landing model with the given model, provider, and extensions.
func NewLandingModel(model, provider string, extensions []string) LandingModel {
	return LandingModel{
		model:      model,
		provider:   provider,
		extensions: extensions,
		styles:     styles.New(palette.DefaultTheme()),
	}
}

// SetStyles sets the style set used for rendering.
func (m LandingModel) SetStyles(s *styles.Styles) LandingModel {
	m.styles = s
	return m
}

// SetSize updates the landing model's available dimensions.
func (m LandingModel) SetSize(width, height int) LandingModel {
	m.width = width
	m.height = height

	return m
}

// Draw renders the landing screen into the given screen buffer area.
func (m LandingModel) Draw(scr uv.Screen, area uv.Rectangle, theme *palette.Theme) {
	if area.Dx() <= 0 || area.Dy() <= 0 {
		return
	}

	var s *styles.Styles

	switch {
	case theme != nil:
		s = styles.New(theme)
	case m.styles != nil:
		s = m.styles
	default:
		s = styles.New(palette.DefaultTheme())
	}

	w := area.Dx()
	lines := m.buildLines()

	// Vertically center if there's room
	y := area.Min.Y
	if area.Dy() > len(lines) {
		y = area.Min.Y + (area.Dy()-len(lines))/2
	}

	accentStyle := s.Accent().Bold(true)
	accentBrightStyle := s.AccentBright()
	mutedStyle := s.Muted()
	mutedBrightStyle := s.MutedBright()

	for i, line := range lines {
		if y+i >= area.Max.Y {
			break
		}

		var rendered string

		switch {
		case strings.HasPrefix(line, "logo:"):
			rendered = accentBrightStyle.Render(strings.TrimPrefix(line, "logo:"))
		case strings.HasPrefix(line, "title:"):
			rendered = accentBrightStyle.Render(strings.TrimPrefix(line, "title:"))
		case strings.HasPrefix(line, "kv:"):
			parts := strings.SplitN(strings.TrimPrefix(line, "kv:"), "|", 2)
			if len(parts) == 2 {
				rendered = mutedStyle.Render(parts[0]) + " " + accentStyle.Render(parts[1])
			} else {
				rendered = mutedStyle.Render(parts[0])
			}
		case strings.HasPrefix(line, "hint:"):
			rendered = mutedStyle.Render(strings.TrimPrefix(line, "hint:"))
		case strings.HasPrefix(line, "rule:"):
			ruleWidth := min(w, m.contentWidth())
			if ruleWidth > 0 {
				rendered = mutedStyle.Render(strings.Repeat("─", ruleWidth))
			}
		case strings.HasPrefix(line, "muted:"):
			rendered = mutedBrightStyle.Render(strings.TrimPrefix(line, "muted:"))
		default:
			rendered = line
		}

		r := uv.Rect(area.Min.X, y+i, w, 1)
		uv.NewStyledString(rendered).Draw(scr, r)
	}
}

func (m LandingModel) buildLines() []string {
	lines := append([]string{}, m.logo()...)

	if m.model != "" {
		lines = append(lines, fmt.Sprintf("kv:%-*s|%s", landingLabelWidth, "Model", m.model))
	}

	if m.provider != "" {
		lines = append(lines, fmt.Sprintf("kv:%-*s|%s", landingLabelWidth, "Provider", m.provider))
	}

	if len(m.extensions) > 0 {
		lines = append(lines, "", "rule:", fmt.Sprintf("kv:%-*s|", landingLabelWidth, "Extensions"))
		for _, extLine := range wrapList(m.extensions, m.contentWidth()) {
			lines = append(lines, "muted:"+extLine)
		}
	}

	lines = append(
		lines,
		"",
		"hint:  ctrl+p model  ·  ctrl+l select  ·  shift+tab thinking",
		"hint:  ctrl+n new    ·  ctrl+o expand  ·  ctrl+t toggle",
	)

	return lines
}

// wrapList formats a list of items as comma-separated strings, wrapping when
// the line exceeds the available width. Each continuation line is prefixed with 4 spaces.
func wrapList(items []string, width int) []string {
	if len(items) == 0 {
		return nil
	}

	const prefix = "    "

	if width <= 0 {
		return []string{prefix + strings.Join(items, ", ")}
	}

	var (
		lines []string
		b     strings.Builder
	)

	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(item)
	}

	text := b.String()
	maxLine := max(width-lipgloss.Width(prefix), 10)

	for text != "" {
		if len(text) <= maxLine {
			lines = append(lines, prefix+text)
			break
		}

		cut := maxLine
		for cut > 0 && text[cut] != ' ' && text[cut] != ',' {
			cut--
		}

		if cut == 0 {
			cut = maxLine
		}

		lines = append(lines, prefix+text[:cut])
		text = strings.TrimLeft(text[cut:], " ")
	}

	return lines
}

func (m LandingModel) contentWidth() int {
	if m.width > 0 {
		return min(m.width, landingContentWidth)
	}

	return landingContentWidth
}

func (m LandingModel) logo() []string {
	return []string{
		"",
		"logo: █████ ███ █████  ██████   ██████   █████ █████  ██████ ",
		"logo:░░███ ░███░░███  ███░░███ ░░░░░███ ░░███ ░░███  ███░░███",
		"logo: ░███ ░███ ░███ ░███████   ███████  ░███  ░███ ░███████ ",
		"logo: ░░███████████  ░███░░░   ███░░███  ░░███ ███  ░███░░░  ",
		"logo:  ░░████░████   ░░██████ ░░████████  ░░█████   ░░██████ ",
		"logo:   ░░░░ ░░░░     ░░░░░░   ░░░░░░░░    ░░░░░     ░░░░░░ ",
		"",
	}
}
