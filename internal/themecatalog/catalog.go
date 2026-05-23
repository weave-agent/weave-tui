package themecatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/weave-agent/weave-tui/internal/palette"
)

const (
	defaultThemeName = "default"
	jsonExt          = ".json"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Source identifies where a catalog entry came from.
type Source string

const (
	SourceBuiltin Source = "builtin"
	SourceUser    Source = "user"
)

// Entry is a named theme plus catalog metadata.
type Entry struct {
	Name   string
	Theme  *palette.Theme
	Source Source
	Path   string
}

// Catalog contains built-in and user themes, keyed by canonical name.
type Catalog struct {
	entries map[string]Entry
}

// Load builds a catalog from built-ins and all regular JSON files in themesDir.
// User files are loaded after built-ins, so they override built-in themes with
// the same name.
func Load(themesDir string) (*Catalog, error) {
	c := &Catalog{entries: make(map[string]Entry)}
	for name, theme := range BuiltinThemes() {
		c.entries[name] = Entry{Name: name, Theme: cloneTheme(theme), Source: SourceBuiltin}
	}

	if themesDir == "" {
		return c, nil
	}

	infos, err := os.ReadDir(themesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return nil, fmt.Errorf("read theme directory: %w", err)
	}

	for _, info := range infos {
		if info.IsDir() || strings.ToLower(filepath.Ext(info.Name())) != jsonExt {
			continue
		}

		fileInfo, err := info.Info()
		if err != nil {
			return nil, fmt.Errorf("stat theme file %q: %w", info.Name(), err)
		}
		if !fileInfo.Mode().IsRegular() {
			continue
		}

		path := filepath.Join(themesDir, info.Name())
		entry, err := loadUserTheme(path)
		if err != nil {
			return nil, err
		}
		c.entries[entry.Name] = entry
	}

	return c, nil
}

// BuiltinThemes returns the trusted built-in themes.
func BuiltinThemes() map[string]*palette.Theme {
	return map[string]*palette.Theme{
		defaultThemeName: palette.DefaultTheme(),
	}
}

// Theme returns a copy of the named theme.
func (c *Catalog) Theme(name string) (*palette.Theme, error) {
	if c == nil {
		return nil, errors.New("theme catalog is nil")
	}

	entry, ok := c.entries[name]
	if !ok {
		return nil, fmt.Errorf("unknown theme: %s", name)
	}

	return cloneTheme(entry.Theme), nil
}

// Entry returns a copy of the named catalog entry.
func (c *Catalog) Entry(name string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}

	entry, ok := c.entries[name]
	if !ok {
		return Entry{}, false
	}
	entry.Theme = cloneTheme(entry.Theme)
	return entry, true
}

// List returns catalog entries sorted by theme name.
func (c *Catalog) List() []Entry {
	if c == nil {
		return nil
	}

	names := make([]string, 0, len(c.entries))
	for name := range c.entries {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]Entry, 0, len(names))
	for _, name := range names {
		entry := c.entries[name]
		entry.Theme = cloneTheme(entry.Theme)
		entries = append(entries, entry)
	}

	return entries
}

func loadUserTheme(path string) (Entry, error) {
	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	if err := ValidateName(name); err != nil {
		return Entry{}, fmt.Errorf("invalid theme filename %q: %w", filename, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, fmt.Errorf("read theme file %q: %w", path, err)
	}

	var file userThemeFile
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return Entry{}, fmt.Errorf("parse theme file %q: %w", path, err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return Entry{}, fmt.Errorf("parse theme file %q: unexpected trailing JSON", path)
	}

	if file.Name != "" {
		if err := ValidateName(file.Name); err != nil {
			return Entry{}, fmt.Errorf("invalid theme name %q: %w", file.Name, err)
		}
		if file.Name != name {
			return Entry{}, fmt.Errorf("theme name %q must match filename %q", file.Name, name)
		}
	}

	theme, err := file.theme()
	if err != nil {
		return Entry{}, fmt.Errorf("invalid theme file %q: %w", path, err)
	}

	return Entry{Name: name, Theme: theme, Source: SourceUser, Path: path}, nil
}

// ValidateName rejects empty names, path traversal, path separators, control
// characters, and names outside the portable identifier/filename subset.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("theme name cannot be empty")
	}
	if name == "." || name == ".." {
		return errors.New("theme name cannot be . or ..")
	}
	if strings.ContainsAny(name, `/\`) {
		return errors.New("theme name cannot contain path separators")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return errors.New("theme name cannot contain control characters")
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.') {
			return fmt.Errorf("theme name contains unsupported character %q", r)
		}
	}

	return nil
}

type userThemeFile struct {
	Name                  string  `json:"name,omitempty"`
	Foreground            *string `json:"foreground"`
	ForegroundDim         *string `json:"foregroundDim"`
	ForegroundBright      *string `json:"foregroundBright"`
	Muted                 *string `json:"muted"`
	MutedBright           *string `json:"mutedBright"`
	Background            *string `json:"background"`
	BackgroundTint        *string `json:"backgroundTint"`
	BackgroundTint2       *string `json:"backgroundTint2"`
	Border                *string `json:"border"`
	BorderFocused         *string `json:"borderFocused"`
	Success               *string `json:"success"`
	Error                 *string `json:"error"`
	Warning               *string `json:"warning"`
	BackgroundTintPending *string `json:"backgroundTintPending"`
	BackgroundTintSuccess *string `json:"backgroundTintSuccess"`
	BackgroundTintError   *string `json:"backgroundTintError"`
	Accent                *string `json:"accent"`
	AccentDim             *string `json:"accentDim"`
	AccentBright          *string `json:"accentBright"`
}

func (f userThemeFile) theme() (*palette.Theme, error) {
	fields := []struct {
		name  string
		value *string
	}{
		{"foreground", f.Foreground},
		{"foregroundDim", f.ForegroundDim},
		{"foregroundBright", f.ForegroundBright},
		{"muted", f.Muted},
		{"mutedBright", f.MutedBright},
		{"background", f.Background},
		{"backgroundTint", f.BackgroundTint},
		{"backgroundTint2", f.BackgroundTint2},
		{"border", f.Border},
		{"borderFocused", f.BorderFocused},
		{"success", f.Success},
		{"error", f.Error},
		{"warning", f.Warning},
		{"backgroundTintPending", f.BackgroundTintPending},
		{"backgroundTintSuccess", f.BackgroundTintSuccess},
		{"backgroundTintError", f.BackgroundTintError},
		{"accent", f.Accent},
		{"accentDim", f.AccentDim},
		{"accentBright", f.AccentBright},
	}

	for _, field := range fields {
		if field.value == nil {
			return nil, fmt.Errorf("missing required field %q", field.name)
		}
		if !hexColorPattern.MatchString(*field.value) {
			return nil, fmt.Errorf("field %q must be a #RRGGBB color", field.name)
		}
	}

	return &palette.Theme{
		Foreground:            *f.Foreground,
		ForegroundDim:         *f.ForegroundDim,
		ForegroundBright:      *f.ForegroundBright,
		Muted:                 *f.Muted,
		MutedBright:           *f.MutedBright,
		Background:            *f.Background,
		BackgroundTint:        *f.BackgroundTint,
		BackgroundTint2:       *f.BackgroundTint2,
		Border:                *f.Border,
		BorderFocused:         *f.BorderFocused,
		Success:               *f.Success,
		Error:                 *f.Error,
		Warning:               *f.Warning,
		BackgroundTintPending: *f.BackgroundTintPending,
		BackgroundTintSuccess: *f.BackgroundTintSuccess,
		BackgroundTintError:   *f.BackgroundTintError,
		Accent:                *f.Accent,
		AccentDim:             *f.AccentDim,
		AccentBright:          *f.AccentBright,
	}, nil
}

func cloneTheme(theme *palette.Theme) *palette.Theme {
	if theme == nil {
		return nil
	}
	t := *theme
	return &t
}
