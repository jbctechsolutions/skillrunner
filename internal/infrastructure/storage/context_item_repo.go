// Package storage provides SQLite-based storage implementations for context management.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
)

// ContextItemRepository implements ContextItemStoragePort using SQLite.
type ContextItemRepository struct {
	db *sql.DB
}

// NewContextItemRepository creates a new context item repository.
func NewContextItemRepository(db *sql.DB) ports.ContextItemStoragePort {
	return &ContextItemRepository{db: db}
}

// Save persists a context item to storage.
func (r *ContextItemRepository) Save(ctx context.Context, item *domainContext.ContextItem) error {
	tagsJSON, err := json.Marshal(item.Tags())
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO context_items (id, name, type, content, tags, token_estimate, last_used_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		item.ID(),
		item.Name(),
		string(item.Type()),
		item.Content(),
		string(tagsJSON),
		item.TokenEstimate(),
		item.LastUsedAt().Format(time.RFC3339),
		item.CreatedAt().Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to save context item: %w", err)
	}

	return nil
}

// Get retrieves a context item by ID.
func (r *ContextItemRepository) Get(ctx context.Context, id string) (*domainContext.ContextItem, error) {
	query := `
		SELECT id, name, type, content, tags, token_estimate, last_used_at, created_at
		FROM context_items
		WHERE id = ?
	`

	var (
		iid, name, itemType, content, tagsJSON string
		tokenEstimate                          int
		lastUsedAt, createdAt                  string
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&iid, &name, &itemType, &content, &tagsJSON, &tokenEstimate, &lastUsedAt, &createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("context item not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get context item: %w", err)
	}

	return r.scanItem(iid, name, itemType, content, tagsJSON, tokenEstimate, lastUsedAt, createdAt)
}

// GetByName retrieves a context item by name.
func (r *ContextItemRepository) GetByName(ctx context.Context, name string) (*domainContext.ContextItem, error) {
	query := `
		SELECT id, name, type, content, tags, token_estimate, last_used_at, created_at
		FROM context_items
		WHERE name = ?
	`

	var (
		iid, iname, itemType, content, tagsJSON string
		tokenEstimate                           int
		lastUsedAt, createdAt                   string
	)

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&iid, &iname, &itemType, &content, &tagsJSON, &tokenEstimate, &lastUsedAt, &createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("context item not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get context item by name: %w", err)
	}

	return r.scanItem(iid, iname, itemType, content, tagsJSON, tokenEstimate, lastUsedAt, createdAt)
}

// List returns all context items.
func (r *ContextItemRepository) List(ctx context.Context) ([]*domainContext.ContextItem, error) {
	query := `
		SELECT id, name, type, content, tags, token_estimate, last_used_at, created_at
		FROM context_items
		ORDER BY last_used_at DESC
	`

	return r.queryItems(ctx, query)
}

// ListByTag retrieves context items with a specific tag.
func (r *ContextItemRepository) ListByTag(ctx context.Context, tag string) ([]*domainContext.ContextItem, error) {
	// SQLite doesn't have native JSON array search, so we'll use LIKE
	query := `
		SELECT id, name, type, content, tags, token_estimate, last_used_at, created_at
		FROM context_items
		WHERE tags LIKE ?
		ORDER BY last_used_at DESC
	`

	return r.queryItems(ctx, query, "%\""+tag+"\"%")
}

// Update updates an existing context item.
func (r *ContextItemRepository) Update(ctx context.Context, item *domainContext.ContextItem) error {
	tagsJSON, err := json.Marshal(item.Tags())
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE context_items
		SET name = ?, type = ?, content = ?, tags = ?, token_estimate = ?, last_used_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		item.Name(),
		string(item.Type()),
		item.Content(),
		string(tagsJSON),
		item.TokenEstimate(),
		item.LastUsedAt().Format(time.RFC3339),
		item.ID(),
	)

	if err != nil {
		return fmt.Errorf("failed to update context item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("context item not found: %s", item.ID())
	}

	return nil
}

// Delete removes a context item from storage.
func (r *ContextItemRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM context_items WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete context item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("context item not found: %s", id)
	}

	return nil
}

// Exists checks if a context item exists.
func (r *ContextItemRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT COUNT(*) FROM context_items WHERE id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check context item existence: %w", err)
	}

	return count > 0, nil
}

// queryItems executes a query and returns multiple context items.
func (r *ContextItemRepository) queryItems(ctx context.Context, query string, args ...any) ([]*domainContext.ContextItem, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query context items: %w", err)
	}
	defer rows.Close()

	var items []*domainContext.ContextItem
	for rows.Next() {
		var (
			iid, name, itemType, content, tagsJSON string
			tokenEstimate                          int
			lastUsedAt, createdAt                  string
		)

		err := rows.Scan(&iid, &name, &itemType, &content, &tagsJSON, &tokenEstimate, &lastUsedAt, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan context item: %w", err)
		}

		item, err := r.scanItem(iid, name, itemType, content, tagsJSON, tokenEstimate, lastUsedAt, createdAt)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating context items: %w", err)
	}

	return items, nil
}

// scanItem converts database fields to a ContextItem domain entity.
func (r *ContextItemRepository) scanItem(
	id, name, itemType, content, tagsJSON string, tokenEstimate int, lastUsedAt, createdAt string,
) (*domainContext.ContextItem, error) {
	item, err := domainContext.NewContextItem(id, name, domainContext.ItemType(itemType))
	if err != nil {
		return nil, fmt.Errorf("failed to create context item: %w", err)
	}

	item.SetContent(content)
	item.SetTokenEstimate(tokenEstimate)

	// Unmarshal tags
	if tagsJSON != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
		item.SetTags(tags)
	}

	return item, nil
}
