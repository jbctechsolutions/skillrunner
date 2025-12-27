// Package ports defines the application layer port interfaces following hexagonal architecture.
package ports

import (
	"context"

	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// WorkflowSessionStoragePort defines the interface for storing and retrieving workflow sessions.
// These sessions represent individual skill execution sessions within a workspace.
type WorkflowSessionStoragePort interface {
	// Save persists a session to storage.
	Save(ctx context.Context, session *Session) error

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (*Session, error)

	// GetByWorkspace retrieves all sessions for a workspace.
	GetByWorkspace(ctx context.Context, workspaceID string) ([]*Session, error)

	// GetActive retrieves the active session for a workspace, if any.
	GetActive(ctx context.Context, workspaceID string) (*Session, error)

	// Update updates an existing session.
	Update(ctx context.Context, session *Session) error

	// Delete removes a session from storage.
	Delete(ctx context.Context, id string) error
}

// CheckpointStoragePort defines the interface for storing and retrieving checkpoints.
type CheckpointStoragePort interface {
	// Save persists a checkpoint to storage.
	Save(ctx context.Context, checkpoint *domainContext.Checkpoint) error

	// Get retrieves a checkpoint by ID.
	Get(ctx context.Context, id string) (*domainContext.Checkpoint, error)

	// GetBySession retrieves all checkpoints for a session.
	GetBySession(ctx context.Context, sessionID string) ([]*domainContext.Checkpoint, error)

	// GetByWorkspace retrieves all checkpoints for a workspace.
	GetByWorkspace(ctx context.Context, workspaceID string) ([]*domainContext.Checkpoint, error)

	// GetLatest retrieves the most recent checkpoint for a session.
	GetLatest(ctx context.Context, sessionID string) (*domainContext.Checkpoint, error)

	// List returns all checkpoints, optionally limited.
	List(ctx context.Context, limit int) ([]*domainContext.Checkpoint, error)

	// Delete removes a checkpoint from storage.
	Delete(ctx context.Context, id string) error
}

// ContextItemStoragePort defines the interface for storing and retrieving context items.
type ContextItemStoragePort interface {
	// Save persists a context item to storage.
	Save(ctx context.Context, item *domainContext.ContextItem) error

	// Get retrieves a context item by ID.
	Get(ctx context.Context, id string) (*domainContext.ContextItem, error)

	// GetByName retrieves a context item by name.
	GetByName(ctx context.Context, name string) (*domainContext.ContextItem, error)

	// List returns all context items.
	List(ctx context.Context) ([]*domainContext.ContextItem, error)

	// ListByTag retrieves context items with a specific tag.
	ListByTag(ctx context.Context, tag string) ([]*domainContext.ContextItem, error)

	// Update updates an existing context item.
	Update(ctx context.Context, item *domainContext.ContextItem) error

	// Delete removes a context item from storage.
	Delete(ctx context.Context, id string) error

	// Exists checks if a context item exists.
	Exists(ctx context.Context, id string) (bool, error)
}

// RuleStoragePort defines the interface for storing and retrieving rules.
type RuleStoragePort interface {
	// Save persists a rule to storage.
	Save(ctx context.Context, rule *domainContext.Rule) error

	// Get retrieves a rule by ID.
	Get(ctx context.Context, id string) (*domainContext.Rule, error)

	// GetByName retrieves a rule by name.
	GetByName(ctx context.Context, name string) (*domainContext.Rule, error)

	// List returns all rules.
	List(ctx context.Context) ([]*domainContext.Rule, error)

	// ListActive retrieves all active rules.
	ListActive(ctx context.Context) ([]*domainContext.Rule, error)

	// ListByScope retrieves rules for a specific scope.
	ListByScope(ctx context.Context, scope domainContext.RuleScope) ([]*domainContext.Rule, error)

	// Update updates an existing rule.
	Update(ctx context.Context, rule *domainContext.Rule) error

	// Delete removes a rule from storage.
	Delete(ctx context.Context, id string) error

	// Exists checks if a rule exists.
	Exists(ctx context.Context, id string) (bool, error)
}

// Session represents a skill execution session within a workspace.
// This is a simple DTO for now; can be promoted to domain entity if needed.
type Session struct {
	ID          string
	WorkspaceID string
	SkillID     string
	Status      string
	StartedAt   string
	CompletedAt *string
}

// SessionStoragePort defines the interface for storing and retrieving AI coding assistant sessions.
// These sessions represent backend sessions (Aider, Claude Code, OpenCode) managed by the session manager.
type SessionStoragePort interface {
	// SaveSession persists a session to storage.
	SaveSession(ctx context.Context, sess *session.Session) error

	// GetSession retrieves a session by ID.
	GetSession(ctx context.Context, id string) (*session.Session, error)

	// GetActiveByWorkspace retrieves the active session for a workspace, if any.
	GetActiveByWorkspace(ctx context.Context, workspaceID string) (*session.Session, error)

	// ListSessions returns sessions matching the filter.
	ListSessions(ctx context.Context, filter session.Filter) ([]*session.Session, error)

	// UpdateSession updates an existing session.
	UpdateSession(ctx context.Context, sess *session.Session) error

	// DeleteSession removes a session from storage.
	DeleteSession(ctx context.Context, id string) error
}
