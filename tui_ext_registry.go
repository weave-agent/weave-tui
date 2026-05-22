package tui

import (
	"fmt"

	"github.com/weave-agent/weave/sdk"

	"github.com/weave-agent/weave-tui/internal/extensionregistry"
)

// RegisterTUIExtension registers a TUI extension factory.
// The framework will automatically populate the config struct from settings, env vars,
// and CLI flags before calling the factory.
func RegisterTUIExtension[TConfig any](name string, factory func(sdk.Config, sdk.PreferenceReader, TConfig) (TUIExtension, error)) {
	extensionregistry.Register(name, factory)
}

// GetTUIExtension instantiates a TUI extension by name with the given config.
func GetTUIExtension(name string, cfg sdk.Config) (TUIExtension, error) {
	ext, err := extensionregistry.Get(name, cfg)
	if err != nil {
		return nil, fmt.Errorf("get TUI extension %q: %w", name, err)
	}

	return ext, nil
}

// TUIExtensionRegistered reports whether a TUI extension with the given name
// is registered.
func TUIExtensionRegistered(name string) bool {
	return extensionregistry.Registered(name)
}

// ListTUIExtensions returns the names of all registered TUI extensions in sorted order.
func ListTUIExtensions() []string {
	return extensionregistry.List()
}

// ResetTUIExtensionRegistry clears all registered TUI extensions.
func ResetTUIExtensionRegistry() {
	extensionregistry.Reset()
}
