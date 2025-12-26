// Package context provides domain entities for workspace and context management.
package context

import (
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// WorkspaceStatus represents the current state of a workspace.
type WorkspaceStatus string

const (
	// WorkspaceStatusActive indicates the workspace is actively being used.
	WorkspaceStatusActive WorkspaceStatus = "active"

	// WorkspaceStatusIdle indicates the workspace exists but is not currently active.
	WorkspaceStatusIdle WorkspaceStatus = "idle"

	// WorkspaceStatusArchived indicates the workspace has been archived.
	WorkspaceStatusArchived WorkspaceStatus = "archived"
)

// Workspace is the aggregate root representing a development workspace.
// It manages the context for skill execution including the repository path,
// worktree location, current focus, and default backend configuration.
type Workspace struct {
	id             string
	name           string
	repoPath       string
	worktreePath   string
	branch         string
	focus          string
	status         WorkspaceStatus
	defaultBackend string
	lastActiveAt   time.Time
	createdAt      time.Time
}

// NewWorkspace creates a new Workspace with the required fields.
// Returns an error if validation fails:
//   - id is required
//   - name is required
//   - repoPath is required
func NewWorkspace(id, name, repoPath string) (*Workspace, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	repoPath = strings.TrimSpace(repoPath)

	if id == "" {
		return nil, errors.New("workspace", "workspace ID is required")
	}
	if name == "" {
		return nil, errors.New("workspace", "workspace name is required")
	}
	if repoPath == "" {
		return nil, errors.New("workspace", "repository path is required")
	}

	now := time.Now()
	return &Workspace{
		id:           id,
		name:         name,
		repoPath:     repoPath,
		status:       WorkspaceStatusActive,
		lastActiveAt: now,
		createdAt:    now,
	}, nil
}

// ID returns the workspace's unique identifier.
func (w *Workspace) ID() string {
	return w.id
}

// Name returns the workspace's human-readable name.
func (w *Workspace) Name() string {
	return w.name
}

// RepoPath returns the path to the repository.
func (w *Workspace) RepoPath() string {
	return w.repoPath
}

// WorktreePath returns the path to the worktree, if set.
func (w *Workspace) WorktreePath() string {
	return w.worktreePath
}

// Branch returns the current branch.
func (w *Workspace) Branch() string {
	return w.branch
}

// Focus returns the current focus (e.g., issue ID or task).
func (w *Workspace) Focus() string {
	return w.focus
}

// Status returns the workspace's current status.
func (w *Workspace) Status() WorkspaceStatus {
	return w.status
}

// DefaultBackend returns the default backend provider.
func (w *Workspace) DefaultBackend() string {
	return w.defaultBackend
}

// LastActiveAt returns when the workspace was last active.
func (w *Workspace) LastActiveAt() time.Time {
	return w.lastActiveAt
}

// CreatedAt returns when the workspace was created.
func (w *Workspace) CreatedAt() time.Time {
	return w.createdAt
}

// SetWorktreePath sets the worktree path for the workspace.
func (w *Workspace) SetWorktreePath(path string) {
	w.worktreePath = strings.TrimSpace(path)
}

// SetBranch sets the current branch.
func (w *Workspace) SetBranch(branch string) {
	w.branch = strings.TrimSpace(branch)
}

// SetFocus sets the current focus for the workspace.
// This is typically an issue ID or task identifier.
func (w *Workspace) SetFocus(focus string) {
	w.focus = strings.TrimSpace(focus)
	w.lastActiveAt = time.Now()
}

// SetDefaultBackend sets the default backend provider.
func (w *Workspace) SetDefaultBackend(backend string) {
	w.defaultBackend = strings.TrimSpace(backend)
}

// Activate marks the workspace as active and updates the last active time.
func (w *Workspace) Activate() {
	w.status = WorkspaceStatusActive
	w.lastActiveAt = time.Now()
}

// SetIdle marks the workspace as idle.
func (w *Workspace) SetIdle() {
	w.status = WorkspaceStatusIdle
}

// Archive marks the workspace as archived.
func (w *Workspace) Archive() {
	w.status = WorkspaceStatusArchived
}

// Touch updates the last active timestamp.
func (w *Workspace) Touch() {
	w.lastActiveAt = time.Now()
}

// Validate checks if the Workspace is in a valid state.
func (w *Workspace) Validate() error {
	if strings.TrimSpace(w.id) == "" {
		return errors.New("workspace", "workspace ID is required")
	}
	if strings.TrimSpace(w.name) == "" {
		return errors.New("workspace", "workspace name is required")
	}
	if strings.TrimSpace(w.repoPath) == "" {
		return errors.New("workspace", "repository path is required")
	}

	// Validate status
	switch w.status {
	case WorkspaceStatusActive, WorkspaceStatusIdle, WorkspaceStatusArchived:
		// Valid status
	default:
		return errors.New("workspace", "invalid workspace status")
	}

	return nil
}
