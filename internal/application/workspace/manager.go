// Package workspace provides workspace management functionality.
package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/uuid"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workspace"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/git"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/security"
)

// Manager manages development workspaces.
type Manager struct {
	storage     ports.WorkspaceStateStoragePort
	gitWorktree *git.WorktreeManager
	machineID   string
	baseDir     string // Base directory for workspaces
}

// NewManager creates a new workspace manager.
func NewManager(storage ports.WorkspaceStateStoragePort, machineID, baseDir string) (*Manager, error) {
	// Initialize git worktree manager
	gitWorktree, err := git.NewWorktreeManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize git worktree manager: %w", err)
	}

	// Ensure base directory exists
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".skillrunner", "workspaces")
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Manager{
		storage:     storage,
		gitWorktree: gitWorktree,
		machineID:   machineID,
		baseDir:     baseDir,
	}, nil
}

// Create creates a new workspace.
func (m *Manager) Create(ctx context.Context, opts workspace.CreateOptions) (*workspace.Workspace, error) {
	// Validate options
	if opts.Name == "" {
		return nil, fmt.Errorf("workspace name is required")
	}

	// Generate workspace ID
	wsID := uuid.New().String()[:8]

	// Determine workspace path
	wsPath := opts.Path
	if wsPath == "" {
		wsPath = filepath.Join(m.baseDir, opts.Name)
	}

	// Create workspace based on type
	var ws *workspace.Workspace
	var err error

	if opts.GitWorktree {
		ws, err = m.createGitWorktree(ctx, wsID, opts.Name, wsPath, opts.GitBranch, opts.Description)
	} else {
		ws, err = m.createDirectory(ctx, wsID, opts.Name, wsPath, opts.Description)
	}

	if err != nil {
		return nil, err
	}

	// Convert to context workspace and save
	ctxWs, err := m.convertToContextWorkspace(ws)
	if err != nil {
		return nil, fmt.Errorf("failed to convert workspace: %w", err)
	}

	if err := m.storage.Create(ctx, ctxWs); err != nil {
		// Try to clean up (best-effort, ignore errors)
		if opts.GitWorktree {
			_ = m.gitWorktree.Remove(ctx, wsPath, wsPath, true)
		} else {
			_ = os.RemoveAll(wsPath)
		}
		return nil, fmt.Errorf("failed to save workspace: %w", err)
	}

	return ws, nil
}

// createDirectory creates a simple directory workspace.
func (m *Manager) createDirectory(ctx context.Context, id, name, path, description string) (*workspace.Workspace, error) {
	// Create directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	return &workspace.Workspace{
		ID:          id,
		Name:        name,
		Type:        workspace.TypeDirectory,
		Path:        path,
		Status:      workspace.StatusInactive,
		MachineID:   m.machineID,
		Description: description,
	}, nil
}

// createGitWorktree creates a Git worktree workspace.
func (m *Manager) createGitWorktree(ctx context.Context, id, name, path, branch, description string) (*workspace.Workspace, error) {
	// Get current directory and check if it's a git repo
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	isRepo, err := m.gitWorktree.IsGitRepository(ctx, cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to check git repository: %w", err)
	}

	if !isRepo {
		return nil, fmt.Errorf("current directory is not a git repository")
	}

	// Get repository root
	repoRoot, err := m.gitWorktree.GetRepositoryRoot(ctx, cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Determine branch
	if branch == "" {
		// Use current branch
		branch, err = m.gitWorktree.GetCurrentBranch(ctx, cwd)
		if err != nil {
			return nil, fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	// Check if branch already exists
	branchExists, err := m.gitWorktree.BranchExists(ctx, repoRoot, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to check if branch exists: %w", err)
	}

	// Create worktree - only create new branch if it doesn't exist
	newBranch := !branchExists
	if err := m.gitWorktree.Create(ctx, repoRoot, path, branch, newBranch); err != nil {
		return nil, fmt.Errorf("failed to create git worktree: %w", err)
	}

	return &workspace.Workspace{
		ID:          id,
		Name:        name,
		Type:        workspace.TypeWorktree,
		Path:        path,
		GitBranch:   branch,
		Status:      workspace.StatusClean,
		MachineID:   m.machineID,
		ParentRepo:  repoRoot,
		Description: description,
	}, nil
}

// List returns workspaces matching the filter.
func (m *Manager) List(ctx context.Context, filter workspace.Filter) ([]*workspace.Workspace, error) {
	// Get all workspaces from storage
	ctxWorkspaces, err := m.storage.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Convert and filter
	var workspaces []*workspace.Workspace
	for _, ctxWs := range ctxWorkspaces {
		ws := m.convertFromContextWorkspace(ctxWs)

		// Apply filters
		if filter.Type != "" && ws.Type != filter.Type {
			continue
		}
		if filter.MachineID != "" && ws.MachineID != filter.MachineID {
			continue
		}
		if len(filter.Status) > 0 && !slices.Contains(filter.Status, ws.Status) {
			continue
		}

		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// Get retrieves a workspace by ID.
func (m *Manager) Get(ctx context.Context, workspaceID string) (*workspace.Workspace, error) {
	ctxWs, err := m.storage.Get(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return m.convertFromContextWorkspace(ctxWs), nil
}

// GetByName retrieves a workspace by name.
func (m *Manager) GetByName(ctx context.Context, name string) (*workspace.Workspace, error) {
	// List all and find by name
	ctxWorkspaces, err := m.storage.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, ctxWs := range ctxWorkspaces {
		if ctxWs.Name() == name {
			return m.convertFromContextWorkspace(ctxWs), nil
		}
	}

	return nil, fmt.Errorf("workspace not found: %s", name)
}

// Switch switches to a different workspace.
func (m *Manager) Switch(ctx context.Context, workspaceID string) error {
	// Get workspace
	ws, err := m.Get(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("workspace not found: %w", err)
	}

	// Change to workspace directory
	if err := os.Chdir(ws.Path); err != nil {
		return fmt.Errorf("failed to switch to workspace: %w", err)
	}

	// Update workspace status
	ctxWs, err := m.storage.Get(ctx, workspaceID)
	if err == nil {
		ctxWs.Activate()
		if err := m.storage.Update(ctx, ctxWs); err != nil {
			// Not critical, log and continue
		}
	}

	return nil
}

// Status returns the current workspace status.
func (m *Manager) Status(ctx context.Context) (*workspace.Workspace, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find workspace by path
	workspaces, err := m.List(ctx, workspace.Filter{})
	if err != nil {
		return nil, err
	}

	for _, ws := range workspaces {
		if ws.Path == cwd {
			return ws, nil
		}
	}

	return nil, fmt.Errorf("current directory is not a workspace")
}

// Delete removes a workspace.
func (m *Manager) Delete(ctx context.Context, workspaceID string, removeFiles bool) error {
	// Get workspace
	ws, err := m.Get(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("workspace not found: %w", err)
	}

	// Remove files if requested
	if removeFiles {
		// Sanitize path before deletion to prevent dangerous operations
		if err := security.SanitizePathForDeletion(ws.Path); err != nil {
			return fmt.Errorf("cannot delete workspace files: %w", err)
		}

		if ws.Type == workspace.TypeWorktree {
			// Remove git worktree
			if err := m.gitWorktree.Remove(ctx, ws.ParentRepo, ws.Path, true); err != nil {
				return fmt.Errorf("failed to remove worktree: %w", err)
			}
		} else {
			// Remove directory
			if err := os.RemoveAll(ws.Path); err != nil {
				return fmt.Errorf("failed to remove workspace directory: %w", err)
			}
		}
	}

	// Remove from storage
	if err := m.storage.Delete(ctx, workspaceID); err != nil {
		return fmt.Errorf("failed to delete workspace from storage: %w", err)
	}

	return nil
}

// convertToContextWorkspace converts domain workspace to context workspace.
func (m *Manager) convertToContextWorkspace(ws *workspace.Workspace) (*domainContext.Workspace, error) {
	ctxWs, err := domainContext.NewWorkspace(ws.ID, ws.Name, ws.Path)
	if err != nil {
		return nil, err
	}

	if ws.GitBranch != "" {
		ctxWs.SetBranch(ws.GitBranch)
	}
	if ws.ParentRepo != "" {
		ctxWs.SetWorktreePath(ws.Path)
	}

	return ctxWs, nil
}

// convertFromContextWorkspace converts context workspace to domain workspace.
func (m *Manager) convertFromContextWorkspace(ctxWs *domainContext.Workspace) *workspace.Workspace {
	ws := &workspace.Workspace{
		ID:        ctxWs.ID(),
		Name:      ctxWs.Name(),
		Path:      ctxWs.RepoPath(),
		GitBranch: ctxWs.Branch(),
		MachineID: m.machineID,
	}

	// Determine type from worktree path
	if ctxWs.WorktreePath() != "" {
		ws.Type = workspace.TypeWorktree
		ws.Path = ctxWs.WorktreePath()
		ws.ParentRepo = ctxWs.RepoPath()
	} else {
		ws.Type = workspace.TypeDirectory
	}

	// Map status
	switch ctxWs.Status() {
	case domainContext.WorkspaceStatusActive:
		ws.Status = workspace.StatusActive
	case domainContext.WorkspaceStatusIdle:
		ws.Status = workspace.StatusInactive
	case domainContext.WorkspaceStatusArchived:
		ws.Status = workspace.StatusInactive
	default:
		ws.Status = workspace.StatusInactive
	}

	ws.CreatedAt = ctxWs.CreatedAt()
	ws.LastUsedAt = ctxWs.LastActiveAt()

	return ws
}
