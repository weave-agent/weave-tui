package styles

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"

	"github.com/weave-agent/weave-tui/internal/palette"
)

// --- Glyph constants ---

func TestGlyphConstants(t *testing.T) {
	assert.Equal(t, "❯", UserMarker)
	assert.Equal(t, "◆", AssistantMarker)
	assert.Equal(t, "∴", ThinkingMarker)
	assert.Equal(t, "○", ToolPending)
	assert.Equal(t, "✓", ToolSuccess)
	assert.Equal(t, "×", ToolError)
	assert.Equal(t, "■", ToolInterrupted)
}

// --- Constructor ---

func TestNew_WithNilTheme_FallsBackToDefault(t *testing.T) {
	s := New(nil)
	assert.NotNil(t, s)
	assert.NotNil(t, s.Theme())
	assert.Equal(t, palette.DefaultTheme().Foreground, s.Theme().Foreground)
}

func TestNew_WithCustomTheme(t *testing.T) {
	custom := &palette.Theme{
		Foreground: "99",
		Accent:     "88",
	}
	s := New(custom)
	assert.Equal(t, custom, s.Theme())
	assert.Equal(t, "99", s.Theme().Foreground)
}

// --- Role markers ---

func TestUserMarkerRendered_ContainsGlyph(t *testing.T) {
	s := New(palette.DefaultTheme())
	rendered := s.UserMarkerRendered()
	assert.Contains(t, rendered, UserMarker)
}

func TestAssistantMarkerRendered_ContainsGlyph(t *testing.T) {
	s := New(palette.DefaultTheme())
	rendered := s.AssistantMarkerRendered()
	assert.Contains(t, rendered, AssistantMarker)
}

func TestThinkingMarkerRendered_ContainsGlyph(t *testing.T) {
	s := New(palette.DefaultTheme())
	rendered := s.ThinkingMarkerRendered()
	assert.Contains(t, rendered, ThinkingMarker)
}

// --- Theme usage verification ---

func TestStyleHelpers_UseProvidedTheme_NotDefault(t *testing.T) {
	custom := &palette.Theme{
		Foreground:     "99",
		ForegroundDim:  "98",
		Muted:          "97",
		MutedBright:    "96",
		Accent:         "95",
		AccentDim:      "94",
		AccentBright:   "93",
		Border:         "92",
		BorderFocused:  "91",
		Success:        "90",
		Error:          "89",
		Warning:        "88",
		BackgroundTint: "87",
	}

	s := New(custom)

	// Verify that styles carry the custom theme's colors by checking
	// the rendered output contains ANSI escape sequences.
	assertCustomColor(t, s.UserMarkerRendered(), custom.Foreground)
	assertCustomColor(t, s.Foreground().Render("x"), custom.Foreground)
	assertCustomColor(t, s.ForegroundDim().Render("x"), custom.ForegroundDim)
	assertCustomColor(t, s.Muted().Render("x"), custom.Muted)
	assertCustomColor(t, s.MutedBright().Render("x"), custom.MutedBright)
	assertCustomColor(t, s.Accent().Render("x"), custom.Accent)
	assertCustomColor(t, s.AccentDim().Render("x"), custom.AccentDim)
	assertCustomColor(t, s.AccentBright().Render("x"), custom.AccentBright)
	assertCustomColor(t, s.Success().Render("x"), custom.Success)
	assertCustomColor(t, s.Error().Render("x"), custom.Error)
	assertCustomColor(t, s.Warning().Render("x"), custom.Warning)
}

func TestSelectedRow_UsesProvidedTheme(t *testing.T) {
	custom := &palette.Theme{
		Accent:     "42",
		Foreground: "43",
	}
	s := New(custom)
	rendered := s.SelectedRow().Render("text")
	assertCustomColor(t, rendered, custom.Accent)
	assertCustomColor(t, rendered, custom.Foreground)
}

func TestTabStyles_UsesProvidedTheme(t *testing.T) {
	custom := &palette.Theme{
		AccentBright:   "55",
		Accent:         "54",
		Muted:          "53",
		BackgroundTint: "52",
	}
	s := New(custom)

	assertCustomColor(t, s.FocusedTab().Render("x"), custom.AccentBright)
	assertCustomColor(t, s.ActiveTab().Render("x"), custom.Accent)
	assertCustomColor(t, s.InactiveTab().Render("x"), custom.Muted)
}

func TestPillStyles_UsesProvidedTheme(t *testing.T) {
	custom := &palette.Theme{
		BackgroundTint: "66",
		Accent:         "67",
		Error:          "68",
		Muted:          "69",
	}
	s := New(custom)

	assertCustomColor(t, s.Pill().Render("x"), custom.Accent)
	assertCustomColor(t, s.PillError().Render("x"), custom.Error)
	assertCustomColor(t, s.PillMuted().Render("x"), custom.Muted)
}

func TestBannerStyles_UsesProvidedTheme(t *testing.T) {
	custom := &palette.Theme{
		AccentBright:   "77",
		Success:        "78",
		Warning:        "79",
		Error:          "80",
		BackgroundTint: "81",
	}
	s := New(custom)

	assertCustomColor(t, s.BannerInfo().Render("x"), custom.AccentBright)
	assertCustomColor(t, s.BannerSuccess().Render("x"), custom.Success)
	assertCustomColor(t, s.BannerWarning().Render("x"), custom.Warning)
	assertCustomColor(t, s.BannerError().Render("x"), custom.Error)
}

func TestToolStateColors_UsesProvidedTheme(t *testing.T) {
	custom := &palette.Theme{
		AccentDim: "33",
		Border:    "34",
		Success:   "35",
		Error:     "36",
		Muted:     "37",
	}
	s := New(custom)

	assert.Equal(t, custom.AccentDim, s.ToolPendingColor())
	assert.Equal(t, custom.Border, s.ToolSuccessColor())
	assert.Equal(t, custom.Success, s.ToolSuccessFlashedColor())
	assert.Equal(t, custom.Error, s.ToolErrorColor())
	assert.Equal(t, custom.Muted, s.ToolInterruptedColor())
}

func TestOverlayBorder_UsesProvidedTheme(t *testing.T) {
	custom := &palette.Theme{
		Accent: "44",
	}
	s := New(custom)
	rendered := s.OverlayBorder().Render("content")
	assertCustomColor(t, rendered, custom.Accent)
	assert.Contains(t, rendered, "content")
}

func TestBackgroundTintStyles_UsesProvidedTheme(t *testing.T) {
	custom := &palette.Theme{
		BackgroundTint:  "70",
		BackgroundTint2: "71",
	}
	s := New(custom)

	assertCustomColor(t, s.BackgroundTint().Render("x"), custom.BackgroundTint)
	assertCustomColor(t, s.BackgroundTint2().Render("x"), custom.BackgroundTint2)
}

func TestBorderStyles_UsesProvidedTheme(t *testing.T) {
	custom := &palette.Theme{
		Border:        "60",
		BorderFocused: "61",
	}
	s := New(custom)

	// Border styles need an actual border to emit ANSI sequences.
	bordered := s.Border().Border(lipgloss.NormalBorder()).Render("x")
	borderedFocused := s.BorderFocused().Border(lipgloss.NormalBorder()).Render("x")

	assertCustomColor(t, bordered, custom.Border)
	assertCustomColor(t, borderedFocused, custom.BorderFocused)
}

// assertCustomColor checks that a rendered string contains the ANSI 256-color
// SGR parameter for the given color code. Lipgloss may combine multiple
// parameters into one escape sequence (e.g. \x1b[1;38;5;43;48;5;42m), so we
// look for the parameter pattern itself rather than an exact sequence.
func assertCustomColor(t *testing.T, rendered, colorCode string) {
	t.Helper()
	// Look for either foreground (38;5;{code}) or background (48;5;{code}) parameter.
	fgParam := "38;5;" + colorCode
	bgParam := "48;5;" + colorCode
	assert.True(t, strings.Contains(rendered, fgParam) || strings.Contains(rendered, bgParam),
		"expected rendered output to contain ANSI color parameter for %s, got: %q", colorCode, rendered)
}
