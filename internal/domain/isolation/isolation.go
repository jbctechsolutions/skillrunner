// Package isolation defines types for git worktree-based execution isolation.
package isolation

import "time"

// Mode describes how a skill execution is isolated.
type Mode string

const (
	// ModeNone runs the skill in the current working directory (default).
	ModeNone Mode = ""
	// ModeWorktree creates a temporary git worktree for the execution.
	ModeWorktree Mode = "worktree"
)

// Session represents an active isolation session backed by a git worktree.
type Session struct {
	Mode         Mode
	WorktreePath string    // Absolute path to the created worktree
	RepoRoot     string    // Absolute path to the source git repo
	Branch       string    // Branch created for this worktree
	SkillName    string    // Human-readable label
	CreatedAt    time.Time // When the session was established
}

// IsActive reports whether the session is an active worktree session.
func (s *Session) IsActive() bool {
	return s != nil && s.Mode == ModeWorktree && s.WorktreePath != ""
}
