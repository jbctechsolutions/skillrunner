// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewContextFocusCmd creates the focus subcommand for context.
func NewContextFocusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "focus <issue-id>",
		Short: "Set the current task focus for the workspace",
		Long: `Set the current task or issue focus for the workspace.

The focus is included in the headline context for skill executions,
helping the AI understand what you're currently working on.`,
		Example: `  # Set focus to an issue
  sr context focus ISSUE-123

  # Clear focus
  sr context focus --clear`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			clear, _ := cmd.Flags().GetBool("clear")

			if clear {
				formatter.Success("Focus cleared")
				// TODO: Implement clear focus logic
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("issue ID required (or use --clear to remove focus)")
			}

			issueID := args[0]

			// TODO: Implement set focus logic
			formatter.Success("Focus set to: %s", issueID)

			return nil
		},
	}

	cmd.Flags().Bool("clear", false, "clear the current focus")

	return cmd
}
