package ports

import (
	"context"
	"time"
)

// SyncState represents the complete synchronization state
type SyncState struct {
	Workspaces  []WorkspaceSnapshot
	Sessions    []SessionSnapshot
	Checkpoints []CheckpointSnapshot
	Items       []ContextItemSnapshot
	Rules       []RuleSnapshot
	Version     string
	MachineID   string
	SyncedAt    time.Time
}

// WorkspaceSnapshot represents a workspace in sync state
type WorkspaceSnapshot struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SessionSnapshot represents a session in sync state
type SessionSnapshot struct {
	ID          string
	WorkspaceID string
	Name        string
	StartedAt   time.Time
	EndedAt     *time.Time
	Status      string
}

// CheckpointSnapshot represents a checkpoint in sync state
type CheckpointSnapshot struct {
	ID          string
	SessionID   string
	Name        string
	Description string
	CreatedAt   time.Time
}

// ContextItemSnapshot represents a context item in sync state
type ContextItemSnapshot struct {
	ID           string
	CheckpointID string
	Type         string
	Key          string
	Value        string
	CreatedAt    time.Time
}

// RuleSnapshot represents a rule in sync state
type RuleSnapshot struct {
	ID          string
	WorkspaceID string
	Name        string
	Pattern     string
	Action      string
	Priority    int
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SyncBackendPort defines the interface for sync backends
type SyncBackendPort interface {
	// Push uploads the current state to the backend
	Push(ctx context.Context, state *SyncState) error

	// Pull retrieves the latest state from the backend
	Pull(ctx context.Context) (*SyncState, error)

	// HasUpdates checks if the backend has updates since the given timestamp
	HasUpdates(ctx context.Context, since time.Time) (bool, error)

	// IsAvailable checks if the backend is currently available
	IsAvailable(ctx context.Context) (bool, error)
}
