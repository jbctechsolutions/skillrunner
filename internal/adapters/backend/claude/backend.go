// Package claude provides the Claude Code backend implementation.
package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/process"
)

// Backend implements the BackendPort interface for Claude Code.
type Backend struct {
	tmux       *process.TmuxManager
	executable string
	sessionMap map[string]*session.Session
	machineID  string
	hooksDir   string
}

// NewBackend creates a new Claude Code backend.
func NewBackend(machineID string) (*Backend, error) {
	// Check if claude is available
	executable, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Initialize tmux manager
	tmux, err := process.NewTmuxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tmux: %w", err)
	}

	// Get hooks directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	hooksDir := filepath.Join(homeDir, ".claude", "hooks")

	return &Backend{
		tmux:       tmux,
		executable: executable,
		sessionMap: make(map[string]*session.Session),
		machineID:  machineID,
		hooksDir:   hooksDir,
	}, nil
}

// Info returns metadata about the Claude Code backend.
func (b *Backend) Info() ports.BackendInfo {
	return ports.BackendInfo{
		Name:        "claude",
		Version:     "0.1.0", // TODO: Get actual version
		Description: "Claude Code - Anthropic's official coding assistant",
		Executable:  b.executable,
		Features:    []string{"hooks", "rules", "context-injection"},
	}
}

// Start creates and starts a new Claude Code session.
func (b *Backend) Start(ctx context.Context, workspace string, config session.BackendConfig) (*session.Session, error) {
	// Generate session ID
	sessionID := generateSessionID()
	tmuxSession := fmt.Sprintf("claude-%s", sessionID)

	// Ensure hooks are installed
	if err := b.ensureHooks(config); err != nil {
		return nil, fmt.Errorf("failed to install hooks: %w", err)
	}

	// Sync CLAUDE.md if specified
	if config.ClaudeRulesFile != "" {
		if err := b.syncRules(workspace, config.ClaudeRulesFile); err != nil {
			return nil, fmt.Errorf("failed to sync rules: %w", err)
		}
	}

	// Create tmux session
	if err := b.tmux.CreateSession(ctx, tmuxSession, workspace); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Start claude in the tmux session
	cmd := b.executable
	if err := b.tmux.SendKeys(ctx, tmuxSession, cmd); err != nil {
		_ = b.tmux.KillSession(ctx, tmuxSession) // Best-effort cleanup
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Get process ID
	pid, err := b.tmux.GetSessionPID(ctx, tmuxSession)
	if err != nil {
		pid = 0 // Not critical
	}

	// Create session object
	sess := &session.Session{
		ID:          sessionID,
		WorkspaceID: filepath.Base(workspace),
		Backend:     "claude",
		Model:       "claude-sonnet-4-5", // Claude Code uses its own model
		Status:      session.StatusActive,
		StartedAt:   time.Now(),
		MachineID:   b.machineID,
		ProcessID:   pid,
		TmuxSession: tmuxSession,
		Metadata: map[string]string{
			"hooks_dir":  b.hooksDir,
			"rules_file": config.ClaudeRulesFile,
		},
		Context: &session.Context{
			WorkingDirectory: workspace,
		},
	}

	b.sessionMap[sessionID] = sess
	return sess, nil
}

// Attach connects to an existing Claude Code session.
func (b *Backend) Attach(ctx context.Context, sessionID string) error {
	sess, exists := b.sessionMap[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return b.tmux.AttachSession(ctx, sess.TmuxSession)
}

// Detach disconnects from a session without killing it.
func (b *Backend) Detach(ctx context.Context) error {
	return b.tmux.DetachSession(ctx, "")
}

// Kill terminates a Claude Code session.
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

	// Send content as input to claude
	return b.tmux.SendKeys(ctx, sess.TmuxSession, content)
}

// InjectFile injects a file into a running session.
func (b *Backend) InjectFile(ctx context.Context, sessionID, path string) error {
	sess, exists := b.sessionMap[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Read file content and inject
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	prompt := fmt.Sprintf("Please review this file:\n\n```\n%s\n```", string(content))
	return b.tmux.SendKeys(ctx, sess.TmuxSession, prompt)
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
	// Claude Code doesn't expose token usage directly
	return nil, fmt.Errorf("token usage not supported for claude")
}

// SetModel changes the model for a session.
func (b *Backend) SetModel(ctx context.Context, model string) error {
	// Claude Code doesn't support model switching
	return fmt.Errorf("model control not supported for claude")
}

// GetSupportedModels returns a list of models supported by Claude Code.
func (b *Backend) GetSupportedModels(ctx context.Context) ([]string, error) {
	// Claude Code uses its own model selection
	return []string{
		"claude-sonnet-4-5",
		"claude-opus-4-5",
	}, nil
}

// SupportsModelControl indicates whether the backend supports changing models.
func (b *Backend) SupportsModelControl() bool {
	return false
}

// ensureHooks ensures hooks are installed in ~/.claude/hooks/.
func (b *Backend) ensureHooks(config session.BackendConfig) error {
	// Create hooks directory if it doesn't exist
	if err := os.MkdirAll(b.hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Install SessionStart.sh if specified
	if config.ClaudeSessionHook != "" {
		hookPath := filepath.Join(b.hooksDir, "SessionStart.sh")
		if err := os.WriteFile(hookPath, []byte(config.ClaudeSessionHook), 0755); err != nil {
			return fmt.Errorf("failed to write SessionStart hook: %w", err)
		}
	}

	// Install PreCompact.sh if specified
	if config.ClaudePreCompact != "" {
		hookPath := filepath.Join(b.hooksDir, "PreCompact.sh")
		if err := os.WriteFile(hookPath, []byte(config.ClaudePreCompact), 0755); err != nil {
			return fmt.Errorf("failed to write PreCompact hook: %w", err)
		}
	}

	return nil
}

// syncRules syncs CLAUDE.md rules to the workspace.
func (b *Backend) syncRules(workspace, rulesFile string) error {
	// Read rules file
	content, err := os.ReadFile(rulesFile)
	if err != nil {
		return fmt.Errorf("failed to read rules file: %w", err)
	}

	// Write to workspace CLAUDE.md
	claudeMD := filepath.Join(workspace, "CLAUDE.md")
	if err := os.WriteFile(claudeMD, content, 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	return nil
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	return uuid.New().String()[:8]
}
