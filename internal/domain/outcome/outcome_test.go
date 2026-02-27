package outcome_test

import (
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/domain/outcome"
)

func TestAnalyze_NoRecommendationsWithFewSamples(t *testing.T) {
	stats := []outcome.ProfileStats{
		{PhaseID: "p1", Profile: "cheap", SuccessRate: 0.5, TotalRuns: 3},
	}
	recs := outcome.Analyze(stats)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations with < 5 samples, got %d", len(recs))
	}
}

func TestAnalyze_UpgradeCheapLowSuccess(t *testing.T) {
	stats := []outcome.ProfileStats{
		{PhaseID: "p1", PhaseName: "Generate", Profile: "cheap", SuccessRate: 0.65, TotalRuns: 10},
	}
	recs := outcome.Analyze(stats)
	if len(recs) == 0 {
		t.Fatal("expected upgrade recommendation for cheap with 65% success")
	}
	if recs[0].SuggestedProfile != "balanced" {
		t.Errorf("expected suggestion balanced, got %s", recs[0].SuggestedProfile)
	}
}

func TestAnalyze_NoUpgradeForHighSuccess(t *testing.T) {
	stats := []outcome.ProfileStats{
		{PhaseID: "p1", Profile: "cheap", SuccessRate: 0.95, TotalRuns: 10},
	}
	recs := outcome.Analyze(stats)
	// Should not upgrade a high-success cheap profile to balanced
	for _, r := range recs {
		if r.CurrentProfile == "cheap" && r.SuggestedProfile == "balanced" {
			t.Errorf("should not recommend upgrade when success rate is already high")
		}
	}
}

func TestAnalyze_DowngradePremiumPerfectSuccess(t *testing.T) {
	stats := []outcome.ProfileStats{
		{PhaseID: "p1", PhaseName: "Review", Profile: "premium", SuccessRate: 0.98, TotalRuns: 25},
	}
	recs := outcome.Analyze(stats)
	if len(recs) == 0 {
		t.Fatal("expected downgrade recommendation for premium with perfect success")
	}
	if recs[0].SuggestedProfile != "balanced" {
		t.Errorf("expected balanced suggestion for premium downgrade, got %s", recs[0].SuggestedProfile)
	}
}

func TestAnalyze_DowngradeBalancedPerfectSuccess(t *testing.T) {
	stats := []outcome.ProfileStats{
		{PhaseID: "p1", PhaseName: "Summarize", Profile: "balanced", SuccessRate: 0.97, TotalRuns: 22},
	}
	recs := outcome.Analyze(stats)
	if len(recs) == 0 {
		t.Fatal("expected downgrade recommendation for balanced with perfect success")
	}
	if recs[0].SuggestedProfile != "cheap" {
		t.Errorf("expected cheap suggestion for balanced downgrade, got %s", recs[0].SuggestedProfile)
	}
}

func TestConfidence_GrowsWithSamples(t *testing.T) {
	// Low runs → low confidence (no recommendation)
	low := []outcome.ProfileStats{
		{PhaseID: "p1", Profile: "cheap", SuccessRate: 0.5, TotalRuns: 5},
	}
	recs := outcome.Analyze(low)
	if len(recs) == 0 {
		t.Fatal("expected at least one recommendation")
	}
	if recs[0].Confidence > 0.5 {
		t.Errorf("expected low confidence for 5 samples, got %.2f", recs[0].Confidence)
	}

	// More runs → higher confidence
	high := []outcome.ProfileStats{
		{PhaseID: "p1", Profile: "cheap", SuccessRate: 0.5, TotalRuns: 25},
	}
	recs2 := outcome.Analyze(high)
	if len(recs2) == 0 {
		t.Fatal("expected recommendation with 25 runs")
	}
	if recs2[0].Confidence <= recs[0].Confidence {
		t.Errorf("expected higher confidence with more samples: %.2f vs %.2f",
			recs2[0].Confidence, recs[0].Confidence)
	}
}
