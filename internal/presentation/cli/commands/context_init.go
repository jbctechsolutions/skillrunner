// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/filesystem"
)

// NewContextInitCmd creates the init subcommand for context.
func NewContextInitCmd() *cobra.Command {
	var repoPath string

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize .skillrunner directory in a repository",
		Long: `Initialize the .skillrunner directory structure in a git repository.

This creates:
  • .skillrunner/ - Main directory for workspace context
  • .skillrunner/checkpoints/ - Directory for checkpoint files
  • .skillrunner/rules.md - Markdown file for workspace rules

The .skillrunner directory stores workspace-specific configuration,
checkpoints, and context that persists across skill executions.`,
		Example: `  # Initialize in current directory
  sr context init

  # Initialize in specific repository
  sr context init /path/to/repo`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			formatter := GetFormatter()

			// Determine repo path
			if len(args) > 0 {
				repoPath = args[0]
			} else if repoPath == "" {
				// Use current directory
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
				repoPath = cwd
			}

			// Validate it's a git repo
			workspaceFS := filesystem.NewWorkspaceFS()
			if !workspaceFS.IsGitRepo(repoPath) {
				return fmt.Errorf("not a git repository: %s", repoPath)
			}

			// Check if already initialized
			if workspaceFS.Exists(repoPath) {
				formatter.Warning(".skillrunner directory already exists")
				return nil
			}

			// Initialize directory structure
			if err := workspaceFS.InitDirectory(repoPath); err != nil {
				return fmt.Errorf("failed to initialize workspace: %w", err)
			}

			formatter.Success("Initialized .skillrunner directory in %s", repoPath)
			formatter.Println("")
			formatter.Info("Created:")
			formatter.BulletItem(".skillrunner/")
			formatter.BulletItem(".skillrunner/checkpoints/")
			formatter.BulletItem(".skillrunner/rules.md")
			formatter.Println("")
			formatter.Info("Next steps:")
			formatter.BulletItem("Edit .skillrunner/rules.md to add workspace-specific rules")
			formatter.BulletItem("Run 'sr context focus <issue>' to set your current task")
			formatter.BulletItem("Run 'sr context checkpoint' to save progress")

			_ = ctx // Silence unused variable warning
			return nil
		},
	}

	cmd.Flags().StringVarP(&repoPath, "path", "p", "", "path to repository (default: current directory)")

	return cmd
}
