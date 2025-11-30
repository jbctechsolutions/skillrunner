package routing

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProviderConfig represents a provider configuration
type ProviderConfig struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	APIKeyEnv string `yaml:"api_key_env"`
}

// ModelConfig represents a model configuration
type ModelConfig struct {
	ID                     string  `yaml:"id"`
	Provider               string  `yaml:"provider"`
	Model                  string  `yaml:"model"`
	ProfileCostPer1KInput  float64 `yaml:"profile_cost_per_1k_input"`
	ProfileCostPer1KOutput float64 `yaml:"profile_cost_per_1k_output"`
}

// RoutingProfileConfig represents a routing profile configuration
type RoutingProfileConfig struct {
	CandidateModels []string `yaml:"candidate_models"`
}

// CostSimulationConfig represents cost simulation configuration
type CostSimulationConfig struct {
	PremiumModelID string `yaml:"premium_model_id"`
	CheapModelID   string `yaml:"cheap_model_id"`
}

// RoutingConfig represents the full routing configuration
type RoutingConfig struct {
	Providers       []ProviderConfig                `yaml:"providers"`
	Models          []ModelConfig                   `yaml:"-"` // Not unmarshaled directly - populated from routing_models
	RoutingModels   []ModelConfig                   `yaml:"routing_models"`
	RoutingProfiles map[string]RoutingProfileConfig `yaml:"routing_profiles"`
	CostSimulation  CostSimulationConfig            `yaml:"cost_simulation"`
}

// LoadRoutingConfig loads routing configuration from a YAML file
func LoadRoutingConfig(configPath string) (*RoutingConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config RoutingConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config YAML: %w", err)
	}

	// Populate Models from RoutingModels
	// (Models field has yaml:"-" to avoid conflict with the legacy models map)
	config.Models = config.RoutingModels

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

// validateConfig validates the routing configuration
func validateConfig(config *RoutingConfig) error {
	// Validate providers
	if len(config.Providers) == 0 {
		return fmt.Errorf("no providers defined")
	}

	providerNames := make(map[string]bool)
	for _, provider := range config.Providers {
		if provider.Name == "" {
			return fmt.Errorf("provider name is required")
		}
		if provider.Type == "" {
			return fmt.Errorf("provider type is required for provider %s", provider.Name)
		}
		// Ollama doesn't require an API key
		if provider.APIKeyEnv == "" && provider.Type != "ollama" {
			return fmt.Errorf("api_key_env is required for provider %s", provider.Name)
		}
		if providerNames[provider.Name] {
			return fmt.Errorf("duplicate provider name: %s", provider.Name)
		}
		providerNames[provider.Name] = true
	}

	// Validate models
	if len(config.Models) == 0 {
		return fmt.Errorf("no models defined")
	}

	modelIDs := make(map[string]bool)
	providerSet := make(map[string]bool)
	for _, provider := range config.Providers {
		providerSet[provider.Name] = true
	}

	for _, model := range config.Models {
		if model.ID == "" {
			return fmt.Errorf("model id is required")
		}
		if model.Provider == "" {
			return fmt.Errorf("model provider is required for model %s", model.ID)
		}
		if !providerSet[model.Provider] {
			return fmt.Errorf("model %s references unknown provider %s", model.ID, model.Provider)
		}
		if model.Model == "" {
			return fmt.Errorf("model name is required for model %s", model.ID)
		}
		if modelIDs[model.ID] {
			return fmt.Errorf("duplicate model id: %s", model.ID)
		}
		modelIDs[model.ID] = true
	}

	// Validate routing profiles
	if len(config.RoutingProfiles) == 0 {
		return fmt.Errorf("no routing profiles defined")
	}

	for profileName, profile := range config.RoutingProfiles {
		if len(profile.CandidateModels) == 0 {
			return fmt.Errorf("routing profile %s has no candidate models", profileName)
		}
		for _, modelID := range profile.CandidateModels {
			if !modelIDs[modelID] {
				return fmt.Errorf("routing profile %s references unknown model %s", profileName, modelID)
			}
		}
	}

	// Validate cost simulation
	if config.CostSimulation.PremiumModelID != "" && !modelIDs[config.CostSimulation.PremiumModelID] {
		return fmt.Errorf("cost simulation references unknown premium model %s", config.CostSimulation.PremiumModelID)
	}
	if config.CostSimulation.CheapModelID != "" && !modelIDs[config.CostSimulation.CheapModelID] {
		return fmt.Errorf("cost simulation references unknown cheap model %s", config.CostSimulation.CheapModelID)
	}

	return nil
}

// GetModelConfig returns the model configuration for a given model ID
func (c *RoutingConfig) GetModelConfig(modelID string) (*ModelConfig, error) {
	for i := range c.Models {
		if c.Models[i].ID == modelID {
			return &c.Models[i], nil
		}
	}
	return nil, fmt.Errorf("model not found: %s", modelID)
}

// GetProviderConfig returns the provider configuration for a given provider name
func (c *RoutingConfig) GetProviderConfig(providerName string) (*ProviderConfig, error) {
	for i := range c.Providers {
		if c.Providers[i].Name == providerName {
			return &c.Providers[i], nil
		}
	}
	return nil, fmt.Errorf("provider not found: %s", providerName)
}
