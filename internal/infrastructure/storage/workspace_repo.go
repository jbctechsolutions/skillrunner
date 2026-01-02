// Package storage provides SQLite-based storage implementations for state management.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
	domainErrors "github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// Compile-time check that WorkspaceRepository implements WorkspaceStateStoragePort.
var _ ports.WorkspaceStateStoragePort = (*WorkspaceRepository)(nil)

// WorkspaceRepository implements WorkspaceStateStoragePort using SQLite.
type WorkspaceRepository struct {
	db *sql.DB
}

// NewWorkspaceRepository creates a new workspace repository.
func NewWorkspaceRepository(db *sql.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// Create persists a new workspace to storage.
func (r *WorkspaceRepository) Create(ctx context.Context, workspace *domainContext.Workspace) error {
	if err := workspace.Validate(); err != nil {
		return err
	}

	query := `
		INSERT INTO workspaces (id, name, repo_path, worktree_path, branch, focus, status, default_backend, last_active_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		workspace.ID(),
		workspace.Name(),
		workspace.RepoPath(),
		nullableString(workspace.WorktreePath()),
		nullableString(workspace.Branch()),
		nullableString(workspace.Focus()),
		string(workspace.Status()),
		nullableString(workspace.DefaultBackend()),
		workspace.LastActiveAt().Format(time.RFC3339),
		workspace.CreatedAt().Format(time.RFC3339),
	)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return domainErrors.NewError(domainErrors.CodeValidation, "workspace already exists", err)
		}
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	return nil
}

// Get retrieves a workspace by its unique identifier.
func (r *WorkspaceRepository) Get(ctx context.Context, id string) (*domainContext.Workspace, error) {
	query := `
		SELECT id, name, repo_path, worktree_path, branch, focus, status, default_backend, last_active_at, created_at
		FROM workspaces
		WHERE id = ?
	`

	workspace, err := r.scanWorkspaceRow(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workspace not found: %s", id), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	return workspace, nil
}

// GetByName retrieves a workspace by its human-readable name.
func (r *WorkspaceRepository) GetByName(ctx context.Context, name string) (*domainContext.Workspace, error) {
	query := `
		SELECT id, name, repo_path, worktree_path, branch, focus, status, default_backend, last_active_at, created_at
		FROM workspaces
		WHERE name = ?
	`

	workspace, err := r.scanWorkspaceRow(r.db.QueryRowContext(ctx, query, name))
	if err == sql.ErrNoRows {
		return nil, domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workspace not found: %s", name), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace by name: %w", err)
	}

	return workspace, nil
}

// GetByRepoPath retrieves a workspace associated with a repository path.
func (r *WorkspaceRepository) GetByRepoPath(ctx context.Context, repoPath string) (*domainContext.Workspace, error) {
	query := `
		SELECT id, name, repo_path, worktree_path, branch, focus, status, default_backend, last_active_at, created_at
		FROM workspaces
		WHERE repo_path = ?
	`

	workspace, err := r.scanWorkspaceRow(r.db.QueryRowContext(ctx, query, repoPath))
	if err == sql.ErrNoRows {
		return nil, domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workspace not found for path: %s", repoPath), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace by repo path: %w", err)
	}

	return workspace, nil
}

// GetActive retrieves all workspaces with active status.
func (r *WorkspaceRepository) GetActive(ctx context.Context) ([]*domainContext.Workspace, error) {
	query := `
		SELECT id, name, repo_path, worktree_path, branch, focus, status, default_backend, last_active_at, created_at
		FROM workspaces
		WHERE status = ?
		ORDER BY last_active_at DESC
	`

	return r.queryWorkspaces(ctx, query, string(domainContext.WorkspaceStatusActive))
}

// List returns all workspaces matching the optional filter criteria.
func (r *WorkspaceRepository) List(ctx context.Context, filter *ports.WorkspaceFilter) ([]*domainContext.Workspace, error) {
	query := `
		SELECT id, name, repo_path, worktree_path, branch, focus, status, default_backend, last_active_at, created_at
		FROM workspaces
		WHERE 1=1
	`
	args := []any{}

	if filter != nil {
		if filter.Status != "" {
			query += " AND status = ?"
			args = append(args, string(filter.Status))
		}
		if filter.Name != "" {
			query += " AND name LIKE ?"
			args = append(args, "%"+filter.Name+"%")
		}
		if filter.RepoPath != "" {
			query += " AND repo_path = ?"
			args = append(args, filter.RepoPath)
		}
	}

	query += " ORDER BY last_active_at DESC"

	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	return r.queryWorkspaces(ctx, query, args...)
}

// Update persists changes to an existing workspace.
func (r *WorkspaceRepository) Update(ctx context.Context, workspace *domainContext.Workspace) error {
	if err := workspace.Validate(); err != nil {
		return err
	}

	query := `
		UPDATE workspaces
		SET name = ?, repo_path = ?, worktree_path = ?, branch = ?, focus = ?, status = ?, default_backend = ?, last_active_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		workspace.Name(),
		workspace.RepoPath(),
		nullableString(workspace.WorktreePath()),
		nullableString(workspace.Branch()),
		nullableString(workspace.Focus()),
		string(workspace.Status()),
		nullableString(workspace.DefaultBackend()),
		workspace.LastActiveAt().Format(time.RFC3339),
		workspace.ID(),
	)

	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workspace not found: %s", workspace.ID()), nil)
	}

	return nil
}

// SetFocus updates the focus field of a workspace.
func (r *WorkspaceRepository) SetFocus(ctx context.Context, id string, focus string) error {
	query := `
		UPDATE workspaces
		SET focus = ?, last_active_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, nullableString(focus), time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to set workspace focus: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workspace not found: %s", id), nil)
	}

	return nil
}

// SetStatus updates the status of a workspace.
func (r *WorkspaceRepository) SetStatus(ctx context.Context, id string, status domainContext.WorkspaceStatus) error {
	query := `
		UPDATE workspaces
		SET status = ?, last_active_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, string(status), time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to set workspace status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workspace not found: %s", id), nil)
	}

	return nil
}

// Delete removes a workspace from storage.
func (r *WorkspaceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workspaces WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rows == 0 {
		return domainErrors.NewError(domainErrors.CodeNotFound, fmt.Sprintf("workspace not found: %s", id), nil)
	}

	return nil
}

// Exists checks whether a workspace with the given ID exists.
func (r *WorkspaceRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT COUNT(*) FROM workspaces WHERE id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check workspace existence: %w", err)
	}

	return count > 0, nil
}

// queryWorkspaces executes a query and returns multiple workspaces.
func (r *WorkspaceRepository) queryWorkspaces(ctx context.Context, query string, args ...any) ([]*domainContext.Workspace, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []*domainContext.Workspace
	for rows.Next() {
		workspace, err := r.scanWorkspaceRows(rows)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, workspace)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workspaces: %w", err)
	}

	return workspaces, nil
}

// scanWorkspaceRow scans a single row into a workspace.
func (r *WorkspaceRepository) scanWorkspaceRow(row *sql.Row) (*domainContext.Workspace, error) {
	var (
		id, name, repoPath          string
		worktreePath, branch, focus sql.NullString
		status                      string
		defaultBackend              sql.NullString
		lastActiveAt, createdAt     string
	)

	err := row.Scan(
		&id, &name, &repoPath, &worktreePath, &branch, &focus,
		&status, &defaultBackend, &lastActiveAt, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	return buildWorkspace(id, name, repoPath, worktreePath, branch, focus, status, defaultBackend, lastActiveAt, createdAt)
}

// scanWorkspaceRows scans rows into a workspace.
func (r *WorkspaceRepository) scanWorkspaceRows(rows *sql.Rows) (*domainContext.Workspace, error) {
	var (
		id, name, repoPath          string
		worktreePath, branch, focus sql.NullString
		status                      string
		defaultBackend              sql.NullString
		lastActiveAt, createdAt     string
	)

	err := rows.Scan(
		&id, &name, &repoPath, &worktreePath, &branch, &focus,
		&status, &defaultBackend, &lastActiveAt, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan workspace: %w", err)
	}

	return buildWorkspace(id, name, repoPath, worktreePath, branch, focus, status, defaultBackend, lastActiveAt, createdAt)
}

// buildWorkspace constructs a Workspace domain entity from database fields.
func buildWorkspace(
	id, name, repoPath string,
	worktreePath, branch, focus sql.NullString,
	status string,
	defaultBackend sql.NullString,
	lastActiveAt, createdAt string,
) (*domainContext.Workspace, error) {
	workspace, err := domainContext.NewWorkspace(id, name, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	if worktreePath.Valid {
		workspace.SetWorktreePath(worktreePath.String)
	}
	if branch.Valid {
		workspace.SetBranch(branch.String)
	}
	if focus.Valid {
		workspace.SetFocus(focus.String)
	}
	if defaultBackend.Valid {
		workspace.SetDefaultBackend(defaultBackend.String)
	}

	// Set status
	switch domainContext.WorkspaceStatus(status) {
	case domainContext.WorkspaceStatusActive:
		workspace.Activate()
	case domainContext.WorkspaceStatusIdle:
		workspace.SetIdle()
	case domainContext.WorkspaceStatusArchived:
		workspace.Archive()
	}

	return workspace, nil
}

// nullableString returns a sql.NullString for the given string.
func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
