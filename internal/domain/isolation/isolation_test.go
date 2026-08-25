package isolation_test

import (
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/isolation"
)

func TestSession_IsActive(t *testing.T) {
	tests := []struct {
		name string
		sess *isolation.Session
		want bool
	}{
		{"nil session", nil, false},
		{"mode none", &isolation.Session{Mode: isolation.ModeNone}, false},
		{"mode worktree empty path", &isolation.Session{Mode: isolation.ModeWorktree}, false},
		{"mode worktree with path", &isolation.Session{Mode: isolation.ModeWorktree, WorktreePath: "/tmp/wt"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sess.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_Fields(t *testing.T) {
	now := time.Now()
	s := &isolation.Session{
		Mode:         isolation.ModeWorktree,
		WorktreePath: "/home/user/.skillrunner/worktrees/refactor-2026-02-26-abc1",
		RepoRoot:     "/home/user/project",
		Branch:       "sr/isolate/refactor-2026-02-26-abc1",
		SkillName:    "refactor",
		CreatedAt:    now,
	}

	if s.Mode != isolation.ModeWorktree {
		t.Errorf("Mode = %q, want %q", s.Mode, isolation.ModeWorktree)
	}
	if !s.IsActive() {
		t.Error("IsActive() = false for populated session")
	}
	if s.CreatedAt != now {
		t.Error("CreatedAt not preserved")
	}
}
