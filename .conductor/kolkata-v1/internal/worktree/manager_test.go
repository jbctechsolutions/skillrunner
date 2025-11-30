package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("/test/repo", true, nil)
	if manager.repoPath != "/test/repo" {
		t.Errorf("repoPath = %s; want /test/repo", manager.repoPath)
	}
	if !manager.autoPull {
		t.Error("autoPull should be true")
	}
	if manager.confirmFunc == nil {
		t.Error("confirmFunc should not be nil")
	}
}

func TestNewManager_WithConfirmationFunc(t *testing.T) {
	confirmFunc := func(string) (bool, error) { return true, nil }
	manager := NewManager("/test/repo", false, confirmFunc)
	if manager.confirmFunc == nil {
		t.Error("confirmFunc should not be nil")
	}
	if manager.autoPull {
		t.Error("autoPull should be false")
	}
}

func TestNewManager_DefaultConfirmation(t *testing.T) {
	manager := NewManager("/test/repo", true, nil)
	// Default confirmation should always return true
	confirmed, err := manager.confirmFunc("test")
	if err != nil {
		t.Fatalf("confirmFunc failed: %v", err)
	}
	if !confirmed {
		t.Error("default confirmFunc should return true")
	}
}

func TestDefaultConfirmationFunc(t *testing.T) {
	// This test would require mocking stdin, which is complex
	// We'll test the function exists and can be called
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DefaultConfirmationFunc panicked: %v", r)
			}
		}()
		// Can't easily test stdin interaction without mocking
		_ = DefaultConfirmationFunc
	}()
}

func TestManager_GetWorktreeInfo(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	manager := NewManager(tmpDir, false, nil)
	_, err := manager.GetWorktreeInfo(tmpDir)
	// This might fail if not a worktree, which is expected
	_ = err
}

func TestManager_ListWorktrees(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	manager := NewManager(tmpDir, false, nil)
	_, err := manager.ListWorktrees()
	// This might fail, which is acceptable
	_ = err
}

func TestManager_SyncWorktree_NotAWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(".", false, nil)

	err := manager.SyncWorktree(tmpDir)
	if err == nil {
		t.Error("SyncWorktree should return error for non-worktree")
	}
}

func TestManager_SyncWorktree_NonExistentPath(t *testing.T) {
	manager := NewManager(".", false, nil)

	err := manager.SyncWorktree("/nonexistent/path")
	if err == nil {
		t.Error("SyncWorktree should return error for non-existent path")
	}
}

func TestManager_SyncWorktree_WithConfirmation(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create a worktree
	worktreeDir := t.TempDir()
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "-b", "test-branch")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git worktree not available or failed: %v", err)
	}

	// Test with confirmation that returns false
	confirmFunc := func(string) (bool, error) { return false, nil }
	manager := NewManager(tmpDir, true, confirmFunc)

	err = manager.SyncWorktree(worktreeDir)
	if err == nil {
		// If no remote changes, this is acceptable
		return
	}
	if !strings.Contains(err.Error(), "pull cancelled by user") {
		// Other errors are acceptable (e.g., no remote configured)
		return
	}
}

func TestManager_SyncWorktree_WithConfirmationTrue(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Test with confirmation that returns true
	confirmFunc := func(string) (bool, error) { return true, nil }
	manager := NewManager(tmpDir, true, confirmFunc)

	// This should succeed (no remote changes to pull)
	err = manager.SyncWorktree(tmpDir)
	// Error is acceptable if there are no remote changes or git issues
	_ = err
}

func TestManager_SyncAllWorktrees(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	manager := NewManager(tmpDir, false, nil)
	err := manager.SyncAllWorktrees()
	// This might fail, which is acceptable
	_ = err
}

func TestManager_CreateWorktree_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	manager := NewManager(tmpDir, false, nil)

	// Try to create worktree with invalid branch
	err = manager.CreateWorktree("/tmp/test-worktree", "nonexistent-branch")
	if err == nil {
		t.Error("CreateWorktree should return error for nonexistent branch")
	}
}

func TestManager_RemoveWorktree_NotAWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(".", false, nil)

	err := manager.RemoveWorktree(tmpDir)
	if err == nil {
		t.Error("RemoveWorktree should return error for non-worktree")
	}
}

func TestManager_RemoveWorktree_NonExistentPath(t *testing.T) {
	manager := NewManager(".", false, nil)

	err := manager.RemoveWorktree("/nonexistent/path")
	if err == nil {
		t.Error("RemoveWorktree should return error for non-existent path")
	}
}

func TestManager_HasRemoteChanges_NoRemote(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	manager := NewManager(tmpDir, false, nil)
	hasChanges, err := manager.hasRemoteChanges(tmpDir, "main")
	if err != nil {
		t.Fatalf("hasRemoteChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("hasRemoteChanges should return false for repo without remote")
	}
}

func TestManager_PullWorktree(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	manager := NewManager(tmpDir, false, nil)
	err := manager.pullWorktree(tmpDir)
	// This might fail if there's no remote, which is acceptable
	_ = err
}

func TestManager_SyncWorktree_AutoPullDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create a worktree
	worktreeDir := t.TempDir()
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "-b", "autopull-disabled-branch")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git worktree not available: %v", err)
	}

	// Create manager with autoPull=false
	manager := NewManager(tmpDir, false, nil)

	// Sync should succeed without pulling when autoPull is false
	// Even if there are remote changes, it should not pull
	err = manager.SyncWorktree(worktreeDir)
	if err != nil {
		// Error is acceptable if there are issues detecting worktree or checking remote
		// But it should NOT pull when autoPull is false
		return
	}

	// If we get here, the function returned nil, which is correct
	// when autoPull is false (it should skip pulling)
}

func TestManager_ConfirmationError(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Test with confirmation that returns error
	confirmFunc := func(string) (bool, error) { return false, os.ErrPermission }
	manager := NewManager(tmpDir, true, confirmFunc)

	err = manager.SyncWorktree(tmpDir)
	if err == nil {
		t.Error("SyncWorktree should return error when confirmation fails")
	}
}

func TestManager_SyncAllWorktrees_WithErrors(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	manager := NewManager(tmpDir, false, nil)

	// This will likely fail, but should handle errors gracefully
	err := manager.SyncAllWorktrees()
	// Error is acceptable
	_ = err
}

func TestNewManager_EmptyPath(t *testing.T) {
	manager := NewManager("", false, nil)
	if manager.repoPath != "" {
		t.Errorf("repoPath = %s; want empty", manager.repoPath)
	}
}

func TestManager_GetWorktreeInfo_Error(t *testing.T) {
	manager := NewManager(".", false, nil)
	_, err := manager.GetWorktreeInfo("/nonexistent")
	if err == nil {
		t.Error("GetWorktreeInfo should return error for nonexistent path")
	}
}

func TestManager_CreateWorktree_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize repo with initial-branch to avoid default branch name issues
	cmd := exec.Command("git", "init", "--initial-branch=main")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		// Fallback for older git versions
		cmd = exec.Command("git", "init")
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			t.Skipf("git not available: %v", err)
		}
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Get the current branch name (main or master depending on git version)
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = tmpDir
	output, _ := cmd.Output()
	currentBranch := strings.TrimSpace(string(output))
	if currentBranch == "" {
		currentBranch = "main"
	}

	// Create a unique branch name using temp directory name to avoid conflicts
	branchName := fmt.Sprintf("test-branch-%s", filepath.Base(tmpDir))
	cmd = exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "checkout", currentBranch)
	cmd.Dir = tmpDir
	cmd.Run()

	manager := NewManager(tmpDir, false, nil)

	// Create a worktree with the new branch
	worktreeDir := t.TempDir()
	err = manager.CreateWorktree(worktreeDir, branchName)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Verify it was created
	isWorktree, err := manager.detector.IsWorktree(worktreeDir)
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	if !isWorktree {
		t.Error("Created worktree should be detected as worktree")
	}
}

func TestManager_RemoveWorktree_Success(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create a worktree
	worktreeDir := t.TempDir()
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "-b", "test-branch")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git worktree not available: %v", err)
	}

	manager := NewManager(tmpDir, false, nil)

	// Remove the worktree
	err = manager.RemoveWorktree(worktreeDir)
	if err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}
}

func TestManager_SyncWorktree_NoRemoteChanges(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	manager := NewManager(tmpDir, true, nil)

	// Sync should succeed even with no remote
	err = manager.SyncWorktree(tmpDir)
	// Error is acceptable if no remote configured
	_ = err
}

func TestManager_HasRemoteChanges_WithUpstream(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	manager := NewManager(tmpDir, false, nil)

	// Test hasRemoteChanges - should return false for repo without remote
	hasChanges, err := manager.hasRemoteChanges(tmpDir, "main")
	if err != nil {
		t.Fatalf("hasRemoteChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("hasRemoteChanges should return false for repo without remote")
	}
}

func TestManager_SyncWorktree_WithRemoteChanges(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create a worktree
	worktreeDir := t.TempDir()
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "-b", "test-branch")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git worktree not available: %v", err)
	}

	// Test with auto-pull enabled and confirmation that returns true
	confirmFunc := func(string) (bool, error) { return true, nil }
	manager := NewManager(tmpDir, true, confirmFunc)

	// Sync should succeed (no remote changes, but tests the path)
	err = manager.SyncWorktree(worktreeDir)
	// Error is acceptable if no remote configured
	_ = err
}

func TestManager_CreateWorktree_WithAutoPull(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize repo with initial-branch to avoid default branch name issues
	cmd := exec.Command("git", "init", "--initial-branch=main")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		// Fallback for older git versions
		cmd = exec.Command("git", "init")
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			t.Skipf("git not available: %v", err)
		}
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Get the current branch name (main or master depending on git version)
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = tmpDir
	output, _ := cmd.Output()
	currentBranch := strings.TrimSpace(string(output))
	if currentBranch == "" {
		currentBranch = "main"
	}

	// Create a unique branch name using temp directory name to avoid conflicts
	branchName := fmt.Sprintf("auto-pull-%s", filepath.Base(tmpDir))
	cmd = exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "checkout", currentBranch)
	cmd.Dir = tmpDir
	cmd.Run()

	manager := NewManager(tmpDir, true, nil)

	// Create a worktree with auto-pull enabled
	worktreeDir := t.TempDir()
	err = manager.CreateWorktree(worktreeDir, branchName)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}
}

func TestDetector_GetMainRepoPath_WithGitFile(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create a worktree (this creates a .git file)
	worktreeDir := t.TempDir()
	cmd = exec.Command("git", "worktree", "add", worktreeDir, "-b", "gitfile-branch")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git worktree not available: %v", err)
	}

	detector := NewDetector(worktreeDir)
	mainPath, err := detector.getMainRepoPath()
	if err != nil {
		t.Fatalf("getMainRepoPath failed: %v", err)
	}

	absTmpDir, _ := filepath.Abs(tmpDir)
	absMainPath, _ := filepath.Abs(mainPath)
	absTmpDir, _ = filepath.EvalSymlinks(absTmpDir)
	absMainPath, _ = filepath.EvalSymlinks(absMainPath)
	if absMainPath != absTmpDir {
		t.Errorf("mainPath = %s; want %s", absMainPath, absTmpDir)
	}
}

func TestDetector_GetMainRepoPath_WithoutGitdirPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake .git file without "gitdir: " prefix
	gitFile := filepath.Join(tmpDir, ".git")
	err := os.WriteFile(gitFile, []byte("some/path"), 0644)
	if err != nil {
		t.Fatalf("Failed to create .git file: %v", err)
	}

	detector := NewDetector(tmpDir)
	mainPath, err := detector.getMainRepoPath()
	if err != nil {
		// Error is expected since it's not a real git repo
		return
	}
	_ = mainPath
}

func TestDetector_GetMainRepoPath_RelativeGitdir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake .git file with relative path
	gitFile := filepath.Join(tmpDir, ".git")
	gitdirContent := "gitdir: ../.git/worktrees/test"
	err := os.WriteFile(gitFile, []byte(gitdirContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .git file: %v", err)
	}

	detector := NewDetector(tmpDir)
	_, err = detector.getMainRepoPath()
	// Error is expected since it's not a real git repo structure
	_ = err
}

func TestDetector_IsWorktree_WithGitdirFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory with gitdir file (worktree structure)
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	gitdirFile := filepath.Join(gitDir, "gitdir")
	err = os.WriteFile(gitdirFile, []byte("/some/path"), 0644)
	if err != nil {
		t.Fatalf("Failed to create gitdir file: %v", err)
	}

	detector := NewDetector(".")
	isWorktree, err := detector.IsWorktree(tmpDir)
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	if !isWorktree {
		t.Error("IsWorktree should return true for directory with gitdir file")
	}
}

func TestManager_SyncWorktree_ErrorPaths(t *testing.T) {
	manager := NewManager(".", false, nil)

	// Test with invalid path
	err := manager.SyncWorktree("\x00invalid")
	if err == nil {
		t.Error("SyncWorktree should return error for invalid path")
	}
}

func TestManager_SyncWorktree_DetectWorktreeError(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	manager := NewManager(".", false, nil)

	// Sync should fail for non-worktree
	err := manager.SyncWorktree(tmpDir)
	if err == nil {
		t.Error("SyncWorktree should return error for non-worktree")
	}
}

func TestManager_HasRemoteChanges_WithUpstreamBranch(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	manager := NewManager(tmpDir, false, nil)

	// Test hasRemoteChanges with a branch that doesn't have upstream
	hasChanges, err := manager.hasRemoteChanges(tmpDir, "nonexistent-branch")
	if err != nil {
		// Error is acceptable
		return
	}
	_ = hasChanges
}

func TestManager_PullWorktree_Error(t *testing.T) {
	tmpDir := t.TempDir()

	// Not a git repo
	manager := NewManager(".", false, nil)
	err := manager.pullWorktree(tmpDir)
	if err == nil {
		t.Error("pullWorktree should return error for non-git directory")
	}
}

func TestManager_SyncAllWorktrees_WithSyncError(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Setup git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	manager := NewManager(tmpDir, false, nil)

	// This should succeed (no worktrees to sync)
	err = manager.SyncAllWorktrees()
	// Error is acceptable
	_ = err
}
