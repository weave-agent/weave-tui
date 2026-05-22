package keybindings

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeHelpBindings(count int) []Binding {
	bindings := make([]Binding, 0, count)
	for i := range count {
		bindings = append(bindings, Binding{
			Action:      BindingAction("test.action." + string(rune('a'+i))),
			Keys:        []string{"ctrl+x"},
			Description: "Test action description",
		})
	}

	return bindings
}

func TestHelpDialogEscCancels(t *testing.T) {
	dialog := newKeybindingsHelpDialog(DialogKeybindingsHelp, nil)

	updated, cmd := dialog.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	updatedDialog, ok := updated.(*keybindingsHelpDialog)
	require.True(t, ok)

	assert.Nil(t, cmd)
	assert.True(t, updatedDialog.Done())
	require.Error(t, updatedDialog.Result().Err)
	assert.Contains(t, updatedDialog.Result().Err.Error(), "canceled")
}

func TestHelpDialogScrollClampsToBounds(t *testing.T) {
	dialog := newKeybindingsHelpDialog(DialogKeybindingsHelp, makeHelpBindings(20))
	dialog.SetSize(80, 12)

	_, _ = dialog.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	assert.Zero(t, dialog.scroll)

	for range 50 {
		_, _ = dialog.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}

	assert.Equal(t, dialog.maxScroll(), dialog.scroll)

	_, _ = dialog.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	assert.LessOrEqual(t, 0, dialog.scroll)

	_, _ = dialog.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	assert.Equal(t, dialog.maxScroll(), dialog.scroll)

	_, _ = dialog.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	assert.Zero(t, dialog.scroll)
}

func TestHelpDialogViewEmptyTinyAndTruncated(t *testing.T) {
	emptyDialog := newKeybindingsHelpDialog(DialogKeybindingsHelp, nil)
	emptyDialog.SetSize(80, 20)
	assert.Contains(t, emptyDialog.View(), "No keybindings registered")

	tinyDialog := newKeybindingsHelpDialog(DialogKeybindingsHelp, nil)
	tinyDialog.SetSize(3, 3)
	assert.Empty(t, tinyDialog.View())

	longDialog := newKeybindingsHelpDialog(DialogKeybindingsHelp, []Binding{
		{
			Action:      ActionExternalEditor,
			Keys:        []string{"ctrl+super+shift+alt+very-long-key"},
			Description: "A very long description with ANSI \x1b[31mcolored\x1b[0m text",
		},
	})
	longDialog.SetSize(40, 12)

	view := longDialog.View()
	assert.NotEmpty(t, view)
	assert.NotContains(t, view, "very-long-key")
	assert.LessOrEqual(t, widestLine(view), 40)
}

func widestLine(s string) int {
	width := 0
	for line := range strings.SplitSeq(s, "\n") {
		width = max(width, lipgloss.Width(line))
	}

	return width
}
