package model

import (
	tuiauth "github.com/weave-agent/weave-tui/internal/auth"
	tuievents "github.com/weave-agent/weave-tui/internal/events"
	tuiproviders "github.com/weave-agent/weave-tui/internal/providers"

	tea "charm.land/bubbletea/v2"
)

// listModelsCmd returns a tea.Cmd that lists available models.
func listModelsCmd() tea.Cmd {
	return func() tea.Msg {
		return tuievents.ModelListResultMsg{Models: tuiproviders.ListModels()}
	}
}

// listProvidersCmd returns a tea.Cmd that lists providers with key status.
func listProvidersCmd() tea.Cmd {
	return func() tea.Msg {
		return tuievents.ProviderListResultMsg{Providers: tuiproviders.ListProviders()}
	}
}

// loginCmd returns a tea.Cmd that lists providers available for login.
func loginCmd() tea.Cmd {
	return func() tea.Msg {
		return tuievents.LoginListResultMsg{Providers: tuiauth.BuildLoginProviders()}
	}
}

// logoutCmd returns a tea.Cmd that lists providers with configured auth.
func logoutCmd() tea.Cmd {
	return func() tea.Msg {
		return tuievents.LogoutListResultMsg{Providers: tuiauth.BuildLogoutProviders()}
	}
}
