package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// NPMSource implements Source for NPM-based marketplaces like aitmpl.com
type NPMSource struct {
	config     SourceConfig
	skills     map[string]*Skill
	agents     map[string]*Agent
	commands   map[string]*Command
	mu         sync.RWMutex
	httpClient *http.Client
	cache      *npmCache
}

// npmCache stores cached registry data
type npmCache struct {
	skills    map[string]*Skill
	agents    map[string]*Agent
	commands  map[string]*Command
	expiresAt time.Time
}

// NPMPackageInfo represents npm package metadata
type NPMPackageInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Dist        NPMDist           `json:"dist"`
	Keywords    []string          `json:"keywords"`
	Repository  *NPMRepository    `json:"repository,omitempty"`
	Versions    map[string]string `json:"versions,omitempty"`
}

// NPMDist contains tarball distribution info
type NPMDist struct {
	Tarball   string `json:"tarball"`
	Shasum    string `json:"shasum"`
	Integrity string `json:"integrity"`
}

// NPMRepository contains repository info
type NPMRepository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// RegistryManifest represents the manifest from a template registry
type RegistryManifest struct {
	Version     string                    `json:"version" yaml:"version"`
	Name        string                    `json:"name" yaml:"name"`
	Description string                    `json:"description" yaml:"description"`
	Categories  map[string][]ManifestItem `json:"categories" yaml:"categories"`
	Skills      []ManifestItem            `json:"skills" yaml:"skills"`
	Agents      []ManifestItem            `json:"agents" yaml:"agents"`
	Commands    []ManifestItem            `json:"commands" yaml:"commands"`
}

// ManifestItem represents an item in the registry manifest
type ManifestItem struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Version     string   `json:"version" yaml:"version"`
	Author      string   `json:"author" yaml:"author"`
	Keywords    []string `json:"keywords" yaml:"keywords"`
	Path        string   `json:"path" yaml:"path"` // Path within the package
}

// NewNPMSource creates a new NPM-based source
func NewNPMSource(config SourceConfig) (*NPMSource, error) {
	if config.Type != SourceTypeNPM {
		return nil, fmt.Errorf("invalid source type: %s (expected npm)", config.Type)
	}

	if config.Package == "" && config.URL == "" {
		return nil, fmt.Errorf("npm source requires package name or URL")
	}

	source := &NPMSource{
		config:   config,
		skills:   make(map[string]*Skill),
		agents:   make(map[string]*Agent),
		commands: make(map[string]*Command),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Initial load
	if err := source.Refresh(context.Background()); err != nil {
		return nil, err
	}

	return source, nil
}

// Name returns the source identifier
func (s *NPMSource) Name() string {
	return s.config.Name
}

// Type returns the source type
func (s *NPMSource) Type() SourceType {
	return SourceTypeNPM
}

// Priority returns the source priority
func (s *NPMSource) Priority() int {
	return s.config.Priority
}

// ListSkills returns all available skills
func (s *NPMSource) ListSkills(ctx context.Context) ([]*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make([]*Skill, 0, len(s.skills))
	for _, skill := range s.skills {
		skills = append(skills, skill)
	}
	return skills, nil
}

// GetSkill retrieves a specific skill by ID
func (s *NPMSource) GetSkill(ctx context.Context, id string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, ok := s.skills[id]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", id)
	}
	return skill, nil
}

// ListAgents returns all available agents
func (s *NPMSource) ListAgents(ctx context.Context) ([]*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	return agents, nil
}

// GetAgent retrieves a specific agent by ID
func (s *NPMSource) GetAgent(ctx context.Context, id string) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return agent, nil
}

// ListCommands returns all available commands
func (s *NPMSource) ListCommands(ctx context.Context) ([]*Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	commands := make([]*Command, 0, len(s.commands))
	for _, cmd := range s.commands {
		commands = append(commands, cmd)
	}
	return commands, nil
}

// GetCommand retrieves a specific command by ID
func (s *NPMSource) GetCommand(ctx context.Context, id string) (*Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cmd, ok := s.commands[id]
	if !ok {
		return nil, fmt.Errorf("command not found: %s", id)
	}
	return cmd, nil
}

// Search searches for components matching a query
func (s *NPMSource) Search(ctx context.Context, query string, types []ComponentType) ([]*SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*SearchResult

	// Determine which types to search
	searchSkills := len(types) == 0
	searchAgents := len(types) == 0
	searchCommands := len(types) == 0
	for _, t := range types {
		switch t {
		case ComponentTypeSkill:
			searchSkills = true
		case ComponentTypeAgent:
			searchAgents = true
		case ComponentTypeCommand:
			searchCommands = true
		}
	}

	// Search skills
	if searchSkills {
		for _, skill := range s.skills {
			score := s.calculateScore(query, skill.Name, skill.Description, skill.Keywords)
			if score > 0 {
				results = append(results, &SearchResult{
					Type:        ComponentTypeSkill,
					ID:          skill.ID,
					Source:      s.config.Name,
					Name:        skill.Name,
					Description: skill.Description,
					Score:       score,
				})
			}
		}
	}

	// Search agents
	if searchAgents {
		for _, agent := range s.agents {
			score := s.calculateScore(query, agent.Name, agent.Description, nil)
			if score > 0 {
				results = append(results, &SearchResult{
					Type:        ComponentTypeAgent,
					ID:          agent.ID,
					Source:      s.config.Name,
					Name:        agent.Name,
					Description: agent.Description,
					Score:       score,
				})
			}
		}
	}

	// Search commands
	if searchCommands {
		for _, cmd := range s.commands {
			score := s.calculateScore(query, cmd.Name, cmd.Description, nil)
			if score > 0 {
				results = append(results, &SearchResult{
					Type:        ComponentTypeCommand,
					ID:          cmd.ID,
					Source:      s.config.Name,
					Name:        cmd.Name,
					Description: cmd.Description,
					Score:       score,
				})
			}
		}
	}

	return results, nil
}

// calculateScore calculates a simple relevance score
func (s *NPMSource) calculateScore(query, name, description string, keywords []string) float64 {
	query = strings.ToLower(query)
	name = strings.ToLower(name)
	description = strings.ToLower(description)

	score := 0.0

	// Name match (highest weight)
	if strings.Contains(name, query) {
		score += 0.5
		if name == query {
			score += 0.3 // Exact match bonus
		}
	}

	// Description match
	if strings.Contains(description, query) {
		score += 0.2
	}

	// Keyword match
	for _, kw := range keywords {
		if strings.Contains(strings.ToLower(kw), query) {
			score += 0.1
			break
		}
	}

	return score
}

// Refresh reloads the source data from the NPM registry
func (s *NPMSource) Refresh(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check cache
	if s.cache != nil && time.Now().Before(s.cache.expiresAt) {
		s.skills = s.cache.skills
		s.agents = s.cache.agents
		s.commands = s.cache.commands
		return nil
	}

	// Clear existing data
	s.skills = make(map[string]*Skill)
	s.agents = make(map[string]*Agent)
	s.commands = make(map[string]*Command)

	// Load from registry
	if err := s.loadFromRegistry(ctx); err != nil {
		return fmt.Errorf("load from registry: %w", err)
	}

	// Update cache
	cacheTTL := time.Duration(s.config.CacheTTL) * time.Second
	if cacheTTL == 0 {
		cacheTTL = 15 * time.Minute // Default cache TTL for remote sources
	}
	s.cache = &npmCache{
		skills:    s.skills,
		agents:    s.agents,
		commands:  s.commands,
		expiresAt: time.Now().Add(cacheTTL),
	}

	return nil
}

// IsHealthy checks if the source is accessible
func (s *NPMSource) IsHealthy(ctx context.Context) bool {
	url := s.getRegistryURL()
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// getRegistryURL returns the URL to fetch registry data from
func (s *NPMSource) getRegistryURL() string {
	if s.config.URL != "" {
		return s.config.URL
	}
	// Default to npm registry
	return fmt.Sprintf("https://registry.npmjs.org/%s/latest", s.config.Package)
}

// loadFromRegistry loads components from the npm registry
func (s *NPMSource) loadFromRegistry(ctx context.Context) error {
	// First, try to get the package info from npm
	pkgInfo, err := s.getPackageInfo(ctx)
	if err != nil {
		return fmt.Errorf("get package info: %w", err)
	}

	// Try to load manifest from the package's unpkg URL
	manifestURL := fmt.Sprintf("https://unpkg.com/%s@%s/manifest.json", s.config.Package, pkgInfo.Version)
	manifest, err := s.loadManifest(ctx, manifestURL)
	if err != nil {
		// Try YAML manifest
		manifestURL = fmt.Sprintf("https://unpkg.com/%s@%s/manifest.yaml", s.config.Package, pkgInfo.Version)
		manifest, err = s.loadManifest(ctx, manifestURL)
		if err != nil {
			// Fall back to loading from standard paths
			return s.loadFromStandardPaths(ctx, pkgInfo)
		}
	}

	// Load items from manifest
	baseURL := fmt.Sprintf("https://unpkg.com/%s@%s", s.config.Package, pkgInfo.Version)

	// Load skills
	for _, item := range manifest.Skills {
		skill, err := s.loadSkillFromManifest(ctx, baseURL, item)
		if err != nil {
			fmt.Printf("Warning: failed to load skill %s: %v\n", item.ID, err)
			continue
		}
		s.skills[item.ID] = skill
	}

	// Load agents
	for _, item := range manifest.Agents {
		agent, err := s.loadAgentFromManifest(ctx, baseURL, item)
		if err != nil {
			fmt.Printf("Warning: failed to load agent %s: %v\n", item.ID, err)
			continue
		}
		s.agents[item.ID] = agent
	}

	// Load commands
	for _, item := range manifest.Commands {
		cmd, err := s.loadCommandFromManifest(ctx, baseURL, item)
		if err != nil {
			fmt.Printf("Warning: failed to load command %s: %v\n", item.ID, err)
			continue
		}
		s.commands[item.ID] = cmd
	}

	return nil
}

// getPackageInfo gets package info from npm registry
func (s *NPMSource) getPackageInfo(ctx context.Context) (*NPMPackageInfo, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s/latest", s.config.Package)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.AuthToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry error: %s", resp.Status)
	}

	var pkgInfo NPMPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&pkgInfo); err != nil {
		return nil, fmt.Errorf("decode package info: %w", err)
	}

	return &pkgInfo, nil
}

// loadManifest loads the registry manifest
func (s *NPMSource) loadManifest(ctx context.Context, url string) (*RegistryManifest, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest not found: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var manifest RegistryManifest

	// Try JSON first
	if err := json.Unmarshal(body, &manifest); err != nil {
		// Try YAML
		if err := yaml.Unmarshal(body, &manifest); err != nil {
			return nil, fmt.Errorf("parse manifest: %w", err)
		}
	}

	return &manifest, nil
}

// loadFromStandardPaths loads components from standard paths when no manifest exists
func (s *NPMSource) loadFromStandardPaths(ctx context.Context, pkgInfo *NPMPackageInfo) error {
	baseURL := fmt.Sprintf("https://unpkg.com/%s@%s", s.config.Package, pkgInfo.Version)

	// Try loading from standard directories
	// Skills: /skills/
	// Agents: /agents/
	// Commands: /commands/

	// Load skills
	skillsIndexURL := baseURL + "/skills/index.json"
	if skills, err := s.loadSkillsIndex(ctx, skillsIndexURL, baseURL); err == nil {
		for id, skill := range skills {
			s.skills[id] = skill
		}
	}

	// Load agents
	agentsIndexURL := baseURL + "/agents/index.json"
	if agents, err := s.loadAgentsIndex(ctx, agentsIndexURL, baseURL); err == nil {
		for id, agent := range agents {
			s.agents[id] = agent
		}
	}

	// Load commands
	commandsIndexURL := baseURL + "/commands/index.json"
	if commands, err := s.loadCommandsIndex(ctx, commandsIndexURL, baseURL); err == nil {
		for id, cmd := range commands {
			s.commands[id] = cmd
		}
	}

	return nil
}

// loadSkillsIndex loads skills from an index file
func (s *NPMSource) loadSkillsIndex(ctx context.Context, indexURL, baseURL string) (map[string]*Skill, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", indexURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("index not found")
	}

	var items []ManifestItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	skills := make(map[string]*Skill)
	for _, item := range items {
		skill, err := s.loadSkillFromManifest(ctx, baseURL, item)
		if err != nil {
			continue
		}
		skills[item.ID] = skill
	}

	return skills, nil
}

// loadAgentsIndex loads agents from an index file
func (s *NPMSource) loadAgentsIndex(ctx context.Context, indexURL, baseURL string) (map[string]*Agent, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", indexURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("index not found")
	}

	var items []ManifestItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	agents := make(map[string]*Agent)
	for _, item := range items {
		agent, err := s.loadAgentFromManifest(ctx, baseURL, item)
		if err != nil {
			continue
		}
		agents[item.ID] = agent
	}

	return agents, nil
}

// loadCommandsIndex loads commands from an index file
func (s *NPMSource) loadCommandsIndex(ctx context.Context, indexURL, baseURL string) (map[string]*Command, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", indexURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("index not found")
	}

	var items []ManifestItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	commands := make(map[string]*Command)
	for _, item := range items {
		cmd, err := s.loadCommandFromManifest(ctx, baseURL, item)
		if err != nil {
			continue
		}
		commands[item.ID] = cmd
	}

	return commands, nil
}

// loadSkillFromManifest loads a skill from a manifest item
func (s *NPMSource) loadSkillFromManifest(ctx context.Context, baseURL string, item ManifestItem) (*Skill, error) {
	var contentURL string
	if item.Path != "" {
		contentURL = baseURL + "/" + item.Path
	} else {
		contentURL = baseURL + "/skills/" + item.ID + "/SKILL.md"
	}

	content, err := s.fetchContent(ctx, contentURL)
	if err != nil {
		return nil, err
	}

	// Parse frontmatter if present
	var skillContent string
	frontmatter, markdown, parseErr := parseSkillFrontmatter(content)
	if parseErr == nil {
		skillContent = markdown
		// Use frontmatter values if manifest values are empty
		if item.Name == "" {
			item.Name = frontmatter.Name
		}
		if item.Description == "" {
			item.Description = frontmatter.Description
		}
		if item.Version == "" {
			item.Version = frontmatter.Version
		}
		if item.Author == "" {
			item.Author = frontmatter.Author
		}
	} else {
		skillContent = content
	}

	return &Skill{
		ID:          item.ID,
		Source:      s.config.Name,
		Name:        item.Name,
		Description: item.Description,
		Keywords:    item.Keywords,
		Version:     item.Version,
		Author:      item.Author,
		Content:     skillContent,
		SourceURL:   contentURL,
	}, nil
}

// loadAgentFromManifest loads an agent from a manifest item
func (s *NPMSource) loadAgentFromManifest(ctx context.Context, baseURL string, item ManifestItem) (*Agent, error) {
	var contentURL string
	if item.Path != "" {
		contentURL = baseURL + "/" + item.Path
	} else {
		contentURL = baseURL + "/agents/" + item.ID + ".md"
	}

	content, err := s.fetchContent(ctx, contentURL)
	if err != nil {
		return nil, err
	}

	// Parse frontmatter
	frontmatter, markdown, parseErr := parseAgentFrontmatter(content)
	if parseErr != nil {
		// If no frontmatter, use manifest values
		return &Agent{
			ID:          item.ID,
			Source:      s.config.Name,
			Name:        item.Name,
			Description: item.Description,
			Content:     content,
			SourceURL:   contentURL,
		}, nil
	}

	// Parse tools
	var tools []string
	if frontmatter.Tools != "" {
		for _, tool := range strings.Split(frontmatter.Tools, ",") {
			trimmed := strings.TrimSpace(tool)
			if trimmed != "" {
				tools = append(tools, trimmed)
			}
		}
	}

	agent := &Agent{
		ID:               item.ID,
		Source:           s.config.Name,
		Name:             frontmatter.Name,
		Description:      frontmatter.Description,
		Model:            frontmatter.Model,
		PrimarySkill:     frontmatter.PrimarySkill,
		SupportingSkills: frontmatter.SupportingSkills,
		Tools:            tools,
		Content:          markdown,
		SourceURL:        contentURL,
	}

	if frontmatter.Routing != nil {
		agent.Routing = &AgentRouting{
			DeferToSkill:  frontmatter.Routing.DeferToSkill,
			FallbackModel: frontmatter.Routing.FallbackModel,
		}
	}

	if frontmatter.Orchestration != nil {
		agent.Orchestration = &Orchestration{
			Enabled:          frontmatter.Orchestration.Enabled,
			DefaultPhases:    frontmatter.Orchestration.DefaultPhases,
			RoutingStrategy:  frontmatter.Orchestration.RoutingStrategy,
			CostOptimization: frontmatter.Orchestration.CostOptimization,
		}
	}

	return agent, nil
}

// loadCommandFromManifest loads a command from a manifest item
func (s *NPMSource) loadCommandFromManifest(ctx context.Context, baseURL string, item ManifestItem) (*Command, error) {
	var contentURL string
	if item.Path != "" {
		contentURL = baseURL + "/" + item.Path
	} else {
		contentURL = baseURL + "/commands/" + item.ID + ".md"
	}

	content, err := s.fetchContent(ctx, contentURL)
	if err != nil {
		return nil, err
	}

	return &Command{
		ID:          item.ID,
		Source:      s.config.Name,
		Name:        item.Name,
		Description: item.Description,
		Content:     content,
		SourceURL:   contentURL,
	}, nil
}

// fetchContent fetches content from a URL
func (s *NPMSource) fetchContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(body), nil
}
