// Package config provides configuration structs and utilities for the skillrunner application.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadRoutingConfig loads a RoutingConfiguration from a YAML file.
// It reads the file, parses the YAML content, applies defaults, and validates the configuration.
// Returns an error if the file cannot be read, parsed, or fails validation.
func LoadRoutingConfig(path string) (*RoutingConfiguration, error) {
	if path == "" {
		return nil, errors.New("config path is empty")
	}

	// Clean and resolve the path
	cleanPath := filepath.Clean(path)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	return LoadRoutingConfigFromBytes(data)
}

// LoadRoutingConfigFromBytes parses YAML bytes into a RoutingConfiguration.
// It applies default values and validates the resulting configuration.
// Returns an error if the YAML is invalid or the configuration fails validation.
func LoadRoutingConfigFromBytes(data []byte) (*RoutingConfiguration, error) {
	if len(data) == 0 {
		return nil, errors.New("config data is empty")
	}

	cfg := &RoutingConfiguration{}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply defaults to fill in missing values
	cfg.SetDefaults()

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// SaveRoutingConfig writes a RoutingConfiguration to a YAML file.
// It creates parent directories if they don't exist.
// Returns an error if the configuration is nil or file operations fail.
func SaveRoutingConfig(path string, cfg *RoutingConfiguration) error {
	if path == "" {
		return errors.New("config path is empty")
	}

	if cfg == nil {
		return errors.New("config is nil")
	}

	// Clean and resolve the path
	cleanPath := filepath.Clean(path)

	// Ensure parent directory exists
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory %q: %w", dir, err)
	}

	// Marshal configuration to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file with readable permissions
	if err := os.WriteFile(cleanPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file %q: %w", path, err)
	}

	return nil
}

// MergeRoutingConfigs merges multiple RoutingConfigurations into a single configuration.
// Configurations are merged in order, with later configurations taking precedence.
// The first non-nil configuration serves as the base.
// Returns nil if no configurations are provided or all are nil.
func MergeRoutingConfigs(configs ...*RoutingConfiguration) *RoutingConfiguration {
	if len(configs) == 0 {
		return nil
	}

	// Find the first non-nil configuration to use as base
	var result *RoutingConfiguration
	startIdx := 0

	for i, cfg := range configs {
		if cfg != nil {
			// Deep copy the first non-nil config as our base
			result = deepCopyRoutingConfig(cfg)
			startIdx = i + 1
			break
		}
	}

	if result == nil {
		return nil
	}

	// Merge remaining configurations
	for i := startIdx; i < len(configs); i++ {
		if configs[i] != nil {
			result.Merge(configs[i])
		}
	}

	return result
}

// deepCopyRoutingConfig creates a deep copy of a RoutingConfiguration.
func deepCopyRoutingConfig(src *RoutingConfiguration) *RoutingConfiguration {
	if src == nil {
		return nil
	}

	dst := &RoutingConfiguration{
		DefaultProvider: src.DefaultProvider,
	}

	// Copy fallback chain
	if src.FallbackChain != nil {
		dst.FallbackChain = make([]string, len(src.FallbackChain))
		copy(dst.FallbackChain, src.FallbackChain)
	}

	// Deep copy providers
	if src.Providers != nil {
		dst.Providers = make(map[string]*ProviderConfiguration, len(src.Providers))
		for name, provider := range src.Providers {
			dst.Providers[name] = deepCopyProviderConfig(provider)
		}
	}

	// Deep copy profiles
	if src.Profiles != nil {
		dst.Profiles = make(map[string]*ProfileConfiguration, len(src.Profiles))
		for name, profile := range src.Profiles {
			dst.Profiles[name] = deepCopyProfileConfig(profile)
		}
	}

	return dst
}

// deepCopyProviderConfig creates a deep copy of a ProviderConfiguration.
func deepCopyProviderConfig(src *ProviderConfiguration) *ProviderConfiguration {
	if src == nil {
		return nil
	}

	dst := &ProviderConfiguration{
		Enabled:  src.Enabled,
		Priority: src.Priority,
		BaseURL:  src.BaseURL,
		Timeout:  src.Timeout,
	}

	// Deep copy rate limits
	if src.RateLimits != nil {
		dst.RateLimits = &RateLimitConfiguration{
			RequestsPerMinute:  src.RateLimits.RequestsPerMinute,
			TokensPerMinute:    src.RateLimits.TokensPerMinute,
			ConcurrentRequests: src.RateLimits.ConcurrentRequests,
			BurstLimit:         src.RateLimits.BurstLimit,
		}
	}

	// Deep copy models
	if src.Models != nil {
		dst.Models = make(map[string]*ModelConfiguration, len(src.Models))
		for id, model := range src.Models {
			dst.Models[id] = deepCopyModelConfig(model)
		}
	}

	return dst
}

// deepCopyModelConfig creates a deep copy of a ModelConfiguration.
func deepCopyModelConfig(src *ModelConfiguration) *ModelConfiguration {
	if src == nil {
		return nil
	}

	dst := &ModelConfiguration{
		Tier:               src.Tier,
		CostPerInputToken:  src.CostPerInputToken,
		CostPerOutputToken: src.CostPerOutputToken,
		MaxTokens:          src.MaxTokens,
		ContextWindow:      src.ContextWindow,
		Enabled:            src.Enabled,
	}

	// Copy capabilities
	if src.Capabilities != nil {
		dst.Capabilities = make([]string, len(src.Capabilities))
		copy(dst.Capabilities, src.Capabilities)
	}

	// Copy aliases
	if src.Aliases != nil {
		dst.Aliases = make([]string, len(src.Aliases))
		copy(dst.Aliases, src.Aliases)
	}

	return dst
}

// deepCopyProfileConfig creates a deep copy of a ProfileConfiguration.
func deepCopyProfileConfig(src *ProfileConfiguration) *ProfileConfiguration {
	if src == nil {
		return nil
	}

	return &ProfileConfiguration{
		GenerationModel:  src.GenerationModel,
		ReviewModel:      src.ReviewModel,
		FallbackModel:    src.FallbackModel,
		MaxContextTokens: src.MaxContextTokens,
		PreferLocal:      src.PreferLocal,
	}
}

// LoadRoutingConfigWithDefaults loads a RoutingConfiguration from a file,
// falling back to default configuration if the file doesn't exist.
func LoadRoutingConfigWithDefaults(path string) (*RoutingConfiguration, error) {
	if path == "" {
		return NewRoutingConfiguration(), nil
	}

	// Check if file exists
	cleanPath := filepath.Clean(path)
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return NewRoutingConfiguration(), nil
	}

	return LoadRoutingConfig(path)
}

// LoadAndMergeRoutingConfigs loads multiple configuration files and merges them.
// Files are loaded in order, with later files taking precedence.
// Missing files are skipped without error.
// Returns an error only if a file exists but cannot be parsed.
func LoadAndMergeRoutingConfigs(paths ...string) (*RoutingConfiguration, error) {
	var configs []*RoutingConfiguration

	for _, path := range paths {
		if path == "" {
			continue
		}

		cleanPath := filepath.Clean(path)
		if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
			// Skip missing files
			continue
		}

		cfg, err := LoadRoutingConfig(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %q: %w", path, err)
		}

		configs = append(configs, cfg)
	}

	if len(configs) == 0 {
		return NewRoutingConfiguration(), nil
	}

	return MergeRoutingConfigs(configs...), nil
}
