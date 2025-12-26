// Package session provides session management functionality.
package session

import (
	"context"
	"fmt"

	"github.com/jbctechsolutions/skillrunner/internal/adapters/backend"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// Manager manages AI coding assistant sessions.
type Manager struct {
	storage   ports.SessionStoragePort
	backends  *backend.Registry
	machineID string
}

// NewManager creates a new session manager.
func NewManager(storage ports.SessionStoragePort, backends *backend.Registry, machineID string) *Manager {
	return &Manager{
		storage:   storage,
		backends:  backends,
		machineID: machineID,
	}
}

// Start creates and starts a new session.
func (m *Manager) Start(ctx context.Context, opts session.StartOptions) (*session.Session, error) {
	// Validate options
	if opts.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	if opts.Backend == "" {
		return nil, fmt.Errorf("backend is required")
	}

	// Get backend
	backend, err := m.backends.GetRequired(opts.Backend)
	if err != nil {
		return nil, fmt.Errorf("backend not available: %w", err)
	}

	// Start session using backend
	sess, err := backend.Start(ctx, opts.WorkspaceID, opts.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Apply additional context if provided
	if len(opts.ContextFiles) > 0 {
		for _, file := range opts.ContextFiles {
			if err := backend.InjectFile(ctx, sess.ID, file); err != nil {
				// Log warning but continue
				continue
			}
		}
	}

	// Apply documentation if provided
	if len(opts.Documentation) > 0 {
		sess.Context.Documentation = opts.Documentation
	}

	// Apply protocols if provided
	if len(opts.Protocols) > 0 {
		sess.Context.Protocols = opts.Protocols
	}

	// Inject initial task if provided
	if opts.Task != "" {
		if err := backend.InjectContext(ctx, sess.ID, opts.Task); err != nil {
			// Log warning but continue
		}
	}

	// Save session to storage
	if err := m.storage.SaveSession(ctx, sess); err != nil {
		// Try to kill the session if save failed
		backend.Kill(ctx, sess.ID)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// If background mode, detach
	if opts.Background {
		if err := backend.Detach(ctx); err != nil {
			// Not critical, log and continue
		}
		sess.Status = session.StatusDetached
		_ = m.storage.UpdateSession(ctx, sess) // Best-effort update
	}

	return sess, nil
}

// Attach connects to an existing session.
func (m *Manager) Attach(ctx context.Context, sessionID string) error {
	// Get session from storage
	sess, err := m.storage.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get backend
	backend, err := m.backends.GetRequired(sess.Backend)
	if err != nil {
		return fmt.Errorf("backend not available: %w", err)
	}

	// Attach to session
	if err := backend.Attach(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to attach to session: %w", err)
	}

	// Update session status
	sess.Status = session.StatusActive
	if err := m.storage.UpdateSession(ctx, sess); err != nil {
		// Not critical, log and continue
	}

	return nil
}

// Detach disconnects from current session.
func (m *Manager) Detach(ctx context.Context, sessionID string) error {
	// Get session from storage
	sess, err := m.storage.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get backend
	backend, err := m.backends.GetRequired(sess.Backend)
	if err != nil {
		return fmt.Errorf("backend not available: %w", err)
	}

	// Detach from session
	if err := backend.Detach(ctx); err != nil {
		return fmt.Errorf("failed to detach from session: %w", err)
	}

	// Update session status
	sess.Status = session.StatusDetached
	if err := m.storage.UpdateSession(ctx, sess); err != nil {
		// Not critical, log and continue
	}

	return nil
}

// Kill terminates a session.
func (m *Manager) Kill(ctx context.Context, sessionID string, force bool) error {
	// Get session from storage
	sess, err := m.storage.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get backend
	backend, err := m.backends.GetRequired(sess.Backend)
	if err != nil {
		return fmt.Errorf("backend not available: %w", err)
	}

	// Kill session
	if err := backend.Kill(ctx, sessionID); err != nil {
		if !force {
			return fmt.Errorf("failed to kill session: %w", err)
		}
		// In force mode, continue even if kill fails
	}

	// Update session in storage
	sess.Status = session.StatusKilled
	if err := m.storage.UpdateSession(ctx, sess); err != nil {
		// Not critical, log and continue
	}

	return nil
}

// List returns sessions matching the filter.
func (m *Manager) List(ctx context.Context, filter session.Filter) ([]*session.Session, error) {
	return m.storage.ListSessions(ctx, filter)
}

// Inject injects content into a session.
func (m *Manager) Inject(ctx context.Context, sessionID string, content session.InjectContent) error {
	// Get session from storage
	sess, err := m.storage.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get backend
	backend, err := m.backends.GetRequired(sess.Backend)
	if err != nil {
		return fmt.Errorf("backend not available: %w", err)
	}

	// Inject based on type
	switch content.Type {
	case "prompt":
		return backend.InjectContext(ctx, sessionID, content.Content)
	case "file":
		for _, file := range content.Files {
			if err := backend.InjectFile(ctx, sessionID, file); err != nil {
				return fmt.Errorf("failed to inject file %s: %w", file, err)
			}
		}
		return nil
	case "item":
		return backend.InjectContext(ctx, sessionID, content.Content)
	default:
		return fmt.Errorf("unknown inject type: %s", content.Type)
	}
}

// Peek returns recent output from a session.
func (m *Manager) Peek(ctx context.Context, sessionID string, lines int) ([]string, error) {
	// Get session from storage
	sess, err := m.storage.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Get backend
	backend, err := m.backends.GetRequired(sess.Backend)
	if err != nil {
		return nil, fmt.Errorf("backend not available: %w", err)
	}

	// Get status (includes output)
	status, err := backend.GetStatus(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session status: %w", err)
	}

	// Return requested number of lines
	if lines > 0 && len(status.Output) > lines {
		return status.Output[len(status.Output)-lines:], nil
	}

	return status.Output, nil
}

// GetStatus returns the current status of a session.
func (m *Manager) GetStatus(ctx context.Context, sessionID string) (*ports.SessionStatus, error) {
	// Get session from storage
	sess, err := m.storage.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Get backend
	backend, err := m.backends.GetRequired(sess.Backend)
	if err != nil {
		return nil, fmt.Errorf("backend not available: %w", err)
	}

	// Get status from backend
	return backend.GetStatus(ctx, sessionID)
}

// GetTokenUsage returns token usage for a session.
func (m *Manager) GetTokenUsage(ctx context.Context, sessionID string) (*session.TokenUsage, error) {
	// Get session from storage
	sess, err := m.storage.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Get backend
	backend, err := m.backends.GetRequired(sess.Backend)
	if err != nil {
		return nil, fmt.Errorf("backend not available: %w", err)
	}

	// Get token usage from backend
	return backend.GetTokenUsage(ctx, sessionID)
}
