package router

import (
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()

	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}

	if m.attempts == nil {
		t.Error("attempts map not initialized")
	}

	if m.successes == nil {
		t.Error("successes map not initialized")
	}

	if m.failures == nil {
		t.Error("failures map not initialized")
	}

	if m.errors == nil {
		t.Error("errors map not initialized")
	}
}

func TestMetrics_RecordAttempt(t *testing.T) {
	m := NewMetrics()

	m.RecordAttempt("provider1")
	m.RecordAttempt("provider2")
	m.RecordAttempt("provider1")

	if m.attempts["provider1"] != 2 {
		t.Errorf("Expected 2 attempts for provider1, got %d", m.attempts["provider1"])
	}

	if m.attempts["provider2"] != 1 {
		t.Errorf("Expected 1 attempt for provider2, got %d", m.attempts["provider2"])
	}
}

func TestMetrics_RecordSuccess(t *testing.T) {
	m := NewMetrics()

	m.RecordSuccess("provider1")
	m.RecordSuccess("provider2")
	m.RecordSuccess("provider1")

	if m.successes["provider1"] != 2 {
		t.Errorf("Expected 2 successes for provider1, got %d", m.successes["provider1"])
	}

	if m.successes["provider2"] != 1 {
		t.Errorf("Expected 1 success for provider2, got %d", m.successes["provider2"])
	}
}

func TestMetrics_RecordFailure(t *testing.T) {
	m := NewMetrics()

	err1 := NewProviderError("provider1", "error1")
	err2 := NewProviderError("provider2", "error2")

	m.RecordFailure("provider1", err1)
	m.RecordFailure("provider2", err2)
	m.RecordFailure("provider1", err1)

	if m.failures["provider1"] != 2 {
		t.Errorf("Expected 2 failures for provider1, got %d", m.failures["provider1"])
	}

	if m.failures["provider2"] != 1 {
		t.Errorf("Expected 1 failure for provider2, got %d", m.failures["provider2"])
	}
}

func TestMetrics_RecordError(t *testing.T) {
	m := NewMetrics()

	m.RecordError("error1")
	m.RecordError("error2")
	m.RecordError("error1")

	if m.errors["error1"] != 2 {
		t.Errorf("Expected 2 occurrences of error1, got %d", m.errors["error1"])
	}

	if m.errors["error2"] != 1 {
		t.Errorf("Expected 1 occurrence of error2, got %d", m.errors["error2"])
	}
}

func TestMetrics_GetAttempts(t *testing.T) {
	m := NewMetrics()

	m.RecordAttempt("provider1")
	m.RecordAttempt("provider2")
	m.RecordAttempt("provider1")

	attempts := m.GetAttempts("provider1")
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	attempts = m.GetAttempts("nonexistent")
	if attempts != 0 {
		t.Errorf("Expected 0 attempts for nonexistent provider, got %d", attempts)
	}
}

func TestMetrics_GetSuccesses(t *testing.T) {
	m := NewMetrics()

	m.RecordSuccess("provider1")
	m.RecordSuccess("provider2")
	m.RecordSuccess("provider1")

	successes := m.GetSuccesses("provider1")
	if successes != 2 {
		t.Errorf("Expected 2 successes, got %d", successes)
	}

	successes = m.GetSuccesses("nonexistent")
	if successes != 0 {
		t.Errorf("Expected 0 successes for nonexistent provider, got %d", successes)
	}
}

func TestMetrics_GetFailures(t *testing.T) {
	m := NewMetrics()

	err := NewProviderError("provider1", "test error")
	m.RecordFailure("provider1", err)
	m.RecordFailure("provider2", err)
	m.RecordFailure("provider1", err)

	failures := m.GetFailures("provider1")
	if failures != 2 {
		t.Errorf("Expected 2 failures, got %d", failures)
	}

	failures = m.GetFailures("nonexistent")
	if failures != 0 {
		t.Errorf("Expected 0 failures for nonexistent provider, got %d", failures)
	}
}

func TestMetrics_GetErrors(t *testing.T) {
	m := NewMetrics()

	m.RecordError("error1")
	m.RecordError("error2")
	m.RecordError("error1")

	errors := m.GetErrors("error1")
	if errors != 2 {
		t.Errorf("Expected 2 occurrences of error1, got %d", errors)
	}

	errors = m.GetErrors("nonexistent")
	if errors != 0 {
		t.Errorf("Expected 0 occurrences for nonexistent error, got %d", errors)
	}
}

func TestMetrics_GetSuccessRate(t *testing.T) {
	m := NewMetrics()

	// No attempts yet
	rate := m.GetSuccessRate("provider1")
	if rate != 0.0 {
		t.Errorf("Expected 0.0 success rate for no attempts, got %f", rate)
	}

	// 2 successes, 1 failure = 66.67%
	m.RecordAttempt("provider1")
	m.RecordSuccess("provider1")
	m.RecordAttempt("provider1")
	m.RecordSuccess("provider1")
	m.RecordAttempt("provider1")
	m.RecordFailure("provider1", NewProviderError("provider1", "error"))

	rate = m.GetSuccessRate("provider1")
	expected := 2.0 / 3.0
	if rate != expected {
		t.Errorf("Expected success rate %f, got %f", expected, rate)
	}

	// All successes
	m2 := NewMetrics()
	m2.RecordAttempt("provider2")
	m2.RecordSuccess("provider2")
	m2.RecordAttempt("provider2")
	m2.RecordSuccess("provider2")

	rate = m2.GetSuccessRate("provider2")
	if rate != 1.0 {
		t.Errorf("Expected 1.0 success rate for all successes, got %f", rate)
	}

	// All failures
	m3 := NewMetrics()
	m3.RecordAttempt("provider3")
	m3.RecordFailure("provider3", NewProviderError("provider3", "error"))
	m3.RecordAttempt("provider3")
	m3.RecordFailure("provider3", NewProviderError("provider3", "error"))

	rate = m3.GetSuccessRate("provider3")
	if rate != 0.0 {
		t.Errorf("Expected 0.0 success rate for all failures, got %f", rate)
	}
}

func TestMetrics_GetTotalAttempts(t *testing.T) {
	m := NewMetrics()

	m.RecordAttempt("provider1")
	m.RecordAttempt("provider2")
	m.RecordAttempt("provider1")
	m.RecordAttempt("provider3")

	total := m.GetTotalAttempts()
	if total != 4 {
		t.Errorf("Expected 4 total attempts, got %d", total)
	}
}

func TestMetrics_GetTotalSuccesses(t *testing.T) {
	m := NewMetrics()

	m.RecordSuccess("provider1")
	m.RecordSuccess("provider2")
	m.RecordSuccess("provider1")

	total := m.GetTotalSuccesses()
	if total != 3 {
		t.Errorf("Expected 3 total successes, got %d", total)
	}
}

func TestMetrics_GetTotalFailures(t *testing.T) {
	m := NewMetrics()

	err := NewProviderError("provider1", "error")
	m.RecordFailure("provider1", err)
	m.RecordFailure("provider2", err)
	m.RecordFailure("provider1", err)

	total := m.GetTotalFailures()
	if total != 3 {
		t.Errorf("Expected 3 total failures, got %d", total)
	}
}

func TestMetrics_GetTotalErrors(t *testing.T) {
	m := NewMetrics()

	m.RecordError("error1")
	m.RecordError("error2")
	m.RecordError("error1")

	total := m.GetTotalErrors()
	if total != 3 {
		t.Errorf("Expected 3 total errors, got %d", total)
	}
}

func TestMetrics_Reset(t *testing.T) {
	m := NewMetrics()

	m.RecordAttempt("provider1")
	m.RecordSuccess("provider1")
	m.RecordFailure("provider1", NewProviderError("provider1", "error"))
	m.RecordError("error1")

	m.Reset()

	if m.GetTotalAttempts() != 0 {
		t.Error("Expected 0 attempts after reset")
	}

	if m.GetTotalSuccesses() != 0 {
		t.Error("Expected 0 successes after reset")
	}

	if m.GetTotalFailures() != 0 {
		t.Error("Expected 0 failures after reset")
	}

	if m.GetTotalErrors() != 0 {
		t.Error("Expected 0 errors after reset")
	}
}

func TestMetrics_GetSnapshot(t *testing.T) {
	m := NewMetrics()

	m.RecordAttempt("provider1")
	m.RecordSuccess("provider1")
	m.RecordAttempt("provider2")
	m.RecordFailure("provider2", NewProviderError("provider2", "error"))
	m.RecordError("error1")

	snapshot := m.GetSnapshot()

	if snapshot.TotalAttempts != 2 {
		t.Errorf("Expected 2 total attempts in snapshot, got %d", snapshot.TotalAttempts)
	}

	if snapshot.TotalSuccesses != 1 {
		t.Errorf("Expected 1 total success in snapshot, got %d", snapshot.TotalSuccesses)
	}

	if snapshot.TotalFailures != 1 {
		t.Errorf("Expected 1 total failure in snapshot, got %d", snapshot.TotalFailures)
	}

	if snapshot.TotalErrors != 1 {
		t.Errorf("Expected 1 total error in snapshot, got %d", snapshot.TotalErrors)
	}

	if len(snapshot.ProviderStats) != 2 {
		t.Errorf("Expected 2 providers in snapshot, got %d", len(snapshot.ProviderStats))
	}

	// Check provider1 stats
	stats1, ok := snapshot.ProviderStats["provider1"]
	if !ok {
		t.Fatal("provider1 not found in snapshot")
	}
	if stats1.Attempts != 1 {
		t.Errorf("Expected 1 attempt for provider1, got %d", stats1.Attempts)
	}
	if stats1.Successes != 1 {
		t.Errorf("Expected 1 success for provider1, got %d", stats1.Successes)
	}
	if stats1.Failures != 0 {
		t.Errorf("Expected 0 failures for provider1, got %d", stats1.Failures)
	}
	if stats1.SuccessRate != 1.0 {
		t.Errorf("Expected 1.0 success rate for provider1, got %f", stats1.SuccessRate)
	}

	// Check provider2 stats
	stats2, ok := snapshot.ProviderStats["provider2"]
	if !ok {
		t.Fatal("provider2 not found in snapshot")
	}
	if stats2.Attempts != 1 {
		t.Errorf("Expected 1 attempt for provider2, got %d", stats2.Attempts)
	}
	if stats2.Successes != 0 {
		t.Errorf("Expected 0 successes for provider2, got %d", stats2.Successes)
	}
	if stats2.Failures != 1 {
		t.Errorf("Expected 1 failure for provider2, got %d", stats2.Failures)
	}
	if stats2.SuccessRate != 0.0 {
		t.Errorf("Expected 0.0 success rate for provider2, got %f", stats2.SuccessRate)
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	m := NewMetrics()

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				providerName := "provider" + string(rune(id))
				m.RecordAttempt(providerName)
				m.RecordSuccess(providerName)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify totals
	totalAttempts := m.GetTotalAttempts()
	if totalAttempts != 1000 {
		t.Errorf("Expected 1000 total attempts, got %d", totalAttempts)
	}

	totalSuccesses := m.GetTotalSuccesses()
	if totalSuccesses != 1000 {
		t.Errorf("Expected 1000 total successes, got %d", totalSuccesses)
	}
}

func TestProviderError(t *testing.T) {
	err := NewProviderError("provider1", "test error")

	if err.Provider != "provider1" {
		t.Errorf("Expected provider 'provider1', got '%s'", err.Provider)
	}

	if err.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", err.Message)
	}

	if err.Error() == "" {
		t.Error("Error() should return a non-empty string")
	}

	// Check that error message contains provider and message
	errorStr := err.Error()
	if errorStr == "" {
		t.Error("Error string should not be empty")
	}
}

func TestProviderStats(t *testing.T) {
	stats := ProviderStats{
		Attempts:    10,
		Successes:   7,
		Failures:    3,
		SuccessRate: 0.7,
	}

	if stats.Attempts != 10 {
		t.Errorf("Expected 10 attempts, got %d", stats.Attempts)
	}

	if stats.Successes != 7 {
		t.Errorf("Expected 7 successes, got %d", stats.Successes)
	}

	if stats.Failures != 3 {
		t.Errorf("Expected 3 failures, got %d", stats.Failures)
	}

	if stats.SuccessRate != 0.7 {
		t.Errorf("Expected 0.7 success rate, got %f", stats.SuccessRate)
	}
}

func TestMetricsSnapshot_Timestamp(t *testing.T) {
	m := NewMetrics()
	snapshot := m.GetSnapshot()

	// Timestamp should be set
	if snapshot.Timestamp.IsZero() {
		t.Error("Snapshot timestamp should not be zero")
	}

	// Timestamp should be recent (within last second)
	now := time.Now()
	diff := now.Sub(snapshot.Timestamp)
	if diff < 0 || diff > time.Second {
		t.Errorf("Snapshot timestamp should be recent, got diff %v", diff)
	}
}
