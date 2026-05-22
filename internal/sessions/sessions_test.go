package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	tuievents "github.com/weave-agent/weave-tui/internal/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WEAVE_JSONL_DIR", dir)

	sessions, err := List("")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestList_NoDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	t.Setenv("WEAVE_JSONL_DIR", dir)

	sessions, err := List("")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestList_ReadsHeaders(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WEAVE_JSONL_DIR", dir)

	now := time.Now().UTC().Truncate(time.Second)

	writeSessionFile(t, dir, "bbb11122233344455566677788899900", "/project/beta", now.Add(-time.Hour))
	writeSessionFile(t, dir, "aaa11122233344455566677788899900", "/project/alpha", now)

	sessions, err := List("")
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	assert.Equal(t, "aaa11122233344455566677788899900", sessions[0].ID)
	assert.Equal(t, "/project/alpha", sessions[0].CWD)
	assert.Equal(t, now, sessions[0].CreatedAt)

	assert.Equal(t, "bbb11122233344455566677788899900", sessions[1].ID)
	assert.Equal(t, "/project/beta", sessions[1].CWD)
}

func TestList_SkipsNonJSONL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WEAVE_JSONL_DIR", dir)

	writeSessionFile(t, dir, "aaa11122233344455566677788899900", "/project", time.Now())

	err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "bad.jsonl"), []byte("not json\n"), 0o644)
	require.NoError(t, err)

	sessions, err := List("")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "aaa11122233344455566677788899900", sessions[0].ID)
}

func TestLoadEntries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WEAVE_JSONL_DIR", dir)

	sessionID := "aaa11122233344455566677788899900"
	now := time.Now().UTC()

	h := header{Type: "session", ID: sessionID, Timestamp: now, CWD: "/test"}
	headerJSON, _ := json.Marshal(h)

	entry1 := map[string]any{
		"type": "message",
		"data": map[string]any{"role": "user", "content": "hello"},
	}
	entry2 := map[string]any{
		"type": "message",
		"data": map[string]any{"role": "assistant", "content": "hi there"},
	}
	entry3 := map[string]any{
		"type": "summary",
		"data": map[string]any{"removed_count": 5},
	}

	e1JSON, _ := json.Marshal(entry1)
	e2JSON, _ := json.Marshal(entry2)
	e3JSON, _ := json.Marshal(entry3)

	content := string(headerJSON) + "\n" + string(e1JSON) + "\n" + string(e2JSON) + "\n" + string(e3JSON) + "\n"
	err := os.WriteFile(filepath.Join(dir, sessionID+".jsonl"), []byte(content), 0o644)
	require.NoError(t, err)

	entries, err := LoadEntries("", sessionID)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "user", entries[0].Role)
	assert.Equal(t, "hello", entries[0].Content)
	assert.Equal(t, "assistant", entries[1].Role)
	assert.Equal(t, "hi there", entries[1].Content)
}

func TestLoadEntries_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WEAVE_JSONL_DIR", dir)

	_, err := LoadEntries("", "nonexistent0000000000000000000")
	assert.Error(t, err)
}

func TestLoadEntries_EmptySession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WEAVE_JSONL_DIR", dir)

	sessionID := "aaa11122233344455566677788899900"
	h := header{Type: "session", ID: sessionID, Timestamp: time.Now().UTC(), CWD: "/test"}
	headerJSON, _ := json.Marshal(h)

	err := os.WriteFile(filepath.Join(dir, sessionID+".jsonl"), []byte(string(headerJSON)+"\n"), 0o644)
	require.NoError(t, err)

	entries, err := LoadEntries("", sessionID)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLoadEntries_PathTraversalRejected(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{"parent traversal", "../etc/passwd"},
		{"forward slash", "foo/bar"},
		{"backslash", "foo\\bar"},
		{"mixed traversal", "..\\..\\secrets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadEntries("", tt.sessionID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid session ID")
		})
	}
}

func TestShortenCWD(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name string
		cwd  string
		want string
	}{
		{"home prefix", home + "/projects/myapp", "~/projects/myapp"},
		{"no home prefix", "/opt/projects", "/opt/projects"},
		{"exact home", home, "~"},
		{"home string prefix only", home + "-other/projects", home + "-other/projects"},
		{"home embedded later", "/tmp" + home + "/projects", "/tmp" + home + "/projects"},
		{"home sibling path", filepath.Dir(home) + "/" + filepath.Base(home) + "-other", filepath.Dir(home) + "/" + filepath.Base(home) + "-other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShortenCWD(tt.cwd))
		})
	}
}

func TestListCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WEAVE_JSONL_DIR", dir)

	writeSessionFile(t, dir, "aaa11122233344455566677788899900", "/test", time.Now())

	cmd := ListCmd("")
	msg := cmd()

	result, ok := msg.(tuievents.SessionListResultMsg)
	require.True(t, ok)
	require.NoError(t, result.Err)
	require.Len(t, result.Sessions, 1)
	assert.Equal(t, "aaa11122233344455566677788899900", result.Sessions[0].ID)
}

func TestSessionDir_EnvOverride(t *testing.T) {
	custom := "/tmp/weave-test-sessions"
	t.Setenv("WEAVE_JSONL_DIR", custom)

	dir, err := SessionDir()
	require.NoError(t, err)
	assert.Equal(t, custom, dir)
}

func TestResolveDir_ConfigOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	data := fmt.Appendf(nil, `{"jsonl":{"dir":%q}}`, dir)
	require.NoError(t, os.WriteFile(configPath, data, 0o644))

	assert.Equal(t, dir, ResolveDir(configPath))
}

func writeSessionFile(t *testing.T, dir, sessionID, cwd string, ts time.Time) {
	t.Helper()

	h := header{Type: "session", ID: sessionID, Timestamp: ts, CWD: cwd}
	headerJSON, err := json.Marshal(h)
	require.NoError(t, err)

	path := filepath.Join(dir, sessionID+".jsonl")
	line := fmt.Appendf(nil, "%s\n", headerJSON)
	err = os.WriteFile(path, line, 0o644)
	require.NoError(t, err)
}
