package model

import (
	"fmt"

	"github.com/weave-agent/weave-tui/internal/components/messages"
	"github.com/weave-agent/weave-tui/internal/components/overlays"

	tea "charm.land/bubbletea/v2"
)

// Dialog IDs for built-in overlays.
const (
	dialogSessionSelect  = "session-select"
	dialogModelSelect    = "model-select"
	dialogProviderSelect = "provider-select"
	dialogKeyInput       = "key-input"
	dialogLoginSelect    = "login-select"
	dialogLogoutSelect   = "logout-select"
	dialogLoginOAuth     = "login-oauth"
	dialogThemeSelect    = "theme-select"
)

// checkNextPopupCmd returns a tea.Cmd that sends popupPendingMsg
// if there are more queued popups.
func checkNextPopupCmd(ui *TUIImpl) tea.Cmd {
	if ui != nil && ui.HasPendingPopups() {
		return func() tea.Msg { return popupPendingMsg{} }
	}

	return nil
}

// newNotifyAssistantMsg creates a finalized assistant message for notifications.
func newNotifyAssistantMsg(text string) *messages.AssistantMessage {
	am := messages.NewAssistantMessage()
	am.Finalize(text)

	return am
}

// nextPopupDialogID generates a unique ID for popup dialog instances.
func nextPopupDialogID(kind overlayRequestKind, seq *int) string {
	*seq++

	var prefix string

	switch kind {
	case requestSelect:
		prefix = "popup-select"
	case requestConfirm:
		prefix = "popup-confirm"
	case requestInput:
		prefix = "popup-input"
	case requestEditor:
		prefix = "popup-editor"
	case requestMultiSelect:
		prefix = "popup-multiselect"
	}

	return fmt.Sprintf("%s-%d", prefix, *seq)
}

// pushPopupDialog creates a dialog for a popup request and pushes it onto the stack.
func pushPopupDialog(m Model, req *overlayRequest) (Model, tea.Cmd) {
	id := nextPopupDialogID(req.Kind, &m.popupSeq)
	m.popupChans[id] = req.Result
	m.dockedOverlay = req.KeepContent

	dialogWidth := m.width

	dialogHeight := m.height
	if req.KeepContent {
		dialogHeight = dockedOverlayHeight
	}

	switch req.Kind {
	case requestSelect:
		items := make([]overlays.SelectorItem, len(req.Items))
		for i, title := range req.Items {
			items[i] = overlays.SelectorItem{Title: title}
		}

		sel := overlays.NewSelectorModel(req.Title, items).SetStyles(m.styles)
		sel = sel.SetSize(dialogWidth, dialogHeight)
		sel = sel.Show()

		m.dialogStack = m.dialogStack.Push(overlays.NewSelectorDialog(id, sel))

	case requestConfirm:
		conf := overlays.NewConfirmModel(req.Message)
		conf = conf.SetSize(dialogWidth, dialogHeight)
		conf = conf.SetTheme(m.theme)
		conf = conf.Show()

		m.dialogStack = m.dialogStack.Push(overlays.NewConfirmDialog(id, conf))

	case requestInput:
		input := overlays.NewInputModel(req.Message)
		input = input.SetSize(dialogWidth, dialogHeight)
		input = input.SetMask(req.Mask)
		input = input.SetTheme(m.theme)
		input = input.Show()

		m.dialogStack = m.dialogStack.Push(overlays.NewInputDialog(id, input))

	case requestEditor:
		editor := overlays.NewEditorModel(req.Title, req.Initial)
		editor = editor.SetSize(dialogWidth, dialogHeight)
		editor = editor.SetTheme(m.theme)
		editor = editor.Show()

		m.dialogStack = m.dialogStack.Push(overlays.NewEditorDialog(id, editor))

	case requestMultiSelect:
		ms := overlays.NewMultiSelectModel(req.Title, req.Items, req.Defaults)
		ms = ms.SetSize(dialogWidth, dialogHeight)
		ms = ms.SetTheme(m.theme)
		ms = ms.Show()

		m.dialogStack = m.dialogStack.Push(overlays.NewMultiSelectDialog(id, ms))
	}

	return m, nil
}

// handlePopupPending processes queued popup requests by pushing them onto the dialog stack.
func (m Model) handlePopupPending() (Model, tea.Cmd) {
	if m.ui == nil {
		return m, nil
	}

	req := m.ui.DequeuePopup()
	if req == nil {
		return m, nil
	}

	return pushPopupDialog(m, req)
}
