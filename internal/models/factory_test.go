package models

import (
	"os"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/config"
)

func TestExpandEnvVar(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_KEY", "test-value")
	os.Setenv("ANOTHER_KEY", "another-value")
	defer os.Unsetenv("TEST_KEY")
	defer os.Unsetenv("ANOTHER_KEY")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dollar brace format",
			input:    "${TEST_KEY}",
			expected: "test-value",
		},
		{
			name:     "dollar format",
			input:    "$TEST_KEY",
			expected: "test-value",
		},
		{
			name:     "plain text",
			input:    "plain-value",
			expected: "plain-value",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "non-existent variable brace",
			input:    "${NON_EXISTENT}",
			expected: "",
		},
		{
			name:     "non-existent variable dollar",
			input:    "$NON_EXISTENT",
			expected: "",
		},
		{
			name:     "another variable",
			input:    "${ANOTHER_KEY}",
			expected: "another-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVar(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVar(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCreateOllamaProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectNil   bool
		expectError bool
		expectedURL string
	}{
		{
			name: "new config format",
			config: &config.Config{
				Providers: &config.Providers{
					Ollama: &config.OllamaConfig{
						URL:     "http://localhost:11434",
						Enabled: true,
					},
				},
			},
			expectNil:   false,
			expectError: false,
			expectedURL: "http://localhost:11434",
		},
		{
			name: "legacy router URL",
			config: &config.Config{
				Router: config.Router{
					OllamaURL: "http://localhost:18433",
				},
			},
			expectNil:   false,
			expectError: false,
			expectedURL: "http://localhost:18433",
		},
		{
			name: "disabled in new config",
			config: &config.Config{
				Providers: &config.Providers{
					Ollama: &config.OllamaConfig{
						URL:     "http://localhost:11434",
						Enabled: false,
					},
				},
			},
			expectNil:   true,
			expectError: false,
		},
		{
			name:        "default URL when nothing specified",
			config:      &config.Config{},
			expectNil:   false,
			expectError: false,
			expectedURL: "http://localhost:11434",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := createOllamaProvider(tt.config)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectNil && provider != nil {
				t.Error("expected nil provider but got non-nil")
			}
			if !tt.expectNil && provider == nil {
				t.Error("expected non-nil provider but got nil")
			}

			// Verify URL if provider was created
			if provider != nil && !tt.expectNil {
				ollamaProvider, ok := provider.(*OllamaProvider)
				if !ok {
					t.Fatal("provider is not an OllamaProvider")
				}
				if ollamaProvider.baseURL != tt.expectedURL {
					t.Errorf("expected baseURL %q, got %q", tt.expectedURL, ollamaProvider.baseURL)
				}
			}
		})
	}
}

func TestCreateAnthropicProvider(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_ANTHROPIC_KEY", "sk-ant-test")
	defer os.Unsetenv("TEST_ANTHROPIC_KEY")

	tests := []struct {
		name        string
		config      *config.Config
		expectNil   bool
		expectError bool
	}{
		{
			name: "new config format with direct key",
			config: &config.Config{
				Providers: &config.Providers{
					Anthropic: &config.AnthropicConfig{
						APIKey:  "sk-ant-direct",
						Enabled: true,
					},
				},
			},
			expectNil:   false,
			expectError: false,
		},
		{
			name: "new config format with env var",
			config: &config.Config{
				Providers: &config.Providers{
					Anthropic: &config.AnthropicConfig{
						APIKey:  "${TEST_ANTHROPIC_KEY}",
						Enabled: true,
					},
				},
			},
			expectNil:   false,
			expectError: false,
		},
		{
			name: "legacy api_keys format",
			config: &config.Config{
				APIKeys: config.APIKeys{
					Anthropic: "sk-ant-legacy",
				},
			},
			expectNil:   false,
			expectError: false,
		},
		{
			name: "disabled in new config",
			config: &config.Config{
				Providers: &config.Providers{
					Anthropic: &config.AnthropicConfig{
						APIKey:  "sk-ant-test",
						Enabled: false,
					},
				},
			},
			expectNil:   true,
			expectError: false,
		},
		{
			name: "no api key configured",
			config: &config.Config{
				Providers: &config.Providers{
					Anthropic: &config.AnthropicConfig{
						Enabled: true,
					},
				},
			},
			expectNil:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := createAnthropicProvider(tt.config)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectNil && provider != nil {
				t.Error("expected nil provider but got non-nil")
			}
			if !tt.expectNil && provider == nil {
				t.Error("expected non-nil provider but got nil")
			}
		})
	}
}

func TestNewProvidersFromConfig(t *testing.T) {
	tests := []struct {
		name            string
		config          *config.Config
		expectError     bool
		minProviders    int
		maxProviders    int
		expectOllama    bool
		expectAnthropic bool
	}{
		{
			name: "both providers enabled",
			config: &config.Config{
				Providers: &config.Providers{
					Ollama: &config.OllamaConfig{
						URL:     "http://localhost:11434",
						Enabled: true,
					},
					Anthropic: &config.AnthropicConfig{
						APIKey:  "sk-ant-test",
						Enabled: true,
					},
				},
			},
			expectError:     false,
			minProviders:    2,
			maxProviders:    2,
			expectOllama:    true,
			expectAnthropic: true,
		},
		{
			name: "only ollama enabled",
			config: &config.Config{
				Providers: &config.Providers{
					Ollama: &config.OllamaConfig{
						URL:     "http://localhost:11434",
						Enabled: true,
					},
				},
			},
			expectError:     false,
			minProviders:    1,
			maxProviders:    1,
			expectOllama:    true,
			expectAnthropic: false,
		},
		{
			name: "legacy config",
			config: &config.Config{
				APIKeys: config.APIKeys{
					Anthropic: "sk-ant-legacy",
				},
				Router: config.Router{
					OllamaURL: "http://localhost:18433",
				},
			},
			expectError:     false,
			minProviders:    2,
			maxProviders:    2,
			expectOllama:    true,
			expectAnthropic: true,
		},
		{
			name:         "empty config uses defaults",
			config:       &config.Config{},
			expectError:  false,
			minProviders: 1, // At least Ollama with defaults
			maxProviders: 1,
			expectOllama: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers, err := NewProvidersFromConfig(tt.config)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(providers) < tt.minProviders {
				t.Errorf("expected at least %d providers, got %d", tt.minProviders, len(providers))
			}
			if len(providers) > tt.maxProviders {
				t.Errorf("expected at most %d providers, got %d", tt.maxProviders, len(providers))
			}

			// Check for specific providers
			hasOllama := false
			hasAnthropic := false
			for _, p := range providers {
				info := p.Info()
				if info.Name == "ollama" {
					hasOllama = true
				}
				if info.Name == "anthropic-live" {
					hasAnthropic = true
				}
			}

			if tt.expectOllama && !hasOllama {
				t.Error("expected Ollama provider but it was not found")
			}
			if !tt.expectOllama && hasOllama {
				t.Error("did not expect Ollama provider but it was found")
			}
			if tt.expectAnthropic && !hasAnthropic {
				t.Error("expected Anthropic provider but it was not found")
			}
			if !tt.expectAnthropic && hasAnthropic {
				t.Error("did not expect Anthropic provider but it was found")
			}
		})
	}
}
