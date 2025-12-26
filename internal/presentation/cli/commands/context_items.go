// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewContextItemsCmd creates the items subcommand for context.
func NewContextItemsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "items",
		Short: "Manage context items (files, snippets, URLs)",
		Long: `Manage context items that can be loaded into skill execution sessions.

Context items can be:
  • Files - Reference documentation or code files
  • Snippets - Code or text snippets
  • URLs - Links to relevant resources

Items can be tagged for organization and have token estimates calculated automatically.`,
		Example: `  # Add a file reference
  sr context items add --file ./docs/api.md

  # Add a snippet
  sr context items add --snippet "const API_KEY = ..." --name "api-config"

  # Add a URL
  sr context items add --url https://example.com/docs --name "docs"

  # List all items
  sr context items list

  # Remove an item
  sr context items remove api-config`,
	}

	// Add subcommand
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a context item",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			file, _ := cmd.Flags().GetString("file")
			snippet, _ := cmd.Flags().GetString("snippet")
			url, _ := cmd.Flags().GetString("url")
			name, _ := cmd.Flags().GetString("name")
			tags, _ := cmd.Flags().GetStringSlice("tags")

			// Validate: exactly one source type
			sources := 0
			itemType := ""
			content := ""

			if file != "" {
				sources++
				itemType = "file"
				content = file
			}
			if snippet != "" {
				sources++
				itemType = "snippet"
				content = snippet
			}
			if url != "" {
				sources++
				itemType = "url"
				content = url
			}

			if sources == 0 {
				return fmt.Errorf("must specify one of: --file, --snippet, or --url")
			}
			if sources > 1 {
				return fmt.Errorf("can only specify one source type")
			}

			// TODO: Implement add item logic
			formatter.Success("Added %s context item: %s", itemType, name)
			if len(tags) > 0 {
				formatter.Info("Tags: %v", tags)
			}

			_ = content // Silence unused warning
			return nil
		},
	}

	addCmd.Flags().String("file", "", "path to file")
	addCmd.Flags().String("snippet", "", "code or text snippet")
	addCmd.Flags().String("url", "", "URL to reference")
	addCmd.Flags().String("name", "", "name for the context item")
	addCmd.Flags().StringSlice("tags", nil, "tags for the item")
	addCmd.MarkFlagRequired("name")

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all context items",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			// TODO: Implement list logic
			formatter.Header("Context Items")
			formatter.Info("No items found")

			return nil
		},
	}

	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a context item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			// TODO: Implement remove logic
			formatter.Success("Removed context item: %s", name)

			return nil
		},
	}

	// Load subcommand
	loadCmd := &cobra.Command{
		Use:   "load <name>",
		Short: "Load and display a context item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			// TODO: Implement load logic
			formatter.Header("Context Item: " + name)
			formatter.Info("Content would be displayed here")

			return nil
		},
	}

	cmd.AddCommand(addCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(removeCmd)
	cmd.AddCommand(loadCmd)

	return cmd
}
