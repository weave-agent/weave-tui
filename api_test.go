package tui_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weave-agent/weave/sdk"

	tui "github.com/weave-agent/weave-tui"
)

// Verify that the public API surface is usable from an external package.

func TestPublicAPI_TUIConfig(t *testing.T) {
	// TUIConfig should be constructible and usable.
	cfg := tui.TUIConfig{Theme: "dark", EditorMaxLines: 10}
	assert.Equal(t, "dark", cfg.Theme)
	assert.Equal(t, 10, cfg.EditorMaxLines)
}

func TestPublicAPI_RegisterAndGetTUIExtension(t *testing.T) {
	tui.ResetTUIExtensionRegistry()
	t.Cleanup(tui.ResetTUIExtensionRegistry)

	ext := &publicExt{name: "pub-ext"}

	tui.RegisterTUIExtension("pub-ext", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (tui.TUIExtension, error) {
		return ext, nil
	})

	assert.True(t, tui.TUIExtensionRegistered("pub-ext"))

	got, err := tui.GetTUIExtension("pub-ext", sdk.NoopConfig{})
	require.NoError(t, err)
	assert.Equal(t, "pub-ext", got.Name())
}

func TestPublicAPI_ListTUIExtensions(t *testing.T) {
	tui.ResetTUIExtensionRegistry()
	t.Cleanup(tui.ResetTUIExtensionRegistry)

	tui.RegisterTUIExtension("zebra", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (tui.TUIExtension, error) {
		return &publicExt{name: "zebra"}, nil
	})
	tui.RegisterTUIExtension("alpha", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (tui.TUIExtension, error) {
		return &publicExt{name: "alpha"}, nil
	})

	names := tui.ListTUIExtensions()
	assert.Equal(t, []string{"alpha", "zebra"}, names)
}

func TestPublicAPI_GetTUIExtensionNotRegistered(t *testing.T) {
	tui.ResetTUIExtensionRegistry()
	t.Cleanup(tui.ResetTUIExtensionRegistry)

	_, err := tui.GetTUIExtension("missing", sdk.NoopConfig{})
	require.Error(t, err)
	assert.ErrorIs(t, err, sdk.ErrNotRegistered)
}

func TestPublicAPI_ResetTUIExtensionRegistry(t *testing.T) {
	tui.ResetTUIExtensionRegistry()
	t.Cleanup(tui.ResetTUIExtensionRegistry)

	tui.RegisterTUIExtension("temporary", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (tui.TUIExtension, error) {
		return &publicExt{name: "temporary"}, nil
	})

	require.True(t, tui.TUIExtensionRegistered("temporary"))

	tui.ResetTUIExtensionRegistry()

	assert.Empty(t, tui.ListTUIExtensions())
}

func TestPublicAPI_PanelTypes(t *testing.T) {
	// PanelConfig, PanelPlacement, and PanelDrawer should be usable.
	cfg := tui.PanelConfig{
		ID:        "test-panel",
		Placement: tui.AboveEditor,
		Blocking:  false,
		Width:     40,
		Height:    10,
		Title:     "Test",
	}
	assert.Equal(t, "test-panel", cfg.ID)
	assert.Equal(t, tui.AboveEditor, cfg.Placement)

	// All placement constants should be accessible.
	_ = tui.AsOverlay
	_ = tui.BelowEditor
	_ = tui.TrayOnly
}

func TestPublicAPI_ThemeDef(t *testing.T) {
	td := tui.ThemeDef{
		Accent:     "60",
		Foreground: "255",
	}
	assert.Equal(t, "60", td.Accent)
	assert.Equal(t, "255", td.Foreground)
}

func TestPublicAPI_KeyEvent(t *testing.T) {
	ev := tui.KeyEvent{Code: 'a', Mod: 1, String: "a"}
	assert.Equal(t, 'a', ev.Code)
}

func TestPublicAPI_AutocompleteTypes(t *testing.T) {
	ctx := tui.AutocompleteContext{Text: "hello", Cursor: 2, Line: "hello"}
	assert.Equal(t, "hello", ctx.Text)

	sugg := tui.AutocompleteSuggestion{Label: "hi", Description: "greeting", Value: "hi"}
	assert.Equal(t, "hi", sugg.Label)
}

func TestPackageVisibility_RenderingSupportIsInternal(t *testing.T) {
	cmd := exec.Command("go", "list", "./...")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	packages := strings.Split(strings.TrimSpace(string(out)), "\n")
	require.NotEmpty(t, packages)

	pkgSet := make(map[string]bool, len(packages))
	for _, pkg := range packages {
		pkgSet[pkg] = true
	}

	for _, pkg := range []string{
		"github.com/weave-agent/weave-tui/palette",
		"github.com/weave-agent/weave-tui/styles",
		"github.com/weave-agent/weave-tui/xchroma",
	} {
		assert.False(t, pkgSet[pkg], "%s should not be a public package", pkg)
	}

	for _, pkg := range []string{
		"github.com/weave-agent/weave-tui/internal/palette",
		"github.com/weave-agent/weave-tui/internal/styles",
		"github.com/weave-agent/weave-tui/internal/xchroma",
	} {
		assert.True(t, pkgSet[pkg], "%s should be available internally", pkg)
	}
}

// publicExt is a minimal TUIExtension implementation for testing.
type publicExt struct {
	name string
}

func (e *publicExt) Name() string                { return e.name }
func (e *publicExt) RegisterTUI(_ tui.TUIExtAPI) {}

// Verify that publicExt actually implements TUIExtension.
var _ tui.TUIExtension = (*publicExt)(nil)
