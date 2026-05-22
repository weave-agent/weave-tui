package tui

import (
	"testing"

	"github.com/weave-agent/weave/sdk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootRegistrationUsesUIScope(t *testing.T) {
	sdk.ResetExtensionRegistry()
	t.Cleanup(func() {
		sdk.ResetExtensionRegistry()
		registerExtension()
	})

	registerExtension()

	assert.True(t, sdk.ExtensionRegistered("tui"))

	schema, ok := sdk.GetSchema("ui", "tui")
	require.True(t, ok)

	fields := make(map[string]sdk.SchemaField, len(schema.Fields))
	for _, field := range schema.Fields {
		fields[field.JSONName] = field
	}

	assert.Contains(t, fields, "theme")
	assert.Contains(t, fields, "editor_max_lines")
}
