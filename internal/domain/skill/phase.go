// Package skill contains domain types for skill execution.
package skill

import (
	"errors"
	"fmt"
	"strings"
)

// Routing profiles for phase execution.
const (
	RoutingProfileCheap    = "cheap"
	RoutingProfileBalanced = "balanced"
	RoutingProfilePremium  = "premium"
)

// Default values for Phase configuration.
const (
	DefaultRoutingProfile = RoutingProfileBalanced
	DefaultMaxTokens      = 4096
	DefaultTemperature    = 0.7
)

// Phase validation errors.
var (
	ErrPhaseIDRequired             = errors.New("phase id is required")
	ErrPhaseNameRequired           = errors.New("phase name is required")
	ErrPhasePromptTemplateRequired = errors.New("phase prompt template is required")
	ErrInvalidRoutingProfile       = errors.New("invalid routing profile: must be cheap, balanced, or premium")
	ErrInvalidMaxTokens            = errors.New("max tokens must be positive")
	ErrInvalidTemperature          = errors.New("temperature must be between 0.0 and 2.0")
)

// Phase represents a discrete step in a skill execution workflow.
// It is a value object that defines how a particular phase should be executed,
// including its prompt template, routing preferences, and dependencies.
type Phase struct {
	ID             string
	Name           string
	PromptTemplate string
	RoutingProfile string   // cheap, balanced, premium
	DependsOn      []string // phase IDs this depends on
	MaxTokens      int
	Temperature    float32
}

// NewPhase creates a new Phase with the required fields and default values for optional fields.
// Returns an error if id, name, or promptTemplate are empty.
func NewPhase(id, name, promptTemplate string) (*Phase, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	promptTemplate = strings.TrimSpace(promptTemplate)

	if id == "" {
		return nil, ErrPhaseIDRequired
	}
	if name == "" {
		return nil, ErrPhaseNameRequired
	}
	if promptTemplate == "" {
		return nil, ErrPhasePromptTemplateRequired
	}

	return &Phase{
		ID:             id,
		Name:           name,
		PromptTemplate: promptTemplate,
		RoutingProfile: DefaultRoutingProfile,
		DependsOn:      nil,
		MaxTokens:      DefaultMaxTokens,
		Temperature:    DefaultTemperature,
	}, nil
}

// WithRoutingProfile sets the routing profile for the phase.
// Valid values are: cheap, balanced, premium.
func (p *Phase) WithRoutingProfile(profile string) *Phase {
	p.RoutingProfile = strings.TrimSpace(profile)
	return p
}

// WithDependencies sets the phase IDs that this phase depends on.
// These phases must complete before this phase can execute.
func (p *Phase) WithDependencies(deps []string) *Phase {
	if deps == nil {
		p.DependsOn = nil
		return p
	}
	// Make a copy to avoid external mutation
	p.DependsOn = make([]string, len(deps))
	copy(p.DependsOn, deps)
	return p
}

// WithMaxTokens sets the maximum number of tokens for the phase output.
func (p *Phase) WithMaxTokens(max int) *Phase {
	p.MaxTokens = max
	return p
}

// WithTemperature sets the temperature for LLM inference.
func (p *Phase) WithTemperature(temp float32) *Phase {
	p.Temperature = temp
	return p
}

// Validate checks if the Phase is in a valid state.
// Returns an error describing any validation failures.
func (p *Phase) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return ErrPhaseIDRequired
	}
	if strings.TrimSpace(p.Name) == "" {
		return ErrPhaseNameRequired
	}
	if strings.TrimSpace(p.PromptTemplate) == "" {
		return ErrPhasePromptTemplateRequired
	}
	if !isValidRoutingProfile(p.RoutingProfile) {
		return fmt.Errorf("%w: got %q", ErrInvalidRoutingProfile, p.RoutingProfile)
	}
	if p.MaxTokens <= 0 {
		return ErrInvalidMaxTokens
	}
	if p.Temperature < 0.0 || p.Temperature > 2.0 {
		return ErrInvalidTemperature
	}
	return nil
}

// isValidRoutingProfile checks if the given profile is a valid routing profile.
func isValidRoutingProfile(profile string) bool {
	switch profile {
	case RoutingProfileCheap, RoutingProfileBalanced, RoutingProfilePremium:
		return true
	default:
		return false
	}
}

// HasDependencies returns true if this phase has dependencies on other phases.
func (p *Phase) HasDependencies() bool {
	return len(p.DependsOn) > 0
}

// DependsOnPhase checks if this phase depends on the given phase ID.
func (p *Phase) DependsOnPhase(phaseID string) bool {
	for _, dep := range p.DependsOn {
		if dep == phaseID {
			return true
		}
	}
	return false
}
