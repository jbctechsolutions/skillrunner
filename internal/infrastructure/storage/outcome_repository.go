package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/outcome"
)

// OutcomeRepository implements ports.OutcomeStoragePort using SQLite.
type OutcomeRepository struct {
	db *sql.DB
}

// NewOutcomeRepository creates a new OutcomeRepository.
func NewOutcomeRepository(db *sql.DB) ports.OutcomeStoragePort {
	return &OutcomeRepository{db: db}
}

// Record persists a single phase outcome.
func (r *OutcomeRepository) Record(ctx context.Context, o *outcome.Outcome) error {
	if o == nil {
		return fmt.Errorf("outcome is nil")
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO outcomes (
			id, skill_id, skill_name, phase_id, phase_name,
			profile, model, success, quality_score, retry_count,
			duration_ms, recorded_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.ID, o.SkillID, o.SkillName, o.PhaseID, o.PhaseName,
		o.Profile, o.Model, o.Success, o.QualityScore, o.RetryCount,
		o.DurationMS, o.RecordedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to record outcome: %w", err)
	}

	return nil
}

// GetBySkill retrieves recent outcomes for a skill (most recent first).
func (r *OutcomeRepository) GetBySkill(ctx context.Context, skillID string, limit int) ([]outcome.Outcome, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, skill_id, skill_name, phase_id, phase_name,
		       profile, model, success, quality_score, retry_count,
		       duration_ms, recorded_at
		FROM outcomes
		WHERE skill_id = ?
		ORDER BY recorded_at DESC
		LIMIT ?`, skillID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query outcomes: %w", err)
	}
	defer rows.Close()

	var outcomes []outcome.Outcome
	for rows.Next() {
		var o outcome.Outcome
		var recordedAt string
		err := rows.Scan(
			&o.ID, &o.SkillID, &o.SkillName, &o.PhaseID, &o.PhaseName,
			&o.Profile, &o.Model, &o.Success, &o.QualityScore, &o.RetryCount,
			&o.DurationMS, &recordedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outcome: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, recordedAt); err == nil {
			o.RecordedAt = t
		}
		outcomes = append(outcomes, o)
	}

	return outcomes, rows.Err()
}

// GetStats returns aggregated per-phase per-profile statistics for a skill.
func (r *OutcomeRepository) GetStats(ctx context.Context, skillID string) ([]outcome.ProfileStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			skill_id,
			phase_id,
			phase_name,
			profile,
			model,
			COUNT(*) AS total_runs,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS success_count,
			AVG(CASE WHEN quality_score > 0 THEN quality_score ELSE NULL END) AS avg_quality,
			AVG(retry_count) AS avg_retries,
			AVG(duration_ms) AS avg_duration_ms
		FROM outcomes
		WHERE skill_id = ?
		GROUP BY skill_id, phase_id, phase_name, profile, model
		ORDER BY phase_id, profile`, skillID)
	if err != nil {
		return nil, fmt.Errorf("failed to query outcome stats: %w", err)
	}
	defer rows.Close()

	var stats []outcome.ProfileStats
	for rows.Next() {
		var s outcome.ProfileStats
		var avgQuality sql.NullFloat64
		var avgDuration sql.NullFloat64
		err := rows.Scan(
			&s.SkillID, &s.PhaseID, &s.PhaseName,
			&s.Profile, &s.Model,
			&s.TotalRuns, &s.SuccessCount,
			&avgQuality, &s.AvgRetries, &avgDuration,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		if s.TotalRuns > 0 {
			s.SuccessRate = float64(s.SuccessCount) / float64(s.TotalRuns)
		}
		if avgQuality.Valid {
			s.AvgQuality = avgQuality.Float64
		}
		if avgDuration.Valid {
			s.AvgDurationMS = int64(avgDuration.Float64)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// Reset deletes all outcome history for a skill.
func (r *OutcomeRepository) Reset(ctx context.Context, skillID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM outcomes WHERE skill_id = ?`, skillID)
	if err != nil {
		return fmt.Errorf("failed to reset outcomes: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("no outcomes found for skill %q", skillID)
	}
	return nil
}
