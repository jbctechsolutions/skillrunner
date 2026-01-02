// Package sync provides the sync backend registry for managing sync backends.
package sync

import (
	"context"
	"fmt"
	"sync"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// Registry manages the registration and lookup of sync backends.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]ports.SyncBackendPort
	order    []string // maintains registration order
}

// NewRegistry creates a new empty sync backend registry.
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]ports.SyncBackendPort),
		order:    make([]string, 0),
	}
}

// Register adds a sync backend to the registry.
// If a backend with the same name already exists, it will be replaced.
func (r *Registry) Register(name string, backend ports.SyncBackendPort) error {
	if backend == nil {
		return fmt.Errorf("backend cannot be nil")
	}

	if name == "" {
		return fmt.Errorf("backend name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered
	if _, exists := r.backends[name]; !exists {
		r.order = append(r.order, name)
	}

	r.backends[name] = backend
	return nil
}

// Get retrieves a sync backend by name.
// Returns nil if the backend is not found.
func (r *Registry) Get(name string) ports.SyncBackendPort {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.backends[name]
}

// GetRequired retrieves a sync backend by name, returning an error if not found.
func (r *Registry) GetRequired(name string) (ports.SyncBackendPort, error) {
	backend := r.Get(name)
	if backend == nil {
		return nil, fmt.Errorf("sync backend not found: %s", name)
	}
	return backend, nil
}

// List returns all registered backend names in registration order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

// ListBackends returns all registered backends in registration order.
func (r *Registry) ListBackends() []ports.SyncBackendPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.SyncBackendPort, 0, len(r.order))
	for _, name := range r.order {
		if b, ok := r.backends[name]; ok {
			result = append(result, b)
		}
	}
	return result
}

// Remove removes a sync backend from the registry.
// Returns true if the backend was found and removed.
func (r *Registry) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[name]; !exists {
		return false
	}

	delete(r.backends, name)

	// Remove from order slice
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}

	return true
}

// Count returns the number of registered backends.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.backends)
}

// Clear removes all backends from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.backends = make(map[string]ports.SyncBackendPort)
	r.order = make([]string, 0)
}

// FindAvailable returns backends that are currently available.
func (r *Registry) FindAvailable(ctx context.Context) []ports.SyncBackendPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.SyncBackendPort, 0)
	for _, name := range r.order {
		backend := r.backends[name]
		available, err := backend.IsAvailable(ctx)
		if err == nil && available {
			result = append(result, backend)
		}
	}

	return result
}

// GetPrimary returns the first available backend.
// This is useful for getting the default/primary sync backend.
func (r *Registry) GetPrimary(ctx context.Context) (ports.SyncBackendPort, error) {
	available := r.FindAvailable(ctx)
	if len(available) == 0 {
		return nil, fmt.Errorf("no available sync backends")
	}
	return available[0], nil
}
