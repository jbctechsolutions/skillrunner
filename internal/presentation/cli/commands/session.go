// Package commands implements CLI commands for session management.
package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	domainSession "github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// NewSessionCmd creates the session command group.
func NewSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage AI coding assistant sessions",
		Long: `Manage AI coding assistant sessions.

Sessions represent active or historical instances of AI coding assistants
(Aider, Claude Code, OpenCode) running in specific workspaces.`,
	}

	// Add subcommands
	cmd.AddCommand(newSessionStartCmd())
	cmd.AddCommand(newSessionListCmd())
	cmd.AddCommand(newSessionAttachCmd())
	cmd.AddCommand(newSessionDetachCmd())
	cmd.AddCommand(newSessionInjectCmd())
	cmd.AddCommand(newSessionPeekCmd())
	cmd.AddCommand(newSessionKillCmd())

	return cmd
}

// newSessionStartCmd creates the 'session start' command.
func newSessionStartCmd() *cobra.Command {
	var (
		workspace string
		backend   string
		model     string
		profile   string
		bg        bool
		task      string
		files     []string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new AI coding session",
		Long: `Start a new AI coding assistant session.

Examples:
  # Start an Aider session in current directory
  sr session start --backend aider --model gpt-4o

  # Start Claude Code in background
  sr session start --backend claude --bg

  # Start with initial task
  sr session start --backend aider --task "Add unit tests"

  # Start with context files
  sr session start --backend aider --files main.go,utils.go`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get workspace (default to current directory)
			if workspace == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
				workspace = cwd
			}

			// Get session manager from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			manager := container.SessionManager()

			// Build start options
			opts := domainSession.StartOptions{
				WorkspaceID:  workspace,
				Backend:      backend,
				Model:        model,
				Profile:      profile,
				Background:   bg,
				Task:         task,
				ContextFiles: files,
			}

			// Start the session
			ctx := context.Background()
			sess, err := manager.Start(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to start session: %w", err)
			}

			// Display session info
			formatter := GetFormatter()
			formatter.Success("Session started: %s", sess.ID)
			formatter.Info("Backend: %s", sess.Backend)
			formatter.Info("Status: %s", sess.Status)
			if bg {
				formatter.Info("Running in background. Use 'sr session attach %s' to connect.", sess.ID)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace directory (default: current directory)")
	cmd.Flags().StringVar(&backend, "backend", "aider", "backend to use (aider, claude, opencode)")
	cmd.Flags().StringVar(&model, "model", "", "LLM model to use")
	cmd.Flags().StringVar(&profile, "profile", "", "profile name (if supported)")
	cmd.Flags().BoolVar(&bg, "bg", false, "run in background (detached)")
	cmd.Flags().StringVar(&task, "task", "", "initial task/prompt")
	cmd.Flags().StringSliceVar(&files, "files", nil, "files to include in context")

	return cmd
}

// newSessionListCmd creates the 'session list' command.
func newSessionListCmd() *cobra.Command {
	var (
		all bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List AI coding sessions",
		Long: `List AI coding assistant sessions.

By default, shows only active sessions. Use --all to see all sessions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get session manager from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			manager := container.SessionManager()

			// Build filter
			filter := domainSession.Filter{}
			if !all {
				filter.Status = []domainSession.Status{
					domainSession.StatusActive,
					domainSession.StatusIdle,
					domainSession.StatusDetached,
				}
			}

			// List sessions
			ctx := context.Background()
			sessions, err := manager.List(ctx, filter)
			if err != nil {
				return fmt.Errorf("failed to list sessions: %w", err)
			}

			if len(sessions) == 0 {
				formatter := GetFormatter()
				formatter.Info("No sessions found")
				return nil
			}

			// Display sessions in table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tBACKEND\tSTATUS\tWORKSPACE\tDURATION")
			fmt.Fprintln(w, "--\t-------\t------\t---------\t--------")
			for _, sess := range sessions {
				duration := sess.Duration().Truncate(1e9) // Truncate to seconds
				// Shorten workspace path for display
				workspace := sess.WorkspaceID
				if len(workspace) > 30 {
					workspace = "..." + workspace[len(workspace)-27:]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					shortenID(sess.ID),
					sess.Backend,
					sess.Status,
					workspace,
					duration.String(),
				)
			}
			_ = w.Flush()

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show all sessions (including completed)")

	return cmd
}

// shortenID returns a shortened version of a session ID for display.
func shortenID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// newSessionAttachCmd creates the 'session attach' command.
func newSessionAttachCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach SESSION_ID",
		Short: "Attach to an existing session",
		Long: `Attach to an existing AI coding assistant session.

This will connect your terminal to the session's interactive interface.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			// Get session manager from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			manager := container.SessionManager()

			// Attach to session
			ctx := context.Background()
			if err := manager.Attach(ctx, sessionID); err != nil {
				return fmt.Errorf("failed to attach to session: %w", err)
			}

			// Attach should block until detached, so if we get here, the session ended
			formatter := GetFormatter()
			formatter.Info("Detached from session %s", shortenID(sessionID))

			return nil
		},
	}

	return cmd
}

// newSessionDetachCmd creates the 'session detach' command.
func newSessionDetachCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detach [SESSION_ID]",
		Short: "Detach from current or specified session",
		Long: `Detach from the current AI coding assistant session.

The session will continue running in the background.
If SESSION_ID is not provided, attempts to find an active session in the current workspace.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get session manager from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			manager := container.SessionManager()

			var sessionID string
			if len(args) > 0 {
				sessionID = args[0]
			} else {
				// Try to find active session in current workspace
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				ctx := context.Background()
				filter := domainSession.Filter{
					WorkspaceID: cwd,
					Status: []domainSession.Status{
						domainSession.StatusActive,
					},
				}
				sessions, err := manager.List(ctx, filter)
				if err != nil {
					return fmt.Errorf("failed to find active session: %w", err)
				}
				if len(sessions) == 0 {
					return fmt.Errorf("no active session found in current workspace")
				}
				sessionID = sessions[0].ID
			}

			// Detach from session
			ctx := context.Background()
			if err := manager.Detach(ctx, sessionID); err != nil {
				return fmt.Errorf("failed to detach from session: %w", err)
			}

			formatter := GetFormatter()
			formatter.Success("Detached from session %s", shortenID(sessionID))
			formatter.Info("Session is now running in background. Use 'sr session attach %s' to reconnect.", shortenID(sessionID))

			return nil
		},
	}

	return cmd
}

// newSessionInjectCmd creates the 'session inject' command.
func newSessionInjectCmd() *cobra.Command {
	var (
		item   string
		file   string
		prompt string
	)

	cmd := &cobra.Command{
		Use:   "inject SESSION_ID",
		Short: "Inject content into a session",
		Long: `Inject content into a running session.

You can inject:
  - A prompt (--prompt "text")
  - A file (--file path/to/file)
  - An item reference (--item "description")`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			// Get session manager from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			manager := container.SessionManager()

			// Determine inject type and validate
			var content domainSession.InjectContent
			if prompt != "" {
				content = domainSession.InjectContent{Type: "prompt", Content: prompt}
			} else if file != "" {
				content = domainSession.InjectContent{Type: "file", Files: []string{file}}
			} else if item != "" {
				content = domainSession.InjectContent{Type: "item", Content: item}
			} else {
				return fmt.Errorf("must specify one of: --prompt, --file, or --item")
			}

			// Inject content
			ctx := context.Background()
			if err := manager.Inject(ctx, sessionID, content); err != nil {
				return fmt.Errorf("failed to inject content: %w", err)
			}

			formatter := GetFormatter()
			formatter.Success("Injected %s into session %s", content.Type, shortenID(sessionID))

			return nil
		},
	}

	cmd.Flags().StringVar(&item, "item", "", "inject an item reference")
	cmd.Flags().StringVar(&file, "file", "", "inject a file")
	cmd.Flags().StringVar(&prompt, "prompt", "", "inject a prompt")

	return cmd
}

// newSessionPeekCmd creates the 'session peek' command.
func newSessionPeekCmd() *cobra.Command {
	var (
		lines int
	)

	cmd := &cobra.Command{
		Use:   "peek SESSION_ID",
		Short: "View recent output from a session",
		Long: `View recent output from a session without attaching.

This is useful for checking on background sessions.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			// Get session manager from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			manager := container.SessionManager()

			// Get session output
			ctx := context.Background()
			output, err := manager.Peek(ctx, sessionID, lines)
			if err != nil {
				return fmt.Errorf("failed to peek session: %w", err)
			}

			if len(output) == 0 {
				formatter := GetFormatter()
				formatter.Info("No output available for session %s", shortenID(sessionID))
				return nil
			}

			// Display output
			fmt.Printf("=== Session %s (last %d lines) ===\n\n", shortenID(sessionID), len(output))
			for _, line := range output {
				fmt.Println(line)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&lines, "lines", 50, "number of lines to show")

	return cmd
}

// newSessionKillCmd creates the 'session kill' command.
func newSessionKillCmd() *cobra.Command {
	var (
		force bool
	)

	cmd := &cobra.Command{
		Use:   "kill SESSION_ID",
		Short: "Terminate a session",
		Long: `Terminate an AI coding assistant session.

Use --force to forcefully kill the session (SIGKILL instead of graceful shutdown).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			// Get session manager from container
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			manager := container.SessionManager()

			// Kill the session
			ctx := context.Background()
			if err := manager.Kill(ctx, sessionID, force); err != nil {
				return fmt.Errorf("failed to kill session: %w", err)
			}

			formatter := GetFormatter()
			if force {
				formatter.Success("Session %s forcefully terminated", shortenID(sessionID))
			} else {
				formatter.Success("Session %s terminated", shortenID(sessionID))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "forcefully kill the session")

	return cmd
}
