package tui

import (
	"sort"
	"strings"

	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	tuievents "github.com/weave-agent/weave-tui/internal/events"
)

// buildLoginProviders returns all providers available for login: registered
// providers (API key) plus OAuth providers.
func buildLoginProviders() []tuievents.LoginProviderEntry {
	seen := make(map[string]bool)
	entries := make([]tuievents.LoginProviderEntry, 0)

	// Add OAuth providers first
	for _, oauth := range sdk.ListOAuthProviders() {
		seen[oauth.ID] = true
		entries = append(entries, tuievents.LoginProviderEntry{
			Name:    oauth.Name,
			ID:      oauth.ID,
			IsOAuth: true,
			HasAuth: sdkmodel.ProviderHasAuth(oauth.ID),
		})
	}

	// Add regular providers
	for _, name := range sdk.ListProviders() {
		if seen[name] {
			continue
		}

		seen[name] = true
		entries = append(entries, tuievents.LoginProviderEntry{
			Name:    displayNameForProvider(name),
			ID:      name,
			IsOAuth: false,
			HasAuth: sdkmodel.ProviderHasAuth(name),
		})
	}

	// Add providers from model registry that aren't registered yet
	for _, md := range sdkmodel.ListAllModels() {
		if seen[md.Provider] {
			continue
		}

		seen[md.Provider] = true
		_, isOAuth := sdk.GetOAuthProvider(md.Provider)
		entries = append(entries, tuievents.LoginProviderEntry{
			Name:    displayNameForProvider(md.Provider),
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

// displayNameForProvider returns a human-readable name for a provider ID.
func displayNameForProvider(id string) string {
	if md, ok := sdkmodel.GetModel(id); ok && md.DisplayName != "" {
		return md.DisplayName
	}

	if id != "" {
		return strings.ToUpper(id[:1]) + id[1:]
	}

	return id
}

// buildLogoutProviders returns only providers that have auth configured.
func buildLogoutProviders() []tuievents.LogoutProviderEntry {
	seen := make(map[string]bool)
	entries := make([]tuievents.LogoutProviderEntry, 0)

	for _, name := range sdk.ListProviders() {
		if !sdkmodel.ProviderHasAuth(name) {
			continue
		}

		seen[name] = true
		entries = append(entries, tuievents.LogoutProviderEntry{
			Name: displayNameForProvider(name),
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
			Name: displayNameForProvider(md.Provider),
			ID:   md.Provider,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries
}
