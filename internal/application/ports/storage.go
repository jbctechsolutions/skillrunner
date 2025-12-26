// Package ports defines the application layer port interfaces following hexagonal architecture.
// Ports are abstractions that allow the application core to interact with external systems
// (adapters) without knowing their implementation details.
package ports

import (
	"context"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// CostFilter defines criteria for querying cost data.
type CostFilter struct {
	SkillID   string    // Filter by specific skill ID (empty for all)
	StartDate time.Time // Include costs from this date (zero for no lower bound)
	EndDate   time.Time // Include costs until this date (zero for no upper bound)
	Provider  string    // Filter by provider name (empty for all)
}

// CostSummary contains aggregated cost data from executions.
type CostSummary struct {
	TotalCost   float64            // Total cost across all matching executions
	TotalTokens int                // Total tokens used across all executions
	ByProvider  map[string]float64 // Cost breakdown by provider
	BySkill     map[string]float64 // Cost breakdown by skill
}

// OutcomeStats contains statistics about skill execution outcomes.
type OutcomeStats struct {
	TotalExecutions int           // Total number of executions
	SuccessCount    int           // Number of successful executions
	FailureCount    int           // Number of failed executions
	AvgDuration     time.Duration // Average execution duration
	SuccessRate     float64       // Success rate as a decimal (0.0 to 1.0)
}

// SkillSummary provides a lightweight representation of a skill for listing purposes.
type SkillSummary struct {
	ID          string // Unique skill identifier
	Name        string // Human-readable skill name
	Version     string // Skill version string
	Description string // Brief description of the skill
	PhaseCount  int    // Number of phases in the skill
}

// MetricsStoragePort defines the interface for storing and retrieving execution metrics.
// Implementations might use SQLite, PostgreSQL, or other storage backends.
type MetricsStoragePort interface {
	// SaveExecution persists an execution result to the metrics store.
	// Returns an error if the save operation fails.
	SaveExecution(ctx context.Context, exec *workflow.ExecutionResult) error

	// GetExecutionsBySkill retrieves execution results for a specific skill.
	// The limit parameter controls the maximum number of results returned.
	// Results are typically ordered by execution time (most recent first).
	GetExecutionsBySkill(ctx context.Context, skillID string, limit int) ([]workflow.ExecutionResult, error)

	// GetCostSummary retrieves aggregated cost data based on the provided filter.
	// Returns nil and an error if the query fails.
	GetCostSummary(ctx context.Context, filter CostFilter) (*CostSummary, error)

	// GetOutcomeStats retrieves execution statistics for a specific skill.
	// Returns nil and an error if the query fails.
	GetOutcomeStats(ctx context.Context, skillID string) (*OutcomeStats, error)
}

// SkillLoaderPort defines the interface for loading and discovering skills.
// Implementations might load from local YAML files, remote registries, or databases.
type SkillLoaderPort interface {
	// Load retrieves a skill by its ID.
	// Returns the skill or an error if not found or loading fails.
	Load(ctx context.Context, skillID string) (*skill.Skill, error)

	// List returns summaries of all available skills.
	// Returns an empty slice if no skills are found.
	List(ctx context.Context) ([]SkillSummary, error)

	// Exists checks whether a skill with the given ID exists.
	// Returns true if the skill exists, false otherwise.
	Exists(ctx context.Context, skillID string) (bool, error)

	// Refresh reloads the skill index from the underlying source.
	// This is useful when skills have been added or modified externally.
	Refresh(ctx context.Context) error
}
