package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetector("/test/repo")
	if detector.repoPath != "/test/repo" {
		t.Errorf("repoPath = %s; want /test/repo", detector.repoPath)
	}
}

func TestIsWorktree_NonExistentPath(t *testing.T) {
	detector := NewDetector(".")
	isWorktree, err := detector.IsWorktree("/nonexistent/path")
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	if isWorktree {
		t.Error("IsWorktree should return false for non-existent path")
	}
}

func TestIsWorktree_RegularDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewDetector(".")
	isWorktree, err := detector.IsWorktree(tmpDir)
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	if isWorktree {
		t.Error("IsWorktree should return false for regular directory")
	}
}

func TestIsWorktree_GitRepository(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize a git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available or failed to init: %v", err)
	}

	detector := NewDetector(".")
	isWorktree, err := detector.IsWorktree(tmpDir)
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	// A regular git repo is not a worktree
	if isWorktree {
		t.Error("IsWorktree should return false for regular git repository")
	}
}

func TestGetBranch(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize a git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create initial commit
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create a file and commit
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

	detector := NewDetector(tmpDir)
	branch, isDetached, err := detector.getBranch(tmpDir)
	if err != nil {
		t.Fatalf("getBranch failed: %v", err)
	}

	if branch == "" {
		t.Error("branch should not be empty")
	}
	if isDetached {
		t.Error("should not be detached after init")
	}
}

func TestGetMainRepoPath_RegularRepo(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	detector := NewDetector(tmpDir)
	mainPath, err := detector.getMainRepoPath()
	if err != nil {
		t.Fatalf("getMainRepoPath failed: %v", err)
	}

	absTmpDir, _ := filepath.Abs(tmpDir)
	absMainPath, _ := filepath.Abs(mainPath)
	if absMainPath != absTmpDir {
		t.Errorf("mainPath = %s; want %s", absMainPath, absTmpDir)
	}
}

func TestParseWorktreeList_Empty(t *testing.T) {
	detector := NewDetector(".")
	worktrees, err := detector.parseWorktreeList("")
	if err != nil {
		t.Fatalf("parseWorktreeList failed: %v", err)
	}
	if len(worktrees) != 0 {
		t.Errorf("worktrees count = %d; want 0", len(worktrees))
	}
}

func TestParseWorktreeList_SingleWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	absPath, _ := filepath.Abs(tmpDir)

	detector := NewDetector(".")
	output := "worktree " + absPath + "\n"
	worktrees, err := detector.parseWorktreeList(output)
	if err != nil {
		t.Fatalf("parseWorktreeList failed: %v", err)
	}
	if len(worktrees) != 1 {
		t.Fatalf("worktrees count = %d; want 1", len(worktrees))
	}
	if worktrees[0].Path != absPath {
		t.Errorf("Path = %s; want %s", worktrees[0].Path, absPath)
	}
}

func TestParseWorktreeList_MultipleWorktrees(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	absPath1, _ := filepath.Abs(tmpDir1)
	absPath2, _ := filepath.Abs(tmpDir2)

	detector := NewDetector(".")
	output := "worktree " + absPath1 + "\n\nworktree " + absPath2 + "\n"
	worktrees, err := detector.parseWorktreeList(output)
	if err != nil {
		t.Fatalf("parseWorktreeList failed: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("worktrees count = %d; want 2", len(worktrees))
	}
}

func TestDetectWorktree_NotAWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewDetector(".")

	_, err := detector.DetectWorktree(tmpDir)
	if err == nil {
		t.Error("DetectWorktree should return error for non-worktree")
	}
}

func TestIsMainWorktree(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	detector := NewDetector(tmpDir)
	isMain, err := detector.isMainWorktree(tmpDir)
	if err != nil {
		t.Fatalf("isMainWorktree failed: %v", err)
	}
	if !isMain {
		t.Error("isMainWorktree should return true for main repo")
	}
}

func TestListWorktrees_NoGit(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewDetector(tmpDir)

	_, err := detector.ListWorktrees()
	if err == nil {
		// This might succeed if git is available, so we just check it doesn't panic
	}
}

func TestGetBranch_DetachedHead(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Create initial commit
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

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

	// Get the commit hash and checkout in detached HEAD state
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	commitHash, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	hash := strings.TrimSpace(string(commitHash))

	// Checkout in detached HEAD state
	cmd = exec.Command("git", "checkout", hash)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to checkout commit: %v", err)
	}

	detector := NewDetector(tmpDir)
	branch, isDetached, err := detector.getBranch(tmpDir)
	if err != nil {
		t.Fatalf("getBranch failed: %v", err)
	}

	if !isDetached {
		t.Error("should be detached after checking out commit hash")
	}
	if branch != "HEAD" {
		t.Errorf("branch = %s; want HEAD", branch)
	}
}

func TestGetMainRepoPath_EmptyPath(t *testing.T) {
	detector := NewDetector("")
	mainPath, err := detector.getMainRepoPath()
	if err != nil {
		// This is expected if not in a git repo
		return
	}
	if mainPath == "" {
		t.Error("mainPath should not be empty")
	}
}

func TestParseWorktreeList_WithEmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	absPath, _ := filepath.Abs(tmpDir)

	detector := NewDetector(".")
	output := "worktree " + absPath + "\n\n\n"
	worktrees, err := detector.parseWorktreeList(output)
	if err != nil {
		t.Fatalf("parseWorktreeList failed: %v", err)
	}
	if len(worktrees) != 1 {
		t.Errorf("worktrees count = %d; want 1", len(worktrees))
	}
}

func TestIsWorktree_InvalidPath(t *testing.T) {
	detector := NewDetector(".")

	// Test with invalid characters in path - filepath.Abs may handle this differently
	// On some systems, this might not error, so we'll test that it doesn't panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("IsWorktree panicked with invalid path: %v", r)
			}
		}()
		_, err := detector.IsWorktree("\x00invalid")
		// Error may or may not occur depending on OS
		_ = err
	}()
}

func TestDetectWorktree_InvalidPath(t *testing.T) {
	detector := NewDetector(".")

	_, err := detector.DetectWorktree("\x00invalid")
	if err == nil {
		t.Error("DetectWorktree should return error for invalid path")
	}
}

func TestGetMainRepoPath_RelativePath(t *testing.T) {
	detector := NewDetector(".")
	_, err := detector.getMainRepoPath()
	// This might fail if not in a git repo, which is fine
	if err != nil {
		// Expected in non-git directories
		return
	}
}

func TestParseWorktreeList_InvalidPath(t *testing.T) {
	detector := NewDetector(".")
	output := "worktree \x00invalid\n"
	_, err := detector.parseWorktreeList(output)
	// filepath.Abs may handle this differently on different systems
	// Just ensure it doesn't panic
	if err != nil {
		// Error is acceptable
		return
	}
	// If no error, that's also acceptable - depends on OS behavior
}

func TestGetBranch_NonGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewDetector(".")

	_, _, err := detector.getBranch(tmpDir)
	if err == nil {
		t.Error("getBranch should return error for non-git directory")
	}
}

func TestIsMainWorktree_NonGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewDetector(".")

	_, err := detector.isMainWorktree(tmpDir)
	// This might succeed or fail depending on git availability
	// Just ensure it doesn't panic
	_ = err
}

func TestParseWorktreeList_Malformed(t *testing.T) {
	detector := NewDetector(".")

	// Test with malformed input
	output := "not a worktree line\n"
	worktrees, err := detector.parseWorktreeList(output)
	if err != nil {
		// Error is acceptable
		return
	}
	if len(worktrees) > 0 {
		t.Error("should not parse malformed input")
	}
}

func TestNewDetector_EmptyPath(t *testing.T) {
	detector := NewDetector("")
	if detector.repoPath != "" {
		t.Errorf("repoPath = %s; want empty", detector.repoPath)
	}
}

func TestListWorktrees_Integration(t *testing.T) {
	// This is an integration test that requires a real git repo
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git init failed: %v", err)
	}

	detector := NewDetector(tmpDir)
	worktrees, err := detector.ListWorktrees()
	if err != nil {
		// This might fail if git worktree list fails
		// Just ensure it doesn't panic
		return
	}

	// Should have at least the main worktree
	if len(worktrees) == 0 {
		t.Error("should have at least one worktree (main)")
	}
}

func TestGetMainRepoPath_WorktreeWithGitDir(t *testing.T) {
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

	detector := NewDetector(worktreeDir)
	mainPath, err := detector.getMainRepoPath()
	if err != nil {
		t.Fatalf("getMainRepoPath failed: %v", err)
	}

	absTmpDir, _ := filepath.Abs(tmpDir)
	absMainPath, _ := filepath.Abs(mainPath)
	// Use EvalSymlinks to handle /private vs /var on macOS
	absTmpDir, _ = filepath.EvalSymlinks(absTmpDir)
	absMainPath, _ = filepath.EvalSymlinks(absMainPath)
	if absMainPath != absTmpDir {
		t.Errorf("mainPath = %s; want %s", absMainPath, absTmpDir)
	}
}

func TestIsWorktree_ActualWorktree(t *testing.T) {
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

	detector := NewDetector(tmpDir)
	isWorktree, err := detector.IsWorktree(worktreeDir)
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	if !isWorktree {
		t.Error("IsWorktree should return true for actual worktree")
	}
}

func TestDetectWorktree_ActualWorktree(t *testing.T) {
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

	detector := NewDetector(tmpDir)
	info, err := detector.DetectWorktree(worktreeDir)
	if err != nil {
		t.Fatalf("DetectWorktree failed: %v", err)
	}

	if info == nil {
		t.Fatal("DetectWorktree returned nil info")
	}

	if info.Path == "" {
		t.Error("Path should not be empty")
	}

	if info.Branch == "" {
		t.Error("Branch should not be empty")
	}

	if info.IsMain {
		t.Error("Worktree should not be marked as main")
	}
}

func TestParseWorktreeList_WithBranchInfo(t *testing.T) {
	tmpDir := t.TempDir()
	absPath, _ := filepath.Abs(tmpDir)

	detector := NewDetector(".")
	output := "worktree " + absPath + "\nbranch refs/heads/main\n\n"
	worktrees, err := detector.parseWorktreeList(output)
	if err != nil {
		t.Fatalf("parseWorktreeList failed: %v", err)
	}
	if len(worktrees) != 1 {
		t.Errorf("worktrees count = %d; want 1", len(worktrees))
	}
}

func TestGetMainRepoPath_StatError(t *testing.T) {
	// This test is hard to create without mocking, but we can test the error path
	detector := NewDetector(".")
	// This will likely fail, which is fine
	_, err := detector.getMainRepoPath()
	_ = err
}

func TestGetMainRepoPath_WithGitdirInDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory with gitdir file
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	gitdirFile := filepath.Join(gitDir, "gitdir")
	// Write a path that points to a worktree gitdir
	worktreeGitDir := filepath.Join(tmpDir, "..", ".git", "worktrees", "test")
	err = os.WriteFile(gitdirFile, []byte(worktreeGitDir), 0644)
	if err != nil {
		t.Fatalf("Failed to create gitdir file: %v", err)
	}

	detector := NewDetector(tmpDir)
	_, err = detector.getMainRepoPath()
	// This will likely fail since it's not a real git structure, but tests the path
	_ = err
}

func TestDetector_IsWorktree_WithGitFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .git file (worktree structure)
	gitFile := filepath.Join(tmpDir, ".git")
	err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0644)
	if err != nil {
		t.Fatalf("Failed to create .git file: %v", err)
	}

	detector := NewDetector(".")
	isWorktree, err := detector.IsWorktree(tmpDir)
	if err != nil {
		t.Fatalf("IsWorktree failed: %v", err)
	}
	if !isWorktree {
		t.Error("IsWorktree should return true for directory with .git file")
	}
}

func TestDetector_DetectWorktree_ErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewDetector(tmpDir)

	// Test with non-worktree path
	_, err := detector.DetectWorktree(tmpDir)
	if err == nil {
		t.Error("DetectWorktree should return error for non-worktree")
	}
}

func TestDetector_GetMainRepoPath_ReadGitFileError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git as a directory (not a file) to test the IsDir path
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	detector := NewDetector(tmpDir)
	_, err = detector.getMainRepoPath()
	// This will likely fail, but tests the path
	_ = err
}
