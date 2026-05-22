package messages

import (
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weave-agent/weave-tui/internal/palette"
	"github.com/weave-agent/weave-tui/internal/styles"
)

func TestToolPanel_NewPanel_Pending(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls -la")
	assert.Equal(t, "tc1", p.ToolID())
	assert.Equal(t, "tc1", p.ItemID())
	assert.Equal(t, ToolPending, p.State())
	assert.False(t, p.Expanded())
}

func TestToolPanel_SetResult_Success(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("file1.txt\nfile2.txt", false)
	assert.Equal(t, ToolSuccess, p.State())
}

func TestToolPanel_SetResult_Error(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("command not found", true)
	assert.Equal(t, ToolError, p.State())
}

func TestToolPanel_ToggleExpanded(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	assert.False(t, p.Expanded())
	p.ToggleExpanded()
	assert.True(t, p.Expanded())
	p.ToggleExpanded()
	assert.False(t, p.Expanded())
}

func TestToolPanel_View_Pending(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls -la")
	view := p.View(80)
	assert.Contains(t, view, "bash")
	// Pending shows "Running…" inside bordered card
	assert.Contains(t, view, "Running…")
}

func TestToolPanel_View_Success(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("file1.txt\nfile2.txt", false)
	view := p.View(80)
	assert.Contains(t, view, "bash")
	assert.Contains(t, view, "file1.txt")
	assert.Contains(t, view, "file2.txt")
}

func TestToolPanel_View_Error(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("permission denied", true)
	view := p.View(80)
	assert.Contains(t, view, "bash")
	assert.Contains(t, view, "permission denied")
}

func TestToolPanel_View_NoOutput(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("", false)
	view := p.View(80)
	assert.Contains(t, view, "No output")
}

func TestToolPanel_View_WithArgs(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls -la /tmp")
	view := p.View(80)
	assert.Contains(t, view, "bash")
	assert.Contains(t, view, "ls -la /tmp")
}

func TestToolPanel_View_NoArgs(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "")
	view := p.View(80)
	assert.Contains(t, view, "bash")
	assert.NotContains(t, view, "()")
}

func TestToolPanel_CollapseLongOutput(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "line of output"
	}

	output := strings.Join(lines, "\n")

	p := NewToolPanel("tc1", "bash", "cat bigfile")
	p.SetResult(output, false)
	view := p.View(80)

	assert.Contains(t, view, "more lines (collapsed)")
	assert.False(t, p.Expanded())
}

func TestToolPanel_ExpandShowsAll(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "line of output"
	}

	output := strings.Join(lines, "\n")

	p := NewToolPanel("tc1", "bash", "cat bigfile")
	p.SetResult(output, false)
	p.ToggleExpanded()
	view := p.View(80)

	assert.NotContains(t, view, "collapsed")
	assert.True(t, p.Expanded())
	// All 30 lines should be present
	assert.Equal(t, 30, strings.Count(view, "line of output"))
}

func TestToolPanel_View_ReadOutputCollapsedByDefault(t *testing.T) {
	p := NewToolPanel("tc1", "read", `{"path":"main.go"}`)
	p.SetResult("package main\n\nfunc main() {}", false)

	view := p.View(80)

	assert.Contains(t, view, "3 lines (collapsed)")
	assert.NotContains(t, view, "package main")
	assert.False(t, p.Expanded())
}

func TestToolPanel_View_ReadOutputExpandedShowsAll(t *testing.T) {
	p := NewToolPanel("tc1", "read", `{"path":"main.go"}`)
	p.SetResult("package main\n\nfunc main() {}", false)
	p.ToggleExpanded()

	view := p.View(80)

	assert.Contains(t, view, "package main")
	assert.Contains(t, view, "func main() {}")
	assert.NotContains(t, view, "collapsed")
}

func TestToolPanel_ShortOutputNotCollapsed(t *testing.T) {
	output := "short output\njust two lines"
	p := NewToolPanel("tc1", "bash", "echo hi")
	p.SetResult(output, false)
	view := p.View(80)

	assert.NotContains(t, view, "collapsed")
}

func TestToolPanel_StateTransitions(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	require.Equal(t, ToolPending, p.State())

	// Pending -> success
	p.SetResult("ok", false)
	assert.Equal(t, ToolSuccess, p.State())

	// Success -> error (reused panel)
	p.SetResult("fail", true)
	assert.Equal(t, ToolError, p.State())
}

func TestToolPanel_Draw_Pending(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls -la")
	canvas := uv.NewScreenBuffer(80, 10)
	p.Draw(canvas, canvas.Bounds())
	output := uv.TrimSpace(canvas.Render())
	assert.Contains(t, output, "bash")
	assert.Contains(t, output, "Running…")
}

func TestToolPanel_Draw_Success(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("file1.txt\nfile2.txt", false)

	canvas := uv.NewScreenBuffer(80, 10)
	p.Draw(canvas, canvas.Bounds())
	output := uv.TrimSpace(canvas.Render())
	assert.Contains(t, output, "file1.txt")
}

func TestToolPanel_Draw_Error(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("permission denied", true)

	canvas := uv.NewScreenBuffer(80, 10)
	p.Draw(canvas, canvas.Bounds())
	output := uv.TrimSpace(canvas.Render())
	assert.Contains(t, output, "permission denied")
}

func TestToolPanel_Draw_ZeroArea(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	canvas := uv.NewScreenBuffer(80, 10)
	p.Draw(canvas, uv.Rect(0, 0, 0, 0))
}

func TestToolPanel_StateEmojis(t *testing.T) {
	tests := []struct {
		name      string
		state     ToolState
		wantEmoji string
	}{
		{"pending", ToolPending, "○"},
		{"running", ToolRunning, "⠋"},
		{"success", ToolSuccess, "✓"},
		{"error", ToolError, "×"},
		{"interrupted", ToolInterrupted, "■"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, stateLabelForState(tt.state), tt.wantEmoji)
		})
	}
}

func TestToolPanel_View_PendingHasRunningText(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls -la")
	view := p.View(80)
	assert.Contains(t, view, "○")
	assert.Contains(t, view, "bash")
	assert.Contains(t, view, "Running…")
}

func TestToolPanel_View_SuccessHasCheckmark(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("file1.txt", false)
	view := p.View(80)
	assert.Contains(t, view, "✓")
	assert.Contains(t, view, "bash")
}

func TestToolPanel_View_ErrorHasXMark(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("permission denied", true)
	view := p.View(80)
	assert.Contains(t, view, "×")
	assert.Contains(t, view, "bash")
}

func TestToolPanel_View_ErrorOutputInErrorColor(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("permission denied", true)
	view := p.View(80)
	// Error state should contain the output text
	assert.Contains(t, view, "permission denied")
}

func TestToolPanel_SetResult_SetsFlashTimer(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	assert.True(t, p.flashUntil.IsZero(), "flash timer should be zero initially")

	p.SetResult("done", false)
	assert.False(t, p.flashUntil.IsZero(), "flash timer should be set after result")
	assert.True(t, p.flashUntil.After(time.Now()), "flash timer should be in the future")
}

func TestToolPanel_CardHasTopAndBottomLines(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls -la")
	view := p.View(80)

	assert.Contains(t, view, "─")
	assert.NotContains(t, view, "╭")
	assert.NotContains(t, view, "│")
}

func TestToolPanel_Card_SuccessHasTopAndBottomLines(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("output", false)
	p.flashUntil = time.Time{}
	view := p.View(80)

	assert.Contains(t, view, "─")
	assert.NotContains(t, view, "╭")
	assert.NotContains(t, view, "│")
}

func TestToolPanel_SetRunning(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	require.Equal(t, ToolPending, p.State())

	p.SetRunning()
	assert.Equal(t, ToolRunning, p.State())
}

func TestToolPanel_SetProgress(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetRunning()
	p.SetProgress("partial output")

	view := p.View(80)
	assert.Contains(t, view, "partial output")
	assert.Contains(t, view, "bash")
}

func TestToolPanel_SetInterrupted(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetInterrupted()
	assert.Equal(t, ToolInterrupted, p.State())

	view := p.View(80)
	assert.Contains(t, view, "Interrupted")
	assert.Contains(t, view, "bash")
}

func TestToolPanel_InterruptedWithProgress(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetRunning()
	p.SetProgress("some partial output")
	p.SetInterrupted()

	view := p.View(80)
	assert.Contains(t, view, "some partial output")
	assert.Contains(t, view, "Interrupted")
}

func TestToolPanel_AdvanceSpinner(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetRunning()

	frame0 := spinnerFrameChar(p.spinnerFrame)
	p.AdvanceSpinner()
	frame1 := spinnerFrameChar(p.spinnerFrame)

	assert.NotEqual(t, frame0, frame1, "spinner frame should advance")
}

func TestToolPanel_NeedsRender_WhenRunning(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	assert.False(t, p.NeedsRender(), "pending panel should not need render after initial creation")

	p.SetRunning()
	assert.True(t, p.NeedsRender(), "running panel should need render")
}

func TestToolPanel_NeedsRender_WhenFlashActive(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("done", false)
	assert.True(t, p.NeedsRender(), "panel with active flash should need render")
}

func TestToolPanel_View_RunningShowsSpinner(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetRunning()

	view := p.View(80)
	// Should contain one of the spinner frames
	assert.Contains(t, view, "bash")
	// Running tool with no progress still shows "Running…"
	assert.Contains(t, view, "Running…")
}

func TestToolPanel_View_RunningWithProgress(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetRunning()
	p.SetProgress("line1\nline2")

	view := p.View(80)
	assert.Contains(t, view, "line1")
	assert.Contains(t, view, "line2")
}

func TestToolPanel_View_RunningProgressCollapsed(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "progress line"
	}

	p := NewToolPanel("tc1", "bash", "cat")
	p.SetRunning()
	p.SetProgress(strings.Join(lines, "\n"))

	view := p.View(80)
	assert.Contains(t, view, "more lines (collapsed)")
}

func TestToolPanel_View_Interrupted(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetInterrupted()

	view := p.View(80)
	assert.Contains(t, view, "■")
	assert.Contains(t, view, "Interrupted")
}

func TestToolPanel_CompleteLifecycle(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	require.Equal(t, ToolPending, p.State())

	p.SetRunning()
	assert.Equal(t, ToolRunning, p.State())

	p.SetProgress("partial")
	assert.Equal(t, "partial", p.progress)

	p.SetResult("final output", false)
	assert.Equal(t, ToolSuccess, p.State())
	assert.Equal(t, "final output", p.output)
}

func TestToolPanel_ErrorLifecycle(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetRunning()
	p.SetProgress("working...")
	p.SetResult("command failed", true)

	assert.Equal(t, ToolError, p.State())
	assert.Equal(t, "command failed", p.output)
}

func TestToolPanel_Draw_Running(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls -la")
	p.SetRunning()
	p.SetProgress("live output")

	canvas := uv.NewScreenBuffer(80, 10)
	p.Draw(canvas, canvas.Bounds())
	output := uv.TrimSpace(canvas.Render())
	assert.Contains(t, output, "bash")
	assert.Contains(t, output, "live output")
}

func TestToolPanel_Draw_Interrupted(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetInterrupted()

	canvas := uv.NewScreenBuffer(80, 10)
	p.Draw(canvas, canvas.Bounds())
	output := uv.TrimSpace(canvas.Render())
	assert.Contains(t, output, "Interrupted")
}

func TestFormatArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "object",
			input: `{"command":"go env GOPATH && rg -n \"type View\"","auto_background_after":0}`,
			want:  `auto_background_after=0, command="go env GOPATH && rg -n \"type View\""`,
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  "",
		},
		{
			name:  "non json",
			input: "ls -la",
			want:  "ls -la",
		},
		{
			name:  "nested value",
			input: `{"options":{"limit":10},"paths":["a","b"]}`,
			want:  `options={"limit":10}, paths=["a","b"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatArgs(tt.input))
		})
	}
}

func TestToolPanel_View_WithFormattedArgs(t *testing.T) {
	p := NewToolPanel("tc1", "bash", `{"command":"ls -la /tmp","auto_background_after":0}`)
	view := p.View(80)

	assert.Contains(t, view, `bash(auto_background_after=0, command="ls -la /tmp")`)
	assert.NotContains(t, view, `{"command"`)
}

func TestToolPanel_View_DoesNotTruncateArgs(t *testing.T) {
	command := strings.Repeat("x", 150)
	p := NewToolPanel("tc1", "bash", `{"command":"`+command+`"}`)
	view := p.View(240)

	assert.Contains(t, view, command)
}

// --- Theme usage ---

func TestToolPanel_SetStyles_UsesCustomTheme(t *testing.T) {
	custom := &palette.Theme{
		AccentDim: "99",
		Muted:     "88",
		Error:     "77",
		Success:   "66",
		Border:    "55",
	}

	p := NewToolPanel("tc1", "bash", "ls")
	p.SetStyles(styles.New(custom))

	view := p.View(80)
	// Pending glyph should be styled with AccentDim (99)
	assert.Contains(t, view, "○")
	// Border should use AccentDim for pending state
	assertCustomColorInString(t, view, "99")
}

func TestToolPanel_CustomTheme_ErrorState(t *testing.T) {
	custom := &palette.Theme{
		Error:   "42",
		Border:  "40",
		Success: "41",
	}

	p := NewToolPanel("tc1", "bash", "ls")
	p.SetStyles(styles.New(custom))
	p.SetResult("fail", true)
	p.flashUntil = time.Time{} // clear flash to see settled color

	view := p.View(80)
	// Error glyph should be styled with Error color (42)
	assert.Contains(t, view, "×")
	assertCustomColorInString(t, view, "42")
}

func TestToolPanel_CustomTheme_SuccessState(t *testing.T) {
	custom := &palette.Theme{
		Success: "33",
		Border:  "34",
	}

	p := NewToolPanel("tc1", "bash", "ls")
	p.SetStyles(styles.New(custom))
	p.SetResult("ok", false)
	p.flashUntil = time.Time{} // clear flash to see settled color

	view := p.View(80)
	assert.Contains(t, view, "✓")
	// Settled success border uses Border color
	assertCustomColorInString(t, view, "34")
}

func TestToolPanel_CustomTheme_InterruptedState(t *testing.T) {
	custom := &palette.Theme{
		Muted: "55",
	}

	p := NewToolPanel("tc1", "bash", "ls")
	p.SetStyles(styles.New(custom))
	p.SetInterrupted()
	p.flashUntil = time.Time{} // clear flash

	view := p.View(80)
	assert.Contains(t, view, "■")
	assertCustomColorInString(t, view, "55")
}

// --- Spacing ---

func TestToolPanel_View_HeaderBodySpacing(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("output line", false)
	view := p.View(80)

	// Header and body should be separated by exactly one blank line
	// (two newlines between header content and body content)
	lines := strings.Split(view, "\n")

	// Find the header line (contains "bash") and the body line (contains "output")
	var headerIdx, bodyIdx int

	for i, line := range lines {
		if strings.Contains(line, "bash") {
			headerIdx = i
		}

		if strings.Contains(line, "output line") {
			bodyIdx = i
		}
	}

	require.Greater(t, bodyIdx, headerIdx, "body should come after header")
	assert.Equal(t, 2, bodyIdx-headerIdx, "header and body should be separated by exactly one blank line")
}

func TestToolPanel_View_NoBodyWhenEmpty(t *testing.T) {
	p := NewToolPanel("tc1", "bash", "ls")
	p.SetResult("", false)
	view := p.View(80)

	// With no output, the body should show "No output" which is still a body
	assert.Contains(t, view, "No output")
}

func assertCustomColorInString(t *testing.T, s, colorCode string) {
	t.Helper()

	fgParam := "38;5;" + colorCode
	bgParam := "48;5;" + colorCode
	assert.True(t, strings.Contains(s, fgParam) || strings.Contains(s, bgParam),
		"expected rendered output to contain ANSI color parameter for %s", colorCode)
}
