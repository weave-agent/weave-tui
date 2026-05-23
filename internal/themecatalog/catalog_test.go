package themecatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadIncludesBuiltins(t *testing.T) {
	catalog, err := Load("")
	require.NoError(t, err)

	entry, ok := catalog.Entry("default")
	require.True(t, ok)
	assert.Equal(t, SourceBuiltin, entry.Source)
	assert.Equal(t, "245", entry.Theme.Accent)
}

func TestLoadUserThemesAndSortedListing(t *testing.T) {
	dir := t.TempDir()
	writeThemeFile(t, dir, "zeta", "#101010")
	writeThemeFile(t, dir, "alpha", "#202020")

	catalog, err := Load(dir)
	require.NoError(t, err)

	entries := catalog.List()

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}

	assert.Equal(t, []string{"alpha", "default", "zeta"}, names)

	entry, ok := catalog.Entry("alpha")
	require.True(t, ok)
	assert.Equal(t, SourceUser, entry.Source)
	assert.Equal(t, filepath.Join(dir, "alpha.json"), entry.Path)
	assert.Equal(t, "#202020", entry.Theme.Accent)
}

func TestLoadUserThemeOverridesBuiltin(t *testing.T) {
	dir := t.TempDir()
	writeThemeFile(t, dir, "default", "#303030")

	catalog, err := Load(dir)
	require.NoError(t, err)

	entry, ok := catalog.Entry("default")
	require.True(t, ok)
	assert.Equal(t, SourceUser, entry.Source)
	assert.Equal(t, "#303030", entry.Theme.Accent)
}

func TestThemeReturnsUnknownThemeError(t *testing.T) {
	catalog, err := Load("")
	require.NoError(t, err)

	theme, err := catalog.Theme("missing")

	require.Error(t, err)
	assert.Nil(t, theme)
	assert.Contains(t, err.Error(), "unknown theme: missing")
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"accent":`), 0o600))

	catalog, err := Load(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse theme file")
	assert.NotNil(t, catalog)
}

func TestLoadRejectsMissingRequiredField(t *testing.T) {
	dir := t.TempDir()
	theme := validThemeJSON("missing", "#404040")
	delete(theme, "accentBright")
	writeJSON(t, filepath.Join(dir, "missing.json"), theme)

	catalog, err := Load(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `missing required field "accentBright"`)
	assert.NotNil(t, catalog)
}

func TestLoadRejectsInvalidFilenameThemeName(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "bad name.json"), validThemeJSON("bad name", "#505050"))

	catalog, err := Load(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid theme filename")
	assert.NotNil(t, catalog)
}

func TestLoadRejectsInvalidJSONThemeName(t *testing.T) {
	dir := t.TempDir()
	theme := validThemeJSON("good", "#606060")
	theme["name"] = "../bad"
	writeJSON(t, filepath.Join(dir, "good.json"), theme)

	catalog, err := Load(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid theme name")
	assert.NotNil(t, catalog)
}

func TestLoadRejectsNameMismatch(t *testing.T) {
	dir := t.TempDir()
	theme := validThemeJSON("one", "#707070")
	theme["name"] = "two"
	writeJSON(t, filepath.Join(dir, "one.json"), theme)

	catalog, err := Load(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `must match filename "one"`)
	assert.NotNil(t, catalog)
}

func TestLoadRejectsInvalidColor(t *testing.T) {
	dir := t.TempDir()
	theme := validThemeJSON("bad-color", "#808080")
	theme["accent"] = "196"
	writeJSON(t, filepath.Join(dir, "bad-color.json"), theme)

	catalog, err := Load(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `field "accent" must be a #RRGGBB color`)
	assert.NotNil(t, catalog)
}

func TestLoadRejectsUnknownField(t *testing.T) {
	dir := t.TempDir()
	theme := validThemeJSON("unknown", "#808080")
	theme["extra"] = "#ffffff"
	writeJSON(t, filepath.Join(dir, "unknown.json"), theme)

	catalog, err := Load(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "extra"`)
	assert.NotNil(t, catalog)
}

func TestLoadRejectsTrailingJSON(t *testing.T) {
	dir := t.TempDir()
	data, err := json.Marshal(validThemeJSON("trailing", "#808080"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "trailing.json"), append(data, []byte(` {}`)...), 0o600))

	catalog, err := Load(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected trailing JSON")
	assert.NotNil(t, catalog)
}

func TestLoadSkipsInvalidFilesAndKeepsValidThemes(t *testing.T) {
	dir := t.TempDir()
	writeThemeFile(t, dir, "valid", "#909090")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"accent":`), 0o600))

	catalog, err := Load(dir)

	require.Error(t, err)

	entry, ok := catalog.Entry("valid")
	require.True(t, ok)
	assert.Equal(t, SourceUser, entry.Source)
	assert.Equal(t, "#909090", entry.Theme.Accent)
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "theme"},
		{name: "catppuccin-mocha"},
		{name: "foo_bar.1"},
		{name: ""},
		{name: "."},
		{name: ".."},
		{name: "../theme", wantErr: true},
		{name: `foo\bar`, wantErr: true},
		{name: "bad name", wantErr: true},
		{name: "bad\nname", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name)
			if tt.wantErr || tt.name == "" || tt.name == "." || tt.name == ".." {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func writeThemeFile(t *testing.T, dir, name, accent string) {
	t.Helper()
	writeJSON(t, filepath.Join(dir, name+".json"), validThemeJSON(name, accent))
}

func writeJSON(t *testing.T, path string, value map[string]string) {
	t.Helper()

	data, err := json.Marshal(value)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func validThemeJSON(name, accent string) map[string]string {
	return map[string]string{
		"name":                  name,
		"foreground":            "#F8F8F2",
		"foregroundDim":         "#C0C0C0",
		"foregroundBright":      "#FFFFFF",
		"muted":                 "#666666",
		"mutedBright":           "#999999",
		"background":            "#000000",
		"backgroundTint":        "#111111",
		"backgroundTint2":       "#222222",
		"border":                "#333333",
		"borderFocused":         "#444444",
		"success":               "#50FA7B",
		"error":                 "#FF5555",
		"warning":               "#F1FA8C",
		"backgroundTintPending": "#181818",
		"backgroundTintSuccess": "#102010",
		"backgroundTintError":   "#201010",
		"accent":                accent,
		"accentDim":             "#555555",
		"accentBright":          "#AAAAAA",
	}
}
