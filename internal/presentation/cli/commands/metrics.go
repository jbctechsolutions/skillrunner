package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/application"
	"github.com/jbctechsolutions/skillrunner/internal/domain/metrics"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/output"
)

// ProviderMetrics represents usage metrics for a single provider.
type ProviderMetrics struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	TotalRequests   int     `json:"total_requests"`
	SuccessfulCount int     `json:"successful_count"`
	FailedCount     int     `json:"failed_count"`
	TokensInput     int64   `json:"tokens_input"`
	TokensOutput    int64   `json:"tokens_output"`
	EstimatedCost   float64 `json:"estimated_cost"`
	AvgLatencyMs    int64   `json:"avg_latency_ms"`
}

// SkillMetrics represents usage metrics for a single skill.
type SkillMetrics struct {
	Name        string  `json:"name"`
	Executions  int     `json:"executions"`
	SuccessRate float64 `json:"success_rate"`
	AvgDuration string  `json:"avg_duration"`
}

// UsageMetrics represents the complete usage metrics.
type UsageMetrics struct {
	Period             string            `json:"period"`
	StartDate          string            `json:"start_date"`
	EndDate            string            `json:"end_date"`
	TotalRequests      int               `json:"total_requests"`
	SuccessfulCount    int               `json:"successful_count"`
	FailedCount        int               `json:"failed_count"`
	SuccessRate        float64           `json:"success_rate"`
	TotalTokensInput   int64             `json:"total_tokens_input"`
	TotalTokensOutput  int64             `json:"total_tokens_output"`
	TotalEstimatedCost float64           `json:"total_estimated_cost"`
	ProviderMetrics    []ProviderMetrics `json:"provider_metrics"`
	TopSkills          []SkillMetrics    `json:"top_skills"`
}

// NewMetricsCmd creates the metrics command.
func NewMetricsCmd() *cobra.Command {
	var since string

	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Display usage and cost metrics",
		Long: `Display usage statistics and cost metrics for skillrunner.

This includes:
  • Total requests and success/failure rates
  • Tokens used per provider (input and output)
  • Estimated costs per provider
  • Top skills by usage

Use --since to filter by time range (e.g., "24h", "7d", "30d").`,
		Example: `  # Show metrics for the last 24 hours
  sr metrics --since 24h

  # Show metrics for the last 7 days
  sr metrics --since 7d

  # Get metrics as JSON for scripting
  sr metrics --since 30d -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMetrics(since)
		},
	}

	cmd.Flags().StringVar(&since, "since", "24h", "time range for metrics (e.g., 24h, 7d, 30d)")

	return cmd
}

func runMetrics(since string) error {
	formatter := GetFormatter()

	// Parse the since duration
	duration, err := parseDuration(since)
	if err != nil {
		return fmt.Errorf("invalid time range: %w", err)
	}

	// Try to get real metrics from the database
	usageMetrics, err := getRealMetrics(duration)
	if err != nil {
		// Fall back to mock data if metrics unavailable
		formatter.Println("%s Could not retrieve metrics: %v",
			formatter.Colorize("Warning:", output.ColorYellow), err)
		formatter.Println("%s Showing mock data for demonstration purposes",
			formatter.Colorize("Info:", output.ColorBlue))
		formatter.Println("")
		usageMetrics = getMockMetrics(duration)
	}

	// Handle JSON output
	if formatter.Format() == output.FormatJSON {
		return formatter.JSON(usageMetrics)
	}

	// Print text output
	return printMetricsText(formatter, usageMetrics)
}

// getRealMetrics retrieves actual metrics from the database.
func getRealMetrics(duration time.Duration) (UsageMetrics, error) {
	ctx := context.Background()

	// Create container to access metrics repository
	container, err := application.NewContainer(nil, false)
	if err != nil {
		return UsageMetrics{}, fmt.Errorf("failed to initialize container: %w", err)
	}
	defer func() { _ = container.Close() }()

	// Get the metrics repository
	metricsRepo := container.MetricsRepository()
	if metricsRepo == nil {
		return UsageMetrics{}, fmt.Errorf("metrics not enabled in configuration")
	}

	// Create filter for the time period
	now := time.Now()
	startTime := now.Add(-duration)
	filter := metrics.MetricsFilter{
		StartDate: startTime,
		EndDate:   now,
	}

	// Get aggregated metrics
	aggregated, err := metricsRepo.GetAggregatedMetrics(ctx, filter)
	if err != nil {
		return UsageMetrics{}, fmt.Errorf("failed to get aggregated metrics: %w", err)
	}

	// Convert domain metrics to CLI format
	return convertToUsageMetrics(aggregated, duration), nil
}

// convertToUsageMetrics converts domain metrics to the CLI display format.
func convertToUsageMetrics(agg *metrics.AggregatedMetrics, duration time.Duration) UsageMetrics {
	// Convert provider metrics
	providers := make([]ProviderMetrics, 0, len(agg.Providers))
	for _, p := range agg.Providers {
		providerType := "cloud"
		if p.Name == "ollama" {
			providerType = "local"
		}

		providers = append(providers, ProviderMetrics{
			Name:            p.Name,
			Type:            providerType,
			TotalRequests:   int(p.TotalRequests),
			SuccessfulCount: int(p.SuccessCount),
			FailedCount:     int(p.FailedCount),
			TokensInput:     p.TokensInput,
			TokensOutput:    p.TokensOutput,
			EstimatedCost:   p.TotalCost,
			AvgLatencyMs:    p.AvgLatency.Milliseconds(),
		})
	}

	// Convert skill metrics
	skills := make([]SkillMetrics, 0, len(agg.Skills))
	for _, s := range agg.Skills {
		skills = append(skills, SkillMetrics{
			Name:        s.SkillName,
			Executions:  int(s.TotalRuns),
			SuccessRate: s.SuccessRate * 100, // Convert to percentage
			AvgDuration: formatMetricsDuration(s.AvgDuration),
		})
	}

	// Calculate success rate as percentage
	var successRate float64
	if agg.TotalExecutions > 0 {
		successRate = agg.SuccessRate * 100
	}

	return UsageMetrics{
		Period:             duration.String(),
		StartDate:          agg.Period.Start.Format(time.RFC3339),
		EndDate:            agg.Period.End.Format(time.RFC3339),
		TotalRequests:      int(agg.TotalExecutions),
		SuccessfulCount:    int(agg.SuccessCount),
		FailedCount:        int(agg.FailedCount),
		SuccessRate:        successRate,
		TotalTokensInput:   agg.InputTokens,
		TotalTokensOutput:  agg.OutputTokens,
		TotalEstimatedCost: agg.TotalCost,
		ProviderMetrics:    providers,
		TopSkills:          skills,
	}
}

// formatMetricsDuration formats a duration for human display in metrics output.
func formatMetricsDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// parseDuration parses a duration string like "24h", "7d", "30d".
func parseDuration(s string) (time.Duration, error) {
	// Check for day format
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}

	// Try standard duration parsing
	return time.ParseDuration(s)
}

// getMockMetrics returns simulated metrics data.
func getMockMetrics(duration time.Duration) UsageMetrics {
	now := time.Now()
	startTime := now.Add(-duration)

	// Scale metrics based on duration
	scale := float64(duration) / float64(24*time.Hour)
	if scale < 1 {
		scale = 1
	}

	totalRequests := int(150 * scale)
	successfulCount := int(142 * scale)
	failedCount := totalRequests - successfulCount

	return UsageMetrics{
		Period:             duration.String(),
		StartDate:          startTime.Format(time.RFC3339),
		EndDate:            now.Format(time.RFC3339),
		TotalRequests:      totalRequests,
		SuccessfulCount:    successfulCount,
		FailedCount:        failedCount,
		SuccessRate:        float64(successfulCount) / float64(totalRequests) * 100,
		TotalTokensInput:   int64(485000 * scale),
		TotalTokensOutput:  int64(127000 * scale),
		TotalEstimatedCost: 2.47 * scale,
		ProviderMetrics: []ProviderMetrics{
			{
				Name:            "ollama",
				Type:            "local",
				TotalRequests:   int(89 * scale),
				SuccessfulCount: int(88 * scale),
				FailedCount:     int(1 * scale),
				TokensInput:     int64(245000 * scale),
				TokensOutput:    int64(67000 * scale),
				EstimatedCost:   0.00, // Local is free
				AvgLatencyMs:    45,
			},
			{
				Name:            "anthropic",
				Type:            "cloud",
				TotalRequests:   int(42 * scale),
				SuccessfulCount: int(41 * scale),
				FailedCount:     int(1 * scale),
				TokensInput:     int64(156000 * scale),
				TokensOutput:    int64(38000 * scale),
				EstimatedCost:   1.85 * scale,
				AvgLatencyMs:    320,
			},
			{
				Name:            "openai",
				Type:            "cloud",
				TotalRequests:   int(15 * scale),
				SuccessfulCount: int(12 * scale),
				FailedCount:     int(3 * scale),
				TokensInput:     int64(68000 * scale),
				TokensOutput:    int64(18000 * scale),
				EstimatedCost:   0.52 * scale,
				AvgLatencyMs:    445,
			},
			{
				Name:            "groq",
				Type:            "cloud",
				TotalRequests:   int(4 * scale),
				SuccessfulCount: int(1 * scale),
				FailedCount:     int(3 * scale),
				TokensInput:     int64(16000 * scale),
				TokensOutput:    int64(4000 * scale),
				EstimatedCost:   0.10 * scale,
				AvgLatencyMs:    89,
			},
		},
		TopSkills: []SkillMetrics{
			{
				Name:        "code-review",
				Executions:  int(45 * scale),
				SuccessRate: 97.8,
				AvgDuration: "4.2s",
			},
			{
				Name:        "test-gen",
				Executions:  int(38 * scale),
				SuccessRate: 94.7,
				AvgDuration: "6.8s",
			},
			{
				Name:        "doc-gen",
				Executions:  int(32 * scale),
				SuccessRate: 100.0,
				AvgDuration: "3.1s",
			},
		},
	}
}

// printMetricsText prints the metrics in human-readable format.
func printMetricsText(formatter *output.Formatter, metrics UsageMetrics) error {
	// Header
	formatter.Header("Skillrunner Metrics")
	formatter.Println("")

	// Time period
	formatter.Println("  %s  %s to %s",
		formatter.Dim("Period:"),
		formatDateTime(metrics.StartDate),
		formatDateTime(metrics.EndDate))
	formatter.Println("")

	// Overall summary
	formatter.SubHeader("Summary")
	formatter.Println("")

	successRateColor := output.ColorGreen
	if metrics.SuccessRate < 90 {
		successRateColor = output.ColorYellow
	}
	if metrics.SuccessRate < 75 {
		successRateColor = output.ColorRed
	}

	formatter.Println("  %s  %d", formatter.Dim("Total Requests:"), metrics.TotalRequests)
	formatter.Println("  %s  %s (%d successful, %d failed)",
		formatter.Dim("Success Rate:"),
		formatter.Colorize(fmt.Sprintf("%.1f%%", metrics.SuccessRate), successRateColor),
		metrics.SuccessfulCount,
		metrics.FailedCount)
	formatter.Println("  %s  %s input, %s output",
		formatter.Dim("Total Tokens:"),
		formatNumber(metrics.TotalTokensInput),
		formatNumber(metrics.TotalTokensOutput))
	formatter.Println("  %s  %s",
		formatter.Dim("Estimated Cost:"),
		formatter.Colorize(fmt.Sprintf("$%.2f", metrics.TotalEstimatedCost), output.ColorYellow))
	formatter.Println("")

	// Provider breakdown
	formatter.SubHeader("Provider Usage")
	formatter.Println("")

	// Table for providers
	tableData := output.TableData{
		Columns: []output.TableColumn{
			{Header: "Provider", Width: 12, Align: output.AlignLeft},
			{Header: "Type", Width: 8, Align: output.AlignLeft},
			{Header: "Requests", Width: 10, Align: output.AlignRight},
			{Header: "Success", Width: 10, Align: output.AlignRight},
			{Header: "Tokens In", Width: 12, Align: output.AlignRight},
			{Header: "Tokens Out", Width: 12, Align: output.AlignRight},
			{Header: "Cost", Width: 10, Align: output.AlignRight},
			{Header: "Avg Latency", Width: 12, Align: output.AlignRight},
		},
		Rows: make([][]string, 0, len(metrics.ProviderMetrics)),
	}

	for _, p := range metrics.ProviderMetrics {
		successRate := float64(p.SuccessfulCount) / float64(p.TotalRequests) * 100
		if p.TotalRequests == 0 {
			successRate = 0
		}

		costStr := "$0.00"
		if p.EstimatedCost > 0 {
			costStr = fmt.Sprintf("$%.2f", p.EstimatedCost)
		}

		tableData.Rows = append(tableData.Rows, []string{
			p.Name,
			p.Type,
			fmt.Sprintf("%d", p.TotalRequests),
			fmt.Sprintf("%.1f%%", successRate),
			formatNumber(p.TokensInput),
			formatNumber(p.TokensOutput),
			costStr,
			fmt.Sprintf("%dms", p.AvgLatencyMs),
		})
	}

	if err := formatter.Table(tableData); err != nil {
		return err
	}

	formatter.Println("")

	// Top skills
	formatter.SubHeader("Top Skills")
	formatter.Println("")

	skillTableData := output.TableData{
		Columns: []output.TableColumn{
			{Header: "Skill", Width: 20, Align: output.AlignLeft},
			{Header: "Executions", Width: 12, Align: output.AlignRight},
			{Header: "Success Rate", Width: 14, Align: output.AlignRight},
			{Header: "Avg Duration", Width: 14, Align: output.AlignRight},
		},
		Rows: make([][]string, 0, len(metrics.TopSkills)),
	}

	for _, s := range metrics.TopSkills {
		skillTableData.Rows = append(skillTableData.Rows, []string{
			s.Name,
			fmt.Sprintf("%d", s.Executions),
			fmt.Sprintf("%.1f%%", s.SuccessRate),
			s.AvgDuration,
		})
	}

	if err := formatter.Table(skillTableData); err != nil {
		return err
	}

	formatter.Println("")

	return nil
}

// formatDateTime formats an RFC3339 date string for display.
func formatDateTime(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.Format("Jan 02, 2006 15:04")
}

// formatNumber formats a large number with K/M suffixes.
func formatNumber(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
