package skills

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

var (
	// validSkillNameRegex matches valid skill names (alphanumeric, hyphens, underscores)
	validSkillNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ValidateSkill validates a skill configuration
func ValidateSkill(skill *types.SkillConfig) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}

	// Validate name
	if skill.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	// Validate name format (alphanumeric, hyphens, underscores only, no whitespace)
	if !validSkillNameRegex.MatchString(skill.Name) {
		return fmt.Errorf("skill name '%s' contains invalid characters. Only alphanumeric characters, hyphens, and underscores are allowed", skill.Name)
	}

	// Validate version
	if skill.Version == "" {
		return fmt.Errorf("skill version is required")
	}

	// Validate description
	if skill.Description == "" {
		return fmt.Errorf("skill description is required")
	}

	// Validate default model
	if skill.DefaultModel == "" {
		return fmt.Errorf("skill default_model is required")
	}

	// Trim whitespace and validate non-empty
	if strings.TrimSpace(skill.Name) == "" {
		return fmt.Errorf("skill name cannot be empty or whitespace only")
	}

	if strings.TrimSpace(skill.Version) == "" {
		return fmt.Errorf("skill version cannot be empty or whitespace only")
	}

	if strings.TrimSpace(skill.Description) == "" {
		return fmt.Errorf("skill description cannot be empty or whitespace only")
	}

	if strings.TrimSpace(skill.DefaultModel) == "" {
		return fmt.Errorf("skill default_model cannot be empty or whitespace only")
	}

	return nil
}
