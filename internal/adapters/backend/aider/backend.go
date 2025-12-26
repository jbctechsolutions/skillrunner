// Package aider provides the Aider backend implementation.
package aider

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/process"
)

// Backend implements the BackendPort interface for Aider.
type Backend struct {
	tmux       *process.TmuxManager
	executable string
	sessionMap map[string]*session.Session
	machineID  string
}

// NewBackend creates a new Aider backend.
func NewBackend(machineID string) (*Backend, error) {
	// Check if aider is available
	executable, err := exec.LookPath("aider")
	if err != nil {
		return nil, fmt.Errorf("aider not found in PATH: %w", err)
	}

	// Initialize tmux manager
	tmux, err := process.NewTmuxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tmux: %w", err)
	}

	return &Backend{
		tmux:       tmux,
		executable: executable,
		sessionMap: make(map[string]*session.Session),
		machineID:  machineID,
	}, nil
}

// Info returns metadata about the Aider backend.
func (b *Backend) Info() ports.BackendInfo {
	return ports.BackendInfo{
		Name:        "aider",
		Version:     "0.1.0", // TODO: Get actual version from aider --version
		Description: "AI pair programming in your terminal",
		Executable:  b.executable,
		Features:    []string{"model-control", "auto-commit", "test-driven", "repo-map"},
	}
}

// Start creates and starts a new Aider session.
func (b *Backend) Start(ctx context.Context, workspace string, config session.BackendConfig) (*session.Session, error) {
	// Generate session ID
	sessionID := generateSessionID()
	tmuxSession := fmt.Sprintf("aider-%s", sessionID)

	// Create tmux session
	if err := b.tmux.CreateSession(ctx, tmuxSession, workspace); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Build aider command
	cmd := b.buildCommand(workspace, config)

	// Send command to tmux session
	if err := b.tmux.SendKeys(ctx, tmuxSession, cmd); err != nil {
		_ = b.tmux.KillSession(ctx, tmuxSession) // Best-effort cleanup
		return nil, fmt.Errorf("failed to start aider: %w", err)
	}

	// Get process ID
	pid, err := b.tmux.GetSessionPID(ctx, tmuxSession)
	if err != nil {
		// Not critical, continue without PID
		pid = 0
	}

	// Create session object
	sess := &session.Session{
		ID:          sessionID,
		WorkspaceID: filepath.Base(workspace),
		Backend:     "aider",
		Model:       config.AiderWeakModel,
		Status:      session.StatusActive,
		StartedAt:   time.Now(),
		MachineID:   b.machineID,
		ProcessID:   pid,
		TmuxSession: tmuxSession,
		Metadata: map[string]string{
			"edit_format": config.AiderEditFormat,
			"auto_commit": fmt.Sprintf("%t", config.AiderAutoCommit),
		},
		Context: &session.Context{
			WorkingDirectory: workspace,
		},
	}

	b.sessionMap[sessionID] = sess
	return sess, nil
}

// Attach connects to an existing Aider session.
func (b *Backend) Attach(ctx context.Context, sessionID string) error {
	sess, exists := b.sessionMap[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return b.tmux.AttachSession(ctx, sess.TmuxSession)
}

// Detach disconnects from a session without killing it.
func (b *Backend) Detach(ctx context.Context) error {
	// Detach from current tmux session
	return b.tmux.DetachSession(ctx, "")
}

// Kill terminates an Aider session.
func (b *Backend) Kill(ctx context.Context, sessionID string) error {
	sess, exists := b.sessionMap[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Kill tmux session
	if err := b.tmux.KillSession(ctx, sess.TmuxSession); err != nil {
		return err
	}

	// Update session status
	now := time.Now()
	sess.Status = session.StatusKilled
	sess.EndedAt = &now

	delete(b.sessionMap, sessionID)
	return nil
}

// InjectContext injects contextual information into a running session.
func (b *Backend) InjectContext(ctx context.Context, sessionID, content string) error {
	sess, exists := b.sessionMap[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Send content as a message to aider
	return b.tmux.SendKeys(ctx, sess.TmuxSession, content)
}

// InjectFile injects a file into a running session.
func (b *Backend) InjectFile(ctx context.Context, sessionID, path string) error {
	sess, exists := b.sessionMap[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Use aider's /add command to add file to chat
	cmd := fmt.Sprintf("/add %s", path)
	return b.tmux.SendKeys(ctx, sess.TmuxSession, cmd)
}

// GetStatus retrieves the current status of a session.
func (b *Backend) GetStatus(ctx context.Context, sessionID string) (*ports.SessionStatus, error) {
	sess, exists := b.sessionMap[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if tmux session exists
	exists, err := b.tmux.SessionExists(ctx, sess.TmuxSession)
	if err != nil {
		return nil, err
	}

	// Capture recent output
	output, err := b.tmux.CapturePane(ctx, sess.TmuxSession, 50)
	if err != nil {
		output = []string{}
	}

	return &ports.SessionStatus{
		Session:   sess,
		IsRunning: exists,
		Output:    output,
	}, nil
}

// GetTokenUsage retrieves token usage statistics.
func (b *Backend) GetTokenUsage(ctx context.Context, sessionID string) (*session.TokenUsage, error) {
	// Aider doesn't expose token usage directly
	// Would need to parse output or use API
	return nil, fmt.Errorf("token usage not supported for aider")
}

// SetModel changes the model for a session.
func (b *Backend) SetModel(ctx context.Context, model string) error {
	// Use aider's /model command
	// Note: This requires an active session context
	return fmt.Errorf("model switching requires session context")
}

// GetSupportedModels returns a list of models supported by Aider.
func (b *Backend) GetSupportedModels(ctx context.Context) ([]string, error) {
	// Aider supports many models through different providers
	// This is a common subset
	return []string{
		"gpt-4o",
		"gpt-4-turbo",
		"claude-3-5-sonnet-20241022",
		"claude-3-opus-20240229",
		"deepseek/deepseek-chat",
		"ollama/qwen2.5-coder:32b",
	}, nil
}

// SupportsModelControl indicates whether the backend supports changing models.
func (b *Backend) SupportsModelControl() bool {
	return true
}

// buildCommand builds the aider command with configuration.
func (b *Backend) buildCommand(workspace string, config session.BackendConfig) string {
	args := []string{b.executable}

	// Model
	if config.AiderWeakModel != "" {
		args = append(args, "--model", config.AiderWeakModel)
	}

	// Edit format
	if config.AiderEditFormat != "" {
		args = append(args, "--edit-format", config.AiderEditFormat)
	}

	// Auto-commit
	if config.AiderAutoCommit {
		args = append(args, "--auto-commits")
	} else {
		args = append(args, "--no-auto-commits")
	}

	// Dirty commits
	if config.AiderDirtyCommits {
		args = append(args, "--dirty-commits")
	}

	// Map tokens
	if config.AiderMapTokens > 0 {
		args = append(args, "--map-tokens", fmt.Sprintf("%d", config.AiderMapTokens))
	}

	// Cache prompts
	if config.AiderCachePrompts {
		args = append(args, "--cache-prompts")
	}

	// Test command
	if config.AiderTestCmd != "" {
		args = append(args, "--test-cmd", config.AiderTestCmd)
	}

	// Lint command
	if config.AiderLintCmd != "" {
		args = append(args, "--lint-cmd", config.AiderLintCmd)
	}

	// Auto test
	if config.AiderAutoTest {
		args = append(args, "--auto-test")
	}

	// Auto lint
	if config.AiderAutoLint {
		args = append(args, "--auto-lint")
	}

	return strings.Join(args, " ")
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	return uuid.New().String()[:8]
}
