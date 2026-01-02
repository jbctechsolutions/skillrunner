// Package ports defines the application layer port interfaces following hexagonal architecture.
// Ports are abstractions that allow the application core to interact with external systems
// (adapters) without knowing their implementation details.
package ports

import (
	"context"

	"github.com/jbctechsolutions/skillrunner/internal/domain/metrics"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

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
	// SaveExecution persists an execution record to the metrics store.
	// Returns an error if the save operation fails.
	SaveExecution(ctx context.Context, exec *metrics.ExecutionRecord) error

	// SavePhaseExecution persists a phase execution record.
	// Returns an error if the save operation fails.
	SavePhaseExecution(ctx context.Context, phase *metrics.PhaseExecutionRecord) error

	// GetExecutions retrieves execution records matching the filter.
	// Results are ordered by execution time (most recent first).
	GetExecutions(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.ExecutionRecord, error)

	// GetAggregatedMetrics retrieves aggregated metrics for the given filter.
	// Returns complete metrics including provider and skill breakdowns.
	GetAggregatedMetrics(ctx context.Context, filter metrics.MetricsFilter) (*metrics.AggregatedMetrics, error)

	// GetProviderMetrics retrieves aggregated metrics for all providers.
	GetProviderMetrics(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.ProviderMetrics, error)

	// GetSkillMetrics retrieves aggregated metrics for all skills.
	GetSkillMetrics(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.SkillMetrics, error)

	// GetCostSummary retrieves aggregated cost data based on the provided filter.
	GetCostSummary(ctx context.Context, filter metrics.MetricsFilter) (*metrics.CostSummary, error)
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
