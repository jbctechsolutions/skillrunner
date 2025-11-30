package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MarketplaceSkill represents a Claude Code skill from the marketplace
type MarketplaceSkill struct {
	ID          string
	Name        string
	Description string
	Keywords    string
	Version     string
	Author      string
	LastUpdated string
	Content     string // Full markdown content
}

// MarketplaceAgent represents a Claude Code agent from the marketplace
type MarketplaceAgent struct {
	ID               string
	Name             string
	Description      string
	Model            string
	PrimarySkill     string
	SupportingSkills []string
	Tools            []string
	Routing          *AgentRouting
	Orchestration    *OrchestrationConfig
	Content          string // Full markdown content
}

// AgentRouting represents routing configuration for an agent
type AgentRouting struct {
	DeferToSkill  bool   `yaml:"defer_to_skill"`
	FallbackModel string `yaml:"fallback_model"`
}

// OrchestrationConfig represents orchestration configuration for an agent
type OrchestrationConfig struct {
	Enabled          bool     `yaml:"enabled"`
	DefaultPhases    []string `yaml:"default_phases"`
	RoutingStrategy  string   `yaml:"routing_strategy"`
	CostOptimization bool     `yaml:"cost_optimization"`
}

// MarketplaceLoader loads Claude Code skills and agents from jbctech-claude-marketplace
type MarketplaceLoader struct {
	marketplacePath string
	agentsPath      string
	skills          map[string]*MarketplaceSkill
	agents          map[string]*MarketplaceAgent
}

// NewMarketplaceLoader creates a new marketplace loader
func NewMarketplaceLoader(marketplacePath string) (*MarketplaceLoader, error) {
	var agentsPath string

	if marketplacePath == "" {
		// Default to jbctech-claude-marketplace
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		marketplaceRoot := filepath.Join(home, ".repos", "github.com", "jbctechsolutions", "jbctech-claude-marketplace")
		marketplacePath = filepath.Join(marketplaceRoot, "skills")
		agentsPath = filepath.Join(marketplaceRoot, "agents")
	} else {
		// Derive agents path from provided marketplace path
		marketplaceRoot := filepath.Dir(marketplacePath)
		agentsPath = filepath.Join(marketplaceRoot, "agents")
	}

	loader := &MarketplaceLoader{
		marketplacePath: marketplacePath,
		agentsPath:      agentsPath,
		skills:          make(map[string]*MarketplaceSkill),
		agents:          make(map[string]*MarketplaceAgent),
	}

	// Load all skills
	if err := loader.loadAllSkills(); err != nil {
		return nil, err
	}

	// Load all agents
	if err := loader.loadAllAgents(); err != nil {
		// Log warning but don't fail - agents may not exist in all environments
		fmt.Printf("Warning: failed to load agents: %v\n", err)
	}

	return loader, nil
}

// loadAllSkills scans the marketplace and loads all skills
func (ml *MarketplaceLoader) loadAllSkills() error {
	entries, err := os.ReadDir(ml.marketplacePath)
	if err != nil {
		return fmt.Errorf("read marketplace directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillPath := filepath.Join(ml.marketplacePath, skillID, "SKILL.md")

		// Check if SKILL.md exists
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			continue // Skip directories without SKILL.md
		}

		skill, err := ml.loadSkill(skillPath, skillID)
		if err != nil {
			// Log warning but continue loading other skills
			fmt.Printf("Warning: failed to load skill %s: %v\n", skillID, err)
			continue
		}

		ml.skills[skillID] = skill
	}

	if len(ml.skills) == 0 {
		return fmt.Errorf("no skills found in %s", ml.marketplacePath)
	}

	return nil
}

// loadSkill loads a single skill from SKILL.md
func (ml *MarketplaceLoader) loadSkill(path string, skillID string) (*MarketplaceSkill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	// Parse YAML frontmatter
	frontmatter, markdown, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	skill := &MarketplaceSkill{
		ID:          skillID,
		Name:        frontmatter.Name,
		Description: frontmatter.Description,
		Keywords:    frontmatter.Keywords,
		Version:     frontmatter.Version,
		Author:      frontmatter.Author,
		LastUpdated: frontmatter.LastUpdated,
		Content:     markdown,
	}

	return skill, nil
}

// GetSkill retrieves a loaded skill by ID
func (ml *MarketplaceLoader) GetSkill(skillID string) (*MarketplaceSkill, error) {
	skill, ok := ml.skills[skillID]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}
	return skill, nil
}

// ListSkills returns all loaded skills
func (ml *MarketplaceLoader) ListSkills() []*MarketplaceSkill {
	skills := make([]*MarketplaceSkill, 0, len(ml.skills))
	for _, skill := range ml.skills {
		skills = append(skills, skill)
	}
	return skills
}

// SearchSkills searches for skills matching a query
func (ml *MarketplaceLoader) SearchSkills(query string) []*MarketplaceSkill {
	query = strings.ToLower(query)
	var results []*MarketplaceSkill

	for _, skill := range ml.skills {
		// Search in name, description, and keywords
		if strings.Contains(strings.ToLower(skill.Name), query) ||
			strings.Contains(strings.ToLower(skill.Description), query) ||
			strings.Contains(strings.ToLower(skill.Keywords), query) {
			results = append(results, skill)
		}
	}

	return results
}

// BuildPromptWithSkill creates a prompt that includes skill knowledge
func (ml *MarketplaceLoader) BuildPromptWithSkill(skillID string, userRequest string) (string, error) {
	skill, err := ml.GetSkill(skillID)
	if err != nil {
		return "", err
	}

	// Build prompt with skill context
	prompt := fmt.Sprintf(`You are an expert assistant with the following specialized knowledge:

# %s

%s

---

Using the above expertise, please help with the following request:

%s

Provide a detailed, professional response based on the skill knowledge above.`,
		skill.Name,
		skill.Content,
		userRequest)

	return prompt, nil
}

// loadAllAgents scans the marketplace agents directory and loads all agents
func (ml *MarketplaceLoader) loadAllAgents() error {
	// Check if agents directory exists
	if _, err := os.Stat(ml.agentsPath); os.IsNotExist(err) {
		return fmt.Errorf("agents directory not found: %s", ml.agentsPath)
	}

	// Walk through all categories in agents directory
	categories, err := os.ReadDir(ml.agentsPath)
	if err != nil {
		return fmt.Errorf("read agents directory: %w", err)
	}

	for _, category := range categories {
		if !category.IsDir() {
			continue
		}

		categoryPath := filepath.Join(ml.agentsPath, category.Name())
		agents, err := os.ReadDir(categoryPath)
		if err != nil {
			fmt.Printf("Warning: failed to read category %s: %v\n", category.Name(), err)
			continue
		}

		// Look for agent markdown files in category
		for _, agentFile := range agents {
			if agentFile.IsDir() || !strings.HasSuffix(agentFile.Name(), ".md") {
				continue
			}

			agentPath := filepath.Join(categoryPath, agentFile.Name())
			agentID := strings.TrimSuffix(agentFile.Name(), ".md")
			categoryAgentID := filepath.Join(category.Name(), agentID)

			agent, err := ml.loadAgent(agentPath, categoryAgentID)
			if err != nil {
				// Log warning but continue loading other agents
				fmt.Printf("Warning: failed to load agent %s: %v\n", categoryAgentID, err)
				continue
			}

			ml.agents[categoryAgentID] = agent
		}
	}

	if len(ml.agents) == 0 {
		return fmt.Errorf("no agents found in %s", ml.agentsPath)
	}

	return nil
}

// loadAgent loads a single agent from an agent markdown file
func (ml *MarketplaceLoader) loadAgent(path string, agentID string) (*MarketplaceAgent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent file: %w", err)
	}

	// Parse YAML frontmatter
	frontmatter, markdown, err := parseAgentFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	// Parse tools string into slice
	var tools []string
	if frontmatter.Tools != "" {
		toolParts := strings.Split(frontmatter.Tools, ",")
		for _, tool := range toolParts {
			trimmed := strings.TrimSpace(tool)
			if trimmed != "" {
				tools = append(tools, trimmed)
			}
		}
	}

	agent := &MarketplaceAgent{
		ID:               agentID,
		Name:             frontmatter.Name,
		Description:      frontmatter.Description,
		Model:            frontmatter.Model,
		PrimarySkill:     frontmatter.PrimarySkill,
		SupportingSkills: frontmatter.SupportingSkills,
		Tools:            tools,
		Routing:          frontmatter.Routing,
		Orchestration:    frontmatter.Orchestration,
		Content:          markdown,
	}

	return agent, nil
}

// GetAgent retrieves a loaded agent by ID
func (ml *MarketplaceLoader) GetAgent(agentID string) (*MarketplaceAgent, error) {
	agent, ok := ml.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	return agent, nil
}

// ListAgents returns all loaded agents
func (ml *MarketplaceLoader) ListAgents() []*MarketplaceAgent {
	agents := make([]*MarketplaceAgent, 0, len(ml.agents))
	for _, agent := range ml.agents {
		agents = append(agents, agent)
	}
	return agents
}

// SearchAgents searches for agents matching a query
func (ml *MarketplaceLoader) SearchAgents(query string) []*MarketplaceAgent {
	query = strings.ToLower(query)
	var results []*MarketplaceAgent

	for _, agent := range ml.agents {
		// Search in name, description, and primary skill
		if strings.Contains(strings.ToLower(agent.Name), query) ||
			strings.Contains(strings.ToLower(agent.Description), query) ||
			strings.Contains(strings.ToLower(agent.PrimarySkill), query) {
			results = append(results, agent)
		}
	}

	return results
}

// BuildPromptWithAgent creates a prompt that includes agent and skill knowledge
func (ml *MarketplaceLoader) BuildPromptWithAgent(agentID string, userRequest string) (string, error) {
	agent, err := ml.GetAgent(agentID)
	if err != nil {
		return "", err
	}

	// Build base prompt with agent content
	var promptBuilder strings.Builder
	promptBuilder.WriteString(agent.Content)
	promptBuilder.WriteString("\n\n---\n\n")

	// Load primary skill if available
	if agent.PrimarySkill != "" {
		skill, err := ml.GetSkill(agent.PrimarySkill)
		if err == nil {
			promptBuilder.WriteString(fmt.Sprintf("## Primary Skill: %s\n\n", skill.Name))
			promptBuilder.WriteString(skill.Content)
			promptBuilder.WriteString("\n\n")
		} else {
			fmt.Printf("Warning: could not load primary skill %s: %v\n", agent.PrimarySkill, err)
		}
	}

	// Load supporting skills if available
	if len(agent.SupportingSkills) > 0 {
		promptBuilder.WriteString("## Supporting Skills\n\n")
		for _, skillID := range agent.SupportingSkills {
			skill, err := ml.GetSkill(skillID)
			if err == nil {
				promptBuilder.WriteString(fmt.Sprintf("### %s\n\n", skill.Name))
				promptBuilder.WriteString(skill.Content)
				promptBuilder.WriteString("\n\n")
			} else {
				fmt.Printf("Warning: could not load supporting skill %s: %v\n", skillID, err)
			}
		}
	}

	promptBuilder.WriteString("---\n\n")
	promptBuilder.WriteString("Using the above agent configuration and skill knowledge, please help with the following request:\n\n")
	promptBuilder.WriteString(userRequest)
	promptBuilder.WriteString("\n\nProvide a detailed, professional response based on the agent's expertise and skill knowledge.")

	return promptBuilder.String(), nil
}

// SkillFrontmatter represents the YAML frontmatter in SKILL.md files
type SkillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Keywords    string `yaml:"keywords"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author"`
	LastUpdated string `yaml:"last_updated"`
}

// AgentFrontmatter represents the YAML frontmatter in agent markdown files
type AgentFrontmatter struct {
	Name             string               `yaml:"name"`
	Description      string               `yaml:"description"`
	Model            string               `yaml:"model"`
	PrimarySkill     string               `yaml:"primary_skill"`
	SupportingSkills []string             `yaml:"supporting_skills"`
	Tools            string               `yaml:"tools"`
	Routing          *AgentRouting        `yaml:"routing"`
	Orchestration    *OrchestrationConfig `yaml:"orchestration"`
}

// parseFrontmatter extracts YAML frontmatter from markdown
func parseFrontmatter(content string) (*SkillFrontmatter, string, error) {
	// Check for frontmatter delimiters (---)
	if !strings.HasPrefix(content, "---\n") {
		return nil, "", fmt.Errorf("no frontmatter found")
	}

	// Find the end of frontmatter
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return nil, "", fmt.Errorf("unterminated frontmatter")
	}

	// Extract frontmatter and markdown
	frontmatterText := content[4 : endIdx+4]
	markdown := strings.TrimSpace(content[endIdx+9:])

	// Parse YAML
	var frontmatter SkillFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatterText), &frontmatter); err != nil {
		return nil, "", fmt.Errorf("parse YAML: %w", err)
	}

	return &frontmatter, markdown, nil
}

// parseAgentFrontmatter extracts YAML frontmatter from agent markdown
func parseAgentFrontmatter(content string) (*AgentFrontmatter, string, error) {
	// Check for frontmatter delimiters (---)
	if !strings.HasPrefix(content, "---\n") {
		return nil, "", fmt.Errorf("no frontmatter found")
	}

	// Find the end of frontmatter
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return nil, "", fmt.Errorf("unterminated frontmatter")
	}

	// Extract frontmatter and markdown
	frontmatterText := content[4 : endIdx+4]
	markdown := strings.TrimSpace(content[endIdx+9:])

	// Parse YAML
	var frontmatter AgentFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatterText), &frontmatter); err != nil {
		return nil, "", fmt.Errorf("parse YAML: %w", err)
	}

	return &frontmatter, markdown, nil
}
