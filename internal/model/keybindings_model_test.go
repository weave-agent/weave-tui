package model

import (
	"testing"
	"time"

	tuievents "github.com/weave-agent/weave-tui/internal/events"
	tuikeybindings "github.com/weave-agent/weave-tui/internal/keybindings"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_BindingsRegistryInitialized(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	assert.NotNil(t, m.bindings)

	action, ok := m.bindings.Resolve("ctrl+d")
	require.True(t, ok)
	assert.Equal(t, tuikeybindings.ActionExit, action)
}

func TestModel_CtrlDExitsViaBinding(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "ctrl+d should quit via keybinding")
}

func TestModel_CtrlCClearsEditorOnFirstPress(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	m.editor = m.editor.SetValue("some text")

	model, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = model.(Model)

	assert.NotNil(t, cmd)
	assert.Empty(t, m.editor.Value())
}

func TestModel_EscapeNoOpViaBinding(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	model, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.NotNil(t, cmd)

	_ = model
}

func TestModel_CtrlLOpensModelSelectorViaBinding(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	model, cmd := m.Update(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)

	_ = model
}

func TestModel_ExtensionKeybinding(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	customAction := tuikeybindings.BindingAction("app.test.custom")
	m.bindings.Register(customAction, []string{"ctrl+f"}, "Custom test action")

	action, ok := m.bindings.Resolve("ctrl+f")
	require.True(t, ok)
	assert.Equal(t, customAction, action)

	model, cmd := m.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	assert.Nil(t, cmd, "unhandled action should return nil cmd")

	_ = model
}

func TestModel_OverlayDismissStillWorks(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	sessions := []tuievents.SessionEntry{
		{ID: "aaa11122233344455566677788899900", CWD: "/project", CreatedAt: time.Now()},
	}
	model, _ := m.Update(tuievents.SessionListResultMsg{Sessions: sessions})
	m = model.(Model)
	assert.False(t, m.dialogStack.Empty())

	model, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = model.(Model)
	assert.True(t, m.dialogStack.Empty())
	assert.Nil(t, cmd, "overlay dismiss should not quit")
}

func TestModel_V2KeyDispatch_CtrlD(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "ctrl+d should quit via keybinding")
}

func TestModel_V2KeyDispatch_CtrlCClearsEditor(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24
	m.editor = m.editor.SetValue("some text")

	model, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = model.(Model)

	assert.NotNil(t, cmd)
	assert.Empty(t, m.editor.Value())
}

func TestModel_V2KeyDispatch_EscapeInterrupts(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	model, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.NotNil(t, cmd)

	_ = model
}

func TestModel_V2KeyDispatch_PrintableChars(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	model, cmd := m.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	m = model.(Model)

	assert.NotNil(t, cmd)
	assert.Equal(t, "h", m.editor.Value())
}

func TestModel_V2KeyDispatch_MultiRuneText(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 80
	m.height = 24

	model, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyExtended, Text: "abc"})
	m = model.(Model)

	assert.NotNil(t, cmd)
	assert.Equal(t, "abc", m.editor.Value())
}

func TestModel_KeybindHelpCommandOpensDialog(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 100
	m.height = 30

	model, cmd := m.onSubmit("/keybind-help")
	m = model.(Model)

	assert.Nil(t, cmd)
	require.False(t, m.dialogStack.Empty())
	assert.Equal(t, tuikeybindings.DialogKeybindingsHelp, m.dialogStack.Peek().ID())
}

func TestModel_KeybindingsHelpRendersResolvedBindings(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 100
	m.height = 30
	m = m.showKeybindingsHelp()

	canvas := uv.NewScreenBuffer(m.width, m.height)
	m.Draw(canvas, canvas.Bounds())
	rendered := canvas.Render()

	assert.Contains(t, rendered, "Keybindings")
	assert.Contains(t, rendered, "ctrl+d")
}

func TestModel_KeybindingsHelpEscapeClosesDialog(t *testing.T) {
	m := newModel(nil, nil, nil, nil)
	m.width = 100
	m.height = 30
	m = m.showKeybindingsHelp()

	model, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = model.(Model)

	assert.Nil(t, cmd)
	assert.True(t, m.dialogStack.Empty())
}
