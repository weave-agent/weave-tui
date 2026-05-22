package extensionregistry

import (
	"fmt"
	"log/slog"

	"github.com/weave-agent/weave/sdk"
	"github.com/weave-agent/weave/sdk/registry"

	"github.com/weave-agent/weave-tui/internal/contract"
)

type entry struct {
	factory func(sdk.Config) (contract.TUIExtension, error)
}

var reg = registry.New[entry](
	registry.WithWarn[entry](func(name string) {
		slog.Warn("duplicate registration", "name", name, "kind", "tui extension")
	}),
)

// Register registers a TUI extension factory.
// The framework will automatically populate the config struct from settings,
// env vars, and CLI flags before calling the factory.
func Register[TConfig any](name string, factory func(sdk.Config, sdk.PreferenceReader, TConfig) (contract.TUIExtension, error)) {
	var zero TConfig
	sdk.RegisterExtensionSchema("ui_extensions", name, zero)

	wrapper := func(cfg sdk.Config) (contract.TUIExtension, error) {
		var t TConfig

		if err := cfg.ExtensionConfig("ui_extensions", name, &t); err != nil {
			return nil, fmt.Errorf("load tui extension config: %w", err)
		}

		return factory(sdk.ConfigReadOnly(cfg), sdk.PreferenceStoreFrom(cfg), t)
	}

	reg.Register(name, entry{factory: wrapper})
}

// Get instantiates a TUI extension by name with the given config.
func Get(name string, cfg sdk.Config) (contract.TUIExtension, error) {
	entry, ok := reg.Get(name)
	if !ok {
		return nil, fmt.Errorf("tui extension %q: %w", name, sdk.ErrNotRegistered)
	}

	return entry.factory(sdk.ConfigOrDefault(cfg))
}

// Registered reports whether a TUI extension with the given name is registered.
func Registered(name string) bool {
	return reg.Exists(name)
}

// List returns the names of all registered TUI extensions in sorted order.
func List() []string {
	return reg.List()
}

// GetAll instantiates all registered TUI extensions with the given config.
func GetAll(cfg sdk.Config) []contract.TUIExtension {
	names := reg.List()
	exts := make([]contract.TUIExtension, 0, len(names))

	for _, name := range names {
		ext, err := Get(name, cfg)
		if err != nil {
			sdk.Logger("tui").Error("failed to instantiate TUI extension", "name", name, "error", err)

			continue
		}

		exts = append(exts, ext)
	}

	return exts
}

// Reset clears all registered TUI extensions.
func Reset() {
	reg.Reset()
}
