package tui

import (
	"sort"

	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	tuievents "github.com/weave-agent/weave-tui/internal/events"
)

// listProviders builds a list of all known providers with their API key status.
// Combines registered providers from sdk.ListProviders() with the model registry
// to include providers that may not be registered yet but have known models.
func listProviders() []tuievents.ProviderEntry {
	seen := make(map[string]bool)

	var entries []tuievents.ProviderEntry

	for _, name := range sdk.ListProviders() {
		seen[name] = true
		entries = append(entries, tuievents.ProviderEntry{
			Name:   name,
			HasKey: sdkmodel.ProviderHasAuth(name),
		})
	}

	for _, md := range sdkmodel.ListAllModels() {
		if !seen[md.Provider] {
			seen[md.Provider] = true
			entries = append(entries, tuievents.ProviderEntry{
				Name:   md.Provider,
				HasKey: sdkmodel.ProviderHasAuth(md.Provider),
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries
}
