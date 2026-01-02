// Package storage provides SQLite-based storage implementations for context management.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
)

// RuleRepository implements RuleStoragePort using SQLite.
type RuleRepository struct {
	db *sql.DB
}

// NewRuleRepository creates a new rule repository.
func NewRuleRepository(db *sql.DB) ports.RuleStoragePort {
	return &RuleRepository{db: db}
}

// Save persists a rule to storage.
func (r *RuleRepository) Save(ctx context.Context, rule *domainContext.Rule) error {
	query := `
		INSERT INTO rules (id, name, content, scope, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		rule.ID(),
		rule.Name(),
		rule.Content(),
		string(rule.Scope()),
		rule.IsActive(),
		rule.CreatedAt().Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to save rule: %w", err)
	}

	return nil
}

// Get retrieves a rule by ID.
func (r *RuleRepository) Get(ctx context.Context, id string) (*domainContext.Rule, error) {
	query := `
		SELECT id, name, content, scope, is_active, created_at
		FROM rules
		WHERE id = ?
	`

	var (
		rid, name, content, scope string
		isActive                  bool
		createdAt                 string
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(&rid, &name, &content, &scope, &isActive, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	return r.scanRule(rid, name, content, scope, isActive, createdAt)
}

// GetByName retrieves a rule by name.
func (r *RuleRepository) GetByName(ctx context.Context, name string) (*domainContext.Rule, error) {
	query := `
		SELECT id, name, content, scope, is_active, created_at
		FROM rules
		WHERE name = ?
	`

	var (
		rid, rname, content, scope string
		isActive                   bool
		createdAt                  string
	)

	err := r.db.QueryRowContext(ctx, query, name).Scan(&rid, &rname, &content, &scope, &isActive, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule by name: %w", err)
	}

	return r.scanRule(rid, rname, content, scope, isActive, createdAt)
}

// List returns all rules.
func (r *RuleRepository) List(ctx context.Context) ([]*domainContext.Rule, error) {
	query := `
		SELECT id, name, content, scope, is_active, created_at
		FROM rules
		ORDER BY created_at DESC
	`

	return r.queryRules(ctx, query)
}

// ListActive retrieves all active rules.
func (r *RuleRepository) ListActive(ctx context.Context) ([]*domainContext.Rule, error) {
	query := `
		SELECT id, name, content, scope, is_active, created_at
		FROM rules
		WHERE is_active = 1
		ORDER BY created_at DESC
	`

	return r.queryRules(ctx, query)
}

// ListByScope retrieves rules for a specific scope.
func (r *RuleRepository) ListByScope(ctx context.Context, scope domainContext.RuleScope) ([]*domainContext.Rule, error) {
	query := `
		SELECT id, name, content, scope, is_active, created_at
		FROM rules
		WHERE scope = ?
		ORDER BY created_at DESC
	`

	return r.queryRules(ctx, query, string(scope))
}

// Update updates an existing rule.
func (r *RuleRepository) Update(ctx context.Context, rule *domainContext.Rule) error {
	query := `
		UPDATE rules
		SET name = ?, content = ?, scope = ?, is_active = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		rule.Name(),
		rule.Content(),
		string(rule.Scope()),
		rule.IsActive(),
		rule.ID(),
	)

	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("rule not found: %s", rule.ID())
	}

	return nil
}

// Delete removes a rule from storage.
func (r *RuleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM rules WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	return nil
}

// Exists checks if a rule exists.
func (r *RuleRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT COUNT(*) FROM rules WHERE id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check rule existence: %w", err)
	}

	return count > 0, nil
}

// queryRules executes a query and returns multiple rules.
func (r *RuleRepository) queryRules(ctx context.Context, query string, args ...any) ([]*domainContext.Rule, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []*domainContext.Rule
	for rows.Next() {
		var (
			rid, name, content, scope string
			isActive                  bool
			createdAt                 string
		)

		err := rows.Scan(&rid, &name, &content, &scope, &isActive, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		rule, err := r.scanRule(rid, name, content, scope, isActive, createdAt)
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rules, nil
}

// scanRule converts database fields to a Rule domain entity.
func (r *RuleRepository) scanRule(id, name, content, scope string, isActive bool, createdAt string) (*domainContext.Rule, error) {
	rule, err := domainContext.NewRule(id, name, content, domainContext.RuleScope(scope))
	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	if !isActive {
		rule.Deactivate()
	}

	return rule, nil
}
