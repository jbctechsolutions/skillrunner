// Package ports defines the application layer port interfaces following hexagonal architecture.
package ports

import (
	"context"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// -----------------------------------------------------------------------------
// Workflow Checkpoint Storage Port
// -----------------------------------------------------------------------------

// WorkflowCheckpointFilter defines criteria for querying workflow checkpoints.
type WorkflowCheckpointFilter struct {
	// SkillID filters by skill identifier.
	SkillID string

	// ExecutionID filters by execution correlation ID.
	ExecutionID string

	// MachineID filters by machine identifier.
	MachineID string

	// Status filters by checkpoint statuses (empty for all statuses).
	Status []workflow.CheckpointStatus

	// CreatedAfter filters to checkpoints created on or after this time.
	CreatedAfter time.Time

	// CreatedBefore filters to checkpoints created before this time.
	CreatedBefore time.Time

	// Limit specifies the maximum number of results to return (0 for unlimited).
	Limit int

	// Offset specifies the number of results to skip for pagination.
	Offset int
}

// WorkflowCheckpointPort defines the interface for workflow checkpoint persistence.
// This port stores execution state for crash recovery, enabling interrupted skill
// executions to be resumed from the last completed batch.
//
// Implementations might use SQLite, PostgreSQL, or other storage backends.
// All methods accept a context.Context for cancellation and timeout support.
// Methods return domain errors on failure.
type WorkflowCheckpointPort interface {
	// Create persists a new workflow checkpoint to storage.
	// Returns an error if a checkpoint with the same ID already exists or if validation fails.
	Create(ctx context.Context, checkpoint *workflow.WorkflowCheckpoint) error

	// Get retrieves a checkpoint by its unique identifier.
	// Returns ErrNotFound if no checkpoint exists with the given ID.
	Get(ctx context.Context, id string) (*workflow.WorkflowCheckpoint, error)

	// GetLatestInProgress retrieves the most recent in-progress checkpoint for a skill/input combination.
	// This is used for auto-detection of resumable executions.
	// Returns nil (not error) if no in-progress checkpoint exists.
	GetLatestInProgress(ctx context.Context, skillID string, inputHash string) (*workflow.WorkflowCheckpoint, error)

	// GetByExecutionID retrieves all checkpoints for an execution.
	// Returns checkpoints ordered by updatedAt descending.
	// Returns an empty slice if no checkpoints exist for the execution.
	GetByExecutionID(ctx context.Context, executionID string) ([]*workflow.WorkflowCheckpoint, error)

	// Update persists changes to an existing checkpoint.
	// Used to update batch progress, status, and token counts.
	// Returns ErrNotFound if the checkpoint does not exist.
	Update(ctx context.Context, checkpoint *workflow.WorkflowCheckpoint) error

	// List returns all checkpoints matching the optional filter criteria.
	// Pass nil filter to retrieve all checkpoints.
	// Results are ordered by updatedAt descending (most recent first).
	List(ctx context.Context, filter *WorkflowCheckpointFilter) ([]*workflow.WorkflowCheckpoint, error)

	// Delete removes a checkpoint from storage.
	// Returns ErrNotFound if the checkpoint does not exist.
	Delete(ctx context.Context, id string) error

	// DeleteByExecutionID removes all checkpoints for an execution.
	// Called after successful completion or explicit abandonment.
	// Returns the number of checkpoints deleted.
	DeleteByExecutionID(ctx context.Context, executionID string) (int, error)

	// MarkAbandoned marks all in-progress checkpoints for a machine as abandoned.
	// Called on startup to handle checkpoints from crashed processes.
	// Returns the number of checkpoints marked as abandoned.
	MarkAbandoned(ctx context.Context, machineID string) (int, error)

	// Cleanup removes checkpoints older than the specified duration.
	// Only removes checkpoints with status completed, failed, or abandoned.
	// Returns the number of checkpoints removed.
	Cleanup(ctx context.Context, olderThan time.Duration) (int, error)
}
