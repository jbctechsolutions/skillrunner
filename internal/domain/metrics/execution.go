// Package metrics provides domain types for execution metrics and observability.
package metrics

import (
	"time"
)

// ExecutionRecord represents a single workflow execution record.
type ExecutionRecord struct {
	ID            string        // Unique execution ID
	SkillID       string        // ID of the executed skill
	SkillName     string        // Human-readable skill name
	Status        string        // Execution status (completed, failed, timeout)
	InputTokens   int           // Total input tokens consumed
	OutputTokens  int           // Total output tokens generated
	TotalCost     float64       // Total cost of execution
	Duration      time.Duration // Total execution duration
	PhaseCount    int           // Number of phases executed
	CacheHits     int           // Number of cache hits
	CacheMisses   int           // Number of cache misses
	PrimaryModel  string        // Primary model used
	StartedAt     time.Time     // When execution started
	CompletedAt   time.Time     // When execution completed
	CorrelationID string        // Correlation ID for tracing
}

// PhaseExecutionRecord represents a single phase execution within a workflow.
type PhaseExecutionRecord struct {
	ID           string        // Unique phase execution ID
	ExecutionID  string        // Parent execution ID
	PhaseID      string        // Phase ID from skill definition
	PhaseName    string        // Human-readable phase name
	Status       string        // Execution status
	Provider     string        // Provider used (ollama, anthropic, etc.)
	Model        string        // Model used
	InputTokens  int           // Input tokens consumed
	OutputTokens int           // Output tokens generated
	Cost         float64       // Cost of this phase
	Duration     time.Duration // Phase duration
	CacheHit     bool          // Whether result was served from cache
	StartedAt    time.Time     // When phase started
	CompletedAt  time.Time     // When phase completed
	ErrorMessage string        // Error message if failed
}

// ProviderMetrics represents aggregated metrics for a provider.
type ProviderMetrics struct {
	Name          string        // Provider name (ollama, anthropic, etc.)
	Type          string        // Provider type (local, cloud)
	TotalRequests int64         // Total number of requests
	SuccessCount  int64         // Number of successful requests
	FailedCount   int64         // Number of failed requests
	TokensInput   int64         // Total input tokens
	TokensOutput  int64         // Total output tokens
	TotalCost     float64       // Total cost
	AvgLatency    time.Duration // Average latency
	MinLatency    time.Duration // Minimum latency
	MaxLatency    time.Duration // Maximum latency
	CacheHits     int64         // Number of cache hits
	CacheMisses   int64         // Number of cache misses
	Period        TimePeriod    // Time period for these metrics
}

// SkillMetrics represents aggregated metrics for a skill.
type SkillMetrics struct {
	SkillID      string        // Skill ID
	SkillName    string        // Skill name
	TotalRuns    int64         // Total number of executions
	SuccessCount int64         // Number of successful executions
	FailedCount  int64         // Number of failed executions
	SuccessRate  float64       // Success rate (0.0 to 1.0)
	AvgDuration  time.Duration // Average execution duration
	MinDuration  time.Duration // Minimum execution duration
	MaxDuration  time.Duration // Maximum execution duration
	TotalTokens  int64         // Total tokens used
	TotalCost    float64       // Total cost
	Period       TimePeriod    // Time period for these metrics
}

// TimePeriod represents a time period for metrics aggregation.
type TimePeriod struct {
	Start time.Time
	End   time.Time
}

// Duration returns the duration of the time period.
func (p TimePeriod) Duration() time.Duration {
	return p.End.Sub(p.Start)
}

// CostSummary represents a summary of costs.
type CostSummary struct {
	TotalCost    float64            // Total cost across all matching records
	ByProvider   map[string]float64 // Cost breakdown by provider
	BySkill      map[string]float64 // Cost breakdown by skill
	ByModel      map[string]float64 // Cost breakdown by model
	TotalTokens  int64              // Total tokens used
	InputTokens  int64              // Total input tokens
	OutputTokens int64              // Total output tokens
	Period       TimePeriod         // Time period for this summary
}

// NewCostSummary creates a new initialized CostSummary.
func NewCostSummary(period TimePeriod) *CostSummary {
	return &CostSummary{
		ByProvider: make(map[string]float64),
		BySkill:    make(map[string]float64),
		ByModel:    make(map[string]float64),
		Period:     period,
	}
}

// AggregatedMetrics represents a complete metrics summary.
type AggregatedMetrics struct {
	Period          TimePeriod        // Time period for these metrics
	TotalExecutions int64             // Total workflow executions
	SuccessCount    int64             // Successful executions
	FailedCount     int64             // Failed executions
	SuccessRate     float64           // Overall success rate (0.0 to 1.0)
	TotalTokens     int64             // Total tokens used
	InputTokens     int64             // Total input tokens
	OutputTokens    int64             // Total output tokens
	TotalCost       float64           // Total cost
	AvgLatency      time.Duration     // Average execution latency
	Providers       []ProviderMetrics // Per-provider metrics
	Skills          []SkillMetrics    // Per-skill metrics
	CacheHits       int64             // Total cache hits
	CacheMisses     int64             // Total cache misses
	CacheHitRate    float64           // Cache hit rate (0.0 to 1.0)
}

// MetricsFilter defines criteria for querying metrics.
type MetricsFilter struct {
	SkillID   string    // Filter by skill ID (empty for all)
	Provider  string    // Filter by provider name (empty for all)
	Model     string    // Filter by model (empty for all)
	Status    string    // Filter by status (empty for all)
	StartDate time.Time // Include metrics from this date (zero for no lower bound)
	EndDate   time.Time // Include metrics until this date (zero for no upper bound)
	Limit     int       // Maximum number of records (0 for no limit)
	Offset    int       // Offset for pagination
}

// DefaultFilter returns a MetricsFilter with sensible defaults.
func DefaultFilter() MetricsFilter {
	return MetricsFilter{
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now(),
		Limit:     100,
	}
}

// WithPeriod sets the time period for the filter.
func (f MetricsFilter) WithPeriod(start, end time.Time) MetricsFilter {
	f.StartDate = start
	f.EndDate = end
	return f
}

// WithSkill sets the skill ID filter.
func (f MetricsFilter) WithSkill(skillID string) MetricsFilter {
	f.SkillID = skillID
	return f
}

// WithProvider sets the provider filter.
func (f MetricsFilter) WithProvider(provider string) MetricsFilter {
	f.Provider = provider
	return f
}

// Last24Hours returns a filter for the last 24 hours.
func Last24Hours() MetricsFilter {
	now := time.Now()
	return MetricsFilter{
		StartDate: now.Add(-24 * time.Hour),
		EndDate:   now,
	}
}

// Last7Days returns a filter for the last 7 days.
func Last7Days() MetricsFilter {
	now := time.Now()
	return MetricsFilter{
		StartDate: now.Add(-7 * 24 * time.Hour),
		EndDate:   now,
	}
}

// Last30Days returns a filter for the last 30 days.
func Last30Days() MetricsFilter {
	now := time.Now()
	return MetricsFilter{
		StartDate: now.Add(-30 * 24 * time.Hour),
		EndDate:   now,
	}
}
