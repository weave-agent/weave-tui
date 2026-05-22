package auth

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	tuievents "github.com/weave-agent/weave-tui/internal/events"

	tea "charm.land/bubbletea/v2"
)

// BuildLoginProviders returns all providers available for login: registered
// providers (API key) plus OAuth providers.
func BuildLoginProviders() []tuievents.LoginProviderEntry {
	seen := make(map[string]bool)
	entries := make([]tuievents.LoginProviderEntry, 0)

	// Add OAuth providers first.
	for _, oauth := range sdk.ListOAuthProviders() {
		seen[oauth.ID] = true
		entries = append(entries, tuievents.LoginProviderEntry{
			Name:    oauth.Name,
			ID:      oauth.ID,
			IsOAuth: true,
			HasAuth: sdkmodel.ProviderHasAuth(oauth.ID),
		})
	}

	for _, name := range sdk.ListProviders() {
		if seen[name] {
			continue
		}

		seen[name] = true
		entries = append(entries, tuievents.LoginProviderEntry{
			Name:    DisplayNameForProvider(name),
			ID:      name,
			IsOAuth: false,
			HasAuth: sdkmodel.ProviderHasAuth(name),
		})
	}

	for _, md := range sdkmodel.ListAllModels() {
		if seen[md.Provider] {
			continue
		}

		seen[md.Provider] = true
		_, isOAuth := sdk.GetOAuthProvider(md.Provider)
		entries = append(entries, tuievents.LoginProviderEntry{
			Name:    DisplayNameForProvider(md.Provider),
			ID:      md.Provider,
			IsOAuth: isOAuth,
			HasAuth: sdkmodel.ProviderHasAuth(md.Provider),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries
}

// DisplayNameForProvider returns a human-readable name for a provider ID.
func DisplayNameForProvider(id string) string {
	if md, ok := sdkmodel.GetModel(id); ok && md.DisplayName != "" {
		return md.DisplayName
	}

	if id != "" {
		return strings.ToUpper(id[:1]) + id[1:]
	}

	return id
}

// BuildLogoutProviders returns only providers that have auth configured.
func BuildLogoutProviders() []tuievents.LogoutProviderEntry {
	seen := make(map[string]bool)
	entries := make([]tuievents.LogoutProviderEntry, 0)

	for _, name := range sdk.ListProviders() {
		if !sdkmodel.ProviderHasAuth(name) {
			continue
		}

		seen[name] = true
		entries = append(entries, tuievents.LogoutProviderEntry{
			Name: DisplayNameForProvider(name),
			ID:   name,
		})
	}

	for _, oauth := range sdk.ListOAuthProviders() {
		if seen[oauth.ID] {
			continue
		}

		if !sdkmodel.ProviderHasAuth(oauth.ID) {
			continue
		}

		seen[oauth.ID] = true
		entries = append(entries, tuievents.LogoutProviderEntry{
			Name: oauth.Name,
			ID:   oauth.ID,
		})
	}

	for _, md := range sdkmodel.ListAllModels() {
		if seen[md.Provider] {
			continue
		}

		if !sdkmodel.ProviderHasAuth(md.Provider) {
			continue
		}

		seen[md.Provider] = true
		entries = append(entries, tuievents.LogoutProviderEntry{
			Name: DisplayNameForProvider(md.Provider),
			ID:   md.Provider,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries
}

// RunOAuthFlowCmd starts the OAuth authorization code flow, opens the browser,
// and returns a LoginAuthURLMsg containing the full authorization URL.
func RunOAuthFlowCmd(parentCtx context.Context, provider sdk.OAuthProvider, gen int) tea.Cmd {
	return func() tea.Msg {
		authCodeURL, handle, err := sdk.StartAuthorizationCodeFlow(parentCtx, provider.AuthURL, provider.TokenURL, provider.ClientID, provider.RedirectURI, provider.Scopes, provider.ExtraAuthParams)
		if err != nil {
			return tuievents.LoginFlowResultMsg{
				Provider: provider.ID,
				Error:    err,
				Gen:      gen,
			}
		}

		return tuievents.LoginAuthURLMsg{
			Provider: provider.ID,
			URL:      authCodeURL,
			Handle:   handle,
			Gen:      gen,
		}
	}
}

// CompleteOAuthFlowCmd waits for the OAuth callback and exchanges the
// authorization code for tokens.
func CompleteOAuthFlowCmd(parentCtx context.Context, handle *sdk.AuthorizationFlowHandle, providerID string, gen int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()

		cred, err := sdk.CompleteAuthorizationCodeFlow(ctx, handle)

		return tuievents.LoginFlowResultMsg{
			Provider:   providerID,
			Credential: cred,
			Error:      err,
			Gen:        gen,
		}
	}
}

// PollDeviceCodeCmd polls the token endpoint for a device code flow and
// returns a LoginFlowResultMsg.
func PollDeviceCodeCmd(parentCtx context.Context, providerID, deviceCode string, intervalSecs int, tokenURL, clientID string, gen int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()

		tokenResp, err := sdk.PollDeviceToken(ctx, tokenURL, clientID, deviceCode, intervalSecs)
		if err != nil {
			return tuievents.LoginFlowResultMsg{
				Provider: providerID,
				Error:    err,
				Gen:      gen,
			}
		}

		cred := sdk.OAuthCredential{
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			TokenType:    tokenResp.TokenType,
		}

		if tokenResp.ExpiresIn > 0 {
			cred.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		}

		return tuievents.LoginFlowResultMsg{
			Provider:   providerID,
			Credential: cred,
			Gen:        gen,
		}
	}
}
