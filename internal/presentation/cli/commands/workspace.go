// Package commands implements CLI commands for workspace management.
package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// sanitizePathForDeletion validates that a path is safe to delete.
// It ensures the path:
// - Is absolute
// - Does not traverse outside allowed directories
// - Is not a system directory or home directory itself
// Returns an error if the path is unsafe.
func sanitizePathForDeletion(path string) error {
	// Must be absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(path)

	// Check for path traversal after cleaning
	if cleanPath != path && strings.Contains(path, "..") {
		return fmt.Errorf("path contains traversal components: %s", path)
	}

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Don't allow deleting the home directory itself
	if cleanPath == homeDir {
		return fmt.Errorf("cannot delete home directory")
	}

	// Don't allow deleting critical system directories
	criticalPaths := []string{
		"/",
		"/bin",
		"/sbin",
		"/usr",
		"/etc",
		"/var",
		"/tmp",
		"/opt",
		"/lib",
		"/System",
		"/Library",
		"/Applications",
	}
	for _, critical := range criticalPaths {
		if cleanPath == critical || strings.HasPrefix(cleanPath, critical+"/") && len(cleanPath) <= len(critical)+5 {
			return fmt.Errorf("cannot delete system directory: %s", path)
		}
	}

	// Don't allow deleting .skillrunner config directory itself
	skillrunnerDir := filepath.Join(homeDir, ".skillrunner")
	if cleanPath == skillrunnerDir {
		return fmt.Errorf("cannot delete skillrunner config directory")
	}

	// Path should be within home directory or a recognizable project directory
	if !strings.HasPrefix(cleanPath, homeDir) {
		// Allow paths in common project locations
		allowed := false
		commonRoots := []string{"/tmp/", "/var/tmp/", "/projects/", "/work/"}
		for _, root := range commonRoots {
			if strings.HasPrefix(cleanPath, root) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path is outside allowed directories: %s", path)
		}
	}

	return nil
}

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

By default, creates a workspace for the current directory. Use --worktree
to create a Git worktree workspace instead.

Examples:
  # Create a workspace for current directory
  sr workspace create my-project

  # Create a workspace with custom path
  sr workspace create my-feature --path /path/to/project

  # Create a Git worktree workspace (requires git worktree support)
  sr workspace create my-feature --worktree --branch feature/new-feature`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			wsRepo := container.WorkspaceRepository()
			ctx := context.Background()

			// Determine workspace path
			wsPath := path
			if wsPath == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
				wsPath = cwd
			}

			// Convert to absolute path
			absPath, err := filepath.Abs(wsPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path: %w", err)
			}

			// Check if path exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return fmt.Errorf("path does not exist: %s", absPath)
			}

			// Check if workspace already exists for this path
			existing, _ := wsRepo.GetByRepoPath(ctx, absPath)
			if existing != nil {
				return fmt.Errorf("workspace already exists for this path: %s", existing.Name())
			}

			// Check if workspace with same name exists
			existingByName, _ := wsRepo.GetByName(ctx, name)
			if existingByName != nil {
				return fmt.Errorf("workspace with name '%s' already exists", name)
			}

			// Generate workspace ID
			wsID := uuid.New().String()[:8]

			// Create the workspace domain object
			ws, err := domainContext.NewWorkspace(wsID, name, absPath)
			if err != nil {
				return fmt.Errorf("failed to create workspace: %w", err)
			}

			// Set optional fields
			if branch != "" {
				ws.SetBranch(branch)
			}
			if worktree {
				ws.SetWorktreePath(absPath)
			}

			// Save workspace to storage
			if err := wsRepo.Create(ctx, ws); err != nil {
				return fmt.Errorf("failed to save workspace: %w", err)
			}

			// Display success
			formatter := GetFormatter()
			formatter.Success("Workspace created: %s", name)
			formatter.Info("Path: %s", absPath)
			if branch != "" {
				formatter.Info("Branch: %s", branch)
			}
			if description != "" {
				formatter.Info("Description: %s", description)
			}

			return nil
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
	var (
		all bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Long: `List all development workspaces.

Shows workspace name, type, status, and path.`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			// Get workspace repository
			wsRepo := container.WorkspaceRepository()

			// Build filter
			var filter *ports.WorkspaceFilter
			if !all {
				// Only show active workspaces by default
				filter = &ports.WorkspaceFilter{
					Status: domainContext.WorkspaceStatusActive,
				}
			}

			// List workspaces
			ctx := context.Background()
			workspaces, err := wsRepo.List(ctx, filter)
			if err != nil {
				return fmt.Errorf("failed to list workspaces: %w", err)
			}

			if len(workspaces) == 0 {
				formatter := GetFormatter()
				formatter.Info("No workspaces found")
				return nil
			}

			// Display workspaces in table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSTATUS\tBRANCH\tPATH\tLAST ACTIVE")
			fmt.Fprintln(w, "----\t------\t------\t----\t-----------")
			for _, ws := range workspaces {
				branch := ws.Branch()
				if branch == "" {
					branch = "-"
				}
				// Shorten path for display
				path := ws.RepoPath()
				if ws.WorktreePath() != "" {
					path = ws.WorktreePath()
				}
				if len(path) > 40 {
					path = "..." + path[len(path)-37:]
				}
				// Format last active time
				lastActive := formatRelativeTime(ws.LastActiveAt())
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					ws.Name(),
					ws.Status(),
					branch,
					path,
					lastActive,
				)
			}
			_ = w.Flush()

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show all workspaces (including idle/archived)")

	return cmd
}

// formatRelativeTime formats a time as a relative duration (e.g., "2h ago", "3d ago").
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// newWorkspaceSwitchCmd creates the 'workspace switch' command.
func newWorkspaceSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch NAME",
		Short: "Switch to a different workspace",
		Long: `Switch the current working directory to a different workspace.

This changes the shell's current directory to the workspace path.
Note: Due to shell limitations, this command outputs a 'cd' command
that you can execute with: eval $(sr workspace switch NAME)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			wsRepo := container.WorkspaceRepository()
			ctx := context.Background()

			// Get workspace by name
			ws, err := wsRepo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("workspace not found: %s", name)
			}

			// Determine the workspace path
			wsPath := ws.RepoPath()
			if ws.WorktreePath() != "" {
				wsPath = ws.WorktreePath()
			}

			// Update workspace status to active
			ws.Activate()
			if updateErr := wsRepo.Update(ctx, ws); updateErr != nil {
				// Not critical, just log and continue
				formatter := GetFormatter()
				formatter.Warning("Could not update workspace status: %v", updateErr)
			}

			// Output the cd command for shell evaluation
			// Users should run: eval $(sr workspace switch NAME)
			fmt.Printf("cd %q\n", wsPath)

			return nil
		},
	}

	return cmd
}

// newWorkspaceStatusCmd creates the 'workspace status' command.
func newWorkspaceStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [NAME]",
		Short: "Show workspace status",
		Long: `Show information about a workspace.

If NAME is provided, shows that workspace. Otherwise shows the workspace
for the current directory.

Displays workspace name, path, branch (if Git), status, and active sessions.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			wsRepo := container.WorkspaceRepository()
			sessionManager := container.SessionManager()
			ctx := context.Background()

			var ws *domainContext.Workspace
			var err error

			if len(args) > 0 {
				// Get workspace by name
				ws, err = wsRepo.GetByName(ctx, args[0])
				if err != nil {
					return fmt.Errorf("workspace not found: %s", args[0])
				}
			} else {
				// Find workspace for current directory
				cwd, cwdErr := os.Getwd()
				if cwdErr != nil {
					return fmt.Errorf("failed to get current directory: %w", cwdErr)
				}

				ws, err = wsRepo.GetByRepoPath(ctx, cwd)
				if err != nil {
					// Try with worktree path
					workspaces, listErr := wsRepo.List(ctx, nil)
					if listErr != nil {
						return fmt.Errorf("failed to find workspace for current directory")
					}
					for _, w := range workspaces {
						if w.WorktreePath() == cwd || w.RepoPath() == cwd {
							ws = w
							break
						}
					}
					if ws == nil {
						return fmt.Errorf("no workspace found for current directory: %s", cwd)
					}
				}
			}

			// Display workspace info
			formatter := GetFormatter()
			fmt.Println()
			formatter.Info("Workspace: %s", ws.Name())
			fmt.Println()

			// Show details in a nice format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "  ID:\t%s\n", ws.ID())
			fmt.Fprintf(w, "  Status:\t%s\n", ws.Status())
			fmt.Fprintf(w, "  Path:\t%s\n", ws.RepoPath())
			if ws.WorktreePath() != "" {
				fmt.Fprintf(w, "  Worktree:\t%s\n", ws.WorktreePath())
			}
			if ws.Branch() != "" {
				fmt.Fprintf(w, "  Branch:\t%s\n", ws.Branch())
			}
			if ws.Focus() != "" {
				fmt.Fprintf(w, "  Focus:\t%s\n", ws.Focus())
			}
			if ws.DefaultBackend() != "" {
				fmt.Fprintf(w, "  Default Backend:\t%s\n", ws.DefaultBackend())
			}
			fmt.Fprintf(w, "  Created:\t%s\n", ws.CreatedAt().Format(time.RFC3339))
			fmt.Fprintf(w, "  Last Active:\t%s\n", formatRelativeTime(ws.LastActiveAt()))
			_ = w.Flush()

			// Show active sessions
			wsPath := ws.RepoPath()
			if ws.WorktreePath() != "" {
				wsPath = ws.WorktreePath()
			}
			sessions, err := sessionManager.List(ctx, session.Filter{
				WorkspaceID: wsPath,
				Status: []session.Status{
					session.StatusActive,
					session.StatusIdle,
					session.StatusDetached,
				},
			})
			if err == nil && len(sessions) > 0 {
				fmt.Println()
				formatter.Info("Active Sessions:")
				fmt.Println()
				sessTable := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(sessTable, "  ID\tBACKEND\tSTATUS\tDURATION")
				fmt.Fprintln(sessTable, "  --\t-------\t------\t--------")
				for _, sess := range sessions {
					duration := sess.Duration().Truncate(time.Second)
					fmt.Fprintf(sessTable, "  %s\t%s\t%s\t%s\n",
						shortenID(sess.ID),
						sess.Backend,
						sess.Status,
						duration.String(),
					)
				}
				_ = sessTable.Flush()
			} else {
				fmt.Println()
				formatter.Info("No active sessions")
			}

			fmt.Println()
			return nil
		},
	}

	return cmd
}

// spawnTerminal spawns a new terminal window in the specified directory.
// It handles platform-specific terminal spawning for darwin (macOS) and linux.
// The terminalType can be "auto" (detect automatically), or a specific terminal:
// - darwin: "terminal" (Terminal.app), "iterm2" (iTerm2)
// - linux: "gnome-terminal", "konsole", "xterm", "xfce4-terminal"
// If command is non-empty, it will be executed in the spawned terminal.
// If background is true, the terminal process is detached.
func spawnTerminal(dir, terminalType, command string, background bool) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = spawnTerminalDarwin(dir, terminalType, command)
	case "linux":
		var err error
		cmd, err = spawnTerminalLinux(dir, terminalType, command)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if background {
		// Start the process and don't wait for it
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to spawn terminal: %w", err)
		}
		// Release resources so the process can run independently
		return cmd.Process.Release()
	}

	// Run and wait for completion
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to spawn terminal: %w", err)
	}
	return nil
}

// spawnTerminalDarwin handles terminal spawning on macOS.
func spawnTerminalDarwin(dir, terminalType, command string) *exec.Cmd {
	// Build the AppleScript to run in terminal
	var script string

	if terminalType == "iterm2" {
		// iTerm2 AppleScript
		if command != "" {
			script = fmt.Sprintf(`
				tell application "iTerm"
					create window with default profile
					tell current session of current window
						write text "cd %q && %s"
					end tell
				end tell
			`, dir, command)
		} else {
			script = fmt.Sprintf(`
				tell application "iTerm"
					create window with default profile
					tell current session of current window
						write text "cd %q"
					end tell
				end tell
			`, dir)
		}
		return exec.Command("osascript", "-e", script)
	}

	// Default: Terminal.app
	if command != "" {
		script = fmt.Sprintf(`
			tell application "Terminal"
				do script "cd %q && %s"
				activate
			end tell
		`, dir, command)
	} else {
		script = fmt.Sprintf(`
			tell application "Terminal"
				do script "cd %q"
				activate
			end tell
		`, dir)
	}
	return exec.Command("osascript", "-e", script)
}

// spawnTerminalLinux handles terminal spawning on Linux.
func spawnTerminalLinux(dir, terminalType, command string) (*exec.Cmd, error) {
	// If auto-detect, try common terminals in order of preference
	if terminalType == "auto" || terminalType == "" {
		terminals := []struct {
			name string
			args func(dir, cmd string) []string
		}{
			{"gnome-terminal", func(d, c string) []string {
				if c != "" {
					return []string{"--working-directory=" + d, "--", "bash", "-c", c + "; exec bash"}
				}
				return []string{"--working-directory=" + d}
			}},
			{"konsole", func(d, c string) []string {
				if c != "" {
					return []string{"--workdir", d, "-e", "bash", "-c", c + "; exec bash"}
				}
				return []string{"--workdir", d}
			}},
			{"xfce4-terminal", func(d, c string) []string {
				if c != "" {
					return []string{"--working-directory=" + d, "-x", "bash", "-c", c + "; exec bash"}
				}
				return []string{"--working-directory=" + d}
			}},
			{"xterm", func(d, c string) []string {
				if c != "" {
					return []string{"-e", "bash", "-c", fmt.Sprintf("cd %q && %s; exec bash", d, c)}
				}
				return []string{"-e", "bash", "-c", fmt.Sprintf("cd %q && exec bash", d)}
			}},
		}

		for _, term := range terminals {
			if path, err := exec.LookPath(term.name); err == nil {
				args := term.args(dir, command)
				return exec.Command(path, args...), nil
			}
		}

		return nil, fmt.Errorf("no supported terminal emulator found (tried gnome-terminal, konsole, xfce4-terminal, xterm)")
	}

	// Specific terminal requested
	switch terminalType {
	case "gnome-terminal":
		if command != "" {
			return exec.Command("gnome-terminal", "--working-directory="+dir, "--", "bash", "-c", command+"; exec bash"), nil
		}
		return exec.Command("gnome-terminal", "--working-directory="+dir), nil
	case "konsole":
		if command != "" {
			return exec.Command("konsole", "--workdir", dir, "-e", "bash", "-c", command+"; exec bash"), nil
		}
		return exec.Command("konsole", "--workdir", dir), nil
	case "xfce4-terminal":
		if command != "" {
			return exec.Command("xfce4-terminal", "--working-directory="+dir, "-x", "bash", "-c", command+"; exec bash"), nil
		}
		return exec.Command("xfce4-terminal", "--working-directory="+dir), nil
	case "xterm":
		if command != "" {
			return exec.Command("xterm", "-e", "bash", "-c", fmt.Sprintf("cd %q && %s; exec bash", dir, command)), nil
		}
		return exec.Command("xterm", "-e", "bash", "-c", fmt.Sprintf("cd %q && exec bash", dir)), nil
	default:
		return nil, fmt.Errorf("unsupported terminal type: %s", terminalType)
	}
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
			name := args[0]

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			wsRepo := container.WorkspaceRepository()
			ctx := context.Background()

			// Get workspace by name
			ws, err := wsRepo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("workspace not found: %s", name)
			}

			// Determine workspace path
			wsPath := ws.RepoPath()
			if ws.WorktreePath() != "" {
				wsPath = ws.WorktreePath()
			}

			// Verify workspace path exists
			if _, err := os.Stat(wsPath); os.IsNotExist(err) {
				return fmt.Errorf("workspace path does not exist: %s", wsPath)
			}

			// Spawn terminal in the workspace directory
			formatter := GetFormatter()
			formatter.Info("Spawning terminal in: %s", wsPath)

			if err := spawnTerminal(wsPath, terminal, command, bg); err != nil {
				return fmt.Errorf("failed to spawn terminal: %w", err)
			}

			if bg {
				formatter.Success("Terminal spawned in background")
			} else {
				formatter.Success("Terminal spawned successfully")
			}

			return nil
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

			// Get container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			wsRepo := container.WorkspaceRepository()
			ctx := context.Background()

			// Get workspace by name
			ws, err := wsRepo.GetByName(ctx, name)
			if err != nil {
				return fmt.Errorf("workspace not found: %s", name)
			}

			// Confirm deletion if removing files
			if removeFiles && !force {
				fmt.Printf("This will permanently delete the workspace and all its files.\n")
				fmt.Printf("Type the workspace name to confirm: ")
				var confirmation string
				if _, scanErr := fmt.Scanln(&confirmation); scanErr != nil {
					return fmt.Errorf("failed to read confirmation")
				}
				if confirmation != name {
					return fmt.Errorf("confirmation failed")
				}
			}

			// Determine workspace path for file removal
			wsPath := ws.RepoPath()
			if ws.WorktreePath() != "" {
				wsPath = ws.WorktreePath()
			}

			// Delete workspace from registry
			if err := wsRepo.Delete(ctx, ws.ID()); err != nil {
				return fmt.Errorf("failed to delete workspace: %w", err)
			}

			// Remove files if requested
			if removeFiles {
				// Sanitize path before deletion to prevent dangerous operations
				if err := sanitizePathForDeletion(wsPath); err != nil {
					return fmt.Errorf("cannot delete workspace files: %w", err)
				}

				if err := os.RemoveAll(wsPath); err != nil {
					formatter := GetFormatter()
					formatter.Warning("Workspace deleted from registry, but failed to remove files: %v", err)
					return nil
				}
			}

			// Display success
			formatter := GetFormatter()
			formatter.Success("Workspace deleted: %s", name)
			if removeFiles {
				formatter.Info("Files removed: %s", wsPath)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&removeFiles, "remove-files", false, "remove workspace files")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation")

	return cmd
}
