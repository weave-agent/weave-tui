package app

import (
	"testing"

	"github.com/weave-agent/weave/sdk"

	"github.com/weave-agent/weave-tui/internal/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func forceTerminal(t *testing.T, ok bool) {
	t.Helper()

	old := terminalCheck
	terminalCheck = func() bool { return ok }

	t.Cleanup(func() {
		terminalCheck = old
	})
}

func newTestTUI(t *testing.T) *TUI {
	t.Helper()

	forceTerminal(t, true)

	tui, err := NewTUI(nil, nil, contract.TUIConfig{})
	require.NoError(t, err)

	return tui
}

func TestNewTUIRequiresTTY(t *testing.T) {
	forceTerminal(t, false)

	tui, err := NewTUI(nil, nil, contract.TUIConfig{})

	require.ErrorIs(t, err, ErrNoTTY)
	assert.Nil(t, tui)
	assert.Contains(t, ErrNoTTY.Error(), "stdin")
}

func TestNewExtensionRegistersUI(t *testing.T) {
	forceTerminal(t, true)
	sdk.ResetUIRegistry()
	t.Cleanup(sdk.ResetUIRegistry)

	ext, err := NewExtension(nil, nil, contract.TUIConfig{})
	require.NoError(t, err)
	assert.Equal(t, "tui", ext.Name())
	require.IsType(t, &TUI{}, ext)

	registeredUI, err := sdk.GetUI("tui")
	require.NoError(t, err)
	assert.Same(t, ext.(*TUI).ui, registeredUI)
}

func TestTUIName(t *testing.T) {
	tui := newTestTUI(t)

	assert.Equal(t, "tui", tui.Name())
}

func TestTUICloseWithoutSubscribe(t *testing.T) {
	tui := newTestTUI(t)

	require.NoError(t, tui.Close())
}

// mockUIExtension records whether Register was called and with what UI.
type mockUIExtension struct {
	name           string
	registerCalled bool
	registeredUI   sdk.UI
}

func (m *mockUIExtension) Name() string { return m.name }
func (m *mockUIExtension) Register(ui sdk.UI) {
	m.registerCalled = true
	m.registeredUI = ui
}

func TestTUIWireUIExtensions(t *testing.T) {
	sdk.ResetUIExtensionRegistry()
	defer sdk.ResetUIExtensionRegistry()

	ext := &mockUIExtension{name: "test-ext"}

	sdk.RegisterUIExtension("test-ext", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (sdk.UIExtension, error) {
		return ext, nil
	})

	tui := newTestTUI(t)

	tui.wireUIExtensions(nil)

	assert.True(t, ext.registerCalled, "expected Register to be called on UI extension")
	assert.Equal(t, tui.ui, ext.registeredUI, "expected UI extension to receive TUI's UI implementation")
}

func TestTUIWireUIExtensionsMultiple(t *testing.T) {
	sdk.ResetUIExtensionRegistry()
	defer sdk.ResetUIExtensionRegistry()

	ext1 := &mockUIExtension{name: "ext-one"}
	ext2 := &mockUIExtension{name: "ext-two"}

	sdk.RegisterUIExtension("ext-one", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (sdk.UIExtension, error) {
		return ext1, nil
	})
	sdk.RegisterUIExtension("ext-two", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (sdk.UIExtension, error) {
		return ext2, nil
	})

	tui := newTestTUI(t)

	tui.wireUIExtensions(nil)

	assert.True(t, ext1.registerCalled, "expected Register to be called on ext-one")
	assert.True(t, ext2.registerCalled, "expected Register to be called on ext-two")
	assert.Equal(t, tui.ui, ext1.registeredUI)
	assert.Equal(t, tui.ui, ext2.registeredUI)
}

func TestTUIWireUIExtensionsEmptyRegistry(t *testing.T) {
	sdk.ResetUIExtensionRegistry()
	defer sdk.ResetUIExtensionRegistry()

	tui := newTestTUI(t)

	assert.NotPanics(t, func() {
		tui.wireUIExtensions(nil)
	})
}

// mockUIExtensionWithBus records whether RegisterWithBus was called.
type mockUIExtensionWithBus struct {
	name             string
	registerCalled   bool
	registeredUI     sdk.UI
	busCalled        bool
	registeredUIWith sdk.UI
	registeredBus    sdk.Bus
}

// mockBus is a minimal sdk.Bus implementation for tests.
type mockBus struct{}

func (m *mockBus) Publish(_ sdk.Event)        {}
func (m *mockBus) On(_ string, _ sdk.Handler) {}
func (m *mockBus) OnAll(_ sdk.Handler)        {}
func (m *mockBus) Off(_ sdk.Handler)          {}
func (m *mockBus) Close() error               { return nil }

func (m *mockUIExtensionWithBus) Name() string { return m.name }
func (m *mockUIExtensionWithBus) Register(ui sdk.UI) {
	m.registerCalled = true
	m.registeredUI = ui
}

func (m *mockUIExtensionWithBus) RegisterWithBus(ui sdk.UI, bus sdk.Bus) {
	m.busCalled = true
	m.registeredUIWith = ui
	m.registeredBus = bus
}

func TestTUIWireUIExtensionsWithBus(t *testing.T) {
	sdk.ResetUIExtensionRegistry()
	defer sdk.ResetUIExtensionRegistry()

	ext := &mockUIExtensionWithBus{name: "bus-ext"}

	sdk.RegisterUIExtension("bus-ext", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (sdk.UIExtension, error) {
		return ext, nil
	})

	tui := newTestTUI(t)

	bus := &mockBus{}
	tui.wireUIExtensions(bus)

	assert.True(t, ext.registerCalled, "expected Register to be called")
	assert.True(t, ext.busCalled, "expected RegisterWithBus to be called")
	assert.Equal(t, tui.ui, ext.registeredUIWith)
	assert.Equal(t, bus, ext.registeredBus)
}

func TestTUIWireUIExtensionsPlainExtensionNoBus(t *testing.T) {
	sdk.ResetUIExtensionRegistry()
	defer sdk.ResetUIExtensionRegistry()

	ext := &mockUIExtension{name: "plain-ext"}

	sdk.RegisterUIExtension("plain-ext", func(_ sdk.Config, _ sdk.PreferenceReader, _ struct{}) (sdk.UIExtension, error) {
		return ext, nil
	})

	tui := newTestTUI(t)

	assert.NotPanics(t, func() {
		tui.wireUIExtensions(&mockBus{})
	})

	assert.True(t, ext.registerCalled)
}
