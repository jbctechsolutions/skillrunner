package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeInfo represents information about a Git worktree
type WorktreeInfo struct {
	Path       string // Absolute path to the worktree
	Branch     string // Branch name
	IsMain     bool   // Whether this is the main worktree
	IsDetached bool   // Whether HEAD is detached
}

// Detector detects and manages Git worktree information
type Detector struct {
	repoPath string // Path to the Git repository
}

// NewDetector creates a new worktree detector for the given repository path
func NewDetector(repoPath string) *Detector {
	return &Detector{
		repoPath: repoPath,
	}
}

// IsWorktree checks if the given path is a Git worktree
func (d *Detector) IsWorktree(path string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if .git file exists (worktrees have .git as a file pointing to the main repo)
	gitPath := filepath.Join(absPath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false, nil // Not a worktree if .git doesn't exist
	}

	// If .git is a file (not a directory), it's likely a worktree
	if !info.IsDir() {
		return true, nil
	}

	// Check if it's a worktree by looking for gitdir file
	gitdirPath := filepath.Join(gitPath, "gitdir")
	if _, err := os.Stat(gitdirPath); err == nil {
		return true, nil
	}

	return false, nil
}

// DetectWorktree detects worktree information for the given path
func (d *Detector) DetectWorktree(path string) (*WorktreeInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	isWorktree, err := d.IsWorktree(absPath)
	if err != nil {
		return nil, err
	}

	if !isWorktree {
		return nil, fmt.Errorf("path is not a Git worktree: %s", absPath)
	}

	// Get branch name
	branch, isDetached, err := d.getBranch(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	// Check if this is the main worktree
	isMain, err := d.isMainWorktree(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if main worktree: %w", err)
	}

	return &WorktreeInfo{
		Path:       absPath,
		Branch:     branch,
		IsMain:     isMain,
		IsDetached: isDetached,
	}, nil
}

// ListWorktrees lists all worktrees for the repository
func (d *Detector) ListWorktrees() ([]*WorktreeInfo, error) {
	// Get the main repository path
	mainRepoPath, err := d.getMainRepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get main repo path: %w", err)
	}

	// Run git worktree list
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = mainRepoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return d.parseWorktreeList(string(output))
}

// getBranch gets the current branch name for a worktree
func (d *Detector) getBranch(path string) (string, bool, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", false, err
	}

	branch := strings.TrimSpace(string(output))
	isDetached := branch == "HEAD"

	return branch, isDetached, nil
}

// isMainWorktree checks if the given path is the main worktree
func (d *Detector) isMainWorktree(path string) (bool, error) {
	mainRepoPath, err := d.getMainRepoPath()
	if err != nil {
		return false, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	absMainPath, err := filepath.Abs(mainRepoPath)
	if err != nil {
		return false, err
	}

	return absPath == absMainPath, nil
}

// getMainRepoPath gets the main repository path
func (d *Detector) getMainRepoPath() (string, error) {
	path := d.repoPath
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Check if this is a worktree
	isWorktree, err := d.IsWorktree(absPath)
	if err != nil {
		return "", err
	}

	if !isWorktree {
		// Already the main repo
		return absPath, nil
	}

	// For worktrees, find the main repo
	gitPath := filepath.Join(absPath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat .git: %w", err)
	}

	if info.IsDir() {
		// Check for gitdir file
		gitdirPath := filepath.Join(gitPath, "gitdir")
		gitdirContent, err := os.ReadFile(gitdirPath)
		if err != nil {
			return absPath, nil // Assume main if can't read
		}

		gitdir := strings.TrimSpace(string(gitdirContent))
		// gitdir points to .git/worktrees/<name>
		// Main repo is the parent of .git
		worktreeGitDir := filepath.Dir(gitdir)
		mainGitDir := filepath.Dir(worktreeGitDir)
		mainRepo := filepath.Dir(mainGitDir)
		return mainRepo, nil
	}

	// .git is a file, read it to find the main repo
	gitContent, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("failed to read .git file: %w", err)
	}

	gitdir := strings.TrimSpace(string(gitContent))
	if !strings.HasPrefix(gitdir, "gitdir: ") {
		return absPath, nil
	}

	gitdir = strings.TrimPrefix(gitdir, "gitdir: ")
	gitdir = strings.TrimSpace(gitdir)

	// Resolve relative paths
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(absPath, gitdir)
	}

	// gitdir points to .git/worktrees/<name>
	// Main repo is the parent of .git
	worktreeGitDir := filepath.Dir(gitdir)
	mainGitDir := filepath.Dir(worktreeGitDir)
	mainRepo := filepath.Dir(mainGitDir)

	return mainRepo, nil
}

// parseWorktreeList parses the output of `git worktree list --porcelain`
func (d *Detector) parseWorktreeList(output string) ([]*WorktreeInfo, error) {
	lines := strings.Split(output, "\n")
	var worktrees []*WorktreeInfo
	var current *WorktreeInfo

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path at line %d: %w", i+1, err)
			}

			branch, isDetached, err := d.getBranch(absPath)
			if err != nil {
				// Continue even if branch detection fails
				branch = "unknown"
			}

			isMain, err := d.isMainWorktree(absPath)
			if err != nil {
				isMain = false
			}

			current = &WorktreeInfo{
				Path:       absPath,
				Branch:     branch,
				IsMain:     isMain,
				IsDetached: isDetached,
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}
