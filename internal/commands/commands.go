package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/weave-agent/weave/sdk"

	tuibridge "github.com/weave-agent/weave-tui/internal/bridge"
	tuievents "github.com/weave-agent/weave-tui/internal/events"

	tea "charm.land/bubbletea/v2"
)

// CommandResult is returned by a command handler to signal side effects.
type CommandResult struct {
	// Quit exits the TUI.
	Quit bool
	// ClearChat clears all chat messages.
	ClearChat bool
	// Prompt resets prompt state so next submit sends agent.prompt.
	ResetPrompt bool
	// Notify shows a message in chat (info).
	Notify string
	// ShowKeybindings opens the keybindings help overlay.
	ShowKeybindings bool
	// Command publishes a tea.Cmd.
	Command tea.Cmd
}

// CommandHandler processes a slash command with its arguments.
type CommandHandler func(args string) CommandResult

// CommandInfo describes a registered slash command.
type CommandInfo struct {
	Name         string
	Description  string
	Handler      CommandHandler
	AcceptsFiles bool
}

// CommandRegistry manages slash commands.
type CommandRegistry struct {
	mu         sync.Mutex
	commands   map[string]CommandInfo
	sessionDir string
	runtime    RuntimeCommands
}

// RuntimeCommands supplies command implementations that still live in
// other internalization targets during the staged package reorg.
type RuntimeCommands struct {
	ListSessions func(sessionDir string) tea.Cmd
	Login        func() tea.Cmd
	Logout       func() tea.Cmd
}

// NewCommandRegistry creates a registry with built-in commands.
func NewCommandRegistry(bus sdk.Bus, sessionDir string, runtime ...RuntimeCommands) *CommandRegistry {
	r := &CommandRegistry{
		commands:   make(map[string]CommandInfo),
		sessionDir: sessionDir,
	}

	if len(runtime) > 0 {
		r.runtime = runtime[0]
	}

	r.register("/new", "Start a new conversation", false, func(_ string) CommandResult {
		return CommandResult{ClearChat: true, ResetPrompt: true}
	})

	r.register("/clear", "Start a new conversation (alias for /new)", false, func(_ string) CommandResult {
		return CommandResult{ClearChat: true, ResetPrompt: true}
	})

	r.register("/quit", "Exit weave", false, func(_ string) CommandResult {
		return CommandResult{Quit: true}
	})

	r.register("/help", "Show available commands", false, func(_ string) CommandResult {
		return CommandResult{Notify: r.helpText()}
	})

	r.register("/keybind-help", "Show keybindings help", false, func(_ string) CommandResult {
		return CommandResult{ShowKeybindings: true}
	})

	r.register("/compact", "Compact conversation history", false, func(args string) CommandResult {
		payload := "compact"
		if args != "" {
			payload = "compact " + args
		}

		return CommandResult{Command: tuibridge.PublishSteer(bus, payload)}
	})

	r.register("/name", "Set conversation name", false, func(args string) CommandResult {
		return CommandResult{Command: tuibridge.PublishSteer(bus, "name "+args)}
	})

	r.register("/resume", "View a previous session", false, func(_ string) CommandResult {
		return CommandResult{Command: r.listSessionsCmd()}
	})

	r.register("/reload", "Rebuild extensions and restart", false, func(_ string) CommandResult {
		return CommandResult{Command: reloadCmd()}
	})

	r.register("/login", "Authenticate with a provider", false, func(_ string) CommandResult {
		return CommandResult{Command: r.loginCmd()}
	})

	r.register("/logout", "Clear authentication for a provider", false, func(_ string) CommandResult {
		return CommandResult{Command: r.logoutCmd()}
	})

	return r
}

func (r *CommandRegistry) register(name, description string, acceptsFiles bool, handler CommandHandler) {
	r.commands[name] = CommandInfo{
		Name:         name,
		Description:  description,
		Handler:      handler,
		AcceptsFiles: acceptsFiles,
	}
}

// Register adds a command to the registry. Not safe for concurrent use with Dispatch.
func (r *CommandRegistry) Register(name, description string, acceptsFiles bool, handler CommandHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.register(name, description, acceptsFiles, handler)
}

// Dispatch parses input and, if it starts with /, runs the matching command.
// Returns (true, result) if handled as a command, (false, zero) otherwise.
func (r *CommandRegistry) Dispatch(input string) (bool, CommandResult) {
	if !strings.HasPrefix(input, "/") {
		return false, CommandResult{}
	}

	name, args := parseCommand(input)

	r.mu.Lock()
	cmd, ok := r.commands[name]
	r.mu.Unlock()

	if !ok {
		return true, CommandResult{
			Notify: "unknown command: " + name,
		}
	}

	return true, cmd.Handler(args)
}

// Names returns sorted command names for autocomplete.
func (r *CommandRegistry) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	names := make([]string, 0, len(r.commands))
	for k := range r.commands {
		names = append(names, k)
	}

	sort.Strings(names)

	return names
}

// Lookup returns command info by name.
func (r *CommandRegistry) Lookup(name string) (CommandInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd, ok := r.commands[name]

	return cmd, ok
}

func (r *CommandRegistry) helpText() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	names := make([]string, 0, len(r.commands))
	for k := range r.commands {
		names = append(names, k)
	}

	sort.Strings(names)

	var b strings.Builder
	b.WriteString("Available commands:\n")

	for _, name := range names {
		info := r.commands[name]
		fmt.Fprintf(&b, "  %-12s %s\n", name, info.Description)
	}

	return b.String()
}

func (r *CommandRegistry) listSessionsCmd() tea.Cmd {
	if r.runtime.ListSessions == nil {
		return unavailableCmd("/resume")
	}

	return r.runtime.ListSessions(r.sessionDir)
}

func (r *CommandRegistry) loginCmd() tea.Cmd {
	if r.runtime.Login == nil {
		return unavailableCmd("/login")
	}

	return r.runtime.Login()
}

func (r *CommandRegistry) logoutCmd() tea.Cmd {
	if r.runtime.Logout == nil {
		return unavailableCmd("/logout")
	}

	return r.runtime.Logout()
}

func unavailableCmd(name string) tea.Cmd {
	return func() tea.Msg {
		return tuievents.NotifyMsg{Message: name + ": not available"}
	}
}

// ParseCommand splits "/name arg1 arg2" into ("/name", "arg1 arg2").
func ParseCommand(input string) (string, string) {
	return parseCommand(input)
}

// parseCommand splits "/name arg1 arg2" into ("/name", "arg1 arg2").
func parseCommand(input string) (string, string) {
	input = strings.TrimSpace(input)
	name, args, _ := strings.Cut(input, " ")

	return name, strings.TrimSpace(args)
}

// ReloadMsg is a tea.Msg that signals the program should reload.
type ReloadMsg struct {
	launcherPath string
	origArgs     []string
	env          []string
}

// reloadCmd returns a tea.Cmd that reads reload env vars and returns a reloadMsg.
// If the env vars are not set (e.g., not launched via the weave launcher), it
// returns a NotifyMsg instead.
func reloadCmd() tea.Cmd {
	return func() tea.Msg {
		launcherPath := os.Getenv("WEAVE_LAUNCHER_PATH")
		buildHash := os.Getenv("WEAVE_BUILD_HASH")
		origArgsJSON := os.Getenv("WEAVE_ORIG_ARGS")

		if launcherPath == "" {
			return tuievents.NotifyMsg{Message: "/reload: not available — weave was not launched via the launcher"}
		}

		// Remove the cache directory for the current build hash. The hash is
		// validated as a SHA-256 hex string before being joined into the path
		// so a malicious value (e.g. "../../victim") cannot escape the cache
		// root and trick os.RemoveAll into deleting unrelated files.
		if buildHash != "" {
			if !isSHA256Hex(buildHash) {
				return tuievents.NotifyMsg{Message: fmt.Sprintf("/reload: invalid build hash %q", buildHash)}
			}

			home, _ := os.UserHomeDir()
			if home != "" {
				cacheDir := filepath.Join(home, ".weave", "bin", buildHash)
				if err := os.RemoveAll(cacheDir); err != nil { //nolint:gosec // G703 — cleaning our own cache dir
					return tuievents.NotifyMsg{Message: fmt.Sprintf("/reload: failed to remove cache: %v", err)}
				}
			}
		}

		var origArgs []string
		if origArgsJSON != "" {
			_ = json.Unmarshal([]byte(origArgsJSON), &origArgs)
		}

		if len(origArgs) == 0 {
			origArgs = []string{launcherPath}
		}

		return ReloadMsg{
			launcherPath: launcherPath,
			origArgs:     origArgs,
			env:          os.Environ(),
		}
	}
}

// HandleReload performs the actual syscall.Exec to replace the process.
func HandleReload(msg ReloadMsg) error {
	return fmt.Errorf("exec: %w", syscall.Exec(msg.launcherPath, msg.origArgs, msg.env))
}

// isSHA256Hex reports whether s is a 64-character lowercase hexadecimal string,
// matching the format produced by ComputeHash. Used to reject path-traversal
// values from WEAVE_BUILD_HASH before joining into a filesystem path.
func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}

	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}

	return true
}
