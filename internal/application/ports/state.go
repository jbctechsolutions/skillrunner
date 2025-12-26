// Package ports defines the application layer port interfaces following hexagonal architecture.
// Ports are abstractions that allow the application core to interact with external systems
// (adapters) without knowing their implementation details.
package ports

import (
	"context"
	"time"

	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// -----------------------------------------------------------------------------
// Workspace Storage Port
// -----------------------------------------------------------------------------

// WorkspaceFilter defines criteria for querying workspaces.
type WorkspaceFilter struct {
	// Status filters by workspace status (empty for all statuses).
	Status domainContext.WorkspaceStatus

	// Name filters by workspace name using pattern matching.
	Name string

	// RepoPath filters by repository path.
	RepoPath string

	// Limit specifies the maximum number of results to return (0 for unlimited).
	Limit int

	// Offset specifies the number of results to skip for pagination.
	Offset int
}

// WorkspaceStateStoragePort defines the interface for storing and retrieving workspace state.
// This port provides comprehensive CRUD operations for workspaces along with
// specialized query methods for finding workspaces by name, status, and focus.
//
// Implementations might use SQLite, PostgreSQL, or other storage backends.
// All methods accept a context.Context for cancellation and timeout support.
// Methods return domain errors on failure.
type WorkspaceStateStoragePort interface {
	// Create persists a new workspace to storage.
	// Returns an error if a workspace with the same ID already exists or if validation fails.
	Create(ctx context.Context, workspace *domainContext.Workspace) error

	// Get retrieves a workspace by its unique identifier.
	// Returns ErrNotFound if no workspace exists with the given ID.
	Get(ctx context.Context, id string) (*domainContext.Workspace, error)

	// GetByName retrieves a workspace by its human-readable name.
	// Returns ErrNotFound if no workspace exists with the given name.
	GetByName(ctx context.Context, name string) (*domainContext.Workspace, error)

	// GetByRepoPath retrieves a workspace associated with a repository path.
	// Returns ErrNotFound if no workspace exists for the given repository path.
	GetByRepoPath(ctx context.Context, repoPath string) (*domainContext.Workspace, error)

	// GetActive retrieves all workspaces with active status.
	// Returns an empty slice if no active workspaces exist.
	GetActive(ctx context.Context) ([]*domainContext.Workspace, error)

	// List returns all workspaces matching the optional filter criteria.
	// Pass nil filter to retrieve all workspaces.
	// Results are ordered by last active time (most recent first).
	List(ctx context.Context, filter *WorkspaceFilter) ([]*domainContext.Workspace, error)

	// Update persists changes to an existing workspace.
	// Returns ErrNotFound if the workspace does not exist.
	Update(ctx context.Context, workspace *domainContext.Workspace) error

	// SetFocus updates the focus field of a workspace (e.g., current issue or task).
	// This is a convenience method that atomically updates the focus without
	// requiring a full workspace load-modify-save cycle.
	// Returns ErrNotFound if the workspace does not exist.
	SetFocus(ctx context.Context, id string, focus string) error

	// SetStatus updates the status of a workspace.
	// Returns ErrNotFound if the workspace does not exist.
	SetStatus(ctx context.Context, id string, status domainContext.WorkspaceStatus) error

	// Delete removes a workspace from storage.
	// Returns ErrNotFound if the workspace does not exist.
	Delete(ctx context.Context, id string) error

	// Exists checks whether a workspace with the given ID exists.
	Exists(ctx context.Context, id string) (bool, error)
}

// -----------------------------------------------------------------------------
// Session Storage Port
// -----------------------------------------------------------------------------

// SessionStateStoragePort defines the interface for storing and retrieving AI coding
// assistant session state. Sessions represent backend executions (Aider, Claude Code,
// OpenCode) managed by the session manager.
//
// This port supports the full session lifecycle including creation, status updates,
// token usage tracking, and querying by various criteria.
//
// Implementations might use SQLite, PostgreSQL, or other storage backends.
// All methods accept a context.Context for cancellation and timeout support.
// Methods return domain errors on failure.
type SessionStateStoragePort interface {
	// Create persists a new session to storage.
	// Returns an error if a session with the same ID already exists or if validation fails.
	Create(ctx context.Context, sess *session.Session) error

	// Get retrieves a session by its unique identifier.
	// Returns ErrNotFound if no session exists with the given ID.
	Get(ctx context.Context, id string) (*session.Session, error)

	// GetByWorkspace retrieves all sessions associated with a workspace.
	// Returns an empty slice if no sessions exist for the workspace.
	// Results are ordered by start time (most recent first).
	GetByWorkspace(ctx context.Context, workspaceID string) ([]*session.Session, error)

	// GetActive retrieves all currently active sessions (status: active, idle, or detached).
	// Returns an empty slice if no active sessions exist.
	GetActive(ctx context.Context) ([]*session.Session, error)

	// GetActiveByWorkspace retrieves the active session for a specific workspace.
	// Returns nil if no active session exists for the workspace.
	GetActiveByWorkspace(ctx context.Context, workspaceID string) (*session.Session, error)

	// List returns sessions matching the filter criteria.
	// Pass an empty filter to retrieve all sessions.
	// Results are ordered by start time (most recent first).
	List(ctx context.Context, filter session.Filter) ([]*session.Session, error)

	// Update persists changes to an existing session.
	// Returns ErrNotFound if the session does not exist.
	Update(ctx context.Context, sess *session.Session) error

	// UpdateStatus atomically updates the status of a session.
	// If the new status is a terminal state (completed, failed, killed),
	// the EndedAt field is also set to the current time.
	// Returns ErrNotFound if the session does not exist.
	UpdateStatus(ctx context.Context, id string, status session.Status) error

	// UpdateTokenUsage atomically updates the token usage statistics for a session.
	// This method is optimized for frequent updates during session execution.
	// Returns ErrNotFound if the session does not exist.
	UpdateTokenUsage(ctx context.Context, id string, usage *session.TokenUsage) error

	// Delete removes a session from storage.
	// Returns ErrNotFound if the session does not exist.
	Delete(ctx context.Context, id string) error
}

// -----------------------------------------------------------------------------
// Checkpoint Storage Port
// -----------------------------------------------------------------------------

// CheckpointFilter defines criteria for querying checkpoints.
type CheckpointFilter struct {
	// SessionID filters by session ID.
	SessionID string

	// WorkspaceID filters by workspace ID.
	WorkspaceID string

	// MachineID filters by machine ID.
	MachineID string

	// CreatedAfter filters checkpoints created after this time.
	CreatedAfter time.Time

	// CreatedBefore filters checkpoints created before this time.
	CreatedBefore time.Time

	// Limit specifies the maximum number of results to return (0 for unlimited).
	Limit int

	// Offset specifies the number of results to skip for pagination.
	Offset int
}

// CheckpointStateStoragePort defines the interface for storing and retrieving checkpoint state.
// Checkpoints capture snapshots of skill execution sessions, including summaries,
// modified files, and decisions made during the session.
//
// Implementations might use SQLite, PostgreSQL, or other storage backends.
// All methods accept a context.Context for cancellation and timeout support.
// Methods return domain errors on failure.
type CheckpointStateStoragePort interface {
	// Create persists a new checkpoint to storage.
	// Returns an error if a checkpoint with the same ID already exists or if validation fails.
	Create(ctx context.Context, checkpoint *domainContext.Checkpoint) error

	// Get retrieves a checkpoint by its unique identifier.
	// Returns ErrNotFound if no checkpoint exists with the given ID.
	Get(ctx context.Context, id string) (*domainContext.Checkpoint, error)

	// GetBySession retrieves all checkpoints for a specific session.
	// Returns an empty slice if no checkpoints exist for the session.
	// Results are ordered by creation time (most recent first).
	GetBySession(ctx context.Context, sessionID string) ([]*domainContext.Checkpoint, error)

	// GetByWorkspace retrieves all checkpoints for a specific workspace.
	// Returns an empty slice if no checkpoints exist for the workspace.
	// Results are ordered by creation time (most recent first).
	GetByWorkspace(ctx context.Context, workspaceID string) ([]*domainContext.Checkpoint, error)

	// GetLatest retrieves the most recent checkpoint for a session.
	// Returns nil if no checkpoints exist for the session.
	GetLatest(ctx context.Context, sessionID string) (*domainContext.Checkpoint, error)

	// List returns checkpoints matching the filter criteria.
	// Pass nil filter to retrieve all checkpoints.
	// Results are ordered by creation time (most recent first).
	List(ctx context.Context, filter *CheckpointFilter) ([]*domainContext.Checkpoint, error)

	// Update persists changes to an existing checkpoint.
	// Returns ErrNotFound if the checkpoint does not exist.
	Update(ctx context.Context, checkpoint *domainContext.Checkpoint) error

	// Delete removes a checkpoint from storage.
	// Returns ErrNotFound if the checkpoint does not exist.
	Delete(ctx context.Context, id string) error

	// DeleteBySession removes all checkpoints for a specific session.
	// Returns the number of checkpoints deleted.
	DeleteBySession(ctx context.Context, sessionID string) (int, error)
}

// -----------------------------------------------------------------------------
// Context Item Storage Port
// -----------------------------------------------------------------------------

// ContextItemFilter defines criteria for querying context items.
type ContextItemFilter struct {
	// Name filters by item name using pattern matching.
	Name string

	// ItemType filters by item type (file, snippet, url).
	ItemType domainContext.ItemType

	// Tags filters items that have ALL of the specified tags.
	Tags []string

	// AnyTags filters items that have ANY of the specified tags.
	AnyTags []string

	// CreatedAfter filters items created after this time.
	CreatedAfter time.Time

	// LastUsedAfter filters items last used after this time.
	LastUsedAfter time.Time

	// Limit specifies the maximum number of results to return (0 for unlimited).
	Limit int

	// Offset specifies the number of results to skip for pagination.
	Offset int
}

// ContextItemStateStoragePort defines the interface for storing and retrieving context items.
// Context items are pieces of information (files, snippets, URLs) that can be loaded
// into skill execution sessions to provide relevant context.
//
// Implementations might use SQLite, PostgreSQL, or other storage backends.
// All methods accept a context.Context for cancellation and timeout support.
// Methods return domain errors on failure.
type ContextItemStateStoragePort interface {
	// Create persists a new context item to storage.
	// Returns an error if an item with the same ID already exists or if validation fails.
	Create(ctx context.Context, item *domainContext.ContextItem) error

	// Get retrieves a context item by its unique identifier.
	// Returns ErrNotFound if no item exists with the given ID.
	Get(ctx context.Context, id string) (*domainContext.ContextItem, error)

	// GetByName retrieves a context item by its name.
	// Returns ErrNotFound if no item exists with the given name.
	GetByName(ctx context.Context, name string) (*domainContext.ContextItem, error)

	// GetByTags retrieves context items that have ALL of the specified tags.
	// Returns an empty slice if no matching items exist.
	// Results are ordered by last used time (most recent first).
	GetByTags(ctx context.Context, tags []string) ([]*domainContext.ContextItem, error)

	// Search performs a text search across item names and content.
	// The query supports basic substring matching.
	// Returns an empty slice if no matching items exist.
	// Results are ordered by relevance, then by last used time.
	Search(ctx context.Context, query string) ([]*domainContext.ContextItem, error)

	// List returns context items matching the filter criteria.
	// Pass nil filter to retrieve all items.
	// Results are ordered by last used time (most recent first).
	List(ctx context.Context, filter *ContextItemFilter) ([]*domainContext.ContextItem, error)

	// Update persists changes to an existing context item.
	// Returns ErrNotFound if the item does not exist.
	Update(ctx context.Context, item *domainContext.ContextItem) error

	// UpdateLastUsed atomically updates the last used timestamp for an item.
	// This method is optimized for frequent updates during session execution.
	// Returns ErrNotFound if the item does not exist.
	UpdateLastUsed(ctx context.Context, id string, lastUsed time.Time) error

	// Delete removes a context item from storage.
	// Returns ErrNotFound if the item does not exist.
	Delete(ctx context.Context, id string) error

	// Exists checks whether a context item with the given ID exists.
	Exists(ctx context.Context, id string) (bool, error)
}

// -----------------------------------------------------------------------------
// Rule Storage Port
// -----------------------------------------------------------------------------

// RuleFilter defines criteria for querying rules.
type RuleFilter struct {
	// Name filters by rule name using pattern matching.
	Name string

	// Scope filters by rule scope (global, workspace, session).
	Scope domainContext.RuleScope

	// IsActive filters by active status (nil for all, true for active only, false for inactive only).
	IsActive *bool

	// CreatedAfter filters rules created after this time.
	CreatedAfter time.Time

	// Limit specifies the maximum number of results to return (0 for unlimited).
	Limit int

	// Offset specifies the number of results to skip for pagination.
	Offset int
}

// RuleStateStoragePort defines the interface for storing and retrieving rule state.
// Rules are context guidelines that apply during skill execution.
// They can be global, workspace-specific, or session-specific.
//
// Implementations might use SQLite, PostgreSQL, or other storage backends.
// All methods accept a context.Context for cancellation and timeout support.
// Methods return domain errors on failure.
type RuleStateStoragePort interface {
	// Create persists a new rule to storage.
	// Returns an error if a rule with the same ID already exists or if validation fails.
	Create(ctx context.Context, rule *domainContext.Rule) error

	// Get retrieves a rule by its unique identifier.
	// Returns ErrNotFound if no rule exists with the given ID.
	Get(ctx context.Context, id string) (*domainContext.Rule, error)

	// GetByName retrieves a rule by its name.
	// Returns ErrNotFound if no rule exists with the given name.
	GetByName(ctx context.Context, name string) (*domainContext.Rule, error)

	// GetActive retrieves all rules that are currently active.
	// Returns an empty slice if no active rules exist.
	// Results are ordered by creation time (oldest first, for consistent ordering).
	GetActive(ctx context.Context) ([]*domainContext.Rule, error)

	// GetByScope retrieves all rules for a specific scope.
	// Returns an empty slice if no rules exist for the scope.
	// Results are ordered by creation time (oldest first).
	GetByScope(ctx context.Context, scope domainContext.RuleScope) ([]*domainContext.Rule, error)

	// List returns rules matching the filter criteria.
	// Pass nil filter to retrieve all rules.
	// Results are ordered by creation time (oldest first).
	List(ctx context.Context, filter *RuleFilter) ([]*domainContext.Rule, error)

	// Update persists changes to an existing rule.
	// Returns ErrNotFound if the rule does not exist.
	Update(ctx context.Context, rule *domainContext.Rule) error

	// Activate sets a rule's active status to true.
	// Returns ErrNotFound if the rule does not exist.
	Activate(ctx context.Context, id string) error

	// Deactivate sets a rule's active status to false.
	// Returns ErrNotFound if the rule does not exist.
	Deactivate(ctx context.Context, id string) error

	// Delete removes a rule from storage.
	// Returns ErrNotFound if the rule does not exist.
	Delete(ctx context.Context, id string) error

	// Exists checks whether a rule with the given ID exists.
	Exists(ctx context.Context, id string) (bool, error)
}
