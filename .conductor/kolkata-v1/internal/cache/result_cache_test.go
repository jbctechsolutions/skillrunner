package cache

import (
	"os"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	mgr, err := NewManager(24)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("Manager should not be nil")
	}

	if mgr.ttl != 24*time.Hour {
		t.Errorf("TTL = %v, want 24h", mgr.ttl)
	}
}

func TestManager_GetSet(t *testing.T) {
	mgr, err := NewManager(1) // 1 hour TTL
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Set a value
	err = mgr.Set("test-skill", "test request", "test-model", "test result")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get the value
	result, found := mgr.Get("test-skill", "test request", "test-model")
	if !found {
		t.Error("Get should find the cached value")
	}
	if result != "test result" {
		t.Errorf("Get() = %s, want 'test result'", result)
	}

	// Get non-existent value
	_, found = mgr.Get("non-existent", "request", "model")
	if found {
		t.Error("Get should not find non-existent value")
	}
}

func TestManager_Expiration(t *testing.T) {
	// Create manager with very short TTL
	mgr, err := NewManager(0) // Use default 24h, but we'll test with manual expiration
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Manually set an expired entry
	key := mgr.generateKey("test", "request", "model")
	mgr.mu.Lock()
	mgr.entries[key] = &CacheEntry{
		Key:       key,
		Skill:     "test",
		Request:   "request",
		Model:     "model",
		Result:    "result",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	mgr.mu.Unlock()

	// Get should not find expired entry
	_, found := mgr.Get("test", "request", "model")
	if found {
		t.Error("Get should not return expired entries")
	}
}

func TestManager_Clear(t *testing.T) {
	mgr, err := NewManager(24)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Set some values
	_ = mgr.Set("skill1", "req1", "model1", "result1")
	_ = mgr.Set("skill2", "req2", "model2", "result2")

	// Clear
	err = mgr.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify cleared
	_, found := mgr.Get("skill1", "req1", "model1")
	if found {
		t.Error("Cache should be cleared")
	}
}

func TestManager_Stats(t *testing.T) {
	mgr, err := NewManager(24)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Set some values
	_ = mgr.Set("skill1", "req1", "model1", "result1")
	_ = mgr.Set("skill2", "req2", "model2", "result2")

	total, valid, expired := mgr.Stats()
	if total != 2 {
		t.Errorf("Stats total = %d, want 2", total)
	}
	if valid != 2 {
		t.Errorf("Stats valid = %d, want 2", valid)
	}
	if expired != 0 {
		t.Errorf("Stats expired = %d, want 0", expired)
	}
}

func TestManager_GenerateKey(t *testing.T) {
	mgr, err := NewManager(24)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	key1 := mgr.generateKey("skill", "request", "model")
	key2 := mgr.generateKey("skill", "request", "model")

	// Same inputs should generate same key
	if key1 != key2 {
		t.Error("Same inputs should generate same key")
	}

	// Different inputs should generate different keys
	key3 := mgr.generateKey("skill2", "request", "model")
	if key1 == key3 {
		t.Error("Different inputs should generate different keys")
	}
}

func TestManager_CleanExpired(t *testing.T) {
	mgr, err := NewManager(24)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Add expired entry manually
	key := mgr.generateKey("expired", "req", "model")
	mgr.mu.Lock()
	mgr.entries[key] = &CacheEntry{
		Key:       key,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	mgr.mu.Unlock()

	// Clean expired
	mgr.ClearExpired()

	// Verify expired entry is removed
	mgr.mu.RLock()
	_, exists := mgr.entries[key]
	mgr.mu.RUnlock()
	if exists {
		t.Error("Expired entry should be removed")
	}
}

func TestManager_LoadSave(t *testing.T) {
	// Use temp directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)

	mgr, err := NewManager(24)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Set a value
	err = mgr.Set("test-skill", "test-request", "test-model", "test-result")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Create new manager (should load from disk)
	mgr2, err := NewManager(24)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Should find the cached value
	result, found := mgr2.Get("test-skill", "test-request", "test-model")
	if !found {
		t.Error("Should load cached value from disk")
	}
	if result != "test-result" {
		t.Errorf("Loaded result = %s, want 'test-result'", result)
	}
}
