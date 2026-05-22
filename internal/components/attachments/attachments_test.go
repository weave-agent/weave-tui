package attachments

import (
	"regexp"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/stretchr/testify/assert"
)

// stripANSI removes ANSI escape sequences from a string.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestNew(t *testing.T) {
	m := New()
	assert.Empty(t, m.Items())
	assert.Equal(t, 0, m.Height())
}

func TestAdd(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "test.go", Content: "hello", Lines: 1})
	assert.Len(t, m.Items(), 1)
	assert.Equal(t, "test.go", m.Items()[0].Path)
	assert.Equal(t, 1, m.Height())
}

func TestAddMultiple(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "a.go", Lines: 1})
	m = m.Add(Attachment{Path: "b.go", Lines: 5})
	assert.Len(t, m.Items(), 2)
	assert.Equal(t, 2, m.Height())
}

func TestAddPaste(t *testing.T) {
	content := "line1\nline2\nline3"
	m := New()
	m = m.AddPaste(content)
	assert.Len(t, m.Items(), 1)
	assert.Equal(t, "paste-1.txt", m.Items()[0].Path)
	assert.Equal(t, 3, m.Items()[0].Lines)
}

func TestAddPaste_UniqueNames(t *testing.T) {
	m := New()
	m = m.AddPaste("content1\nline2")
	m = m.AddPaste("content2\nline2")
	assert.Len(t, m.Items(), 2)
	assert.Equal(t, "paste-1.txt", m.Items()[0].Path)
	assert.Equal(t, "paste-2.txt", m.Items()[1].Path)
}

func TestRemove(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "a.go", Lines: 1})
	m = m.Add(Attachment{Path: "b.go", Lines: 2})
	m = m.Add(Attachment{Path: "c.go", Lines: 3})

	m = m.Remove(1) // remove b.go
	assert.Len(t, m.Items(), 2)
	assert.Equal(t, "a.go", m.Items()[0].Path)
	assert.Equal(t, "c.go", m.Items()[1].Path)
}

func TestRemove_OutOfBounds(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "a.go", Lines: 1})
	m = m.Remove(-1)
	m = m.Remove(5)
	assert.Len(t, m.Items(), 1)
}

func TestRemove_LastItem(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "a.go", Lines: 1})
	m = m.Remove(0)
	assert.Empty(t, m.Items())
}

func TestClear(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "a.go", Lines: 1})
	m = m.Add(Attachment{Path: "b.go", Lines: 2})
	m = m.Clear()
	assert.Empty(t, m.Items())
}

func TestIsPastedContent_Newlines(t *testing.T) {
	text := strings.Repeat("line\n", 12)
	assert.True(t, IsPastedContent(text))

	short := strings.Repeat("line\n", 5)
	assert.False(t, IsPastedContent(short))
}

func TestIsPastedContent_Length(t *testing.T) {
	longText := strings.Repeat("x", 1001)
	assert.True(t, IsPastedContent(longText))

	shortText := strings.Repeat("x", 999)
	assert.False(t, IsPastedContent(shortText))
}

func TestIsPastedContent_Empty(t *testing.T) {
	assert.False(t, IsPastedContent(""))
}

func TestRenderPrompt_NoAttachments(t *testing.T) {
	m := New()
	assert.Equal(t, "hello", m.RenderPrompt("hello"))
}

func TestRenderPrompt_WithAttachments(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "/path/to/test.go", Content: "package main", Lines: 1})
	result := m.RenderPrompt("fix this")
	assert.Contains(t, result, "fix this")
	assert.Contains(t, result, "File: test.go")
	assert.Contains(t, result, "package main")
}

func TestRenderPrompt_MultipleAttachments(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "a.go", Content: "aaa", Lines: 1})
	m = m.Add(Attachment{Path: "b.go", Content: "bbb", Lines: 1})
	result := m.RenderPrompt("check these")
	assert.Contains(t, result, "File: a.go")
	assert.Contains(t, result, "File: b.go")
}

func TestRenderPrompt_EmptyEditorText(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "a.go", Content: "aaa", Lines: 1})
	result := m.RenderPrompt("")
	assert.Contains(t, result, "File: a.go")
	assert.NotContains(t, result, "\n\nFile:")
}

func TestDraw_NoAttachments(t *testing.T) {
	m := New()
	scr := uv.NewScreenBuffer(80, 5)
	m.Draw(scr, uv.Rect(0, 0, 80, 5))
	// Should not panic, renders nothing
}

func TestDraw_ZeroArea(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "a.go", Lines: 1})
	scr := uv.NewScreenBuffer(80, 5)
	m.Draw(scr, uv.Rect(0, 0, 0, 0))
	m.Draw(scr, uv.Rect(0, 0, 80, 0))
	// Should not panic
}

func TestDraw_RendersAttachment(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "test.go", Lines: 42})
	scr := uv.NewScreenBuffer(80, 5)
	m.Draw(scr, uv.Rect(0, 0, 80, 1))
	rendered := scr.Render()
	assert.Contains(t, rendered, "test.go")
	assert.Contains(t, rendered, "42 lines")
}

func TestDraw_PillShapeRendering(t *testing.T) {
	m := New()
	m = m.Add(Attachment{Path: "test.go", Lines: 42})

	scr := uv.NewScreenBuffer(80, 5)
	m.Draw(scr, uv.Rect(0, 0, 80, 1))
	rendered := stripANSI(scr.Render())

	// Pill should contain attachment info without brackets
	assert.Contains(t, rendered, "test.go")
	assert.Contains(t, rendered, "42 lines")
	assert.NotContains(t, rendered, "[")
	assert.NotContains(t, rendered, "]")
}
