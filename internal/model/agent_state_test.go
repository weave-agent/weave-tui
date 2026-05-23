package model

import (
	"testing"

	tuievents "github.com/weave-agent/weave-tui/internal/events"
	"github.com/weave-agent/weave-tui/internal/palette"
	"github.com/weave-agent/weave-tui/internal/themecatalog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_AgentStateChangeUpdatesTheme(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	originalAccent := m.theme.Accent

	// Streaming state
	model, _ := m.Update(tuievents.AgentStateChangeMsg{State: palette.StateStreaming})
	m = model.(Model)

	assert.Equal(t, palette.StateStreaming, m.agentState)
	assert.NotEqual(t, originalAccent, m.theme.Accent)
	assert.Equal(t, "45", m.theme.Accent) // Streaming accent color

	// ToolRunning state
	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.StateToolRunning})
	m = model.(Model)

	assert.Equal(t, palette.StateToolRunning, m.agentState)
	assert.Equal(t, "172", m.theme.Accent) // ToolRunning accent color

	// Error state
	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.StateError})
	m = model.(Model)

	assert.Equal(t, palette.StateError, m.agentState)
	assert.Equal(t, "167", m.theme.Accent) // Error accent color

	// Back to Idle
	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.StateIdle})
	m = model.(Model)

	assert.Equal(t, palette.StateIdle, m.agentState)
	assert.Equal(t, "245", m.theme.Accent) // Idle accent color
}

func TestModel_AgentStateChangeUsesCustomThemeAsBase(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	custom := palette.DefaultTheme()
	custom.Accent = "#112233"
	custom.AccentDim = "#223344"
	custom.AccentBright = "#334455"
	custom.Foreground = "#ddeeff"
	m.themeEntries = append(m.themeEntries, themecatalog.Entry{
		Name:   "custom-state",
		Theme:  custom,
		Source: themecatalog.SourceUser,
	})
	require.NoError(t, m.ui.RegisterPaletteTheme("custom-state", custom))

	m2, err := m.applyThemeByName("custom-state")
	require.NoError(t, err)
	m = m2

	model, _ := m.Update(tuievents.AgentStateChangeMsg{State: palette.StateStreaming})
	m = model.(Model)

	assert.Equal(t, palette.StateStreaming, m.agentState)
	assert.Equal(t, "45", m.theme.Accent)
	assert.Equal(t, "39", m.theme.AccentDim)
	assert.Equal(t, "51", m.theme.AccentBright)
	assert.Equal(t, "#ddeeff", m.theme.Foreground)

	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.StateIdle})
	m = model.(Model)

	assert.Equal(t, palette.StateIdle, m.agentState)
	assert.Equal(t, "#112233", m.theme.Accent)
	assert.Equal(t, "#223344", m.theme.AccentDim)
	assert.Equal(t, "#334455", m.theme.AccentBright)
	assert.Equal(t, "#ddeeff", m.theme.Foreground)
	assert.Equal(t, "#112233", m.styles.Theme().Accent)
	assert.Equal(t, "#334455", m.styles.Theme().AccentBright)
}

func TestModel_AgentStateChangeUnknownRestoresCustomThemeAccent(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	custom := palette.DefaultTheme()
	custom.Accent = "#102030"
	custom.AccentDim = "#203040"
	custom.AccentBright = "#304050"
	m.themeEntries = append(m.themeEntries, themecatalog.Entry{
		Name:   "custom-fallback",
		Theme:  custom,
		Source: themecatalog.SourceUser,
	})
	require.NoError(t, m.ui.RegisterPaletteTheme("custom-fallback", custom))

	m2, err := m.applyThemeByName("custom-fallback")
	require.NoError(t, err)
	m = m2

	model, _ := m.Update(tuievents.AgentStateChangeMsg{State: palette.StateToolRunning})
	m = model.(Model)
	require.Equal(t, "172", m.theme.Accent)

	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.State(-1)})
	m = model.(Model)

	assert.Equal(t, "#102030", m.theme.Accent)
	assert.Equal(t, "#203040", m.theme.AccentDim)
	assert.Equal(t, "#304050", m.theme.AccentBright)
}

func TestModel_AgentStateChangeUpdatesEditorBorder(t *testing.T) {
	m := newModel(nil, nil, nil, nil)

	model, _ := m.Update(tuievents.AgentStateChangeMsg{State: palette.StateStreaming})
	m = model.(Model)

	assert.Equal(t, "45", m.editor.BorderColor) // Streaming accent
}

func TestModel_AgentStateChangePulseActive(t *testing.T) {
	m := newModel(nil, nil, nil, nil)

	// Streaming enables pulse
	model, _ := m.Update(tuievents.AgentStateChangeMsg{State: palette.StateStreaming})
	m = model.(Model)
	assert.True(t, m.editor.PulseActive)

	// ToolRunning keeps pulse active
	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.StateToolRunning})
	m = model.(Model)
	assert.True(t, m.editor.PulseActive)

	// Idle disables pulse
	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.StateIdle})
	m = model.(Model)
	assert.False(t, m.editor.PulseActive)
	assert.Equal(t, 0, m.editor.PulsePos)

	// Error disables pulse
	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.StateStreaming})
	m = model.(Model)
	assert.True(t, m.editor.PulseActive)
	model, _ = m.Update(tuievents.AgentStateChangeMsg{State: palette.StateError})
	m = model.(Model)
	assert.False(t, m.editor.PulseActive)
}
