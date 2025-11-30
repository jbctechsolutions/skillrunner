package orchestration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbctechsolutions/skillrunner/internal/types"
	"gopkg.in/yaml.v3"
)

// SkillLoader loads orchestrated skills from YAML files
type SkillLoader struct {
	skillsDir string
}

// NewSkillLoader creates a new skill loader
func NewSkillLoader(skillsDir string) *SkillLoader {
	return &SkillLoader{
		skillsDir: skillsDir,
	}
}

// LoadSkill loads a skill by name or path
func (sl *SkillLoader) LoadSkill(nameOrPath string) (*types.OrchestratedSkill, error) {
	// Check if it's a direct path
	var skillPath string
	if filepath.IsAbs(nameOrPath) || filepath.Ext(nameOrPath) == ".yaml" || filepath.Ext(nameOrPath) == ".yml" {
		skillPath = nameOrPath
	} else {
		// Treat as skill name, look in skills directory
		skillPath = filepath.Join(sl.skillsDir, nameOrPath, "skill.yaml")

		// Try .yml extension if .yaml doesn't exist
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			skillPath = filepath.Join(sl.skillsDir, nameOrPath, "skill.yml")
		}
	}

	// Read skill file
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	// Parse YAML
	var skill types.OrchestratedSkill
	if err := yaml.Unmarshal(data, &skill); err != nil {
		return nil, fmt.Errorf("parse skill YAML: %w", err)
	}

	// Validate skill
	if err := sl.validateSkill(&skill); err != nil {
		return nil, fmt.Errorf("validate skill: %w", err)
	}

	return &skill, nil
}

// validateSkill validates a skill definition
func (sl *SkillLoader) validateSkill(skill *types.OrchestratedSkill) error {
	if skill.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	if len(skill.Phases) == 0 {
		return fmt.Errorf("skill must have at least one phase")
	}

	// Validate each phase
	phaseIDs := make(map[string]bool)
	for i, phase := range skill.Phases {
		if phase.ID == "" {
			return fmt.Errorf("phase %d: ID is required", i)
		}

		if phaseIDs[phase.ID] {
			return fmt.Errorf("duplicate phase ID: %s", phase.ID)
		}
		phaseIDs[phase.ID] = true

		if phase.PromptTemplate == "" && phase.PromptFile == "" {
			return fmt.Errorf("phase %s: prompt_template or prompt_file is required", phase.ID)
		}

		if phase.OutputKey == "" {
			return fmt.Errorf("phase %s: output_key is required", phase.ID)
		}
	}

	// Validate dependencies exist
	for _, phase := range skill.Phases {
		for _, depID := range phase.DependsOn {
			if !phaseIDs[depID] {
				return fmt.Errorf("phase %s depends on non-existent phase: %s", phase.ID, depID)
			}
		}
	}

	return nil
}

// ListSkills lists all available orchestrated skills
func (sl *SkillLoader) ListSkills() ([]string, error) {
	entries, err := os.ReadDir(sl.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read skills directory: %w", err)
	}

	var skills []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if skill.yaml or skill.yml exists
		skillPath := filepath.Join(sl.skillsDir, entry.Name(), "skill.yaml")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			skillPath = filepath.Join(sl.skillsDir, entry.Name(), "skill.yml")
			if _, err := os.Stat(skillPath); os.IsNotExist(err) {
				continue
			}
		}

		skills = append(skills, entry.Name())
	}

	return skills, nil
}
