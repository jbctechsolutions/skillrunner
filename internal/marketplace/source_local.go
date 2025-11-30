package marketplace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// LocalSource implements Source for local filesystem marketplaces
type LocalSource struct {
	config SourceConfig
	skills map[string]*Skill
	agents map[string]*Agent
	mu     sync.RWMutex
}

// NewLocalSource creates a new local filesystem source
func NewLocalSource(config SourceConfig) (*LocalSource, error) {
	if config.Type != SourceTypeLocal {
		return nil, fmt.Errorf("invalid source type: %s (expected local)", config.Type)
	}

	// If no path specified, use default jbctech-marketplace location
	if config.Path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		config.Path = filepath.Join(home, ".repos", "github.com", "jbctechsolutions", "jbctech-claude-marketplace")
	}

	source := &LocalSource{
		config: config,
		skills: make(map[string]*Skill),
		agents: make(map[string]*Agent),
	}

	// Initial load
	if err := source.Refresh(context.Background()); err != nil {
		return nil, err
	}

	return source, nil
}

// Name returns the source identifier
func (s *LocalSource) Name() string {
	return s.config.Name
}

// Type returns the source type
func (s *LocalSource) Type() SourceType {
	return SourceTypeLocal
}

// Priority returns the source priority
func (s *LocalSource) Priority() int {
	return s.config.Priority
}

// ListSkills returns all available skills
func (s *LocalSource) ListSkills(ctx context.Context) ([]*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make([]*Skill, 0, len(s.skills))
	for _, skill := range s.skills {
		skills = append(skills, skill)
	}
	return skills, nil
}

// GetSkill retrieves a specific skill by ID
func (s *LocalSource) GetSkill(ctx context.Context, id string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, ok := s.skills[id]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", id)
	}
	return skill, nil
}

// ListAgents returns all available agents
func (s *LocalSource) ListAgents(ctx context.Context) ([]*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*Agent, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	return agents, nil
}

// GetAgent retrieves a specific agent by ID
func (s *LocalSource) GetAgent(ctx context.Context, id string) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return agent, nil
}

// ListCommands returns all available commands
func (s *LocalSource) ListCommands(ctx context.Context) ([]*Command, error) {
	// TODO: Implement command loading
	return nil, nil
}

// GetCommand retrieves a specific command by ID
func (s *LocalSource) GetCommand(ctx context.Context, id string) (*Command, error) {
	return nil, fmt.Errorf("command not found: %s", id)
}

// Search searches for components matching a query
func (s *LocalSource) Search(ctx context.Context, query string, types []ComponentType) ([]*SearchResult, error) {
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
func (s *LocalSource) calculateScore(query, name, description string, keywords []string) float64 {
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

// Refresh reloads the source data
func (s *LocalSource) Refresh(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing data
	s.skills = make(map[string]*Skill)
	s.agents = make(map[string]*Agent)

	// Load skills
	skillsPath := filepath.Join(s.config.Path, "skills")
	if err := s.loadSkills(skillsPath); err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	// Load agents
	agentsPath := filepath.Join(s.config.Path, "agents")
	if err := s.loadAgents(agentsPath); err != nil {
		// Warn but don't fail - agents directory may not exist
		fmt.Printf("Warning: %v\n", err)
	}

	return nil
}

// IsHealthy checks if the source is accessible
func (s *LocalSource) IsHealthy(ctx context.Context) bool {
	_, err := os.Stat(s.config.Path)
	return err == nil
}

// loadSkills loads all skills from the skills directory
func (s *LocalSource) loadSkills(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillPath := filepath.Join(path, skillID, "SKILL.md")

		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			continue
		}

		skill, err := s.loadSkill(skillPath, skillID)
		if err != nil {
			fmt.Printf("Warning: failed to load skill %s: %v\n", skillID, err)
			continue
		}

		s.skills[skillID] = skill
	}

	if len(s.skills) == 0 {
		return fmt.Errorf("no skills found in %s", path)
	}

	return nil
}

// loadSkill loads a single skill from a SKILL.md file
func (s *LocalSource) loadSkill(path string, skillID string) (*Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	frontmatter, markdown, err := parseSkillFrontmatter(string(content))
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
		SourceURL:   "file://" + path,
	}

	return skill, nil
}

// loadAgents loads all agents from the agents directory
func (s *LocalSource) loadAgents(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("agents directory not found: %s", path)
	}

	categories, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read agents directory: %w", err)
	}

	for _, category := range categories {
		if !category.IsDir() {
			continue
		}

		categoryPath := filepath.Join(path, category.Name())
		agents, err := os.ReadDir(categoryPath)
		if err != nil {
			fmt.Printf("Warning: failed to read category %s: %v\n", category.Name(), err)
			continue
		}

		for _, agentFile := range agents {
			if agentFile.IsDir() || !strings.HasSuffix(agentFile.Name(), ".md") {
				continue
			}

			agentPath := filepath.Join(categoryPath, agentFile.Name())
			agentID := strings.TrimSuffix(agentFile.Name(), ".md")
			categoryAgentID := filepath.Join(category.Name(), agentID)

			agent, err := s.loadAgent(agentPath, categoryAgentID)
			if err != nil {
				fmt.Printf("Warning: failed to load agent %s: %v\n", categoryAgentID, err)
				continue
			}

			s.agents[categoryAgentID] = agent
		}
	}

	return nil
}

// loadAgent loads a single agent from a markdown file
func (s *LocalSource) loadAgent(path string, agentID string) (*Agent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent file: %w", err)
	}

	frontmatter, markdown, err := parseAgentFrontmatter(string(content))
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
		SourceURL:        "file://" + path,
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

// Frontmatter types for parsing

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Keywords    string `yaml:"keywords"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author"`
	LastUpdated string `yaml:"last_updated"`
}

type agentFrontmatterLocal struct {
	Name             string                    `yaml:"name"`
	Description      string                    `yaml:"description"`
	Model            string                    `yaml:"model"`
	PrimarySkill     string                    `yaml:"primary_skill"`
	SupportingSkills []string                  `yaml:"supporting_skills"`
	Tools            string                    `yaml:"tools"`
	Routing          *agentRoutingFrontmatter  `yaml:"routing"`
	Orchestration    *orchestrationFrontmatter `yaml:"orchestration"`
}

type agentRoutingFrontmatter struct {
	DeferToSkill  bool   `yaml:"defer_to_skill"`
	FallbackModel string `yaml:"fallback_model"`
}

type orchestrationFrontmatter struct {
	Enabled          bool     `yaml:"enabled"`
	DefaultPhases    []string `yaml:"default_phases"`
	RoutingStrategy  string   `yaml:"routing_strategy"`
	CostOptimization bool     `yaml:"cost_optimization"`
}

func parseSkillFrontmatter(content string) (*skillFrontmatter, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, "", fmt.Errorf("no frontmatter found")
	}

	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return nil, "", fmt.Errorf("unterminated frontmatter")
	}

	frontmatterText := content[4 : endIdx+4]
	markdown := strings.TrimSpace(content[endIdx+9:])

	var frontmatter skillFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatterText), &frontmatter); err != nil {
		return nil, "", fmt.Errorf("parse YAML: %w", err)
	}

	return &frontmatter, markdown, nil
}

func parseAgentFrontmatter(content string) (*agentFrontmatterLocal, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, "", fmt.Errorf("no frontmatter found")
	}

	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return nil, "", fmt.Errorf("unterminated frontmatter")
	}

	frontmatterText := content[4 : endIdx+4]
	markdown := strings.TrimSpace(content[endIdx+9:])

	var frontmatter agentFrontmatterLocal
	if err := yaml.Unmarshal([]byte(frontmatterText), &frontmatter); err != nil {
		return nil, "", fmt.Errorf("parse YAML: %w", err)
	}

	return &frontmatter, markdown, nil
}
