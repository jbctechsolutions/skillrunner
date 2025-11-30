package router

import (
	"fmt"
	"sync"
	"time"
)

// Metrics tracks router operation metrics
type Metrics struct {
	mu        sync.RWMutex
	attempts  map[string]int64
	successes map[string]int64
	failures  map[string]int64
	errors    map[string]int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		attempts:  make(map[string]int64),
		successes: make(map[string]int64),
		failures:  make(map[string]int64),
		errors:    make(map[string]int64),
	}
}

// RecordAttempt records an attempt for a provider
func (m *Metrics) RecordAttempt(provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts[provider]++
}

// RecordSuccess records a success for a provider
func (m *Metrics) RecordSuccess(provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successes[provider]++
}

// RecordFailure records a failure for a provider
func (m *Metrics) RecordFailure(provider string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failures[provider]++
}

// RecordError records a generic error
func (m *Metrics) RecordError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[errorType]++
}

// GetAttempts returns the number of attempts for a provider
func (m *Metrics) GetAttempts(provider string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.attempts[provider]
}

// GetSuccesses returns the number of successes for a provider
func (m *Metrics) GetSuccesses(provider string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.successes[provider]
}

// GetFailures returns the number of failures for a provider
func (m *Metrics) GetFailures(provider string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failures[provider]
}

// GetErrors returns the number of occurrences of an error type
func (m *Metrics) GetErrors(errorType string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errors[errorType]
}

// GetSuccessRate returns the success rate for a provider (0.0 to 1.0)
func (m *Metrics) GetSuccessRate(provider string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	attempts := m.attempts[provider]
	if attempts == 0 {
		return 0.0
	}

	successes := m.successes[provider]
	return float64(successes) / float64(attempts)
}

// GetTotalAttempts returns the total number of attempts across all providers
func (m *Metrics) GetTotalAttempts() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, count := range m.attempts {
		total += count
	}
	return total
}

// GetTotalSuccesses returns the total number of successes across all providers
func (m *Metrics) GetTotalSuccesses() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, count := range m.successes {
		total += count
	}
	return total
}

// GetTotalFailures returns the total number of failures across all providers
func (m *Metrics) GetTotalFailures() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, count := range m.failures {
		total += count
	}
	return total
}

// GetTotalErrors returns the total number of errors across all error types
func (m *Metrics) GetTotalErrors() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, count := range m.errors {
		total += count
	}
	return total
}

// Reset clears all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.attempts = make(map[string]int64)
	m.successes = make(map[string]int64)
	m.failures = make(map[string]int64)
	m.errors = make(map[string]int64)
}

// ProviderStats represents statistics for a single provider
type ProviderStats struct {
	Attempts    int64   `json:"attempts"`
	Successes   int64   `json:"successes"`
	Failures    int64   `json:"failures"`
	SuccessRate float64 `json:"success_rate"`
}

// MetricsSnapshot represents a snapshot of all metrics at a point in time
type MetricsSnapshot struct {
	Timestamp      time.Time                `json:"timestamp"`
	TotalAttempts  int64                    `json:"total_attempts"`
	TotalSuccesses int64                    `json:"total_successes"`
	TotalFailures  int64                    `json:"total_failures"`
	TotalErrors    int64                    `json:"total_errors"`
	ProviderStats  map[string]ProviderStats `json:"provider_stats"`
}

// GetSnapshot returns a snapshot of all metrics
func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := &MetricsSnapshot{
		Timestamp:     time.Now(),
		ProviderStats: make(map[string]ProviderStats),
	}

	// Collect all provider names
	providers := make(map[string]bool)
	for provider := range m.attempts {
		providers[provider] = true
	}
	for provider := range m.successes {
		providers[provider] = true
	}
	for provider := range m.failures {
		providers[provider] = true
	}

	// Build provider stats
	for provider := range providers {
		attempts := m.attempts[provider]
		successes := m.successes[provider]
		failures := m.failures[provider]

		var successRate float64
		if attempts > 0 {
			successRate = float64(successes) / float64(attempts)
		}

		snapshot.ProviderStats[provider] = ProviderStats{
			Attempts:    attempts,
			Successes:   successes,
			Failures:    failures,
			SuccessRate: successRate,
		}

		snapshot.TotalAttempts += attempts
		snapshot.TotalSuccesses += successes
		snapshot.TotalFailures += failures
	}

	// Add error totals
	for _, count := range m.errors {
		snapshot.TotalErrors += count
	}

	return snapshot
}

// ProviderError represents an error from a provider
type ProviderError struct {
	Provider string
	Message  string
}

// NewProviderError creates a new provider error
func NewProviderError(provider, message string) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Message:  message,
	}
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s: %s", e.Provider, e.Message)
}
