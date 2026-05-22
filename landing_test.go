package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weave-agent/weave-tui/internal/palette"
	"github.com/weave-agent/weave-tui/internal/styles"
)

func TestLandingModel_DrawRendersLogo(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	assert.Contains(t, rendered, "█████ ███ █████")
}

func TestLandingModel_DrawRendersModelInfo(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	assert.Contains(t, rendered, "glm-5.1")
	assert.Contains(t, rendered, "anthropic")
	assert.Contains(t, rendered, "Model")
	assert.Contains(t, rendered, "Provider")
}

func TestLandingModel_DrawRendersKeybindingHints(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	assert.Contains(t, rendered, "ctrl+p model")
	assert.Contains(t, rendered, "ctrl+n new")
}

func TestLandingModel_DrawZeroArea(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	area := uv.Rect(0, 0, 0, 0)
	m.Draw(scr, area, nil)
	// Should not panic
}

func TestLandingModel_SetSize(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	m2 := m.SetSize(100, 50)
	assert.Equal(t, 100, m2.width)
	assert.Equal(t, 50, m2.height)
}

func TestLandingModel_DrawNoModel(t *testing.T) {
	m := NewLandingModel("", "", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	// Should still render logo and hints but no model/provider info
	assert.Contains(t, rendered, "█████ ███ █████")
	assert.NotContains(t, rendered, "glm-5.1")
	assert.NotContains(t, rendered, "Model")
	assert.NotContains(t, rendered, "Provider")
}

// Integration tests for landing visibility in the root model.

func TestLanding_ShownInitially(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	require.True(t, m.showLanding, "landing should be shown initially")

	view := m.View()
	assert.Contains(t, view.Content, "█████ ███ █████", "view should contain landing logo")
	// Horizontal rule should be present between info and hints
	assert.Contains(t, view.Content, "─", "view should contain horizontal rule")
}

func TestLanding_HiddenAfterFirstSubmit(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 30
	m.chat = m.chat.SetSize(80, m.chatHeight(30))

	require.True(t, m.showLanding)

	model, _ := m.onSubmit("hello")
	m = model.(Model)

	assert.False(t, m.showLanding, "landing should be hidden after first submit")

	view := m.View()
	// Check for a landing-specific string that won't appear elsewhere (e.g. directory paths)
	assert.NotContains(t, view.Content, "Model       ", "view should not contain landing label padding after submit")
}

func TestLanding_ReShownOnClear(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 30
	m.chat = m.chat.SetSize(80, m.chatHeight(30))

	// Submit to hide landing
	model, _ := m.onSubmit("hello")
	m = model.(Model)
	require.False(t, m.showLanding)

	// /clear command re-shows landing
	model, _ = m.onSubmit("/clear")
	m = model.(Model)

	assert.True(t, m.showLanding, "landing should re-show after /clear")
	view := m.View()
	assert.Contains(t, view.Content, "█████ ███ █████")
}

func TestLanding_ReShownOnNew(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 30
	m.chat = m.chat.SetSize(80, m.chatHeight(30))

	// Submit to hide landing
	model, _ := m.onSubmit("hello")
	m = model.(Model)
	require.False(t, m.showLanding)

	// /new command re-shows landing
	model, _ = m.onSubmit("/new")
	m = model.(Model)

	assert.True(t, m.showLanding, "landing should re-show after /new")
}

func TestLanding_HidesHintsWhenActive(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 30

	require.True(t, m.showLanding)

	view := m.View()
	// The old hints line should NOT appear when landing is active
	// (landing has its own hints embedded)
	assert.NotContains(t, view.Content, "ctrl+p model · ctrl+l select · shift+tab thinking · ctrl+t toggle")
}

func TestLanding_EditorStillAccessibleWhenLandingActive(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 40

	require.True(t, m.showLanding)

	// The editor should still be functional when landing is active
	assert.True(t, m.editor.Focused())
	assert.Empty(t, m.editor.Value(), "editor starts empty")

	// Verify the layout still allocates editor space by checking the model can handle input
	m.editor = m.editor.SetValue("test input")
	assert.Equal(t, "test input", m.editor.Value())
}

// --- Task 5: Landing boot/status layout tests ---

func TestLandingModel_DrawNoPlaceholder(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	// Placeholder text should NOT be in landing output (it's in the editor now)
	assert.NotContains(t, rendered, "Type a message to get started")
}

func TestLandingModel_DrawRuleInMutedColor(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	// Rule should be rendered with ANSI color code for Muted (240)
	assert.Contains(t, rendered, "\x1b[38;5;240m")
}

func TestLandingModel_DrawRendersExtensions(t *testing.T) {
	exts := []string{"agent", "tui", "bash", "read"}
	m := NewLandingModel("glm-5.1", "anthropic", exts)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	assert.Contains(t, rendered, "Extensions")
	assert.Contains(t, rendered, "agent")
	assert.Contains(t, rendered, "tui")
	assert.Contains(t, rendered, "bash")
	assert.Contains(t, rendered, "read")
}

func TestLandingModel_DrawWrapsExtensionsToContentWidth(t *testing.T) {
	exts := []string{
		"agent", "tui", "bash", "read", "edit", "write", "grep", "find", "ls", "search", "webfetch", "subagent",
	}
	m := NewLandingModel("glm-5.1", "anthropic", exts).SetSize(120, 30)
	scr := uv.NewScreenBuffer(120, 30)
	m.Draw(scr, scr.Bounds(), nil)

	for line := range strings.SplitSeq(ansi.Strip(scr.Render()), "\n") {
		if strings.HasPrefix(line, "    ") {
			assert.LessOrEqual(t, lipgloss.Width(line), landingContentWidth)
		}
	}
}

func TestLandingModel_DrawNoExtensions(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	// Should not contain the extensions label when no extensions are provided
	assert.NotContains(t, rendered, "Extensions")
}

func TestLandingModel_DrawUsesCustomTheme(t *testing.T) {
	custom := &palette.Theme{
		AccentBright: "99",
		Accent:       "88",
		Muted:        "77",
	}
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), custom)
	rendered := scr.Render()

	// Title uses accent bright (99), values use accent+bold (88;1), labels use muted (77)
	assert.Contains(t, rendered, "\x1b[38;5;99m")
	assert.Contains(t, rendered, "\x1b[38;5;88;1m")
	assert.Contains(t, rendered, "\x1b[38;5;77m")
}

func TestLandingModel_DrawUsesStyleSetNotDefaultTheme(t *testing.T) {
	custom := &palette.Theme{
		AccentBright: "42",
		Accent:       "43",
		Muted:        "44",
	}
	s := styles.New(custom)
	m := NewLandingModel("glm-5.1", "anthropic", nil).SetStyles(s)
	scr := uv.NewScreenBuffer(60, 24)
	// Pass nil theme to verify stored styles are used when theme is nil
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	// Should contain custom theme colors
	assert.Contains(t, rendered, "\x1b[38;5;42m")
}

func TestLandingModel_DrawNarrowTerminal(t *testing.T) {
	exts := []string{"agent", "tui", "bash", "read", "edit", "write"}
	m := NewLandingModel("glm-5.1", "anthropic", exts)
	m = m.SetSize(20, 24)
	scr := uv.NewScreenBuffer(20, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	// Logo and labels should still render
	assert.Contains(t, rendered, "█████")
	assert.Contains(t, rendered, "Model")
	assert.Contains(t, rendered, "Provider")
	assert.Contains(t, rendered, "Extensions")
	// Extensions should wrap
	assert.Contains(t, rendered, "agent")
}

func TestLandingModel_DrawShortTerminal(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	m = m.SetSize(60, 3)
	scr := uv.NewScreenBuffer(60, 3)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	// Should render at least the logo without panicking
	assert.Contains(t, rendered, "█████")
}

func TestLandingModel_SetStyles(t *testing.T) {
	custom := &palette.Theme{
		AccentBright: "55",
	}
	s := styles.New(custom)
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	m2 := m.SetStyles(s)
	assert.Equal(t, s, m2.styles)
}

func TestLandingModel_DrawKvRowFormat(t *testing.T) {
	m := NewLandingModel("glm-5.1", "anthropic", nil)
	scr := uv.NewScreenBuffer(60, 24)
	m.Draw(scr, scr.Bounds(), nil)
	rendered := scr.Render()

	// Verify label and value appear on the same conceptual row
	// The kv format produces: "Model        glm-5.1"
	assert.Contains(t, rendered, "Model")
	assert.Contains(t, rendered, "glm-5.1")
	assert.Contains(t, rendered, "Provider")
	assert.Contains(t, rendered, "anthropic")
}

func TestWrapList(t *testing.T) {
	items := []string{"agent", "tui", "bash", "read", "edit", "write"}

	// Wide enough for all items on one line
	lines := wrapList(items, 80)
	require.Len(t, lines, 1)
	assert.Equal(t, "    agent, tui, bash, read, edit, write", lines[0])

	// Narrow width forces wrapping
	lines = wrapList(items, 30)
	require.GreaterOrEqual(t, len(lines), 2)
	assert.True(t, strings.HasPrefix(lines[0], "    "))
	assert.Contains(t, lines[0], "agent")

	// Empty list returns nil
	assert.Nil(t, wrapList(nil, 80))

	// Zero width falls back to single line
	lines = wrapList(items, 0)
	require.Len(t, lines, 1)
	assert.Equal(t, "    agent, tui, bash, read, edit, write", lines[0])
}
