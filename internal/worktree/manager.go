package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConfirmationFunc is a function type for user confirmation
type ConfirmationFunc func(message string) (bool, error)

// Manager manages Git worktrees with auto-pull and confirmation
type Manager struct {
	detector    *Detector
	confirmFunc ConfirmationFunc
	autoPull    bool
	repoPath    string
}

// NewManager creates a new worktree manager
func NewManager(repoPath string, autoPull bool, confirmFunc ConfirmationFunc) *Manager {
	if confirmFunc == nil {
		// Default confirmation: always return true (auto-confirm)
		confirmFunc = func(string) (bool, error) { return true, nil }
	}

	return &Manager{
		detector:    NewDetector(repoPath),
		confirmFunc: confirmFunc,
		autoPull:    autoPull,
		repoPath:    repoPath,
	}
}

// SyncWorktree syncs a worktree (pull changes) with confirmation
func (m *Manager) SyncWorktree(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify it's a worktree
	isWorktree, err := m.detector.IsWorktree(absPath)
	if err != nil {
		return fmt.Errorf("failed to check if worktree: %w", err)
	}

	if !isWorktree {
		return fmt.Errorf("path is not a Git worktree: %s", absPath)
	}

	// Get worktree info
	info, err := m.detector.DetectWorktree(absPath)
	if err != nil {
		return fmt.Errorf("failed to detect worktree: %w", err)
	}

	// Check if there are remote changes
	hasRemoteChanges, err := m.hasRemoteChanges(absPath, info.Branch)
	if err != nil {
		return fmt.Errorf("failed to check for remote changes: %w", err)
	}

	if !hasRemoteChanges {
		return nil // No changes to pull
	}

	// If auto-pull is disabled, don't pull
	if !m.autoPull {
		return nil // Auto-pull is disabled, skip pulling
	}

	// Auto-pull is enabled, ask for confirmation
	message := fmt.Sprintf(
		"Remote changes detected for worktree '%s' (branch: %s). Pull changes?",
		absPath,
		info.Branch,
	)

	confirmed, err := m.confirmFunc(message)
	if err != nil {
		return fmt.Errorf("confirmation error: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("pull cancelled by user")
	}

	// Perform the pull
	return m.pullWorktree(absPath)
}

// SyncAllWorktrees syncs all worktrees in the repository
func (m *Manager) SyncAllWorktrees() error {
	worktrees, err := m.detector.ListWorktrees()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	var errors []string
	for _, worktree := range worktrees {
		err := m.SyncWorktree(worktree.Path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("worktree %s: %v", worktree.Path, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to sync some worktrees:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// CreateWorktree creates a new worktree
func (m *Manager) CreateWorktree(path, branch string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get main repo path
	mainRepoPath, err := m.detector.getMainRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get main repo path: %w", err)
	}

	// Create the worktree
	cmd := exec.Command("git", "worktree", "add", absPath, branch)
	cmd.Dir = mainRepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %s: %w", string(output), err)
	}

	// If auto-pull is enabled, sync the new worktree
	if m.autoPull {
		return m.SyncWorktree(absPath)
	}

	return nil
}

// RemoveWorktree removes a worktree
func (m *Manager) RemoveWorktree(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify it's a worktree
	isWorktree, err := m.detector.IsWorktree(absPath)
	if err != nil {
		return fmt.Errorf("failed to check if worktree: %w", err)
	}

	if !isWorktree {
		return fmt.Errorf("path is not a Git worktree: %s", absPath)
	}

	// Get main repo path
	mainRepoPath, err := m.detector.getMainRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get main repo path: %w", err)
	}

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", absPath)
	cmd.Dir = mainRepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %s: %w", string(output), err)
	}

	return nil
}

// GetWorktreeInfo gets information about a worktree
func (m *Manager) GetWorktreeInfo(path string) (*WorktreeInfo, error) {
	return m.detector.DetectWorktree(path)
}

// ListWorktrees lists all worktrees
func (m *Manager) ListWorktrees() ([]*WorktreeInfo, error) {
	return m.detector.ListWorktrees()
}

// hasRemoteChanges checks if there are remote changes to pull
func (m *Manager) hasRemoteChanges(path, branch string) (bool, error) {
	// Fetch first to get latest remote info
	cmd := exec.Command("git", "fetch", "--quiet")
	cmd.Dir = path
	err := cmd.Run()
	if err != nil {
		// If fetch fails, assume no changes
		return false, nil
	}

	// Check if branch is tracking a remote branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	cmd.Dir = path
	err = cmd.Run()
	if err != nil {
		// No upstream branch, so no remote changes
		return false, nil
	}

	// Compare local and remote
	cmd = exec.Command("git", "rev-list", "--count", branch+".."+branch+"@{upstream}")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// pullWorktree performs a git pull on the worktree
func (m *Manager) pullWorktree(path string) error {
	cmd := exec.Command("git", "pull", "--quiet")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s: %w", string(output), err)
	}

	return nil
}

// DefaultConfirmationFunc provides a default confirmation function that reads from stdin
func DefaultConfirmationFunc(message string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", message)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}
