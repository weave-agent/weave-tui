package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"testing"

	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	tuievents "github.com/weave-agent/weave-tui/internal/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCycleModel(t *testing.T) {
	entries := []tuievents.ModelEntry{
		{Provider: "anthropic", Model: "claude-sonnet-4-6"},
		{Provider: "openai", Model: "gpt-5.5"},
		{Provider: "zai", Model: "glm-5.1"},
	}

	next := CycleModel(entries, entries[0])
	assert.Equal(t, "openai", next.Provider)

	next = CycleModel(entries, entries[1])
	assert.Equal(t, "zai", next.Provider)

	next = CycleModel(entries, entries[2])
	assert.Equal(t, "anthropic", next.Provider)
}

func TestCycleModel_SingleEntry(t *testing.T) {
	entries := []tuievents.ModelEntry{
		{Provider: "anthropic", Model: "claude-sonnet-4-6"},
	}
	next := CycleModel(entries, entries[0])
	assert.Equal(t, "anthropic", next.Provider)
}

func TestCycleModel_Empty(t *testing.T) {
	cur := tuievents.ModelEntry{Provider: "anthropic", Model: "claude-sonnet-4-6"}
	next := CycleModel(nil, cur)
	assert.Equal(t, cur, next)
}

func TestCurrentModel(t *testing.T) {
	entries := []tuievents.ModelEntry{
		{Provider: "openai", Model: "gpt-5.5"},
		{Provider: "zai", Model: "glm-5.1"},
	}
	cur := CurrentModel(entries, nil)
	assert.Equal(t, "openai", cur.Provider)

	cur = CurrentModel(nil, nil)
	assert.Empty(t, cur.Provider)
}

func TestCurrentModel_EnvProvider(t *testing.T) {
	entries := []tuievents.ModelEntry{
		{Provider: "anthropic", Model: "claude-sonnet-4-6"},
		{Provider: "openai", Model: "gpt-5.5"},
	}

	t.Setenv("WEAVE_PROVIDER", "openai")

	cur := CurrentModel(entries, nil)
	assert.Equal(t, "openai", cur.Provider)
	assert.Equal(t, "gpt-5.5", cur.Model)
}

func TestCurrentModel_EnvProviderNotInEntries(t *testing.T) {
	entries := []tuievents.ModelEntry{
		{Provider: "openai", Model: "gpt-5.5"},
	}

	t.Setenv("WEAVE_PROVIDER", "anthropic")

	cur := CurrentModel(entries, nil)
	assert.Equal(t, "openai", cur.Provider)
}

func TestCurrentModel_PreferencesProviderOnly(t *testing.T) {
	sdkmodel.ResetModelRegistry()
	defer sdkmodel.ResetModelRegistry()

	sdkmodel.RegisterModel(sdkmodel.ModelDef{ID: "gpt-5.5", Provider: "openai", Default: true})
	sdkmodel.RegisterModel(sdkmodel.ModelDef{ID: "gpt-5.4", Provider: "openai"})

	entries := []tuievents.ModelEntry{
		{Provider: "openai", Model: "gpt-5.5"},
		{Provider: "anthropic", Model: "claude-sonnet-4-6"},
	}

	mock := &mockPreferenceStore{
		preferences: map[string]string{
			"provider": "openai",
		},
	}

	cur := CurrentModel(entries, mock)
	assert.Equal(t, "openai", cur.Provider)
	assert.Equal(t, "gpt-5.5", cur.Model)
}

func TestCurrentModel_PreferencesProviderOnly_NoRegistryFallback(t *testing.T) {
	sdkmodel.ResetModelRegistry()
	defer sdkmodel.ResetModelRegistry()

	entries := []tuievents.ModelEntry{
		{Provider: "openai", Model: "gpt-5.5"},
		{Provider: "anthropic", Model: "claude-sonnet-4-6"},
	}

	mock := &mockPreferenceStore{
		preferences: map[string]string{
			"provider": "openai",
		},
	}

	cur := CurrentModel(entries, mock)
	assert.Equal(t, "openai", cur.Provider)
	assert.Equal(t, "gpt-5.5", cur.Model)
}

func TestCurrentModel_LayeredSettings(t *testing.T) {
	entries := []tuievents.ModelEntry{
		{Provider: "openai", Model: "gpt-5.5"},
		{Provider: "anthropic", Model: "claude-sonnet-4-6"},
	}

	mock := &mockPreferenceStore{
		preferences: map[string]string{
			"provider": "openai",
			"model":    "gpt-5.5",
		},
	}

	cur := CurrentModel(entries, mock)
	assert.Equal(t, "openai", cur.Provider)
	assert.Equal(t, "gpt-5.5", cur.Model)
}

func TestCurrentModel_EnvOverridesSettings(t *testing.T) {
	entries := []tuievents.ModelEntry{
		{Provider: "anthropic", Model: "claude-sonnet-4-6"},
		{Provider: "openai", Model: "gpt-5.5"},
	}

	t.Setenv("WEAVE_PROVIDER", "openai")

	mock := &mockPreferenceStore{
		preferences: map[string]string{
			"provider": "anthropic",
			"model":    "claude-sonnet-4-6",
		},
	}

	cur := CurrentModel(entries, mock)
	assert.Equal(t, "openai", cur.Provider)
	assert.Equal(t, "gpt-5.5", cur.Model)
}

func TestListProvidersIncludesRegisteredAndModelRegistryProviders(t *testing.T) {
	resetProviderTestRegistries(t)

	sdk.RegisterProvider[struct{}, struct{}]("openai", func(_ sdk.Config, _, _ struct{}) (sdk.Provider, error) {
		return nil, errors.New("test provider not implemented")
	})
	sdkmodel.RegisterModel(sdkmodel.ModelDef{ID: "claude-sonnet-4-6", Provider: "anthropic"})
	sdkmodel.SetProviderAuth("openai", true)

	entries := ListProviders()
	require.Len(t, entries, 2)
	assert.Equal(t, []tuievents.ProviderEntry{
		{Name: "anthropic", HasKey: false},
		{Name: "openai", HasKey: true},
	}, entries)
}

func TestListModelsWithRegistry(t *testing.T) {
	resetProviderTestRegistries(t)

	for _, def := range []sdkmodel.ModelDef{
		{ID: "claude-opus-4-7", Provider: "anthropic", DisplayName: "Claude Opus 4.7", Reasoning: true, SupportsXHigh: true},
		{ID: "claude-sonnet-4-6", Provider: "anthropic", DisplayName: "Claude Sonnet 4.6", Reasoning: true, Default: true},
		{ID: "claude-opus-4-5", Provider: "anthropic", DisplayName: "Claude Opus 4.5", Reasoning: true},
		{ID: "claude-sonnet-4-5", Provider: "anthropic", DisplayName: "Claude Sonnet 4.5", Reasoning: true},
		{ID: "claude-haiku-4-5", Provider: "anthropic", DisplayName: "Claude Haiku 4.5", Reasoning: true},
		{ID: "gpt-5.5", Provider: "openai", DisplayName: "GPT-5.5", Reasoning: true, Default: true},
		{ID: "gpt-5.4", Provider: "openai", DisplayName: "GPT-5.4", Reasoning: true},
		{ID: "gpt-5.2", Provider: "openai", DisplayName: "GPT-5.2", Reasoning: true},
		{ID: "gpt-4.1", Provider: "openai", DisplayName: "GPT-4.1"},
		{ID: "o4-mini", Provider: "openai", DisplayName: "o4-mini", Reasoning: true},
		{ID: "o3", Provider: "openai", DisplayName: "o3", Reasoning: true},
		{ID: "glm-5.1", Provider: "zai", DisplayName: "GLM-5.1", Reasoning: true, Default: true},
		{ID: "glm-5", Provider: "zai", DisplayName: "GLM-5", Reasoning: true},
		{ID: "glm-4.7", Provider: "zai", DisplayName: "GLM-4.7", Reasoning: true},
		{ID: "glm-4.7-flash", Provider: "zai", DisplayName: "GLM-4.7 Flash", Reasoning: true},
		{ID: "glm-4.7-flashx", Provider: "zai", DisplayName: "GLM-4.7 FlashX", Reasoning: true},
		{ID: "glm-4.5-air", Provider: "zai", DisplayName: "GLM-4.5 Air", Reasoning: true},
	} {
		sdkmodel.RegisterModel(def)
	}

	sdk.RegisterProvider[struct{}, struct{}]("anthropic", func(_ sdk.Config, _, _ struct{}) (sdk.Provider, error) { return nil, nil }) //nolint:nilnil // stub registration for model list tests
	sdk.RegisterProvider[struct{}, struct{}]("openai", func(_ sdk.Config, _, _ struct{}) (sdk.Provider, error) { return nil, nil })    //nolint:nilnil // stub registration for model list tests
	sdk.RegisterProvider[struct{}, struct{}]("zai", func(_ sdk.Config, _, _ struct{}) (sdk.Provider, error) { return nil, nil })       //nolint:nilnil // stub registration for model list tests

	sdkmodel.SetProviderAuth("anthropic", true)
	sdkmodel.SetProviderAuth("openai", true)
	sdkmodel.SetProviderAuth("zai", true)

	entries := ListModels()
	assert.NotEmpty(t, entries, "should return models from registry")

	providerSet := make(map[string]bool)
	for _, e := range entries {
		providerSet[e.Provider] = true
	}

	assert.True(t, providerSet["anthropic"], "should include anthropic models")
	assert.True(t, providerSet["openai"], "should include openai models")
	assert.True(t, providerSet["zai"], "should include zai models")
}

func TestListModelsEmpty(t *testing.T) {
	resetProviderTestRegistries(t)

	entries := ListModels()
	assert.Nil(t, entries)
}

func TestListModelsIgnoresEnvOverrides(t *testing.T) {
	resetProviderTestRegistries(t)

	for _, def := range []sdkmodel.ModelDef{
		{ID: "claude-opus-4-7", Provider: "anthropic", Reasoning: true},
		{ID: "claude-sonnet-4-6", Provider: "anthropic", Reasoning: true},
		{ID: "claude-opus-4-5", Provider: "anthropic", Reasoning: true},
		{ID: "claude-sonnet-4-5", Provider: "anthropic", Reasoning: true},
		{ID: "claude-haiku-4-5", Provider: "anthropic", Reasoning: true},
	} {
		sdkmodel.RegisterModel(def)
	}

	sdk.RegisterProvider[struct{}, struct{}]("anthropic", func(_ sdk.Config, _, _ struct{}) (sdk.Provider, error) { return nil, nil }) //nolint:nilnil // stub registration for model list tests
	sdkmodel.SetProviderAuth("anthropic", true)

	entries := ListModels()

	anthropicCount := 0

	for _, e := range entries {
		if e.Provider == "anthropic" {
			anthropicCount++
		}
	}

	assert.Equal(t, 5, anthropicCount, "should show all anthropic models")
}

func TestInitialThinkingLevel_LayeredSettings(t *testing.T) {
	mock := &mockPreferenceStore{
		preferences: map[string]string{
			"thinking_level": "high",
		},
	}

	level := InitialThinkingLevel(mock)
	assert.Equal(t, sdkmodel.ThinkingHigh, level)
}

func TestInitialThinkingLevel_InvalidPreference(t *testing.T) {
	mock := &mockPreferenceStore{
		preferences: map[string]string{
			"thinking_level": "bogus",
		},
	}

	level := InitialThinkingLevel(mock)
	assert.Equal(t, sdkmodel.ThinkingMedium, level)
}

func TestSaveSettings_CallsSavePreferences(t *testing.T) {
	var captured preferences

	mock := &mockPreferenceStore{
		savePreferences: func(target any) error {
			data, err := json.Marshal(target)
			if err != nil {
				return fmt.Errorf("marshal: %w", err)
			}

			if err := json.Unmarshal(data, &captured); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}

			return nil
		},
	}

	SaveSettings(mock, tuievents.ModelEntry{Provider: "openai", Model: "gpt-5.5"}, sdkmodel.ThinkingHigh)

	assert.Equal(t, "openai", captured.Provider)
	assert.Equal(t, "gpt-5.5", captured.Model)
	assert.Equal(t, "high", captured.ThinkingLevel)
}

func TestSaveSettings_PreservesUIFields(t *testing.T) {
	stored := map[string]any{
		"provider":       "anthropic",
		"model":          "claude-sonnet-4-6",
		"thinking_level": "medium",
		"ui": map[string]any{
			"theme":            "dark",
			"editor_max_lines": 30,
		},
	}

	mock := &mockPreferenceStore{
		preferences: stored,
		savePreferences: func(target any) error {
			targetData, _ := json.Marshal(target)

			var targetMap map[string]any

			_ = json.Unmarshal(targetData, &targetMap)
			maps.Copy(stored, targetMap)

			return nil
		},
	}

	SaveSettings(mock, tuievents.ModelEntry{Provider: "openai", Model: "gpt-5.5"}, sdkmodel.ThinkingHigh)

	assert.Equal(t, "openai", stored["provider"])
	assert.Equal(t, "gpt-5.5", stored["model"])
	assert.Equal(t, "high", stored["thinking_level"])

	ui, ok := stored["ui"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "dark", ui["theme"])
	assert.Equal(t, 30, ui["editor_max_lines"])
}

func TestSaveSettings_NilConfig(t *testing.T) {
	assert.NotPanics(t, func() {
		SaveSettings(nil, tuievents.ModelEntry{Provider: "openai", Model: "gpt-5.5"}, sdkmodel.ThinkingHigh)
	})
}

type mockPreferenceStore struct {
	preferences     any
	savePreferences func(target any) error
}

func (m *mockPreferenceStore) Preferences(target any) error {
	if m.preferences == nil {
		return nil
	}

	data, err := json.Marshal(m.preferences)
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal preferences: %w", err)
	}

	return nil
}

func (m *mockPreferenceStore) SavePreferences(target any) error {
	if m.savePreferences != nil {
		return m.savePreferences(target)
	}

	return nil
}

func (m *mockPreferenceStore) SaveProviderKey(_, _ string) error { return nil }

func resetProviderTestRegistries(t *testing.T) {
	t.Helper()

	sdkmodel.ResetModelRegistry()
	sdk.ResetProviderRegistry()
	sdkmodel.ResetAuthRegistry()

	t.Cleanup(func() {
		sdkmodel.ResetModelRegistry()
		sdk.ResetProviderRegistry()
		sdkmodel.ResetAuthRegistry()
	})
}
