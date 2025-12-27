package metrics

import (
	"testing"
	"time"
)

func TestTimePeriod_Duration(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	period := TimePeriod{Start: start, End: end}
	expected := 24 * time.Hour

	if period.Duration() != expected {
		t.Errorf("expected duration %v, got %v", expected, period.Duration())
	}
}

func TestNewCostSummary(t *testing.T) {
	period := TimePeriod{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
	}

	summary := NewCostSummary(period)

	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	if summary.ByProvider == nil {
		t.Error("expected initialized ByProvider map")
	}

	if summary.BySkill == nil {
		t.Error("expected initialized BySkill map")
	}

	if summary.ByModel == nil {
		t.Error("expected initialized ByModel map")
	}

	if summary.Period != period {
		t.Error("expected period to be set")
	}
}

func TestDefaultFilter(t *testing.T) {
	filter := DefaultFilter()

	if filter.StartDate.IsZero() {
		t.Error("expected StartDate to be set")
	}

	if filter.EndDate.IsZero() {
		t.Error("expected EndDate to be set")
	}

	// StartDate should be approximately 24 hours before EndDate
	diff := filter.EndDate.Sub(filter.StartDate)
	if diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("expected approximately 24 hour difference, got %v", diff)
	}

	if filter.Limit != 100 {
		t.Errorf("expected limit 100, got %d", filter.Limit)
	}
}

func TestMetricsFilter_WithMethods(t *testing.T) {
	filter := DefaultFilter()

	t.Run("WithPeriod", func(t *testing.T) {
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC)

		newFilter := filter.WithPeriod(start, end)

		if newFilter.StartDate != start {
			t.Errorf("expected StartDate %v, got %v", start, newFilter.StartDate)
		}
		if newFilter.EndDate != end {
			t.Errorf("expected EndDate %v, got %v", end, newFilter.EndDate)
		}
	})

	t.Run("WithSkill", func(t *testing.T) {
		newFilter := filter.WithSkill("code-review")

		if newFilter.SkillID != "code-review" {
			t.Errorf("expected SkillID 'code-review', got %s", newFilter.SkillID)
		}
	})

	t.Run("WithProvider", func(t *testing.T) {
		newFilter := filter.WithProvider("anthropic")

		if newFilter.Provider != "anthropic" {
			t.Errorf("expected Provider 'anthropic', got %s", newFilter.Provider)
		}
	})
}

func TestLast24Hours(t *testing.T) {
	filter := Last24Hours()

	diff := filter.EndDate.Sub(filter.StartDate)
	if diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("expected approximately 24 hour difference, got %v", diff)
	}
}

func TestLast7Days(t *testing.T) {
	filter := Last7Days()

	diff := filter.EndDate.Sub(filter.StartDate)
	expected := 7 * 24 * time.Hour
	if diff < expected-time.Hour || diff > expected+time.Hour {
		t.Errorf("expected approximately 7 day difference, got %v", diff)
	}
}

func TestLast30Days(t *testing.T) {
	filter := Last30Days()

	diff := filter.EndDate.Sub(filter.StartDate)
	expected := 30 * 24 * time.Hour
	if diff < expected-time.Hour || diff > expected+time.Hour {
		t.Errorf("expected approximately 30 day difference, got %v", diff)
	}
}

func TestExecutionRecord(t *testing.T) {
	now := time.Now()
	record := ExecutionRecord{
		ID:            "exec-123",
		SkillID:       "code-review",
		SkillName:     "Code Review",
		Status:        "completed",
		InputTokens:   1000,
		OutputTokens:  500,
		TotalCost:     0.015,
		Duration:      5 * time.Second,
		PhaseCount:    3,
		CacheHits:     1,
		CacheMisses:   2,
		PrimaryModel:  "claude-3-5-sonnet",
		StartedAt:     now.Add(-5 * time.Second),
		CompletedAt:   now,
		CorrelationID: "corr-456",
	}

	if record.ID != "exec-123" {
		t.Errorf("unexpected ID: %s", record.ID)
	}
	if record.TotalCost != 0.015 {
		t.Errorf("unexpected TotalCost: %f", record.TotalCost)
	}
}

func TestPhaseExecutionRecord(t *testing.T) {
	now := time.Now()
	record := PhaseExecutionRecord{
		ID:           "phase-exec-789",
		ExecutionID:  "exec-123",
		PhaseID:      "analysis",
		PhaseName:    "Pattern Analysis",
		Status:       "completed",
		Provider:     "anthropic",
		Model:        "claude-3-5-sonnet",
		InputTokens:  500,
		OutputTokens: 200,
		Cost:         0.005,
		Duration:     2 * time.Second,
		CacheHit:     false,
		StartedAt:    now.Add(-2 * time.Second),
		CompletedAt:  now,
	}

	if record.Provider != "anthropic" {
		t.Errorf("unexpected Provider: %s", record.Provider)
	}
	if record.CacheHit != false {
		t.Error("expected CacheHit to be false")
	}
}
