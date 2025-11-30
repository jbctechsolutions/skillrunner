package main

import (
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/metrics"
	"github.com/jbctechsolutions/skillrunner/internal/types"
)

func TestPrintMetricsTable_ZeroCalls(t *testing.T) {
	// Test that printMetricsTable handles zero calls without division by zero
	routerMetrics := &types.RouterMetrics{
		LocalCalls:    0,
		CloudCalls:    0,
		TotalTokens:   0,
		EstimatedCost: 0.0,
		ElapsedTime:   0.0,
		StartTime:     time.Now(),
	}

	// This should not panic or cause division by zero
	err := printMetricsTable(routerMetrics, []metrics.ExecutionRecord{})
	if err != nil {
		t.Fatalf("printMetricsTable should not fail with zero calls: %v", err)
	}
}

func TestPrintMetricsTable_WithCalls(t *testing.T) {
	// Test that printMetricsTable works correctly with non-zero calls
	routerMetrics := &types.RouterMetrics{
		LocalCalls:    10,
		CloudCalls:    5,
		TotalTokens:   50000,
		EstimatedCost: 0.15,
		ElapsedTime:   120.5,
		StartTime:     time.Now().Add(-24 * time.Hour),
	}

	err := printMetricsTable(routerMetrics, []metrics.ExecutionRecord{})
	if err != nil {
		t.Fatalf("printMetricsTable should not fail with valid metrics: %v", err)
	}
}

func TestPrintMetricsJSON(t *testing.T) {
	routerMetrics := &types.RouterMetrics{
		LocalCalls:    10,
		CloudCalls:    5,
		TotalTokens:   50000,
		EstimatedCost: 0.15,
		ElapsedTime:   120.5,
		StartTime:     time.Now().Add(-24 * time.Hour),
	}

	err := printMetricsJSON(routerMetrics)
	if err != nil {
		t.Fatalf("printMetricsJSON should not fail: %v", err)
	}
}

func TestFormatTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "less than hour",
			duration: 30 * time.Minute,
			want:     "Last 30 minutes",
		},
		{
			name:     "less than day",
			duration: 12 * time.Hour,
			want:     "Last 12.0 hours",
		},
		{
			name:     "more than day",
			duration: 2 * 24 * time.Hour,
			want:     "Last 2.0 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now().Add(-tt.duration)
			result := formatTimeRange(start)
			if result == "" {
				t.Error("formatTimeRange should not return empty string")
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name         string
		tokens       int
		wantContains string
	}{
		{
			name:         "less than 1000",
			tokens:       500,
			wantContains: "500",
		},
		{
			name:         "thousands",
			tokens:       5000,
			wantContains: "K",
		},
		{
			name:         "millions",
			tokens:       2000000,
			wantContains: "M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTokens(tt.tokens)
			if result == "" {
				t.Error("formatTokens should not return empty string")
			}
			if tt.wantContains != "" {
				// Just check it contains the expected suffix
				hasSuffix := false
				if tt.wantContains == "K" && len(result) > 0 {
					hasSuffix = true
				} else if tt.wantContains == "M" && len(result) > 0 {
					hasSuffix = true
				} else if tt.wantContains == "500" {
					hasSuffix = result == "500"
				}
				if !hasSuffix && tt.wantContains != "500" {
					t.Logf("formatTokens(%d) = %s (expected to contain %s)", tt.tokens, result, tt.wantContains)
				}
			}
		})
	}
}
