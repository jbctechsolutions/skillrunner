// Package skill provides domain types for skill configuration and routing.
package skill

import (
	"errors"
	"fmt"
)

// Valid routing profiles
const (
	ProfileCheap    = "cheap"
	ProfileBalanced = "balanced"
	ProfilePremium  = "premium"
)

// RoutingConfig defines model routing configuration for skill execution.
// It specifies which models to use for different phases of execution
// and provides fallback options when primary models are unavailable.
type RoutingConfig struct {
	DefaultProfile   string // cheap, balanced, premium
	GenerationModel  string // model for generation phases
	ReviewModel      string // model for review phases
	FallbackModel    string // fallback when primary unavailable
	MaxContextTokens int
}

// NewRoutingConfig creates a new RoutingConfig with sensible defaults.
// Default profile is "balanced" and max context tokens is 4096.
func NewRoutingConfig() *RoutingConfig {
	return &RoutingConfig{
		DefaultProfile:   ProfileBalanced,
		MaxContextTokens: 4096,
	}
}

// WithDefaultProfile sets the default routing profile.
// Valid profiles are: cheap, balanced, premium.
func (r *RoutingConfig) WithDefaultProfile(profile string) *RoutingConfig {
	r.DefaultProfile = profile
	return r
}

// WithGenerationModel sets the model to use for generation phases.
func (r *RoutingConfig) WithGenerationModel(model string) *RoutingConfig {
	r.GenerationModel = model
	return r
}

// WithReviewModel sets the model to use for review phases.
func (r *RoutingConfig) WithReviewModel(model string) *RoutingConfig {
	r.ReviewModel = model
	return r
}

// WithFallbackModel sets the fallback model when primary is unavailable.
func (r *RoutingConfig) WithFallbackModel(model string) *RoutingConfig {
	r.FallbackModel = model
	return r
}

// WithMaxContextTokens sets the maximum context tokens allowed.
func (r *RoutingConfig) WithMaxContextTokens(max int) *RoutingConfig {
	r.MaxContextTokens = max
	return r
}

// Validate checks if the RoutingConfig is valid.
// It returns an error if validation fails.
func (r *RoutingConfig) Validate() error {
	if r == nil {
		return errors.New("routing config is nil")
	}

	// Validate default profile
	validProfiles := map[string]bool{
		ProfileCheap:    true,
		ProfileBalanced: true,
		ProfilePremium:  true,
	}

	if r.DefaultProfile == "" {
		return errors.New("default profile is required")
	}

	if !validProfiles[r.DefaultProfile] {
		return fmt.Errorf("invalid default profile %q: must be one of cheap, balanced, premium", r.DefaultProfile)
	}

	// Validate max context tokens
	if r.MaxContextTokens <= 0 {
		return errors.New("max context tokens must be positive")
	}

	return nil
}
