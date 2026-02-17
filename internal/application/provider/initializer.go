// Package provider provides provider initialization and health checking functionality.
package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/adapters/provider/anthropic"
	"github.com/jbctechsolutions/skillrunner/internal/adapters/provider/groq"
	"github.com/jbctechsolutions/skillrunner/internal/adapters/provider/ollama"
	"github.com/jbctechsolutions/skillrunner/internal/adapters/provider/openai"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/crypto"
)

// ProviderHealth contains health status information for a provider.
type ProviderHealth struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"` // "local" or "cloud"
	Enabled   bool          `json:"enabled"`
	Healthy   bool          `json:"healthy"`
	Latency   time.Duration `json:"latency,omitempty"`
	Error     string        `json:"error,omitempty"`
	Models    []string      `json:"models,omitempty"`
	Endpoint  string        `json:"endpoint,omitempty"`
	APIKeySet bool          `json:"api_key_set,omitempty"` // For cloud providers
}

// Initializer manages provider initialization from configuration.
type Initializer struct {
	registry  *adapterProvider.Registry
	config    *config.Config
	encryptor *crypto.Encryptor
	mu        sync.RWMutex
	health    map[string]*ProviderHealth
}

// NewInitializer creates a new provider initializer.
// Returns an error if the encryptor cannot be initialized.
func NewInitializer(registry *adapterProvider.Registry) (*Initializer, error) {
	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encryptor: %w", err)
	}

	return &Initializer{
		registry:  registry,
		encryptor: encryptor,
		health:    make(map[string]*ProviderHealth),
	}, nil
}

// InitFromConfig initializes providers based on the configuration.
// It registers enabled providers with the registry.
func (i *Initializer) InitFromConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	i.mu.Lock()
	i.config = cfg
	i.mu.Unlock()

	var errs []error

	// Initialize Ollama if enabled
	if cfg.Providers.Ollama.Enabled {
		if err := i.initOllama(cfg.Providers.Ollama); err != nil {
			errs = append(errs, fmt.Errorf("ollama: %w", err))
		}
	} else {
		i.setProviderHealth("ollama", &ProviderHealth{
			Name:     "ollama",
			Type:     "local",
			Enabled:  false,
			Healthy:  false,
			Endpoint: cfg.Providers.Ollama.URL,
		})
	}

	// Initialize Anthropic if enabled
	if cfg.Providers.Anthropic.Enabled {
		if err := i.initAnthropic(cfg.Providers.Anthropic); err != nil {
			errs = append(errs, fmt.Errorf("anthropic: %w", err))
		}
	} else {
		i.setProviderHealth("anthropic", &ProviderHealth{
			Name:      "anthropic",
			Type:      "cloud",
			Enabled:   false,
			Healthy:   false,
			APIKeySet: cfg.Providers.Anthropic.APIKeyEncrypted != "",
		})
	}

	// Initialize OpenAI if enabled
	if cfg.Providers.OpenAI.Enabled {
		if err := i.initOpenAI(cfg.Providers.OpenAI); err != nil {
			errs = append(errs, fmt.Errorf("openai: %w", err))
		}
	} else {
		i.setProviderHealth("openai", &ProviderHealth{
			Name:      "openai",
			Type:      "cloud",
			Enabled:   false,
			Healthy:   false,
			APIKeySet: cfg.Providers.OpenAI.APIKeyEncrypted != "",
		})
	}

	// Initialize Groq if enabled
	if cfg.Providers.Groq.Enabled {
		if err := i.initGroq(cfg.Providers.Groq); err != nil {
			errs = append(errs, fmt.Errorf("groq: %w", err))
		}
	} else {
		i.setProviderHealth("groq", &ProviderHealth{
			Name:      "groq",
			Type:      "cloud",
			Enabled:   false,
			Healthy:   false,
			APIKeySet: cfg.Providers.Groq.APIKeyEncrypted != "",
		})
	}

	if len(errs) > 0 {
		// Return combined error but don't fail completely
		// Some providers may have initialized successfully
		return fmt.Errorf("some providers failed to initialize: %v", errs)
	}

	return nil
}

// initOllama initializes the Ollama provider.
func (i *Initializer) initOllama(cfg config.OllamaConfig) error {
	url := cfg.URL
	if url == "" {
		url = config.DefaultOllamaURL
	}

	clientOpts := []ollama.ClientOption{ollama.WithBaseURL(url)}
	if cfg.Timeout > 0 {
		clientOpts = append(clientOpts, ollama.WithTimeout(cfg.Timeout))
	}
	provider := ollama.NewProvider(ollama.WithClient(ollama.NewClient(clientOpts...)))
	if err := i.registry.Register(provider); err != nil {
		return err
	}

	i.setProviderHealth("ollama", &ProviderHealth{
		Name:     "ollama",
		Type:     "local",
		Enabled:  true,
		Endpoint: url,
	})

	return nil
}

// initAnthropic initializes the Anthropic provider.
func (i *Initializer) initAnthropic(cfg config.CloudConfig) error {
	if cfg.APIKeyEncrypted == "" {
		return fmt.Errorf("API key not configured")
	}

	// Decrypt the API key using AES-256-GCM
	apiKey, err := i.encryptor.Decrypt(cfg.APIKeyEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt API key: %w", err)
	}

	providerCfg := anthropic.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		providerCfg.BaseURL = cfg.BaseURL
	}
	if cfg.Timeout > 0 {
		providerCfg.Timeout = cfg.Timeout
	}

	provider := anthropic.NewProvider(providerCfg)
	if err := i.registry.Register(provider); err != nil {
		return err
	}

	i.setProviderHealth("anthropic", &ProviderHealth{
		Name:      "anthropic",
		Type:      "cloud",
		Enabled:   true,
		APIKeySet: true,
		Endpoint:  providerCfg.BaseURL,
	})

	return nil
}

// initOpenAI initializes the OpenAI provider.
func (i *Initializer) initOpenAI(cfg config.CloudConfig) error {
	if cfg.APIKeyEncrypted == "" {
		return fmt.Errorf("API key not configured")
	}

	// Decrypt the API key using AES-256-GCM
	apiKey, err := i.encryptor.Decrypt(cfg.APIKeyEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt API key: %w", err)
	}

	providerCfg := openai.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		providerCfg.BaseURL = cfg.BaseURL
	}
	if cfg.Timeout > 0 {
		providerCfg.Timeout = cfg.Timeout
	}

	provider := openai.NewProvider(providerCfg)
	if err := i.registry.Register(provider); err != nil {
		return err
	}

	i.setProviderHealth("openai", &ProviderHealth{
		Name:      "openai",
		Type:      "cloud",
		Enabled:   true,
		APIKeySet: true,
		Endpoint:  providerCfg.BaseURL,
	})

	return nil
}

// initGroq initializes the Groq provider.
func (i *Initializer) initGroq(cfg config.CloudConfig) error {
	if cfg.APIKeyEncrypted == "" {
		return fmt.Errorf("API key not configured")
	}

	// Decrypt the API key using AES-256-GCM
	apiKey, err := i.encryptor.Decrypt(cfg.APIKeyEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt API key: %w", err)
	}

	providerCfg := groq.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		providerCfg.BaseURL = cfg.BaseURL
	}
	if cfg.Timeout > 0 {
		providerCfg.Timeout = cfg.Timeout
	}

	provider := groq.NewProvider(providerCfg)
	if err := i.registry.Register(provider); err != nil {
		return err
	}

	i.setProviderHealth("groq", &ProviderHealth{
		Name:      "groq",
		Type:      "cloud",
		Enabled:   true,
		APIKeySet: true,
		Endpoint:  providerCfg.BaseURL,
	})

	return nil
}

// CheckHealth performs health checks on all registered providers.
// It updates the internal health state and returns the results.
func (i *Initializer) CheckHealth(ctx context.Context) map[string]*ProviderHealth {
	providers := i.registry.ListProviders()
	results := make(map[string]*ProviderHealth)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, p := range providers {
		wg.Add(1)
		go func(provider ports.ProviderPort) {
			defer wg.Done()

			info := provider.Info()
			health := i.checkProviderHealth(ctx, provider)

			mu.Lock()
			results[info.Name] = health
			mu.Unlock()

			i.setProviderHealth(info.Name, health)
		}(p)
	}

	wg.Wait()

	// Include disabled providers in results
	i.mu.RLock()
	for name, h := range i.health {
		if _, exists := results[name]; !exists {
			results[name] = h
		}
	}
	i.mu.RUnlock()

	return results
}

// checkProviderHealth performs a health check on a single provider.
func (i *Initializer) checkProviderHealth(ctx context.Context, provider ports.ProviderPort) *ProviderHealth {
	info := provider.Info()

	health := &ProviderHealth{
		Name:     info.Name,
		Enabled:  true,
		Endpoint: info.BaseURL,
	}

	if info.IsLocal {
		health.Type = "local"
	} else {
		health.Type = "cloud"
		health.APIKeySet = true // If registered, the key was set
	}

	// Get available models
	models, err := provider.ListModels(ctx)
	if err != nil {
		health.Healthy = false
		health.Error = fmt.Sprintf("failed to list models: %v", err)
		return health
	}
	health.Models = models

	// Perform health check
	status, err := provider.HealthCheck(ctx, "")
	if err != nil {
		health.Healthy = false
		health.Error = err.Error()
		return health
	}

	health.Healthy = status.Healthy
	health.Latency = status.Latency
	if !status.Healthy {
		health.Error = status.Message
	}

	return health
}

// GetHealth returns the cached health status for a provider.
func (i *Initializer) GetHealth(name string) *ProviderHealth {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.health[name]
}

// GetAllHealth returns the cached health status for all providers.
func (i *Initializer) GetAllHealth() map[string]*ProviderHealth {
	i.mu.RLock()
	defer i.mu.RUnlock()

	result := make(map[string]*ProviderHealth)
	for k, v := range i.health {
		result[k] = v
	}
	return result
}

// GetAvailableModels returns models grouped by provider.
func (i *Initializer) GetAvailableModels(ctx context.Context) map[string][]string {
	providers := i.registry.ListProviders()
	result := make(map[string][]string)

	for _, p := range providers {
		info := p.Info()
		models, err := p.ListModels(ctx)
		if err != nil {
			continue
		}
		result[info.Name] = models
	}

	return result
}

// setProviderHealth updates the cached health for a provider.
func (i *Initializer) setProviderHealth(name string, health *ProviderHealth) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.health[name] = health
}

// Registry returns the underlying provider registry.
func (i *Initializer) Registry() *adapterProvider.Registry {
	return i.registry
}
