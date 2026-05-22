package tui

import (
	"github.com/weave-agent/weave-tui/internal/contract"
	"github.com/weave-agent/weave-tui/internal/palette"
)

// Public API aliases — canonical definitions live in internal/contract.

// TUIExtAPI provides TUI-specific extension capabilities.
type TUIExtAPI = contract.TUIExtAPI

// TUIExtension is a TUI-specific plugin that registers with the TUI's
// extension API. TUI extensions are discovered by the launcher and wired
// by the TUI at startup. They are silently skipped in headless mode.
type TUIExtension = contract.TUIExtension

// PanelTrayAPI provides access to the panel tray for tab ordering.
type PanelTrayAPI = contract.PanelTrayAPI

// RichToolRenderer renders tool output with theme access.
type RichToolRenderer = contract.RichToolRenderer

// TUIComponent is a replaceable UI component (header/footer).
type TUIComponent = contract.TUIComponent

// KeyEvent represents a raw terminal key event.
type KeyEvent = contract.KeyEvent

// AutocompleteProvider provides completion suggestions for the editor.
type AutocompleteProvider = contract.AutocompleteProvider

// AutocompleteContext provides context for autocomplete suggestions.
type AutocompleteContext = contract.AutocompleteContext

// AutocompleteSuggestion is a single autocomplete suggestion.
type AutocompleteSuggestion = contract.AutocompleteSuggestion

// ThemeDef defines a custom theme for registration.
type ThemeDef = contract.ThemeDef

// PanelPlacement determines where a panel is rendered relative to the editor.
type PanelPlacement = contract.PanelPlacement

// Panel placement constants.
const (
	AsOverlay   = contract.AsOverlay
	AboveEditor = contract.AboveEditor
	BelowEditor = contract.BelowEditor
	TrayOnly    = contract.TrayOnly
)

// PanelConfig configures a panel.
type PanelConfig = contract.PanelConfig

// PanelDrawer is the interface for panel content rendering and interaction.
type PanelDrawer = contract.PanelDrawer

// toPaletteTheme converts a ThemeDef to a palette.Theme.
func toPaletteTheme(td ThemeDef) *palette.Theme {
	return &palette.Theme{
		Accent:                td.Accent,
		AccentDim:             td.AccentDim,
		AccentBright:          td.AccentBright,
		Success:               td.Success,
		Error:                 td.Error,
		Warning:               td.Warning,
		Muted:                 td.Muted,
		MutedBright:           td.MutedBright,
		Border:                td.Border,
		BorderFocused:         td.BorderFocused,
		BackgroundTint:        td.BackgroundTint,
		BackgroundTintPending: td.BackgroundTintPending,
		BackgroundTintSuccess: td.BackgroundTintSuccess,
		BackgroundTintError:   td.BackgroundTintError,
		Foreground:            td.Foreground,
		ForegroundDim:         td.ForegroundDim,
		ForegroundBright:      td.ForegroundBright,
		Background:            td.Background,
		BackgroundTint2:       td.BackgroundTint2,
	}
}
