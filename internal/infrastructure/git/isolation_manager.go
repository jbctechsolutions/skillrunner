package git

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/isolation"
)

const (
	worktreeDirName = ".skillrunner/worktrees"
	staleDays       = 7
)

// IsolationManager creates and manages worktree-backed isolation sessions.
type IsolationManager struct {
	wm      *WorktreeManager
	baseDir string // root directory under which worktrees are created
}

// NewIsolationManager creates an IsolationManager rooted at baseDir.
// If baseDir is empty, it defaults to ~/.skillrunner/worktrees.
func NewIsolationManager(baseDir string) (*IsolationManager, error) {
	wm, err := NewWorktreeManager()
	if err != nil {
		return nil, err
	}

	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		baseDir = filepath.Join(home, worktreeDirName)
	}

	return &IsolationManager{wm: wm, baseDir: baseDir}, nil
}

// Setup creates a new worktree session for the given repo root and skill name.
// The worktree is created at baseDir/<skillName>-<date>-<hash>/.
func (im *IsolationManager) Setup(ctx context.Context, repoRoot, skillName string) (*isolation.Session, error) {
	// Ensure worktree base directory exists
	if err := os.MkdirAll(im.baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create worktree base dir: %w", err)
	}

	// Generate unique suffix
	suffix, err := randomHex(4)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random suffix: %w", err)
	}

	dateStr := time.Now().Format("2006-01-02")
	safeSkill := sanitizeLabel(skillName)
	label := fmt.Sprintf("%s-%s-%s", safeSkill, dateStr, suffix)
	branch := "sr/isolate/" + label
	worktreePath := filepath.Join(im.baseDir, label)

	if err := im.wm.Create(ctx, repoRoot, worktreePath, branch, true); err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	return &isolation.Session{
		Mode:         isolation.ModeWorktree,
		WorktreePath: worktreePath,
		RepoRoot:     repoRoot,
		Branch:       branch,
		SkillName:    skillName,
		CreatedAt:    time.Now(),
	}, nil
}

// Diff returns the unified diff of changes made in the worktree vs HEAD.
func (im *IsolationManager) Diff(ctx context.Context, sess *isolation.Session) (string, error) {
	if !sess.IsActive() {
		return "", nil
	}

	cmd := im.wm.command(ctx, "diff", "HEAD")
	cmd.Dir = sess.WorktreePath

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// Exit code 1 from git diff means there are differences — that's normal.
		return out.String(), nil
	}

	return out.String(), nil
}

// ChangedFiles returns paths (relative to repo root) of files changed in the worktree.
func (im *IsolationManager) ChangedFiles(ctx context.Context, sess *isolation.Session) ([]string, error) {
	if !sess.IsActive() {
		return nil, nil
	}

	cmd := im.wm.command(ctx, "status", "--porcelain")
	cmd.Dir = sess.WorktreePath

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if len(line) < 4 {
			continue
		}
		// git status --porcelain: "XY filename" where XY are status codes
		relPath := strings.TrimSpace(line[3:])
		if relPath != "" {
			files = append(files, relPath)
		}
	}

	return files, nil
}

// Apply copies all changed files from the worktree back to the repo root.
func (im *IsolationManager) Apply(ctx context.Context, sess *isolation.Session) error {
	if !sess.IsActive() {
		return nil
	}

	files, err := im.ChangedFiles(ctx, sess)
	if err != nil {
		return err
	}

	for _, relPath := range files {
		src := filepath.Join(sess.WorktreePath, relPath)
		dst := filepath.Join(sess.RepoRoot, relPath)

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", relPath, err)
		}
	}

	return nil
}

// Discard removes the worktree without applying any changes.
func (im *IsolationManager) Discard(ctx context.Context, sess *isolation.Session) error {
	if !sess.IsActive() {
		return nil
	}
	return im.wm.Remove(ctx, sess.RepoRoot, sess.WorktreePath, true)
}

// ListWorktrees returns all sessions in the base directory (for management/cleanup).
func (im *IsolationManager) ListWorktrees() ([]isolation.Session, error) {
	entries, err := os.ReadDir(im.baseDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read worktree dir: %w", err)
	}

	var sessions []isolation.Session
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		sessions = append(sessions, isolation.Session{
			Mode:         isolation.ModeWorktree,
			WorktreePath: filepath.Join(im.baseDir, e.Name()),
			SkillName:    e.Name(),
			CreatedAt:    info.ModTime(),
		})
	}

	return sessions, nil
}

// CleanStale removes worktrees older than staleDays days.
// It returns the number of worktrees cleaned.
func (im *IsolationManager) CleanStale(ctx context.Context, repoRoot string) (int, error) {
	sessions, err := im.ListWorktrees()
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -staleDays)
	removed := 0

	for _, sess := range sessions {
		if sess.CreatedAt.Before(cutoff) {
			localSess := sess // avoid loop variable capture
			localSess.RepoRoot = repoRoot
			if err := im.Discard(ctx, &localSess); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}

// command is a helper that builds an exec.Cmd using the worktree manager's git path.
func (wm *WorktreeManager) command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, wm.gitPath, args...)
}

// randomHex returns n random hex bytes.
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sanitizeLabel replaces characters unsuitable for directory/branch names with hyphens.
func sanitizeLabel(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	result := sb.String()
	if len(result) > 30 {
		result = result[:30]
	}
	return result
}

// copyFile copies src to dst, creating dst if it does not exist.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
