package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/budget"
)

// BudgetRepository tracks API spending in SQLite for budget enforcement.
type BudgetRepository struct {
	db *sql.DB
}

// NewBudgetRepository creates a new BudgetRepository and ensures the table exists.
func NewBudgetRepository(db *sql.DB) (*BudgetRepository, error) {
	r := &BudgetRepository{db: db}
	if err := r.initTable(); err != nil {
		return nil, fmt.Errorf("budget repository init: %w", err)
	}
	return r, nil
}

// initTable creates the budget_spend table if it doesn't exist.
func (r *BudgetRepository) initTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS budget_spend (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			amount_usd  REAL    NOT NULL,
			spent_at    TEXT    NOT NULL,
			skill_id    TEXT,
			phase_id    TEXT,
			model_id    TEXT
		)
	`)
	return err
}

// RecordSpend records a cost entry for a completed phase execution.
func (r *BudgetRepository) RecordSpend(ctx context.Context, amountUSD float64, skillID, phaseID, modelID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO budget_spend (amount_usd, spent_at, skill_id, phase_id, model_id) VALUES (?, ?, ?, ?, ?)`,
		amountUSD,
		time.Now().UTC().Format(time.RFC3339),
		skillID,
		phaseID,
		modelID,
	)
	return err
}

// GetUsage returns current daily and monthly spend as of now.
func (r *BudgetRepository) GetUsage(ctx context.Context) (budget.Usage, error) {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var usage budget.Usage
	usage.AsOf = now

	row := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount_usd), 0) FROM budget_spend WHERE spent_at >= ?`,
		startOfDay.Format(time.RFC3339),
	)
	if err := row.Scan(&usage.DailySpend); err != nil {
		return usage, fmt.Errorf("query daily spend: %w", err)
	}

	row = r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount_usd), 0) FROM budget_spend WHERE spent_at >= ?`,
		startOfMonth.Format(time.RFC3339),
	)
	if err := row.Scan(&usage.MonthlySpend); err != nil {
		return usage, fmt.Errorf("query monthly spend: %w", err)
	}

	return usage, nil
}
