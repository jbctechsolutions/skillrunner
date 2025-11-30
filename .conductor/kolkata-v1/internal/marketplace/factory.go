package marketplace

import (
	"fmt"
)

// NewSource creates a Source from a SourceConfig
func NewSource(config SourceConfig) (Source, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("source %s is disabled", config.Name)
	}

	switch config.Type {
	case SourceTypeLocal:
		return NewLocalSource(config)
	case SourceTypeGitHub:
		return NewGitHubSource(config)
	case SourceTypeNPM:
		return NewNPMSource(config)
	case SourceTypeHTTP, SourceTypeRegistry:
		// HTTP and Registry sources are similar - both fetch from URLs
		// For now, treat registry as HTTP
		return NewHTTPSource(config)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", config.Type)
	}
}

// InitializeDefaultSources creates and adds default sources to a registry
func InitializeDefaultSources(registry *Registry) error {
	for _, config := range registry.DefaultConfigs() {
		if !config.Enabled {
			continue
		}

		source, err := NewSource(config)
		if err != nil {
			// Log warning but continue with other sources
			fmt.Printf("Warning: failed to initialize source %s: %v\n", config.Name, err)
			continue
		}

		if err := registry.AddSource(source); err != nil {
			fmt.Printf("Warning: failed to add source %s: %v\n", config.Name, err)
			continue
		}
	}

	return nil
}

// NewRegistryWithDefaults creates a new registry and initializes default sources
func NewRegistryWithDefaults() (*Registry, error) {
	registry := NewRegistry()

	if err := InitializeDefaultSources(registry); err != nil {
		return nil, err
	}

	return registry, nil
}
