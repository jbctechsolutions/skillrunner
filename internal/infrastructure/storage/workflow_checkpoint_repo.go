// Package storage provides SQLite-based storage implementations for state management.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainErrors "github.com/jbctechsolutions/skillrunner/internal/domain/errors"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// Compile-time check that WorkflowCheckpointRepository implements WorkflowCheckpointPort.
var _ ports.WorkflowCheckpointPort = (*WorkflowCheckpointRepository)(nil)

// WorkflowCheckpointRepository implements WorkflowCheckpointPort using SQLite.
type WorkflowCheckpointRepository struct {
	db *sql.DB
}

// NewWorkflowCheckpointRepository creates a new workflow checkpoint repository.
func NewWorkflowCheckpointRepository(db *sql.DB) *WorkflowCheckpointRepository {
	return &WorkflowCheckpointRepository{db: db}
}

// Create persists a new workflow checkpoint to storage.
func (r *WorkflowCheckpointRepository) Create(ctx context.Context, checkpoint *workflow.WorkflowCheckpoint) error {
	if err := checkpoint.Validate(); err != nil {
		return err
	}

	phaseResultsJSON, err := json.Marshal(checkpoint.PhaseResults())
	if err != nil {
		return fmt.Errorf("failed to marshal phase results: %w", err)
	}

	phaseOutputsJSON, err := json.Marshal(checkpoint.PhaseOutputs())
	if err != nil {
		return fmt.Errorf("failed to marshal phase outputs: %w", err)
	}

	query := `
		INSERT INTO workflow_checkpoints (
			id, execution_id, skill_id, skill_name, input, input_hash,
			completed_batch, total_batches, phase_results, phase_outputs,
			status, input_tokens, output_tokens, machine_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		checkpoint.ID(),
		checkpoint.ExecutionID(),
		checkpoint.SkillID(),
		checkpoint.SkillName(),
		checkpoint.Input(),
		checkpoint.InputHash(),
		checkpoint.CompletedBatch(),
		checkpoint.TotalBatches(),
		string(phaseResultsJSON),
		string(phaseOutputsJSON),
		string(checkpoint.Status()),
		checkpoint.InputTokens(),
		checkpoint.OutputTokens(),
		nullableString(checkpoint.MachineID()),
		checkpoint.CreatedAt().Format(time.RFC3339),
		checkpoint.UpdatedAt().Format(time.RFC3339),
	)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return domainErrors.NewError(domainErrors.CodeValidation, "workflow checkpoint already exists", err)
		}
		return fmt.Errorf("failed to create workflow checkpoint: %w", err)
	}

	return nil
}

// Get retrieves a checkpoint by its unique identifier.
func (r *WorkflowCheckpointRepository) Get(ctx context.Context, id string) (*workflow.WorkflowCheckpoint, error) {
	query := `
		SELECT id, execution_id, skill_id, skill_name, input, input_hash,
			   completed_batch, total_batches, phase_results, phase_outputs,
			   status, input_tokens, output_tokens, machine_id, created_at, updated_at
		FROM workflow_checkpoints
		WHERE id = ?
	`

	checkpoint, err := r.scanRow(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workflow checkpoint not found: %s", id), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow checkpoint: %w", err)
	}

	return checkpoint, nil
}

// GetLatestInProgress retrieves the most recent in-progress checkpoint for a skill/input combination.
func (r *WorkflowCheckpointRepository) GetLatestInProgress(ctx context.Context, skillID string, inputHash string) (*workflow.WorkflowCheckpoint, error) {
	query := `
		SELECT id, execution_id, skill_id, skill_name, input, input_hash,
			   completed_batch, total_batches, phase_results, phase_outputs,
			   status, input_tokens, output_tokens, machine_id, created_at, updated_at
		FROM workflow_checkpoints
		WHERE skill_id = ? AND input_hash = ? AND status = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`

	checkpoint, err := r.scanRow(r.db.QueryRowContext(ctx, query, skillID, inputHash, string(workflow.CheckpointStatusInProgress)))
	if err == sql.ErrNoRows {
		return nil, nil // No in-progress checkpoint found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest in-progress checkpoint: %w", err)
	}

	return checkpoint, nil
}

// GetByExecutionID retrieves all checkpoints for an execution.
func (r *WorkflowCheckpointRepository) GetByExecutionID(ctx context.Context, executionID string) ([]*workflow.WorkflowCheckpoint, error) {
	query := `
		SELECT id, execution_id, skill_id, skill_name, input, input_hash,
			   completed_batch, total_batches, phase_results, phase_outputs,
			   status, input_tokens, output_tokens, machine_id, created_at, updated_at
		FROM workflow_checkpoints
		WHERE execution_id = ?
		ORDER BY updated_at DESC
	`

	return r.queryCheckpoints(ctx, query, executionID)
}

// Update persists changes to an existing checkpoint.
func (r *WorkflowCheckpointRepository) Update(ctx context.Context, checkpoint *workflow.WorkflowCheckpoint) error {
	if err := checkpoint.Validate(); err != nil {
		return err
	}

	phaseResultsJSON, err := json.Marshal(checkpoint.PhaseResults())
	if err != nil {
		return fmt.Errorf("failed to marshal phase results: %w", err)
	}

	phaseOutputsJSON, err := json.Marshal(checkpoint.PhaseOutputs())
	if err != nil {
		return fmt.Errorf("failed to marshal phase outputs: %w", err)
	}

	query := `
		UPDATE workflow_checkpoints
		SET completed_batch = ?, phase_results = ?, phase_outputs = ?,
			status = ?, input_tokens = ?, output_tokens = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		checkpoint.CompletedBatch(),
		string(phaseResultsJSON),
		string(phaseOutputsJSON),
		string(checkpoint.Status()),
		checkpoint.InputTokens(),
		checkpoint.OutputTokens(),
		checkpoint.UpdatedAt().Format(time.RFC3339),
		checkpoint.ID(),
	)

	if err != nil {
		return fmt.Errorf("failed to update workflow checkpoint: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workflow checkpoint not found: %s", checkpoint.ID()), nil)
	}

	return nil
}

// List returns all checkpoints matching the optional filter criteria.
func (r *WorkflowCheckpointRepository) List(ctx context.Context, filter *ports.WorkflowCheckpointFilter) ([]*workflow.WorkflowCheckpoint, error) {
	query := `
		SELECT id, execution_id, skill_id, skill_name, input, input_hash,
			   completed_batch, total_batches, phase_results, phase_outputs,
			   status, input_tokens, output_tokens, machine_id, created_at, updated_at
		FROM workflow_checkpoints
		WHERE 1=1
	`
	args := []any{}

	if filter != nil {
		if filter.SkillID != "" {
			query += " AND skill_id = ?"
			args = append(args, filter.SkillID)
		}
		if filter.ExecutionID != "" {
			query += " AND execution_id = ?"
			args = append(args, filter.ExecutionID)
		}
		if filter.MachineID != "" {
			query += " AND machine_id = ?"
			args = append(args, filter.MachineID)
		}
		if len(filter.Status) > 0 {
			placeholders := make([]string, len(filter.Status))
			for i, s := range filter.Status {
				placeholders[i] = "?"
				args = append(args, string(s))
			}
			query += fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ","))
		}
		if !filter.CreatedAfter.IsZero() {
			query += " AND created_at >= ?"
			args = append(args, filter.CreatedAfter.Format(time.RFC3339))
		}
		if !filter.CreatedBefore.IsZero() {
			query += " AND created_at < ?"
			args = append(args, filter.CreatedBefore.Format(time.RFC3339))
		}
	}

	query += " ORDER BY updated_at DESC"

	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	return r.queryCheckpoints(ctx, query, args...)
}

// Delete removes a checkpoint from storage.
func (r *WorkflowCheckpointRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workflow_checkpoints WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow checkpoint: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workflow checkpoint not found: %s", id), nil)
	}

	return nil
}

// DeleteByExecutionID removes all checkpoints for an execution.
func (r *WorkflowCheckpointRepository) DeleteByExecutionID(ctx context.Context, executionID string) (int, error) {
	query := `DELETE FROM workflow_checkpoints WHERE execution_id = ?`

	result, err := r.db.ExecContext(ctx, query, executionID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete workflow checkpoints by execution: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check delete result: %w", err)
	}

	return int(rows), nil
}

// MarkAbandoned marks all in-progress checkpoints for a machine as abandoned.
func (r *WorkflowCheckpointRepository) MarkAbandoned(ctx context.Context, machineID string) (int, error) {
	query := `
		UPDATE workflow_checkpoints
		SET status = ?, updated_at = ?
		WHERE machine_id = ? AND status = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		string(workflow.CheckpointStatusAbandoned),
		time.Now().Format(time.RFC3339),
		machineID,
		string(workflow.CheckpointStatusInProgress),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to mark checkpoints as abandoned: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check update result: %w", err)
	}

	return int(rows), nil
}

// Cleanup removes checkpoints older than the specified duration.
func (r *WorkflowCheckpointRepository) Cleanup(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	query := `
		DELETE FROM workflow_checkpoints
		WHERE created_at < ? AND status IN (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		cutoff.Format(time.RFC3339),
		string(workflow.CheckpointStatusCompleted),
		string(workflow.CheckpointStatusFailed),
		string(workflow.CheckpointStatusAbandoned),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old checkpoints: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check cleanup result: %w", err)
	}

	return int(rows), nil
}

// queryCheckpoints executes a query and returns multiple checkpoints.
func (r *WorkflowCheckpointRepository) queryCheckpoints(ctx context.Context, query string, args ...any) ([]*workflow.WorkflowCheckpoint, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []*workflow.WorkflowCheckpoint
	for rows.Next() {
		checkpoint, err := r.scanRows(rows)
		if err != nil {
			return nil, err
		}
		checkpoints = append(checkpoints, checkpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workflow checkpoints: %w", err)
	}

	return checkpoints, nil
}

// scanRow scans a single row into a workflow checkpoint.
func (r *WorkflowCheckpointRepository) scanRow(row *sql.Row) (*workflow.WorkflowCheckpoint, error) {
	var (
		id, executionID, skillID, skillName, input, inputHash string
		completedBatch, totalBatches                          int
		phaseResultsJSON, phaseOutputsJSON                    sql.NullString
		status                                                string
		inputTokens, outputTokens                             int
		machineID                                             sql.NullString
		createdAt, updatedAt                                  string
	)

	err := row.Scan(
		&id, &executionID, &skillID, &skillName, &input, &inputHash,
		&completedBatch, &totalBatches, &phaseResultsJSON, &phaseOutputsJSON,
		&status, &inputTokens, &outputTokens, &machineID, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return buildWorkflowCheckpoint(
		id, executionID, skillID, skillName, input, inputHash,
		completedBatch, totalBatches, phaseResultsJSON, phaseOutputsJSON,
		status, inputTokens, outputTokens, machineID, createdAt, updatedAt,
	)
}

// scanRows scans rows into a workflow checkpoint.
func (r *WorkflowCheckpointRepository) scanRows(rows *sql.Rows) (*workflow.WorkflowCheckpoint, error) {
	var (
		id, executionID, skillID, skillName, input, inputHash string
		completedBatch, totalBatches                          int
		phaseResultsJSON, phaseOutputsJSON                    sql.NullString
		status                                                string
		inputTokens, outputTokens                             int
		machineID                                             sql.NullString
		createdAt, updatedAt                                  string
	)

	err := rows.Scan(
		&id, &executionID, &skillID, &skillName, &input, &inputHash,
		&completedBatch, &totalBatches, &phaseResultsJSON, &phaseOutputsJSON,
		&status, &inputTokens, &outputTokens, &machineID, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan workflow checkpoint: %w", err)
	}

	return buildWorkflowCheckpoint(
		id, executionID, skillID, skillName, input, inputHash,
		completedBatch, totalBatches, phaseResultsJSON, phaseOutputsJSON,
		status, inputTokens, outputTokens, machineID, createdAt, updatedAt,
	)
}

// buildWorkflowCheckpoint constructs a WorkflowCheckpoint domain entity from database fields.
func buildWorkflowCheckpoint(
	id, executionID, skillID, skillName, input, inputHash string,
	completedBatch, totalBatches int,
	phaseResultsJSON, phaseOutputsJSON sql.NullString,
	status string,
	inputTokens, outputTokens int,
	machineID sql.NullString,
	createdAtStr, updatedAtStr string,
) (*workflow.WorkflowCheckpoint, error) {
	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	// Unmarshal phase results
	var phaseResults map[string]*workflow.PhaseResultData
	if phaseResultsJSON.Valid && phaseResultsJSON.String != "" && phaseResultsJSON.String != "null" {
		if err := json.Unmarshal([]byte(phaseResultsJSON.String), &phaseResults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal phase results: %w", err)
		}
	}

	// Unmarshal phase outputs
	var phaseOutputs map[string]string
	if phaseOutputsJSON.Valid && phaseOutputsJSON.String != "" && phaseOutputsJSON.String != "null" {
		if err := json.Unmarshal([]byte(phaseOutputsJSON.String), &phaseOutputs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal phase outputs: %w", err)
		}
	}

	// Parse machine ID
	var machine string
	if machineID.Valid {
		machine = machineID.String
	}

	// Use ReconstructCheckpoint to build the entity
	checkpoint := workflow.ReconstructCheckpoint(
		id, executionID, skillID, skillName, input, inputHash,
		completedBatch, totalBatches,
		phaseResults, phaseOutputs,
		workflow.CheckpointStatus(status),
		inputTokens, outputTokens,
		machine,
		createdAt, updatedAt,
	)

	return checkpoint, nil
}
