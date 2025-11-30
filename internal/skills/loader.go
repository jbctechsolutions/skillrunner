package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jbctechsolutions/skillrunner/internal/types"
	"gopkg.in/yaml.v3"
)

// Loader handles loading and caching of skill configurations from YAML files
type Loader struct {
	skillsDir string
	cache     map[string]*types.SkillConfig
	mu        sync.RWMutex
}

// NewLoader creates a new Loader instance
func NewLoader(skillsDir string) *Loader {
	return &Loader{
		skillsDir: skillsDir,
		cache:     make(map[string]*types.SkillConfig),
	}
}

// LoadSkill loads a skill configuration from a YAML file
// It caches the result for subsequent calls
func (l *Loader) LoadSkill(skillName string) (*types.SkillConfig, error) {
	// Check cache first
	l.mu.RLock()
	if cached, ok := l.cache[skillName]; ok {
		l.mu.RUnlock()
		return cached, nil
	}
	l.mu.RUnlock()

	// Load from file
	skill, err := l.loadSkillFromFile(skillName)
	if err != nil {
		return nil, err
	}

	// Validate skill
	if err := ValidateSkill(skill); err != nil {
		return nil, fmt.Errorf("validation failed for skill '%s': %w", skillName, err)
	}

	// Cache the skill
	l.mu.Lock()
	l.cache[skillName] = skill
	l.mu.Unlock()

	return skill, nil
}

// loadSkillFromFile loads a skill configuration from a YAML file
func (l *Loader) loadSkillFromFile(skillName string) (*types.SkillConfig, error) {
	// Try both .yaml and .yml extensions
	var skillPath string
	var err error
	var data []byte

	// Try .yaml first
	if l.skillsDir != "" {
		skillPath = filepath.Join(l.skillsDir, skillName+".yaml")
	} else {
		skillPath = skillName + ".yaml"
	}

	data, err = os.ReadFile(skillPath)
	if err != nil && os.IsNotExist(err) {
		// Try .yml if .yaml doesn't exist
		if l.skillsDir != "" {
			skillPath = filepath.Join(l.skillsDir, skillName+".yml")
		} else {
			skillPath = skillName + ".yml"
		}
		data, err = os.ReadFile(skillPath)
	}

	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("skill '%s' not found (tried .yaml and .yml)", skillName)
		}
		return nil, fmt.Errorf("failed to read skill file '%s': %w", skillPath, err)
	}

	// Check if file is empty
	if len(data) == 0 {
		return nil, fmt.Errorf("skill file '%s' is empty", skillPath)
	}

	// Parse YAML
	var skill types.SkillConfig
	if err := yaml.Unmarshal(data, &skill); err != nil {
		return nil, fmt.Errorf("failed to parse YAML for skill '%s': %w", skillName, err)
	}

	// Ensure skill name matches filename
	if skill.Name == "" {
		skill.Name = skillName
	} else if skill.Name != skillName {
		return nil, fmt.Errorf("skill name '%s' in file does not match filename '%s'", skill.Name, skillName)
	}

	return &skill, nil
}

// LoadAllSkills loads all skill configurations from the skills directory
func (l *Loader) LoadAllSkills() ([]*types.SkillConfig, error) {
	if l.skillsDir == "" {
		return []*types.SkillConfig{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory '%s': %w", l.skillsDir, err)
	}

	var skills []*types.SkillConfig
	var errors []string

	for _, entry := range entries {
		// Skip non-YAML files
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".yaml") && !strings.HasSuffix(strings.ToLower(entry.Name()), ".yml") {
			continue
		}

		// Extract skill name from filename
		skillName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

		// Load skill
		skill, err := l.LoadSkill(skillName)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to load skill '%s': %v", skillName, err))
			continue
		}

		skills = append(skills, skill)
	}

	// If we have errors and no valid skills, return error
	if len(errors) > 0 && len(skills) == 0 {
		return nil, fmt.Errorf("failed to load any skills: %s", strings.Join(errors, "; "))
	}

	// If we have some errors but also some valid skills, we could log them
	// For now, we'll return the valid skills and let the caller decide

	return skills, nil
}

// ClearCache clears the skill cache
func (l *Loader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]*types.SkillConfig)
}

// GetCachedSkill returns a cached skill if available, nil otherwise
func (l *Loader) GetCachedSkill(skillName string) *types.SkillConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cache[skillName]
}
