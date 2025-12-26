// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
)

// NewContextCheckpointCmd creates the checkpoint subcommand for context.
func NewContextCheckpointCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkpoint",
		Short: "Create or manage checkpoints",
		Long: `Create a checkpoint to save the current session state, or manage existing checkpoints.

Checkpoints capture:
  - Summary of what was accomplished
  - Files that were modified
  - Key decisions that were made
  - Machine/environment context

Use checkpoints to pause work and resume later with full context.`,
		Example: `  # Create a checkpoint with summary
  sr context checkpoint create "Completed user authentication"

  # Create with details
  sr context checkpoint create "Auth module" --details "Implemented JWT tokens"

  # List all checkpoints
  sr context checkpoint list

  # Resume from latest checkpoint
  sr context checkpoint resume

  # Restore a specific checkpoint
  sr context checkpoint restore <checkpoint-id>`,
	}

	// Create subcommand
	createCmd := &cobra.Command{
		Use:   "create <summary>",
		Short: "Create a new checkpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			summary := args[0]

			details, _ := cmd.Flags().GetString("details")
			files, _ := cmd.Flags().GetStringSlice("files")
			decisions, _ := cmd.Flags().GetStringToString("decisions")

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			checkpointRepo := container.CheckpointRepository()
			wsRepo := container.WorkspaceRepository()

			// Get current workspace
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			ctx := context.Background()
			workspace, err := wsRepo.GetByRepoPath(ctx, cwd)
			if err != nil {
				return fmt.Errorf("no workspace found for current directory. Use 'sr workspace init' to initialize a workspace")
			}

			// Get active session (if any) for the workspace
			sessionRepo := container.SessionRepository()
			activeSession, _ := sessionRepo.GetActiveByWorkspace(ctx, workspace.ID())
			sessionID := ""
			if activeSession != nil {
				sessionID = activeSession.ID
			} else {
				// Use a placeholder session ID if no active session
				sessionID = "manual-" + uuid.New().String()[:8]
			}

			// Create the checkpoint
			id := uuid.New().String()
			checkpoint, err := domainContext.NewCheckpoint(id, workspace.ID(), sessionID, summary)
			if err != nil {
				return fmt.Errorf("failed to create checkpoint: %w", err)
			}

			if details != "" {
				checkpoint.SetDetails(details)
			}

			for _, file := range files {
				checkpoint.AddFile(file)
			}

			for key, value := range decisions {
				checkpoint.AddDecision(key, value)
			}

			checkpoint.SetMachineID(container.MachineID())

			// Save to repository
			if err := checkpointRepo.Create(ctx, checkpoint); err != nil {
				return fmt.Errorf("failed to save checkpoint: %w", err)
			}

			formatter.Success("Checkpoint created: %s", summary)
			formatter.Info("ID: %s", id)
			formatter.Info("Workspace: %s", workspace.Name())
			if details != "" {
				formatter.Info("Details: %s", details)
			}
			if len(files) > 0 {
				formatter.Info("Files: %v", files)
			}

			return nil
		},
	}

	createCmd.Flags().String("details", "", "detailed description of the checkpoint")
	createCmd.Flags().StringSlice("files", nil, "files modified in this checkpoint")
	createCmd.Flags().StringToString("decisions", nil, "key decisions made (key=value pairs)")

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all checkpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			limit, _ := cmd.Flags().GetInt("limit")

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			checkpointRepo := container.CheckpointRepository()
			wsRepo := container.WorkspaceRepository()

			ctx := context.Background()

			// Try to get current workspace
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			workspace, err := wsRepo.GetByRepoPath(ctx, cwd)
			var checkpoints []*domainContext.Checkpoint

			if err != nil {
				// No workspace, list all checkpoints
				checkpoints, err = checkpointRepo.List(ctx, nil)
				if err != nil {
					return fmt.Errorf("failed to list checkpoints: %w", err)
				}
			} else {
				// List checkpoints for this workspace
				checkpoints, err = checkpointRepo.GetByWorkspace(ctx, workspace.ID())
				if err != nil {
					return fmt.Errorf("failed to list checkpoints: %w", err)
				}
			}

			// Apply limit
			if limit > 0 && len(checkpoints) > limit {
				checkpoints = checkpoints[:limit]
			}

			if len(checkpoints) == 0 {
				formatter.Header("Checkpoints")
				formatter.Info("No checkpoints found")
				return nil
			}

			// Display checkpoints in table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSUMMARY\tCREATED\tFILES")
			fmt.Fprintln(w, "--\t-------\t-------\t-----")
			for _, cp := range checkpoints {
				summary := cp.Summary()
				if len(summary) > 40 {
					summary = summary[:37] + "..."
				}
				created := cp.CreatedAt().Format(time.RFC3339)
				files := len(cp.FilesModified())
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
					shortenID(cp.ID()),
					summary,
					created,
					files,
				)
			}
			_ = w.Flush()

			return nil
		},
	}

	listCmd.Flags().Int("limit", 10, "maximum number of checkpoints to show")

	// Resume subcommand (resume from latest)
	resumeCmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume from the latest checkpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			checkpointRepo := container.CheckpointRepository()
			wsRepo := container.WorkspaceRepository()

			// Get current workspace
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			ctx := context.Background()
			workspace, err := wsRepo.GetByRepoPath(ctx, cwd)
			if err != nil {
				return fmt.Errorf("no workspace found for current directory. Use 'sr workspace init' to initialize a workspace")
			}

			// Get checkpoints for this workspace
			checkpoints, err := checkpointRepo.GetByWorkspace(ctx, workspace.ID())
			if err != nil {
				return fmt.Errorf("failed to get checkpoints: %w", err)
			}

			if len(checkpoints) == 0 {
				return fmt.Errorf("no checkpoints found for workspace: %s", workspace.Name())
			}

			// Get the latest checkpoint (first in the list, ordered by created_at DESC)
			latest := checkpoints[0]

			// Display checkpoint information for resuming
			formatter.Header("Resuming from Checkpoint")
			formatter.Info("ID: %s", latest.ID())
			formatter.Info("Summary: %s", latest.Summary())
			if latest.Details() != "" {
				formatter.Info("Details: %s", latest.Details())
			}
			formatter.Info("Created: %s", latest.CreatedAt().Format(time.RFC3339))

			if len(latest.FilesModified()) > 0 {
				formatter.Println("")
				formatter.Info("Files Modified:")
				for _, file := range latest.FilesModified() {
					formatter.Println("  - " + file)
				}
			}

			if len(latest.Decisions()) > 0 {
				formatter.Println("")
				formatter.Info("Decisions Made:")
				for key, value := range latest.Decisions() {
					formatter.Println(fmt.Sprintf("  - %s: %s", key, value))
				}
			}

			formatter.Println("")
			formatter.Success("Checkpoint context loaded. You can now continue your work.")

			return nil
		},
	}

	// Restore subcommand (restore specific checkpoint)
	restoreCmd := &cobra.Command{
		Use:   "restore <checkpoint-id>",
		Short: "Restore a specific checkpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			checkpointID := args[0]

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			checkpointRepo := container.CheckpointRepository()

			ctx := context.Background()
			checkpoint, err := checkpointRepo.Get(ctx, checkpointID)
			if err != nil {
				return fmt.Errorf("failed to get checkpoint: %w", err)
			}

			// Display checkpoint information
			formatter.Header("Restoring Checkpoint")
			formatter.Info("ID: %s", checkpoint.ID())
			formatter.Info("Summary: %s", checkpoint.Summary())
			if checkpoint.Details() != "" {
				formatter.Info("Details: %s", checkpoint.Details())
			}
			formatter.Info("Created: %s", checkpoint.CreatedAt().Format(time.RFC3339))

			if len(checkpoint.FilesModified()) > 0 {
				formatter.Println("")
				formatter.Info("Files Modified:")
				for _, file := range checkpoint.FilesModified() {
					formatter.Println("  - " + file)
				}
			}

			if len(checkpoint.Decisions()) > 0 {
				formatter.Println("")
				formatter.Info("Decisions Made:")
				for key, value := range checkpoint.Decisions() {
					formatter.Println(fmt.Sprintf("  - %s: %s", key, value))
				}
			}

			formatter.Println("")
			formatter.Success("Checkpoint context loaded. You can now continue your work.")

			return nil
		},
	}

	// Delete subcommand
	deleteCmd := &cobra.Command{
		Use:   "delete <checkpoint-id>",
		Short: "Delete a checkpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			checkpointID := args[0]

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			checkpointRepo := container.CheckpointRepository()

			ctx := context.Background()
			if err := checkpointRepo.Delete(ctx, checkpointID); err != nil {
				return fmt.Errorf("failed to delete checkpoint: %w", err)
			}

			formatter.Success("Checkpoint deleted: %s", checkpointID)

			return nil
		},
	}

	cmd.AddCommand(createCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(resumeCmd)
	cmd.AddCommand(restoreCmd)
	cmd.AddCommand(deleteCmd)

	return cmd
}
