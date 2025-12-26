package cache

import (
	"context"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

func TestMemoryCache_SetGet(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0) // 1MB, no cleanup
	defer cache.Close()

	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "key1", "value1", time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get the value
	val, found := cache.Get(ctx, "key1")
	if !found {
		t.Fatal("Get() returned not found")
	}
	if val != "value1" {
		t.Errorf("Get() = %v, want value1", val)
	}
}

func TestMemoryCache_GetEntry(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "key1", "value1", time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get the entry
	entry, found := cache.GetEntry(ctx, "key1")
	if !found {
		t.Fatal("GetEntry() returned not found")
	}
	if entry.Key != "key1" {
		t.Errorf("entry.Key = %v, want key1", entry.Key)
	}
	if entry.Value != "value1" {
		t.Errorf("entry.Value = %v, want value1", entry.Value)
	}
	if entry.TTL != time.Hour {
		t.Errorf("entry.TTL = %v, want 1h", entry.TTL)
	}
}

func TestMemoryCache_Miss(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Get non-existent key
	_, found := cache.Get(ctx, "nonexistent")
	if found {
		t.Error("Get() returned found for nonexistent key")
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Set a value with very short TTL
	err := cache.Set(ctx, "key1", "value1", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get immediately should succeed
	_, found := cache.Get(ctx, "key1")
	if !found {
		t.Fatal("Get() should find value before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Get after expiration should fail
	_, found = cache.Get(ctx, "key1")
	if found {
		t.Error("Get() should not find expired value")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "key1", "value1", time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Delete the value
	err = cache.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Get should fail
	_, found := cache.Get(ctx, "key1")
	if found {
		t.Error("Get() should not find deleted value")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 5; i++ {
		err := cache.Set(ctx, "key"+string(rune('0'+i)), "value", time.Hour)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	// Clear the cache
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// All values should be gone
	for i := 0; i < 5; i++ {
		_, found := cache.Get(ctx, "key"+string(rune('0'+i)))
		if found {
			t.Errorf("Get() should not find value after clear for key%d", i)
		}
	}
}

func TestMemoryCache_Has(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Check non-existent key
	if cache.Has(ctx, "key1") {
		t.Error("Has() should return false for non-existent key")
	}

	// Set a value
	_ = cache.Set(ctx, "key1", "value1", time.Hour)

	// Check existing key
	if !cache.Has(ctx, "key1") {
		t.Error("Has() should return true for existing key")
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Set some values
	_ = cache.Set(ctx, "key1", "value1", time.Hour)
	_ = cache.Set(ctx, "key2", "value2", time.Hour)

	// Generate some hits and misses
	cache.Get(ctx, "key1") // hit
	cache.Get(ctx, "key1") // hit
	cache.Get(ctx, "key3") // miss

	stats, err := cache.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.TotalEntries != 2 {
		t.Errorf("TotalEntries = %d, want 2", stats.TotalEntries)
	}
	if stats.HitCount != 2 {
		t.Errorf("HitCount = %d, want 2", stats.HitCount)
	}
	if stats.MissCount != 1 {
		t.Errorf("MissCount = %d, want 1", stats.MissCount)
	}
}

func TestMemoryCache_Keys(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Set some values
	_ = cache.Set(ctx, "foo1", "value", time.Hour)
	_ = cache.Set(ctx, "foo2", "value", time.Hour)
	_ = cache.Set(ctx, "bar1", "value", time.Hour)

	// Get all keys
	keys, err := cache.Keys(ctx, "")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("Keys() returned %d keys, want 3", len(keys))
	}

	// Get keys matching pattern
	keys, err = cache.Keys(ctx, "^foo")
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("Keys('^foo') returned %d keys, want 2", len(keys))
	}
}

func TestMemoryCache_Size(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Initially empty
	size, err := cache.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 0 {
		t.Errorf("Size() = %d, want 0 for empty cache", size)
	}

	// Add some data
	_ = cache.Set(ctx, "key1", "value1", time.Hour)

	size, err = cache.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size <= 0 {
		t.Errorf("Size() = %d, want > 0 after adding data", size)
	}
}

func TestMemoryCache_Eviction(t *testing.T) {
	// Very small cache - 50 bytes can only hold 1-2 small entries
	cache := NewMemoryCache(50, 0)
	defer cache.Close()

	ctx := context.Background()

	// Add entries with longer values to trigger eviction
	for i := 0; i < 20; i++ {
		_ = cache.Set(ctx, "key"+string(rune('a'+i)), "this is a longer value that takes more space", time.Hour)
	}

	// Cache should have evicted some entries
	stats, _ := cache.Stats(ctx)
	if stats.EvictionCount == 0 {
		t.Errorf("EvictionCount should be > 0 for small cache with many entries, got entries=%d, size=%d",
			stats.TotalEntries, stats.TotalSize)
	}
}

func TestMemoryCache_SetWithMetadata(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()
	now := time.Now()

	entry := &ports.CacheEntry{
		Key:        "key1",
		Value:      "value1",
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Hour),
		TTL:        time.Hour,
		ModelID:    "gpt-4",
		PromptHash: "abc123",
	}

	err := cache.SetWithMetadata(ctx, entry)
	if err != nil {
		t.Fatalf("SetWithMetadata() error = %v", err)
	}

	retrieved, found := cache.GetEntry(ctx, "key1")
	if !found {
		t.Fatal("GetEntry() returned not found")
	}
	if retrieved.ModelID != "gpt-4" {
		t.Errorf("ModelID = %v, want gpt-4", retrieved.ModelID)
	}
	if retrieved.PromptHash != "abc123" {
		t.Errorf("PromptHash = %v, want abc123", retrieved.PromptHash)
	}
}

func TestMemoryCache_Cleanup(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 0)
	defer cache.Close()

	ctx := context.Background()

	// Add some expired entries
	_ = cache.Set(ctx, "expired1", "value", -time.Hour)
	_ = cache.Set(ctx, "expired2", "value", -time.Hour)
	_ = cache.Set(ctx, "valid", "value", time.Hour)

	// Force cleanup
	removed, err := cache.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if removed != 2 {
		t.Errorf("Cleanup() removed %d entries, want 2", removed)
	}

	// Valid entry should still exist
	if !cache.Has(ctx, "valid") {
		t.Error("valid entry should still exist after cleanup")
	}
}

func TestMemoryCache_CloseIdempotent(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 100*time.Millisecond)

	// Close multiple times should not panic
	cache.Close()
	cache.Close()
	cache.Close()
}
