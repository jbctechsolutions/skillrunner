// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewContextRulesCmd creates the rules subcommand for context.
func NewContextRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Manage workspace rules and guidelines",
		Long: `Manage rules and guidelines that apply during skill execution.

Rules can have different scopes:
  • Global - Apply to all workspaces
  • Workspace - Apply to specific workspace (stored in .skillrunner/rules.md)
  • Session - Apply to specific session

Active rules are included in the headline context for skill executions.`,
		Example: `  # Add a workspace rule
  sr context rules add "Code Style" --content "Follow PEP 8 guidelines"

  # List all rules
  sr context rules list

  # Activate a rule
  sr context rules activate "Code Style"

  # Deactivate a rule
  sr context rules deactivate "Code Style"`,
	}

	// Add subcommand
	addCmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			content, _ := cmd.Flags().GetString("content")
			scope, _ := cmd.Flags().GetString("scope")

			if content == "" {
				return fmt.Errorf("content is required (use --content)")
			}

			// TODO: Implement add rule logic
			formatter.Success("Added %s rule: %s", scope, name)

			return nil
		},
	}

	addCmd.Flags().String("content", "", "rule content/description")
	addCmd.Flags().String("scope", "workspace", "rule scope: global, workspace, session")
	addCmd.MarkFlagRequired("content")

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			scope, _ := cmd.Flags().GetString("scope")
			activeOnly, _ := cmd.Flags().GetBool("active")

			// TODO: Implement list logic
			formatter.Header("Rules")
			if scope != "" {
				formatter.Info("Scope: %s", scope)
			}
			if activeOnly {
				formatter.Info("Active rules only")
			}
			formatter.Println("")
			formatter.Info("No rules found")

			return nil
		},
	}

	listCmd.Flags().String("scope", "", "filter by scope: global, workspace, session")
	listCmd.Flags().Bool("active", false, "show only active rules")

	// Activate subcommand
	activateCmd := &cobra.Command{
		Use:   "activate <name>",
		Short: "Activate a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			// TODO: Implement activate logic
			formatter.Success("Activated rule: %s", name)

			return nil
		},
	}

	// Deactivate subcommand
	deactivateCmd := &cobra.Command{
		Use:   "deactivate <name>",
		Short: "Deactivate a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			// TODO: Implement deactivate logic
			formatter.Success("Deactivated rule: %s", name)

			return nil
		},
	}

	cmd.AddCommand(addCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(activateCmd)
	cmd.AddCommand(deactivateCmd)

	return cmd
}
