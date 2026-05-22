package ui

import (
	"github.com/weave-agent/weave-tui/internal/contract"
	tuievents "github.com/weave-agent/weave-tui/internal/events"
	"github.com/weave-agent/weave-tui/internal/palette"
)

type (
	PanelConfig          = contract.PanelConfig
	PanelDrawer          = contract.PanelDrawer
	PanelTrayAPI         = contract.PanelTrayAPI
	RichToolRenderer     = contract.RichToolRenderer
	TUIComponent         = contract.TUIComponent
	KeyEvent             = contract.KeyEvent
	AutocompleteProvider = contract.AutocompleteProvider
	ThemeDef             = contract.ThemeDef
)

type (
	overlayRequest  = tuievents.OverlayRequest
	overlayResponse = tuievents.OverlayResponse
)

const (
	requestSelect      = tuievents.RequestSelect
	requestConfirm     = tuievents.RequestConfirm
	requestInput       = tuievents.RequestInput
	requestEditor      = tuievents.RequestEditor
	requestMultiSelect = tuievents.RequestMultiSelect
)

type (
	popupPendingMsg         = tuievents.PopupPendingMsg
	extStatusMsg            = tuievents.ExtStatusMsg
	slashCommandsUpdatedMsg = tuievents.SlashCommandsUpdatedMsg
	themeChangedMsg         = tuievents.ThemeChangedMsg
	panelChangedMsg         = tuievents.PanelChangedMsg
	setEditorTextMsg        = tuievents.SetEditorTextMsg
	pasteToEditorMsg        = tuievents.PasteToEditorMsg
	editorTextRequestMsg    = tuievents.EditorTextRequestMsg
	setFooterMsg            = tuievents.SetFooterMsg
	setHeaderMsg            = tuievents.SetHeaderMsg
	setWorkingFramesMsg     = tuievents.SetWorkingFramesMsg
)

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
