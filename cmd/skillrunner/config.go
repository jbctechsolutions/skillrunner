package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the CLI configuration
type Config struct {
	Workspace     string   `json:"workspace,omitempty"`
	DefaultModel  string   `json:"default_model,omitempty"`
	OutputFormat  string   `json:"output_format,omitempty"`
	CompactOutput bool     `json:"compact_output,omitempty"`
	Models        []string `json:"models,omitempty"`
}

// ConfigManager handles configuration loading and saving
type ConfigManager struct {
	configPath string
	config     *Config
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager() *ConfigManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configPath := filepath.Join(homeDir, ".skillrunner", "config.json")
	return &ConfigManager{
		configPath: configPath,
		config:     &Config{},
	}
}

// Load loads configuration from file or returns defaults
func (cm *ConfigManager) Load() (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Return default config
		return &Config{
			OutputFormat:  "table",
			CompactOutput: false,
			Models:        []string{"gpt-4", "claude-3", "gpt-3.5-turbo"},
		}, nil
	}

	// Read config file
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cm.config = &config
	return &config, nil
}

// Save saves configuration to file
func (cm *ConfigManager) Save(config *Config) error {
	// Ensure config directory exists
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	cm.config = config
	return nil
}

// GetConfigPath returns the path to the config file
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *Config {
	return &Config{
		OutputFormat:  "table",
		CompactOutput: false,
		Models:        []string{"gpt-4", "claude-3", "gpt-3.5-turbo"},
	}
}

// MergeConfig merges command-line flags with config file values
func MergeConfig(fileConfig *Config, workspace, model, format string, compact bool) *Config {
	merged := &Config{
		Workspace:     fileConfig.Workspace,
		DefaultModel:  fileConfig.DefaultModel,
		OutputFormat:  fileConfig.OutputFormat,
		CompactOutput: fileConfig.CompactOutput,
		Models:        fileConfig.Models,
	}

	// Override with command-line flags
	if workspace != "" {
		merged.Workspace = workspace
	}
	if model != "" {
		merged.DefaultModel = model
	}
	if format != "" {
		merged.OutputFormat = format
	}
	merged.CompactOutput = compact || fileConfig.CompactOutput

	return merged
}
