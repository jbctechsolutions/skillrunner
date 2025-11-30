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

// HTTPSource implements Source for generic HTTP/HTTPS endpoints
type HTTPSource struct {
	config     SourceConfig
	skills     map[string]*Skill
	agents     map[string]*Agent
	commands   map[string]*Command
	mu         sync.RWMutex
	httpClient *http.Client
	cache      *httpCache
}

// httpCache stores cached data
type httpCache struct {
	skills    map[string]*Skill
	agents    map[string]*Agent
	commands  map[string]*Command
	expiresAt time.Time
}

// HTTPManifest represents the manifest from an HTTP source
type HTTPManifest struct {
	Version     string         `json:"version" yaml:"version"`
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Skills      []ManifestItem `json:"skills" yaml:"skills"`
	Agents      []ManifestItem `json:"agents" yaml:"agents"`
	Commands    []ManifestItem `json:"commands" yaml:"commands"`
}

// NewHTTPSource creates a new HTTP-based source
func NewHTTPSource(config SourceConfig) (*HTTPSource, error) {
	if config.Type != SourceTypeHTTP && config.Type != SourceTypeRegistry {
		return nil, fmt.Errorf("invalid source type: %s (expected http or registry)", config.Type)
	}

	if config.URL == "" {
		return nil, fmt.Errorf("http source requires URL")
	}

	source := &HTTPSource{
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
func (s *HTTPSource) Name() string {
	return s.config.Name
}

// Type returns the source type
func (s *HTTPSource) Type() SourceType {
	return s.config.Type
}

// Priority returns the source priority
func (s *HTTPSource) Priority() int {
	return s.config.Priority
}

// ListSkills returns all available skills
func (s *HTTPSource) ListSkills(ctx context.Context) ([]*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make([]*Skill, 0, len(s.skills))
	for _, skill := range s.skills {
		skills = append(skills, skill)
	}
	return skills, nil
}

// GetSkill retrieves a specific skill by ID
func (s *HTTPSource) GetSkill(ctx context.Context, id string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, ok := s.skills[id]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", id)
	}
	return skill, nil
}

// ListAgents returns all available agents
func (s *HTTPSource) ListAgents(ctx context.Context) ([]*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	return agents, nil
}

// GetAgent retrieves a specific agent by ID
func (s *HTTPSource) GetAgent(ctx context.Context, id string) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return agent, nil
}

// ListCommands returns all available commands
func (s *HTTPSource) ListCommands(ctx context.Context) ([]*Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	commands := make([]*Command, 0, len(s.commands))
	for _, cmd := range s.commands {
		commands = append(commands, cmd)
	}
	return commands, nil
}

// GetCommand retrieves a specific command by ID
func (s *HTTPSource) GetCommand(ctx context.Context, id string) (*Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cmd, ok := s.commands[id]
	if !ok {
		return nil, fmt.Errorf("command not found: %s", id)
	}
	return cmd, nil
}

// Search searches for components matching a query
func (s *HTTPSource) Search(ctx context.Context, query string, types []ComponentType) ([]*SearchResult, error) {
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
func (s *HTTPSource) calculateScore(query, name, description string, keywords []string) float64 {
	query = strings.ToLower(query)
	name = strings.ToLower(name)
	description = strings.ToLower(description)

	score := 0.0

	// Name match (highest weight)
	if strings.Contains(name, query) {
		score += 0.5
		if name == query {
			score += 0.3
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

// Refresh reloads the source data from the HTTP endpoint
func (s *HTTPSource) Refresh(ctx context.Context) error {
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

	// Load manifest
	manifest, err := s.loadManifest(ctx)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Get base URL (remove manifest.json or manifest.yaml from URL)
	baseURL := s.config.URL
	if strings.HasSuffix(baseURL, "/manifest.json") {
		baseURL = strings.TrimSuffix(baseURL, "/manifest.json")
	} else if strings.HasSuffix(baseURL, "/manifest.yaml") {
		baseURL = strings.TrimSuffix(baseURL, "/manifest.yaml")
	}

	// Load skills
	for _, item := range manifest.Skills {
		skill, err := s.loadSkill(ctx, baseURL, item)
		if err != nil {
			fmt.Printf("Warning: failed to load skill %s: %v\n", item.ID, err)
			continue
		}
		s.skills[item.ID] = skill
	}

	// Load agents
	for _, item := range manifest.Agents {
		agent, err := s.loadAgent(ctx, baseURL, item)
		if err != nil {
			fmt.Printf("Warning: failed to load agent %s: %v\n", item.ID, err)
			continue
		}
		s.agents[item.ID] = agent
	}

	// Load commands
	for _, item := range manifest.Commands {
		cmd, err := s.loadCommand(ctx, baseURL, item)
		if err != nil {
			fmt.Printf("Warning: failed to load command %s: %v\n", item.ID, err)
			continue
		}
		s.commands[item.ID] = cmd
	}

	// Update cache
	cacheTTL := time.Duration(s.config.CacheTTL) * time.Second
	if cacheTTL == 0 {
		cacheTTL = 10 * time.Minute
	}
	s.cache = &httpCache{
		skills:    s.skills,
		agents:    s.agents,
		commands:  s.commands,
		expiresAt: time.Now().Add(cacheTTL),
	}

	return nil
}

// IsHealthy checks if the source is accessible
func (s *HTTPSource) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "HEAD", s.config.URL, nil)
	if err != nil {
		return false
	}

	if s.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.AuthToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// loadManifest loads the manifest from the HTTP endpoint
func (s *HTTPSource) loadManifest(ctx context.Context) (*HTTPManifest, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.config.URL, nil)
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
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var manifest HTTPManifest

	// Try JSON first
	if err := json.Unmarshal(body, &manifest); err != nil {
		// Try YAML
		if err := yaml.Unmarshal(body, &manifest); err != nil {
			return nil, fmt.Errorf("parse manifest: %w", err)
		}
	}

	return &manifest, nil
}

// loadSkill loads a skill from the manifest
func (s *HTTPSource) loadSkill(ctx context.Context, baseURL string, item ManifestItem) (*Skill, error) {
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

// loadAgent loads an agent from the manifest
func (s *HTTPSource) loadAgent(ctx context.Context, baseURL string, item ManifestItem) (*Agent, error) {
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

// loadCommand loads a command from the manifest
func (s *HTTPSource) loadCommand(ctx context.Context, baseURL string, item ManifestItem) (*Command, error) {
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
func (s *HTTPSource) fetchContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	if s.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.AuthToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(body), nil
}
