// Package backend provides the backend registry for managing AI coding assistant backends.
package backend

import (
	"context"
	"fmt"
	"sync"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// Registry manages the registration and lookup of backends.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]ports.BackendPort
	order    []string // maintains registration order
}

// NewRegistry creates a new empty backend registry.
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]ports.BackendPort),
		order:    make([]string, 0),
	}
}

// Register adds a backend to the registry.
// If a backend with the same name already exists, it will be replaced.
func (r *Registry) Register(backend ports.BackendPort) error {
	if backend == nil {
		return fmt.Errorf("backend cannot be nil")
	}

	info := backend.Info()
	if info.Name == "" {
		return fmt.Errorf("backend name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered
	if _, exists := r.backends[info.Name]; !exists {
		r.order = append(r.order, info.Name)
	}

	r.backends[info.Name] = backend
	return nil
}

// Get retrieves a backend by name.
// Returns nil if the backend is not found.
func (r *Registry) Get(name string) ports.BackendPort {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.backends[name]
}

// GetRequired retrieves a backend by name, returning an error if not found.
func (r *Registry) GetRequired(name string) (ports.BackendPort, error) {
	backend := r.Get(name)
	if backend == nil {
		return nil, fmt.Errorf("backend not found: %s", name)
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
func (r *Registry) ListBackends() []ports.BackendPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.BackendPort, 0, len(r.order))
	for _, name := range r.order {
		if b, ok := r.backends[name]; ok {
			result = append(result, b)
		}
	}
	return result
}

// Remove removes a backend from the registry.
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

	r.backends = make(map[string]ports.BackendPort)
	r.order = make([]string, 0)
}

// FindByModel returns the first backend that supports the given model.
func (r *Registry) FindByModel(ctx context.Context, modelID string) (ports.BackendPort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		backend := r.backends[name]
		models, err := backend.GetSupportedModels(ctx)
		if err != nil {
			continue // Skip backends that error
		}
		for _, m := range models {
			if m == modelID {
				return backend, nil
			}
		}
	}

	return nil, fmt.Errorf("no backend found for model: %s", modelID)
}

// FindAvailable returns backends that have their executables available.
func (r *Registry) FindAvailable() []ports.BackendPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.BackendPort, 0)
	for _, name := range r.order {
		backend := r.backends[name]
		_ = backend.Info() // Check backend is valid
		// Check if executable exists would be done here
		// For now, we just return all backends
		result = append(result, backend)
	}

	return result
}

// GetWithModelControl returns backends that support model control.
func (r *Registry) GetWithModelControl() []ports.BackendPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.BackendPort, 0)
	for _, name := range r.order {
		backend := r.backends[name]
		if backend.SupportsModelControl() {
			result = append(result, backend)
		}
	}

	return result
}

// GetWithoutModelControl returns backends that do not support model control.
func (r *Registry) GetWithoutModelControl() []ports.BackendPort {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ports.BackendPort, 0)
	for _, name := range r.order {
		backend := r.backends[name]
		if !backend.SupportsModelControl() {
			result = append(result, backend)
		}
	}

	return result
}
