package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/weave-agent/weave/bus"
	"github.com/weave-agent/weave/sdk"

	tuibridge "github.com/weave-agent/weave-tui/internal/bridge"
	tuievents "github.com/weave-agent/weave-tui/internal/events"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func subscribeToChan(b *bus.Bus, topic string) <-chan sdk.Event {
	ch := make(chan sdk.Event, 64)

	b.On(topic, func(ev sdk.Event) error {
		select {
		case ch <- ev:
		default:
		}

		return nil
	})

	return ch
}

func newTestRegistry(b sdk.Bus) *CommandRegistry {
	return NewCommandRegistry(b, "", RuntimeCommands{
		ListSessions: func(_ string) tea.Cmd {
			return func() tea.Msg { return tuievents.SessionListResultMsg{} }
		},
		Login: func() tea.Cmd {
			return func() tea.Msg {
				return tuievents.LoginListResultMsg{Providers: []tuievents.LoginProviderEntry{}}
			}
		},
		Logout: func() tea.Cmd {
			return func() tea.Msg {
				return tuievents.LogoutListResultMsg{Providers: []tuievents.LogoutProviderEntry{}}
			}
		},
	})
}

func TestCommandRegistry_BuiltinCommands(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	names := r.Names()
	assert.Contains(t, names, "/new")
	assert.Contains(t, names, "/clear")
	assert.Contains(t, names, "/quit")
	assert.Contains(t, names, "/help")
	assert.Contains(t, names, "/keybind-help")
	assert.Contains(t, names, "/compact")
	assert.Contains(t, names, "/name")
	assert.Contains(t, names, "/login")
	assert.Contains(t, names, "/logout")
}

func TestCommandRegistry_DispatchNonCommand(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("hello world")
	assert.False(t, handled)
	assert.Equal(t, CommandResult{}, result)
}

func TestCommandRegistry_DispatchQuit(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/quit")
	require.True(t, handled)
	assert.True(t, result.Quit)
	assert.False(t, result.ClearChat)
}

func TestCommandRegistry_DispatchNew(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/new")
	require.True(t, handled)
	assert.True(t, result.ClearChat)
	assert.True(t, result.ResetPrompt)
	assert.False(t, result.Quit)
}

func TestCommandRegistry_DispatchClearAliasForNew(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/clear")
	require.True(t, handled)
	assert.True(t, result.ClearChat)
	assert.True(t, result.ResetPrompt)

	// Same result as /new
	handledNew, resultNew := r.Dispatch("/new")
	assert.Equal(t, resultNew, result)
	assert.Equal(t, handledNew, handled)
}

func TestCommandRegistry_DispatchHelp(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/help")
	require.True(t, handled)
	assert.Contains(t, result.Notify, "Available commands")
	assert.Contains(t, result.Notify, "/quit")
	assert.Contains(t, result.Notify, "/new")
}

func TestCommandRegistry_DispatchCompact(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, tuibridge.TopicSteer)

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/compact")
	require.True(t, handled)
	assert.NotNil(t, result.Command)

	// Execute the command
	msg := result.Command()
	assert.Nil(t, msg)

	select {
	case evt := <-ch:
		assert.Equal(t, "compact", evt.Payload)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for compact event")
	}
}

func TestCommandRegistry_DispatchCompactWithArgs(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, tuibridge.TopicSteer)

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/compact focus on the auth refactor")
	require.True(t, handled)
	assert.NotNil(t, result.Command)

	msg := result.Command()
	assert.Nil(t, msg)

	select {
	case evt := <-ch:
		assert.Equal(t, "compact focus on the auth refactor", evt.Payload)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for compact event with args")
	}
}

func TestCommandRegistry_DispatchNameWithArgs(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, tuibridge.TopicSteer)

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/name my session")
	require.True(t, handled)
	assert.NotNil(t, result.Command)

	msg := result.Command()
	assert.Nil(t, msg)

	evt := <-ch
	assert.Equal(t, "name my session", evt.Payload)
}

func TestCommandRegistry_DispatchUnknownCommand(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/unknown")
	require.True(t, handled)
	assert.Contains(t, result.Notify, "unknown command: /unknown")
}

func TestCommandRegistry_DispatchCommandWithArgs(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	// /name with args should work
	handled, result := r.Dispatch("/name test")
	require.True(t, handled)
	assert.NotNil(t, result.Command)

	// /quit with extra args still quits
	handled, result = r.Dispatch("/quit now")
	require.True(t, handled)
	assert.True(t, result.Quit)
}

func TestCommandRegistry_NamesSorted(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	names := r.Names()
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "names should be sorted: %v", names)
	}
}

func TestCommandRegistry_Lookup(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	info, ok := r.Lookup("/quit")
	require.True(t, ok)
	assert.Equal(t, "/quit", info.Name)
	assert.Equal(t, "Exit weave", info.Description)

	_, ok = r.Lookup("/nonexistent")
	assert.False(t, ok)
}

func TestCommandRegistry_Register(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	called := false

	r.Register("/custom", "custom command", false, func(args string) CommandResult {
		called = true

		assert.Equal(t, "arg1", args)

		return CommandResult{}
	})

	names := r.Names()
	assert.Contains(t, names, "/custom")

	handled, _ := r.Dispatch("/custom arg1")
	require.True(t, handled)
	assert.True(t, called)
}

func TestCommandRegistry_AcceptsFilesDefaultsToFalse(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	// All built-in commands should have AcceptsFiles = false
	for _, name := range r.Names() {
		info, ok := r.Lookup(name)
		require.True(t, ok, "should find %s", name)
		assert.False(t, info.AcceptsFiles, "%s should not accept files by default", name)
	}

	// Register with explicit true
	r.Register("/filecmd", "accepts files", true, func(_ string) CommandResult { return CommandResult{} })
	info, ok := r.Lookup("/filecmd")
	require.True(t, ok)
	assert.True(t, info.AcceptsFiles, "/filecmd should accept files")

	// Register with explicit false
	r.Register("/nocmd", "no files", false, func(_ string) CommandResult { return CommandResult{} })
	info, ok = r.Lookup("/nocmd")
	require.True(t, ok)
	assert.False(t, info.AcceptsFiles, "/nocmd should not accept files")
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input string
		name  string
		args  string
	}{
		{"/quit", "/quit", ""},
		{"/quit ", "/quit", ""},
		{"/name my session", "/name", "my session"},
		{"/help  ", "/help", ""},
		{"  /compact  ", "/compact", ""},
		{"/name  extra  spaces  ", "/name", "extra  spaces"},
	}

	for _, tt := range tests {
		name, args := parseCommand(tt.input)
		assert.Equal(t, tt.name, name, "name for %q", tt.input)
		assert.Equal(t, tt.args, args, "args for %q", tt.input)
	}
}

func TestCommandRegistry_HelpListsAll(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	r.Register("/testcmd", "a test", false, func(_ string) CommandResult { return CommandResult{} })

	help := r.helpText()
	assert.Contains(t, help, "/testcmd")
	assert.Contains(t, help, "a test")
}

func TestCommandRegistry_ReloadRegistered(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	names := r.Names()
	assert.Contains(t, names, "/reload")

	info, ok := r.Lookup("/reload")
	require.True(t, ok)
	assert.Equal(t, "Rebuild extensions and restart", info.Description)
}

func TestReloadCmd_NoLauncherPath(t *testing.T) {
	t.Setenv("WEAVE_LAUNCHER_PATH", "")
	t.Setenv("WEAVE_BUILD_HASH", "")
	t.Setenv("WEAVE_ORIG_ARGS", "")

	cmd := reloadCmd()
	msg := cmd()

	notify, ok := msg.(tuievents.NotifyMsg)
	require.True(t, ok, "expected tuievents.NotifyMsg when WEAVE_LAUNCHER_PATH is empty")
	assert.Contains(t, notify.Message, "not available")
}

func TestReloadCmd_RemovesCacheDir(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	testHash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	cacheDir := filepath.Join(fakeHome, ".weave", "bin", testHash)

	// Create a fake cache dir to verify it gets removed.
	require.NoError(t, os.MkdirAll(cacheDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "weave"), []byte("fake"), 0o600))

	origArgs, _ := json.Marshal([]string{"weave", "arg1"})

	t.Setenv("WEAVE_LAUNCHER_PATH", "/usr/local/bin/weave")
	t.Setenv("WEAVE_BUILD_HASH", testHash)
	t.Setenv("WEAVE_ORIG_ARGS", string(origArgs))

	cmd := reloadCmd()
	msg := cmd()

	reload, ok := msg.(ReloadMsg)
	require.True(t, ok, "expected ReloadMsg")
	assert.Equal(t, "/usr/local/bin/weave", reload.launcherPath)
	assert.Equal(t, []string{"weave", "arg1"}, reload.origArgs)

	// Verify cache dir was removed.
	_, statErr := os.Stat(cacheDir)
	assert.True(t, os.IsNotExist(statErr), "cache dir should be removed")
}

func TestReloadCmd_MissingOrigArgs(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	testHash := "0000000000000000000000000000000000000000000000000000000000000001"
	cacheDir := filepath.Join(fakeHome, ".weave", "bin", testHash)
	_ = os.MkdirAll(cacheDir, 0o750)

	t.Setenv("WEAVE_LAUNCHER_PATH", "/usr/bin/weave")
	t.Setenv("WEAVE_BUILD_HASH", testHash)
	t.Setenv("WEAVE_ORIG_ARGS", "")

	cmd := reloadCmd()
	msg := cmd()

	reload, ok := msg.(ReloadMsg)
	require.True(t, ok)
	assert.Equal(t, []string{"/usr/bin/weave"}, reload.origArgs)
}

func TestReloadCmd_EmptyBuildHash(t *testing.T) {
	t.Setenv("WEAVE_LAUNCHER_PATH", "/usr/bin/weave")
	t.Setenv("WEAVE_BUILD_HASH", "")
	t.Setenv("WEAVE_ORIG_ARGS", `["weave"]`)

	cmd := reloadCmd()
	msg := cmd()

	reload, ok := msg.(ReloadMsg)
	require.True(t, ok)
	assert.Equal(t, "/usr/bin/weave", reload.launcherPath)
}

// TestReloadCmd_RejectsPathTraversal verifies a malicious WEAVE_BUILD_HASH
// value is rejected before being joined into a filesystem path. Without the
// validation, "../../victim" would escape ~/.weave/bin and let os.RemoveAll
// delete unrelated files.
func TestReloadCmd_RejectsPathTraversal(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	// Create a sibling directory that must NOT be removed.
	victim := filepath.Join(fakeHome, "victim")
	require.NoError(t, os.MkdirAll(victim, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(victim, "data"), []byte("important"), 0o600))

	t.Setenv("WEAVE_LAUNCHER_PATH", "/usr/bin/weave")
	t.Setenv("WEAVE_BUILD_HASH", "../../victim")
	t.Setenv("WEAVE_ORIG_ARGS", `["weave"]`)

	cmd := reloadCmd()
	msg := cmd()

	notify, ok := msg.(tuievents.NotifyMsg)
	require.True(t, ok, "expected tuievents.NotifyMsg for invalid hash, got %T", msg)
	assert.Contains(t, notify.Message, "invalid build hash")

	// Victim directory must still exist.
	_, err := os.Stat(victim)
	assert.NoError(t, err, "victim directory must not be deleted")
}

func TestIsSHA256Hex(t *testing.T) {
	valid := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	assert.True(t, isSHA256Hex(valid))

	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"too short", "abc123"},
		{"too long", valid + "0"},
		{"path traversal", "../../victim"},
		{"uppercase letters", strings.ToUpper(valid)},
		{"non-hex char", strings.Repeat("g", 64)},
		{"slash", strings.Repeat("a", 63) + "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, isSHA256Hex(tt.in))
		})
	}
}

func TestCommandRegistry_DispatchLogin(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/login")
	require.True(t, handled)
	assert.NotNil(t, result.Command)

	msg := result.Command()
	loginResult, ok := msg.(tuievents.LoginListResultMsg)
	require.True(t, ok, "expected tuievents.LoginListResultMsg, got %T", msg)
	assert.NotNil(t, loginResult.Providers)
}

func TestCommandRegistry_DispatchLogout(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/logout")
	require.True(t, handled)
	assert.NotNil(t, result.Command)

	msg := result.Command()
	logoutResult, ok := msg.(tuievents.LogoutListResultMsg)
	require.True(t, ok, "expected tuievents.LogoutListResultMsg, got %T", msg)
	assert.NotNil(t, logoutResult.Providers)
}

func TestCommandRegistry_LoginLogoutInHelp(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	help := r.helpText()
	assert.Contains(t, help, "/login")
	assert.Contains(t, help, "/logout")
}

func TestCommandRegistry_DispatchKeybindHelp(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	r := newTestRegistry(b)

	handled, result := r.Dispatch("/keybind-help")
	require.True(t, handled)
	assert.True(t, result.ShowKeybindings)
}
