package commands

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	domainOutcome "github.com/jbctechsolutions/skillrunner/internal/domain/outcome"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// NewOutcomesCmd creates the `sr outcomes` command group.
func NewOutcomesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outcomes",
		Short: "View and manage skill execution outcome history",
		Long:  "Inspect per-skill execution history and get routing profile recommendations.",
	}

	cmd.AddCommand(newOutcomesShowCmd())
	cmd.AddCommand(newOutcomesResetCmd())

	return cmd
}

// ─────────────────────────────────────────────────────────────────
// sr outcomes show <skill>
// ─────────────────────────────────────────────────────────────────

func newOutcomesShowCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "show <skill-id-or-name>",
		Short: "Show outcome history and routing recommendations for a skill",
		Args:  cobra.ExactArgs(1),
		Example: `  sr outcomes show code-review
  sr outcomes show test-gen --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOutcomesShow(cmd.Context(), args[0], limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 200, "Maximum number of recent outcomes to load")

	return cmd
}

func runOutcomesShow(ctx context.Context, skillArg string, limit int) error {
	formatter := GetFormatter()

	repo, skillID, skillName, err := resolveOutcomeSkill(ctx, skillArg)
	if err != nil {
		return err
	}

	// Fetch stats
	stats, err := repo.GetStats(ctx, skillID)
	if err != nil {
		return fmt.Errorf("failed to get outcome stats: %w", err)
	}

	// Fetch recent outcomes for totals
	recent, err := repo.GetBySkill(ctx, skillID, limit)
	if err != nil {
		return fmt.Errorf("failed to get outcomes: %w", err)
	}

	formatter.Header(fmt.Sprintf("Outcomes — %s", skillName))
	formatter.Println("")

	if len(recent) == 0 {
		formatter.Info("No outcomes recorded yet for this skill.")
		formatter.Println("")
		formatter.Println("  Outcomes are recorded automatically after each %s run.", formatter.Dim("sr run"))
		return nil
	}

	formatter.Println("  %s  %d", formatter.Dim("Total recorded:"), len(recent))
	formatter.Println("")

	if len(stats) == 0 {
		formatter.Info("No aggregated stats available.")
		return nil
	}

	// Stats table
	formatter.SubHeader("Stats by Phase & Profile")
	formatter.Println("")

	tableData := output.TableData{
		Columns: []output.TableColumn{
			{Header: "Phase", Width: 20, Align: output.AlignLeft},
			{Header: "Profile", Width: 10, Align: output.AlignLeft},
			{Header: "Runs", Width: 7, Align: output.AlignRight},
			{Header: "Success%", Width: 10, Align: output.AlignRight},
			{Header: "Avg Retries", Width: 12, Align: output.AlignRight},
			{Header: "Avg ms", Width: 10, Align: output.AlignRight},
			{Header: "Model", Width: 18, Align: output.AlignLeft},
		},
	}

	// Sort: phase then profile order
	profileOrder := map[string]int{"cheap": 0, "balanced": 1, "premium": 2}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].PhaseID != stats[j].PhaseID {
			return stats[i].PhaseID < stats[j].PhaseID
		}
		return profileOrder[stats[i].Profile] < profileOrder[stats[j].Profile]
	})

	for _, s := range stats {
		successColor := output.ColorGreen
		if s.SuccessRate < 0.80 {
			successColor = output.ColorRed
		} else if s.SuccessRate < 0.90 {
			successColor = output.ColorYellow
		}

		tableData.Rows = append(tableData.Rows, []string{
			s.PhaseName,
			s.Profile,
			fmt.Sprintf("%d", s.TotalRuns),
			formatter.Colorize(fmt.Sprintf("%.1f%%", s.SuccessRate*100), successColor),
			fmt.Sprintf("%.1f", s.AvgRetries),
			fmt.Sprintf("%d", s.AvgDurationMS),
			s.Model,
		})
	}

	if err := formatter.Table(tableData); err != nil {
		return err
	}
	formatter.Println("")

	// Recommendations
	recs := domainOutcome.Analyze(stats)
	if len(recs) > 0 {
		formatter.SubHeader("Routing Recommendations")
		formatter.Println("")
		for _, r := range recs {
			conf := fmt.Sprintf("%.0f%% confidence", r.Confidence*100)
			formatter.Println("  %s  %s: %s → %s",
				formatter.Colorize("→", output.ColorGreen),
				formatter.Dim(r.PhaseName),
				r.CurrentProfile,
				formatter.Colorize(r.SuggestedProfile, output.ColorGreen))
			formatter.Println("    %s  %s", formatter.Dim(r.Reason), formatter.Dim("("+conf+")"))
			formatter.Println("")
		}
	} else {
		formatter.Println("  %s  No profile changes recommended yet.",
			formatter.Colorize("✓", output.ColorGreen))
		formatter.Println("  %s  Recommendations appear after 5+ runs per profile.", formatter.Dim("Tip:"))
		formatter.Println("")
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────
// sr outcomes reset <skill>
// ─────────────────────────────────────────────────────────────────

func newOutcomesResetCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset <skill-id-or-name>",
		Short: "Clear outcome history for a skill",
		Args:  cobra.ExactArgs(1),
		Example: `  sr outcomes reset code-review
  sr outcomes reset code-review --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOutcomesReset(cmd.Context(), args[0], force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runOutcomesReset(ctx context.Context, skillArg string, force bool) error {
	formatter := GetFormatter()

	repo, skillID, skillName, err := resolveOutcomeSkill(ctx, skillArg)
	if err != nil {
		return err
	}

	if !force {
		formatter.Warning("This will delete all outcome history for skill %q.", skillName)
		formatter.Println("  Use --force to confirm.")
		return nil
	}

	if err := repo.Reset(ctx, skillID); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

	formatter.Success("Cleared outcome history for %q.", skillName)
	return nil
}

// ─────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────

func resolveOutcomeSkill(ctx context.Context, skillArg string) (
	repo interface {
		GetStats(ctx context.Context, skillID string) ([]domainOutcome.ProfileStats, error)
		GetBySkill(ctx context.Context, skillID string, limit int) ([]domainOutcome.Outcome, error)
		Reset(ctx context.Context, skillID string) error
	},
	skillID, skillName string,
	err error,
) {
	container := GetContainer()
	if container == nil {
		return nil, "", "", fmt.Errorf("application not initialized")
	}

	outcomeRepo := container.OutcomeRepository()
	if outcomeRepo == nil {
		return nil, "", "", fmt.Errorf("outcome storage not available")
	}

	// Resolve skill by ID or name
	registry := container.SkillRegistry()
	if registry != nil {
		if sk := registry.GetSkill(skillArg); sk != nil {
			return outcomeRepo, sk.ID(), sk.Name(), nil
		}
		if sk := registry.GetSkillByName(skillArg); sk != nil {
			return outcomeRepo, sk.ID(), sk.Name(), nil
		}
	}

	// Fall back to using the arg as-is (for outcomes that exist in DB but skill was deleted)
	return outcomeRepo, skillArg, skillArg, nil
}
