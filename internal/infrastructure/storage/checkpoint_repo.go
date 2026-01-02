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
	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
	domainErrors "github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// Compile-time check that CheckpointRepository implements CheckpointStateStoragePort.
var _ ports.CheckpointStateStoragePort = (*CheckpointRepository)(nil)

// CheckpointRepository implements CheckpointStateStoragePort using SQLite.
type CheckpointRepository struct {
	db *sql.DB
}

// NewCheckpointRepository creates a new checkpoint repository.
func NewCheckpointRepository(db *sql.DB) *CheckpointRepository {
	return &CheckpointRepository{db: db}
}

// Create persists a new checkpoint to storage.
func (r *CheckpointRepository) Create(ctx context.Context, checkpoint *domainContext.Checkpoint) error {
	if err := checkpoint.Validate(); err != nil {
		return err
	}

	filesJSON, err := json.Marshal(checkpoint.FilesModified())
	if err != nil {
		return fmt.Errorf("failed to marshal files: %w", err)
	}

	decisionsJSON, err := json.Marshal(checkpoint.Decisions())
	if err != nil {
		return fmt.Errorf("failed to marshal decisions: %w", err)
	}

	query := `
		INSERT INTO checkpoints (id, workspace_id, session_id, summary, details, files_modified, decisions, machine_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		checkpoint.ID(),
		checkpoint.WorkspaceID(),
		checkpoint.SessionID(),
		checkpoint.Summary(),
		nullableString(checkpoint.Details()),
		string(filesJSON),
		string(decisionsJSON),
		nullableString(checkpoint.MachineID()),
		checkpoint.CreatedAt().Format(time.RFC3339),
	)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return domainErrors.NewError(domainErrors.CodeValidation, "checkpoint already exists", err)
		}
		return fmt.Errorf("failed to create checkpoint: %w", err)
	}

	return nil
}

// Get retrieves a checkpoint by its unique identifier.
func (r *CheckpointRepository) Get(ctx context.Context, id string) (*domainContext.Checkpoint, error) {
	query := `
		SELECT id, workspace_id, session_id, summary, details, files_modified, decisions, machine_id, created_at
		FROM checkpoints
		WHERE id = ?
	`

	checkpoint, err := r.scanCheckpointRow(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("checkpoint not found: %s", id), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	return checkpoint, nil
}

// GetBySession retrieves all checkpoints for a specific session.
func (r *CheckpointRepository) GetBySession(ctx context.Context, sessionID string) ([]*domainContext.Checkpoint, error) {
	query := `
		SELECT id, workspace_id, session_id, summary, details, files_modified, decisions, machine_id, created_at
		FROM checkpoints
		WHERE session_id = ?
		ORDER BY created_at DESC
	`

	return r.queryCheckpoints(ctx, query, sessionID)
}

// GetByWorkspace retrieves all checkpoints for a specific workspace.
func (r *CheckpointRepository) GetByWorkspace(ctx context.Context, workspaceID string) ([]*domainContext.Checkpoint, error) {
	query := `
		SELECT id, workspace_id, session_id, summary, details, files_modified, decisions, machine_id, created_at
		FROM checkpoints
		WHERE workspace_id = ?
		ORDER BY created_at DESC
	`

	return r.queryCheckpoints(ctx, query, workspaceID)
}

// GetLatest retrieves the most recent checkpoint for a session.
func (r *CheckpointRepository) GetLatest(ctx context.Context, sessionID string) (*domainContext.Checkpoint, error) {
	query := `
		SELECT id, workspace_id, session_id, summary, details, files_modified, decisions, machine_id, created_at
		FROM checkpoints
		WHERE session_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	checkpoint, err := r.scanCheckpointRow(r.db.QueryRowContext(ctx, query, sessionID))
	if err == sql.ErrNoRows {
		return nil, nil // No checkpoints found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest checkpoint: %w", err)
	}

	return checkpoint, nil
}

// List returns checkpoints matching the filter criteria.
func (r *CheckpointRepository) List(ctx context.Context, filter *ports.CheckpointFilter) ([]*domainContext.Checkpoint, error) {
	query := `
		SELECT id, workspace_id, session_id, summary, details, files_modified, decisions, machine_id, created_at
		FROM checkpoints
		WHERE 1=1
	`
	args := []any{}

	if filter != nil {
		if filter.SessionID != "" {
			query += " AND session_id = ?"
			args = append(args, filter.SessionID)
		}
		if filter.WorkspaceID != "" {
			query += " AND workspace_id = ?"
			args = append(args, filter.WorkspaceID)
		}
		if filter.MachineID != "" {
			query += " AND machine_id = ?"
			args = append(args, filter.MachineID)
		}
		if !filter.CreatedAfter.IsZero() {
			query += " AND created_at >= ?"
			args = append(args, filter.CreatedAfter.Format(time.RFC3339))
		}
		if !filter.CreatedBefore.IsZero() {
			query += " AND created_at <= ?"
			args = append(args, filter.CreatedBefore.Format(time.RFC3339))
		}
	}

	query += " ORDER BY created_at DESC"

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

// Update persists changes to an existing checkpoint.
func (r *CheckpointRepository) Update(ctx context.Context, checkpoint *domainContext.Checkpoint) error {
	if err := checkpoint.Validate(); err != nil {
		return err
	}

	filesJSON, err := json.Marshal(checkpoint.FilesModified())
	if err != nil {
		return fmt.Errorf("failed to marshal files: %w", err)
	}

	decisionsJSON, err := json.Marshal(checkpoint.Decisions())
	if err != nil {
		return fmt.Errorf("failed to marshal decisions: %w", err)
	}

	query := `
		UPDATE checkpoints
		SET workspace_id = ?, session_id = ?, summary = ?, details = ?, files_modified = ?, decisions = ?, machine_id = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		checkpoint.WorkspaceID(),
		checkpoint.SessionID(),
		checkpoint.Summary(),
		nullableString(checkpoint.Details()),
		string(filesJSON),
		string(decisionsJSON),
		nullableString(checkpoint.MachineID()),
		checkpoint.ID(),
	)

	if err != nil {
		return fmt.Errorf("failed to update checkpoint: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("checkpoint not found: %s", checkpoint.ID()), nil)
	}

	return nil
}

// Delete removes a checkpoint from storage.
func (r *CheckpointRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM checkpoints WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("checkpoint not found: %s", id), nil)
	}

	return nil
}

// DeleteBySession removes all checkpoints for a specific session.
func (r *CheckpointRepository) DeleteBySession(ctx context.Context, sessionID string) (int, error) {
	query := `DELETE FROM checkpoints WHERE session_id = ?`

	result, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete checkpoints by session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check delete result: %w", err)
	}

	return int(rows), nil
}

// queryCheckpoints executes a query and returns multiple checkpoints.
func (r *CheckpointRepository) queryCheckpoints(ctx context.Context, query string, args ...any) ([]*domainContext.Checkpoint, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []*domainContext.Checkpoint
	for rows.Next() {
		checkpoint, err := r.scanCheckpointRows(rows)
		if err != nil {
			return nil, err
		}
		checkpoints = append(checkpoints, checkpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating checkpoints: %w", err)
	}

	return checkpoints, nil
}

// scanCheckpointRow scans a single row into a checkpoint.
func (r *CheckpointRepository) scanCheckpointRow(row *sql.Row) (*domainContext.Checkpoint, error) {
	var (
		id, workspaceID, sessionID, summary string
		details, filesJSON, decisionsJSON   sql.NullString
		machineID                           sql.NullString
		createdAt                           string
	)

	err := row.Scan(
		&id, &workspaceID, &sessionID, &summary, &details,
		&filesJSON, &decisionsJSON, &machineID, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	return buildCheckpoint(id, workspaceID, sessionID, summary, details, filesJSON, decisionsJSON, machineID)
}

// scanCheckpointRows scans rows into a checkpoint.
func (r *CheckpointRepository) scanCheckpointRows(rows *sql.Rows) (*domainContext.Checkpoint, error) {
	var (
		id, workspaceID, sessionID, summary string
		details, filesJSON, decisionsJSON   sql.NullString
		machineID                           sql.NullString
		createdAt                           string
	)

	err := rows.Scan(
		&id, &workspaceID, &sessionID, &summary, &details,
		&filesJSON, &decisionsJSON, &machineID, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan checkpoint: %w", err)
	}

	return buildCheckpoint(id, workspaceID, sessionID, summary, details, filesJSON, decisionsJSON, machineID)
}

// buildCheckpoint constructs a Checkpoint domain entity from database fields.
func buildCheckpoint(
	id, workspaceID, sessionID, summary string,
	details, filesJSON, decisionsJSON, machineID sql.NullString,
) (*domainContext.Checkpoint, error) {
	checkpoint, err := domainContext.NewCheckpoint(id, workspaceID, sessionID, summary)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %w", err)
	}

	if details.Valid {
		checkpoint.SetDetails(details.String)
	}
	if machineID.Valid {
		checkpoint.SetMachineID(machineID.String)
	}

	// Unmarshal files
	if filesJSON.Valid && filesJSON.String != "" && filesJSON.String != "null" {
		var files []string
		if err := json.Unmarshal([]byte(filesJSON.String), &files); err != nil {
			return nil, fmt.Errorf("failed to unmarshal files: %w", err)
		}
		checkpoint.SetFiles(files)
	}

	// Unmarshal decisions
	if decisionsJSON.Valid && decisionsJSON.String != "" && decisionsJSON.String != "null" {
		var decisions map[string]string
		if err := json.Unmarshal([]byte(decisionsJSON.String), &decisions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal decisions: %w", err)
		}
		checkpoint.SetDecisions(decisions)
	}

	return checkpoint, nil
}
