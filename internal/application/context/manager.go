// Package context provides application-level context management services.
package context

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
)

const (
	// SkillrunnerDir is the name of the skillrunner workspace directory.
	SkillrunnerDir = ".skillrunner"

	// RulesFile is the name of the rules markdown file.
	RulesFile = "rules.md"

	// CheckpointsDir is the name of the checkpoints directory.
	CheckpointsDir = "checkpoints"
)

// Manager handles context management operations for workspaces.
type Manager struct {
	workspaceRepo  ports.WorkspaceStateStoragePort
	sessionRepo    ports.WorkflowSessionStoragePort
	checkpointRepo ports.CheckpointStoragePort
	itemRepo       ports.ContextItemStoragePort
	ruleRepo       ports.RuleStoragePort
	estimator      *Estimator
}

// NewManager creates a new context manager.
func NewManager(
	workspaceRepo ports.WorkspaceStateStoragePort,
	sessionRepo ports.WorkflowSessionStoragePort,
	checkpointRepo ports.CheckpointStoragePort,
	itemRepo ports.ContextItemStoragePort,
	ruleRepo ports.RuleStoragePort,
) *Manager {
	return &Manager{
		workspaceRepo:  workspaceRepo,
		sessionRepo:    sessionRepo,
		checkpointRepo: checkpointRepo,
		itemRepo:       itemRepo,
		ruleRepo:       ruleRepo,
		estimator:      NewEstimator(),
	}
}

// InitWorkspace initializes the .skillrunner directory in a repository.
// Creates the directory structure and default files.
func (m *Manager) InitWorkspace(ctx context.Context, repoPath string) (*domainContext.Workspace, error) {
	// Validate repo path
	if repoPath == "" {
		return nil, fmt.Errorf("repository path is required")
	}

	// Check if .git exists to confirm it's a git repo
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Create .skillrunner directory
	skillrunnerPath := filepath.Join(repoPath, SkillrunnerDir)
	if err := os.MkdirAll(skillrunnerPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .skillrunner directory: %w", err)
	}

	// Create checkpoints subdirectory
	checkpointsPath := filepath.Join(skillrunnerPath, CheckpointsDir)
	if err := os.MkdirAll(checkpointsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoints directory: %w", err)
	}

	// Create default rules.md file if it doesn't exist
	rulesPath := filepath.Join(skillrunnerPath, RulesFile)
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		defaultRules := `# Workspace Rules

Add your custom rules and guidelines for this workspace here.
These rules will be included in the context for skill execution.

## Example Rules

- Follow existing code style and patterns
- Write tests for new functionality
- Update documentation when changing APIs
`
		if err := os.WriteFile(rulesPath, []byte(defaultRules), 0644); err != nil {
			return nil, fmt.Errorf("failed to create rules.md: %w", err)
		}
	}

	// Generate workspace ID from repo path
	workspaceID := generateWorkspaceID(repoPath)
	workspaceName := filepath.Base(repoPath)

	// Check if workspace already exists
	exists, err := m.workspaceRepo.Exists(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check workspace existence: %w", err)
	}

	var workspace *domainContext.Workspace
	if exists {
		// Load existing workspace
		workspace, err = m.workspaceRepo.Get(ctx, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to load workspace: %w", err)
		}
		workspace.Activate()
		if err := m.workspaceRepo.Update(ctx, workspace); err != nil {
			return nil, fmt.Errorf("failed to update workspace: %w", err)
		}
	} else {
		// Create new workspace
		workspace, err = domainContext.NewWorkspace(workspaceID, workspaceName, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create workspace: %w", err)
		}

		if err := m.workspaceRepo.Create(ctx, workspace); err != nil {
			return nil, fmt.Errorf("failed to save workspace: %w", err)
		}
	}

	return workspace, nil
}

// SetFocus sets the current task focus for a workspace.
func (m *Manager) SetFocus(ctx context.Context, workspaceID, issueID string) error {
	workspace, err := m.workspaceRepo.Get(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	workspace.SetFocus(issueID)
	workspace.Touch()

	if err := m.workspaceRepo.Update(ctx, workspace); err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	return nil
}

// CreateCheckpoint saves a checkpoint for the current session.
func (m *Manager) CreateCheckpoint(ctx context.Context, sessionID, summary string) (*domainContext.Checkpoint, error) {
	session, err := m.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	checkpointID := generateCheckpointID()
	checkpoint, err := domainContext.NewCheckpoint(checkpointID, session.WorkspaceID, sessionID, summary)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint: %w", err)
	}

	if err := m.checkpointRepo.Save(ctx, checkpoint); err != nil {
		return nil, fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return checkpoint, nil
}

// GetHeadlineContext generates a compact context summary for a workspace.
// Aims to stay under 500 tokens.
func (m *Manager) GetHeadlineContext(ctx context.Context, workspaceID string) (string, int, error) {
	workspace, err := m.workspaceRepo.Get(ctx, workspaceID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Build headline context
	var context string

	// Add focus if set
	if workspace.Focus() != "" {
		context += fmt.Sprintf("Current Focus: %s\n\n", workspace.Focus())
	}

	// Add active rules
	rules, err := m.ruleRepo.ListActive(ctx)
	if err == nil && len(rules) > 0 {
		context += "Active Rules:\n"
		for _, rule := range rules {
			// Truncate rule content if needed
			content := rule.Content()
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			context += fmt.Sprintf("- %s: %s\n", rule.Name(), content)
		}
		context += "\n"
	}

	// Estimate tokens
	tokens := m.estimator.Estimate(context)

	// If over budget, truncate
	if tokens > 500 {
		context, tokens = m.estimator.TruncateToFit(context, 500)
	}

	return context, tokens, nil
}

// generateWorkspaceID generates a workspace ID from the repo path.
func generateWorkspaceID(repoPath string) string {
	// Use the absolute path hash or a cleaned version
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		abs = repoPath
	}
	// For now, just use the base name; could use a hash for uniqueness
	return filepath.Base(abs)
}

// generateCheckpointID generates a unique checkpoint ID.
func generateCheckpointID() string {
	// In production, use UUID or similar
	// For now, use timestamp-based ID
	return fmt.Sprintf("cp_%d", os.Getpid())
}
