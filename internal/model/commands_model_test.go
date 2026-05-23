package model

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weave-agent/weave/bus"

	"github.com/weave-agent/weave-tui/internal/components/messages"
	"github.com/weave-agent/weave-tui/internal/components/overlays"
)

func TestModel_SlashCommandQuit(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)

	model, cmd := m.onSubmit("/quit")
	require.NotNil(t, cmd)
	assert.Empty(t, m.chat.Items())

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)

	_ = model
}

func TestModel_SlashCommandNewClearsChat(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)

	m.AddUserMessage("hello")
	m.prompted = true

	model, _ := m.onSubmit("/new")
	m2 := model.(Model)

	assert.Empty(t, m2.chat.Items())
	assert.False(t, m2.prompted)
	assert.Empty(t, m2.toolPanels)
}

func TestModel_SlashCommandNewPublishesAgentReset(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, "agent.reset")

	m := newModel(b, nil, nil, nil)
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)
	m.prompted = true

	_, _ = m.onSubmit("/new")

	evt := <-ch
	assert.Equal(t, "agent.reset", evt.Topic)
}

func TestModel_SlashCommandClear(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)

	m.AddUserMessage("hello")
	m.prompted = true

	model, _ := m.onSubmit("/clear")
	m2 := model.(Model)

	assert.Empty(t, m2.chat.Items())
	assert.False(t, m2.prompted)
}

func TestModel_SlashCommandHelpShowsMessage(t *testing.T) {
	m := newModelNoLanding()
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)

	model, _ := m.onSubmit("/help")
	m2 := model.(Model)

	items := m2.chat.Items()
	require.Len(t, items, 1)

	am, ok := items[0].(*messages.AssistantMessage)
	require.True(t, ok)
	assert.Contains(t, am.Content(), "Available commands")

	view := am.View(80)
	assert.Contains(t, view, "◆")
}

func TestModel_RegularSubmitPublishesPrompt(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, topicPrompt)

	m := newModel(b, nil, nil, nil)
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)

	model, cmd := m.onSubmit("hello world")
	require.NotNil(t, cmd)

	msg := cmd()
	assert.NotNil(t, msg)

	evt := <-ch
	assert.Equal(t, "hello world", evt.Payload)

	m2 := model.(Model)
	assert.True(t, m2.prompted)
}

func TestModel_RegularSubmitFollowup(t *testing.T) {
	b := bus.New()
	defer func() { _ = b.Close() }()

	ch := subscribeToChan(b, topicFollowup)

	m := newModel(b, nil, nil, nil)
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)
	m.prompted = true

	model, cmd := m.onSubmit("follow up text")
	require.NotNil(t, cmd)

	msg := cmd()
	assert.NotNil(t, msg)

	evt := <-ch
	assert.Equal(t, "follow up text", evt.Payload)

	m2 := model.(Model)
	assert.True(t, m2.prompted)
}

func TestModel_UnknownCommandShowsError(t *testing.T) {
	m := newModelNoLanding()
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)

	model, _ := m.onSubmit("/bogus")
	m2 := model.(Model)

	items := m2.chat.Items()
	require.Len(t, items, 1)

	am, ok := items[0].(*messages.AssistantMessage)
	require.True(t, ok)
	assert.Contains(t, am.Content(), "unknown command: /bogus")

	view := am.View(80)
	assert.Contains(t, view, "◆")
}

func TestModel_ThinkingCommandRegistered(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	_, ok := m.commands.Lookup("/thinking")
	assert.True(t, ok, "/thinking command should be registered")
}

func TestModel_ThinkingCommandInHelp(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 10
	m.chat = m.chat.SetSize(80, 10)

	model, _ := m.onSubmit("/help")
	m2 := model.(Model)

	items := m2.chat.Items()
	require.Len(t, items, 1)
	am, ok := items[0].(*messages.AssistantMessage)
	require.True(t, ok)
	assert.Contains(t, am.Content(), "/thinking")
}

func TestModel_ThemeCommandRegistered(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	m := newModel(nil, nil, nil, nil)
	info, ok := m.commands.Lookup("/theme")
	require.True(t, ok, "/theme command should be registered")
	assert.Equal(t, "Select TUI theme", info.Description)
	assert.False(t, info.AcceptsFiles)
}

func TestModel_ThemeCommandOpensSelector(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	info, ok := m.commands.Lookup("/theme")
	require.True(t, ok)

	cmd := info.Handler("").Command
	require.NotNil(t, cmd)

	model, _ := m.Update(cmd())
	m2 := model.(Model)
	require.False(t, m2.dialogStack.Empty())
	assert.Equal(t, dialogThemeSelect, m2.dialogStack.Peek().ID())

	dlg, ok := m2.dialogStack.Peek().(*overlays.SelectorDialog)
	require.True(t, ok)
	assert.Equal(t, 0, dlg.Model().Cursor())
	assert.Contains(t, dlg.Model().View(), "default")
	assert.Contains(t, dlg.Model().View(), "built-in")
}

func TestModel_ThemeSelectorIncludesBuiltInAndUserThemes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	theme := startupTestTheme("#445566")
	theme["name"] = "user-theme"
	writeStartupTheme(t, home, "user-theme", theme)

	m := NewModelWithConfig(nil, nil, nil, nil, TUIConfig{Theme: "user-theme"})
	m.width = 80
	m.height = 24

	info, ok := m.commands.Lookup("/theme")
	require.True(t, ok)

	cmd := info.Handler("").Command
	require.NotNil(t, cmd)

	model, _ := m.Update(cmd())
	m2 := model.(Model)
	require.False(t, m2.dialogStack.Empty())

	dlg, ok := m2.dialogStack.Peek().(*overlays.SelectorDialog)
	require.True(t, ok)

	view := dlg.Model().View()
	assert.Contains(t, view, "default")
	assert.Contains(t, view, "built-in")
	assert.Contains(t, view, "user-theme")
	assert.Contains(t, view, "user")
	assert.Equal(t, 1, dlg.Model().Cursor())
}
