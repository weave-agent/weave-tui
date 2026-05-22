package keybindings

import (
	"errors"
	"strconv"
	"strings"

	"github.com/weave-agent/weave-tui/internal/components/overlays"
	"github.com/weave-agent/weave-tui/internal/palette"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mattn/go-runewidth"
)

const dialogKeybindingsHelp = "keybindings-help"

// DialogKeybindingsHelp is the dialog ID for the keybindings help overlay.
const DialogKeybindingsHelp = dialogKeybindingsHelp

type keybindingsHelpDialog struct {
	id       string
	bindings []Binding
	width    int
	height   int
	scroll   int
	done     bool
	result   overlays.DialogResult
}

func newKeybindingsHelpDialog(id string, bindings []Binding) *keybindingsHelpDialog {
	return &keybindingsHelpDialog{id: id, bindings: bindings}
}

// NewHelpDialog creates the keybindings help overlay dialog.
func NewHelpDialog(id string, bindings []Binding) overlays.Dialog {
	return newKeybindingsHelpDialog(id, bindings)
}

func (d *keybindingsHelpDialog) ID() string { return d.id }

func (d *keybindingsHelpDialog) Done() bool { return d.done }

func (d *keybindingsHelpDialog) Result() overlays.DialogResult { return d.result }

func (d *keybindingsHelpDialog) SetSize(width, height int) overlays.Dialog {
	d.width = width
	d.height = height

	return d
}

func (d *keybindingsHelpDialog) Handles(msg tea.Msg) bool {
	_, ok := msg.(tea.KeyPressMsg)
	return ok
}

func (d *keybindingsHelpDialog) Update(msg tea.Msg) (overlays.Dialog, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return d, nil
	}

	switch key.Code {
	case tea.KeyEsc:
		d.done = true
		d.result = overlays.DialogResult{Err: errors.New("canceled")}
	case tea.KeyUp:
		d.scroll = max(0, d.scroll-1)
	case tea.KeyDown:
		d.scroll = min(d.maxScroll(), d.scroll+1)
	case tea.KeyPgUp:
		d.scroll = max(0, d.scroll-d.pageSize())
	case tea.KeyPgDown:
		d.scroll = min(d.maxScroll(), d.scroll+d.pageSize())
	case tea.KeyHome:
		d.scroll = 0
	case tea.KeyEnd:
		d.scroll = d.maxScroll()
	}

	return d, nil
}

func (d *keybindingsHelpDialog) Draw(scr uv.Screen, area uv.Rectangle) {
	if area.Dx() <= 0 || area.Dy() <= 0 {
		return
	}

	uv.NewStyledString(d.View()).Draw(scr, area)
}

func (d *keybindingsHelpDialog) View() string {
	if d.width < 4 || d.height < 4 {
		return ""
	}

	theme := palette.DefaultTheme()
	boxWidth := min(88, max(36, d.width-4))
	boxHeight := min(max(10, d.height-4), max(1, d.height))
	contentWidth := max(1, boxWidth-4)
	contentHeight := max(1, boxHeight-4)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Foreground)).
		Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.AccentBright))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Foreground))
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Accent)).
		Width(boxWidth-2).
		Padding(0, 1)

	rows := d.rows(contentWidth, keyStyle, descStyle, actionStyle)
	visibleRows := rows[d.scroll:min(len(rows), d.scroll+contentHeight)]

	footer := "esc close · ↑/↓ scroll"
	if len(rows) > contentHeight {
		footer = footer + " · " + d.scrollPosition(len(rows), contentHeight)
	}

	lines := append([]string{titleStyle.Render("Keybindings")}, visibleRows...)
	for len(lines) < contentHeight+1 {
		lines = append(lines, "")
	}

	lines = append(lines, footerStyle.Render(footer))

	box := borderStyle.Render(strings.Join(lines, "\n"))
	boxLines := strings.Split(box, "\n")

	return lipgloss.NewStyle().
		MarginTop(max(0, (d.height-len(boxLines))/2)).
		MarginLeft(max(0, (d.width-boxWidth)/2)).
		Render(strings.Join(boxLines, "\n"))
}

func (d *keybindingsHelpDialog) rows(width int, keyStyle, descStyle, actionStyle lipgloss.Style) []string {
	if len(d.bindings) == 0 {
		return []string{"No keybindings registered"}
	}

	keyWidth := 0
	for _, binding := range d.bindings {
		keyWidth = max(keyWidth, lipgloss.Width(strings.Join(binding.Keys, ", ")))
	}

	keyWidth = min(keyWidth, max(10, width/2))
	descriptionWidth := max(8, width-keyWidth-3)

	rows := make([]string, 0, len(d.bindings))
	for _, binding := range d.bindings {
		keys := truncateDisplayWidth(strings.Join(binding.Keys, ", "), keyWidth)

		description := binding.Description
		if description == "" {
			description = string(binding.Action)
		}

		description = truncateDisplayWidth(description, descriptionWidth)

		row := lipgloss.NewStyle().Width(keyWidth).Render(keyStyle.Render(keys)) + "  " + descStyle.Render(description)
		if binding.Description != "" && lipgloss.Width(row)+lipgloss.Width(string(binding.Action))+3 <= width {
			row += actionStyle.Render(" · " + string(binding.Action))
		}

		rows = append(rows, row)
	}

	return rows
}

func (d *keybindingsHelpDialog) pageSize() int {
	return max(1, min(max(10, d.height-4), d.height)-4)
}

func (d *keybindingsHelpDialog) maxScroll() int {
	return max(0, len(d.bindings)-d.pageSize())
}

func (d *keybindingsHelpDialog) scrollPosition(rowCount, pageSize int) string {
	return strconv.Itoa(min(rowCount, d.scroll+pageSize)) + "/" + strconv.Itoa(rowCount)
}

func truncateDisplayWidth(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	var b strings.Builder

	w := 0
	inEsc := false

	for _, r := range s {
		if inEsc {
			b.WriteRune(r)

			if r == 'm' {
				inEsc = false
			}

			continue
		}

		if r == '\x1b' {
			inEsc = true

			b.WriteRune(r)

			continue
		}

		rw := runewidth.RuneWidth(r)
		if w+rw > maxWidth {
			break
		}

		w += rw

		b.WriteRune(r)
	}

	return b.String()
}
