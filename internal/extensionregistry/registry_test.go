package extensionregistry

import (
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
