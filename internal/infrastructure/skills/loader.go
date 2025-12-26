// Package skills provides infrastructure for loading and managing skill definitions.
package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// SkillDefinition represents the YAML structure of a skill definition file.
type SkillDefinition struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Phases      []PhaseDefinition `yaml:"phases"`
	Routing     RoutingDefinition `yaml:"routing"`
	Metadata    map[string]any    `yaml:"metadata"`
}

// PhaseDefinition represents the YAML structure of a phase within a skill.
type PhaseDefinition struct {
	ID             string   `yaml:"id"`
	Name           string   `yaml:"name"`
	PromptTemplate string   `yaml:"prompt_template"`
	RoutingProfile string   `yaml:"routing_profile"`
	DependsOn      []string `yaml:"depends_on"`
	MaxTokens      int      `yaml:"max_tokens"`
	Temperature    float32  `yaml:"temperature"`
}

// RoutingDefinition represents the YAML structure of routing configuration.
type RoutingDefinition struct {
	DefaultProfile   string `yaml:"default_profile"`
	GenerationModel  string `yaml:"generation_model"`
	ReviewModel      string `yaml:"review_model"`
	FallbackModel    string `yaml:"fallback_model"`
	MaxContextTokens int    `yaml:"max_context_tokens"`
}

// Loader errors.
var (
	ErrInvalidPath      = errors.New("invalid file path")
	ErrNotYAMLFile      = errors.New("file is not a YAML file")
	ErrEmptyFile        = errors.New("file is empty")
	ErrInvalidStructure = errors.New("invalid skill structure")
)

// Loader handles loading skill definitions from the filesystem.
type Loader struct{}

// NewLoader creates a new skill loader.
func NewLoader() *Loader {
	return &Loader{}
}

// LoadSkill loads a single skill definition from a YAML file.
// It reads the file, parses the YAML content, validates the structure,
// and converts it to a domain Skill type.
func (l *Loader) LoadSkill(path string) (*skill.Skill, error) {
	// Validate the path
	if err := validatePath(path); err != nil {
		return nil, err
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrEmptyFile, path)
	}

	// Parse YAML
	var def SkillDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	// Validate the definition structure
	if err := validateDefinition(&def); err != nil {
		return nil, fmt.Errorf("invalid skill definition in %s: %w", path, err)
	}

	// Convert to domain type
	return convertToDomainSkill(&def)
}

// LoadSkillsDir loads all skill definitions from a directory.
// It recursively scans for .yaml and .yml files, loading each as a skill.
// Returns a map of skill ID to Skill, and any errors encountered.
func (l *Loader) LoadSkillsDir(dir string) (map[string]*skill.Skill, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	skills := make(map[string]*skill.Skill)
	var loadErrors []error

	// Walk the directory
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Load the skill
		s, err := l.LoadSkill(path)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("failed to load %s: %w", path, err))
			return nil // Continue loading other files
		}

		// Check for duplicate IDs
		if existing, ok := skills[s.ID()]; ok {
			loadErrors = append(loadErrors, fmt.Errorf(
				"duplicate skill ID %q: found in both %s and current file",
				s.ID(), existing.Name(),
			))
			return nil
		}

		skills[s.ID()] = s
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	// Aggregate errors if any occurred
	if len(loadErrors) > 0 {
		return skills, errors.Join(loadErrors...)
	}

	return skills, nil
}

// validatePath checks if the path is valid for a skill file.
func validatePath(path string) error {
	if path == "" {
		return ErrInvalidPath
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("%w: expected .yaml or .yml extension, got %s", ErrNotYAMLFile, ext)
	}

	return nil
}

// validateDefinition validates the skill definition structure.
func validateDefinition(def *SkillDefinition) error {
	var errs []error

	if strings.TrimSpace(def.ID) == "" {
		errs = append(errs, errors.New("skill id is required"))
	}

	if strings.TrimSpace(def.Name) == "" {
		errs = append(errs, errors.New("skill name is required"))
	}

	if len(def.Phases) == 0 {
		errs = append(errs, errors.New("at least one phase is required"))
	}

	// Validate each phase
	phaseIDs := make(map[string]bool)
	for i, phase := range def.Phases {
		if strings.TrimSpace(phase.ID) == "" {
			errs = append(errs, fmt.Errorf("phase %d: id is required", i))
		} else {
			if phaseIDs[phase.ID] {
				errs = append(errs, fmt.Errorf("phase %d: duplicate phase id %q", i, phase.ID))
			}
			phaseIDs[phase.ID] = true
		}

		if strings.TrimSpace(phase.Name) == "" {
			errs = append(errs, fmt.Errorf("phase %d (%s): name is required", i, phase.ID))
		}

		if strings.TrimSpace(phase.PromptTemplate) == "" {
			errs = append(errs, fmt.Errorf("phase %d (%s): prompt_template is required", i, phase.ID))
		}

		// Validate routing profile if provided
		if phase.RoutingProfile != "" {
			if !isValidRoutingProfile(phase.RoutingProfile) {
				errs = append(errs, fmt.Errorf("phase %d (%s): invalid routing_profile %q", i, phase.ID, phase.RoutingProfile))
			}
		}
	}

	// Validate phase dependencies
	for i, phase := range def.Phases {
		for _, depID := range phase.DependsOn {
			if !phaseIDs[depID] {
				errs = append(errs, fmt.Errorf("phase %d (%s): unknown dependency %q", i, phase.ID, depID))
			}
		}
	}

	// Validate routing config if provided
	if def.Routing.DefaultProfile != "" {
		if !isValidRoutingProfile(def.Routing.DefaultProfile) {
			errs = append(errs, fmt.Errorf("routing: invalid default_profile %q", def.Routing.DefaultProfile))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// isValidRoutingProfile checks if the profile is a valid routing profile.
func isValidRoutingProfile(profile string) bool {
	switch profile {
	case skill.RoutingProfileCheap, skill.RoutingProfileBalanced, skill.RoutingProfilePremium:
		return true
	default:
		return false
	}
}

// convertToDomainSkill converts a YAML definition to a domain Skill.
func convertToDomainSkill(def *SkillDefinition) (*skill.Skill, error) {
	// Convert phases
	phases := make([]skill.Phase, 0, len(def.Phases))
	for _, phaseDef := range def.Phases {
		phase, err := convertToDomainPhase(&phaseDef)
		if err != nil {
			return nil, fmt.Errorf("failed to convert phase %s: %w", phaseDef.ID, err)
		}
		phases = append(phases, *phase)
	}

	// Create the skill
	s, err := skill.NewSkill(def.ID, def.Name, def.Version, phases)
	if err != nil {
		return nil, fmt.Errorf("failed to create skill: %w", err)
	}

	// Set optional fields
	if def.Description != "" {
		s.SetDescription(def.Description)
	}

	// Set routing configuration
	routing := convertToDomainRouting(&def.Routing)
	s.SetRouting(routing)

	// Set metadata
	for k, v := range def.Metadata {
		s.SetMetadata(k, v)
	}

	// Validate the complete skill
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("skill validation failed: %w", err)
	}

	return s, nil
}

// convertToDomainPhase converts a YAML phase definition to a domain Phase.
func convertToDomainPhase(def *PhaseDefinition) (*skill.Phase, error) {
	phase, err := skill.NewPhase(def.ID, def.Name, def.PromptTemplate)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if def.RoutingProfile != "" {
		phase.WithRoutingProfile(def.RoutingProfile)
	}

	if len(def.DependsOn) > 0 {
		phase.WithDependencies(def.DependsOn)
	}

	if def.MaxTokens > 0 {
		phase.WithMaxTokens(def.MaxTokens)
	}

	if def.Temperature > 0 {
		phase.WithTemperature(def.Temperature)
	}

	return phase, nil
}

// convertToDomainRouting converts a YAML routing definition to a domain RoutingConfig.
func convertToDomainRouting(def *RoutingDefinition) skill.RoutingConfig {
	routing := skill.NewRoutingConfig()

	if def.DefaultProfile != "" {
		routing.WithDefaultProfile(def.DefaultProfile)
	}

	if def.GenerationModel != "" {
		routing.WithGenerationModel(def.GenerationModel)
	}

	if def.ReviewModel != "" {
		routing.WithReviewModel(def.ReviewModel)
	}

	if def.FallbackModel != "" {
		routing.WithFallbackModel(def.FallbackModel)
	}

	if def.MaxContextTokens > 0 {
		routing.WithMaxContextTokens(def.MaxContextTokens)
	}

	return *routing
}
