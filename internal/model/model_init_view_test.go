package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelViewDefaultContent(t *testing.T) {
	m := NewModelWithConfig(nil, nil, nil, nil, TUIConfig{})

	view := m.View()

	assert.Contains(t, view.Content, "weave")
	assert.Contains(t, view.Content, "\n")
	assert.True(t, view.AltScreen)
	assert.True(t, view.KeyboardEnhancements.ReportAllKeysAsEscapeCodes)
	assert.True(t, view.KeyboardEnhancements.ReportAssociatedText)
}

func TestModelInitReturnsCommand(t *testing.T) {
	m := NewModelWithConfig(nil, nil, nil, nil, TUIConfig{})

	cmd := m.Init()

	assert.NotNil(t, cmd)
}
