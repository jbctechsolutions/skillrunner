package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/config"
)

// CacheEntry represents a cached execution result
type CacheEntry struct {
	Key       string    `json:"key"`
	Skill     string    `json:"skill"`
	Request   string    `json:"request"`
	Model     string    `json:"model"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	HitCount  int       `json:"hit_count"`
}

// Manager manages result caching with TTL
type Manager struct {
	cacheDir string
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	ttl      time.Duration
}

// NewManager creates a new cache manager
func NewManager(ttlHours int) (*Manager, error) {
	// Check if path is overridden in config
	var skillrunnerDir string
	cfgManager, err := config.NewManager("")
	if err == nil {
		cfg := cfgManager.Get()
		if cfg.Paths != nil && cfg.Paths.SkillrunnerDir != "" {
			skillrunnerDir = cfg.Paths.SkillrunnerDir
		}
	}

	// Default to ~/.skillrunner if not overridden
	if skillrunnerDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		skillrunnerDir = filepath.Join(homeDir, ".skillrunner")
	}

	cacheDir := filepath.Join(skillrunnerDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	ttl := time.Duration(ttlHours) * time.Hour
	if ttl == 0 {
		ttl = 24 * time.Hour // Default 24 hours
	}

	manager := &Manager{
		cacheDir: cacheDir,
		entries:  make(map[string]*CacheEntry),
		ttl:      ttl,
	}

	// Load existing cache entries
	if err := manager.load(); err != nil {
		// Non-fatal: continue with empty cache
		// Only log if it's not a "file doesn't exist" error
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: Failed to load cache: %v\n", err)
		}
	}

	// Clean expired entries
	manager.cleanExpired()

	return manager, nil
}

// Get retrieves a cached result if available and not expired
func (m *Manager) Get(skill, request, model string) (string, bool) {
	key := m.generateKey(skill, request, model)

	m.mu.RLock()
	entry, exists := m.entries[key]
	m.mu.RUnlock()

	if !exists {
		return "", false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		m.mu.Lock()
		delete(m.entries, key)
		m.mu.Unlock()
		m.save()
		return "", false
	}

	// Update hit count
	m.mu.Lock()
	entry.HitCount++
	m.entries[key] = entry
	m.mu.Unlock()

	return entry.Result, true
}

// Set stores a result in the cache
func (m *Manager) Set(skill, request, model, result string) error {
	key := m.generateKey(skill, request, model)
	now := time.Now()

	entry := &CacheEntry{
		Key:       key,
		Skill:     skill,
		Request:   request,
		Model:     model,
		Result:    result,
		CreatedAt: now,
		ExpiresAt: now.Add(m.ttl),
		HitCount:  0,
	}

	m.mu.Lock()
	m.entries[key] = entry
	m.mu.Unlock()

	return m.save()
}

// Clear removes all cache entries
func (m *Manager) Clear() error {
	m.mu.Lock()
	m.entries = make(map[string]*CacheEntry)
	m.mu.Unlock()

	return m.save()
}

// ClearExpired removes expired entries
func (m *Manager) ClearExpired() {
	m.cleanExpired()
}

// Stats returns cache statistics
func (m *Manager) Stats() (total, valid, expired int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	for _, entry := range m.entries {
		total++
		if now.After(entry.ExpiresAt) {
			expired++
		} else {
			valid++
		}
	}

	return total, valid, expired
}

// generateKey generates a cache key from skill, request, and model
func (m *Manager) generateKey(skill, request, model string) string {
	data := fmt.Sprintf("%s:%s:%s", skill, request, model)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// load loads cache entries from disk
func (m *Manager) load() error {
	cacheFile := filepath.Join(m.cacheDir, "results.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file yet
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var entries []*CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshal cache: %w", err)
	}

	m.mu.Lock()
	for _, entry := range entries {
		m.entries[entry.Key] = entry
	}
	m.mu.Unlock()

	return nil
}

// save saves cache entries to disk
func (m *Manager) save() error {
	m.mu.RLock()
	entries := make([]*CacheEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}
	m.mu.RUnlock()

	cacheFile := filepath.Join(m.cacheDir, "results.json")
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	// Write to temp file first, then rename (atomic write)
	tmpFile := cacheFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	if err := os.Rename(tmpFile, cacheFile); err != nil {
		return fmt.Errorf("rename cache file: %w", err)
	}

	return nil
}

// cleanExpired removes expired entries
func (m *Manager) cleanExpired() {
	m.mu.Lock()
	now := time.Now()
	expiredKeys := []string{}
	for key, entry := range m.entries {
		if now.After(entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	for _, key := range expiredKeys {
		delete(m.entries, key)
	}
	m.mu.Unlock()

	// Save after cleanup (outside the lock to avoid deadlock)
	if len(expiredKeys) > 0 {
		_ = m.save()
	}
}
