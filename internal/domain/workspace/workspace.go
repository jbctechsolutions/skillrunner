// Package workspace defines domain models for workspace management.
package workspace

import (
	"time"
)

// Type represents the type of workspace.
type Type string

const (
	TypeDirectory Type = "directory" // Regular directory workspace
	TypeWorktree  Type = "worktree"  // Git worktree workspace
	TypeContainer Type = "container" // Container-based workspace
)

// Status represents the current state of a workspace.
type Status string

const (
	StatusActive   Status = "active"   // Currently in use
	StatusInactive Status = "inactive" // Not in use
	StatusDirty    Status = "dirty"    // Has uncommitted changes
	StatusClean    Status = "clean"    // No uncommitted changes
)

// Workspace represents a development workspace.
type Workspace struct {
	ID          string            // Unique workspace identifier
	Name        string            // Human-readable name
	Type        Type              // Workspace type
	Path        string            // Absolute path to workspace directory
	GitBranch   string            // Associated git branch (if git workspace)
	Status      Status            // Current status
	CreatedAt   time.Time         // When workspace was created
	LastUsedAt  time.Time         // When workspace was last used
	MachineID   string            // Machine identifier for remote workspaces
	ParentRepo  string            // Parent repository path (for worktrees)
	Description string            // Optional description
	Metadata    map[string]string // Additional metadata
}

// IsGitWorkspace returns true if the workspace is backed by git.
func (w *Workspace) IsGitWorkspace() bool {
	return w.Type == TypeWorktree || (w.GitBranch != "" && w.ParentRepo != "")
}

// CreateOptions contains parameters for creating a new workspace.
type CreateOptions struct {
	Name        string // Workspace name (required)
	Path        string // Path for workspace (optional, auto-generated if empty)
	Type        Type   // Workspace type
	GitWorktree bool   // Create as git worktree
	GitBranch   string // Branch name for worktree
	Description string // Optional description
}

// Filter defines criteria for querying workspaces.
type Filter struct {
	Type      Type     // Filter by type
	Status    []Status // Filter by status
	MachineID string   // Filter by machine (empty for current machine)
}

// SpawnOptions contains parameters for spawning a terminal in a workspace.
type SpawnOptions struct {
	WorkspaceID string // Workspace to spawn terminal in
	Terminal    string // Terminal type (iterm2, terminal, tmux, auto)
	Command     string // Command to run in new terminal
	Background  bool   // Run in background
}
