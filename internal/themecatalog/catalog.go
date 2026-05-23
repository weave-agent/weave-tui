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

	var file map[string]string

	dec := json.NewDecoder(strings.NewReader(string(data)))

	if decodeErr := dec.Decode(&file); decodeErr != nil {
		return Entry{}, fmt.Errorf("parse theme file %q: %w", path, decodeErr)
	}

	var trailing any

	if decodeErr := dec.Decode(&trailing); decodeErr != io.EOF {
		return Entry{}, fmt.Errorf("parse theme file %q: unexpected trailing JSON", path)
	}

	if fileName := file["name"]; fileName != "" {
		if nameErr := ValidateName(fileName); nameErr != nil {
			return Entry{}, fmt.Errorf("invalid theme name %q: %w", fileName, nameErr)
		}

		if fileName != name {
			return Entry{}, fmt.Errorf("theme name %q must match filename %q", fileName, name)
		}
	}

	theme, err := themeFromFile(file)
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
		return errors.New("theme name cannot be dot or dot-dot")
	}

	if strings.ContainsAny(name, `/\`) {
		return errors.New("theme name cannot contain path separators")
	}

	for _, r := range name {
		if unicode.IsControl(r) {
			return errors.New("theme name cannot contain control characters")
		}

		if !isThemeNameRune(r) {
			return fmt.Errorf("theme name contains unsupported character %q", r)
		}
	}

	return nil
}

func isThemeNameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.'
}

var themeColorFields = []string{
	"foreground",
	"foregroundDim",
	"foregroundBright",
	"muted",
	"mutedBright",
	"background",
	"backgroundTint",
	"backgroundTint2",
	"border",
	"borderFocused",
	"success",
	"error",
	"warning",
	"backgroundTintPending",
	"backgroundTintSuccess",
	"backgroundTintError",
	"accent",
	"accentDim",
	"accentBright",
}

func themeFromFile(values map[string]string) (*palette.Theme, error) {
	allowed := map[string]bool{"name": true}
	for _, field := range themeColorFields {
		allowed[field] = true
	}

	for field := range values {
		if !allowed[field] {
			return nil, fmt.Errorf("unknown field %q", field)
		}
	}

	for _, field := range themeColorFields {
		value, ok := values[field]
		if !ok {
			return nil, fmt.Errorf("missing required field %q", field)
		}

		if !hexColorPattern.MatchString(value) {
			return nil, fmt.Errorf("field %q must be a #RRGGBB color", field)
		}
	}

	return &palette.Theme{
		Foreground:            values["foreground"],
		ForegroundDim:         values["foregroundDim"],
		ForegroundBright:      values["foregroundBright"],
		Muted:                 values["muted"],
		MutedBright:           values["mutedBright"],
		Background:            values["background"],
		BackgroundTint:        values["backgroundTint"],
		BackgroundTint2:       values["backgroundTint2"],
		Border:                values["border"],
		BorderFocused:         values["borderFocused"],
		Success:               values["success"],
		Error:                 values["error"],
		Warning:               values["warning"],
		BackgroundTintPending: values["backgroundTintPending"],
		BackgroundTintSuccess: values["backgroundTintSuccess"],
		BackgroundTintError:   values["backgroundTintError"],
		Accent:                values["accent"],
		AccentDim:             values["accentDim"],
		AccentBright:          values["accentBright"],
	}, nil
}

func cloneTheme(theme *palette.Theme) *palette.Theme {
	if theme == nil {
		return nil
	}

	t := *theme

	return &t
}
