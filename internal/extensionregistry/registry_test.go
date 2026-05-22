package extensionregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weave-agent/weave/sdk"

	"github.com/weave-agent/weave-tui/internal/contract"
)

type stubTUIExtension struct {
	name       string
	config     stubTUIExtConfig
	registered bool
}

type stubTUIExtConfig struct {
	Enabled bool `json:"enabled"`
}

type configBackedPreferenceStore struct {
	sdk.NoopConfig
	extensionConfig any
	extensionErr    error
}

func (c *configBackedPreferenceStore) ExtensionConfig(scope, name string, target any) error {
	if c.extensionErr != nil {
		return c.extensionErr
	}

	if scope != "ui_extensions" || name == "" || c.extensionConfig == nil {
		return nil
	}

	data, err := json.Marshal(c.extensionConfig)
	if err != nil {
		return fmt.Errorf("marshal extension config: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal extension config: %w", err)
	}

	return nil
}

func (c *configBackedPreferenceStore) Preferences(any) error { return nil }

func (c *configBackedPreferenceStore) SavePreferences(any) error { return nil }

func (c *configBackedPreferenceStore) SaveProviderKey(_, _ string) error { return nil }

func (e *stubTUIExtension) Name() string                     { return e.name }
func (e *stubTUIExtension) RegisterTUI(_ contract.TUIExtAPI) { e.registered = true }

func TestRegister(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	ext := &stubTUIExtension{name: "test-ext"}

	Register("test-ext", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return ext, nil
	})

	exts := GetAll(sdk.NoopConfig{})
	require.Len(t, exts, 1)
	assert.Equal(t, "test-ext", exts[0].Name())
}

func TestRegisterWithConfig(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var receivedCfg stubTUIExtConfig

	Register("config-ext", func(_ sdk.Config, _ sdk.PreferenceReader, cfg stubTUIExtConfig) (contract.TUIExtension, error) {
		receivedCfg = cfg
		return &stubTUIExtension{name: "config-ext", config: cfg}, nil
	})

	ext, err := Get("config-ext", sdk.NoopConfig{})
	require.NoError(t, err)
	assert.Equal(t, "config-ext", ext.Name())
	assert.False(t, receivedCfg.Enabled)
}

func TestRegisterLoadsConfiguredValuesAndReadOnlyInputs(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var receivedCfg stubTUIExtConfig

	var (
		configIsWriter bool
		prefsAreWriter bool
	)

	Register("configured-ext", func(cfg sdk.Config, prefs sdk.PreferenceReader, cfgValue stubTUIExtConfig) (contract.TUIExtension, error) {
		receivedCfg = cfgValue
		_, configIsWriter = cfg.(sdk.PreferenceWriter)
		_, prefsAreWriter = prefs.(sdk.PreferenceWriter)

		return &stubTUIExtension{name: "configured-ext", config: cfgValue}, nil
	})

	ext, err := Get("configured-ext", &configBackedPreferenceStore{
		extensionConfig: stubTUIExtConfig{Enabled: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "configured-ext", ext.Name())
	assert.True(t, receivedCfg.Enabled)
	assert.False(t, configIsWriter)
	assert.False(t, prefsAreWriter)
}

func TestRegisterReturnsExtensionConfigError(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	configErr := errors.New("bad extension config")

	Register("bad-config-ext", func(_ sdk.Config, _ sdk.PreferenceReader, _ stubTUIExtConfig) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "bad-config-ext"}, nil
	})

	_, err := Get("bad-config-ext", &configBackedPreferenceStore{extensionErr: configErr})
	require.Error(t, err)
	require.ErrorIs(t, err, configErr)
	assert.Contains(t, err.Error(), "load tui extension config")
}

func TestRegisterDuplicateWarns(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	first := &stubTUIExtension{name: "dup"}

	Register("dup", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return first, nil
	})

	// Second registration should be a no-op with a warning (no panic).
	Register("dup", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "dup"}, nil
	})

	// First registration wins.
	exts := GetAll(sdk.NoopConfig{})
	require.Len(t, exts, 1)
	assert.Equal(t, "dup", exts[0].Name())
}

func TestGetAllEmpty(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	exts := GetAll(sdk.NoopConfig{})
	assert.Empty(t, exts)
}

func TestGetAllMultiple(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	Register("charlie", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "charlie"}, nil
	})
	Register("alpha", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "alpha"}, nil
	})
	Register("bravo", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "bravo"}, nil
	})

	exts := GetAll(sdk.NoopConfig{})
	require.Len(t, exts, 3)

	assert.Equal(t, "alpha", exts[0].Name())
	assert.Equal(t, "bravo", exts[1].Name())
	assert.Equal(t, "charlie", exts[2].Name())
}

func TestReset(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	Register("temp", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "temp"}, nil
	})

	Reset()

	assert.Empty(t, GetAll(sdk.NoopConfig{}))
}

func TestGetNotRegistered(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := Get("missing", sdk.NoopConfig{})
	require.Error(t, err)
	assert.ErrorIs(t, err, sdk.ErrNotRegistered)
}

func TestList(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	Register("beta", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "beta"}, nil
	})
	Register("alpha", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "alpha"}, nil
	})

	names := List()
	require.Len(t, names, 2)
	assert.Equal(t, "alpha", names[0])
	assert.Equal(t, "beta", names[1])
}

func TestRegistered(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	assert.False(t, Registered("none"))

	Register("exists", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "exists"}, nil
	})

	assert.True(t, Registered("exists"))
	assert.False(t, Registered("missing"))
}

func TestGetAllFactoryErrorSkipped(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	Register("good", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return &stubTUIExtension{name: "good"}, nil
	})
	Register("bad", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (contract.TUIExtension, error) {
		return nil, assert.AnError
	})

	exts := GetAll(sdk.NoopConfig{})
	require.Len(t, exts, 1)
	assert.Equal(t, "good", exts[0].Name())
}
