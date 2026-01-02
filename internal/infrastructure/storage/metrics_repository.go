// Package storage provides storage implementations for the application layer ports.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/metrics"
)

// MetricsRepository implements ports.MetricsStoragePort using SQLite.
type MetricsRepository struct {
	db *sql.DB
}

// NewMetricsRepository creates a new MetricsRepository.
func NewMetricsRepository(db *sql.DB) ports.MetricsStoragePort {
	return &MetricsRepository{db: db}
}

// SaveExecution persists an execution record to the database.
func (r *MetricsRepository) SaveExecution(ctx context.Context, exec *metrics.ExecutionRecord) error {
	if exec == nil {
		return fmt.Errorf("execution record is nil")
	}

	query := `
		INSERT INTO execution_records (
			id, skill_id, skill_name, status, input_tokens, output_tokens,
			total_cost, duration_ns, phase_count, cache_hits, cache_misses,
			primary_model, started_at, completed_at, correlation_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		exec.ID,
		exec.SkillID,
		exec.SkillName,
		exec.Status,
		exec.InputTokens,
		exec.OutputTokens,
		exec.TotalCost,
		exec.Duration.Nanoseconds(),
		exec.PhaseCount,
		exec.CacheHits,
		exec.CacheMisses,
		exec.PrimaryModel,
		exec.StartedAt.UTC().Format(time.RFC3339),
		exec.CompletedAt.UTC().Format(time.RFC3339),
		exec.CorrelationID,
	)

	if err != nil {
		return fmt.Errorf("failed to save execution record: %w", err)
	}

	return nil
}

// SavePhaseExecution persists a phase execution record.
func (r *MetricsRepository) SavePhaseExecution(ctx context.Context, phase *metrics.PhaseExecutionRecord) error {
	if phase == nil {
		return fmt.Errorf("phase execution record is nil")
	}

	query := `
		INSERT INTO phase_execution_records (
			id, execution_id, phase_id, phase_name, status, provider, model,
			input_tokens, output_tokens, cost, duration_ns, cache_hit,
			started_at, completed_at, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		phase.ID,
		phase.ExecutionID,
		phase.PhaseID,
		phase.PhaseName,
		phase.Status,
		phase.Provider,
		phase.Model,
		phase.InputTokens,
		phase.OutputTokens,
		phase.Cost,
		phase.Duration.Nanoseconds(),
		phase.CacheHit,
		phase.StartedAt.UTC().Format(time.RFC3339),
		phase.CompletedAt.UTC().Format(time.RFC3339),
		phase.ErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to save phase execution record: %w", err)
	}

	return nil
}

// GetExecutions retrieves execution records matching the filter.
func (r *MetricsRepository) GetExecutions(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.ExecutionRecord, error) {
	query := `
		SELECT id, skill_id, skill_name, status, input_tokens, output_tokens,
			total_cost, duration_ns, phase_count, cache_hits, cache_misses,
			primary_model, started_at, completed_at, correlation_id
		FROM execution_records
		WHERE 1=1
	`
	args := make([]any, 0)

	if filter.SkillID != "" {
		query += " AND skill_id = ?"
		args = append(args, filter.SkillID)
	}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	if !filter.StartDate.IsZero() {
		query += " AND started_at >= ?"
		args = append(args, filter.StartDate.UTC().Format(time.RFC3339))
	}

	if !filter.EndDate.IsZero() {
		query += " AND started_at <= ?"
		args = append(args, filter.EndDate.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY started_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	defer rows.Close()

	var executions []metrics.ExecutionRecord
	for rows.Next() {
		var exec metrics.ExecutionRecord
		var durationNs int64
		var startedAt, completedAt string

		err := rows.Scan(
			&exec.ID,
			&exec.SkillID,
			&exec.SkillName,
			&exec.Status,
			&exec.InputTokens,
			&exec.OutputTokens,
			&exec.TotalCost,
			&durationNs,
			&exec.PhaseCount,
			&exec.CacheHits,
			&exec.CacheMisses,
			&exec.PrimaryModel,
			&startedAt,
			&completedAt,
			&exec.CorrelationID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution record: %w", err)
		}

		exec.Duration = time.Duration(durationNs)
		exec.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		exec.CompletedAt, _ = time.Parse(time.RFC3339, completedAt)

		executions = append(executions, exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating execution records: %w", err)
	}

	return executions, nil
}

// GetAggregatedMetrics retrieves aggregated metrics for the given filter.
func (r *MetricsRepository) GetAggregatedMetrics(ctx context.Context, filter metrics.MetricsFilter) (*metrics.AggregatedMetrics, error) {
	period := metrics.TimePeriod{Start: filter.StartDate, End: filter.EndDate}
	if period.End.IsZero() {
		period.End = time.Now()
	}
	if period.Start.IsZero() {
		period.Start = period.End.Add(-24 * time.Hour)
	}

	result := &metrics.AggregatedMetrics{
		Period: period,
	}

	// Get execution totals
	execQuery := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as success,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost,
			COALESCE(AVG(duration_ns), 0) as avg_duration,
			COALESCE(SUM(cache_hits), 0) as cache_hits,
			COALESCE(SUM(cache_misses), 0) as cache_misses
		FROM execution_records
		WHERE started_at >= ? AND started_at <= ?
	`
	args := []any{
		period.Start.UTC().Format(time.RFC3339),
		period.End.UTC().Format(time.RFC3339),
	}

	if filter.SkillID != "" {
		execQuery += " AND skill_id = ?"
		args = append(args, filter.SkillID)
	}

	var avgDurationNs float64
	err := r.db.QueryRowContext(ctx, execQuery, args...).Scan(
		&result.TotalExecutions,
		&result.SuccessCount,
		&result.FailedCount,
		&result.InputTokens,
		&result.OutputTokens,
		&result.TotalCost,
		&avgDurationNs,
		&result.CacheHits,
		&result.CacheMisses,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query aggregated metrics: %w", err)
	}

	result.TotalTokens = result.InputTokens + result.OutputTokens
	result.AvgLatency = time.Duration(avgDurationNs)

	if result.TotalExecutions > 0 {
		result.SuccessRate = float64(result.SuccessCount) / float64(result.TotalExecutions)
	}

	totalCacheRequests := result.CacheHits + result.CacheMisses
	if totalCacheRequests > 0 {
		result.CacheHitRate = float64(result.CacheHits) / float64(totalCacheRequests)
	}

	// Get provider metrics
	providerMetrics, err := r.GetProviderMetrics(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider metrics: %w", err)
	}
	result.Providers = providerMetrics

	// Get skill metrics
	skillMetrics, err := r.GetSkillMetrics(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill metrics: %w", err)
	}
	result.Skills = skillMetrics

	return result, nil
}

// GetProviderMetrics retrieves aggregated metrics for all providers.
func (r *MetricsRepository) GetProviderMetrics(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.ProviderMetrics, error) {
	period := metrics.TimePeriod{Start: filter.StartDate, End: filter.EndDate}
	if period.End.IsZero() {
		period.End = time.Now()
	}
	if period.Start.IsZero() {
		period.Start = period.End.Add(-24 * time.Hour)
	}

	query := `
		SELECT
			provider,
			COUNT(*) as total_requests,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(SUM(input_tokens), 0) as tokens_input,
			COALESCE(SUM(output_tokens), 0) as tokens_output,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(AVG(duration_ns), 0) as avg_latency,
			COALESCE(MIN(duration_ns), 0) as min_latency,
			COALESCE(MAX(duration_ns), 0) as max_latency,
			SUM(CASE WHEN cache_hit = 1 THEN 1 ELSE 0 END) as cache_hits,
			SUM(CASE WHEN cache_hit = 0 THEN 1 ELSE 0 END) as cache_misses
		FROM phase_execution_records
		WHERE started_at >= ? AND started_at <= ?
		GROUP BY provider
		ORDER BY total_requests DESC
	`

	rows, err := r.db.QueryContext(ctx, query,
		period.Start.UTC().Format(time.RFC3339),
		period.End.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query provider metrics: %w", err)
	}
	defer rows.Close()

	var results []metrics.ProviderMetrics
	for rows.Next() {
		var pm metrics.ProviderMetrics
		var avgLatencyNs, minLatencyNs, maxLatencyNs float64

		err := rows.Scan(
			&pm.Name,
			&pm.TotalRequests,
			&pm.SuccessCount,
			&pm.FailedCount,
			&pm.TokensInput,
			&pm.TokensOutput,
			&pm.TotalCost,
			&avgLatencyNs,
			&minLatencyNs,
			&maxLatencyNs,
			&pm.CacheHits,
			&pm.CacheMisses,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider metrics: %w", err)
		}

		pm.AvgLatency = time.Duration(avgLatencyNs)
		pm.MinLatency = time.Duration(minLatencyNs)
		pm.MaxLatency = time.Duration(maxLatencyNs)
		pm.Period = period

		// Determine provider type
		if pm.Name == "ollama" {
			pm.Type = "local"
		} else {
			pm.Type = "cloud"
		}

		results = append(results, pm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating provider metrics: %w", err)
	}

	return results, nil
}

// GetSkillMetrics retrieves aggregated metrics for all skills.
func (r *MetricsRepository) GetSkillMetrics(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.SkillMetrics, error) {
	period := metrics.TimePeriod{Start: filter.StartDate, End: filter.EndDate}
	if period.End.IsZero() {
		period.End = time.Now()
	}
	if period.Start.IsZero() {
		period.Start = period.End.Add(-24 * time.Hour)
	}

	query := `
		SELECT
			skill_id,
			skill_name,
			COUNT(*) as total_runs,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(AVG(duration_ns), 0) as avg_duration,
			COALESCE(MIN(duration_ns), 0) as min_duration,
			COALESCE(MAX(duration_ns), 0) as max_duration,
			COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost
		FROM execution_records
		WHERE started_at >= ? AND started_at <= ?
		GROUP BY skill_id, skill_name
		ORDER BY total_runs DESC
	`

	rows, err := r.db.QueryContext(ctx, query,
		period.Start.UTC().Format(time.RFC3339),
		period.End.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query skill metrics: %w", err)
	}
	defer rows.Close()

	var results []metrics.SkillMetrics
	for rows.Next() {
		var sm metrics.SkillMetrics
		var avgDurationNs, minDurationNs, maxDurationNs float64

		err := rows.Scan(
			&sm.SkillID,
			&sm.SkillName,
			&sm.TotalRuns,
			&sm.SuccessCount,
			&sm.FailedCount,
			&avgDurationNs,
			&minDurationNs,
			&maxDurationNs,
			&sm.TotalTokens,
			&sm.TotalCost,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan skill metrics: %w", err)
		}

		sm.AvgDuration = time.Duration(avgDurationNs)
		sm.MinDuration = time.Duration(minDurationNs)
		sm.MaxDuration = time.Duration(maxDurationNs)
		sm.Period = period

		if sm.TotalRuns > 0 {
			sm.SuccessRate = float64(sm.SuccessCount) / float64(sm.TotalRuns)
		}

		results = append(results, sm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating skill metrics: %w", err)
	}

	return results, nil
}

// GetCostSummary retrieves aggregated cost data based on the provided filter.
func (r *MetricsRepository) GetCostSummary(ctx context.Context, filter metrics.MetricsFilter) (*metrics.CostSummary, error) {
	period := metrics.TimePeriod{Start: filter.StartDate, End: filter.EndDate}
	if period.End.IsZero() {
		period.End = time.Now()
	}
	if period.Start.IsZero() {
		period.Start = period.End.Add(-24 * time.Hour)
	}

	summary := metrics.NewCostSummary(period)

	// Get total cost and tokens from executions
	totalQuery := `
		SELECT
			COALESCE(SUM(total_cost), 0) as total_cost,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens
		FROM execution_records
		WHERE started_at >= ? AND started_at <= ?
	`
	args := []any{
		period.Start.UTC().Format(time.RFC3339),
		period.End.UTC().Format(time.RFC3339),
	}

	if filter.SkillID != "" {
		totalQuery += " AND skill_id = ?"
		args = append(args, filter.SkillID)
	}

	err := r.db.QueryRowContext(ctx, totalQuery, args...).Scan(
		&summary.TotalCost,
		&summary.InputTokens,
		&summary.OutputTokens,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query total cost: %w", err)
	}

	summary.TotalTokens = summary.InputTokens + summary.OutputTokens

	// Get cost by provider
	providerQuery := `
		SELECT provider, COALESCE(SUM(cost), 0) as cost
		FROM phase_execution_records
		WHERE started_at >= ? AND started_at <= ?
		GROUP BY provider
	`
	providerRows, err := r.db.QueryContext(ctx, providerQuery,
		period.Start.UTC().Format(time.RFC3339),
		period.End.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query cost by provider: %w", err)
	}
	defer providerRows.Close()

	for providerRows.Next() {
		var provider string
		var cost float64
		if err := providerRows.Scan(&provider, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan provider cost: %w", err)
		}
		summary.ByProvider[provider] = cost
	}

	// Get cost by skill
	skillQuery := `
		SELECT skill_id, COALESCE(SUM(total_cost), 0) as cost
		FROM execution_records
		WHERE started_at >= ? AND started_at <= ?
		GROUP BY skill_id
	`
	skillRows, err := r.db.QueryContext(ctx, skillQuery,
		period.Start.UTC().Format(time.RFC3339),
		period.End.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query cost by skill: %w", err)
	}
	defer skillRows.Close()

	for skillRows.Next() {
		var skillID string
		var cost float64
		if err := skillRows.Scan(&skillID, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan skill cost: %w", err)
		}
		summary.BySkill[skillID] = cost
	}

	// Get cost by model
	modelQuery := `
		SELECT model, COALESCE(SUM(cost), 0) as cost
		FROM phase_execution_records
		WHERE started_at >= ? AND started_at <= ?
		GROUP BY model
	`
	modelRows, err := r.db.QueryContext(ctx, modelQuery,
		period.Start.UTC().Format(time.RFC3339),
		period.End.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query cost by model: %w", err)
	}
	defer modelRows.Close()

	for modelRows.Next() {
		var model string
		var cost float64
		if err := modelRows.Scan(&model, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan model cost: %w", err)
		}
		summary.ByModel[model] = cost
	}

	return summary, nil
}

// Ensure MetricsRepository implements MetricsStoragePort.
var _ ports.MetricsStoragePort = (*MetricsRepository)(nil)
