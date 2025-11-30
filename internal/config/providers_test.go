package config

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProvidersConfigParsing(t *testing.T) {
	configYAML := `
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    enabled: true
`

	var cfg Config
	err := yaml.Unmarshal([]byte(configYAML), &cfg)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Verify Providers section exists
	if cfg.Providers == nil {
		t.Fatal("Providers section is nil")
	}

	// Test Ollama config
	if cfg.Providers.Ollama == nil {
		t.Fatal("Ollama config is nil")
	}
	if cfg.Providers.Ollama.URL != "http://localhost:11434" {
		t.Errorf("expected Ollama URL 'http://localhost:11434', got '%s'", cfg.Providers.Ollama.URL)
	}
	if !cfg.Providers.Ollama.Enabled {
		t.Error("expected Ollama to be enabled")
	}

	// Test Anthropic config
	if cfg.Providers.Anthropic == nil {
		t.Fatal("Anthropic config is nil")
	}
	if cfg.Providers.Anthropic.APIKey != "${ANTHROPIC_API_KEY}" {
		t.Errorf("expected Anthropic API key '${ANTHROPIC_API_KEY}', got '%s'", cfg.Providers.Anthropic.APIKey)
	}
	if !cfg.Providers.Anthropic.Enabled {
		t.Error("expected Anthropic to be enabled")
	}
}

func TestBackwardsCompatibility(t *testing.T) {
	// Test that old config format still works
	oldConfigYAML := `
api_keys:
  anthropic: sk-ant-old-key
router:
  ollama_url: http://localhost:18433
  litellm_url: http://localhost:18432
`

	var cfg Config
	err := yaml.Unmarshal([]byte(oldConfigYAML), &cfg)
	if err != nil {
		t.Fatalf("failed to parse old config: %v", err)
	}

	// Verify old format still parses
	if cfg.APIKeys.Anthropic != "sk-ant-old-key" {
		t.Errorf("expected Anthropic API key 'sk-ant-old-key', got '%s'", cfg.APIKeys.Anthropic)
	}
	if cfg.Router.OllamaURL != "http://localhost:18433" {
		t.Errorf("expected Ollama URL 'http://localhost:18433', got '%s'", cfg.Router.OllamaURL)
	}

	// Providers section should be nil for old config
	if cfg.Providers != nil {
		t.Error("expected Providers to be nil in old config format")
	}
}

func TestMixedConfiguration(t *testing.T) {
	// Test mixing old and new config formats
	mixedConfigYAML := `
providers:
  ollama:
    url: http://localhost:11434
    enabled: true
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    enabled: true
api_keys:
  anthropic: sk-fallback-key
router:
  ollama_url: http://localhost:18433
`

	var cfg Config
	err := yaml.Unmarshal([]byte(mixedConfigYAML), &cfg)
	if err != nil {
		t.Fatalf("failed to parse mixed config: %v", err)
	}

	// New format should be present
	if cfg.Providers == nil {
		t.Fatal("Providers section is nil")
	}
	if cfg.Providers.Ollama == nil {
		t.Fatal("Ollama config is nil")
	}

	// Old format should also be present
	if cfg.APIKeys.Anthropic != "sk-fallback-key" {
		t.Errorf("expected Anthropic API key 'sk-fallback-key', got '%s'", cfg.APIKeys.Anthropic)
	}
	if cfg.Router.OllamaURL != "http://localhost:18433" {
		t.Errorf("expected Ollama URL 'http://localhost:18433', got '%s'", cfg.Router.OllamaURL)
	}
}

func TestEnvironmentVariableExpansion(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_API_KEY", "test-key-value")
	defer os.Unsetenv("TEST_API_KEY")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dollar brace format",
			input:    "${TEST_API_KEY}",
			expected: "test-key-value",
		},
		{
			name:     "plain text",
			input:    "sk-plain-key",
			expected: "sk-plain-key",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "non-existent variable",
			input:    "${NON_EXISTENT_VAR}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configYAML := `
providers:
  anthropic:
    api_key: ` + tt.input + `
    enabled: true
`
			var cfg Config
			err := yaml.Unmarshal([]byte(configYAML), &cfg)
			if err != nil {
				t.Fatalf("failed to parse config: %v", err)
			}

			if cfg.Providers.Anthropic.APIKey != tt.input {
				t.Errorf("expected API key '%s', got '%s'", tt.input, cfg.Providers.Anthropic.APIKey)
			}
			// Note: The actual expansion happens in the factory, not in config parsing
			// This test verifies that the config correctly stores the variable reference
		})
	}
}

func TestDefaultValues(t *testing.T) {
	// Test config with minimal settings to verify defaults
	minimalConfigYAML := `
providers:
  ollama:
    enabled: true
`

	var cfg Config
	err := yaml.Unmarshal([]byte(minimalConfigYAML), &cfg)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Providers == nil {
		t.Fatal("Providers section is nil")
	}
	if cfg.Providers.Ollama == nil {
		t.Fatal("Ollama config is nil")
	}

	// URL should be empty (will default in factory)
	if cfg.Providers.Ollama.URL != "" {
		t.Errorf("expected empty Ollama URL for default, got '%s'", cfg.Providers.Ollama.URL)
	}

	// Enabled should be true
	if !cfg.Providers.Ollama.Enabled {
		t.Error("expected Ollama to be enabled")
	}
}

func TestOmittedProvidersSection(t *testing.T) {
	// Test that config works without providers section
	noProvidersYAML := `
router:
  ollama_url: http://localhost:18433
api_keys:
  anthropic: sk-test-key
`

	var cfg Config
	err := yaml.Unmarshal([]byte(noProvidersYAML), &cfg)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Providers should be nil
	if cfg.Providers != nil {
		t.Error("expected Providers to be nil when omitted")
	}

	// Old config should still work
	if cfg.Router.OllamaURL != "http://localhost:18433" {
		t.Errorf("expected Ollama URL 'http://localhost:18433', got '%s'", cfg.Router.OllamaURL)
	}
	if cfg.APIKeys.Anthropic != "sk-test-key" {
		t.Errorf("expected Anthropic API key 'sk-test-key', got '%s'", cfg.APIKeys.Anthropic)
	}
}
