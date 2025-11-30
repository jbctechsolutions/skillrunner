package marketplace

import (
	"strings"
	"sync"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// SkillCache caches marketplace skill metadata
type SkillCache struct {
	skills        map[string]*CachedSkill
	searchResults map[string]*CachedSearchResult
	mu            sync.RWMutex
	ttl           time.Duration
}

// CachedSkill wraps a skill with caching metadata
type CachedSkill struct {
	Skill      *types.MarketplaceSkill
	CachedAt   time.Time
	AccessedAt time.Time
}

// CachedSearchResult wraps search results with caching metadata
type CachedSearchResult struct {
	Query      string
	Skills     []*types.MarketplaceSkill
	CachedAt   time.Time
	AccessedAt time.Time
}

// NewSkillCache creates a new skill cache
func NewSkillCache() *SkillCache {
	return &SkillCache{
		skills:        make(map[string]*CachedSkill),
		searchResults: make(map[string]*CachedSearchResult),
		ttl:           24 * time.Hour, // Default 24 hour TTL
	}
}

// Add adds a skill to the cache
func (c *SkillCache) Add(skill *types.MarketplaceSkill) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.skills[skill.ID] = &CachedSkill{
		Skill:      skill,
		CachedAt:   time.Now(),
		AccessedAt: time.Now(),
	}
}

// Get retrieves a skill from the cache
func (c *SkillCache) Get(skillID string) *types.MarketplaceSkill {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.skills[skillID]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Since(cached.CachedAt) > c.ttl {
		return nil
	}

	// Update access time
	cached.AccessedAt = time.Now()

	return cached.Skill
}

// AddSearchResults adds search results to the cache
func (c *SkillCache) AddSearchResults(query string, skills []*types.MarketplaceSkill) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Normalize query for caching
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	c.searchResults[normalizedQuery] = &CachedSearchResult{
		Query:      query,
		Skills:     skills,
		CachedAt:   time.Now(),
		AccessedAt: time.Now(),
	}

	// Also cache individual skills
	for _, skill := range skills {
		c.skills[skill.ID] = &CachedSkill{
			Skill:      skill,
			CachedAt:   time.Now(),
			AccessedAt: time.Now(),
		}
	}
}

// Search retrieves search results from the cache
func (c *SkillCache) Search(query string) []*types.MarketplaceSkill {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Normalize query
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	cached, exists := c.searchResults[normalizedQuery]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Since(cached.CachedAt) > c.ttl {
		return nil
	}

	// Update access time
	cached.AccessedAt = time.Now()

	return cached.Skills
}

// Clear removes all cached data
func (c *SkillCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.skills = make(map[string]*CachedSkill)
	c.searchResults = make(map[string]*CachedSearchResult)
}

// ClearExpired removes expired cache entries
func (c *SkillCache) ClearExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Clear expired skills
	for id, cached := range c.skills {
		if now.Sub(cached.CachedAt) > c.ttl {
			delete(c.skills, id)
		}
	}

	// Clear expired search results
	for query, cached := range c.searchResults {
		if now.Sub(cached.CachedAt) > c.ttl {
			delete(c.searchResults, query)
		}
	}
}

// Stats returns cache statistics
func (c *SkillCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		SkillCount:        len(c.skills),
		SearchResultCount: len(c.searchResults),
		TTL:               c.ttl,
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	SkillCount        int
	SearchResultCount int
	TTL               time.Duration
}
