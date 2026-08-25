package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/domain/budget"
	infraStorage "github.com/jbctechsolutions/skillrunner/internal/infrastructure/storage"
)

// NewBudgetCmd creates the `sr budget` command group.
func NewBudgetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget",
		Short: "View and manage API spend budgets",
		Long:  `Show current spending, check limits, and review budget alerts.`,
	}
	cmd.AddCommand(newBudgetStatusCmd())
	return cmd
}

// newBudgetStatusCmd implements `sr budget status`.
func newBudgetStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current spend vs. configured limits",
		Long: `Display daily and monthly API spending alongside configured budget limits.

Alerts are shown when spending reaches 50%, 80%, 90%, or 100% of a limit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}
			formatter := GetFormatter()
			ctx := context.Background()

			// Load limits from config
			appCtx := GetAppContext()
			var limits budget.Limits
			if appCtx != nil && appCtx.Config != nil {
				limits = budget.NewLimits(
					appCtx.Config.Budget.DailyLimit,
					appCtx.Config.Budget.MonthlyLimit,
				)
			}

			// Load current usage from SQLite
			budgetRepo, err := infraStorage.NewBudgetRepository(container.DB())
			if err != nil {
				return fmt.Errorf("budget repository: %w", err)
			}
			usage, err := budgetRepo.GetUsage(ctx)
			if err != nil {
				return fmt.Errorf("get usage: %w", err)
			}

			// Print spend summary
			_ = formatter.Header("Budget Status")
			_ = formatter.Println("")
			_ = formatter.Item("Daily spend", fmt.Sprintf("$%.4f", usage.DailySpend))
			if limits.DailyLimit > 0 {
				pct := usage.DailySpend / limits.DailyLimit * 100
				_ = formatter.Item("Daily limit", fmt.Sprintf("$%.2f  (%.1f%% used)", limits.DailyLimit, pct))
			} else {
				_ = formatter.Item("Daily limit", "not set")
			}
			_ = formatter.Println("")
			_ = formatter.Item("Monthly spend", fmt.Sprintf("$%.4f", usage.MonthlySpend))
			if limits.MonthlyLimit > 0 {
				pct := usage.MonthlySpend / limits.MonthlyLimit * 100
				_ = formatter.Item("Monthly limit", fmt.Sprintf("$%.2f  (%.1f%% used)", limits.MonthlyLimit, pct))
			} else {
				_ = formatter.Item("Monthly limit", "not set")
			}

			// Show alerts
			alerts := budget.CheckAlerts(limits, usage)
			if len(alerts) > 0 {
				_ = formatter.Println("")
				for _, a := range alerts {
					if a.IsError() {
						_ = formatter.Error("Alert: %s", a.Message())
					} else {
						_ = formatter.Warning("Alert: %s", a.Message())
					}
				}
			}

			// Tip if no limits configured
			if !limits.Enabled() {
				_ = formatter.Println("")
				_ = formatter.Info("No budget limits configured.")
				_ = formatter.Info("Set limits in ~/.skillrunner/config.yaml:")
				_ = formatter.Println("  budget:")
				_ = formatter.Println("    daily_limit: 5.00")
				_ = formatter.Println("    monthly_limit: 50.00")
			}

			return nil
		},
	}
}
