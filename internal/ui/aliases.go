package ui

import (
	"github.com/weave-agent/weave-tui/internal/contract"
	tuievents "github.com/weave-agent/weave-tui/internal/events"
	"github.com/weave-agent/weave-tui/internal/palette"
)

type PanelConfig = contract.PanelConfig
type PanelDrawer = contract.PanelDrawer
type PanelTrayAPI = contract.PanelTrayAPI
type RichToolRenderer = contract.RichToolRenderer
type TUIComponent = contract.TUIComponent
type KeyEvent = contract.KeyEvent
type AutocompleteProvider = contract.AutocompleteProvider
type ThemeDef = contract.ThemeDef

type overlayRequest = tuievents.OverlayRequest
type overlayResponse = tuievents.OverlayResponse

const (
	requestSelect      = tuievents.RequestSelect
	requestConfirm     = tuievents.RequestConfirm
	requestInput       = tuievents.RequestInput
	requestEditor      = tuievents.RequestEditor
	requestMultiSelect = tuievents.RequestMultiSelect
)

type popupPendingMsg = tuievents.PopupPendingMsg
type extStatusMsg = tuievents.ExtStatusMsg
type slashCommandsUpdatedMsg = tuievents.SlashCommandsUpdatedMsg
type themeChangedMsg = tuievents.ThemeChangedMsg
type panelChangedMsg = tuievents.PanelChangedMsg
type setEditorTextMsg = tuievents.SetEditorTextMsg
type pasteToEditorMsg = tuievents.PasteToEditorMsg
type editorTextRequestMsg = tuievents.EditorTextRequestMsg
type setFooterMsg = tuievents.SetFooterMsg
type setHeaderMsg = tuievents.SetHeaderMsg
type setWorkingFramesMsg = tuievents.SetWorkingFramesMsg

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
