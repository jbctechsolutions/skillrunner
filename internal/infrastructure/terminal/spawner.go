// Package terminal provides terminal spawning and management functionality.
package terminal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// TerminalType represents the type of terminal emulator.
type TerminalType string

const (
	TerminalITerm2    TerminalType = "iterm2"
	TerminalApp       TerminalType = "terminal"
	TerminalTmux      TerminalType = "tmux"
	TerminalKitty     TerminalType = "kitty"
	TerminalAlacritty TerminalType = "alacritty"
	TerminalGnome     TerminalType = "gnome-terminal"
	TerminalAuto      TerminalType = "auto"
)

// Spawner manages terminal spawning.
type Spawner struct {
	terminalType TerminalType
}

// NewSpawner creates a new terminal spawner.
func NewSpawner(terminalType TerminalType) *Spawner {
	return &Spawner{
		terminalType: terminalType,
	}
}

// SpawnOptions contains options for spawning a terminal.
type SpawnOptions struct {
	WorkingDir string   // Working directory
	Command    string   // Command to run
	Args       []string // Command arguments
	Title      string   // Window title
	Background bool     // Run in background
}

// Spawn spawns a new terminal window.
func (s *Spawner) Spawn(ctx context.Context, opts SpawnOptions) error {
	// Detect terminal type if auto
	termType := s.terminalType
	if termType == TerminalAuto {
		detected, err := s.detectTerminal()
		if err != nil {
			return fmt.Errorf("failed to detect terminal: %w", err)
		}
		termType = detected
	}

	// Spawn based on terminal type
	switch termType {
	case TerminalITerm2:
		return s.spawnITerm2(ctx, opts)
	case TerminalApp:
		return s.spawnTerminalApp(ctx, opts)
	case TerminalTmux:
		return s.spawnTmux(ctx, opts)
	case TerminalKitty:
		return s.spawnKitty(ctx, opts)
	case TerminalAlacritty:
		return s.spawnAlacritty(ctx, opts)
	case TerminalGnome:
		return s.spawnGnomeTerminal(ctx, opts)
	default:
		return fmt.Errorf("unsupported terminal type: %s", termType)
	}
}

// detectTerminal detects the available terminal emulator.
func (s *Spawner) detectTerminal() (TerminalType, error) {
	// Check if running in tmux
	if os.Getenv("TMUX") != "" {
		return TerminalTmux, nil
	}

	// Platform-specific detection
	switch runtime.GOOS {
	case "darwin":
		// macOS: Check for iTerm2, then Terminal.app
		if s.isITerm2Available() {
			return TerminalITerm2, nil
		}
		return TerminalApp, nil

	case "linux":
		// Linux: Check for various terminals
		if s.isCommandAvailable("gnome-terminal") {
			return TerminalGnome, nil
		}
		if s.isCommandAvailable("kitty") {
			return TerminalKitty, nil
		}
		if s.isCommandAvailable("alacritty") {
			return TerminalAlacritty, nil
		}
		return "", fmt.Errorf("no supported terminal found")

	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// spawnITerm2 spawns a new iTerm2 window.
func (s *Spawner) spawnITerm2(ctx context.Context, opts SpawnOptions) error {
	// Build AppleScript
	script := fmt.Sprintf(`
tell application "iTerm"
	create window with default profile
	tell current session of current window
		write text "cd %s"
		write text "%s"
	end tell
end tell
`, s.escapeShell(opts.WorkingDir), s.buildCommand(opts))

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	return cmd.Run()
}

// spawnTerminalApp spawns a new Terminal.app window.
func (s *Spawner) spawnTerminalApp(ctx context.Context, opts SpawnOptions) error {
	// Build AppleScript
	script := fmt.Sprintf(`
tell application "Terminal"
	do script "cd %s && %s"
	activate
end tell
`, s.escapeShell(opts.WorkingDir), s.buildCommand(opts))

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	return cmd.Run()
}

// spawnTmux spawns a new tmux window.
func (s *Spawner) spawnTmux(ctx context.Context, opts SpawnOptions) error {
	// Create new tmux window
	args := []string{"new-window"}

	if opts.WorkingDir != "" {
		args = append(args, "-c", opts.WorkingDir)
	}

	if opts.Title != "" {
		args = append(args, "-n", opts.Title)
	}

	command := s.buildCommand(opts)
	if command != "" {
		args = append(args, command)
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	return cmd.Run()
}

// spawnKitty spawns a new Kitty terminal.
func (s *Spawner) spawnKitty(ctx context.Context, opts SpawnOptions) error {
	args := []string{}

	if opts.WorkingDir != "" {
		args = append(args, "--directory", opts.WorkingDir)
	}

	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	command := s.buildCommand(opts)
	if command != "" {
		args = append(args, "sh", "-c", command)
	}

	cmd := exec.CommandContext(ctx, "kitty", args...)
	if opts.Background {
		cmd.Start()
		return nil
	}
	return cmd.Run()
}

// spawnAlacritty spawns a new Alacritty terminal.
func (s *Spawner) spawnAlacritty(ctx context.Context, opts SpawnOptions) error {
	args := []string{}

	if opts.WorkingDir != "" {
		args = append(args, "--working-directory", opts.WorkingDir)
	}

	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	command := s.buildCommand(opts)
	if command != "" {
		args = append(args, "-e", "sh", "-c", command)
	}

	cmd := exec.CommandContext(ctx, "alacritty", args...)
	if opts.Background {
		cmd.Start()
		return nil
	}
	return cmd.Run()
}

// spawnGnomeTerminal spawns a new GNOME Terminal.
func (s *Spawner) spawnGnomeTerminal(ctx context.Context, opts SpawnOptions) error {
	args := []string{}

	if opts.WorkingDir != "" {
		args = append(args, "--working-directory", opts.WorkingDir)
	}

	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	command := s.buildCommand(opts)
	if command != "" {
		args = append(args, "--", "sh", "-c", command)
	}

	cmd := exec.CommandContext(ctx, "gnome-terminal", args...)
	if opts.Background {
		cmd.Start()
		return nil
	}
	return cmd.Run()
}

// buildCommand builds the command string from options.
func (s *Spawner) buildCommand(opts SpawnOptions) string {
	if opts.Command == "" {
		return ""
	}

	parts := []string{opts.Command}
	parts = append(parts, opts.Args...)
	return strings.Join(parts, " ")
}

// escapeShell escapes a string for shell use.
func (s *Spawner) escapeShell(str string) string {
	// Simple escaping - replace single quotes
	return strings.ReplaceAll(str, "'", "'\\''")
}

// isITerm2Available checks if iTerm2 is available on macOS.
func (s *Spawner) isITerm2Available() bool {
	_, err := os.Stat("/Applications/iTerm.app")
	return err == nil
}

// isCommandAvailable checks if a command is available in PATH.
func (s *Spawner) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// GetAvailableTerminals returns a list of available terminal emulators.
func (s *Spawner) GetAvailableTerminals() []TerminalType {
	terminals := []TerminalType{}

	// Check tmux
	if os.Getenv("TMUX") != "" {
		terminals = append(terminals, TerminalTmux)
	}

	// Platform-specific checks
	switch runtime.GOOS {
	case "darwin":
		if s.isITerm2Available() {
			terminals = append(terminals, TerminalITerm2)
		}
		terminals = append(terminals, TerminalApp)

	case "linux":
		if s.isCommandAvailable("gnome-terminal") {
			terminals = append(terminals, TerminalGnome)
		}
		if s.isCommandAvailable("kitty") {
			terminals = append(terminals, TerminalKitty)
		}
		if s.isCommandAvailable("alacritty") {
			terminals = append(terminals, TerminalAlacritty)
		}
	}

	return terminals
}
