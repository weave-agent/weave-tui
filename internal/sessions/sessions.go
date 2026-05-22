package sessions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tuievents "github.com/weave-agent/weave-tui/internal/events"

	tea "charm.land/bubbletea/v2"
)

const entryTypeMessage = "message"

// EntryData is the JSON payload of a message entry in a session file.
type EntryData struct {
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Tool      json.RawMessage `json:"tool,omitempty"`
}

// SessionDir returns the directory where session JSONL files are stored.
// Checks WEAVE_JSONL_DIR env var, then falls back to ~/.weave/sessions.
func SessionDir() (string, error) {
	if dir := os.Getenv("WEAVE_JSONL_DIR"); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("session dir: %w", err)
	}

	return filepath.Join(home, ".weave", "sessions"), nil
}

// ResolveDir loads the session directory from config (matching the jsonl store's
// resolution), falling back to env var and default.
func ResolveDir(cfgPath string) string {
	dir := resolveDirFromConfig(cfgPath)
	if dir != "" {
		return dir
	}

	dir, err := SessionDir()
	if err != nil {
		return ""
	}

	return dir
}

func resolveDirFromConfig(cfgPath string) string {
	if cfgPath == "" {
		return ""
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return ""
	}

	var raw map[string]any
	if json.Unmarshal(data, &raw) != nil {
		return ""
	}

	jsonl, ok := raw["jsonl"].(map[string]any)
	if !ok {
		return ""
	}

	dir, ok := jsonl["dir"].(string)
	if !ok {
		return ""
	}

	return dir
}

// header matches the first-line JSON of each JSONL session file.
type header struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	CWD       string    `json:"cwd"`
}

// List reads session headers from the session directory.
// Returns sessions sorted by most recent first.
// dirOverride, when non-empty, is used instead of the default session directory.
func List(dirOverride string) ([]tuievents.SessionEntry, error) {
	dir := dirOverride
	if dir == "" {
		var err error

		dir, err = SessionDir()
		if err != nil {
			return nil, err
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("read session dir: %w", err)
	}

	var sessions []tuievents.SessionEntry

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}

		path := filepath.Join(dir, e.Name())

		h, err := readHeader(path)
		if err != nil {
			continue
		}

		fi, err := e.Info()
		if err != nil {
			continue
		}

		sessions = append(sessions, tuievents.SessionEntry{
			ID:        h.ID,
			CWD:       h.CWD,
			CreatedAt: h.Timestamp,
			UpdatedAt: fi.ModTime(),
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

func readHeader(path string) (*header, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session header: %w", err)
	}

	defer func() { _ = f.Close() }()

	dec := json.NewDecoder(f)

	var h header

	if err := dec.Decode(&h); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	return &h, nil
}

// LoadEntries reads all message entries from a session file.
// dirOverride, when non-empty, is used instead of the default session directory.
func LoadEntries(dirOverride, sessionID string) ([]EntryData, error) {
	dir := dirOverride
	if dir == "" {
		var err error

		dir, err = SessionDir()
		if err != nil {
			return nil, err
		}
	}

	if !isValidID(sessionID) {
		return nil, fmt.Errorf("invalid session ID: %s", sessionID)
	}

	path := filepath.Join(dir, sessionID+".jsonl")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}

	lines := splitLines(data)
	if len(lines) <= 1 {
		return []EntryData{}, nil
	}

	var entries []EntryData

	for _, line := range lines[1:] { // skip header
		var raw struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		if raw.Type != entryTypeMessage {
			continue
		}

		var d EntryData
		if err := json.Unmarshal(raw.Data, &d); err != nil {
			continue
		}

		entries = append(entries, d)
	}

	return entries, nil
}

// ShortenCWD replaces the home directory prefix with ~.
func ShortenCWD(cwd string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return cwd
	}

	home = filepath.Clean(home)

	cleanCWD := filepath.Clean(cwd)
	if cleanCWD == home {
		return "~"
	}

	if strings.HasPrefix(cleanCWD, home+string(os.PathSeparator)) {
		return "~" + strings.TrimPrefix(cleanCWD, home)
	}

	return cwd
}

// ListCmd returns a tea.Cmd that reads session headers and returns SessionListResultMsg.
func ListCmd(dirOverride string) tea.Cmd {
	return func() tea.Msg {
		sessions, err := List(dirOverride)
		return tuievents.SessionListResultMsg{Sessions: sessions, Err: err}
	}
}

func isValidID(id string) bool {
	if id == "" {
		return false
	}

	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}

	return true
}

func splitLines(data []byte) []string {
	var lines []string

	for line := range bytes.SplitSeq(data, []byte{'\n'}) {
		if len(line) > 0 {
			lines = append(lines, string(line))
		}
	}

	return lines
}
