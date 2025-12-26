// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewContextFocusCmd creates the focus subcommand for context.
func NewContextFocusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "focus [issue-id]",
		Short: "Set the current task focus for the workspace",
		Long: `Set the current task or issue focus for the workspace.

The focus is included in the headline context for skill executions,
helping the AI understand what you're currently working on.`,
		Example: `  # Set focus to an issue
  sr context focus ISSUE-123

  # Show current focus
  sr context focus --show

  # Clear focus
  sr context focus --clear`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			clear, _ := cmd.Flags().GetBool("clear")
			show, _ := cmd.Flags().GetBool("show")

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			wsRepo := container.WorkspaceRepository()

			// Get current workspace (by current directory)
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			ctx := context.Background()
			workspace, err := wsRepo.GetByRepoPath(ctx, cwd)
			if err != nil {
				// Create a helpful error message
				return fmt.Errorf("no workspace found for current directory. Use 'sr workspace init' to initialize a workspace")
			}

			// Handle --show flag
			if show {
				focus := workspace.Focus()
				if focus == "" {
					formatter.Info("No focus set for workspace: %s", workspace.Name())
				} else {
					formatter.Header("Current Focus")
					formatter.Info("Workspace: %s", workspace.Name())
					formatter.Info("Focus: %s", focus)
				}
				return nil
			}

			// Handle --clear flag
			if clear {
				if err := wsRepo.SetFocus(ctx, workspace.ID(), ""); err != nil {
					return fmt.Errorf("failed to clear focus: %w", err)
				}
				formatter.Success("Focus cleared for workspace: %s", workspace.Name())
				return nil
			}

			// Set focus
			if len(args) == 0 {
				// No arg provided, show current focus
				focus := workspace.Focus()
				if focus == "" {
					formatter.Info("No focus set for workspace: %s", workspace.Name())
					formatter.Info("Use 'sr context focus ISSUE-123' to set a focus")
				} else {
					formatter.Header("Current Focus")
					formatter.Info("Workspace: %s", workspace.Name())
					formatter.Info("Focus: %s", focus)
				}
				return nil
			}

			issueID := args[0]

			if err := wsRepo.SetFocus(ctx, workspace.ID(), issueID); err != nil {
				return fmt.Errorf("failed to set focus: %w", err)
			}

			formatter.Success("Focus set to: %s", issueID)
			formatter.Info("Workspace: %s", workspace.Name())

			return nil
		},
	}

	cmd.Flags().Bool("clear", false, "clear the current focus")
	cmd.Flags().Bool("show", false, "show the current focus")

	return cmd
}
