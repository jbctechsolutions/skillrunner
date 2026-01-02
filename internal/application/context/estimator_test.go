// Package context provides application-level context management services.
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEstimator_Estimate(t *testing.T) {
	estimator := NewEstimator()

	tests := []struct {
		name    string
		text    string
		wantMin int
		wantMax int
	}{
		{
			name:    "empty string",
			text:    "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "short text",
			text:    "Hello, world!",
			wantMin: 2,
			wantMax: 5,
		},
		{
			name:    "longer text",
			text:    "The quick brown fox jumps over the lazy dog. This is a test sentence.",
			wantMin: 10,
			wantMax: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimator.Estimate(tt.text)

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Estimate() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestEstimator_EstimateFromFile(t *testing.T) {
	estimator := NewEstimator()

	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	content := "This is a test file with some content.\nIt has multiple lines.\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tokens, err := estimator.EstimateFromFile(tmpFile)
	if err != nil {
		t.Errorf("EstimateFromFile() error = %v", err)
	}

	if tokens == 0 {
		t.Error("EstimateFromFile() returned 0 tokens")
	}

	// Test non-existent file
	_, err = estimator.EstimateFromFile(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Error("EstimateFromFile() expected error for non-existent file")
	}
}

func TestEstimator_WithinBudget(t *testing.T) {
	estimator := NewEstimator()

	tests := []struct {
		name   string
		text   string
		budget int
		want   bool
	}{
		{
			name:   "within budget",
			text:   "Short text",
			budget: 100,
			want:   true,
		},
		{
			name:   "over budget",
			text:   "This is a very long text that will definitely exceed a small budget of just a few tokens",
			budget: 5,
			want:   false,
		},
		{
			name:   "empty text",
			text:   "",
			budget: 10,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimator.WithinBudget(tt.text, tt.budget)

			if got != tt.want {
				t.Errorf("WithinBudget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEstimator_TruncateToFit(t *testing.T) {
	estimator := NewEstimator()

	text := "This is a longer text that needs to be truncated to fit within a token budget."
	budget := 10

	truncated, tokens := estimator.TruncateToFit(text, budget)

	if tokens > budget {
		t.Errorf("TruncateToFit() tokens = %v, want <= %v", tokens, budget)
	}

	if len(truncated) > len(text) {
		t.Errorf("TruncateToFit() truncated length > original length")
	}

	// Test text already within budget
	shortText := "Short"
	result, resultTokens := estimator.TruncateToFit(shortText, 100)

	if result != shortText {
		t.Errorf("TruncateToFit() should not modify text within budget")
	}

	if resultTokens > 100 {
		t.Errorf("TruncateToFit() tokens = %v, want <= 100", resultTokens)
	}
}

func TestEstimator_EstimateMultiple(t *testing.T) {
	estimator := NewEstimator()

	texts := []string{
		"First text",
		"Second text",
		"Third text",
	}

	total := estimator.EstimateMultiple(texts...)

	if total == 0 {
		t.Error("EstimateMultiple() returned 0")
	}

	// Total should be approximately sum of individual estimates
	sum := 0
	for _, text := range texts {
		sum += estimator.Estimate(text)
	}

	if total != sum {
		t.Errorf("EstimateMultiple() = %v, want %v", total, sum)
	}
}

func TestNewEstimatorWithRatio(t *testing.T) {
	// Test with custom ratio
	estimator := NewEstimatorWithRatio(3.0)

	text := "Test text"
	tokens := estimator.Estimate(text)

	// With ratio of 3, should get more tokens than default ratio of 4
	defaultEstimator := NewEstimator()
	defaultTokens := defaultEstimator.Estimate(text)

	if tokens <= defaultTokens {
		t.Errorf("Custom ratio should produce more tokens, got %v vs %v", tokens, defaultTokens)
	}

	// Test with invalid ratio (should use default)
	badEstimator := NewEstimatorWithRatio(-1)
	badTokens := badEstimator.Estimate(text)

	if badTokens == 0 {
		t.Error("Estimator with invalid ratio should still work")
	}
}
