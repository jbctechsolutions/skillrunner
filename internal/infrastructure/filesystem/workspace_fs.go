// Package filesystem provides filesystem operations for workspace management.
package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// SkillrunnerDir is the name of the skillrunner workspace directory.
	SkillrunnerDir = ".skillrunner"

	// CheckpointsDir is the subdirectory for checkpoints.
	CheckpointsDir = "checkpoints"

	// RulesFile is the rules markdown file.
	RulesFile = "rules.md"

	// DefaultRulesContent is the default content for rules.md.
	DefaultRulesContent = `# Workspace Rules

Add your custom rules and guidelines for this workspace here.
These rules will be included in the context for skill execution.

## Example Rules

- Follow existing code style and patterns
- Write tests for new functionality
- Update documentation when changing APIs
- Keep commits focused and atomic
`
)

// WorkspaceFS handles filesystem operations for workspaces.
type WorkspaceFS struct{}

// NewWorkspaceFS creates a new workspace filesystem handler.
func NewWorkspaceFS() *WorkspaceFS {
	return &WorkspaceFS{}
}

// InitDirectory initializes the .skillrunner directory structure.
func (fs *WorkspaceFS) InitDirectory(repoPath string) error {
	// Create main .skillrunner directory
	skillrunnerPath := filepath.Join(repoPath, SkillrunnerDir)
	if err := os.MkdirAll(skillrunnerPath, 0755); err != nil {
		return fmt.Errorf("failed to create .skillrunner directory: %w", err)
	}

	// Create checkpoints subdirectory
	checkpointsPath := filepath.Join(skillrunnerPath, CheckpointsDir)
	if err := os.MkdirAll(checkpointsPath, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoints directory: %w", err)
	}

	// Create rules.md if it doesn't exist
	rulesPath := filepath.Join(skillrunnerPath, RulesFile)
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		if err := os.WriteFile(rulesPath, []byte(DefaultRulesContent), 0644); err != nil {
			return fmt.Errorf("failed to create rules.md: %w", err)
		}
	}

	return nil
}

// GetSkillrunnerPath returns the .skillrunner directory path for a repo.
func (fs *WorkspaceFS) GetSkillrunnerPath(repoPath string) string {
	return filepath.Join(repoPath, SkillrunnerDir)
}

// GetRulesPath returns the rules.md file path for a repo.
func (fs *WorkspaceFS) GetRulesPath(repoPath string) string {
	return filepath.Join(repoPath, SkillrunnerDir, RulesFile)
}

// GetCheckpointsPath returns the checkpoints directory path for a repo.
func (fs *WorkspaceFS) GetCheckpointsPath(repoPath string) string {
	return filepath.Join(repoPath, SkillrunnerDir, CheckpointsDir)
}

// Exists checks if the .skillrunner directory exists.
func (fs *WorkspaceFS) Exists(repoPath string) bool {
	skillrunnerPath := fs.GetSkillrunnerPath(repoPath)
	_, err := os.Stat(skillrunnerPath)
	return !os.IsNotExist(err)
}

// EnsureExists ensures the .skillrunner directory exists, creating it if necessary.
func (fs *WorkspaceFS) EnsureExists(repoPath string) error {
	if !fs.Exists(repoPath) {
		return fs.InitDirectory(repoPath)
	}
	return nil
}

// IsGitRepo checks if the given path is a git repository.
func (fs *WorkspaceFS) IsGitRepo(repoPath string) bool {
	gitDir := filepath.Join(repoPath, ".git")
	info, err := os.Stat(gitDir)
	if os.IsNotExist(err) {
		return false
	}
	// .git can be a directory or a file (for worktrees)
	return err == nil && (info.IsDir() || !info.IsDir())
}
