package budget

import "fmt"

// AlertLevel describes how urgent a budget alert is.
type AlertLevel int

const (
	AlertNone  AlertLevel = iota
	AlertAt50             // 50% of limit consumed
	AlertAt80             // 80% of limit consumed (warning)
	AlertAt90             // 90% of limit consumed (critical)
	AlertAt100            // limit reached / exceeded
)

// Alert holds a single budget alert message.
type Alert struct {
	Level   AlertLevel
	Period  string  // "daily" or "monthly"
	Spend   float64 // current spend
	Limit   float64 // configured limit
	Percent float64 // spend / limit * 100
}

// Message returns a human-readable description of the alert.
func (a Alert) Message() string {
	switch a.Level {
	case AlertAt50:
		return fmt.Sprintf("%s budget at 50%% ($%.2f / $%.2f)", a.Period, a.Spend, a.Limit)
	case AlertAt80:
		return fmt.Sprintf("%s budget at 80%% ($%.2f / $%.2f) — approaching limit", a.Period, a.Spend, a.Limit)
	case AlertAt90:
		return fmt.Sprintf("%s budget at 90%% ($%.2f / $%.2f) — critical", a.Period, a.Spend, a.Limit)
	case AlertAt100:
		return fmt.Sprintf("%s budget limit reached ($%.2f / $%.2f)", a.Period, a.Spend, a.Limit)
	default:
		return ""
	}
}

// IsError returns true for alerts that should be presented as errors (90%+).
func (a Alert) IsError() bool { return a.Level >= AlertAt90 }

// CheckAlerts evaluates current usage against limits and returns any active alerts.
// It checks both daily and monthly limits.
func CheckAlerts(limits Limits, usage Usage) []Alert {
	var alerts []Alert

	if limits.DailyLimit > 0 {
		if a, ok := checkPeriod("daily", usage.DailySpend, limits.DailyLimit); ok {
			alerts = append(alerts, a)
		}
	}
	if limits.MonthlyLimit > 0 {
		if a, ok := checkPeriod("monthly", usage.MonthlySpend, limits.MonthlyLimit); ok {
			alerts = append(alerts, a)
		}
	}

	return alerts
}

// CostEstimate holds a pre-execution spend estimate for a workflow.
type CostEstimate struct {
	EstimatedUSD    float64
	CheapSavingsUSD float64 // potential saving by switching to cheap profile
	CheapSavingsPct float64 // percentage saving
}

// EstimateCost produces a rough pre-execution cost estimate based on input size.
// Uses a simple heuristic: ~$0.003 per 1 000 input chars at balanced tier.
func EstimateCost(inputLen int, profile string) CostEstimate {
	const (
		balancedRatePerKChar = 0.003
		cheapRatePerKChar    = 0.0003 // ~10x cheaper
	)

	kchars := float64(inputLen) / 1000.0

	var rate float64
	switch profile {
	case "cheap":
		rate = cheapRatePerKChar
	case "premium":
		rate = balancedRatePerKChar * 5
	default:
		rate = balancedRatePerKChar
	}

	est := kchars * rate
	cheapEst := kchars * cheapRatePerKChar
	saving := est - cheapEst
	var pct float64
	if est > 0 {
		pct = saving / est * 100
	}

	return CostEstimate{
		EstimatedUSD:    est,
		CheapSavingsUSD: saving,
		CheapSavingsPct: pct,
	}
}

// checkPeriod returns an Alert for one budget period (daily or monthly).
func checkPeriod(period string, spend, limit float64) (Alert, bool) {
	if limit <= 0 {
		return Alert{}, false
	}
	pct := spend / limit * 100
	var level AlertLevel
	switch {
	case pct >= 100:
		level = AlertAt100
	case pct >= 90:
		level = AlertAt90
	case pct >= 80:
		level = AlertAt80
	case pct >= 50:
		level = AlertAt50
	default:
		return Alert{}, false
	}
	return Alert{Level: level, Period: period, Spend: spend, Limit: limit, Percent: pct}, true
}
