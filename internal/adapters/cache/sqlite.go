// Package cache provides cache adapters for response caching.
package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// SQLiteCache implements CachePort using SQLite for persistent caching.
type SQLiteCache struct {
	db      *sql.DB
	maxSize int64 // Maximum cache size in bytes

	// Statistics (in addition to DB stats)
	hitCount  int64
	missCount int64
}

// NewSQLiteCache creates a new SQLite-backed cache.
func NewSQLiteCache(db *sql.DB, maxSize int64) *SQLiteCache {
	return &SQLiteCache{
		db:      db,
		maxSize: maxSize,
	}
}

// Get retrieves an item from cache.
func (s *SQLiteCache) Get(ctx context.Context, key string) (any, bool) {
	entry, found := s.GetEntry(ctx, key)
	if !found {
		return nil, false
	}
	return entry.Value, true
}

// GetEntry retrieves the full cache entry including metadata.
func (s *SQLiteCache) GetEntry(ctx context.Context, key string) (*ports.CacheEntry, bool) {
	row := s.db.QueryRowContext(ctx, `
		SELECT key, fingerprint, model_id, response_content, input_tokens, output_tokens,
			   finish_reason, model_used, duration_ns, size_bytes, hit_count,
			   ttl_seconds, created_at, expires_at, last_accessed_at
		FROM response_cache
		WHERE key = ? AND expires_at > datetime('now')
	`, key)

	var entry ports.CacheEntry
	var fingerprint, modelID, responseContent string
	var finishReason, modelUsed sql.NullString
	var inputTokens, outputTokens, durationNs, sizeBytes, hitCount int64
	var ttlSeconds int64
	var createdAt, expiresAt, lastAccessedAt time.Time

	err := row.Scan(
		&entry.Key, &fingerprint, &modelID, &responseContent,
		&inputTokens, &outputTokens, &finishReason, &modelUsed,
		&durationNs, &sizeBytes, &hitCount, &ttlSeconds,
		&createdAt, &expiresAt, &lastAccessedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			atomic.AddInt64(&s.missCount, 1)
			s.incrementStat(ctx, "miss_count")
		}
		return nil, false
	}

	atomic.AddInt64(&s.hitCount, 1)

	// Update hit count and last accessed time
	_, _ = s.db.ExecContext(ctx, `
		UPDATE response_cache
		SET hit_count = hit_count + 1, last_accessed_at = datetime('now')
		WHERE key = ?
	`, key)

	// Update stats
	s.incrementStat(ctx, "hit_count")
	s.incrementStat(ctx, "input_tokens_saved")
	s.incrementStatBy(ctx, "output_tokens_saved", outputTokens)

	// Reconstruct the CompletionResponse
	response := &ports.CompletionResponse{
		Content:      responseContent,
		InputTokens:  int(inputTokens),
		OutputTokens: int(outputTokens),
		FinishReason: finishReason.String,
		ModelUsed:    modelUsed.String,
		Duration:     time.Duration(durationNs),
	}

	entry.Value = response
	entry.PromptHash = fingerprint
	entry.ModelID = modelID
	entry.Size = sizeBytes
	entry.HitCount = hitCount + 1
	entry.TTL = time.Duration(ttlSeconds) * time.Second
	entry.CreatedAt = createdAt
	entry.ExpiresAt = expiresAt

	return &entry, true
}

// Set stores an item in cache with the specified TTL.
func (s *SQLiteCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	now := time.Now()
	entry := &ports.CacheEntry{
		Key:       key,
		Value:     value,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		TTL:       ttl,
		HitCount:  0,
	}

	return s.SetWithMetadata(ctx, entry)
}

// SetWithMetadata stores an item with additional metadata for tracking.
func (s *SQLiteCache) SetWithMetadata(ctx context.Context, entry *ports.CacheEntry) error {
	// Serialize value for storage
	data, err := json.Marshal(entry.Value)
	if err != nil {
		return err
	}

	entry.Size = int64(len(data))

	// Check if we need to evict entries
	if s.maxSize > 0 {
		currentSize, _ := s.Size(ctx)
		for currentSize+entry.Size > s.maxSize {
			evicted, err := s.evictOldest(ctx)
			if err != nil || !evicted {
				break
			}
			currentSize, _ = s.Size(ctx)
		}
	}

	// Try to extract CompletionResponse details
	var responseContent, finishReason, modelUsed string
	var inputTokens, outputTokens int
	var durationNs int64

	if resp, ok := entry.Value.(*ports.CompletionResponse); ok {
		responseContent = resp.Content
		finishReason = resp.FinishReason
		modelUsed = resp.ModelUsed
		inputTokens = resp.InputTokens
		outputTokens = resp.OutputTokens
		durationNs = int64(resp.Duration)
	} else {
		// Store as JSON
		responseContent = string(data)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO response_cache
		(key, fingerprint, model_id, response_content, input_tokens, output_tokens,
		 finish_reason, model_used, duration_ns, size_bytes, hit_count,
		 ttl_seconds, created_at, expires_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.Key, entry.PromptHash, entry.ModelID, responseContent,
		inputTokens, outputTokens, finishReason, modelUsed,
		durationNs, entry.Size, entry.HitCount,
		int64(entry.TTL.Seconds()), entry.CreatedAt, entry.ExpiresAt, time.Now(),
	)

	return err
}

// evictOldest removes the oldest entry from the cache.
func (s *SQLiteCache) evictOldest(ctx context.Context) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM response_cache
		WHERE key = (
			SELECT key FROM response_cache
			ORDER BY last_accessed_at ASC
			LIMIT 1
		)
	`)
	if err != nil {
		return false, err
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		s.incrementStat(ctx, "eviction_count")
		return true, nil
	}

	return false, nil
}

// Delete removes an item from cache.
func (s *SQLiteCache) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM response_cache WHERE key = ?`, key)
	return err
}

// Clear removes all items from cache.
func (s *SQLiteCache) Clear(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM response_cache`)
	if err != nil {
		return err
	}

	// Reset stats
	_, err = s.db.ExecContext(ctx, `
		UPDATE cache_stats
		SET stat_value = 0, updated_at = datetime('now')
	`)
	return err
}

// Has checks if a key exists in cache without retrieving it.
func (s *SQLiteCache) Has(ctx context.Context, key string) bool {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM response_cache
		WHERE key = ? AND expires_at > datetime('now')
	`, key).Scan(&count)

	return err == nil && count > 0
}

// Stats returns cache statistics.
func (s *SQLiteCache) Stats(ctx context.Context) (*ports.CacheStats, error) {
	stats := &ports.CacheStats{}

	// Get entry count and total size
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(size_bytes), 0)
		FROM response_cache
		WHERE expires_at > datetime('now')
	`).Scan(&stats.TotalEntries, &stats.TotalSize)
	if err != nil {
		return nil, err
	}

	// Get stats from cache_stats table
	rows, err := s.db.QueryContext(ctx, `
		SELECT stat_type, stat_value FROM cache_stats WHERE model_id IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statType string
		var statValue int64
		if err := rows.Scan(&statType, &statValue); err != nil {
			continue
		}
		switch statType {
		case "hit_count":
			stats.HitCount = statValue
		case "miss_count":
			stats.MissCount = statValue
		case "eviction_count":
			stats.EvictionCount = statValue
		case "expired_count":
			stats.ExpiredCount = statValue
		case "input_tokens_saved":
			stats.TokensSaved += statValue
		case "output_tokens_saved":
			stats.TokensSaved += statValue
		}
	}

	if stats.HitCount+stats.MissCount > 0 {
		stats.HitRate = float64(stats.HitCount) / float64(stats.HitCount+stats.MissCount) * 100
	}

	// Get oldest and newest entries
	_ = s.db.QueryRowContext(ctx, `
		SELECT MIN(created_at), MAX(created_at), AVG(ttl_seconds)
		FROM response_cache
		WHERE expires_at > datetime('now')
	`).Scan(&stats.OldestEntry, &stats.NewestEntry, &stats.AvgTTL)

	return stats, nil
}

// Cleanup removes expired entries. Returns number of entries removed.
func (s *SQLiteCache) Cleanup(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM response_cache WHERE expires_at <= datetime('now')
	`)
	if err != nil {
		return 0, err
	}

	removed, _ := result.RowsAffected()
	if removed > 0 {
		s.incrementStatBy(ctx, "expired_count", removed)
	}

	return removed, nil
}

// Keys returns all keys matching a pattern (empty pattern = all keys).
func (s *SQLiteCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key FROM response_cache WHERE expires_at > datetime('now')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	var re *regexp.Regexp

	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			continue
		}
		if re == nil || re.MatchString(key) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Size returns the total size of the cache in bytes.
func (s *SQLiteCache) Size(ctx context.Context) (int64, error) {
	var size int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(size_bytes), 0) FROM response_cache
		WHERE expires_at > datetime('now')
	`).Scan(&size)
	return size, err
}

// incrementStat increments a stat in the cache_stats table.
func (s *SQLiteCache) incrementStat(ctx context.Context, statType string) {
	s.incrementStatBy(ctx, statType, 1)
}

// incrementStatBy increments a stat by a specific amount.
func (s *SQLiteCache) incrementStatBy(ctx context.Context, statType string, amount int64) {
	_, _ = s.db.ExecContext(ctx, `
		UPDATE cache_stats
		SET stat_value = stat_value + ?, updated_at = datetime('now')
		WHERE stat_type = ? AND model_id IS NULL
	`, amount, statType)
}

// Ensure SQLiteCache implements CachePort
var _ ports.CachePort = (*SQLiteCache)(nil)
