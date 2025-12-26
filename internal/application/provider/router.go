// Package provider provides model routing and provider selection for LLM requests.
package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
)

// Router errors
var (
	ErrNoProfileConfig   = errors.New("no configuration found for profile")
	ErrNoModelAvailable  = errors.New("no model available for profile")
	ErrNoFallbackModel   = errors.New("no fallback model available")
	ErrInvalidProfile    = errors.New("invalid routing profile")
	ErrProviderNotFound  = errors.New("provider not found")
	ErrModelNotSupported = errors.New("model not supported by any provider")
	ErrConfigurationNil  = errors.New("routing configuration is nil")
	ErrRegistryNil       = errors.New("provider registry is nil")
)

// ModelSelection represents the result of model selection.
type ModelSelection struct {
	ModelID      string
	ProviderName string
	IsFallback   bool
}

// Router handles profile-based model selection with fallback support.
// It uses routing configuration to determine which models to use for different
// profiles and phases, and integrates with the provider registry to check availability.
type Router struct {
	mu       sync.RWMutex
	config   *config.RoutingConfiguration
	registry *adapterProvider.Registry
}

// NewRouter creates a new Router with the given configuration and registry.
// Returns an error if config or registry is nil.
func NewRouter(cfg *config.RoutingConfiguration, registry *adapterProvider.Registry) (*Router, error) {
	if cfg == nil {
		return nil, ErrConfigurationNil
	}
	if registry == nil {
		return nil, ErrRegistryNil
	}

	return &Router{
		config:   cfg,
		registry: registry,
	}, nil
}

// SelectModel selects a model based on the given routing profile.
// It returns the model ID and provider name for the selected model.
// If the primary model is unavailable, it attempts to use the fallback model.
func (r *Router) SelectModel(ctx context.Context, profile string) (*ModelSelection, error) {
	if !isValidProfile(profile) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProfile, profile)
	}

	r.mu.RLock()
	profileConfig := r.config.GetProfile(profile)
	r.mu.RUnlock()

	if profileConfig == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoProfileConfig, profile)
	}

	// Try the generation model first (default for general selection)
	modelID := profileConfig.GenerationModel
	if modelID != "" {
		providerName, available := r.findAvailableProvider(ctx, modelID)
		if available {
			return &ModelSelection{
				ModelID:      modelID,
				ProviderName: providerName,
				IsFallback:   false,
			}, nil
		}
	}

	// Try fallback model
	return r.GetFallbackModel(ctx, profile)
}

// SelectModelForPhase selects a model based on the phase's routing profile.
// It chooses between generation and review models based on the phase configuration.
func (r *Router) SelectModelForPhase(ctx context.Context, phase *skill.Phase) (*ModelSelection, error) {
	if phase == nil {
		return nil, errors.New("phase is nil")
	}

	profile := phase.RoutingProfile
	if !isValidProfile(profile) {
		profile = skill.ProfileBalanced // Default to balanced
	}

	r.mu.RLock()
	profileConfig := r.config.GetProfile(profile)
	r.mu.RUnlock()

	if profileConfig == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoProfileConfig, profile)
	}

	// Determine which model to use based on phase characteristics
	modelID := r.selectModelForPhaseType(phase, profileConfig)

	if modelID != "" {
		providerName, available := r.findAvailableProvider(ctx, modelID)
		if available {
			return &ModelSelection{
				ModelID:      modelID,
				ProviderName: providerName,
				IsFallback:   false,
			}, nil
		}
	}

	// Try fallback
	return r.GetFallbackModel(ctx, profile)
}

// selectModelForPhaseType determines the appropriate model based on phase type.
// Review phases use the review model, all others use the generation model.
func (r *Router) selectModelForPhaseType(phase *skill.Phase, profileConfig *config.ProfileConfiguration) string {
	// Check if this is a review phase by looking at the phase ID or name
	if isReviewPhase(phase) {
		if profileConfig.ReviewModel != "" {
			return profileConfig.ReviewModel
		}
	}

	// Default to generation model
	return profileConfig.GenerationModel
}

// isReviewPhase determines if a phase is a review phase based on naming conventions.
func isReviewPhase(phase *skill.Phase) bool {
	if phase == nil {
		return false
	}

	// Check for common review phase indicators
	id := phase.ID
	name := phase.Name

	reviewIndicators := []string{"review", "validate", "check", "verify", "audit"}
	for _, indicator := range reviewIndicators {
		if containsIgnoreCase(id, indicator) || containsIgnoreCase(name, indicator) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}

	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (ASCII only).
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// GetFallbackModel returns the fallback model for the given profile.
// It tries the profile's fallback model first, then walks the fallback chain.
func (r *Router) GetFallbackModel(ctx context.Context, profile string) (*ModelSelection, error) {
	if !isValidProfile(profile) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProfile, profile)
	}

	r.mu.RLock()
	profileConfig := r.config.GetProfile(profile)
	fallbackChain := r.config.FallbackChain
	r.mu.RUnlock()

	// Try the profile's configured fallback model
	if profileConfig != nil && profileConfig.FallbackModel != "" {
		providerName, available := r.findAvailableProvider(ctx, profileConfig.FallbackModel)
		if available {
			return &ModelSelection{
				ModelID:      profileConfig.FallbackModel,
				ProviderName: providerName,
				IsFallback:   true,
			}, nil
		}
	}

	// Try the fallback chain (providers in order of preference)
	for _, providerName := range fallbackChain {
		provider := r.registry.Get(providerName)
		if provider == nil {
			continue
		}

		// Get available models from this provider
		models, err := provider.ListModels(ctx)
		if err != nil || len(models) == 0 {
			continue
		}

		// Check if any model is available
		for _, modelID := range models {
			available, err := provider.IsAvailable(ctx, modelID)
			if err == nil && available {
				return &ModelSelection{
					ModelID:      modelID,
					ProviderName: providerName,
					IsFallback:   true,
				}, nil
			}
		}
	}

	return nil, ErrNoFallbackModel
}

// IsModelAvailable checks if a specific model is available through any registered provider.
func (r *Router) IsModelAvailable(ctx context.Context, modelID string) bool {
	_, available := r.findAvailableProvider(ctx, modelID)
	return available
}

// findAvailableProvider finds a provider that supports and has the model available.
// Returns the provider name and true if found, empty string and false otherwise.
func (r *Router) findAvailableProvider(ctx context.Context, modelID string) (string, bool) {
	provider, err := r.registry.FindByModel(ctx, modelID)
	if err != nil {
		return "", false
	}

	// Check if the model is actually available (not just supported)
	available, err := provider.IsAvailable(ctx, modelID)
	if err != nil || !available {
		return "", false
	}

	return provider.Info().Name, true
}

// GetModelConfig returns the model configuration for a given model ID and provider.
// Returns nil if not found.
func (r *Router) GetModelConfig(providerName, modelID string) *config.ModelConfiguration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providerConfig := r.config.GetProvider(providerName)
	if providerConfig == nil {
		return nil
	}

	return providerConfig.GetModel(modelID)
}

// GetProfileConfig returns the profile configuration for the given profile name.
func (r *Router) GetProfileConfig(profile string) *config.ProfileConfiguration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.GetProfile(profile)
}

// UpdateConfig updates the router's configuration.
// This is thread-safe and can be called during runtime.
func (r *Router) UpdateConfig(cfg *config.RoutingConfiguration) error {
	if cfg == nil {
		return ErrConfigurationNil
	}

	r.mu.Lock()
	r.config = cfg
	r.mu.Unlock()

	return nil
}

// GetEnabledProviders returns the list of enabled provider names in priority order.
func (r *Router) GetEnabledProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.GetEnabledProviders()
}

// GetDefaultProvider returns the default provider name.
func (r *Router) GetDefaultProvider() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.DefaultProvider
}

// isValidProfile checks if the profile is a valid routing profile.
func isValidProfile(profile string) bool {
	switch profile {
	case skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium:
		return true
	default:
		return false
	}
}

// SelectModelWithCapabilities selects a model that has the required capabilities.
// Returns an error if no model with the required capabilities is available.
func (r *Router) SelectModelWithCapabilities(ctx context.Context, profile string, capabilities []string) (*ModelSelection, error) {
	if !isValidProfile(profile) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidProfile, profile)
	}

	r.mu.RLock()
	providers := r.config.Providers
	r.mu.RUnlock()

	// Search through all providers for a model with matching capabilities
	for providerName, providerConfig := range providers {
		if !providerConfig.Enabled {
			continue
		}

		for modelID, modelConfig := range providerConfig.Models {
			if !modelConfig.Enabled {
				continue
			}

			// Check if model has all required capabilities
			if hasAllCapabilities(modelConfig, capabilities) {
				// Verify model is actually available
				providerFound, available := r.findAvailableProvider(ctx, modelID)
				if available {
					return &ModelSelection{
						ModelID:      modelID,
						ProviderName: providerFound,
						IsFallback:   false,
					}, nil
				}
			}
			// Avoid unused variable warning
			_ = providerName
		}
	}

	// Fall back to regular selection
	return r.SelectModel(ctx, profile)
}

// hasAllCapabilities checks if the model has all the required capabilities.
func hasAllCapabilities(model *config.ModelConfiguration, required []string) bool {
	if model == nil || len(required) == 0 {
		return len(required) == 0
	}

	for _, cap := range required {
		if !model.HasCapability(cap) {
			return false
		}
	}

	return true
}
