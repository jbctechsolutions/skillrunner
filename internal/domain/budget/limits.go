// Package budget provides domain types for cost budget enforcement.
package budget

import (
	"errors"
	"time"
)

// ErrBudgetExceeded is returned when a cost limit is exceeded.
var ErrBudgetExceeded = errors.New("budget limit exceeded")

// ErrBudgetWarning is returned when spending reaches the warning threshold.
var ErrBudgetWarning = errors.New("budget warning threshold reached")

// Limits defines the configured budget thresholds.
type Limits struct {
	DailyLimit   float64 // USD per day (0 = disabled)
	MonthlyLimit float64 // USD per month (0 = disabled)
}

// NewLimits returns a Limits with the given daily and monthly caps.
func NewLimits(daily, monthly float64) Limits {
	return Limits{DailyLimit: daily, MonthlyLimit: monthly}
}

// Enabled returns true if at least one limit is configured.
func (l Limits) Enabled() bool {
	return l.DailyLimit > 0 || l.MonthlyLimit > 0
}

// Usage holds current spending for a time period.
type Usage struct {
	DailySpend   float64
	MonthlySpend float64
	AsOf         time.Time
}

// CheckLimit returns ErrBudgetExceeded if any configured limit has been exceeded,
// or ErrBudgetWarning if spending is at or above 80% of any limit.
// Returns nil if all limits are within acceptable range.
func CheckLimit(limits Limits, usage Usage, additional float64) error {
	const warningPct = 0.80

	if limits.DailyLimit > 0 {
		projected := usage.DailySpend + additional
		if projected >= limits.DailyLimit {
			return ErrBudgetExceeded
		}
		if projected >= limits.DailyLimit*warningPct {
			return ErrBudgetWarning
		}
	}
	if limits.MonthlyLimit > 0 {
		projected := usage.MonthlySpend + additional
		if projected >= limits.MonthlyLimit {
			return ErrBudgetExceeded
		}
		if projected >= limits.MonthlyLimit*warningPct {
			return ErrBudgetWarning
		}
	}
	return nil
}
