package ports

import (
	"context"

	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// BackendInfo contains metadata about a backend.
type BackendInfo struct {
	Name        string   // Backend name (aider, claude, opencode)
	Version     string   // Backend version
	Description string   // Human-readable description
	Executable  string   // Path to executable or command name
	Features    []string // Supported features (model-control, hooks, etc)
}

// SessionStatus represents detailed status of a running session.
type SessionStatus struct {
	Session     *session.Session // Session metadata
	IsRunning   bool             // Whether process is running
	CPUUsage    float64          // CPU usage percentage
	MemoryUsage int64            // Memory usage in bytes
	Output      []string         // Recent output lines
}

// BackendPort defines the interface for AI coding assistant backends.
type BackendPort interface {
	// Info returns metadata about the backend.
	Info() BackendInfo

	// Start creates and starts a new session.
	// Returns the created session or an error.
	Start(ctx context.Context, workspace string, config session.BackendConfig) (*session.Session, error)

	// Attach connects to an existing session.
	// Returns an error if the session doesn't exist or cannot be attached.
	Attach(ctx context.Context, sessionID string) error

	// Detach disconnects from a session without killing it.
	// The session continues running in the background.
	Detach(ctx context.Context) error

	// Kill terminates a session.
	// Use force=true to send SIGKILL instead of graceful shutdown.
	Kill(ctx context.Context, sessionID string) error

	// InjectContext injects contextual information into a running session.
	// This could be documentation, code snippets, or instructions.
	InjectContext(ctx context.Context, sessionID, content string) error

	// InjectFile injects a file's contents into a running session.
	// The backend may add the file to the chat context.
	InjectFile(ctx context.Context, sessionID, path string) error

	// GetStatus retrieves the current status of a session.
	GetStatus(ctx context.Context, sessionID string) (*SessionStatus, error)

	// GetTokenUsage retrieves token usage statistics for a session.
	// Returns nil if token tracking is not supported.
	GetTokenUsage(ctx context.Context, sessionID string) (*session.TokenUsage, error)

	// SetModel changes the model for a session.
	// Returns an error if model control is not supported.
	SetModel(ctx context.Context, model string) error

	// GetSupportedModels returns a list of models supported by this backend.
	GetSupportedModels(ctx context.Context) ([]string, error)

	// SupportsModelControl indicates whether the backend supports changing models.
	SupportsModelControl() bool
}

// Note: SessionStoragePort and WorkspaceStoragePort are defined in context_storage.go
// to avoid conflicts with existing storage interfaces.
