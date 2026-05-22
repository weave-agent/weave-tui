package overlays

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEditorModel(t *testing.T) {
	m := NewEditorModel("Edit note:", "hello")
	assert.Equal(t, "Edit note:", m.title)
	assert.Equal(t, "hello", m.Value())
	assert.False(t, m.Visible())
}

func TestEditorShowHide(t *testing.T) {
	m := NewEditorModel("Test", "")
	assert.False(t, m.Visible())

	m = m.Show()
	assert.True(t, m.Visible())

	m = m.Hide()
	assert.False(t, m.Visible())
}

func TestEditorSetSize(t *testing.T) {
	m := NewEditorModel("Test", "")
	m = m.SetSize(80, 24)
	assert.Equal(t, 80, m.Width())
	assert.Equal(t, 24, m.Height())
}

func TestEditorTyping(t *testing.T) {
	m := NewEditorModel("Note:", "").Show()

	m, _ = m.Update(tea.KeyPressMsg{Text: "hi", Code: tea.KeyExtended})
	assert.Equal(t, "hi", m.Value())
}

func TestEditorEnterInsertsNewline(t *testing.T) {
	m := NewEditorModel("Note:", "").Show()
	m, _ = m.Update(tea.KeyPressMsg{Text: "line1", Code: tea.KeyExtended})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Contains(t, m.Value(), "\n")
	assert.Contains(t, m.Value(), "line1")
}

func TestEditorCtrlSSubmits(t *testing.T) {
	m := NewEditorModel("Note:", "").Show()
	m, _ = m.Update(tea.KeyPressMsg{Text: "content", Code: tea.KeyExtended})

	m, cmd := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	assert.False(t, m.Visible())

	msg := cmd()
	result, ok := msg.(EditorResultMsg)
	require.True(t, ok)
	assert.Equal(t, "content", result.Value)
	assert.True(t, result.Ok)
}

func TestEditorShiftEnterStaysOpen(t *testing.T) {
	m := NewEditorModel("Note:", "").Show()
	m, _ = m.Update(tea.KeyPressMsg{Text: "content", Code: tea.KeyExtended})

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	assert.Nil(t, cmd)
	assert.True(t, m.Visible())
}

func TestEditorEscapeCancels(t *testing.T) {
	m := NewEditorModel("Note:", "").Show()
	m, _ = m.Update(tea.KeyPressMsg{Text: "x", Code: 'x'})

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	require.NotNil(t, cmd)
	assert.False(t, m.Visible())

	msg := cmd()
	result, ok := msg.(EditorResultMsg)
	require.True(t, ok)
	assert.False(t, result.Ok)
	assert.Empty(t, result.Value)
}

func TestEditorPrefill(t *testing.T) {
	m := NewEditorModel("Edit:", "prefilled text").Show()
	assert.Equal(t, "prefilled text", m.Value())
}

func TestEditorViewInvisible(t *testing.T) {
	m := NewEditorModel("Test", "")
	assert.Empty(t, m.View())
}

func TestEditorViewVisible(t *testing.T) {
	m := NewEditorModel("Edit note:", "").Show().SetSize(60, 20)
	view := m.View()
	assert.Contains(t, view, "Edit note:")
	assert.Contains(t, view, "Ctrl+S")
	assert.Contains(t, view, "cancel")
}

func TestEditorViewZeroWidth(t *testing.T) {
	m := NewEditorModel("Test", "").Show()
	assert.Empty(t, m.View())
}

func TestEditorView_StaysWithinSmallScreen(t *testing.T) {
	m := NewEditorModel("Edit:", "hello").Show().SetSize(30, 10)
	view := m.View()

	assert.LessOrEqual(t, lipgloss.Width(view), 30)
	assert.LessOrEqual(t, lipgloss.Height(view), 10)
}

func TestEditorDraw_Visible(t *testing.T) {
	m := NewEditorModel("Edit:", "").Show().SetSize(60, 20)
	canvas := uv.NewScreenBuffer(60, 20)
	m.Draw(canvas, canvas.Bounds())
	output := uv.TrimSpace(canvas.Render())
	assert.Contains(t, output, "Edit:")
	assert.Contains(t, output, "Ctrl+S")
}

func TestEditorDraw_WithText(t *testing.T) {
	m := NewEditorModel("Edit:", "").Show().SetSize(60, 20)
	m, _ = m.Update(tea.KeyPressMsg{Text: "hello", Code: tea.KeyExtended})
	canvas := uv.NewScreenBuffer(60, 20)
	m.Draw(canvas, canvas.Bounds())
	output := uv.TrimSpace(canvas.Render())
	assert.Contains(t, output, "hello")
}

func TestEditorDraw_Invisible(t *testing.T) {
	m := NewEditorModel("Test", "")
	canvas := uv.NewScreenBuffer(60, 20)
	m.Draw(canvas, canvas.Bounds())
	// Draw is a no-op when invisible
}

func TestEditorDraw_ZeroArea(t *testing.T) {
	m := NewEditorModel("Test", "").Show().SetSize(60, 20)
	canvas := uv.NewScreenBuffer(60, 20)
	m.Draw(canvas, uv.Rect(0, 0, 0, 0))
}

func TestEditorView_StyledWithRoundedBorder(t *testing.T) {
	m := NewEditorModel("Edit:", "").Show().SetSize(60, 20)
	view := m.View()
	assert.Contains(t, view, "Edit:")
	assert.Contains(t, view, "Ctrl+S")
	// Rounded border should be present
	assert.Contains(t, view, "╭")
}
