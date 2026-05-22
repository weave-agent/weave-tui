package tui

import (
	"testing"
	"time"

	"github.com/weave-agent/weave/sdk"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBanner_EphemeralInfo(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	updated, cmd := m.Update(notifyTypedMsg{message: "info banner", level: sdk.NotifyInfo})
	m = updated.(Model)

	assert.Equal(t, "info banner", m.bannerMsg)
	assert.Equal(t, sdk.NotifyInfo, m.bannerLevel)
	assert.NotZero(t, m.bannerGen)
	require.NotNil(t, cmd)

	// Simulate timeout
	msg := executeCmd(t, cmd)
	bto, ok := msg.(bannerTimeoutMsg)
	require.True(t, ok)
	assert.Equal(t, m.bannerGen, bto.gen)

	updated, _ = m.Update(bto)
	m = updated.(Model)
	assert.Empty(t, m.bannerMsg)
	assert.Zero(t, m.bannerLevel)
}

func TestBanner_EphemeralSuccess(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	updated, cmd := m.Update(notifyTypedMsg{message: "success banner", level: sdk.NotifySuccess})
	m = updated.(Model)

	assert.Equal(t, "success banner", m.bannerMsg)
	assert.Equal(t, sdk.NotifySuccess, m.bannerLevel)
	require.NotNil(t, cmd)
}

func TestBanner_PersistentWarning(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	updated, cmd := m.Update(notifyTypedMsg{message: "warning banner", level: sdk.NotifyWarning})
	m = updated.(Model)

	assert.Equal(t, "warning banner", m.bannerMsg)
	assert.Equal(t, sdk.NotifyWarning, m.bannerLevel)
	assert.Nil(t, cmd) // no timer for persistent banners
}

func TestBanner_PersistentError(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	updated, cmd := m.Update(notifyTypedMsg{message: "error banner", level: sdk.NotifyError})
	m = updated.(Model)

	assert.Equal(t, "error banner", m.bannerMsg)
	assert.Equal(t, sdk.NotifyError, m.bannerLevel)
	assert.Nil(t, cmd)
}

func TestBanner_StaleTimeoutIgnored(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	// First banner
	updated, _ := m.Update(notifyTypedMsg{message: "first", level: sdk.NotifyInfo})
	m = updated.(Model)
	oldGen := m.bannerGen

	// Second banner replaces first
	updated, _ = m.Update(notifyTypedMsg{message: "second", level: sdk.NotifyInfo})
	m = updated.(Model)
	assert.Equal(t, "second", m.bannerMsg)

	// Stale timeout from first banner should be ignored
	updated, _ = m.Update(bannerTimeoutMsg{gen: oldGen})
	m = updated.(Model)
	assert.Equal(t, "second", m.bannerMsg)
}

func TestBanner_DismissOnSubmit(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24
	m.prompted = true
	m.editor = m.editor.SetValue("hello")

	// Set a persistent warning banner
	updated, _ := m.Update(notifyTypedMsg{message: "warning", level: sdk.NotifyWarning})
	m = updated.(Model)
	require.Equal(t, "warning", m.bannerMsg)

	// Submit should dismiss persistent banner
	updated, _ = m.onSubmit("hello")
	m = updated.(Model)
	assert.Empty(t, m.bannerMsg)
}

func TestBanner_DismissOnPaste(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24
	m.chat = m.chat.SetSize(80, 10)

	// Set a persistent error banner
	updated, _ := m.Update(notifyTypedMsg{message: "error", level: sdk.NotifyError})
	m = updated.(Model)
	require.Equal(t, "error", m.bannerMsg)

	// Paste should dismiss persistent banner
	updated, _ = m.Update(tea.PasteMsg{Content: "short"})
	m = updated.(Model)
	assert.Empty(t, m.bannerMsg)
}

func TestBanner_DismissOnEditorContentChange(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24
	m.chat = m.chat.SetSize(80, 10)

	// Set a persistent error banner
	updated, _ := m.Update(notifyTypedMsg{message: "error", level: sdk.NotifyError})
	m = updated.(Model)
	require.Equal(t, "error", m.bannerMsg)

	// Typing a character should dismiss persistent banner
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = updated.(Model)
	assert.Empty(t, m.bannerMsg)
}

func TestBanner_EphemeralNotDismissedOnUserAction(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24
	m.chat = m.chat.SetSize(80, 10)

	// Set an ephemeral info banner
	updated, _ := m.Update(notifyTypedMsg{message: "info", level: sdk.NotifyInfo})
	m = updated.(Model)
	require.Equal(t, "info", m.bannerMsg)

	// Typing should NOT dismiss ephemeral banners (they have their own timer)
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = updated.(Model)
	assert.Equal(t, "info", m.bannerMsg)
}

func TestBanner_ReplaceExistingBanner(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	// First banner
	updated, _ := m.Update(notifyTypedMsg{message: "first", level: sdk.NotifyWarning})
	m = updated.(Model)
	gen1 := m.bannerGen

	// Second banner replaces first
	updated, _ = m.Update(notifyTypedMsg{message: "second", level: sdk.NotifyError})
	m = updated.(Model)

	assert.Equal(t, "second", m.bannerMsg)
	assert.Equal(t, sdk.NotifyError, m.bannerLevel)
	assert.Greater(t, m.bannerGen, gen1)
}

func TestBanner_LandingHidden(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24
	m.showLanding = true

	updated, _ := m.Update(notifyMsg{message: "hello"})
	m = updated.(Model)

	assert.False(t, m.showLanding)
}

func TestBannerMarkerForLevel(t *testing.T) {
	assert.Equal(t, "i", bannerMarkerForLevel(sdk.NotifyInfo))
	assert.Equal(t, "✓", bannerMarkerForLevel(sdk.NotifySuccess))
	assert.Equal(t, "!", bannerMarkerForLevel(sdk.NotifyWarning))
	assert.Equal(t, "×", bannerMarkerForLevel(sdk.NotifyError))
	assert.Equal(t, "i", bannerMarkerForLevel(999)) // unknown level defaults to info
}

func TestBanner_CountLayoutRows(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.statusMsg = "" // clear default status so we have a clean baseline

	// No banner = no extra pill rows
	header, pills := m.countLayoutRows()
	assert.Equal(t, 0, header)
	assert.Equal(t, 0, pills)

	// With banner = one extra pill row
	m.bannerMsg = "test"
	header, pills = m.countLayoutRows()
	assert.Equal(t, 0, header)
	assert.Equal(t, 1, pills)

	// Banner + status = two pill rows
	m.statusMsg = "status"
	header, pills = m.countLayoutRows()
	assert.Equal(t, 0, header)
	assert.Equal(t, 2, pills)
}

func executeCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	require.NotNil(t, cmd)
	msg := cmd()
	// Handle batch commands by returning the first non-nil message
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				m := c()
				if m != nil {
					return m
				}
			}
		}
	}
	return msg
}

// AdvanceCmdTicks advances time-based tea.Cmd functions by the given duration.
func AdvanceCmdTicks(cmd tea.Cmd, duration time.Duration) tea.Msg {
	if cmd == nil {
		return nil
	}

	// Tea tick commands use time.After internally; we can't easily advance
	// them in tests. Return the message directly for tick-based commands.
	return cmd()
}
