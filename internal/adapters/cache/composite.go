// Package cache provides cache adapters for response caching.
package cache

import (
	"context"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// CompositeCache combines in-memory and SQLite caches in a two-tier architecture.
// Hot data is served from memory, while cold data is persisted to SQLite.
type CompositeCache struct {
	memory *MemoryCache
	sqlite *SQLiteCache
}

// NewCompositeCache creates a new composite cache with memory and SQLite tiers.
func NewCompositeCache(memory *MemoryCache, sqlite *SQLiteCache) *CompositeCache {
	return &CompositeCache{
		memory: memory,
		sqlite: sqlite,
	}
}

// Get retrieves an item from cache, checking memory first then SQLite.
func (c *CompositeCache) Get(ctx context.Context, key string) (any, bool) {
	// Check memory first
	if value, found := c.memory.Get(ctx, key); found {
		return value, true
	}

	// Check SQLite
	if value, found := c.sqlite.Get(ctx, key); found {
		// Promote to memory cache
		entry, _ := c.sqlite.GetEntry(ctx, key)
		if entry != nil {
			remainingTTL := time.Until(entry.ExpiresAt)
			if remainingTTL > 0 {
				_ = c.memory.SetWithMetadata(ctx, entry)
			}
		}
		return value, true
	}

	return nil, false
}

// GetEntry retrieves the full cache entry including metadata.
func (c *CompositeCache) GetEntry(ctx context.Context, key string) (*ports.CacheEntry, bool) {
	// Check memory first
	if entry, found := c.memory.GetEntry(ctx, key); found {
		return entry, true
	}

	// Check SQLite
	if entry, found := c.sqlite.GetEntry(ctx, key); found {
		// Promote to memory cache
		remainingTTL := time.Until(entry.ExpiresAt)
		if remainingTTL > 0 {
			_ = c.memory.SetWithMetadata(ctx, entry)
		}
		return entry, true
	}

	return nil, false
}

// Set stores an item in both memory and SQLite caches.
func (c *CompositeCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	// Set in memory first (fast)
	if err := c.memory.Set(ctx, key, value, ttl); err != nil {
		return err
	}

	// Persist to SQLite (durable)
	return c.sqlite.Set(ctx, key, value, ttl)
}

// SetWithMetadata stores an item with additional metadata for tracking.
func (c *CompositeCache) SetWithMetadata(ctx context.Context, entry *ports.CacheEntry) error {
	// Set in memory first
	if err := c.memory.SetWithMetadata(ctx, entry); err != nil {
		return err
	}

	// Persist to SQLite
	return c.sqlite.SetWithMetadata(ctx, entry)
}

// Delete removes an item from both caches.
func (c *CompositeCache) Delete(ctx context.Context, key string) error {
	// Delete from memory
	if err := c.memory.Delete(ctx, key); err != nil {
		return err
	}

	// Delete from SQLite
	return c.sqlite.Delete(ctx, key)
}

// Clear removes all items from both caches.
func (c *CompositeCache) Clear(ctx context.Context) error {
	// Clear memory
	if err := c.memory.Clear(ctx); err != nil {
		return err
	}

	// Clear SQLite
	return c.sqlite.Clear(ctx)
}

// Has checks if a key exists in either cache.
func (c *CompositeCache) Has(ctx context.Context, key string) bool {
	if c.memory.Has(ctx, key) {
		return true
	}
	return c.sqlite.Has(ctx, key)
}

// Stats returns combined cache statistics.
func (c *CompositeCache) Stats(ctx context.Context) (*ports.CacheStats, error) {
	// Get SQLite stats as the source of truth for persistent stats
	stats, err := c.sqlite.Stats(ctx)
	if err != nil {
		return nil, err
	}

	// Get memory stats for current size info
	memStats, err := c.memory.Stats(ctx)
	if err != nil {
		return nil, err
	}

	// Combine stats
	stats.TotalEntries = memStats.TotalEntries + stats.TotalEntries
	// Use memory hit/miss for recent session
	stats.HitCount += memStats.HitCount
	stats.MissCount += memStats.MissCount

	if stats.HitCount+stats.MissCount > 0 {
		stats.HitRate = float64(stats.HitCount) / float64(stats.HitCount+stats.MissCount) * 100
	}

	return stats, nil
}

// Cleanup removes expired entries from both caches.
func (c *CompositeCache) Cleanup(ctx context.Context) (int64, error) {
	memRemoved, _ := c.memory.Cleanup(ctx)
	sqlRemoved, err := c.sqlite.Cleanup(ctx)

	return memRemoved + sqlRemoved, err
}

// Keys returns all unique keys matching a pattern from both caches.
func (c *CompositeCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	// Get keys from both caches
	memKeys, err := c.memory.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	sqlKeys, err := c.sqlite.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	// Combine and deduplicate
	keySet := make(map[string]struct{})
	for _, key := range memKeys {
		keySet[key] = struct{}{}
	}
	for _, key := range sqlKeys {
		keySet[key] = struct{}{}
	}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}

	return keys, nil
}

// Size returns the total size of both caches in bytes.
func (c *CompositeCache) Size(ctx context.Context) (int64, error) {
	memSize, err := c.memory.Size(ctx)
	if err != nil {
		return 0, err
	}

	sqlSize, err := c.sqlite.Size(ctx)
	if err != nil {
		return 0, err
	}

	return memSize + sqlSize, nil
}

// Close closes the composite cache and its underlying caches.
func (c *CompositeCache) Close() error {
	return c.memory.Close()
}

// Ensure CompositeCache implements CachePort
var _ ports.CachePort = (*CompositeCache)(nil)
