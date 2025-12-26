// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewCacheCmd creates the cache management command.
func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the response cache",
		Long: `Manage the LLM response cache for improved performance and cost savings.

The cache stores LLM responses and serves them for identical requests,
reducing API calls and improving response times.`,
	}

	// Add subcommands
	cmd.AddCommand(NewCacheStatsCmd())
	cmd.AddCommand(NewCacheClearCmd())
	cmd.AddCommand(NewCacheListCmd())
	cmd.AddCommand(NewCacheConfigCmd())

	return cmd
}

// NewCacheStatsCmd creates the cache stats command.
func NewCacheStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show cache statistics",
		Long:  `Display detailed statistics about the response cache including hit rate, size, and token savings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			formatter := GetFormatter()

			cache := container.ResponseCache()
			if cache == nil {
				formatter.Warning("Cache is not enabled")
				return nil
			}

			stats, err := cache.Stats(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get cache stats: %w", err)
			}

			// Display stats
			formatter.Header("Cache Statistics")
			formatter.Info("")

			formatter.Info("Entries:")
			formatter.Info("  Total:     %d", stats.TotalEntries)
			formatter.Info("  Size:      %s", formatCacheBytes(stats.TotalSize))
			formatter.Info("")

			formatter.Info("Performance:")
			formatter.Info("  Hits:      %d", stats.HitCount)
			formatter.Info("  Misses:    %d", stats.MissCount)
			formatter.Info("  Hit Rate:  %.1f%%", stats.HitRate)
			formatter.Info("")

			formatter.Info("Maintenance:")
			formatter.Info("  Evictions: %d", stats.EvictionCount)
			formatter.Info("  Expired:   %d", stats.ExpiredCount)
			formatter.Info("")

			if !stats.OldestEntry.IsZero() {
				formatter.Info("Timeline:")
				formatter.Info("  Oldest:    %s ago", formatCacheDuration(time.Since(stats.OldestEntry)))
				formatter.Info("  Newest:    %s ago", formatCacheDuration(time.Since(stats.NewestEntry)))
				formatter.Info("  Avg TTL:   %s", formatCacheDuration(stats.AvgTTL))
				formatter.Info("")
			}

			if stats.TokensSaved > 0 || stats.CostSaved > 0 {
				formatter.Info("Savings:")
				formatter.Info("  Tokens:    %d", stats.TokensSaved)
				if stats.CostSaved > 0 {
					formatter.Info("  Est. Cost: $%.4f", stats.CostSaved)
				}
			}

			return nil
		},
	}

	return cmd
}

// NewCacheClearCmd creates the cache clear command.
func NewCacheClearCmd() *cobra.Command {
	var confirm bool
	var expired bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear the cache",
		Long:  `Clear all entries from the response cache or just expired entries.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			formatter := GetFormatter()

			cache := container.ResponseCache()
			if cache == nil {
				formatter.Warning("Cache is not enabled")
				return nil
			}

			if expired {
				// Only clear expired entries
				removed, err := cache.Cleanup(cmd.Context())
				if err != nil {
					return fmt.Errorf("failed to cleanup cache: %w", err)
				}
				formatter.Success("Removed %d expired cache entries", removed)
				return nil
			}

			if !confirm {
				formatter.Warning("This will clear ALL cached entries.")
				formatter.Info("Use --confirm to proceed, or --expired to only clear expired entries.")
				return nil
			}

			if err := cache.Clear(cmd.Context()); err != nil {
				return fmt.Errorf("failed to clear cache: %w", err)
			}

			formatter.Success("Cache cleared successfully")
			return nil
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm clearing all cache entries")
	cmd.Flags().BoolVar(&expired, "expired", false, "Only clear expired entries")

	return cmd
}

// NewCacheListCmd creates the cache list command.
func NewCacheListCmd() *cobra.Command {
	var pattern string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cached entries",
		Long:  `List entries in the response cache with optional filtering.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			container := GetContainer()
			if container == nil {
				return fmt.Errorf("application not initialized")
			}

			formatter := GetFormatter()

			cache := container.ResponseCache()
			if cache == nil {
				formatter.Warning("Cache is not enabled")
				return nil
			}

			keys, err := cache.Keys(cmd.Context(), pattern)
			if err != nil {
				return fmt.Errorf("failed to list cache keys: %w", err)
			}

			if len(keys) == 0 {
				formatter.Info("No cached entries found")
				return nil
			}

			formatter.Header("Cached Entries")
			formatter.Info("")

			displayed := 0
			for _, key := range keys {
				if limit > 0 && displayed >= limit {
					formatter.Info("... and %d more entries", len(keys)-displayed)
					break
				}

				entry, found := cache.GetEntry(cmd.Context(), key)
				if !found {
					continue
				}

				// Format the entry
				ttlRemaining := time.Until(entry.ExpiresAt)
				formatter.Info("Key: %s", truncateCacheKey(key, 40))
				formatter.Info("  Model:   %s", entry.ModelID)
				formatter.Info("  Size:    %s", formatCacheBytes(entry.Size))
				formatter.Info("  Hits:    %d", entry.HitCount)
				formatter.Info("  TTL:     %s remaining", formatCacheDuration(ttlRemaining))
				formatter.Info("")

				displayed++
			}

			formatter.Info("Total: %d entries", len(keys))
			return nil
		},
	}

	cmd.Flags().StringVarP(&pattern, "pattern", "p", "", "Filter by key pattern (regex)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum entries to display")

	return cmd
}

// NewCacheConfigCmd creates the cache config command.
func NewCacheConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show cache configuration",
		Long:  `Display the current cache configuration settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := GetAppContext()
			if ctx == nil {
				return fmt.Errorf("application not initialized")
			}

			formatter := GetFormatter()
			cfg := ctx.Config.Cache

			formatter.Header("Cache Configuration")
			formatter.Info("")
			formatter.Info("Enabled:        %v", cfg.Enabled)
			formatter.Info("Default TTL:    %s", formatCacheDuration(cfg.DefaultTTL))
			formatter.Info("Max Memory:     %s", formatCacheBytes(cfg.MaxMemorySize))
			formatter.Info("Max Disk:       %s", formatCacheBytes(cfg.MaxDiskSize))
			formatter.Info("Cleanup Period: %s", formatCacheDuration(cfg.CleanupPeriod))

			return nil
		},
	}

	return cmd
}

// Helper functions for cache formatting

func formatCacheBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatCacheDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}

	switch {
	case d >= 24*time.Hour:
		days := d / (24 * time.Hour)
		return fmt.Sprintf("%dd", days)
	case d >= time.Hour:
		hours := d / time.Hour
		return fmt.Sprintf("%dh", hours)
	case d >= time.Minute:
		mins := d / time.Minute
		return fmt.Sprintf("%dm", mins)
	case d >= time.Second:
		secs := d / time.Second
		return fmt.Sprintf("%ds", secs)
	default:
		return fmt.Sprintf("%dms", d/time.Millisecond)
	}
}

func truncateCacheKey(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// EscapePattern escapes special regex characters in a string.
func EscapePattern(s string) string {
	special := []string{"\\", ".", "+", "*", "?", "(", ")", "[", "]", "{", "}", "|", "^", "$"}
	for _, c := range special {
		s = strings.ReplaceAll(s, c, "\\"+c)
	}
	return s
}
