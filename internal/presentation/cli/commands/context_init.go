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
				_ = formatter.Warning(".skillrunner directory already exists")
				return nil
			}

			// Initialize directory structure
			if err := workspaceFS.InitDirectory(repoPath); err != nil {
				return fmt.Errorf("failed to initialize workspace: %w", err)
			}

			_ = formatter.Success("Initialized .skillrunner directory in %s", repoPath)
			_ = formatter.Println("")
			_ = formatter.Info("Created:")
			_ = formatter.BulletItem(".skillrunner/")
			_ = formatter.BulletItem(".skillrunner/checkpoints/")
			_ = formatter.BulletItem(".skillrunner/rules.md")
			_ = formatter.Println("")
			_ = formatter.Info("Next steps:")
			_ = formatter.BulletItem("Edit .skillrunner/rules.md to add workspace-specific rules")
			_ = formatter.BulletItem("Run 'sr context focus <issue>' to set your current task")
			_ = formatter.BulletItem("Run 'sr context checkpoint' to save progress")

			_ = ctx // Silence unused variable warning
			return nil
		},
	}

	cmd.Flags().StringVarP(&repoPath, "path", "p", "", "path to repository (default: current directory)")

	return cmd
}
