package tui

import (
	"github.com/weave-agent/weave/sdk"

	tuiapp "github.com/weave-agent/weave-tui/internal/app"
)

//nolint:gochecknoinits // Extension registration requires init-time side effects for SDK discovery
func init() {
	registerExtension()
}

func registerExtension() {
	sdk.RegisterExtensionWithScopeAndWriter("tui", "ui", tuiapp.NewExtension)
}
