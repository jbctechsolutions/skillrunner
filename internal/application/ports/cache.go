package ports

import (
	"context"
	"time"
)

// CacheEntry represents a cached item with metadata.
type CacheEntry struct {
	Key        string        `json:"key"`
	Value      interface{}   `json:"value"`
	CreatedAt  time.Time     `json:"created_at"`
	ExpiresAt  time.Time     `json:"expires_at"`
	TTL        time.Duration `json:"ttl"`
	HitCount   int64         `json:"hit_count"`
	Size       int64         `json:"size"` // Size in bytes
	ModelID    string        `json:"model_id,omitempty"`
	PromptHash string        `json:"prompt_hash,omitempty"`
}

// CacheStats represents cache statistics.
type CacheStats struct {
	TotalEntries  int64         `json:"total_entries"`
	TotalSize     int64         `json:"total_size"` // Total cache size in bytes
	HitCount      int64         `json:"hit_count"`
	MissCount     int64         `json:"miss_count"`
	HitRate       float64       `json:"hit_rate"` // Percentage
	EvictionCount int64         `json:"eviction_count"`
	ExpiredCount  int64         `json:"expired_count"`
	OldestEntry   time.Time     `json:"oldest_entry"`
	NewestEntry   time.Time     `json:"newest_entry"`
	AvgTTL        time.Duration `json:"avg_ttl"`
	TokensSaved   int64         `json:"tokens_saved"` // Total tokens saved by cache hits
	CostSaved     float64       `json:"cost_saved"`   // Estimated cost saved in USD
}

// CachePort for caching data with extended capabilities for Wave 10.
type CachePort interface {
	// Get retrieves an item from cache. Returns the value and true if found.
	Get(ctx context.Context, key string) (interface{}, bool)

	// GetEntry retrieves the full cache entry including metadata.
	GetEntry(ctx context.Context, key string) (*CacheEntry, bool)

	// Set stores an item in cache with the specified TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// SetWithMetadata stores an item with additional metadata for tracking.
	SetWithMetadata(ctx context.Context, entry *CacheEntry) error

	// Delete removes an item from cache.
	Delete(ctx context.Context, key string) error

	// Clear removes all items from cache.
	Clear(ctx context.Context) error

	// Has checks if a key exists in cache without retrieving it.
	Has(ctx context.Context, key string) bool

	// Stats returns cache statistics.
	Stats(ctx context.Context) (*CacheStats, error)

	// Cleanup removes expired entries. Returns number of entries removed.
	Cleanup(ctx context.Context) (int64, error)

	// Keys returns all keys matching a pattern (empty pattern = all keys).
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Size returns the total size of the cache in bytes.
	Size(ctx context.Context) (int64, error)
}

// ResponseCachePort extends CachePort with LLM-specific caching operations.
type ResponseCachePort interface {
	CachePort

	// GetResponse retrieves a cached LLM response by request fingerprint.
	GetResponse(ctx context.Context, fingerprint string) (*CompletionResponse, bool)

	// SetResponse caches an LLM response with its request fingerprint.
	SetResponse(ctx context.Context, fingerprint string, response *CompletionResponse, ttl time.Duration) error

	// GetByModel retrieves all cached responses for a specific model.
	GetByModel(ctx context.Context, modelID string) ([]*CacheEntry, error)

	// GetTokenStats returns token-related cache statistics.
	GetTokenStats(ctx context.Context) (inputTokensSaved, outputTokensSaved int64, err error)
}

// SecretStorePort for secure credential storage (v1.1 placeholder)
type SecretStorePort interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]string, error)
}
