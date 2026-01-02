// Package process provides process and terminal management utilities.
package process

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// TmuxManager manages tmux sessions for backends.
type TmuxManager struct {
	tmuxPath string
}

// NewTmuxManager creates a new tmux manager.
func NewTmuxManager() (*TmuxManager, error) {
	// Check if tmux is available
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found in PATH: %w", err)
	}

	return &TmuxManager{
		tmuxPath: path,
	}, nil
}

// IsAvailable checks if tmux is available on the system.
func (tm *TmuxManager) IsAvailable() bool {
	return tm.tmuxPath != ""
}

// CreateSession creates a new tmux session.
func (tm *TmuxManager) CreateSession(ctx context.Context, sessionName, workDir string) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	// Create detached session with working directory
	args := []string{"new-session", "-d", "-s", sessionName}
	if workDir != "" {
		args = append(args, "-c", workDir)
	}

	cmd := exec.CommandContext(ctx, tm.tmuxPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	return nil
}

// AttachSession attaches to an existing tmux session.
func (tm *TmuxManager) AttachSession(ctx context.Context, sessionName string) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	cmd := exec.CommandContext(ctx, tm.tmuxPath, "attach-session", "-t", sessionName)
	cmd.Stdin = nil // Will be handled by terminal
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to attach to tmux session: %w", err)
	}

	// Don't wait for command to complete - it's interactive
	return nil
}

// DetachSession detaches from a tmux session.
func (tm *TmuxManager) DetachSession(ctx context.Context, sessionName string) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	cmd := exec.CommandContext(ctx, tm.tmuxPath, "detach-client", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to detach from tmux session: %w", err)
	}

	return nil
}

// KillSession terminates a tmux session.
func (tm *TmuxManager) KillSession(ctx context.Context, sessionName string) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	cmd := exec.CommandContext(ctx, tm.tmuxPath, "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		// Check if session doesn't exist
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(err.Error(), "can't find session") {
			return nil // Session already gone
		}
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}

	return nil
}

// SendKeys sends keystrokes to a tmux session.
func (tm *TmuxManager) SendKeys(ctx context.Context, sessionName, keys string) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	cmd := exec.CommandContext(ctx, tm.tmuxPath, "send-keys", "-t", sessionName, keys, "Enter")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send keys to tmux session: %w", err)
	}

	return nil
}

// CapturePane captures the visible content of a tmux pane.
func (tm *TmuxManager) CapturePane(ctx context.Context, sessionName string, lines int) ([]string, error) {
	if sessionName == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}

	args := []string{"capture-pane", "-t", sessionName, "-p"}
	if lines > 0 {
		args = append(args, "-S", fmt.Sprintf("-%d", lines))
	}

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, tm.tmuxPath, args...)
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to capture tmux pane: %w", err)
	}

	// Split output into lines
	output := strings.TrimSpace(out.String())
	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// SessionExists checks if a tmux session exists.
func (tm *TmuxManager) SessionExists(ctx context.Context, sessionName string) (bool, error) {
	if sessionName == "" {
		return false, fmt.Errorf("session name cannot be empty")
	}

	cmd := exec.CommandContext(ctx, tm.tmuxPath, "has-session", "-t", sessionName)
	err := cmd.Run()
	if err != nil {
		// Check if it's just "session not found" vs actual error
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil // Session doesn't exist
			}
		}
		return false, fmt.Errorf("failed to check tmux session: %w", err)
	}

	return true, nil
}

// ListSessions returns all tmux session names.
func (tm *TmuxManager) ListSessions(ctx context.Context) ([]string, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, tm.tmuxPath, "list-sessions", "-F", "#{session_name}")
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		// No sessions is not an error
		if strings.Contains(err.Error(), "no server running") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// GetSessionPID returns the process ID of a tmux session.
func (tm *TmuxManager) GetSessionPID(ctx context.Context, sessionName string) (int, error) {
	if sessionName == "" {
		return 0, fmt.Errorf("session name cannot be empty")
	}

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, tm.tmuxPath, "list-panes", "-t", sessionName, "-F", "#{pane_pid}")
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to get tmux session PID: %w", err)
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		return 0, fmt.Errorf("no PID found for session")
	}

	var pid int
	_, err := fmt.Sscanf(output, "%d", &pid)
	if err != nil {
		return 0, fmt.Errorf("failed to parse PID: %w", err)
	}

	return pid, nil
}

// ResizePane resizes a tmux pane.
func (tm *TmuxManager) ResizePane(ctx context.Context, sessionName string, width, height int) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	args := []string{"resize-pane", "-t", sessionName}
	if width > 0 {
		args = append(args, "-x", fmt.Sprintf("%d", width))
	}
	if height > 0 {
		args = append(args, "-y", fmt.Sprintf("%d", height))
	}

	cmd := exec.CommandContext(ctx, tm.tmuxPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to resize tmux pane: %w", err)
	}

	return nil
}

// SetEnvironment sets an environment variable in a tmux session.
func (tm *TmuxManager) SetEnvironment(ctx context.Context, sessionName, key, value string) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	cmd := exec.CommandContext(ctx, tm.tmuxPath, "set-environment", "-t", sessionName, key, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set tmux environment: %w", err)
	}

	return nil
}
