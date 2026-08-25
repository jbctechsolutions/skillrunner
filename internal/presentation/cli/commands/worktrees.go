package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	infraGit "github.com/jbctechsolutions/skillrunner/internal/infrastructure/git"
)

// NewWorktreesCmd creates the worktrees command for managing isolation worktrees.
func NewWorktreesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktrees",
		Short: "Manage isolation worktrees",
		Long: `List and clean up git worktrees created by --isolate.

Examples:
  sr worktrees list        # show all isolation worktrees
  sr worktrees clean       # remove stale worktrees (>7 days old)`,
	}

	cmd.AddCommand(newWorktreesListCmd())
	cmd.AddCommand(newWorktreesCleanCmd())
	return cmd
}

func newWorktreesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List isolation worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			im, err := infraGit.NewIsolationManager("")
			if err != nil {
				return fmt.Errorf("git not available: %w", err)
			}

			sessions, err := im.ListWorktrees()
			if err != nil {
				return fmt.Errorf("failed to list worktrees: %w", err)
			}

			if len(sessions) == 0 {
				_ = formatter.Info("No isolation worktrees found.")
				return nil
			}

			_ = formatter.Header("Isolation Worktrees")
			now := time.Now()
			for _, s := range sessions {
				age := now.Sub(s.CreatedAt).Truncate(time.Hour)
				stale := ""
				if age > 7*24*time.Hour {
					stale = " [stale]"
				}
				_ = formatter.Item(s.SkillName, fmt.Sprintf("%s  (%s old%s)", s.WorktreePath, formatAge(age), stale))
			}
			return nil
		},
	}
}

func newWorktreesCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Remove stale isolation worktrees (older than 7 days)",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := GetFormatter()
			im, err := infraGit.NewIsolationManager("")
			if err != nil {
				return fmt.Errorf("git not available: %w", err)
			}

			ctx := context.Background()

			// Determine repo root from CWD for pruning
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				return fmt.Errorf("failed to determine working directory: %w", cwdErr)
			}
			repoRoot := cwd
			if wm, wmErr := infraGit.NewWorktreeManager(); wmErr == nil {
				if root, rootErr := wm.GetRepositoryRoot(ctx, cwd); rootErr == nil {
					repoRoot = root
				}
			}

			n, err := im.CleanStale(ctx, repoRoot)
			if err != nil {
				return fmt.Errorf("failed to clean worktrees: %w", err)
			}

			if n == 0 {
				_ = formatter.Info("No stale worktrees found.")
			} else {
				_ = formatter.Success("Removed %d stale worktree(s).", n)
			}
			return nil
		},
	}
}

// formatAge returns a human-friendly age string.
func formatAge(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
