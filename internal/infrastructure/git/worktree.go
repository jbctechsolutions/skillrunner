// Package git provides Git integration functionality.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeManager manages Git worktrees.
type WorktreeManager struct {
	gitPath string
}

// NewWorktreeManager creates a new worktree manager.
func NewWorktreeManager() (*WorktreeManager, error) {
	// Check if git is available
	path, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git not found in PATH: %w", err)
	}

	return &WorktreeManager{
		gitPath: path,
	}, nil
}

// WorktreeInfo contains information about a Git worktree.
type WorktreeInfo struct {
	Path   string // Absolute path to worktree
	Branch string // Branch name
	Commit string // Current commit hash
	IsMain bool   // Whether this is the main worktree
}

// Create creates a new Git worktree.
func (wm *WorktreeManager) Create(ctx context.Context, repoPath, worktreePath, branch string, newBranch bool) error {
	if repoPath == "" {
		return fmt.Errorf("repository path is required")
	}
	if worktreePath == "" {
		return fmt.Errorf("worktree path is required")
	}
	if branch == "" {
		return fmt.Errorf("branch name is required")
	}

	args := []string{"worktree", "add"}

	// Add new branch flag if needed
	if newBranch {
		args = append(args, "-b", branch)
	}

	args = append(args, worktreePath)

	// Add branch reference if not creating new
	if !newBranch {
		args = append(args, branch)
	}

	cmd := exec.CommandContext(ctx, wm.gitPath, args...)
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree: %s: %w", stderr.String(), err)
	}

	return nil
}

// Remove removes a Git worktree.
func (wm *WorktreeManager) Remove(ctx context.Context, repoPath, worktreePath string, force bool) error {
	if repoPath == "" {
		return fmt.Errorf("repository path is required")
	}
	if worktreePath == "" {
		return fmt.Errorf("worktree path is required")
	}

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	cmd := exec.CommandContext(ctx, wm.gitPath, args...)
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %s: %w", stderr.String(), err)
	}

	return nil
}

// List returns all worktrees for a repository.
func (wm *WorktreeManager) List(ctx context.Context, repoPath string) ([]WorktreeInfo, error) {
	if repoPath == "" {
		return nil, fmt.Errorf("repository path is required")
	}

	cmd := exec.CommandContext(ctx, wm.gitPath, "worktree", "list", "--porcelain")
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return wm.parseWorktreeList(stdout.String()), nil
}

// parseWorktreeList parses the output of 'git worktree list --porcelain'.
func (wm *WorktreeManager) parseWorktreeList(output string) []WorktreeInfo {
	var worktrees []WorktreeInfo
	var current *WorktreeInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current = &WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if current != nil {
			if strings.HasPrefix(line, "HEAD ") {
				current.Commit = strings.TrimPrefix(line, "HEAD ")
			} else if strings.HasPrefix(line, "branch ") {
				current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
			} else if line == "bare" {
				current.IsMain = true
			}
		}
	}

	// Add last worktree if exists
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees
}

// Exists checks if a worktree exists at the given path.
func (wm *WorktreeManager) Exists(ctx context.Context, repoPath, worktreePath string) (bool, error) {
	worktrees, err := wm.List(ctx, repoPath)
	if err != nil {
		return false, err
	}

	// Normalize paths for comparison
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	for _, wt := range worktrees {
		absWtPath, err := filepath.Abs(wt.Path)
		if err != nil {
			continue
		}
		if absWtPath == absWorktreePath {
			return true, nil
		}
	}

	return false, nil
}

// Prune removes worktree administrative data for removed worktrees.
func (wm *WorktreeManager) Prune(ctx context.Context, repoPath string) error {
	if repoPath == "" {
		return fmt.Errorf("repository path is required")
	}

	cmd := exec.CommandContext(ctx, wm.gitPath, "worktree", "prune")
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}

	return nil
}

// Lock locks a worktree to prevent it from being pruned.
func (wm *WorktreeManager) Lock(ctx context.Context, repoPath, worktreePath, reason string) error {
	if repoPath == "" {
		return fmt.Errorf("repository path is required")
	}
	if worktreePath == "" {
		return fmt.Errorf("worktree path is required")
	}

	args := []string{"worktree", "lock"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	args = append(args, worktreePath)

	cmd := exec.CommandContext(ctx, wm.gitPath, args...)
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to lock worktree: %w", err)
	}

	return nil
}

// Unlock unlocks a worktree.
func (wm *WorktreeManager) Unlock(ctx context.Context, repoPath, worktreePath string) error {
	if repoPath == "" {
		return fmt.Errorf("repository path is required")
	}
	if worktreePath == "" {
		return fmt.Errorf("worktree path is required")
	}

	cmd := exec.CommandContext(ctx, wm.gitPath, "worktree", "unlock", worktreePath)
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unlock worktree: %w", err)
	}

	return nil
}

// IsGitRepository checks if a path is a Git repository.
func (wm *WorktreeManager) IsGitRepository(ctx context.Context, path string) (bool, error) {
	cmd := exec.CommandContext(ctx, wm.gitPath, "rev-parse", "--git-dir")
	cmd.Dir = path

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 128 {
				return false, nil // Not a git repository
			}
		}
		return false, fmt.Errorf("failed to check git repository: %w", err)
	}

	return true, nil
}

// GetCurrentBranch returns the current branch name.
func (wm *WorktreeManager) GetCurrentBranch(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, wm.gitPath, "branch", "--show-current")
	cmd.Dir = path

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetRepositoryRoot returns the root directory of the Git repository.
func (wm *WorktreeManager) GetRepositoryRoot(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, wm.gitPath, "rev-parse", "--show-toplevel")
	cmd.Dir = path

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// BranchExists checks if a local branch exists in the repository.
func (wm *WorktreeManager) BranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
	if repoPath == "" {
		return false, fmt.Errorf("repository path is required")
	}
	if branch == "" {
		return false, fmt.Errorf("branch name is required")
	}

	// Check if local branch exists using git show-ref
	cmd := exec.CommandContext(ctx, wm.gitPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repoPath

	err := cmd.Run()
	if err == nil {
		return true, nil // Branch exists locally
	}

	// If the command failed, check if it's because the branch doesn't exist
	exitErr, ok := err.(*exec.ExitError)
	if ok && exitErr.ExitCode() == 1 {
		return false, nil // Branch does not exist
	}

	return false, fmt.Errorf("failed to check branch existence: %w", err)
}
