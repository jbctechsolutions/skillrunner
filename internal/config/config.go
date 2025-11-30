package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the Skillrunner configuration
type Config struct {
	APIKeys       APIKeys        `yaml:"api_keys"`
	Router        Router         `yaml:"router"`
	Docker        Docker         `yaml:"docker"`
	ModelDefaults *ModelDefaults `yaml:"model_defaults,omitempty"`
	Providers     *Providers     `yaml:"providers,omitempty"` // NEW unified provider configuration
	Paths         *Paths         `yaml:"paths,omitempty"`     // Paths configuration
}

// Paths contains path configuration
type Paths struct {
	SkillrunnerDir string `yaml:"skillrunner_dir,omitempty"` // Override ~/.skillrunner directory
}

// APIKeys contains API keys for various providers
type APIKeys struct {
	Anthropic string `yaml:"anthropic,omitempty"`
}

// Router contains router-specific settings
type Router struct {
	LiteLLMURL string `yaml:"litellm_url,omitempty"`
	OllamaURL  string `yaml:"ollama_url,omitempty"`
	AutoStart  bool   `yaml:"auto_start,omitempty"`
}

// Docker contains Docker-specific settings
type Docker struct {
	ComposeFile string `yaml:"compose_file,omitempty"`
	ProjectDir  string `yaml:"project_dir,omitempty"`
}

// Manager handles configuration loading and saving
type Manager struct {
	configPath string
	config     *Config
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".skillrunner")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// NewManager creates a new configuration manager
func NewManager(configPath string) (*Manager, error) {
	if configPath == "" {
		var err error
		configPath, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	manager := &Manager{
		configPath: configPath,
		config:     &Config{},
	}

	// Load existing config if it exists
	if _, err := os.Stat(configPath); err == nil {
		if err := manager.Load(); err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
	} else {
		// Create default config
		manager.config = &Config{
			Router: Router{
				LiteLLMURL: "http://localhost:18432",
				OllamaURL:  "http://localhost:18433",
				AutoStart:  false, // User must explicitly enable auto-start
			},
			ModelDefaults: DefaultModelDefaults(),
		}
	}

	return manager, nil
}

// Load loads configuration from file
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	return nil
}

// Save saves configuration to file
func (m *Manager) Save() error {
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Set restrictive permissions (owner read/write only)
	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	return m.config
}

// SetAPIKey sets an API key for a provider
func (m *Manager) SetAPIKey(provider, key string) error {
	switch provider {
	case "anthropic":
		m.config.APIKeys.Anthropic = key
	default:
		return fmt.Errorf("unknown provider: %s (supported: anthropic)", provider)
	}

	return m.Save()
}

// GetAPIKey returns an API key for a provider
func (m *Manager) GetAPIKey(provider string) string {
	switch provider {
	case "anthropic":
		return m.config.APIKeys.Anthropic
	default:
		return ""
	}
}

// GetEnvVars returns environment variables for Docker Compose
// This merges config file values with existing environment variables
func (m *Manager) GetEnvVars() map[string]string {
	env := make(map[string]string)

	// Check environment first (takes precedence)
	if val := os.Getenv("ANTHROPIC_API_KEY"); val != "" {
		env["ANTHROPIC_API_KEY"] = val
	} else if m.config.APIKeys.Anthropic != "" {
		env["ANTHROPIC_API_KEY"] = m.config.APIKeys.Anthropic
	}

	return env
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// Providers contains unified provider configuration
type Providers struct {
	Ollama    *OllamaConfig    `yaml:"ollama,omitempty"`
	Anthropic *AnthropicConfig `yaml:"anthropic,omitempty"`
}

// OllamaConfig contains Ollama-specific configuration
type OllamaConfig struct {
	URL     string `yaml:"url,omitempty"`
	Enabled bool   `yaml:"enabled"`
}

// AnthropicConfig contains Anthropic-specific configuration
type AnthropicConfig struct {
	APIKey  string `yaml:"api_key,omitempty"`
	Enabled bool   `yaml:"enabled"`
}
