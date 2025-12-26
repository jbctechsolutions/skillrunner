// Package cache provides cache adapters for response caching.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// ResponseCache implements ResponseCachePort for caching LLM responses.
type ResponseCache struct {
	*CompositeCache
	defaultTTL time.Duration
}

// NewResponseCache creates a new response cache with the given underlying cache.
func NewResponseCache(cache *CompositeCache, defaultTTL time.Duration) *ResponseCache {
	return &ResponseCache{
		CompositeCache: cache,
		defaultTTL:     defaultTTL,
	}
}

// GetResponse retrieves a cached LLM response by request fingerprint.
func (r *ResponseCache) GetResponse(ctx context.Context, fingerprint string) (*ports.CompletionResponse, bool) {
	entry, found := r.GetEntry(ctx, fingerprint)
	if !found {
		return nil, false
	}

	// Try to convert the value to CompletionResponse
	if resp, ok := entry.Value.(*ports.CompletionResponse); ok {
		return resp, true
	}

	// Try JSON unmarshaling if stored as bytes
	if data, ok := entry.Value.([]byte); ok {
		var resp ports.CompletionResponse
		if err := json.Unmarshal(data, &resp); err == nil {
			return &resp, true
		}
	}

	return nil, false
}

// SetResponse caches an LLM response with its request fingerprint.
func (r *ResponseCache) SetResponse(ctx context.Context, fingerprint string, response *ports.CompletionResponse, ttl time.Duration) error {
	if ttl == 0 {
		ttl = r.defaultTTL
	}

	now := time.Now()
	entry := &ports.CacheEntry{
		Key:        fingerprint,
		Value:      response,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
		TTL:        ttl,
		HitCount:   0,
		ModelID:    response.ModelUsed,
		PromptHash: fingerprint,
	}

	return r.SetWithMetadata(ctx, entry)
}

// GetByModel retrieves all cached responses for a specific model.
func (r *ResponseCache) GetByModel(ctx context.Context, modelID string) ([]*ports.CacheEntry, error) {
	// Get all keys and filter by model
	keys, err := r.Keys(ctx, "")
	if err != nil {
		return nil, err
	}

	var entries []*ports.CacheEntry
	for _, key := range keys {
		entry, found := r.GetEntry(ctx, key)
		if found && entry.ModelID == modelID {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// GetTokenStats returns token-related cache statistics.
func (r *ResponseCache) GetTokenStats(ctx context.Context) (inputTokensSaved, outputTokensSaved int64, err error) {
	stats, err := r.Stats(ctx)
	if err != nil {
		return 0, 0, err
	}

	// TokensSaved is the combined total; we estimate split
	// In a more accurate implementation, we'd track these separately
	return stats.TokensSaved / 2, stats.TokensSaved / 2, nil
}

// Fingerprint generates a cache key from a completion request.
// It creates a deterministic hash of the request parameters.
func Fingerprint(req ports.CompletionRequest) string {
	// Build a deterministic string representation
	var parts []string

	parts = append(parts, "model:"+req.ModelID)

	// Sort messages for determinism
	for i, msg := range req.Messages {
		parts = append(parts, sortedKeyValue("msg", i, msg.Role, msg.Content))
	}

	parts = append(parts, intToString("max_tokens", req.MaxTokens))
	parts = append(parts, floatToString("temperature", req.Temperature))

	if req.SystemPrompt != "" {
		parts = append(parts, "system:"+req.SystemPrompt)
	}

	// Sort all parts for determinism
	sort.Strings(parts)

	// Hash the combined string
	combined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(combined))

	return hex.EncodeToString(hash[:])
}

// FingerprintWithSalt generates a cache key with an additional salt for namespacing.
func FingerprintWithSalt(req ports.CompletionRequest, salt string) string {
	base := Fingerprint(req)
	if salt == "" {
		return base
	}

	combined := salt + ":" + base
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// Helper functions for fingerprinting

func sortedKeyValue(prefix string, index int, role, content string) string {
	return prefix + "[" + intStr(index) + "]:" + role + "=" + truncateForHash(content)
}

func intToString(key string, value int) string {
	return key + ":" + intStr(value)
}

func floatToString(key string, value float32) string {
	return key + ":" + floatStr(value)
}

func intStr(i int) string {
	data, _ := json.Marshal(i)
	return string(data)
}

func floatStr(f float32) string {
	data, _ := json.Marshal(f)
	return string(data)
}

func truncateForHash(s string) string {
	// For very long content, use a hash to avoid extremely long keys
	if len(s) > 1000 {
		hash := sha256.Sum256([]byte(s))
		return "hash:" + hex.EncodeToString(hash[:16])
	}
	return s
}

// Ensure ResponseCache implements ResponseCachePort
var _ ports.ResponseCachePort = (*ResponseCache)(nil)
