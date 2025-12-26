// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
)

// NewContextRulesCmd creates the rules subcommand for context.
func NewContextRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Manage workspace rules and guidelines",
		Long: `Manage rules and guidelines that apply during skill execution.

Rules can have different scopes:
  - Global - Apply to all workspaces
  - Workspace - Apply to specific workspace (stored in .skillrunner/rules.md)
  - Session - Apply to specific session

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
			scopeStr, _ := cmd.Flags().GetString("scope")

			if content == "" {
				return fmt.Errorf("content is required (use --content)")
			}

			// Parse scope
			var scope domainContext.RuleScope
			switch scopeStr {
			case "global":
				scope = domainContext.RuleScopeGlobal
			case "workspace":
				scope = domainContext.RuleScopeWorkspace
			case "session":
				scope = domainContext.RuleScopeSession
			default:
				return fmt.Errorf("invalid scope: %s (must be global, workspace, or session)", scopeStr)
			}

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.RulesRepository()

			// Create the rule
			id := uuid.New().String()
			rule, err := domainContext.NewRule(id, name, content, scope)
			if err != nil {
				return fmt.Errorf("failed to create rule: %w", err)
			}

			// Save to repository
			ctx := context.Background()
			if err := repo.Save(ctx, rule); err != nil {
				return fmt.Errorf("failed to save rule: %w", err)
			}

			formatter.Success("Added %s rule: %s", scope, name)
			formatter.Info("ID: %s", id)

			return nil
		},
	}

	addCmd.Flags().String("content", "", "rule content/description")
	addCmd.Flags().String("scope", "workspace", "rule scope: global, workspace, session")
	_ = addCmd.MarkFlagRequired("content")

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			scopeStr, _ := cmd.Flags().GetString("scope")
			activeOnly, _ := cmd.Flags().GetBool("active")

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.RulesRepository()

			ctx := context.Background()
			var rules []*domainContext.Rule
			var err error

			// Apply filters
			if activeOnly {
				rules, err = repo.ListActive(ctx)
			} else if scopeStr != "" {
				var scope domainContext.RuleScope
				switch scopeStr {
				case "global":
					scope = domainContext.RuleScopeGlobal
				case "workspace":
					scope = domainContext.RuleScopeWorkspace
				case "session":
					scope = domainContext.RuleScopeSession
				default:
					return fmt.Errorf("invalid scope: %s", scopeStr)
				}
				rules, err = repo.ListByScope(ctx, scope)
			} else {
				rules, err = repo.List(ctx)
			}

			if err != nil {
				return fmt.Errorf("failed to list rules: %w", err)
			}

			if len(rules) == 0 {
				formatter.Header("Rules")
				if scopeStr != "" {
					formatter.Info("Scope: %s", scopeStr)
				}
				if activeOnly {
					formatter.Info("Active rules only")
				}
				formatter.Println("")
				formatter.Info("No rules found")
				return nil
			}

			// Display rules in table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSCOPE\tACTIVE\tID")
			fmt.Fprintln(w, "----\t-----\t------\t--")
			for _, rule := range rules {
				active := "No"
				if rule.IsActive() {
					active = "Yes"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					rule.Name(),
					rule.Scope(),
					active,
					shortenID(rule.ID()),
				)
			}
			_ = w.Flush()

			return nil
		},
	}

	listCmd.Flags().String("scope", "", "filter by scope: global, workspace, session")
	listCmd.Flags().Bool("active", false, "show only active rules")

	// Show subcommand
	showCmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show rule details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.RulesRepository()

			ctx := context.Background()
			rule, err := repo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("failed to find rule: %w", err)
			}

			formatter.Header("Rule: " + name)
			formatter.Info("ID: %s", rule.ID())
			formatter.Info("Scope: %s", rule.Scope())
			active := "No"
			if rule.IsActive() {
				active = "Yes"
			}
			formatter.Info("Active: %s", active)
			formatter.Println("")
			formatter.Info("Content:")
			formatter.Println(rule.Content())

			return nil
		},
	}

	// Activate subcommand
	activateCmd := &cobra.Command{
		Use:   "activate <name>",
		Short: "Activate a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.RulesRepository()

			// Find the rule by name
			ctx := context.Background()
			rule, err := repo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("failed to find rule: %w", err)
			}

			// Activate the rule
			rule.Activate()
			if err := repo.Update(ctx, rule); err != nil {
				return fmt.Errorf("failed to activate rule: %w", err)
			}

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

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.RulesRepository()

			// Find the rule by name
			ctx := context.Background()
			rule, err := repo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("failed to find rule: %w", err)
			}

			// Deactivate the rule
			rule.Deactivate()
			if err := repo.Update(ctx, rule); err != nil {
				return fmt.Errorf("failed to deactivate rule: %w", err)
			}

			formatter.Success("Deactivated rule: %s", name)

			return nil
		},
	}

	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.RulesRepository()

			// Find the rule by name
			ctx := context.Background()
			rule, err := repo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("failed to find rule: %w", err)
			}

			// Delete the rule
			if err := repo.Delete(ctx, rule.ID()); err != nil {
				return fmt.Errorf("failed to remove rule: %w", err)
			}

			formatter.Success("Removed rule: %s", name)

			return nil
		},
	}

	cmd.AddCommand(addCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(showCmd)
	cmd.AddCommand(activateCmd)
	cmd.AddCommand(deactivateCmd)
	cmd.AddCommand(removeCmd)

	return cmd
}
