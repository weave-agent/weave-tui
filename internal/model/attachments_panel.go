package model

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/weave-agent/weave-tui/internal/components/attachments"
	"github.com/weave-agent/weave-tui/internal/palette"
	"github.com/weave-agent/weave-tui/internal/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

const attachmentsPanelID = "attachments"

const dialogAttachmentEditor = "attachment-editor"

type editAttachmentMsg struct {
	Index int
}

type externalEditAttachmentMsg struct {
	Index int
}

type removeAttachmentMsg struct {
	Index int
}

type attachmentsPanelDrawer struct {
	Items    []attachments.Attachment
	Selected int
	Theme    *palette.Theme
}

func newAttachmentsPanelDrawer(items []attachments.Attachment, selected int, theme *palette.Theme) *attachmentsPanelDrawer {
	return &attachmentsPanelDrawer{
		Items:    items,
		Selected: normalizeAttachmentSelection(selected, len(items)),
		Theme:    theme,
	}
}

func (d *attachmentsPanelDrawer) Draw(scr uv.Screen, area uv.Rectangle) {
	if area.Dx() <= 0 || area.Dy() <= 0 {
		return
	}

	theme := d.Theme
	if theme == nil {
		theme = palette.DefaultTheme()
	}

	styleSet := styles.New(theme)
	drawLine(scr, area, 0, styleSet.MutedBright().Bold(true).Render("Attachments"))

	if len(d.Items) == 0 {
		drawLine(scr, area, 2, styleSet.Muted().Render("No attachments"))
		drawLine(scr, area, area.Dy()-1, styleSet.Muted().Render("Esc editor"))

		return
	}

	listHeight := max(area.Dy()-4, 1)
	start := visibleAttachmentStart(d.Selected, listHeight, len(d.Items))

	for row := 0; row < listHeight && start+row < len(d.Items); row++ {
		idx := start + row
		item := d.Items[idx]

		label := fmt.Sprintf("%s   %d lines", filepath.Base(item.Path), item.Lines)
		if idx == d.Selected {
			label = styleSet.SelectedRow().Render("› " + label)
		} else {
			label = styleSet.MutedBright().Render("  " + label)
		}

		drawLine(scr, area, row+2, label)
	}

	drawLine(scr, area, area.Dy()-1, styleSet.Muted().Render("Enter edit · Ctrl+G external · Del delete"))
}

func (d *attachmentsPanelDrawer) Update(msg tea.Msg) (PanelDrawer, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return d, nil
	}

	switch key.Code {
	case tea.KeyUp:
		d.Selected = normalizeAttachmentSelection(d.Selected-1, len(d.Items))
		return d, nil
	case tea.KeyDown:
		d.Selected = normalizeAttachmentSelection(d.Selected+1, len(d.Items))
		return d, nil
	case tea.KeyDelete, tea.KeyBackspace:
		idx := d.Selected
		return d, func() tea.Msg { return removeAttachmentMsg{Index: idx} }
	case tea.KeyEnter, tea.KeyKpEnter:
		idx := d.Selected
		return d, func() tea.Msg { return editAttachmentMsg{Index: idx} }
	case 'g', 'G':
		if key.Mod&tea.ModCtrl != 0 {
			idx := d.Selected
			return d, func() tea.Msg { return externalEditAttachmentMsg{Index: idx} }
		}
	}

	return d, nil
}

func (d *attachmentsPanelDrawer) Handles(msg tea.Msg) bool {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false
	}

	switch key.Code {
	case tea.KeyUp, tea.KeyDown, tea.KeyDelete, tea.KeyBackspace, tea.KeyEnter, tea.KeyKpEnter:
		return len(d.Items) > 0
	case 'g', 'G':
		return len(d.Items) > 0 && key.Mod&tea.ModCtrl != 0
	}

	return false
}

func visibleAttachmentStart(selected, height, count int) int {
	if height <= 0 || count <= height {
		return 0
	}

	if selected < 0 {
		return 0
	}

	if selected >= count {
		selected = count - 1
	}

	if selected < height {
		return 0
	}

	return selected - height + 1
}

func normalizeAttachmentSelection(selected, count int) int {
	if count <= 0 {
		return 0
	}

	if selected < 0 {
		return count - 1
	}

	if selected >= count {
		return 0
	}

	return selected
}

func drawLine(scr uv.Screen, area uv.Rectangle, offset int, text string) {
	if offset < 0 || offset >= area.Dy() {
		return
	}

	line := truncateDisplayWidth(text, area.Dx())
	padding := max(area.Dx()-lipgloss.Width(line), 0)
	uv.NewStyledString(line+strings.Repeat(" ", padding)).Draw(scr, uv.Rect(area.Min.X, area.Min.Y+offset, area.Dx(), 1))
}
