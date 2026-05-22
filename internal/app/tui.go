package app

import (
	"errors"
	"fmt"
	"os"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/weave-agent/weave/sdk"

	tuibridge "github.com/weave-agent/weave-tui/internal/bridge"
	"github.com/weave-agent/weave-tui/internal/contract"
	tuimodel "github.com/weave-agent/weave-tui/internal/model"
	tuiui "github.com/weave-agent/weave-tui/internal/ui"
)

// ErrNoTTY is returned when stdin is not a terminal.
var ErrNoTTY = errors.New("stdin is not a terminal (use -p for print mode)")

var terminalCheck = defaultTerminalCheck

// TUI is the terminal UI extension runtime.
type TUI struct {
	cfg    sdk.Config
	ps     sdk.PreferenceStore
	tuiCfg contract.TUIConfig

	mu      sync.Mutex
	program *tea.Program
	done    chan struct{}
	ui      *tuiui.TUIImpl
}

// NewExtension creates the SDK extension and registers its UI implementation.
func NewExtension(cfg sdk.Config, ps sdk.PreferenceWriter, tuiCfg contract.TUIConfig) (sdk.Extension, error) {
	t, err := NewTUI(cfg, ps, tuiCfg)
	if err != nil {
		return nil, err
	}

	sdk.RegisterUI("tui", t.ui)

	return t, nil
}

// NewTUI creates a new TUI extension runtime.
// Returns ErrNoTTY if stdin is not a terminal.
func NewTUI(cfg sdk.Config, ps sdk.PreferenceStore, tuiCfg contract.TUIConfig) (*TUI, error) {
	if !terminalCheck() {
		return nil, ErrNoTTY
	}

	ui := tuiui.NewTUIImpl(nil, nil)

	return &TUI{
		cfg:    cfg,
		ps:     ps,
		done:   make(chan struct{}),
		ui:     ui,
		tuiCfg: tuiCfg,
	}, nil
}

// Name returns the extension name.
func (t *TUI) Name() string { return "tui" }

// Subscribe starts the Bubble Tea program in a goroutine, blocking until it exits.
// The bridge goroutine translates bus events into tea.Msg and forwards them.
func (t *TUI) Subscribe(bus sdk.Bus) error {
	events := make(chan sdk.Event, 256)

	var eventsMu sync.Mutex

	eventsClosed := false

	bus.OnAll(func(ev sdk.Event) error {
		eventsMu.Lock()
		if eventsClosed {
			eventsMu.Unlock()
			return nil
		}

		select {
		case events <- ev:
		default:
			sdk.Logger("tui").Warn("dropped event, channel full", "topic", ev.Topic)
		}

		eventsMu.Unlock()

		return nil
	})

	model := tuimodel.NewModelWithConfig(bus, t.cfg, t.ps, t.ui, t.tuiCfg)

	t.mu.Lock()
	t.program = tea.NewProgram(model)
	t.mu.Unlock()

	// Register UI extensions before setting the program so that
	// SetStatus calls during registration are buffered (not sent).
	t.wireUIExtensions(bus)

	// Wire the UI implementation to the program.
	t.ui.SetProgram(t.program)

	go tuibridge.Bridge(t.program, events)

	_, err := t.program.Run()

	t.ui.Close()

	eventsMu.Lock()
	eventsClosed = true

	close(events)
	eventsMu.Unlock()

	endPayload := any(nil)
	if err != nil {
		endPayload = fmt.Sprintf("tui error: %v", err)
	}

	// Signal shutdown so the launcher's select (waiting on agent.end)
	// can unblock and proceed to wired.Close().
	bus.Publish(sdk.NewEvent(tuibridge.TopicEnd, endPayload))

	close(t.done)

	return nil
}

// wireUIExtensions registers all UI extensions with the TUI's UI implementation.
// Extensions that implement UIExtensionWithBus also receive the event bus.
func (t *TUI) wireUIExtensions(bus sdk.Bus) {
	for _, ext := range sdk.GetUIExtensions(t.cfg) {
		ext.Register(t.ui)

		if withBus, ok := ext.(sdk.UIExtensionWithBus); ok {
			withBus.RegisterWithBus(t.ui, bus)
		}
	}
}

// Close sends a quit message to the Bubble Tea program and waits for it to finish.
func (t *TUI) Close() error {
	t.mu.Lock()
	p := t.program
	t.mu.Unlock()

	if p != nil {
		p.Quit()
		<-t.done
	}

	return nil
}

// defaultTerminalCheck checks whether stdin is connected to a terminal.
func defaultTerminalCheck() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return fi.Mode()&os.ModeCharDevice != 0
}
