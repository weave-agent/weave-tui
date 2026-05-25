package keybindings

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindingRegistry_Defaults(t *testing.T) {
	r := NewBindingRegistry()

	actions := map[string]BindingAction{
		"ctrl+d":      ActionExit,
		"esc":         ActionInterrupt,
		"ctrl+l":      ActionModelSelect,
		"ctrl+p":      ActionModelCycle,
		"ctrl+t":      ActionThinkingCycle,
		"shift+enter": ActionEditorNewline,
		"ctrl+j":      ActionEditorNewline,
	}

	for key, want := range actions {
		action, ok := r.Resolve(key)
		assert.True(t, ok, "expected default binding for %s", key)
		assert.Equal(t, want, action, "wrong action for key %s", key)
	}
}

func TestBindingRegistry_UnknownKey(t *testing.T) {
	r := NewBindingRegistry()

	_, ok := r.Resolve("ctrl+f")
	assert.False(t, ok)
}

func TestBindingRegistry_ExtensionOverridesDefault(t *testing.T) {
	r := NewBindingRegistry()

	// Register ctrl+d for a custom action (overrides exit)
	r.Register("app.custom", []string{"ctrl+d"}, "Custom action")

	action, ok := r.Resolve("ctrl+d")
	require.True(t, ok)
	assert.Equal(t, BindingAction("app.custom"), action)
}

func TestBindingRegistry_ExtensionRegistersNewKey(t *testing.T) {
	r := NewBindingRegistry()

	r.Register("app.search", []string{"ctrl+f"}, "Search")

	action, ok := r.Resolve("ctrl+f")
	require.True(t, ok)
	assert.Equal(t, BindingAction("app.search"), action)
}

func TestBindingRegistry_ExtensionReplacesOwnKeys(t *testing.T) {
	r := NewBindingRegistry()

	r.Register("app.custom", []string{"ctrl+f"}, "First")
	r.Register("app.custom", []string{"ctrl+g"}, "Second")

	// Old key should be gone
	_, ok := r.Resolve("ctrl+f")
	assert.False(t, ok, "old key should be removed when action is re-registered")

	// New key should work
	action, ok := r.Resolve("ctrl+g")
	require.True(t, ok)
	assert.Equal(t, BindingAction("app.custom"), action)
}

func TestBindingRegistry_UserConfigOverridesAll(t *testing.T) {
	r := NewBindingRegistry()

	// User remaps model select from ctrl+l to ctrl+k
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "keybindings.yaml")
	err := os.WriteFile(cfgPath, []byte("keybindings:\n  app.model.select:\n    - ctrl+k\n"), 0o644)
	require.NoError(t, err)

	err = r.LoadUserConfig(cfgPath)
	require.NoError(t, err)

	// ctrl+l no longer triggers model select (unless extension also registered it)
	action, ok := r.Resolve("ctrl+k")
	require.True(t, ok)
	assert.Equal(t, ActionModelSelect, action)

	// ctrl+l still resolves from defaults
	action, ok = r.Resolve("ctrl+l")
	require.True(t, ok)
	assert.Equal(t, ActionModelSelect, action)
}

func TestBindingRegistry_UserConfigRemapsKey(t *testing.T) {
	r := NewBindingRegistry()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "keybindings.yaml")
	err := os.WriteFile(cfgPath, []byte("keybindings:\n  app.exit:\n    - ctrl+q\n"), 0o644)
	require.NoError(t, err)

	err = r.LoadUserConfig(cfgPath)
	require.NoError(t, err)

	// ctrl+q now triggers exit
	action, ok := r.Resolve("ctrl+q")
	require.True(t, ok)
	assert.Equal(t, ActionExit, action)

	// ctrl+d still works from defaults
	action, ok = r.Resolve("ctrl+d")
	require.True(t, ok)
	assert.Equal(t, ActionExit, action)
}

func TestBindingRegistry_UserConfigNonExistent(t *testing.T) {
	r := NewBindingRegistry()

	err := r.LoadUserConfig("/nonexistent/path/keybindings.yaml")
	require.NoError(t, err)

	// Defaults should still work
	action, ok := r.Resolve("ctrl+d")
	require.True(t, ok)
	assert.Equal(t, ActionExit, action)
}

func TestBindingRegistry_UserConfigPriority(t *testing.T) {
	r := NewBindingRegistry()

	// Extension registers ctrl+f for search
	r.Register("app.search", []string{"ctrl+f"}, "Search")

	// User remaps ctrl+f to something else
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "keybindings.yaml")
	err := os.WriteFile(cfgPath, []byte("keybindings:\n  app.exit:\n    - ctrl+f\n"), 0o644)
	require.NoError(t, err)

	err = r.LoadUserConfig(cfgPath)
	require.NoError(t, err)

	// User config wins: ctrl+f -> exit, not search
	action, ok := r.Resolve("ctrl+f")
	require.True(t, ok)
	assert.Equal(t, ActionExit, action)
}

func TestBindingRegistry_AllBindings(t *testing.T) {
	r := NewBindingRegistry()

	bindings := r.AllBindings()
	assert.NotEmpty(t, bindings)

	// Should contain all default actions
	names := make(map[BindingAction]bool)
	for _, b := range bindings {
		names[b.Action] = true
	}

	assert.True(t, names[ActionExit])
	assert.True(t, names[ActionInterrupt])
	assert.True(t, names[ActionModelSelect])
	assert.True(t, names[ActionModelCycle])
	assert.True(t, names[ActionThinkingCycle])
	assert.True(t, names[ActionCursorLineStart])
	assert.True(t, names[ActionSuspend])
}

func TestBindingRegistry_AllBindingsSorted(t *testing.T) {
	r := NewBindingRegistry()

	bindings := r.AllBindings()
	for i := 1; i < len(bindings); i++ {
		assert.LessOrEqual(t, bindings[i-1].Action, bindings[i].Action,
			"bindings should be sorted by action name")
	}
}

func TestKeyString(t *testing.T) {
	tests := []struct {
		key  tea.KeyPressMsg
		want string
	}{
		{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, "ctrl+c"},
		{tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}, "ctrl+d"},
		{tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, "ctrl+l"},
		{tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}, "ctrl+p"},
		{tea.KeyPressMsg{Code: tea.KeyEsc}, "esc"},
		// Keystroke() includes modifier prefix even when Text is set
		{tea.KeyPressMsg{Code: 'a', Mod: tea.ModAlt, Text: "a"}, "alt+a"},
		{tea.KeyPressMsg{Code: 'g', Mod: tea.ModShift, Text: "G"}, "shift+g"},
		{tea.KeyPressMsg{Code: 'd', Mod: tea.ModAlt, Text: "d"}, "alt+d"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, keyString(tt.key), "keyString(%v)", tt.key)
	}
}

func TestKeyString_EscapePassthrough(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyEsc}
	assert.Equal(t, "esc", keyString(msg))
}

func TestLoadKeybindings_ProjectConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".weave", "settings.json")

	// No keybindings file -> empty
	result := loadKeybindings(cfgPath)
	assert.Empty(t, result)
}

func TestLoadKeybindings_NearConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".weave", "settings.json")
	kbPath := filepath.Join(dir, ".weave", "keybindings.yaml")

	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o755))

	err := os.WriteFile(kbPath, []byte("keybindings: {}\n"), 0o644)
	require.NoError(t, err)

	result := loadKeybindings(cfgPath)
	assert.Equal(t, kbPath, result)
}

func TestLoadKeybindings_EmptyConfigPath(t *testing.T) {
	result := loadKeybindings("")
	assert.Empty(t, result)
}

func TestKeyString_V2ModifierCombinations(t *testing.T) {
	tests := []struct {
		key  tea.KeyPressMsg
		want string
	}{
		// Ctrl keys
		{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, "ctrl+c"},
		{tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}, "ctrl+z"},
		// Alt modifier always produces keystroke form with Keystroke()
		{tea.KeyPressMsg{Code: 'a', Mod: tea.ModAlt, Text: "a"}, "alt+a"},
		{tea.KeyPressMsg{Code: 'a', Mod: tea.ModAlt}, "alt+a"},
		// Shift modifier with special keys
		{tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, "shift+tab"},
		{tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}, "shift+enter"},
		{tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl}, "ctrl+j"},
		// Plain printable chars
		{tea.KeyPressMsg{Code: 'x', Text: "x"}, "x"},
		// Special keys
		{tea.KeyPressMsg{Code: tea.KeyEnter}, "enter"},
		{tea.KeyPressMsg{Code: tea.KeyBackspace}, "backspace"},
		{tea.KeyPressMsg{Code: tea.KeyUp}, "up"},
		{tea.KeyPressMsg{Code: tea.KeyDown}, "down"},
		{tea.KeyPressMsg{Code: tea.KeyPgUp}, "pgup"},
		{tea.KeyPressMsg{Code: tea.KeyHome}, "home"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, keyString(tt.key), "keyString(%+v)", tt.key)
	}
}

func TestBindingRegistry_AllBindingsUsesResolvedKeys(t *testing.T) {
	r := NewBindingRegistry()
	r.Register("app.custom", []string{"ctrl+d"}, "Custom action")

	bindings := r.AllBindings()

	byAction := make(map[BindingAction]Binding, len(bindings))
	for _, binding := range bindings {
		byAction[binding.Action] = binding
	}

	assert.NotContains(t, byAction[ActionExit].Keys, "ctrl+d")
	assert.Contains(t, byAction["app.custom"].Keys, "ctrl+d")
}
