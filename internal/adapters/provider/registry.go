// Package provider provides the provider registry for managing LLM providers.
package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// Registry manages the registration and lookup of LLM providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ports.ProviderPort
	order     []string // maintains registration order
}

// NewRegistry creates a new empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]ports.ProviderPort),
		order:     make([]string, 0),
	}
}

// Register adds a provider to the registry.
// If a provider with the same name already exists, it will be replaced.
func (r *Registry) Register(provider ports.ProviderPort) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	info := provider.Info()
	if info.Name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered
	if _, exists := r.providers[info.Name]; !exists {
		r.order = append(r.order, info.Name)
	}

	r.providers[info.Name] = provider
	return nil
}

// Get retrieves a provider by name.
// Returns nil if the provider is not found.
func (r *Registry) Get(name string) ports.ProviderPort {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[name]
}

// GetRequired retrieves a provider by name, returning an error if not found.
func (r *Registry) GetRequired(name string) (ports.ProviderPort, error) {
	provider := r.Get(name)
	if provider == nil {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// List returns all registered provider names in registration order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

// ListProviders returns all registered providers in registration order.
func (r *Registry) ListProviders() []ports.ProviderPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.ProviderPort, 0, len(r.order))
	for _, name := range r.order {
		if p, ok := r.providers[name]; ok {
			result = append(result, p)
		}
	}
	return result
}

// Remove removes a provider from the registry.
// Returns true if the provider was found and removed.
func (r *Registry) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; !exists {
		return false
	}

	delete(r.providers, name)

	// Remove from order slice
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}

	return true
}

// Count returns the number of registered providers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}

// Clear removes all providers from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[string]ports.ProviderPort)
	r.order = make([]string, 0)
}

// FindByModel returns the first provider that supports the given model.
func (r *Registry) FindByModel(ctx context.Context, modelID string) (ports.ProviderPort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		provider := r.providers[name]
		supported, err := provider.SupportsModel(ctx, modelID)
		if err != nil {
			continue // Skip providers that error
		}
		if supported {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("no provider found for model: %s", modelID)
}

// FindAvailable returns providers that are currently available.
func (r *Registry) FindAvailable(ctx context.Context) []ports.ProviderPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.ProviderPort, 0)
	for _, name := range r.order {
		provider := r.providers[name]
		// Check health with empty model to test general availability
		status, err := provider.HealthCheck(ctx, "")
		if err == nil && status.Healthy {
			result = append(result, provider)
		}
	}

	return result
}

// GetLocalProviders returns providers marked as local.
func (r *Registry) GetLocalProviders() []ports.ProviderPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.ProviderPort, 0)
	for _, name := range r.order {
		provider := r.providers[name]
		if provider.Info().IsLocal {
			result = append(result, provider)
		}
	}

	return result
}

// GetCloudProviders returns providers not marked as local.
func (r *Registry) GetCloudProviders() []ports.ProviderPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.ProviderPort, 0)
	for _, name := range r.order {
		provider := r.providers[name]
		if !provider.Info().IsLocal {
			result = append(result, provider)
		}
	}

	return result
}
