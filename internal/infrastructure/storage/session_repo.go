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
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// Compile-time check that SessionRepository implements SessionStateStoragePort.
var _ ports.SessionStateStoragePort = (*SessionRepository)(nil)

// SessionRepository implements SessionStateStoragePort using SQLite.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create persists a new session to storage.
func (r *SessionRepository) Create(ctx context.Context, sess *session.Session) error {
	if sess.ID == "" {
		return domainErrors.NewError(domainErrors.CodeValidation, "session ID is required", nil)
	}
	if sess.WorkspaceID == "" {
		return domainErrors.NewError(domainErrors.CodeValidation, "workspace ID is required", nil)
	}

	metadataJSON, err := json.Marshal(sess.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	tokenUsageJSON, err := json.Marshal(sess.TokenUsage)
	if err != nil {
		return fmt.Errorf("failed to marshal token usage: %w", err)
	}

	contextJSON, err := json.Marshal(sess.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	query := `
		INSERT INTO sessions (id, workspace_id, backend, model, status, started_at, ended_at, machine_id, pid, tmux_session, metadata, token_usage, context)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var endedAt sql.NullString
	if sess.EndedAt != nil {
		endedAt = sql.NullString{String: sess.EndedAt.Format(time.RFC3339), Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		sess.ID,
		sess.WorkspaceID,
		nullableString(sess.Backend),
		nullableString(sess.Model),
		string(sess.Status),
		sess.StartedAt.Format(time.RFC3339),
		endedAt,
		nullableString(sess.MachineID),
		nullableInt(sess.ProcessID),
		nullableString(sess.TmuxSession),
		string(metadataJSON),
		string(tokenUsageJSON),
		string(contextJSON),
	)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return domainErrors.NewError(domainErrors.CodeValidation, "session already exists", err)
		}
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// Get retrieves a session by its unique identifier.
func (r *SessionRepository) Get(ctx context.Context, id string) (*session.Session, error) {
	query := `
		SELECT id, workspace_id, backend, model, status, started_at, ended_at, machine_id, pid, tmux_session, metadata, token_usage, context
		FROM sessions
		WHERE id = ?
	`

	sess, err := r.scanSessionRow(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("session not found: %s", id), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return sess, nil
}

// GetByWorkspace retrieves all sessions associated with a workspace.
func (r *SessionRepository) GetByWorkspace(ctx context.Context, workspaceID string) ([]*session.Session, error) {
	query := `
		SELECT id, workspace_id, backend, model, status, started_at, ended_at, machine_id, pid, tmux_session, metadata, token_usage, context
		FROM sessions
		WHERE workspace_id = ?
		ORDER BY started_at DESC
	`

	return r.querySessions(ctx, query, workspaceID)
}

// GetActive retrieves all currently active sessions.
func (r *SessionRepository) GetActive(ctx context.Context) ([]*session.Session, error) {
	query := `
		SELECT id, workspace_id, backend, model, status, started_at, ended_at, machine_id, pid, tmux_session, metadata, token_usage, context
		FROM sessions
		WHERE status IN (?, ?, ?)
		ORDER BY started_at DESC
	`

	return r.querySessions(ctx, query, string(session.StatusActive), string(session.StatusIdle), string(session.StatusDetached))
}

// GetActiveByWorkspace retrieves the active session for a specific workspace.
func (r *SessionRepository) GetActiveByWorkspace(ctx context.Context, workspaceID string) (*session.Session, error) {
	query := `
		SELECT id, workspace_id, backend, model, status, started_at, ended_at, machine_id, pid, tmux_session, metadata, token_usage, context
		FROM sessions
		WHERE workspace_id = ? AND status IN (?, ?, ?)
		ORDER BY started_at DESC
		LIMIT 1
	`

	sess, err := r.scanSessionRow(r.db.QueryRowContext(ctx, query, workspaceID,
		string(session.StatusActive), string(session.StatusIdle), string(session.StatusDetached)))
	if err == sql.ErrNoRows {
		return nil, nil // No active session
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	return sess, nil
}

// List returns sessions matching the filter criteria.
func (r *SessionRepository) List(ctx context.Context, filter session.Filter) ([]*session.Session, error) {
	query := `
		SELECT id, workspace_id, backend, model, status, started_at, ended_at, machine_id, pid, tmux_session, metadata, token_usage, context
		FROM sessions
		WHERE 1=1
	`
	args := []any{}

	if filter.WorkspaceID != "" {
		query += " AND workspace_id = ?"
		args = append(args, filter.WorkspaceID)
	}
	if filter.Backend != "" {
		query += " AND backend = ?"
		args = append(args, filter.Backend)
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
		query += " AND status IN (" + strings.Join(placeholders, ", ") + ")"
	}

	query += " ORDER BY started_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	return r.querySessions(ctx, query, args...)
}

// Update persists changes to an existing session.
func (r *SessionRepository) Update(ctx context.Context, sess *session.Session) error {
	if sess.ID == "" {
		return domainErrors.NewError(domainErrors.CodeValidation, "session ID is required", nil)
	}

	metadataJSON, err := json.Marshal(sess.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	tokenUsageJSON, err := json.Marshal(sess.TokenUsage)
	if err != nil {
		return fmt.Errorf("failed to marshal token usage: %w", err)
	}

	contextJSON, err := json.Marshal(sess.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	query := `
		UPDATE sessions
		SET workspace_id = ?, backend = ?, model = ?, status = ?, started_at = ?, ended_at = ?, machine_id = ?, pid = ?, tmux_session = ?, metadata = ?, token_usage = ?, context = ?
		WHERE id = ?
	`

	var endedAt sql.NullString
	if sess.EndedAt != nil {
		endedAt = sql.NullString{String: sess.EndedAt.Format(time.RFC3339), Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		sess.WorkspaceID,
		nullableString(sess.Backend),
		nullableString(sess.Model),
		string(sess.Status),
		sess.StartedAt.Format(time.RFC3339),
		endedAt,
		nullableString(sess.MachineID),
		nullableInt(sess.ProcessID),
		nullableString(sess.TmuxSession),
		string(metadataJSON),
		string(tokenUsageJSON),
		string(contextJSON),
		sess.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("session not found: %s", sess.ID), nil)
	}

	return nil
}

// UpdateStatus atomically updates the status of a session.
func (r *SessionRepository) UpdateStatus(ctx context.Context, id string, status session.Status) error {
	var query string
	var args []any

	// If terminal state, also set ended_at
	if status == session.StatusCompleted || status == session.StatusFailed || status == session.StatusKilled {
		query = `
			UPDATE sessions
			SET status = ?, ended_at = ?
			WHERE id = ?
		`
		args = []any{string(status), time.Now().Format(time.RFC3339), id}
	} else {
		query = `
			UPDATE sessions
			SET status = ?
			WHERE id = ?
		`
		args = []any{string(status), id}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("session not found: %s", id), nil)
	}

	return nil
}

// UpdateTokenUsage atomically updates the token usage statistics for a session.
func (r *SessionRepository) UpdateTokenUsage(ctx context.Context, id string, usage *session.TokenUsage) error {
	if usage == nil {
		return domainErrors.NewError(domainErrors.CodeValidation, "token usage is required", nil)
	}

	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf("failed to marshal token usage: %w", err)
	}

	query := `
		UPDATE sessions
		SET token_usage = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, string(usageJSON), id)
	if err != nil {
		return fmt.Errorf("failed to update token usage: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("session not found: %s", id), nil)
	}

	return nil
}

// Delete removes a session from storage.
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("session not found: %s", id), nil)
	}

	return nil
}

// querySessions executes a query and returns multiple sessions.
func (r *SessionRepository) querySessions(ctx context.Context, query string, args ...any) ([]*session.Session, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*session.Session
	for rows.Next() {
		sess, err := r.scanSessionRows(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// scanSessionRow scans a single row into a session.
func (r *SessionRepository) scanSessionRow(row *sql.Row) (*session.Session, error) {
	var (
		id, workspaceID                           string
		backend, model                            sql.NullString
		status                                    string
		startedAt                                 string
		endedAt                                   sql.NullString
		machineID, tmuxSession                    sql.NullString
		pid                                       sql.NullInt64
		metadataJSON, tokenUsageJSON, contextJSON sql.NullString
	)

	err := row.Scan(
		&id, &workspaceID, &backend, &model, &status,
		&startedAt, &endedAt, &machineID, &pid, &tmuxSession,
		&metadataJSON, &tokenUsageJSON, &contextJSON,
	)
	if err != nil {
		return nil, err
	}

	return buildSession(id, workspaceID, backend, model, status, startedAt, endedAt, machineID, pid, tmuxSession, metadataJSON, tokenUsageJSON, contextJSON)
}

// scanSessionRows scans rows into a session.
func (r *SessionRepository) scanSessionRows(rows *sql.Rows) (*session.Session, error) {
	var (
		id, workspaceID                           string
		backend, model                            sql.NullString
		status                                    string
		startedAt                                 string
		endedAt                                   sql.NullString
		machineID, tmuxSession                    sql.NullString
		pid                                       sql.NullInt64
		metadataJSON, tokenUsageJSON, contextJSON sql.NullString
	)

	err := rows.Scan(
		&id, &workspaceID, &backend, &model, &status,
		&startedAt, &endedAt, &machineID, &pid, &tmuxSession,
		&metadataJSON, &tokenUsageJSON, &contextJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	return buildSession(id, workspaceID, backend, model, status, startedAt, endedAt, machineID, pid, tmuxSession, metadataJSON, tokenUsageJSON, contextJSON)
}

// buildSession constructs a Session domain entity from database fields.
func buildSession(
	id, workspaceID string,
	backend, model sql.NullString,
	status, startedAt string,
	endedAt sql.NullString,
	machineID sql.NullString,
	pid sql.NullInt64,
	tmuxSession sql.NullString,
	metadataJSON, tokenUsageJSON, contextJSON sql.NullString,
) (*session.Session, error) {
	sess := &session.Session{
		ID:          id,
		WorkspaceID: workspaceID,
		Status:      session.Status(status),
	}

	if backend.Valid {
		sess.Backend = backend.String
	}
	if model.Valid {
		sess.Model = model.String
	}
	if machineID.Valid {
		sess.MachineID = machineID.String
	}
	if pid.Valid {
		sess.ProcessID = int(pid.Int64)
	}
	if tmuxSession.Valid {
		sess.TmuxSession = tmuxSession.String
	}

	// Parse started_at
	parsedStartedAt, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse started_at: %w", err)
	}
	sess.StartedAt = parsedStartedAt

	// Parse ended_at if present
	if endedAt.Valid {
		parsedEndedAt, err := time.Parse(time.RFC3339, endedAt.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ended_at: %w", err)
		}
		sess.EndedAt = &parsedEndedAt
	}

	// Parse metadata
	if metadataJSON.Valid && metadataJSON.String != "" && metadataJSON.String != "null" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &sess.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Parse token usage
	if tokenUsageJSON.Valid && tokenUsageJSON.String != "" && tokenUsageJSON.String != "null" {
		var usage session.TokenUsage
		if err := json.Unmarshal([]byte(tokenUsageJSON.String), &usage); err != nil {
			return nil, fmt.Errorf("failed to unmarshal token usage: %w", err)
		}
		sess.TokenUsage = &usage
	}

	// Parse context
	if contextJSON.Valid && contextJSON.String != "" && contextJSON.String != "null" {
		var ctx session.Context
		if err := json.Unmarshal([]byte(contextJSON.String), &ctx); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
		sess.Context = &ctx
	}

	return sess, nil
}

// nullableInt returns a sql.NullInt64 for the given int.
func nullableInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}
