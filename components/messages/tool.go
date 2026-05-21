package messages

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/weave-agent/weave-tui/palette"
	"github.com/weave-agent/weave/sdk"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

const maxCollapsedLines = 20

// ToolState represents the execution state of a tool call.
type ToolState int

const (
	ToolPending ToolState = iota
	ToolRunning
	ToolSuccess
	ToolError
	ToolInterrupted
)

// ToolPanel renders a tool call with its output in a panel.
type ToolPanel struct {
	toolID               string
	toolName             string
	args                 string
	output               string
	progress             string
	state                ToolState
	expanded             bool
	diffRenderer         *DiffRenderer
	customRenderer       sdk.ToolRenderer
	flashUntil           time.Time
	needsPostFlashRender bool
	spinnerFrame         int
}

// NewToolPanel creates a new tool panel in pending state.
func NewToolPanel(toolID, toolName, args string) *ToolPanel {
	return &ToolPanel{
		toolID:   toolID,
		toolName: toolName,
		args:     strings.TrimSpace(args),
		state:    ToolPending,
		expanded: false,
	}
}

// ToolID returns the tool call ID.
func (p *ToolPanel) ToolID() string {
	return p.toolID
}

// ItemID implements ChatItemIdentity for in-place updates.
func (p *ToolPanel) ItemID() string {
	return p.toolID
}

// State returns the current tool state.
func (p *ToolPanel) State() ToolState {
	return p.state
}

// Expanded returns whether the panel is expanded.
func (p *ToolPanel) Expanded() bool {
	return p.expanded
}

// SetResult updates the panel with a tool result.
func (p *ToolPanel) SetResult(output string, isError bool) {
	p.output = output
	p.progress = ""
	if isError {
		p.state = ToolError
	} else {
		p.state = ToolSuccess
	}

	p.flashUntil = time.Now().Add(800 * time.Millisecond)
	p.needsPostFlashRender = true
}

// SetRunning marks the tool as actively executing.
func (p *ToolPanel) SetRunning() {
	p.state = ToolRunning
}

// SetProgress updates the panel with partial output from a running tool.
func (p *ToolPanel) SetProgress(content string) {
	p.progress = content
}

// Progress returns the accumulated partial output from the tool.
func (p *ToolPanel) Progress() string {
	return p.progress
}

// SetInterrupted marks the tool as interrupted by the user.
func (p *ToolPanel) SetInterrupted() {
	p.state = ToolInterrupted
	p.flashUntil = time.Now().Add(800 * time.Millisecond)
	p.needsPostFlashRender = true
}

// AdvanceSpinner cycles the spinner animation frame for running tools.
func (p *ToolPanel) AdvanceSpinner() {
	p.spinnerFrame++
}

// NeedsRender returns true while the flash animation is active, and for one
// additional render after it expires so the cache captures the settled color.
// Also returns true for running tools so spinner frames advance.
func (p *ToolPanel) NeedsRender() bool {
	if p.state == ToolRunning {
		return true
	}

	if time.Now().Before(p.flashUntil) {
		return true
	}

	return p.needsPostFlashRender
}

// ToggleExpanded flips the expand/collapse state.
func (p *ToolPanel) ToggleExpanded() {
	p.expanded = !p.expanded
}

// SetDiffRenderer sets the diff renderer for auto-detecting diff output.
func (p *ToolPanel) SetDiffRenderer(r *DiffRenderer) {
	p.diffRenderer = r
}

// SetRenderer sets a custom tool renderer registered via sdk.UI.
func (p *ToolPanel) SetRenderer(r sdk.ToolRenderer) {
	p.customRenderer = r
}

// View renders the tool panel.
func (p *ToolPanel) View(width int) string {
	if width <= 0 {
		width = 80
	}

	// Once the flash has expired, clear the flag so the next render captures
	// the settled border color in the cache.
	if p.needsPostFlashRender && time.Now().After(p.flashUntil) {
		p.needsPostFlashRender = false
	}

	theme := palette.DefaultTheme()
	borderColor := borderColorForState(p.state, p.flashUntil, theme)

	lineStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(lipgloss.Color(borderColor)).
		Width(width)

	header := p.renderHeader()
	body := p.renderBody(width)

	var content strings.Builder
	content.WriteString(header)

	if body != "" {
		content.WriteString("\n\n")
		content.WriteString(body)
	}

	return lineStyle.Render(content.String())
}

// Draw renders the tool panel into a screen buffer region.
func (p *ToolPanel) Draw(scr uv.Screen, area uv.Rectangle) {
	drawView(scr, area, p.View(area.Dx()))
}

func (p *ToolPanel) renderHeader() string {
	stateLabel := stateLabelForState(p.state)
	if p.state == ToolRunning {
		stateLabel = spinnerFrameChar(p.spinnerFrame)
	}

	if p.args != "" {
		formatted := formatArgs(p.args)
		if formatted != "" {
			return fmt.Sprintf(" %s %s(%s)", stateLabel, p.toolName, formatted)
		}
	}

	return fmt.Sprintf(" %s %s", stateLabel, p.toolName)
}

func (p *ToolPanel) renderBody(width int) string {
	theme := palette.DefaultTheme()

	// Show live progress content for running tools
	if p.state == ToolRunning && p.progress != "" {
		lines := strings.Split(p.progress, "\n")
		if !p.expanded && len(lines) > maxCollapsedLines {
			visible := lines[:maxCollapsedLines]
			hidden := len(lines) - maxCollapsedLines
			body := strings.Join(visible, "\n")
			dim := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))

			return dim.Render(body) + fmt.Sprintf("\n... %d more lines (collapsed)", hidden)
		}
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))

		return dim.Render(p.progress)
	}

	if p.output == "" {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
		switch p.state {
		case ToolPending, ToolRunning:
			return dim.Render("running...")
		case ToolInterrupted:
			if p.progress != "" {
				return dim.Render(p.progress + "\n(interrupted)")
			}

			return dim.Render("(interrupted)")
		default:
			return dim.Render("(no output)")
		}
	}

	// Use custom renderer if registered.
	if p.customRenderer != nil {
		return p.customRenderer.Render(p.output, width)
	}

	// Auto-detect diff content and use diff renderer.
	if p.diffRenderer != nil && IsDiffContent(p.output) {
		return p.diffRenderer.Render(p.output, width)
	}

	lines := strings.Split(p.output, "\n")

	if !p.expanded && len(lines) > maxCollapsedLines {
		visible := lines[:maxCollapsedLines]
		hidden := len(lines) - maxCollapsedLines

		body := strings.Join(visible, "\n")
		if p.state == ToolError {
			body = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Error)).Render(body)
		}

		return body + fmt.Sprintf("\n... %d more lines (collapsed)", hidden)
	}

	body := strings.Join(lines, "\n")
	if p.state == ToolError {
		body = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Error)).Render(body)
	}

	return body
}

func borderColorForState(state ToolState, flashUntil time.Time, theme *palette.Theme) string {
	if time.Now().Before(flashUntil) {
		switch state {
		case ToolPending, ToolRunning:
			return theme.AccentDim
		case ToolSuccess:
			return theme.Success
		case ToolError:
			return theme.Error
		case ToolInterrupted:
			return theme.Muted
		}
	}

	switch state {
	case ToolPending, ToolRunning:
		return theme.AccentDim
	case ToolSuccess:
		return theme.Border
	case ToolError:
		return theme.Error
	case ToolInterrupted:
		return theme.Muted
	default:
		return theme.Border
	}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func stateLabelForState(state ToolState) string {
	switch state {
	case ToolPending:
		return "⏳"
	case ToolRunning:
		return "⠋" // default spinner frame; use spinnerFrameChar for animation
	case ToolSuccess:
		return "✓"
	case ToolError:
		return "✗"
	case ToolInterrupted:
		return "⏹"
	default:
		return "?"
	}
}

func spinnerFrameChar(frame int) string {
	return spinnerFrames[frame%len(spinnerFrames)]
}

func truncateArgs(args string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if maxLen < 3 {
		return strings.Repeat(".", maxLen)
	}

	args = strings.TrimSpace(args)
	// Try to keep it on one line.
	args = strings.ReplaceAll(args, "\n", " ")
	if utf8.RuneCountInString(args) > maxLen {
		runes := []rune(args)
		return string(runes[:maxLen-3]) + "..."
	}

	return args
}

// formatArgs converts a JSON object string into compact key=value pairs.
func formatArgs(argsJSON string) string {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" || argsJSON == "{}" {
		return ""
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &m); err != nil {
		return argsJSON
	}

	if len(m) == 0 {
		return ""
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, formatArgValue(m[k])))
	}

	return strings.Join(parts, ", ")
}

func formatArgValue(value any) string {
	switch val := value.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}

		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	default:
		encoded, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}

		return string(encoded)
	}
}
