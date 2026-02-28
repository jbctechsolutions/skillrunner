package commands

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/domain/budget"
	"github.com/jbctechsolutions/skillrunner/internal/domain/metrics"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// NewCostCmd creates the `sr cost` command group.
func NewCostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Cost analysis and savings reports",
		Long:  "Analyze API spending patterns and identify cost-saving opportunities.",
	}

	cmd.AddCommand(newCostReportCmd())
	cmd.AddCommand(newCostBreakdownCmd())
	cmd.AddCommand(newCostSavingsCmd())

	return cmd
}

// ─────────────────────────────────────────────────────────────────
// sr cost report
// ─────────────────────────────────────────────────────────────────

type costReportFlags struct {
	last7Days  bool
	last30Days bool
	since      string
	export     string // json, csv, markdown
}

func newCostReportCmd() *cobra.Command {
	var f costReportFlags

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Show cost report for a time period",
		Long: `Display a cost report with totals, a day-by-day ASCII bar chart,
and budget status for the selected period.`,
		Example: `  sr cost report --last-7-days
  sr cost report --last-30-days
  sr cost report --since 14d
  sr cost report --last-7-days --export json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, label := resolveCostFilter(f.last7Days, f.last30Days, f.since)
			return runCostReport(cmd.Context(), filter, label, f.export)
		},
	}

	cmd.Flags().BoolVar(&f.last7Days, "last-7-days", false, "Report for the last 7 days")
	cmd.Flags().BoolVar(&f.last30Days, "last-30-days", false, "Report for the last 30 days")
	cmd.Flags().StringVar(&f.since, "since", "", "Custom period (e.g. 24h, 14d)")
	cmd.Flags().StringVar(&f.export, "export", "", "Export format: json, csv, markdown")

	return cmd
}

func runCostReport(ctx context.Context, filter metrics.MetricsFilter, label, exportFmt string) error {
	formatter := GetFormatter()

	repo := costRepo()
	if repo == nil {
		return fmt.Errorf("metrics not enabled — enable observability.metrics in config")
	}

	summary, err := repo.GetCostSummary(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get cost summary: %w", err)
	}

	agg, err := repo.GetAggregatedMetrics(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	switch exportFmt {
	case "json":
		return exportCostReportJSON(summary, agg)
	case "csv":
		return exportCostReportCSV(agg)
	case "markdown":
		return exportCostReportMarkdown(summary, agg, label)
	}

	// ── Text output ──────────────────────────────────────────────
	_ = formatter.Header("Cost Report — " + label)
	_ = formatter.Println("")
	_ = formatter.Println("  %s  %s → %s",
		formatter.Dim("Period:"),
		filter.StartDate.Format("Jan 02, 2006"),
		filter.EndDate.Format("Jan 02, 2006"))
	_ = formatter.Println("")

	// Summary
	_ = formatter.SubHeader("Summary")
	_ = formatter.Println("")
	_ = formatter.Println("  %s  %s",
		formatter.Dim("Total Cost:"),
		formatter.Colorize(fmt.Sprintf("$%.4f", summary.TotalCost), output.ColorYellow))
	_ = formatter.Println("  %s  %s input / %s output (%s total)",
		formatter.Dim("Tokens:"),
		formatNumber(summary.InputTokens),
		formatNumber(summary.OutputTokens),
		formatNumber(summary.TotalTokens))
	_ = formatter.Println("  %s  %d", formatter.Dim("Executions:"), agg.TotalExecutions)
	_ = formatter.Println("")

	// Budget status
	printBudgetStatus(formatter)

	// ASCII bar chart: cost by provider
	if len(summary.ByProvider) > 0 {
		_ = formatter.SubHeader("Cost by Provider")
		_ = formatter.Println("")
		printASCIIBar(formatter, summary.ByProvider, summary.TotalCost, "$")
		_ = formatter.Println("")
	}

	// ASCII bar chart: cost by skill (top 5)
	if len(summary.BySkill) > 0 {
		_ = formatter.SubHeader("Cost by Skill")
		_ = formatter.Println("")
		top5 := topN(summary.BySkill, 5)
		printASCIIBar(formatter, top5, summary.TotalCost, "$")
		_ = formatter.Println("")
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────
// sr cost breakdown
// ─────────────────────────────────────────────────────────────────

type costBreakdownFlags struct {
	bySkill    bool
	byProvider bool
	byPhase    bool
	last7Days  bool
	last30Days bool
	since      string
	export     string
}

func newCostBreakdownCmd() *cobra.Command {
	var f costBreakdownFlags

	cmd := &cobra.Command{
		Use:   "breakdown",
		Short: "Detailed cost breakdown by dimension",
		Long: `Show itemized cost breakdown by skill, provider, or phase.
Multiple breakdown flags can be combined.`,
		Example: `  sr cost breakdown --by-skill
  sr cost breakdown --by-provider
  sr cost breakdown --by-skill --last-30-days
  sr cost breakdown --by-skill --export csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !f.bySkill && !f.byProvider && !f.byPhase {
				f.bySkill = true // default
			}
			filter, label := resolveCostFilter(f.last7Days, f.last30Days, f.since)
			return runCostBreakdown(cmd.Context(), filter, label, f, f.export)
		},
	}

	cmd.Flags().BoolVar(&f.bySkill, "by-skill", false, "Break down by skill")
	cmd.Flags().BoolVar(&f.byProvider, "by-provider", false, "Break down by provider")
	cmd.Flags().BoolVar(&f.byPhase, "by-phase", false, "Break down by phase model")
	cmd.Flags().BoolVar(&f.last7Days, "last-7-days", false, "Last 7 days")
	cmd.Flags().BoolVar(&f.last30Days, "last-30-days", false, "Last 30 days")
	cmd.Flags().StringVar(&f.since, "since", "", "Custom period (e.g. 24h, 14d)")
	cmd.Flags().StringVar(&f.export, "export", "", "Export format: json, csv, markdown")

	return cmd
}

func runCostBreakdown(ctx context.Context, filter metrics.MetricsFilter, label string, f costBreakdownFlags, exportFmt string) error {
	formatter := GetFormatter()

	repo := costRepo()
	if repo == nil {
		return fmt.Errorf("metrics not enabled")
	}

	summary, err := repo.GetCostSummary(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get cost summary: %w", err)
	}

	if exportFmt == "json" {
		return exportBreakdownJSON(summary, f)
	}
	if exportFmt == "csv" {
		return exportBreakdownCSV(summary, f)
	}
	if exportFmt == "markdown" {
		return exportBreakdownMarkdown(formatter, summary, f, label)
	}

	_ = formatter.Header("Cost Breakdown — " + label)
	_ = formatter.Println("")

	if f.byProvider && len(summary.ByProvider) > 0 {
		_ = formatter.SubHeader("By Provider")
		_ = formatter.Println("")
		printCostTable(formatter, summary.ByProvider, summary.TotalCost, "Provider")
		_ = formatter.Println("")
	}

	if f.bySkill && len(summary.BySkill) > 0 {
		_ = formatter.SubHeader("By Skill")
		_ = formatter.Println("")
		printCostTable(formatter, summary.BySkill, summary.TotalCost, "Skill")
		_ = formatter.Println("")
	}

	if f.byPhase && len(summary.ByModel) > 0 {
		_ = formatter.SubHeader("By Model (Phase)")
		_ = formatter.Println("")
		printCostTable(formatter, summary.ByModel, summary.TotalCost, "Model")
		_ = formatter.Println("")
	}

	_ = formatter.Println("  %s  %s",
		formatter.Dim("Total:"),
		formatter.Colorize(fmt.Sprintf("$%.4f", summary.TotalCost), output.ColorYellow))

	return nil
}

// ─────────────────────────────────────────────────────────────────
// sr cost savings
// ─────────────────────────────────────────────────────────────────

func newCostSavingsCmd() *cobra.Command {
	var (
		last7Days  bool
		last30Days bool
		since      string
		export     string
	)

	cmd := &cobra.Command{
		Use:   "savings",
		Short: "Estimate potential cost savings from optimization",
		Long: `Analyze your usage and project how much you could save by:
  - Switching to the cheap (local Ollama) routing profile
  - Enabling context compression
  - Improving cache hit rate`,
		Example: `  sr cost savings --potential
  sr cost savings --last-30-days
  sr cost savings --export markdown`,
		Aliases: []string{"potential"},
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, label := resolveCostFilter(last7Days, last30Days, since)
			return runCostSavings(cmd.Context(), filter, label, export)
		},
	}

	cmd.Flags().BoolVar(&last7Days, "last-7-days", false, "Analyse last 7 days")
	cmd.Flags().BoolVar(&last30Days, "last-30-days", false, "Analyse last 30 days")
	cmd.Flags().StringVar(&since, "since", "", "Custom period")
	cmd.Flags().StringVar(&export, "export", "", "Export format: json, markdown")

	return cmd
}

func runCostSavings(ctx context.Context, filter metrics.MetricsFilter, label, exportFmt string) error {
	formatter := GetFormatter()

	repo := costRepo()
	if repo == nil {
		return fmt.Errorf("metrics not enabled")
	}

	summary, err := repo.GetCostSummary(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get cost summary: %w", err)
	}

	agg, err := repo.GetAggregatedMetrics(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	// Compute saving opportunities
	savings := computeSavings(summary, agg)

	if exportFmt == "json" {
		return exportSavingsJSON(savings, summary)
	}
	if exportFmt == "markdown" {
		return exportSavingsMarkdown(savings, summary, label)
	}

	_ = formatter.Header("Cost Savings Analysis — " + label)
	_ = formatter.Println("")
	_ = formatter.Println("  %s  %s",
		formatter.Dim("Current spend:"),
		formatter.Colorize(fmt.Sprintf("$%.4f", summary.TotalCost), output.ColorYellow))
	_ = formatter.Println("")

	_ = formatter.SubHeader("Opportunities")
	_ = formatter.Println("")

	totalSavings := 0.0
	for _, opp := range savings {
		pct := ""
		if opp.SavePct > 0 {
			pct = fmt.Sprintf(" (%.0f%%)", opp.SavePct*100)
		}
		icon := "●"
		color := output.ColorGreen
		if opp.SaveUSD < 0.001 {
			icon = "○"
			color = output.ColorDim
		}
		_ = formatter.Println("  %s  %s%s",
			formatter.Colorize(icon, color),
			opp.Description,
			formatter.Dim(pct))
		if opp.SaveUSD > 0 {
			_ = formatter.Println("      %s  save ~%s",
				formatter.Dim("→"),
				formatter.Colorize(fmt.Sprintf("$%.4f", opp.SaveUSD), output.ColorGreen))
		}
		_ = formatter.Println("")
		totalSavings += opp.SaveUSD
	}

	_ = formatter.Println("  %s  %s",
		formatter.Dim("Total potential savings:"),
		formatter.Colorize(fmt.Sprintf("$%.4f", totalSavings), output.ColorGreen))

	// Annualized estimate if period < 30 days
	days := filter.EndDate.Sub(filter.StartDate).Hours() / 24
	if days > 0 && days <= 30 {
		annualSavings := totalSavings * 365 / days
		_ = formatter.Println("  %s  ~%s/year",
			formatter.Dim("Annualized:"),
			formatter.Colorize(fmt.Sprintf("$%.2f", annualSavings), output.ColorGreen))
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────
// Savings computation
// ─────────────────────────────────────────────────────────────────

// SavingsOpp describes a single optimization opportunity.
type SavingsOpp struct {
	Description string
	SaveUSD     float64
	SavePct     float64
}

func computeSavings(summary *metrics.CostSummary, agg *metrics.AggregatedMetrics) []SavingsOpp {
	var opps []SavingsOpp
	total := summary.TotalCost

	// Opportunity 1: Switch cloud calls to cheap profile
	cloudCost := 0.0
	for provider, cost := range summary.ByProvider {
		if provider != "ollama" {
			cloudCost += cost
		}
	}
	if cloudCost > 0 {
		// Cheap profile (local Ollama) is ~$0 for local, so potential saving ≈ cloud cost * 0.80
		save := cloudCost * 0.80
		opps = append(opps, SavingsOpp{
			Description: "Route to cheap profile (local Ollama) for non-critical tasks",
			SaveUSD:     save,
			SavePct:     savePct(save, total),
		})
	}

	// Opportunity 2: Context compression (saves tokens, thus cost)
	// Estimate: conservative 15% token reduction → 15% cost reduction
	compressionSave := total * 0.15
	opps = append(opps, SavingsOpp{
		Description: "Enable context compression (targets ~15% token reduction)",
		SaveUSD:     compressionSave,
		SavePct:     savePct(compressionSave, total),
	})

	// Opportunity 3: Improve cache hit rate
	cacheHitRate := 0.0
	if agg.CacheHits+agg.CacheMisses > 0 {
		cacheHitRate = float64(agg.CacheHits) / float64(agg.CacheHits+agg.CacheMisses)
	}
	if cacheHitRate < 0.50 {
		// Every additional 10% hit rate saves ~10% of cloud cost
		targetRate := math.Min(cacheHitRate+0.20, 0.70)
		additionalHits := targetRate - cacheHitRate
		cacheSave := cloudCost * additionalHits
		desc := fmt.Sprintf("Improve cache hit rate (%.0f%% → %.0f%%)",
			cacheHitRate*100, targetRate*100)
		opps = append(opps, SavingsOpp{
			Description: desc,
			SaveUSD:     cacheSave,
			SavePct:     savePct(cacheSave, total),
		})
	} else {
		opps = append(opps, SavingsOpp{
			Description: fmt.Sprintf("Cache hit rate already healthy (%.0f%%)", cacheHitRate*100),
		})
	}

	// Sort by savings descending
	sort.Slice(opps, func(i, j int) bool {
		return opps[i].SaveUSD > opps[j].SaveUSD
	})

	return opps
}

func savePct(save, total float64) float64 {
	if total == 0 {
		return 0
	}
	return save / total
}

// ─────────────────────────────────────────────────────────────────
// ASCII chart helpers
// ─────────────────────────────────────────────────────────────────

const barWidth = 30

// printASCIIBar prints a horizontal bar chart for a map of label→value.
func printASCIIBar(formatter *output.Formatter, data map[string]float64, maxVal float64, unit string) {
	if maxVal == 0 {
		return
	}

	// Sort by value descending
	type kv struct {
		K string
		V float64
	}
	items := make([]kv, 0, len(data))
	for k, v := range data {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].V > items[j].V })

	labelWidth := 16
	for _, item := range items {
		if len(item.K) > labelWidth {
			labelWidth = len(item.K)
		}
	}
	if labelWidth > 24 {
		labelWidth = 24
	}

	for _, item := range items {
		bars := int(float64(barWidth) * item.V / maxVal)
		if bars < 1 && item.V > 0 {
			bars = 1
		}
		label := item.K
		if len(label) > labelWidth {
			label = label[:labelWidth-1] + "…"
		}
		barStr := strings.Repeat("█", bars) + strings.Repeat("░", barWidth-bars)
		_ = formatter.Println("  %-*s  %s %s%.4f",
			labelWidth, label, barStr, unit, item.V)
	}
}

// printCostTable prints a table sorted by cost descending.
func printCostTable(formatter *output.Formatter, data map[string]float64, totalCost float64, colHeader string) {
	type kv struct {
		K string
		V float64
	}
	items := make([]kv, 0, len(data))
	for k, v := range data {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].V > items[j].V })

	tableData := output.TableData{
		Columns: []output.TableColumn{
			{Header: colHeader, Width: 24, Align: output.AlignLeft},
			{Header: "Cost (USD)", Width: 12, Align: output.AlignRight},
			{Header: "% of Total", Width: 12, Align: output.AlignRight},
		},
	}

	for _, item := range items {
		pct := 0.0
		if totalCost > 0 {
			pct = item.V / totalCost * 100
		}
		tableData.Rows = append(tableData.Rows, []string{
			item.K,
			fmt.Sprintf("$%.4f", item.V),
			fmt.Sprintf("%.1f%%", pct),
		})
	}

	_ = formatter.Table(tableData)
}

// ─────────────────────────────────────────────────────────────────
// Budget status helper
// ─────────────────────────────────────────────────────────────────

func printBudgetStatus(formatter *output.Formatter) {
	appCtx := GetAppContext()
	if appCtx == nil || appCtx.Config == nil {
		return
	}
	budgetCfg := appCtx.Config.Budget
	limits := budget.NewLimits(budgetCfg.DailyLimit, budgetCfg.MonthlyLimit)
	if !limits.Enabled() {
		return
	}
	_ = formatter.SubHeader("Budget Status")
	_ = formatter.Println("")
	if limits.DailyLimit > 0 {
		_ = formatter.Println("  %s  $%.2f / day", formatter.Dim("Daily limit:"), limits.DailyLimit)
	}
	if limits.MonthlyLimit > 0 {
		_ = formatter.Println("  %s  $%.2f / month", formatter.Dim("Monthly limit:"), limits.MonthlyLimit)
	}
	_ = formatter.Println("")
}

// ─────────────────────────────────────────────────────────────────
// Filter resolution
// ─────────────────────────────────────────────────────────────────

func resolveCostFilter(last7, last30 bool, since string) (metrics.MetricsFilter, string) {
	now := time.Now()

	if last30 {
		return metrics.Last30Days(), "Last 30 days"
	}
	if last7 {
		return metrics.Last7Days(), "Last 7 days"
	}
	if since != "" {
		if d, err := parseDuration(since); err == nil {
			return metrics.MetricsFilter{StartDate: now.Add(-d), EndDate: now}, since
		}
	}
	// Default: last 7 days
	return metrics.Last7Days(), "Last 7 days"
}

// costRepo is a small helper to get the metrics repository via the global container.
func costRepo() interface {
	GetCostSummary(ctx context.Context, filter metrics.MetricsFilter) (*metrics.CostSummary, error)
	GetAggregatedMetrics(ctx context.Context, filter metrics.MetricsFilter) (*metrics.AggregatedMetrics, error)
} {
	c := GetContainer()
	if c == nil {
		return nil
	}
	return c.MetricsRepository()
}

// topN returns the top n entries from a map by value.
func topN(m map[string]float64, n int) map[string]float64 {
	type kv struct {
		K string
		V float64
	}
	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].V > items[j].V })
	result := make(map[string]float64, n)
	for i, item := range items {
		if i >= n {
			break
		}
		result[item.K] = item.V
	}
	return result
}

// ─────────────────────────────────────────────────────────────────
// Export helpers: JSON
// ─────────────────────────────────────────────────────────────────

func exportCostReportJSON(summary *metrics.CostSummary, agg *metrics.AggregatedMetrics) error {
	out := map[string]any{
		"total_cost":    summary.TotalCost,
		"total_tokens":  summary.TotalTokens,
		"input_tokens":  summary.InputTokens,
		"output_tokens": summary.OutputTokens,
		"by_provider":   summary.ByProvider,
		"by_skill":      summary.BySkill,
		"by_model":      summary.ByModel,
		"executions":    agg.TotalExecutions,
		"period": map[string]string{
			"start": summary.Period.Start.Format(time.RFC3339),
			"end":   summary.Period.End.Format(time.RFC3339),
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func exportBreakdownJSON(summary *metrics.CostSummary, f costBreakdownFlags) error {
	out := map[string]any{}
	if f.byProvider {
		out["by_provider"] = summary.ByProvider
	}
	if f.bySkill {
		out["by_skill"] = summary.BySkill
	}
	if f.byPhase {
		out["by_model"] = summary.ByModel
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func exportSavingsJSON(opps []SavingsOpp, summary *metrics.CostSummary) error {
	rows := make([]map[string]any, 0, len(opps))
	for _, o := range opps {
		rows = append(rows, map[string]any{
			"description": o.Description,
			"save_usd":    o.SaveUSD,
			"save_pct":    o.SavePct,
		})
	}
	out := map[string]any{
		"current_cost":  summary.TotalCost,
		"opportunities": rows,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// ─────────────────────────────────────────────────────────────────
// Export helpers: CSV
// ─────────────────────────────────────────────────────────────────

func exportCostReportCSV(agg *metrics.AggregatedMetrics) error {
	w := csv.NewWriter(os.Stdout)
	_ = w.Write([]string{"provider", "cost_usd", "tokens_in", "tokens_out"})
	for _, p := range agg.Providers {
		_ = w.Write([]string{
			p.Name,
			fmt.Sprintf("%.6f", p.TotalCost),
			fmt.Sprintf("%d", p.TokensInput),
			fmt.Sprintf("%d", p.TokensOutput),
		})
	}
	w.Flush()
	return w.Error()
}

func exportBreakdownCSV(summary *metrics.CostSummary, f costBreakdownFlags) error {
	w := csv.NewWriter(os.Stdout)
	_ = w.Write([]string{"dimension", "name", "cost_usd"})
	writeCSVDimension := func(dim string, m map[string]float64) {
		for k, v := range m {
			_ = w.Write([]string{dim, k, fmt.Sprintf("%.6f", v)})
		}
	}
	if f.byProvider {
		writeCSVDimension("provider", summary.ByProvider)
	}
	if f.bySkill {
		writeCSVDimension("skill", summary.BySkill)
	}
	if f.byPhase {
		writeCSVDimension("model", summary.ByModel)
	}
	w.Flush()
	return w.Error()
}

// ─────────────────────────────────────────────────────────────────
// Export helpers: Markdown
// ─────────────────────────────────────────────────────────────────

func exportCostReportMarkdown(summary *metrics.CostSummary, agg *metrics.AggregatedMetrics, label string) error {
	fmt.Printf("# Cost Report — %s\n\n", label)
	fmt.Printf("**Period:** %s → %s\n\n",
		summary.Period.Start.Format("2006-01-02"),
		summary.Period.End.Format("2006-01-02"))
	fmt.Printf("| Metric | Value |\n|--------|-------|\n")
	fmt.Printf("| Total Cost | $%.4f |\n", summary.TotalCost)
	fmt.Printf("| Executions | %d |\n", agg.TotalExecutions)
	fmt.Printf("| Input Tokens | %s |\n", formatNumber(summary.InputTokens))
	fmt.Printf("| Output Tokens | %s |\n", formatNumber(summary.OutputTokens))
	fmt.Printf("\n## By Provider\n\n| Provider | Cost |\n|----------|------|\n")
	for k, v := range summary.ByProvider {
		fmt.Printf("| %s | $%.4f |\n", k, v)
	}
	fmt.Printf("\n## By Skill\n\n| Skill | Cost |\n|-------|------|\n")
	for k, v := range summary.BySkill {
		fmt.Printf("| %s | $%.4f |\n", k, v)
	}
	return nil
}

func exportBreakdownMarkdown(formatter *output.Formatter, summary *metrics.CostSummary, f costBreakdownFlags, label string) error {
	fmt.Printf("# Cost Breakdown — %s\n\n", label)
	if f.byProvider {
		fmt.Printf("## By Provider\n\n| Provider | Cost | %% |\n|----------|------|---|\n")
		for k, v := range summary.ByProvider {
			pct := 0.0
			if summary.TotalCost > 0 {
				pct = v / summary.TotalCost * 100
			}
			fmt.Printf("| %s | $%.4f | %.1f%% |\n", k, v, pct)
		}
		fmt.Println()
	}
	if f.bySkill {
		fmt.Printf("## By Skill\n\n| Skill | Cost | %% |\n|-------|------|---|\n")
		for k, v := range summary.BySkill {
			pct := 0.0
			if summary.TotalCost > 0 {
				pct = v / summary.TotalCost * 100
			}
			fmt.Printf("| %s | $%.4f | %.1f%% |\n", k, v, pct)
		}
		fmt.Println()
	}
	_ = formatter
	return nil
}

func exportSavingsMarkdown(opps []SavingsOpp, summary *metrics.CostSummary, label string) error {
	fmt.Printf("# Cost Savings — %s\n\n", label)
	fmt.Printf("**Current spend:** $%.4f\n\n", summary.TotalCost)
	fmt.Printf("## Opportunities\n\n| Opportunity | Save (USD) | Save (%%) |\n|-------------|-----------|----------|\n")
	total := 0.0
	for _, o := range opps {
		fmt.Printf("| %s | $%.4f | %.0f%% |\n", o.Description, o.SaveUSD, o.SavePct*100)
		total += o.SaveUSD
	}
	fmt.Printf("\n**Total potential savings:** $%.4f\n", total)
	return nil
}
