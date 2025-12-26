// Package commands implements CLI commands for session management.
package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

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

			// TODO: Get session manager from app context
			// For now, return error indicating not implemented
			_ = domainSession.StartOptions{
				WorkspaceID:  workspace,
				Backend:      backend,
				Model:        model,
				Profile:      profile,
				Background:   bg,
				Task:         task,
				ContextFiles: files,
			}
			return fmt.Errorf("session manager not initialized (TODO)")
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
			// Build filter
			filter := domainSession.Filter{}
			if !all {
				filter.Status = []domainSession.Status{
					domainSession.StatusActive,
					domainSession.StatusIdle,
					domainSession.StatusDetached,
				}
			}
			_ = filter

			// TODO: Get session manager from app context
			return fmt.Errorf("session manager not initialized (TODO)")
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show all sessions (including completed)")

	return cmd
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
			_ = args[0] // sessionID - will be used when session manager is implemented

			// TODO: Get session manager from app context
			return fmt.Errorf("session manager not initialized (TODO)")
		},
	}

	return cmd
}

// newSessionDetachCmd creates the 'session detach' command.
func newSessionDetachCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detach",
		Short: "Detach from current session",
		Long: `Detach from the current AI coding assistant session.

The session will continue running in the background.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Detect current session and detach
			return fmt.Errorf("session manager not initialized (TODO)")
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
			_ = args[0] // sessionID - will be used when session manager is implemented

			// Determine inject type and validate
			if prompt != "" {
				_ = domainSession.InjectContent{Type: "prompt", Content: prompt}
			} else if file != "" {
				_ = domainSession.InjectContent{Type: "file", Files: []string{file}}
			} else if item != "" {
				_ = domainSession.InjectContent{Type: "item", Content: item}
			} else {
				return fmt.Errorf("must specify one of: --prompt, --file, or --item")
			}

			// TODO: Get session manager from app context
			return fmt.Errorf("session manager not initialized (TODO)")
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
			_ = args[0] // sessionID - will be used when session manager is implemented
			_ = lines   // lines parameter - will be used when session manager is implemented

			// TODO: Get session manager from app context
			return fmt.Errorf("session manager not initialized (TODO)")
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
			_ = args[0] // sessionID - will be used when session manager is implemented
			_ = force   // force flag - will be used when session manager is implemented

			// TODO: Get session manager from app context
			return fmt.Errorf("session manager not initialized (TODO)")
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "forcefully kill the session")

	return cmd
}

// Helper functions for formatting session output (to be implemented)

func formatSessionList(sessions []*domainSession.Session) string {
	var b strings.Builder

	b.WriteString("ID\tBackend\tModel\tStatus\tStarted\n")
	b.WriteString("--\t-------\t-----\t------\t-------\n")

	for _, s := range sessions {
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
			s.ID,
			s.Backend,
			s.Model,
			s.Status,
			s.StartedAt.Format(time.RFC3339),
		))
	}

	return b.String()
}

func formatSession(s *domainSession.Session) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Session ID:     %s\n", s.ID))
	b.WriteString(fmt.Sprintf("Backend:        %s\n", s.Backend))
	b.WriteString(fmt.Sprintf("Model:          %s\n", s.Model))
	b.WriteString(fmt.Sprintf("Status:         %s\n", s.Status))
	b.WriteString(fmt.Sprintf("Workspace:      %s\n", s.WorkspaceID))
	b.WriteString(fmt.Sprintf("Started:        %s\n", s.StartedAt.Format(time.RFC3339)))
	if s.EndedAt != nil {
		b.WriteString(fmt.Sprintf("Ended:          %s\n", s.EndedAt.Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("Duration:       %s\n", s.Duration()))
	}
	if s.ProcessID > 0 {
		b.WriteString(fmt.Sprintf("Process ID:     %d\n", s.ProcessID))
	}
	if s.TmuxSession != "" {
		b.WriteString(fmt.Sprintf("Tmux Session:   %s\n", s.TmuxSession))
	}

	return b.String()
}
