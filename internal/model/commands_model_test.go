package model

import (
	"testing"

	"github.com/weave-agent/weave-tui/internal/components/messages"
	"github.com/weave-agent/weave/bus"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
