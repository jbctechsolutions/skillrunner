// Package marketplace provides a multi-source registry for loading skills and agents
// from various marketplace sources including local filesystems, GitHub repos, and
// third-party registries like aitmpl.com.
package marketplace

import (
	"context"
	"fmt"
	"sync"
)

// SourceType identifies the type of marketplace source
type SourceType string

const (
	SourceTypeLocal    SourceType = "local"    // Local filesystem (default jbctech-marketplace)
	SourceTypeGitHub   SourceType = "github"   // GitHub repository
	SourceTypeNPM      SourceType = "npm"      // NPM registry (aitmpl.com uses this)
	SourceTypeHTTP     SourceType = "http"     // Generic HTTP/HTTPS endpoint
	SourceTypeRegistry SourceType = "registry" // Generic marketplace registry
)

// SourceConfig represents configuration for a marketplace source
type SourceConfig struct {
	// Name is the unique identifier for this source
	Name string `yaml:"name" json:"name"`

	// Type identifies the source type (local, github, npm, http, registry)
	Type SourceType `yaml:"type" json:"type"`

	// Priority determines order when searching (lower = higher priority)
	Priority int `yaml:"priority" json:"priority"`

	// Enabled controls whether this source is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Local source configuration
	Path string `yaml:"path,omitempty" json:"path,omitempty"` // Filesystem path

	// GitHub source configuration
	Owner  string `yaml:"owner,omitempty" json:"owner,omitempty"`   // GitHub owner/org
	Repo   string `yaml:"repo,omitempty" json:"repo,omitempty"`     // Repository name
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"` // Branch (default: main)

	// NPM source configuration
	Package string `yaml:"package,omitempty" json:"package,omitempty"` // NPM package name

	// HTTP source configuration
	URL string `yaml:"url,omitempty" json:"url,omitempty"` // Base URL

	// Auth configuration (optional)
	AuthToken string `yaml:"auth_token,omitempty" json:"auth_token,omitempty"`

	// Cache configuration
	CacheTTL int `yaml:"cache_ttl,omitempty" json:"cache_ttl,omitempty"` // Cache TTL in seconds
}

// Source is the interface that all marketplace sources must implement
type Source interface {
	// Name returns the source identifier
	Name() string

	// Type returns the source type
	Type() SourceType

	// Priority returns the source priority (lower = higher priority)
	Priority() int

	// ListSkills returns all available skills from this source
	ListSkills(ctx context.Context) ([]*Skill, error)

	// GetSkill retrieves a specific skill by ID
	GetSkill(ctx context.Context, id string) (*Skill, error)

	// ListAgents returns all available agents from this source
	ListAgents(ctx context.Context) ([]*Agent, error)

	// GetAgent retrieves a specific agent by ID
	GetAgent(ctx context.Context, id string) (*Agent, error)

	// ListCommands returns all available commands from this source
	ListCommands(ctx context.Context) ([]*Command, error)

	// GetCommand retrieves a specific command by ID
	GetCommand(ctx context.Context, id string) (*Command, error)

	// Search searches for components matching a query
	Search(ctx context.Context, query string, types []ComponentType) ([]*SearchResult, error)

	// Refresh reloads the source data
	Refresh(ctx context.Context) error

	// IsHealthy checks if the source is accessible
	IsHealthy(ctx context.Context) bool
}

// ComponentType identifies the type of marketplace component
type ComponentType string

const (
	ComponentTypeSkill   ComponentType = "skill"
	ComponentTypeAgent   ComponentType = "agent"
	ComponentTypeCommand ComponentType = "command"
)

// Skill represents a skill from any marketplace source
type Skill struct {
	// Core identification
	ID     string `json:"id"`
	Source string `json:"source"` // Source name this skill came from

	// Metadata
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	LastUpdated string   `json:"last_updated"`

	// Content
	Content string `json:"content"` // Full markdown content

	// Source-specific metadata
	SourceURL string            `json:"source_url,omitempty"` // URL to source (GitHub, npm, etc.)
	Metadata  map[string]string `json:"metadata,omitempty"`   // Additional source-specific data
}

// Agent represents an agent from any marketplace source
type Agent struct {
	// Core identification
	ID     string `json:"id"`
	Source string `json:"source"` // Source name this agent came from

	// Metadata
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model"`

	// Skills
	PrimarySkill     string   `json:"primary_skill"`
	SupportingSkills []string `json:"supporting_skills"`

	// Configuration
	Tools         []string       `json:"tools"`
	Routing       *AgentRouting  `json:"routing,omitempty"`
	Orchestration *Orchestration `json:"orchestration,omitempty"`

	// Content
	Content string `json:"content"` // Full markdown content

	// Source-specific metadata
	SourceURL string            `json:"source_url,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// AgentRouting represents routing configuration for an agent
type AgentRouting struct {
	DeferToSkill  bool   `json:"defer_to_skill" yaml:"defer_to_skill"`
	FallbackModel string `json:"fallback_model" yaml:"fallback_model"`
}

// Orchestration represents orchestration configuration for an agent
type Orchestration struct {
	Enabled          bool     `json:"enabled" yaml:"enabled"`
	DefaultPhases    []string `json:"default_phases" yaml:"default_phases"`
	RoutingStrategy  string   `json:"routing_strategy" yaml:"routing_strategy"`
	CostOptimization bool     `json:"cost_optimization" yaml:"cost_optimization"`
}

// Command represents a command from any marketplace source
type Command struct {
	// Core identification
	ID     string `json:"id"`
	Source string `json:"source"`

	// Metadata
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	ArgumentHint string   `json:"argument_hint"`
	AllowedTools []string `json:"allowed_tools"`

	// Content
	Content string `json:"content"`

	// Source-specific metadata
	SourceURL string            `json:"source_url,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SearchResult represents a search result from any component type
type SearchResult struct {
	Type        ComponentType `json:"type"`
	ID          string        `json:"id"`
	Source      string        `json:"source"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Score       float64       `json:"score"` // Relevance score (0-1)
}

// Registry manages multiple marketplace sources
type Registry struct {
	sources  map[string]Source
	order    []string // Ordered by priority
	mu       sync.RWMutex
	defaults []SourceConfig
}

// NewRegistry creates a new marketplace registry
func NewRegistry() *Registry {
	return &Registry{
		sources:  make(map[string]Source),
		order:    make([]string, 0),
		defaults: getDefaultSources(),
	}
}

// getDefaultSources returns the default marketplace source configurations
func getDefaultSources() []SourceConfig {
	return []SourceConfig{
		{
			Name:     "jbctech-marketplace",
			Type:     SourceTypeLocal,
			Priority: 0,
			Enabled:  true,
			// Path will be computed at runtime
		},
		{
			Name:     "aitmpl",
			Type:     SourceTypeNPM,
			Priority: 10,
			Enabled:  false, // Disabled by default, user can enable
			Package:  "claude-code-templates",
		},
	}
}

// AddSource adds a marketplace source to the registry
func (r *Registry) AddSource(source Source) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := source.Name()
	if _, exists := r.sources[name]; exists {
		return fmt.Errorf("source already exists: %s", name)
	}

	r.sources[name] = source
	r.insertByPriority(name, source.Priority())
	return nil
}

// RemoveSource removes a marketplace source from the registry
func (r *Registry) RemoveSource(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sources[name]; !exists {
		return fmt.Errorf("source not found: %s", name)
	}

	delete(r.sources, name)
	r.removeFromOrder(name)
	return nil
}

// GetSource retrieves a specific source by name
func (r *Registry) GetSource(name string) (Source, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	source, exists := r.sources[name]
	if !exists {
		return nil, fmt.Errorf("source not found: %s", name)
	}
	return source, nil
}

// ListSources returns all registered sources in priority order
func (r *Registry) ListSources() []Source {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources := make([]Source, 0, len(r.order))
	for _, name := range r.order {
		if source, exists := r.sources[name]; exists {
			sources = append(sources, source)
		}
	}
	return sources
}

// ListSkills returns all skills from all sources, merged by priority
func (r *Registry) ListSkills(ctx context.Context) ([]*Skill, error) {
	r.mu.RLock()
	sources := make([]Source, len(r.order))
	for i, name := range r.order {
		sources[i] = r.sources[name]
	}
	r.mu.RUnlock()

	seen := make(map[string]bool)
	var allSkills []*Skill

	for _, source := range sources {
		skills, err := source.ListSkills(ctx)
		if err != nil {
			// Log warning but continue with other sources
			fmt.Printf("Warning: failed to list skills from %s: %v\n", source.Name(), err)
			continue
		}

		for _, skill := range skills {
			// First source wins for duplicate IDs
			if !seen[skill.ID] {
				seen[skill.ID] = true
				allSkills = append(allSkills, skill)
			}
		}
	}

	return allSkills, nil
}

// GetSkill retrieves a skill by ID, searching sources in priority order
func (r *Registry) GetSkill(ctx context.Context, id string) (*Skill, error) {
	r.mu.RLock()
	sources := make([]Source, len(r.order))
	for i, name := range r.order {
		sources[i] = r.sources[name]
	}
	r.mu.RUnlock()

	for _, source := range sources {
		skill, err := source.GetSkill(ctx, id)
		if err == nil {
			return skill, nil
		}
		// Continue searching other sources
	}

	return nil, fmt.Errorf("skill not found: %s", id)
}

// ListAgents returns all agents from all sources, merged by priority
func (r *Registry) ListAgents(ctx context.Context) ([]*Agent, error) {
	r.mu.RLock()
	sources := make([]Source, len(r.order))
	for i, name := range r.order {
		sources[i] = r.sources[name]
	}
	r.mu.RUnlock()

	seen := make(map[string]bool)
	var allAgents []*Agent

	for _, source := range sources {
		agents, err := source.ListAgents(ctx)
		if err != nil {
			fmt.Printf("Warning: failed to list agents from %s: %v\n", source.Name(), err)
			continue
		}

		for _, agent := range agents {
			if !seen[agent.ID] {
				seen[agent.ID] = true
				allAgents = append(allAgents, agent)
			}
		}
	}

	return allAgents, nil
}

// GetAgent retrieves an agent by ID, searching sources in priority order
func (r *Registry) GetAgent(ctx context.Context, id string) (*Agent, error) {
	r.mu.RLock()
	sources := make([]Source, len(r.order))
	for i, name := range r.order {
		sources[i] = r.sources[name]
	}
	r.mu.RUnlock()

	for _, source := range sources {
		agent, err := source.GetAgent(ctx, id)
		if err == nil {
			return agent, nil
		}
	}

	return nil, fmt.Errorf("agent not found: %s", id)
}

// Search searches all sources for components matching a query
func (r *Registry) Search(ctx context.Context, query string, types []ComponentType) ([]*SearchResult, error) {
	r.mu.RLock()
	sources := make([]Source, len(r.order))
	for i, name := range r.order {
		sources[i] = r.sources[name]
	}
	r.mu.RUnlock()

	var allResults []*SearchResult

	for _, source := range sources {
		results, err := source.Search(ctx, query, types)
		if err != nil {
			fmt.Printf("Warning: search failed in %s: %v\n", source.Name(), err)
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// RefreshAll reloads data from all sources
func (r *Registry) RefreshAll(ctx context.Context) error {
	r.mu.RLock()
	sources := make([]Source, len(r.order))
	for i, name := range r.order {
		sources[i] = r.sources[name]
	}
	r.mu.RUnlock()

	var firstErr error
	for _, source := range sources {
		if err := source.Refresh(ctx); err != nil {
			fmt.Printf("Warning: refresh failed for %s: %v\n", source.Name(), err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// HealthCheck checks all sources and returns their status
func (r *Registry) HealthCheck(ctx context.Context) map[string]bool {
	r.mu.RLock()
	sources := make([]Source, len(r.order))
	names := make([]string, len(r.order))
	for i, name := range r.order {
		sources[i] = r.sources[name]
		names[i] = name
	}
	r.mu.RUnlock()

	status := make(map[string]bool)
	for i, source := range sources {
		status[names[i]] = source.IsHealthy(ctx)
	}

	return status
}

// insertByPriority inserts a source name in priority order
func (r *Registry) insertByPriority(name string, priority int) {
	insertAt := len(r.order)
	for i, existingName := range r.order {
		if source, exists := r.sources[existingName]; exists {
			if priority < source.Priority() {
				insertAt = i
				break
			}
		}
	}

	// Insert at position
	r.order = append(r.order, "")
	copy(r.order[insertAt+1:], r.order[insertAt:])
	r.order[insertAt] = name
}

// removeFromOrder removes a source name from the order slice
func (r *Registry) removeFromOrder(name string) {
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			return
		}
	}
}

// DefaultConfigs returns the default source configurations
func (r *Registry) DefaultConfigs() []SourceConfig {
	return r.defaults
}
