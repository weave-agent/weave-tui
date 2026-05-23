package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weave-agent/weave-tui/internal/palette"
)

func TestNewModelWithConfig_AppliesConfiguredThemeAtStartup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	custom := startupTestTheme("#112233")
	writeStartupTheme(t, home, "ocean", custom)

	ui := NewTUIImpl(nil, nil)
	m := NewModelWithConfig(nil, nil, nil, ui, TUIConfig{Theme: "ocean"})

	assert.Equal(t, "#112233", m.theme.Accent)
	assert.Equal(t, "#112233", m.styles.Theme().Accent)
	assert.Equal(t, "#112233", ui.Theme().Accent)
	assert.Equal(t, "ocean", ui.Theme().Name)
	assert.Contains(t, ui.ListThemes(), "ocean")
}

func TestNewModelWithConfig_UnknownConfiguredThemeFallsBackToDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ui := NewTUIImpl(nil, nil)
	m := NewModelWithConfig(nil, nil, nil, ui, TUIConfig{Theme: "missing"})

	assert.Equal(t, palette.DefaultTheme().Accent, m.theme.Accent)
	assert.Equal(t, palette.DefaultTheme().Accent, m.styles.Theme().Accent)
	assert.Equal(t, "default", ui.Theme().Name)
}

func writeStartupTheme(t *testing.T, home, name string, theme map[string]string) {
	t.Helper()

	dir := filepath.Join(home, ".weave", "themes")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	data, err := json.Marshal(theme)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name+".json"), data, 0o644))
}

func startupTestTheme(accent string) map[string]string {
	return map[string]string{
		"name":                  "ocean",
		"foreground":            "#f0f0f0",
		"foregroundDim":         "#c0c0c0",
		"foregroundBright":      "#ffffff",
		"muted":                 "#909090",
		"mutedBright":           "#a0a0a0",
		"background":            "#000000",
		"backgroundTint":        "#101010",
		"backgroundTint2":       "#202020",
		"border":                "#303030",
		"borderFocused":         "#404040",
		"success":               "#00aa66",
		"error":                 "#cc3333",
		"warning":               "#cc9900",
		"backgroundTintPending": "#111111",
		"backgroundTintSuccess": "#112211",
		"backgroundTintError":   "#221111",
		"accent":                accent,
		"accentDim":             "#223344",
		"accentBright":          "#334455",
	}
}
