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

// NewContextItemsCmd creates the items subcommand for context.
func NewContextItemsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "items",
		Short: "Manage context items (files, snippets, URLs)",
		Long: `Manage context items that can be loaded into skill execution sessions.

Context items can be:
  - Files - Reference documentation or code files
  - Snippets - Code or text snippets
  - URLs - Links to relevant resources

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
			var itemType domainContext.ItemType
			content := ""

			if file != "" {
				sources++
				itemType = domainContext.ItemTypeFile
				content = file
			}
			if snippet != "" {
				sources++
				itemType = domainContext.ItemTypeSnippet
				content = snippet
			}
			if url != "" {
				sources++
				itemType = domainContext.ItemTypeURL
				content = url
			}

			if sources == 0 {
				return fmt.Errorf("must specify one of: --file, --snippet, or --url")
			}
			if sources > 1 {
				return fmt.Errorf("can only specify one source type")
			}

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.ContextItemRepository()

			// Create the context item
			id := uuid.New().String()
			item, err := domainContext.NewContextItem(id, name, itemType)
			if err != nil {
				return fmt.Errorf("failed to create context item: %w", err)
			}

			item.SetContent(content)
			for _, tag := range tags {
				item.AddTag(tag)
			}

			// Estimate tokens (simple heuristic: ~4 chars per token)
			tokenEstimate := len(content) / 4
			item.SetTokenEstimate(tokenEstimate)

			// Save to repository
			ctx := context.Background()
			if err := repo.Save(ctx, item); err != nil {
				return fmt.Errorf("failed to save context item: %w", err)
			}

			formatter.Success("Added %s context item: %s", itemType, name)
			formatter.Info("ID: %s", id)
			if len(tags) > 0 {
				formatter.Info("Tags: %v", tags)
			}
			formatter.Info("Estimated tokens: %d", tokenEstimate)

			return nil
		},
	}

	addCmd.Flags().String("file", "", "path to file")
	addCmd.Flags().String("snippet", "", "code or text snippet")
	addCmd.Flags().String("url", "", "URL to reference")
	addCmd.Flags().String("name", "", "name for the context item")
	addCmd.Flags().StringSlice("tags", nil, "tags for the item")
	_ = addCmd.MarkFlagRequired("name")

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all context items",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			tag, _ := cmd.Flags().GetString("tag")

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.ContextItemRepository()

			// List items
			ctx := context.Background()
			var items []*domainContext.ContextItem
			var err error

			if tag != "" {
				items, err = repo.ListByTag(ctx, tag)
			} else {
				items, err = repo.List(ctx)
			}

			if err != nil {
				return fmt.Errorf("failed to list context items: %w", err)
			}

			if len(items) == 0 {
				formatter.Header("Context Items")
				formatter.Info("No items found")
				return nil
			}

			// Display items in table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE\tTOKENS\tTAGS\tID")
			fmt.Fprintln(w, "----\t----\t------\t----\t--")
			for _, item := range items {
				tags := item.Tags()
				tagsStr := "-"
				if len(tags) > 0 {
					tagsStr = fmt.Sprintf("%v", tags)
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
					item.Name(),
					item.Type(),
					item.TokenEstimate(),
					tagsStr,
					shortenID(item.ID()),
				)
			}
			_ = w.Flush()

			return nil
		},
	}

	listCmd.Flags().String("tag", "", "filter by tag")

	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a context item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			name := args[0]

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.ContextItemRepository()

			// Find the item by name
			ctx := context.Background()
			item, err := repo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("failed to find context item: %w", err)
			}

			// Delete the item
			if err := repo.Delete(ctx, item.ID()); err != nil {
				return fmt.Errorf("failed to remove context item: %w", err)
			}

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

			// Get repository from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			repo := container.ContextItemRepository()

			// Find the item by name
			ctx := context.Background()
			item, err := repo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("failed to find context item: %w", err)
			}

			// Display item details
			formatter.Header("Context Item: " + name)
			formatter.Info("Type: %s", item.Type())
			formatter.Info("Token Estimate: %d", item.TokenEstimate())
			if len(item.Tags()) > 0 {
				formatter.Info("Tags: %v", item.Tags())
			}
			formatter.Println("")
			formatter.Info("Content:")
			formatter.Println(item.Content())

			return nil
		},
	}

	cmd.AddCommand(addCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(removeCmd)
	cmd.AddCommand(loadCmd)

	return cmd
}
