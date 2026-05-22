// Package styles provides a structured design grammar that maps palette.Theme
// tokens into product-specific render styles. Glyphs, spacing, and layout
// grammar are fixed constants; only color tokens change with the theme.
package styles

import (
	"github.com/weave-agent/weave-tui/palette"

	"charm.land/lipgloss/v2"
)

// Design grammar constants — these are fixed product identity and never
// change with custom themes.
const (
	UserMarker      = "❯"
	AssistantMarker = "◆"
	ThinkingMarker  = "∴"

	ToolPending     = "○"
	ToolSuccess     = "✓"
	ToolError       = "×"
	ToolInterrupted = "■"
)

// Styles maps palette.Theme tokens into lipgloss styles for TUI components.
// All methods return fresh styles each call so callers can further customize.
type Styles struct {
	theme *palette.Theme
}

// New creates a Styles wrapper around the given theme.
// If theme is nil, it falls back to palette.DefaultTheme().
func New(theme *palette.Theme) *Styles {
	if theme == nil {
		theme = palette.DefaultTheme()
	}
	return &Styles{theme: theme}
}

// Theme returns the underlying palette theme.
func (s *Styles) Theme() *palette.Theme {
	return s.theme
}

// --- Role markers ---

// UserMarkerStyle returns the style for the user message marker glyph.
func (s *Styles) UserMarkerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Foreground))
}

// UserMarkerRendered returns the styled user marker string.
func (s *Styles) UserMarkerRendered() string {
	return s.UserMarkerStyle().Render(UserMarker)
}

// AssistantMarkerStyle returns the style for the assistant message marker glyph.
func (s *Styles) AssistantMarkerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Muted))
}

// AssistantMarkerRendered returns the styled assistant marker string.
func (s *Styles) AssistantMarkerRendered() string {
	return s.AssistantMarkerStyle().Render(AssistantMarker)
}

// ThinkingMarkerStyle returns the style for the thinking block marker glyph.
func (s *Styles) ThinkingMarkerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.ForegroundDim))
}

// ThinkingMarkerRendered returns the styled thinking marker string.
func (s *Styles) ThinkingMarkerRendered() string {
	return s.ThinkingMarkerStyle().Render(ThinkingMarker)
}

// --- Text styles ---

// Foreground returns a style with the primary foreground color.
func (s *Styles) Foreground() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Foreground))
}

// ForegroundDim returns a style with the dimmed foreground color.
func (s *Styles) ForegroundDim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.ForegroundDim))
}

// Muted returns a style with the muted color for hints and secondary text.
func (s *Styles) Muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Muted))
}

// MutedBright returns a style with the brighter muted color.
func (s *Styles) MutedBright() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.MutedBright))
}

// Error returns a style with the error color.
func (s *Styles) Error() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Error))
}

// Success returns a style with the success color.
func (s *Styles) Success() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Success))
}

// Warning returns a style with the warning color.
func (s *Styles) Warning() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Warning))
}

// --- Accent styles ---

// Accent returns a style with the accent color.
func (s *Styles) Accent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.Accent))
}

// AccentDim returns a style with the dimmed accent color.
func (s *Styles) AccentDim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.AccentDim))
}

// AccentBright returns a style with the bright accent color.
func (s *Styles) AccentBright() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.theme.AccentBright))
}

// --- Border styles ---

// Border returns a style with the unfocused border color.
func (s *Styles) Border() lipgloss.Style {
	return lipgloss.NewStyle().BorderForeground(lipgloss.Color(s.theme.Border))
}

// BorderFocused returns a style with the focused border color.
func (s *Styles) BorderFocused() lipgloss.Style {
	return lipgloss.NewStyle().BorderForeground(lipgloss.Color(s.theme.BorderFocused))
}

// --- Selection / focus ---

// SelectedRow returns the style for a selected list row.
func (s *Styles) SelectedRow() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color(s.theme.Accent)).
		Foreground(lipgloss.Color(s.theme.Foreground))
}

// SelectedRowDim returns the style for a selected row with dimmed text.
func (s *Styles) SelectedRowDim() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(s.theme.Accent)).
		Foreground(lipgloss.Color(s.theme.ForegroundDim))
}

// FocusedTab returns the style for a focused tab (bracketed).
func (s *Styles) FocusedTab() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.AccentBright)).
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Padding(0, 1)
}

// ActiveTab returns the style for an active but unfocused tab.
func (s *Styles) ActiveTab() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.Accent)).
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Padding(0, 1)
}

// InactiveTab returns the style for an inactive tab.
func (s *Styles) InactiveTab() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.Muted)).
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Padding(0, 1)
}

// --- Pills ---

// Pill returns the base style for status/attachment pills.
func (s *Styles) Pill() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Foreground(lipgloss.Color(s.theme.Accent)).
		Padding(0, 1)
}

// PillError returns the style for an error/danger pill.
func (s *Styles) PillError() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Foreground(lipgloss.Color(s.theme.Error)).
		Padding(0, 1)
}

// PillMuted returns the style for a muted pill.
func (s *Styles) PillMuted() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Foreground(lipgloss.Color(s.theme.Muted)).
		Padding(0, 1)
}

// --- Tool state colors ---

// ToolPendingColor returns the color for a pending/running tool border.
func (s *Styles) ToolPendingColor() string {
	return s.theme.AccentDim
}

// ToolSuccessColor returns the color for a successful tool border.
func (s *Styles) ToolSuccessColor() string {
	return s.theme.Border
}

// ToolSuccessFlashedColor returns the flash color for a successful tool.
func (s *Styles) ToolSuccessFlashedColor() string {
	return s.theme.Success
}

// ToolErrorColor returns the color for an error tool border.
func (s *Styles) ToolErrorColor() string {
	return s.theme.Error
}

// ToolInterruptedColor returns the color for an interrupted tool border.
func (s *Styles) ToolInterruptedColor() string {
	return s.theme.Muted
}

// --- Overlay boxes ---

// OverlayBorder returns the border style for overlays/dialogs.
func (s *Styles) OverlayBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(s.theme.Accent))
}

// OverlayBox returns a bordered box style for overlays using the given color.
func (s *Styles) OverlayBox(color string) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color))
}

// --- Background tints ---

// BackgroundTint returns a style with the background tint color as background.
func (s *Styles) BackgroundTint() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(s.theme.BackgroundTint))
}

// BackgroundTint2 returns a style with the secondary background tint.
func (s *Styles) BackgroundTint2() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(s.theme.BackgroundTint2))
}

// --- Notification banner ---

// BannerInfo returns the style for an info banner/pill.
func (s *Styles) BannerInfo() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.AccentBright)).
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Padding(0, 1)
}

// BannerSuccess returns the style for a success banner/pill.
func (s *Styles) BannerSuccess() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.Success)).
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Padding(0, 1)
}

// BannerWarning returns the style for a warning banner/pill.
func (s *Styles) BannerWarning() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.Warning)).
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Padding(0, 1)
}

// BannerError returns the style for an error banner/pill.
func (s *Styles) BannerError() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.theme.Error)).
		Background(lipgloss.Color(s.theme.BackgroundTint)).
		Padding(0, 1)
}
