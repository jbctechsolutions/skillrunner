// Package commands implements CLI commands for workspace management.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/domain/workspace"
)

// NewWorkspaceCmd creates the workspace command group.
func NewWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage development workspaces",
		Long: `Manage development workspaces.

Workspaces are isolated development environments that can be backed by
regular directories or Git worktrees.`,
		Aliases: []string{"ws"},
	}

	// Add subcommands
	cmd.AddCommand(newWorkspaceCreateCmd())
	cmd.AddCommand(newWorkspaceListCmd())
	cmd.AddCommand(newWorkspaceSwitchCmd())
	cmd.AddCommand(newWorkspaceStatusCmd())
	cmd.AddCommand(newWorkspaceSpawnCmd())
	cmd.AddCommand(newWorkspaceDeleteCmd())

	return cmd
}

// newWorkspaceCreateCmd creates the 'workspace create' command.
func newWorkspaceCreateCmd() *cobra.Command {
	var (
		worktree    bool
		branch      string
		path        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new workspace",
		Long: `Create a new development workspace.

By default, creates a regular directory workspace. Use --worktree to create
a Git worktree workspace instead.

Examples:
  # Create a regular directory workspace
  sr workspace create my-feature

  # Create a Git worktree workspace on new branch
  sr workspace create my-feature --worktree --branch feature/new-feature

  # Create with custom path
  sr workspace create my-feature --path /path/to/workspace`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Build create options (will be used when manager is implemented)
			_ = workspace.CreateOptions{
				Name:        name,
				Path:        path,
				GitWorktree: worktree,
				GitBranch:   branch,
				Description: description,
			}

			// TODO: Get workspace manager from app context
			return fmt.Errorf("workspace manager not initialized (TODO)")
		},
	}

	cmd.Flags().BoolVar(&worktree, "worktree", false, "create as Git worktree")
	cmd.Flags().StringVar(&branch, "branch", "", "branch name for worktree")
	cmd.Flags().StringVar(&path, "path", "", "custom path for workspace")
	cmd.Flags().StringVar(&description, "description", "", "workspace description")

	return cmd
}

// newWorkspaceListCmd creates the 'workspace list' command.
func newWorkspaceListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Long: `List all development workspaces.

Shows workspace name, type, status, and path.`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Get workspace manager from app context
			// For now, return error
			return fmt.Errorf("workspace manager not initialized (TODO)")

			// Example implementation:
			// manager := GetWorkspaceManager()
			// workspaces, err := manager.List(ctx, workspace.Filter{})
			// if err != nil {
			//     return err
			// }
			//
			// formatter := GetFormatter()
			// formatter.PrintWorkspaceList(workspaces)
			// return nil
		},
	}

	return cmd
}

// newWorkspaceSwitchCmd creates the 'workspace switch' command.
func newWorkspaceSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch NAME",
		Short: "Switch to a different workspace",
		Long: `Switch the current working directory to a different workspace.

This changes the shell's current directory to the workspace path.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args[0] // name - will be used when workspace manager is implemented

			// TODO: Get workspace manager from app context
			// For now, return error
			return fmt.Errorf("workspace manager not initialized (TODO)")

			// Example implementation:
			// manager := GetWorkspaceManager()
			// ws, err := manager.GetByName(ctx, name)
			// if err != nil {
			//     return err
			// }
			//
			// if err := manager.Switch(ctx, ws.ID); err != nil {
			//     return err
			// }
			//
			// formatter := GetFormatter()
			// formatter.Success("Switched to workspace: %s", ws.Name)
			// formatter.Info("Path: %s", ws.Path)
			// return nil
		},
	}

	return cmd
}

// newWorkspaceStatusCmd creates the 'workspace status' command.
func newWorkspaceStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current workspace status",
		Long: `Show information about the current workspace.

Displays workspace name, type, path, branch (if Git), and status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Get workspace manager from app context
			// For now, return error
			return fmt.Errorf("workspace manager not initialized (TODO)")

			// Example implementation:
			// manager := GetWorkspaceManager()
			// ws, err := manager.Status(ctx)
			// if err != nil {
			//     return err
			// }
			//
			// formatter := GetFormatter()
			// formatter.PrintWorkspace(ws)
			// return nil
		},
	}

	return cmd
}

// newWorkspaceSpawnCmd creates the 'workspace spawn' command.
func newWorkspaceSpawnCmd() *cobra.Command {
	var (
		terminal string
		command  string
		bg       bool
	)

	cmd := &cobra.Command{
		Use:   "spawn NAME",
		Short: "Spawn a terminal in a workspace",
		Long: `Spawn a new terminal window in a workspace.

The terminal type is auto-detected unless specified with --terminal.

Examples:
  # Spawn terminal in workspace
  sr workspace spawn my-feature

  # Spawn with custom command
  sr workspace spawn my-feature --command "vim ."

  # Spawn in background
  sr workspace spawn my-feature --bg`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = args[0] // name - will be used when workspace manager is implemented

			// TODO: Get workspace manager and terminal spawner from app context
			// For now, return error
			return fmt.Errorf("workspace manager not initialized (TODO)")

			// Example implementation:
			// manager := GetWorkspaceManager()
			// ws, err := manager.GetByName(ctx, name)
			// if err != nil {
			//     return err
			// }
			//
			// spawner := GetTerminalSpawner()
			// opts := terminal.SpawnOptions{
			//     WorkingDir: ws.Path,
			//     Command:    command,
			//     Background: bg,
			// }
			//
			// if err := spawner.Spawn(ctx, opts); err != nil {
			//     return err
			// }
			//
			// formatter := GetFormatter()
			// formatter.Success("Spawned terminal in workspace: %s", ws.Name)
			// return nil
		},
	}

	cmd.Flags().StringVar(&terminal, "terminal", "auto", "terminal type (auto, iterm2, terminal, tmux)")
	cmd.Flags().StringVar(&command, "command", "", "command to run in terminal")
	cmd.Flags().BoolVar(&bg, "bg", false, "run in background")

	return cmd
}

// newWorkspaceDeleteCmd creates the 'workspace delete' command.
func newWorkspaceDeleteCmd() *cobra.Command {
	var (
		removeFiles bool
		force       bool
	)

	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a workspace",
		Long: `Delete a workspace.

By default, only removes the workspace from the registry but leaves files intact.
Use --remove-files to also delete the workspace directory.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Confirm deletion if removing files
			if removeFiles && !force {
				fmt.Printf("This will permanently delete the workspace and all its files.\n")
				fmt.Printf("Type the workspace name to confirm: ")
				var confirmation string
				fmt.Scanln(&confirmation)
				if confirmation != name {
					return fmt.Errorf("confirmation failed")
				}
			}

			// TODO: Get workspace manager from app context
			// For now, return error
			return fmt.Errorf("workspace manager not initialized (TODO)")

			// Example implementation:
			// manager := GetWorkspaceManager()
			// ws, err := manager.GetByName(ctx, name)
			// if err != nil {
			//     return err
			// }
			//
			// if err := manager.Delete(ctx, ws.ID, removeFiles); err != nil {
			//     return err
			// }
			//
			// formatter := GetFormatter()
			// formatter.Success("Workspace deleted: %s", ws.Name)
			// return nil
		},
	}

	cmd.Flags().BoolVar(&removeFiles, "remove-files", false, "remove workspace files")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation")

	return cmd
}
