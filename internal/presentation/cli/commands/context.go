// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"github.com/spf13/cobra"
)

// NewContextCmd creates the context command for managing workspace context.
func NewContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage workspace context and checkpoints",
		Long: `Manage workspace context including initialization, checkpoints, focus, and rules.

The context system helps maintain state and guidelines for skill execution:
  • Initialize .skillrunner/ directory in your repo
  • Create and manage checkpoints to save session state
  • Set focus on specific tasks or issues
  • Add context items (files, snippets, URLs) for reference
  • Define and manage rules for skill execution`,
		Example: `  # Initialize .skillrunner directory
  sr context init

  # Set focus on an issue
  sr context focus ISSUE-123

  # Create a checkpoint
  sr context checkpoint "Completed authentication module"

  # Add a context item
  sr context items add --file ./docs/architecture.md

  # List active rules
  sr context rules list`,
	}

	// Add subcommands
	cmd.AddCommand(NewContextInitCmd())
	cmd.AddCommand(NewContextCheckpointCmd())
	cmd.AddCommand(NewContextFocusCmd())
	cmd.AddCommand(NewContextItemsCmd())
	cmd.AddCommand(NewContextRulesCmd())

	return cmd
}
