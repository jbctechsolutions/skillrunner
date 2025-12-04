package models

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/config"
)

// NewProvidersFromConfig creates model providers from configuration.
// It supports both the new unified Providers section and legacy config locations
// for backwards compatibility.
//
// Supported providers:
// - Ollama (local models, default enabled)
// - Anthropic (Claude models)
// - Groq (ultra-fast inference with Llama 3.1, Mixtral)
func NewProvidersFromConfig(cfg *config.Config) ([]ModelProvider, error) {
	var providers []ModelProvider

	// Ollama provider (default enabled for backwards compatibility)
	ollamaProvider, err := createOllamaProvider(cfg)
	if err == nil && ollamaProvider != nil {
		providers = append(providers, ollamaProvider)
	}

	// Anthropic provider
	anthropicProvider, err := createAnthropicProvider(cfg)
	if err == nil && anthropicProvider != nil {
		providers = append(providers, anthropicProvider)
	}

	// Groq provider
	groqProvider, err := createGroqProvider(cfg)
	if err == nil && groqProvider != nil {
		providers = append(providers, groqProvider)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers configured or available")
	}

	return providers, nil
}

// createOllamaProvider creates an Ollama provider with backwards compatibility.
func createOllamaProvider(cfg *config.Config) (ModelProvider, error) {
	var url string
	enabled := true // Default to enabled for backwards compatibility

	// Check new config format first
	if cfg.Providers != nil && cfg.Providers.Ollama != nil {
		enabled = cfg.Providers.Ollama.Enabled
		url = cfg.Providers.Ollama.URL
	}

	// Fall back to legacy Router.OllamaURL if not set
	if url == "" && cfg.Router.OllamaURL != "" {
		url = cfg.Router.OllamaURL
	}

	// Use default URL if still not set
	if url == "" {
		url = "http://localhost:11434"
	}

	// Skip if explicitly disabled
	if !enabled {
		return nil, nil
	}

	provider, err := NewOllamaProvider(url, &http.Client{Timeout: 15 * time.Minute})
	if err != nil {
		return nil, fmt.Errorf("create ollama provider: %w", err)
	}

	return provider, nil
}

// createAnthropicProvider creates an Anthropic provider with backwards compatibility.
func createAnthropicProvider(cfg *config.Config) (ModelProvider, error) {
	var apiKey string
	enabled := false

	// Check new config format first
	if cfg.Providers != nil && cfg.Providers.Anthropic != nil {
		enabled = cfg.Providers.Anthropic.Enabled
		apiKey = expandEnvVar(cfg.Providers.Anthropic.APIKey)
	}

	// Fall back to legacy APIKeys if not set
	if apiKey == "" && cfg.APIKeys.Anthropic != "" {
		apiKey = cfg.APIKeys.Anthropic
		enabled = true // If API key exists in old config, assume enabled
	}

	// Check environment variable as final fallback
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey != "" {
			enabled = true
		}
	}

	// Skip if not enabled or no API key
	if !enabled || apiKey == "" {
		return nil, nil
	}

	provider, err := NewAnthropicProvider(apiKey, "", &http.Client{Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("create anthropic provider: %w", err)
	}

	return provider, nil
}

// createGroqProvider creates a Groq provider for ultra-fast inference.
func createGroqProvider(cfg *config.Config) (ModelProvider, error) {
	var apiKey string
	enabled := false

	// Check new config format first
	if cfg.Providers != nil && cfg.Providers.Groq != nil {
		enabled = cfg.Providers.Groq.Enabled
		apiKey = expandEnvVar(cfg.Providers.Groq.APIKey)
	}

	// Check environment variable as final fallback
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
		if apiKey != "" {
			enabled = true
		}
	}

	// Skip if not enabled or no API key
	if !enabled || apiKey == "" {
		return nil, nil
	}

	provider, err := NewGroqProvider(apiKey, "", &http.Client{Timeout: 5 * time.Minute})
	if err != nil {
		return nil, fmt.Errorf("create groq provider: %w", err)
	}

	return provider, nil
}

// expandEnvVar expands environment variable references in a string.
// Supports both ${VAR_NAME} and $VAR_NAME formats.
func expandEnvVar(s string) string {
	if s == "" {
		return s
	}

	// Support ${VAR_NAME} format
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		varName := s[2 : len(s)-1]
		return os.Getenv(varName)
	}

	// Support $VAR_NAME format
	if strings.HasPrefix(s, "$") {
		varName := s[1:]
		return os.Getenv(varName)
	}

	// Return as-is if no environment variable pattern
	return s
}
