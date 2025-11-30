package routing

import (
	"fmt"

	"github.com/jbctechsolutions/skillrunner/internal/llm"
)

// Router handles profile-based model routing
type Router struct {
	providers      map[string]llm.Provider
	models         map[string]ModelConfig
	profiles       map[string]RoutingProfileConfig
	costSimulation CostSimulationConfig
}

// NewRouter creates a new router from configuration
func NewRouter(config *RoutingConfig) (*Router, error) {
	// Initialize providers
	providers := make(map[string]llm.Provider)
	for _, providerConfig := range config.Providers {
		var provider llm.Provider
		var err error

		switch providerConfig.Type {
		case "ollama":
			provider, err = llm.NewOllamaProvider(providerConfig.APIKeyEnv)
		case "anthropic":
			provider, err = llm.NewAnthropicProvider(providerConfig.APIKeyEnv)
		default:
			return nil, fmt.Errorf("unknown provider type: %s (supported: ollama, anthropic)", providerConfig.Type)
		}

		if err != nil {
			// Log warning but continue - provider may not be configured
			// We'll check availability when routing
			continue
		}

		providers[providerConfig.Name] = provider
	}

	// Build models map
	models := make(map[string]ModelConfig)
	for _, model := range config.Models {
		models[model.ID] = model
	}

	// Build profiles map
	profiles := make(map[string]RoutingProfileConfig)
	for name, profile := range config.RoutingProfiles {
		profiles[name] = profile
	}

	return &Router{
		providers:      providers,
		models:         models,
		profiles:       profiles,
		costSimulation: config.CostSimulation,
	}, nil
}

// Route selects a provider and model for the given profile
func (r *Router) Route(profile string) (llm.Provider, ModelConfig, error) {
	p, ok := r.profiles[profile]
	if !ok {
		return nil, ModelConfig{}, fmt.Errorf("unknown profile: %s", profile)
	}

	for _, modelID := range p.CandidateModels {
		mc, ok := r.models[modelID]
		if !ok {
			continue
		}
		prov, ok := r.providers[mc.Provider]
		if !ok {
			continue
		}
		return prov, mc, nil
	}

	return nil, ModelConfig{}, fmt.Errorf("no available providers for profile %s", profile)
}

// GetModelConfig returns the model configuration for a given model ID
func (r *Router) GetModelConfig(modelID string) (*ModelConfig, error) {
	mc, ok := r.models[modelID]
	if !ok {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}
	return &mc, nil
}

// GetCostSimulationConfig returns the cost simulation configuration
func (r *Router) GetCostSimulationConfig() CostSimulationConfig {
	return r.costSimulation
}

// RouteToModel routes to a specific model by ID (from routing_models)
func (r *Router) RouteToModel(modelID string) (llm.Provider, ModelConfig, error) {
	mc, ok := r.models[modelID]
	if !ok {
		return nil, ModelConfig{}, fmt.Errorf("model not found: %s", modelID)
	}

	prov, ok := r.providers[mc.Provider]
	if !ok {
		return nil, ModelConfig{}, fmt.Errorf("provider not available: %s", mc.Provider)
	}

	return prov, mc, nil
}
