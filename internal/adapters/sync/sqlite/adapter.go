package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// Adapter implements the SyncBackendPort interface for SQLite.
type Adapter struct {
	conn *Connection
}

// NewAdapter creates a new SQLite sync adapter.
func NewAdapter(dbPath string) (*Adapter, error) {
	conn, err := NewConnection(dbPath)
	if err != nil {
		return nil, err
	}

	if err := conn.Open(); err != nil {
		return nil, err
	}

	return &Adapter{
		conn: conn,
	}, nil
}

// Close closes the adapter and underlying database connection.
func (a *Adapter) Close() error {
	return a.conn.Close()
}

// Push implements SyncBackendPort.Push
// For SQLite (local storage), Push means writing to the database
func (a *Adapter) Push(ctx context.Context, state *ports.SyncState) error {
	db, err := a.conn.DB()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Push workspaces
	for _, ws := range state.Workspaces {
		if err := a.upsertWorkspace(ctx, tx, &ws); err != nil {
			return fmt.Errorf("could not upsert workspace %s: %w", ws.ID, err)
		}
	}

	// Push sessions
	for _, sess := range state.Sessions {
		if err := a.upsertSession(ctx, tx, &sess); err != nil {
			return fmt.Errorf("could not upsert session %s: %w", sess.ID, err)
		}
	}

	// Push checkpoints
	for _, cp := range state.Checkpoints {
		if err := a.upsertCheckpoint(ctx, tx, &cp); err != nil {
			return fmt.Errorf("could not upsert checkpoint %s: %w", cp.ID, err)
		}
	}

	// Push context items
	for _, item := range state.Items {
		if err := a.upsertContextItem(ctx, tx, &item); err != nil {
			return fmt.Errorf("could not upsert context item %s: %w", item.ID, err)
		}
	}

	// Push rules
	for _, rule := range state.Rules {
		if err := a.upsertRule(ctx, tx, &rule); err != nil {
			return fmt.Errorf("could not upsert rule %s: %w", rule.ID, err)
		}
	}

	return tx.Commit()
}

// Pull implements SyncBackendPort.Pull
// For SQLite (local storage), Pull means reading from the database
func (a *Adapter) Pull(ctx context.Context) (*ports.SyncState, error) {
	db, err := a.conn.DB()
	if err != nil {
		return nil, err
	}

	state := &ports.SyncState{
		Version:   "1.0.0",
		MachineID: a.getMachineID(),
		SyncedAt:  time.Now(),
	}

	// Pull workspaces
	state.Workspaces, err = a.fetchWorkspaces(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("could not fetch workspaces: %w", err)
	}

	// Pull sessions
	state.Sessions, err = a.fetchSessions(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("could not fetch sessions: %w", err)
	}

	// Pull checkpoints
	state.Checkpoints, err = a.fetchCheckpoints(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("could not fetch checkpoints: %w", err)
	}

	// Pull context items
	state.Items, err = a.fetchContextItems(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("could not fetch context items: %w", err)
	}

	// Pull rules
	state.Rules, err = a.fetchRules(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("could not fetch rules: %w", err)
	}

	return state, nil
}

// HasUpdates implements SyncBackendPort.HasUpdates
func (a *Adapter) HasUpdates(ctx context.Context, since time.Time) (bool, error) {
	db, err := a.conn.DB()
	if err != nil {
		return false, err
	}

	// Check if any records have been updated since the given time
	queries := []string{
		"SELECT COUNT(*) FROM workspaces WHERE updated_at > ?",
		"SELECT COUNT(*) FROM sessions WHERE started_at > ?",
		"SELECT COUNT(*) FROM checkpoints WHERE created_at > ?",
		"SELECT COUNT(*) FROM context_items WHERE created_at > ?",
		"SELECT COUNT(*) FROM rules WHERE updated_at > ?",
	}

	sinceStr := since.Format(time.RFC3339)
	for _, query := range queries {
		var count int
		if err := db.QueryRowContext(ctx, query, sinceStr).Scan(&count); err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}

	return false, nil
}

// IsAvailable implements SyncBackendPort.IsAvailable
func (a *Adapter) IsAvailable(ctx context.Context) (bool, error) {
	db, err := a.conn.DB()
	if err != nil {
		return false, nil // Return false, not error, as this is an availability check
	}

	if err := db.PingContext(ctx); err != nil {
		return false, nil
	}

	return true, nil
}

// Helper methods for CRUD operations

func (a *Adapter) upsertWorkspace(ctx context.Context, tx *sql.Tx, ws *ports.WorkspaceSnapshot) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			updated_at = excluded.updated_at
	`, ws.ID, ws.Name, ws.Description, ws.CreatedAt, ws.UpdatedAt)
	return err
}

func (a *Adapter) upsertSession(ctx context.Context, tx *sql.Tx, sess *ports.SessionSnapshot) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (id, workspace_id, name, started_at, ended_at, status)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			ended_at = excluded.ended_at,
			status = excluded.status
	`, sess.ID, sess.WorkspaceID, sess.Name, sess.StartedAt, sess.EndedAt, sess.Status)
	return err
}

func (a *Adapter) upsertCheckpoint(ctx context.Context, tx *sql.Tx, cp *ports.CheckpointSnapshot) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO checkpoints (id, session_id, name, description, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description
	`, cp.ID, cp.SessionID, cp.Name, cp.Description, cp.CreatedAt)
	return err
}

func (a *Adapter) upsertContextItem(ctx context.Context, tx *sql.Tx, item *ports.ContextItemSnapshot) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO context_items (id, checkpoint_id, type, key, value, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			key = excluded.key,
			value = excluded.value
	`, item.ID, item.CheckpointID, item.Type, item.Key, item.Value, item.CreatedAt)
	return err
}

func (a *Adapter) upsertRule(ctx context.Context, tx *sql.Tx, rule *ports.RuleSnapshot) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO rules (id, workspace_id, name, pattern, action, priority, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			pattern = excluded.pattern,
			action = excluded.action,
			priority = excluded.priority,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at
	`, rule.ID, rule.WorkspaceID, rule.Name, rule.Pattern, rule.Action, rule.Priority, rule.Enabled, rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (a *Adapter) fetchWorkspaces(ctx context.Context, db *sql.DB) ([]ports.WorkspaceSnapshot, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, description, created_at, updated_at FROM workspaces")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []ports.WorkspaceSnapshot
	for rows.Next() {
		var ws ports.WorkspaceSnapshot
		var description sql.NullString
		if err := rows.Scan(&ws.ID, &ws.Name, &description, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
			return nil, err
		}
		ws.Description = description.String
		workspaces = append(workspaces, ws)
	}

	return workspaces, rows.Err()
}

func (a *Adapter) fetchSessions(ctx context.Context, db *sql.DB) ([]ports.SessionSnapshot, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, workspace_id, name, started_at, ended_at, status FROM sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ports.SessionSnapshot
	for rows.Next() {
		var sess ports.SessionSnapshot
		var endedAt sql.NullTime
		if err := rows.Scan(&sess.ID, &sess.WorkspaceID, &sess.Name, &sess.StartedAt, &endedAt, &sess.Status); err != nil {
			return nil, err
		}
		if endedAt.Valid {
			sess.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, sess)
	}

	return sessions, rows.Err()
}

func (a *Adapter) fetchCheckpoints(ctx context.Context, db *sql.DB) ([]ports.CheckpointSnapshot, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, session_id, name, description, created_at FROM checkpoints")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkpoints []ports.CheckpointSnapshot
	for rows.Next() {
		var cp ports.CheckpointSnapshot
		var description sql.NullString
		if err := rows.Scan(&cp.ID, &cp.SessionID, &cp.Name, &description, &cp.CreatedAt); err != nil {
			return nil, err
		}
		cp.Description = description.String
		checkpoints = append(checkpoints, cp)
	}

	return checkpoints, rows.Err()
}

func (a *Adapter) fetchContextItems(ctx context.Context, db *sql.DB) ([]ports.ContextItemSnapshot, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, checkpoint_id, type, key, value, created_at FROM context_items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ports.ContextItemSnapshot
	for rows.Next() {
		var item ports.ContextItemSnapshot
		var value sql.NullString
		if err := rows.Scan(&item.ID, &item.CheckpointID, &item.Type, &item.Key, &value, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Value = value.String
		items = append(items, item)
	}

	return items, rows.Err()
}

func (a *Adapter) fetchRules(ctx context.Context, db *sql.DB) ([]ports.RuleSnapshot, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, workspace_id, name, pattern, action, priority, enabled, created_at, updated_at FROM rules")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []ports.RuleSnapshot
	for rows.Next() {
		var rule ports.RuleSnapshot
		if err := rows.Scan(&rule.ID, &rule.WorkspaceID, &rule.Name, &rule.Pattern, &rule.Action, &rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// getMachineID returns a unique machine identifier
func (a *Adapter) getMachineID() string {
	hostname, err := os.Hostname()
	if err != nil {
		return uuid.New().String()
	}
	return hostname
}
