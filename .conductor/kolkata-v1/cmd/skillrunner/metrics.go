package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/metrics"
	"github.com/jbctechsolutions/skillrunner/internal/types"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "View routing metrics and costs",
	Long: `View metrics and cost tracking for Skillrunner routing operations.

Examples:
  skill metrics                    # Show summary
  skill metrics --format json      # JSON output
  skill metrics --since 24h        # Last 24 hours
  skill metrics --export costs.csv # Export to CSV`,
	RunE: showMetrics,
}

var (
	metricsFormat string
	metricsSince  string
	metricsExport string
)

func init() {
	metricsCmd.Flags().StringVarP(&metricsFormat, "format", "f", "table", "Output format (table|json)")
	metricsCmd.Flags().StringVar(&metricsSince, "since", "", "Time range (e.g., 24h, 7d, 1w)")
	metricsCmd.Flags().StringVar(&metricsExport, "export", "", "Export to file (CSV or JSON)")
	rootCmd.AddCommand(metricsCmd)
}

func showMetrics(cmd *cobra.Command, args []string) error {
	// Create metrics storage
	storage, err := metrics.NewStorage()
	if err != nil {
		return fmt.Errorf("create metrics storage: %w", err)
	}

	// Parse time range
	var since time.Time
	if metricsSince != "" {
		duration, err := parseDuration(metricsSince)
		if err != nil {
			return fmt.Errorf("parse time range: %w", err)
		}
		since = time.Now().Add(-duration)
	} else {
		// Default to last 24 hours
		since = time.Now().Add(-24 * time.Hour)
	}

	// Load metrics
	routerMetrics := storage.GetMetrics(since)
	records := storage.GetRecords(since, "", "")

	if metricsExport != "" {
		// Export all records
		return exportMetrics(records, metricsExport)
	}

	if metricsFormat == "json" {
		return printMetricsJSON(routerMetrics)
	}

	return printMetricsTable(routerMetrics, records)
}

func printMetricsTable(metrics *types.RouterMetrics, records []metrics.ExecutionRecord) error {
	fmt.Println("\nSkillrunner Metrics")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("Time Range:     %s\n", formatTimeRange(metrics.StartTime))
	fmt.Printf("Local Calls:    %d\n", metrics.LocalCalls)
	fmt.Printf("Cloud Calls:    %d\n", metrics.CloudCalls)
	fmt.Printf("Total Tokens:   %s\n", formatTokens(metrics.TotalTokens))
	fmt.Printf("Estimated Cost: $%.4f\n", metrics.EstimatedCost)
	totalCalls := metrics.LocalCalls + metrics.CloudCalls
	// Defensive check: prevent division by zero when both LocalCalls and CloudCalls are 0
	if totalCalls > 0 {
		fmt.Printf("Avg Latency:    %.2fs\n", metrics.ElapsedTime/float64(totalCalls))
	} else {
		fmt.Printf("Avg Latency:    N/A (no calls)\n")
	}
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	// Cost breakdown
	fmt.Println("Cost Breakdown:")
	fmt.Printf("  Local (Ollama):  $0.00 (free)\n")
	fmt.Printf("  Cloud APIs:      $%.4f\n", metrics.EstimatedCost)
	// Defensive check: prevent division by zero when both LocalCalls and CloudCalls are 0
	if totalCalls > 0 {
		fmt.Printf("  Savings:         ~%.0f%% (using local models)\n",
			(float64(metrics.LocalCalls)/float64(totalCalls))*100)
	} else {
		fmt.Printf("  Savings:         N/A (no calls)\n")
	}
	fmt.Println()

	// Model usage statistics
	if len(records) > 0 {
		modelStats := calculateModelStats(records)
		if len(modelStats) > 0 {
			fmt.Println("Model Usage:")
			for model, stats := range modelStats {
				fmt.Printf("  %s:\n", model)
				fmt.Printf("    Calls:     %d\n", stats.Calls)
				fmt.Printf("    Tokens:    %s\n", formatTokens(stats.Tokens))
				fmt.Printf("    Cost:      $%.4f\n", stats.Cost)
				if stats.Calls > 0 {
					fmt.Printf("    Avg Time:  %.2fs\n", stats.AvgDuration)
				}
			}
			fmt.Println()
		}

		// Skill usage statistics
		skillStats := calculateSkillStats(records)
		if len(skillStats) > 0 {
			fmt.Println("Skill Usage:")
			for skill, count := range skillStats {
				fmt.Printf("  %s: %d execution(s)\n", skill, count)
			}
			fmt.Println()
		}

		// Performance metrics (p95, p99 if we have enough data)
		if len(records) >= 20 {
			durations := make([]float64, 0, len(records))
			for _, r := range records {
				if r.Success {
					durations = append(durations, float64(r.DurationMs)/1000.0)
				}
			}
			if len(durations) > 0 {
				p95, p99 := calculatePercentiles(durations)
				fmt.Println("Performance Metrics:")
				fmt.Printf("  P95 Latency: %.2fs\n", p95)
				fmt.Printf("  P99 Latency: %.2fs\n", p99)
				fmt.Println()
			}
		}
	}

	return nil
}

// ModelStats contains statistics for a model
type ModelStats struct {
	Calls       int
	Tokens      int
	Cost        float64
	AvgDuration float64
}

// calculateModelStats calculates statistics per model
func calculateModelStats(records []metrics.ExecutionRecord) map[string]ModelStats {
	stats := make(map[string]ModelStats)

	for _, record := range records {
		if !record.Success {
			continue
		}

		modelStat := stats[record.Model]
		modelStat.Calls++
		modelStat.Tokens += record.InputTokens + record.OutputTokens
		modelStat.Cost += record.Cost
		modelStat.AvgDuration += float64(record.DurationMs) / 1000.0
		stats[record.Model] = modelStat
	}

	// Calculate averages
	for model, stat := range stats {
		if stat.Calls > 0 {
			stat.AvgDuration = stat.AvgDuration / float64(stat.Calls)
			stats[model] = stat
		}
	}

	return stats
}

// calculateSkillStats calculates execution count per skill
func calculateSkillStats(records []metrics.ExecutionRecord) map[string]int {
	stats := make(map[string]int)

	for _, record := range records {
		if record.Skill != "" {
			stats[record.Skill]++
		}
	}

	return stats
}

// calculatePercentiles calculates p95 and p99 percentiles
func calculatePercentiles(values []float64) (p95, p99 float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// Simple sort (bubble sort for small arrays)
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p95Idx := int(float64(len(sorted)) * 0.95)
	p99Idx := int(float64(len(sorted)) * 0.99)

	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}
	if p99Idx >= len(sorted) {
		p99Idx = len(sorted) - 1
	}

	return sorted[p95Idx], sorted[p99Idx]
}

func printMetricsJSON(metrics *types.RouterMetrics) error {
	output, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}
	fmt.Println(string(output))
	return nil
}

func exportMetrics(records []metrics.ExecutionRecord, filename string) error {
	if len(filename) == 0 {
		return fmt.Errorf("filename required for export")
	}

	// Determine format from extension
	if strings.HasSuffix(filename, ".csv") {
		return exportCSV(records, filename)
	} else if strings.HasSuffix(filename, ".json") {
		return exportJSON(records, filename)
	}

	// Default to JSON if no extension
	return exportJSON(records, filename)
}

func exportCSV(records []metrics.ExecutionRecord, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "timestamp,skill,model,input_tokens,output_tokens,cost,duration_ms,success,error\n")

	// Write records
	for _, record := range records {
		fmt.Fprintf(file, "%s,%s,%s,%d,%d,%.6f,%d,%t,%s\n",
			record.Timestamp.Format(time.RFC3339),
			record.Skill,
			record.Model,
			record.InputTokens,
			record.OutputTokens,
			record.Cost,
			record.DurationMs,
			record.Success,
			strings.ReplaceAll(record.Error, ",", ";"), // Replace commas in error messages
		)
	}

	return nil
}

func exportJSON(records []metrics.ExecutionRecord, filename string) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal records: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func parseDuration(s string) (time.Duration, error) {
	// Support formats like: 24h, 7d, 1w, 30m
	s = strings.ToLower(strings.TrimSpace(s))

	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	unit := s[len(s)-1:]
	value := s[:len(s)-1]

	var multiplier time.Duration
	switch unit {
	case "m":
		multiplier = time.Minute
	case "h":
		multiplier = time.Hour
	case "d":
		multiplier = 24 * time.Hour
	case "w":
		multiplier = 7 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}

	var num float64
	if _, err := fmt.Sscanf(value, "%f", &num); err != nil {
		return 0, fmt.Errorf("parse duration value: %w", err)
	}

	return time.Duration(num * float64(multiplier)), nil
}

func formatTimeRange(start time.Time) string {
	duration := time.Since(start)
	if duration < time.Hour {
		return fmt.Sprintf("Last %.0f minutes", duration.Minutes())
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("Last %.1f hours", duration.Hours())
	}
	return fmt.Sprintf("Last %.1f days", duration.Hours()/24)
}

func formatTokens(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.2fM", float64(tokens)/1000000)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%.2fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}
