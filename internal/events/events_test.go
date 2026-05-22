package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weave-agent/weave/sdk"
)

func TestModelEntryDisplay(t *testing.T) {
	entry := ModelEntry{Provider: "openai", Model: "gpt-5.5"}

	assert.Equal(t, "openai/gpt-5.5", entry.Display())
}

func TestProviderEntryDisplay(t *testing.T) {
	assert.Equal(t, "openai  key set", ProviderEntry{Name: "openai", HasKey: true}.Display())
	assert.Equal(t, "anthropic  no key", ProviderEntry{Name: "anthropic"}.Display())
}

func TestNotifyTypedMsgCarriesNotificationPayload(t *testing.T) {
	msg := NotifyTypedMsg{Message: "saved", Level: sdk.NotifySuccess}

	assert.Equal(t, "saved", msg.Message)
	assert.Equal(t, sdk.NotifySuccess, msg.Level)
}
