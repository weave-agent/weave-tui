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
	assert.Equal(t, defaultThemeName, ui.Theme().Name)
}

func TestModel_InitAppliesConfiguredExtensionThemeAfterRegistration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ui := NewTUIImpl(nil, nil)
	m := NewModelWithConfig(nil, nil, nil, ui, TUIConfig{Theme: "extension-theme"})
	require.Equal(t, defaultThemeName, ui.Theme().Name)
	require.Equal(t, palette.DefaultTheme().Accent, m.theme.Accent)

	require.NoError(t, ui.RegisterTheme("extension-theme", startupExtensionThemeDef("#445566")))

	msg := executeCmd(t, m.Init())
	require.IsType(t, startupThemeReadyMsg{}, msg)

	model, _ := m.Update(msg)
	updated := model.(Model)

	assert.Equal(t, "extension-theme", ui.Theme().Name)
	assert.Equal(t, "#445566", ui.Theme().Accent)
	assert.Equal(t, "#445566", updated.theme.Accent)
	assert.Equal(t, "#445566", updated.styles.Theme().Accent)
}

func TestNewModelWithConfig_InvalidThemeFileDoesNotDisableValidTheme(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeStartupTheme(t, home, "ocean", startupTestTheme("#112233"))
	dir := filepath.Join(home, ".weave", "themes")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"accent":`), 0o644))

	ui := NewTUIImpl(nil, nil)
	m := NewModelWithConfig(nil, nil, nil, ui, TUIConfig{Theme: "ocean"})

	assert.Equal(t, "#112233", m.theme.Accent)
	assert.Equal(t, "ocean", ui.Theme().Name)
	assert.Contains(t, ui.ListThemes(), "ocean")
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

func startupExtensionThemeDef(accent string) ThemeDef {
	return ThemeDef{
		Foreground:            "#f0f0f0",
		ForegroundDim:         "#c0c0c0",
		ForegroundBright:      "#ffffff",
		Muted:                 "#909090",
		MutedBright:           "#a0a0a0",
		Background:            "#000000",
		BackgroundTint:        "#101010",
		BackgroundTint2:       "#202020",
		Border:                "#303030",
		BorderFocused:         "#404040",
		Success:               "#00aa66",
		Error:                 "#cc3333",
		Warning:               "#cc9900",
		BackgroundTintPending: "#111111",
		BackgroundTintSuccess: "#112211",
		BackgroundTintError:   "#221111",
		Accent:                accent,
		AccentDim:             "#223344",
		AccentBright:          "#334455",
	}
}
