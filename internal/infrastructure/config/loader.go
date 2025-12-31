// Package config provides configuration loading and management.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loader handles loading configuration from files.
type Loader struct {
	configDir string
}

// NewLoader creates a new configuration loader.
// If configDir is empty, it defaults to ~/.skillrunner.
func NewLoader(configDir string) (*Loader, error) {
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".skillrunner")
	}

	return &Loader{configDir: configDir}, nil
}

// Load loads configuration from the specified file or default location.
// If the file doesn't exist, returns the default configuration.
func (l *Loader) Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = filepath.Join(l.configDir, "config.yaml")
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return NewDefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	cfg := NewDefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// LoadFromFile loads configuration from a specific file path.
// Returns an error if the file doesn't exist.
func (l *Loader) LoadFromFile(configPath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	cfg := NewDefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// Save saves configuration to the specified file or default location.
func (l *Loader) Save(cfg *Config, configPath string) error {
	if configPath == "" {
		configPath = filepath.Join(l.configDir, "config.yaml")
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add header comment
	header := `# Skillrunner Configuration
# Documentation: https://github.com/jbctechsolutions/skillrunner
#
`
	content := header + string(data)

	// Write file with restricted permissions
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ConfigDir returns the configuration directory path.
func (l *Loader) ConfigDir() string {
	return l.configDir
}

// DefaultConfigPath returns the default configuration file path.
func (l *Loader) DefaultConfigPath() string {
	return filepath.Join(l.configDir, "config.yaml")
}
