package ui

import (
	"sync"
	"testing"

	tuievents "github.com/weave-agent/weave-tui/internal/events"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSender struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (s *mockSender) Send(msg tea.Msg) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.msgs = append(s.msgs, msg)
}

func (s *mockSender) At(i int) tea.Msg {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.msgs[i]
}

func (s *mockSender) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.msgs)
}

func TestTUIImplSetStatusSendsEvent(t *testing.T) {
	sender := &mockSender{}
	ui := NewTUIImpl(nil, nil)
	ui.SetProgram(sender)

	ui.SetStatus("build", "compiling")

	require.Equal(t, 1, sender.Len())
	msg, ok := sender.At(0).(tuievents.ExtStatusMsg)
	require.True(t, ok)
	assert.Equal(t, "build", msg.Key)
	assert.Equal(t, "compiling", msg.Text)
}

func TestTUIImplPopupQueueFIFO(t *testing.T) {
	sender := &mockSender{}
	ui := NewTUIImpl(nil, nil)
	ui.SetProgram(sender)

	first := &overlayRequest{Kind: requestSelect, Title: "first", Result: make(chan overlayResponse, 1)}
	second := &overlayRequest{Kind: requestConfirm, Message: "second", Result: make(chan overlayResponse, 1)}

	require.NoError(t, ui.EnqueuePopup(first))
	require.NoError(t, ui.EnqueuePopup(second))
	assert.True(t, ui.HasPendingPopups())

	assert.Equal(t, first, ui.DequeuePopup())
	assert.Equal(t, second, ui.DequeuePopup())
	assert.False(t, ui.HasPendingPopups())
	assert.Nil(t, ui.DequeuePopup())
}

func TestTUIImplSetThemeSendsEvent(t *testing.T) {
	sender := &mockSender{}
	ui := NewTUIImpl(nil, nil)
	ui.SetProgram(sender)

	require.NoError(t, ui.RegisterTheme("custom", ThemeDef{Accent: "123"}))
	require.NoError(t, ui.SetTheme("custom"))

	require.Equal(t, 1, sender.Len())
	msg, ok := sender.At(0).(tuievents.ThemeChangedMsg)
	require.True(t, ok)
	require.NotNil(t, msg.Theme)
	assert.Equal(t, "123", msg.Theme.Accent)
}
