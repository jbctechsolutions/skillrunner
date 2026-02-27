package budget_test

import (
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/budget"
)

func TestCheckAlerts_NoLimits(t *testing.T) {
	alerts := budget.CheckAlerts(budget.Limits{}, budget.Usage{AsOf: time.Now()})
	if len(alerts) != 0 {
		t.Errorf("expected no alerts with no limits, got %d", len(alerts))
	}
}

func TestCheckAlerts_Below50(t *testing.T) {
	limits := budget.NewLimits(10.00, 0)
	usage := budget.Usage{DailySpend: 4.00, AsOf: time.Now()}
	alerts := budget.CheckAlerts(limits, usage)
	if len(alerts) != 0 {
		t.Errorf("expected no alerts at 40%%, got %d", len(alerts))
	}
}

func TestCheckAlerts_At50(t *testing.T) {
	limits := budget.NewLimits(10.00, 0)
	usage := budget.Usage{DailySpend: 5.00, AsOf: time.Now()}
	alerts := budget.CheckAlerts(limits, usage)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert at 50%%, got %d", len(alerts))
	}
	if alerts[0].Level != budget.AlertAt50 {
		t.Errorf("expected AlertAt50, got %v", alerts[0].Level)
	}
}

func TestCheckAlerts_At80(t *testing.T) {
	limits := budget.NewLimits(10.00, 0)
	usage := budget.Usage{DailySpend: 8.50, AsOf: time.Now()}
	alerts := budget.CheckAlerts(limits, usage)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert at 85%%, got %d", len(alerts))
	}
	if alerts[0].Level != budget.AlertAt80 {
		t.Errorf("expected AlertAt80, got %v", alerts[0].Level)
	}
}

func TestCheckAlerts_At90(t *testing.T) {
	limits := budget.NewLimits(10.00, 0)
	usage := budget.Usage{DailySpend: 9.20, AsOf: time.Now()}
	alerts := budget.CheckAlerts(limits, usage)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert at 92%%, got %d", len(alerts))
	}
	if alerts[0].Level != budget.AlertAt90 {
		t.Errorf("expected AlertAt90, got %v", alerts[0].Level)
	}
	if !alerts[0].IsError() {
		t.Error("90%+ alert should be IsError() = true")
	}
}

func TestCheckAlerts_At100(t *testing.T) {
	limits := budget.NewLimits(10.00, 0)
	usage := budget.Usage{DailySpend: 10.50, AsOf: time.Now()}
	alerts := budget.CheckAlerts(limits, usage)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert at 105%%, got %d", len(alerts))
	}
	if alerts[0].Level != budget.AlertAt100 {
		t.Errorf("expected AlertAt100, got %v", alerts[0].Level)
	}
}

func TestCheckAlerts_BothPeriods(t *testing.T) {
	limits := budget.NewLimits(10.00, 100.00)
	usage := budget.Usage{
		DailySpend:   8.50, // 85% of daily
		MonthlySpend: 55.0, // 55% of monthly
		AsOf:         time.Now(),
	}
	alerts := budget.CheckAlerts(limits, usage)
	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts (daily+monthly), got %d", len(alerts))
	}
}

func TestAlert_Message(t *testing.T) {
	a := budget.Alert{Level: budget.AlertAt80, Period: "daily", Spend: 8.0, Limit: 10.0, Percent: 80}
	msg := a.Message()
	if msg == "" {
		t.Error("alert message should not be empty")
	}
}

func TestEstimateCost_ReturnsPositive(t *testing.T) {
	est := budget.EstimateCost(1000, "balanced")
	if est.EstimatedUSD <= 0 {
		t.Error("estimate should be positive for non-empty input")
	}
}

func TestEstimateCost_CheapCheaperThanPremium(t *testing.T) {
	cheap := budget.EstimateCost(5000, "cheap")
	premium := budget.EstimateCost(5000, "premium")
	if cheap.EstimatedUSD >= premium.EstimatedUSD {
		t.Errorf("cheap ($%.4f) should cost less than premium ($%.4f)", cheap.EstimatedUSD, premium.EstimatedUSD)
	}
}

func TestEstimateCost_SavingsCalculated(t *testing.T) {
	est := budget.EstimateCost(5000, "premium")
	if est.CheapSavingsUSD <= 0 {
		t.Error("premium profile should show positive cheap savings")
	}
	if est.CheapSavingsPct <= 0 || est.CheapSavingsPct > 100 {
		t.Errorf("cheap savings pct out of range: %.1f", est.CheapSavingsPct)
	}
}
