package tui

import (
	tuicommands "github.com/weave-agent/weave-tui/internal/commands"
	tuikeybindings "github.com/weave-agent/weave-tui/internal/keybindings"
	tuisessions "github.com/weave-agent/weave-tui/internal/sessions"
	"github.com/weave-agent/weave/sdk"

	tea "charm.land/bubbletea/v2"
)

type CommandResult = tuicommands.CommandResult
type CommandRegistry = tuicommands.CommandRegistry
type Binding = tuikeybindings.Binding
type BindingAction = tuikeybindings.BindingAction
type BindingRegistry = tuikeybindings.BindingRegistry

const (
	ActionExit               = tuikeybindings.ActionExit
	ActionInterrupt          = tuikeybindings.ActionInterrupt
	ActionModelSelect        = tuikeybindings.ActionModelSelect
	ActionModelCycle         = tuikeybindings.ActionModelCycle
	ActionCursorLineStart    = tuikeybindings.ActionCursorLineStart
	ActionCursorLineEnd      = tuikeybindings.ActionCursorLineEnd
	ActionCursorWordLeft     = tuikeybindings.ActionCursorWordLeft
	ActionCursorWordRight    = tuikeybindings.ActionCursorWordRight
	ActionEditorNewline      = tuikeybindings.ActionEditorNewline
	ActionScrollUp           = tuikeybindings.ActionScrollUp
	ActionScrollDown         = tuikeybindings.ActionScrollDown
	ActionScrollToBottom     = tuikeybindings.ActionScrollToBottom
	ActionDeleteWordBackward = tuikeybindings.ActionDeleteWordBackward
	ActionDeleteWordForward  = tuikeybindings.ActionDeleteWordForward
	ActionDeleteToLineStart  = tuikeybindings.ActionDeleteToLineStart
	ActionDeleteToLineEnd    = tuikeybindings.ActionDeleteToLineEnd
	ActionSuspend            = tuikeybindings.ActionSuspend
	ActionExternalEditor     = tuikeybindings.ActionExternalEditor
	ActionToggleToolOutput   = tuikeybindings.ActionToggleToolOutput
	ActionThinkingCycle      = tuikeybindings.ActionThinkingCycle
	ActionNewSession         = tuikeybindings.ActionNewSession
	ActionSandboxCycle       = tuikeybindings.ActionSandboxCycle
	ActionPanelPicker        = tuikeybindings.ActionPanelPicker
	ActionCopySelection      = tuikeybindings.ActionCopySelection
)

func NewCommandRegistry(bus sdk.Bus, sessionDir string) *tuicommands.CommandRegistry {
	return tuicommands.NewCommandRegistry(bus, sessionDir, tuicommands.RuntimeCommands{
		ListSessions: tuisessions.ListCmd,
		Login:        loginCmd,
		Logout:       logoutCmd,
	})
}

func NewBindingRegistry() *tuikeybindings.BindingRegistry {
	return tuikeybindings.NewBindingRegistry()
}

func keyString(msg tea.KeyPressMsg) string {
	return tuikeybindings.KeyString(msg)
}
