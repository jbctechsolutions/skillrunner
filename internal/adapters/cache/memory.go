// Package cache provides cache adapters for response caching.
package cache

import (
	"context"
	"encoding/json"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// MemoryCache implements CachePort using an in-memory map with TTL support.
type MemoryCache struct {
	mu          sync.RWMutex
	entries     map[string]*memoryCacheEntry
	maxSize     int64 // Maximum cache size in bytes
	currentSize int64 // Current cache size in bytes

	// Statistics
	hitCount      int64
	missCount     int64
	evictionCount int64
	expiredCount  int64

	// Cleanup
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	closeOnce     sync.Once
}

// memoryCacheEntry wraps a cache entry with additional internal metadata.
type memoryCacheEntry struct {
	entry *ports.CacheEntry
	data  []byte // Serialized value for size calculation
}

// NewMemoryCache creates a new in-memory cache with the specified max size.
func NewMemoryCache(maxSize int64, cleanupPeriod time.Duration) *MemoryCache {
	mc := &MemoryCache{
		entries:     make(map[string]*memoryCacheEntry),
		maxSize:     maxSize,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	if cleanupPeriod > 0 {
		mc.cleanupTicker = time.NewTicker(cleanupPeriod)
		go mc.cleanupLoop()
	}

	return mc
}

// cleanupLoop runs periodic cleanup of expired entries.
func (m *MemoryCache) cleanupLoop() {
	for {
		select {
		case <-m.cleanupTicker.C:
			_, _ = m.Cleanup(context.Background())
		case <-m.stopCleanup:
			m.cleanupTicker.Stop()
			return
		}
	}
}

// Close stops the cleanup goroutine and releases resources.
func (m *MemoryCache) Close() error {
	if m.cleanupTicker != nil {
		// Use sync.Once to safely close the channel exactly once,
		// avoiding races with the cleanup goroutine
		m.closeOnce.Do(func() {
			close(m.stopCleanup)
		})
	}
	return nil
}

// Get retrieves an item from cache.
func (m *MemoryCache) Get(ctx context.Context, key string) (any, bool) {
	entry, found := m.GetEntry(ctx, key)
	if !found {
		return nil, false
	}
	return entry.Value, true
}

// GetEntry retrieves the full cache entry including metadata.
func (m *MemoryCache) GetEntry(ctx context.Context, key string) (*ports.CacheEntry, bool) {
	m.mu.RLock()
	mce, exists := m.entries[key]
	m.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&m.missCount, 1)
		return nil, false
	}

	// Check if expired
	if time.Now().After(mce.entry.ExpiresAt) {
		atomic.AddInt64(&m.missCount, 1)
		atomic.AddInt64(&m.expiredCount, 1)
		// Clean up in background
		go func() {
			_ = m.Delete(context.Background(), key)
		}()
		return nil, false
	}

	atomic.AddInt64(&m.hitCount, 1)
	atomic.AddInt64(&mce.entry.HitCount, 1)

	return mce.entry, true
}

// Set stores an item in cache with the specified TTL.
func (m *MemoryCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	now := time.Now()
	entry := &ports.CacheEntry{
		Key:       key,
		Value:     value,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		TTL:       ttl,
		HitCount:  0,
	}

	return m.SetWithMetadata(ctx, entry)
}

// SetWithMetadata stores an item with additional metadata for tracking.
func (m *MemoryCache) SetWithMetadata(ctx context.Context, entry *ports.CacheEntry) error {
	// Serialize value for size calculation
	data, err := json.Marshal(entry.Value)
	if err != nil {
		return err
	}

	entry.Size = int64(len(data))

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we need to evict entries to make room
	if m.maxSize > 0 {
		// If existing key, subtract its size first
		if existing, exists := m.entries[entry.Key]; exists {
			m.currentSize -= existing.entry.Size
		}

		// Evict entries if necessary
		for m.currentSize+entry.Size > m.maxSize && len(m.entries) > 0 {
			m.evictOldest()
		}
	}

	mce := &memoryCacheEntry{
		entry: entry,
		data:  data,
	}

	m.entries[entry.Key] = mce
	m.currentSize += entry.Size

	return nil
}

// evictOldest removes the oldest entry from the cache. Must be called with lock held.
func (m *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, mce := range m.entries {
		if oldestKey == "" || mce.entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = mce.entry.CreatedAt
		}
	}

	if oldestKey != "" {
		if entry, exists := m.entries[oldestKey]; exists {
			m.currentSize -= entry.entry.Size
			delete(m.entries, oldestKey)
			atomic.AddInt64(&m.evictionCount, 1)
		}
	}
}

// Delete removes an item from cache.
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, exists := m.entries[key]; exists {
		m.currentSize -= entry.entry.Size
		delete(m.entries, key)
	}

	return nil
}

// Clear removes all items from cache.
func (m *MemoryCache) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make(map[string]*memoryCacheEntry)
	m.currentSize = 0

	return nil
}

// Has checks if a key exists in cache without retrieving it.
func (m *MemoryCache) Has(ctx context.Context, key string) bool {
	m.mu.RLock()
	mce, exists := m.entries[key]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	// Check if expired
	return !time.Now().After(mce.entry.ExpiresAt)
}

// Stats returns cache statistics.
func (m *MemoryCache) Stats(ctx context.Context) (*ports.CacheStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hits := atomic.LoadInt64(&m.hitCount)
	misses := atomic.LoadInt64(&m.missCount)

	stats := &ports.CacheStats{
		TotalEntries:  int64(len(m.entries)),
		TotalSize:     m.currentSize,
		HitCount:      hits,
		MissCount:     misses,
		EvictionCount: atomic.LoadInt64(&m.evictionCount),
		ExpiredCount:  atomic.LoadInt64(&m.expiredCount),
	}

	if hits+misses > 0 {
		stats.HitRate = float64(hits) / float64(hits+misses) * 100
	}

	// Calculate oldest/newest entries and average TTL
	var totalTTL time.Duration
	for _, mce := range m.entries {
		if stats.OldestEntry.IsZero() || mce.entry.CreatedAt.Before(stats.OldestEntry) {
			stats.OldestEntry = mce.entry.CreatedAt
		}
		if stats.NewestEntry.IsZero() || mce.entry.CreatedAt.After(stats.NewestEntry) {
			stats.NewestEntry = mce.entry.CreatedAt
		}
		totalTTL += mce.entry.TTL
	}

	if len(m.entries) > 0 {
		stats.AvgTTL = totalTTL / time.Duration(len(m.entries))
	}

	return stats, nil
}

// Cleanup removes expired entries. Returns number of entries removed.
func (m *MemoryCache) Cleanup(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var removed int64

	for key, mce := range m.entries {
		if now.After(mce.entry.ExpiresAt) {
			m.currentSize -= mce.entry.Size
			delete(m.entries, key)
			removed++
		}
	}

	atomic.AddInt64(&m.expiredCount, removed)
	return removed, nil
}

// Keys returns all keys matching a pattern (empty pattern = all keys).
func (m *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	var re *regexp.Regexp
	var err error

	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	for key, mce := range m.entries {
		// Skip expired entries
		if now.After(mce.entry.ExpiresAt) {
			continue
		}

		if re == nil || re.MatchString(key) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Size returns the total size of the cache in bytes.
func (m *MemoryCache) Size(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSize, nil
}

// Ensure MemoryCache implements CachePort
var _ ports.CachePort = (*MemoryCache)(nil)
