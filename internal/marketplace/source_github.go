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
)

// GitHubSource implements Source for GitHub repository marketplaces
type GitHubSource struct {
	config     SourceConfig
	skills     map[string]*Skill
	agents     map[string]*Agent
	mu         sync.RWMutex
	httpClient *http.Client
	cache      *githubCache
}

// githubCache stores cached API responses
type githubCache struct {
	skills    map[string]*Skill
	agents    map[string]*Agent
	expiresAt time.Time
}

// GitHubFile represents a file in a GitHub repo
type GitHubFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "file" or "dir"
	DownloadURL string `json:"download_url"`
	URL         string `json:"url"`
}

// NewGitHubSource creates a new GitHub repository source
func NewGitHubSource(config SourceConfig) (*GitHubSource, error) {
	if config.Type != SourceTypeGitHub {
		return nil, fmt.Errorf("invalid source type: %s (expected github)", config.Type)
	}

	if config.Owner == "" || config.Repo == "" {
		return nil, fmt.Errorf("github source requires owner and repo")
	}

	if config.Branch == "" {
		config.Branch = "main"
	}

	source := &GitHubSource{
		config: config,
		skills: make(map[string]*Skill),
		agents: make(map[string]*Agent),
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
func (s *GitHubSource) Name() string {
	return s.config.Name
}

// Type returns the source type
func (s *GitHubSource) Type() SourceType {
	return SourceTypeGitHub
}

// Priority returns the source priority
func (s *GitHubSource) Priority() int {
	return s.config.Priority
}

// ListSkills returns all available skills
func (s *GitHubSource) ListSkills(ctx context.Context) ([]*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make([]*Skill, 0, len(s.skills))
	for _, skill := range s.skills {
		skills = append(skills, skill)
	}
	return skills, nil
}

// GetSkill retrieves a specific skill by ID
func (s *GitHubSource) GetSkill(ctx context.Context, id string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, ok := s.skills[id]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", id)
	}
	return skill, nil
}

// ListAgents returns all available agents
func (s *GitHubSource) ListAgents(ctx context.Context) ([]*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	return agents, nil
}

// GetAgent retrieves a specific agent by ID
func (s *GitHubSource) GetAgent(ctx context.Context, id string) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return agent, nil
}

// ListCommands returns all available commands
func (s *GitHubSource) ListCommands(ctx context.Context) ([]*Command, error) {
	// TODO: Implement command loading from GitHub
	return nil, nil
}

// GetCommand retrieves a specific command by ID
func (s *GitHubSource) GetCommand(ctx context.Context, id string) (*Command, error) {
	return nil, fmt.Errorf("command not found: %s", id)
}

// Search searches for components matching a query
func (s *GitHubSource) Search(ctx context.Context, query string, types []ComponentType) ([]*SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*SearchResult

	// Check if we should search skills
	searchSkills := len(types) == 0
	searchAgents := len(types) == 0
	for _, t := range types {
		if t == ComponentTypeSkill {
			searchSkills = true
		}
		if t == ComponentTypeAgent {
			searchAgents = true
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

	return results, nil
}

// calculateScore calculates a simple relevance score
func (s *GitHubSource) calculateScore(query, name, description string, keywords []string) float64 {
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

// Refresh reloads the source data from GitHub
func (s *GitHubSource) Refresh(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check cache
	if s.cache != nil && time.Now().Before(s.cache.expiresAt) {
		s.skills = s.cache.skills
		s.agents = s.cache.agents
		return nil
	}

	// Clear existing data
	s.skills = make(map[string]*Skill)
	s.agents = make(map[string]*Agent)

	// Load skills from GitHub
	if err := s.loadSkillsFromGitHub(ctx); err != nil {
		// Warn but don't fail - skills directory may not exist
		fmt.Printf("Warning: failed to load skills from GitHub: %v\n", err)
	}

	// Load agents from GitHub
	if err := s.loadAgentsFromGitHub(ctx); err != nil {
		// Warn but don't fail - agents directory may not exist
		fmt.Printf("Warning: failed to load agents from GitHub: %v\n", err)
	}

	// Update cache
	cacheTTL := time.Duration(s.config.CacheTTL) * time.Second
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute // Default cache TTL
	}
	s.cache = &githubCache{
		skills:    s.skills,
		agents:    s.agents,
		expiresAt: time.Now().Add(cacheTTL),
	}

	return nil
}

// IsHealthy checks if the source is accessible
func (s *GitHubSource) IsHealthy(ctx context.Context) bool {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", s.config.Owner, s.config.Repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	if s.config.AuthToken != "" {
		req.Header.Set("Authorization", "token "+s.config.AuthToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// loadSkillsFromGitHub loads skills from the GitHub repository
func (s *GitHubSource) loadSkillsFromGitHub(ctx context.Context) error {
	// Get contents of skills directory
	files, err := s.listDirectory(ctx, "skills")
	if err != nil {
		return fmt.Errorf("list skills directory: %w", err)
	}

	for _, file := range files {
		if file.Type != "dir" {
			continue
		}

		skillID := file.Name
		skillPath := file.Path + "/SKILL.md"

		// Get SKILL.md content
		content, err := s.getFileContent(ctx, skillPath)
		if err != nil {
			fmt.Printf("Warning: failed to load skill %s: %v\n", skillID, err)
			continue
		}

		skill, err := s.parseSkill(content, skillID)
		if err != nil {
			fmt.Printf("Warning: failed to parse skill %s: %v\n", skillID, err)
			continue
		}

		s.skills[skillID] = skill
	}

	return nil
}

// loadAgentsFromGitHub loads agents from the GitHub repository
func (s *GitHubSource) loadAgentsFromGitHub(ctx context.Context) error {
	// Get contents of agents directory
	categories, err := s.listDirectory(ctx, "agents")
	if err != nil {
		return fmt.Errorf("list agents directory: %w", err)
	}

	for _, category := range categories {
		if category.Type != "dir" {
			continue
		}

		// Get agents in this category
		agentFiles, err := s.listDirectory(ctx, category.Path)
		if err != nil {
			fmt.Printf("Warning: failed to list category %s: %v\n", category.Name, err)
			continue
		}

		for _, agentFile := range agentFiles {
			if agentFile.Type != "file" || !strings.HasSuffix(agentFile.Name, ".md") {
				continue
			}

			agentID := strings.TrimSuffix(agentFile.Name, ".md")
			categoryAgentID := category.Name + "/" + agentID

			// Get agent content
			content, err := s.getFileContent(ctx, agentFile.Path)
			if err != nil {
				fmt.Printf("Warning: failed to load agent %s: %v\n", categoryAgentID, err)
				continue
			}

			agent, err := s.parseAgent(content, categoryAgentID)
			if err != nil {
				fmt.Printf("Warning: failed to parse agent %s: %v\n", categoryAgentID, err)
				continue
			}

			s.agents[categoryAgentID] = agent
		}
	}

	return nil
}

// listDirectory lists contents of a directory in the GitHub repo
func (s *GitHubSource) listDirectory(ctx context.Context, path string) ([]GitHubFile, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		s.config.Owner, s.config.Repo, path, s.config.Branch)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.config.AuthToken != "" {
		req.Header.Set("Authorization", "token "+s.config.AuthToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("directory not found: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var files []GitHubFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return files, nil
}

// getFileContent retrieves content of a file from GitHub
func (s *GitHubSource) getFileContent(ctx context.Context, path string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		s.config.Owner, s.config.Repo, s.config.Branch, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	if s.config.AuthToken != "" {
		req.Header.Set("Authorization", "token "+s.config.AuthToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(body), nil
}

// parseSkill parses skill content into a Skill struct
func (s *GitHubSource) parseSkill(content string, skillID string) (*Skill, error) {
	frontmatter, markdown, err := parseSkillFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	// Parse keywords into slice
	var keywords []string
	if frontmatter.Keywords != "" {
		for _, kw := range strings.Split(frontmatter.Keywords, ",") {
			trimmed := strings.TrimSpace(kw)
			if trimmed != "" {
				keywords = append(keywords, trimmed)
			}
		}
	}

	skill := &Skill{
		ID:          skillID,
		Source:      s.config.Name,
		Name:        frontmatter.Name,
		Description: frontmatter.Description,
		Keywords:    keywords,
		Version:     frontmatter.Version,
		Author:      frontmatter.Author,
		LastUpdated: frontmatter.LastUpdated,
		Content:     markdown,
		SourceURL:   fmt.Sprintf("https://github.com/%s/%s/tree/%s/skills/%s", s.config.Owner, s.config.Repo, s.config.Branch, skillID),
	}

	return skill, nil
}

// parseAgent parses agent content into an Agent struct
func (s *GitHubSource) parseAgent(content string, agentID string) (*Agent, error) {
	frontmatter, markdown, err := parseAgentFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
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
		ID:               agentID,
		Source:           s.config.Name,
		Name:             frontmatter.Name,
		Description:      frontmatter.Description,
		Model:            frontmatter.Model,
		PrimarySkill:     frontmatter.PrimarySkill,
		SupportingSkills: frontmatter.SupportingSkills,
		Tools:            tools,
		Content:          markdown,
		SourceURL:        fmt.Sprintf("https://github.com/%s/%s/tree/%s/agents/%s.md", s.config.Owner, s.config.Repo, s.config.Branch, agentID),
	}

	// Convert routing
	if frontmatter.Routing != nil {
		agent.Routing = &AgentRouting{
			DeferToSkill:  frontmatter.Routing.DeferToSkill,
			FallbackModel: frontmatter.Routing.FallbackModel,
		}
	}

	// Convert orchestration
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
