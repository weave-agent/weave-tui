package app

import (
	"errors"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/weave-agent/weave/sdk"

	tuibridge "github.com/weave-agent/weave-tui/internal/bridge"
	"github.com/weave-agent/weave-tui/internal/contract"
	tuievents "github.com/weave-agent/weave-tui/internal/events"

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

type fakeProgram struct {
	mu         sync.Mutex
	quitOnce   sync.Once
	block      bool
	runErr     error
	quit       chan struct{}
	runStarted chan struct{}
	sent       []tea.Msg
}

func newFakeProgram(block bool, runErr error) *fakeProgram {
	return &fakeProgram{
		block:      block,
		runErr:     runErr,
		quit:       make(chan struct{}),
		runStarted: make(chan struct{}),
	}
}

func (p *fakeProgram) Run() (tea.Model, error) {
	close(p.runStarted)

	if p.block {
		<-p.quit
	}

	return nil, p.runErr
}

func (p *fakeProgram) Quit() {
	p.quitOnce.Do(func() {
		close(p.quit)
	})
}

func (p *fakeProgram) Send(msg tea.Msg) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.sent = append(p.sent, msg)
}

func (p *fakeProgram) hasTurnStart(turn int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, msg := range p.sent {
		if turnStart, ok := msg.(tuievents.TurnStartMsg); ok && turnStart.Turn == turn {
			return true
		}
	}

	return false
}

type recordingBus struct {
	mu        sync.Mutex
	handlers  []sdk.Handler
	published []sdk.Event
}

func (b *recordingBus) Publish(ev sdk.Event) {
	b.mu.Lock()
	b.published = append(b.published, ev)
	handlers := append([]sdk.Handler(nil), b.handlers...)
	b.mu.Unlock()

	for _, handler := range handlers {
		_ = handler(ev)
	}
}

func (b *recordingBus) On(_ string, _ sdk.Handler) {}

func (b *recordingBus) OnAll(handler sdk.Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers = append(b.handlers, handler)
}

func (b *recordingBus) Off(_ sdk.Handler) {}

func (b *recordingBus) Close() error { return nil }

func (b *recordingBus) publishedEvent(topic string) (sdk.Event, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ev := range b.published {
		if ev.Topic == topic {
			return ev, true
		}
	}

	return sdk.Event{}, false
}

func withFakeProgram(t *testing.T, program *fakeProgram) {
	t.Helper()

	old := newProgram
	newProgram = func(tea.Model) appProgram {
		return program
	}

	t.Cleanup(func() {
		newProgram = old
	})
}

func requireEventually(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.After(2 * time.Second)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-deadline:
			require.FailNow(t, "condition was not met before timeout")
		case <-ticker.C:
		}
	}
}

func TestTUISubscribeForwardsEventsAndClosePublishesEnd(t *testing.T) {
	program := newFakeProgram(true, nil)
	withFakeProgram(t, program)

	tui := newTestTUI(t)
	bus := &recordingBus{}

	errCh := make(chan error, 1)
	go func() {
		errCh <- tui.Subscribe(bus)
	}()

	<-program.runStarted
	bus.Publish(sdk.NewEvent(tuibridge.TopicTurnStart, 7))

	requireEventually(t, func() bool {
		return program.hasTurnStart(7)
	})

	closeErrCh := make(chan error, 1)
	go func() {
		closeErrCh <- tui.Close()
	}()

	select {
	case err := <-closeErrCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "Close did not return")
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "Subscribe did not return")
	}

	endEvent, ok := bus.publishedEvent(tuibridge.TopicEnd)
	require.True(t, ok)
	assert.Nil(t, endEvent.Payload)

	bus.Publish(sdk.NewEvent(tuibridge.TopicTurnStart, 8))
	time.Sleep(20 * time.Millisecond)
	assert.False(t, program.hasTurnStart(8))
}

func TestTUISubscribePublishesRunErrorPayloadAndClosesUI(t *testing.T) {
	program := newFakeProgram(false, errors.New("boom"))
	withFakeProgram(t, program)

	tui := newTestTUI(t)
	bus := &recordingBus{}

	require.NoError(t, tui.Subscribe(bus))

	endEvent, ok := bus.publishedEvent(tuibridge.TopicEnd)
	require.True(t, ok)
	assert.Equal(t, "tui error: boom", endEvent.Payload)

	_, err := tui.ui.Select("after close", []string{"one"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shutting down")
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
