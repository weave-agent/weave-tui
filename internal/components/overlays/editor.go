package overlays

import (
	"github.com/weave-agent/weave-tui/internal/palette"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// EditorResultMsg is emitted when the user submits or cancels the editor.
type EditorResultMsg struct {
	Value string
	Ok    bool
}

// EditorModel is a multi-line text editor modal overlay.
type EditorModel struct {
	title   string
	ta      textarea.Model
	width   int
	height  int
	visible bool
	theme   *palette.Theme
	docked  bool
}

// NewEditorModel creates a new editor overlay model.
func NewEditorModel(title, prefill string) EditorModel {
	ta := textarea.New()
	ta.DynamicHeight = false
	ta.ShowLineNumbers = false
	ta.SetVirtualCursor(true)
	ta.Prompt = ""
	ta.Placeholder = ""
	ta.Focus()

	styles := textarea.DefaultStyles(false)
	styles.Focused.Base = lipgloss.NewStyle()
	styles.Blurred.Base = lipgloss.NewStyle()
	styles.Focused.Text = lipgloss.NewStyle()
	styles.Blurred.Text = lipgloss.NewStyle()

	base := lipgloss.NewStyle()
	styles.Focused.CursorLine = base
	styles.Focused.CursorLineNumber = base
	styles.Focused.EndOfBuffer = base
	styles.Focused.LineNumber = base
	styles.Blurred.CursorLine = base
	styles.Blurred.CursorLineNumber = base
	styles.Blurred.EndOfBuffer = base
	styles.Blurred.LineNumber = base

	ta.SetStyles(styles)
	ta.SetValue(prefill)

	return EditorModel{
		title: title,
		ta:    ta,
	}
}

// Visible returns whether the editor modal is shown.
func (m EditorModel) Visible() bool { return m.visible }

// Show makes the editor modal visible.
func (m EditorModel) Show() EditorModel {
	m.visible = true
	m.ta.Focus()

	return m
}

// Hide hides the editor modal.
func (m EditorModel) Hide() EditorModel {
	m.visible = false
	m.ta.Blur()

	return m
}

// SetSize updates the editor modal dimensions.
func (m EditorModel) SetSize(width, height int) EditorModel {
	m.width = width
	m.height = height

	boxWidth := editorBoxWidth(width)
	boxHeight := editorBoxHeight(height)
	if m.docked {
		boxWidth = width
		boxHeight = height
	}

	m.ta.SetWidth(max(10, boxWidth-6))
	m.ta.SetHeight(max(3, boxHeight-4))

	return m
}

// SetDocked updates whether the editor overlay renders as a docked prompt.
func (m EditorModel) SetDocked(docked bool) EditorModel {
	m.docked = docked
	return m.SetSize(m.width, m.height)
}

// SetTheme updates the theme used to render the editor overlay.
func (m EditorModel) SetTheme(theme *palette.Theme) EditorModel {
	m.theme = theme
	return m
}

func editorBoxWidth(width int) int {
	if width < 4 {
		return 0
	}

	return min(max(40, width*4/5), width-4)
}

func editorBoxHeight(height int) int {
	if height < 6 {
		return 0
	}

	return min(max(10, height*3/4), height-2)
}

// Width returns the editor modal width.
func (m EditorModel) Width() int { return m.width }

// Height returns the editor modal height.
func (m EditorModel) Height() int { return m.height }

// Value returns the current editor content.
func (m EditorModel) Value() string { return m.ta.Value() }

// Update handles messages for the editor modal.
func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		return m.handleKey(key)
	}

	var cmd tea.Cmd

	m.ta, cmd = m.ta.Update(msg)

	return m, cmd
}

func (m EditorModel) handleKey(msg tea.KeyPressMsg) (EditorModel, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEsc:
		m.visible = false

		return m, func() tea.Msg { return EditorResultMsg{Ok: false} }

	case 's', 'S':
		if msg.Mod&tea.ModCtrl != 0 {
			m.visible = false
			val := m.ta.Value()

			return m, func() tea.Msg { return EditorResultMsg{Value: val, Ok: true} }
		}

		var cmd tea.Cmd

		m.ta, cmd = m.ta.Update(msg)

		return m, cmd

	case tea.KeyEnter, tea.KeyKpEnter:
		var cmd tea.Cmd

		m.ta, cmd = m.ta.Update(msg)

		return m, cmd

	default:
		var cmd tea.Cmd

		m.ta, cmd = m.ta.Update(msg)

		return m, cmd
	}
}

// View renders the editor modal overlay.
func (m EditorModel) View() string {
	if !m.visible || m.width < 4 {
		return ""
	}

	theme := m.theme
	if theme == nil {
		theme = palette.DefaultTheme()
	}

	boxWidth := editorBoxWidth(m.width)
	if m.docked {
		boxWidth = m.width
	}
	if boxWidth == 0 {
		return ""
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderFocused)).
		Width(boxWidth-2).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Foreground)).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted))

	content := titleStyle.Render(m.title) + "\n" + m.ta.View() + "\n" + hintStyle.Render("Ctrl+S save · Esc cancel")
	box := borderStyle.Render(content)

	return lipgloss.NewStyle().
		MarginTop(editorMarginTop(m.docked, m.height, lipgloss.Height(box))).
		MarginLeft(editorMarginLeft(m.docked, m.width, boxWidth)).
		Render(box)
}

func editorMarginTop(docked bool, height, boxHeight int) int {
	if docked {
		return 0
	}

	return max(0, (height-boxHeight)/2)
}

func editorMarginLeft(docked bool, width, boxWidth int) int {
	if docked {
		return 0
	}

	return max(0, (width-boxWidth)/2)
}

// Draw renders the editor modal overlay into a screen buffer region.
func (m EditorModel) Draw(scr uv.Screen, area uv.Rectangle) {
	if !m.visible || area.Dx() <= 0 || area.Dy() <= 0 {
		return
	}

	uv.NewStyledString(m.View()).Draw(scr, area)
}
