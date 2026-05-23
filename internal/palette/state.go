package palette

// State represents the current agent activity state, driving accent color changes.
type State int

const (
	StateIdle        State = iota // no activity
	StateStreaming                // receiving tokens from provider
	StateToolRunning              // executing a tool call
	StateError                    // error state
)

// AccentForState returns the Accent, AccentDim, and AccentBright color values
// for the given agent state using the default theme for idle and fallback states.
func AccentForState(s State) (accent, accentDim, accentBright string) {
	return AccentForStateInTheme(s, DefaultTheme())
}

// AccentForStateInTheme returns the dynamic accent colors for the given agent
// state. Active states keep the existing hardcoded tints; idle and fallback
// states restore the provided theme's accent values.
func AccentForStateInTheme(s State, theme *Theme) (accent, accentDim, accentBright string) {
	if theme == nil {
		theme = DefaultTheme()
	}

	switch s {
	case StateIdle:
		return theme.Accent, theme.AccentDim, theme.AccentBright
	case StateStreaming:
		return "45", "39", "51"
	case StateToolRunning:
		return "172", "166", "178"
	case StateError:
		return "167", "160", "173"
	default:
		return theme.Accent, theme.AccentDim, theme.AccentBright
	}
}
