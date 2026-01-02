package backend

import (
	"context"
	"fmt"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// mockBackend is a mock implementation of BackendPort for testing
type mockBackend struct {
	name                string
	supportsModel       bool
	modelControlSupport bool
}

func (m *mockBackend) Info() ports.BackendInfo {
	return ports.BackendInfo{
		Name:        m.name,
		Version:     "1.0.0",
		Description: "Mock backend",
		Executable:  "/usr/bin/mock",
		Features:    []string{"test"},
	}
}

func (m *mockBackend) Start(ctx context.Context, workspace string, config session.BackendConfig) (*session.Session, error) {
	return nil, nil
}

func (m *mockBackend) Attach(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockBackend) Detach(ctx context.Context) error {
	return nil
}

func (m *mockBackend) Kill(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockBackend) InjectContext(ctx context.Context, sessionID, content string) error {
	return nil
}

func (m *mockBackend) InjectFile(ctx context.Context, sessionID, path string) error {
	return nil
}

func (m *mockBackend) GetStatus(ctx context.Context, sessionID string) (*ports.SessionStatus, error) {
	return nil, nil
}

func (m *mockBackend) GetTokenUsage(ctx context.Context, sessionID string) (*session.TokenUsage, error) {
	return nil, nil
}

func (m *mockBackend) SetModel(ctx context.Context, model string) error {
	return nil
}

func (m *mockBackend) GetSupportedModels(ctx context.Context) ([]string, error) {
	if m.supportsModel {
		return []string{"gpt-4", "claude-3"}, nil
	}
	return []string{}, nil
}

func (m *mockBackend) SupportsModelControl() bool {
	return m.modelControlSupport
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if registry.Count() != 0 {
		t.Errorf("Expected empty registry, got %d backends", registry.Count())
	}
}

func TestRegister(t *testing.T) {
	registry := NewRegistry()

	// Test registering a backend
	backend := &mockBackend{name: "test-backend"}
	err := registry.Register(backend)
	if err != nil {
		t.Fatalf("Failed to register backend: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Expected 1 backend, got %d", registry.Count())
	}

	// Test registering nil backend
	err = registry.Register(nil)
	if err == nil {
		t.Error("Expected error when registering nil backend")
	}

	// Test registering backend with empty name
	emptyBackend := &mockBackend{name: ""}
	err = registry.Register(emptyBackend)
	if err == nil {
		t.Error("Expected error when registering backend with empty name")
	}
}

func TestGet(t *testing.T) {
	registry := NewRegistry()
	backend := &mockBackend{name: "test-backend"}
	registry.Register(backend)

	// Test getting existing backend
	retrieved := registry.Get("test-backend")
	if retrieved == nil {
		t.Error("Failed to retrieve registered backend")
	}

	if retrieved.Info().Name != "test-backend" {
		t.Errorf("Expected backend name 'test-backend', got '%s'", retrieved.Info().Name)
	}

	// Test getting non-existent backend
	missing := registry.Get("missing-backend")
	if missing != nil {
		t.Error("Expected nil for non-existent backend")
	}
}

func TestGetRequired(t *testing.T) {
	registry := NewRegistry()
	backend := &mockBackend{name: "test-backend"}
	registry.Register(backend)

	// Test getting existing backend
	retrieved, err := registry.GetRequired("test-backend")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if retrieved == nil {
		t.Error("Failed to retrieve registered backend")
	}

	// Test getting non-existent backend
	_, err = registry.GetRequired("missing-backend")
	if err == nil {
		t.Error("Expected error for non-existent backend")
	}
}

func TestList(t *testing.T) {
	registry := NewRegistry()

	// Register multiple backends
	registry.Register(&mockBackend{name: "backend1"})
	registry.Register(&mockBackend{name: "backend2"})
	registry.Register(&mockBackend{name: "backend3"})

	names := registry.List()
	if len(names) != 3 {
		t.Errorf("Expected 3 backends, got %d", len(names))
	}

	// Check order is preserved
	expected := []string{"backend1", "backend2", "backend3"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Expected backend %d to be '%s', got '%s'", i, expected[i], name)
		}
	}
}

func TestListBackends(t *testing.T) {
	registry := NewRegistry()

	// Register multiple backends
	b1 := &mockBackend{name: "backend1"}
	b2 := &mockBackend{name: "backend2"}
	registry.Register(b1)
	registry.Register(b2)

	backends := registry.ListBackends()
	if len(backends) != 2 {
		t.Errorf("Expected 2 backends, got %d", len(backends))
	}
}

func TestRemove(t *testing.T) {
	registry := NewRegistry()
	backend := &mockBackend{name: "test-backend"}
	registry.Register(backend)

	// Test removing existing backend
	removed := registry.Remove("test-backend")
	if !removed {
		t.Error("Failed to remove existing backend")
	}

	if registry.Count() != 0 {
		t.Errorf("Expected 0 backends after removal, got %d", registry.Count())
	}

	// Test removing non-existent backend
	removed = registry.Remove("missing-backend")
	if removed {
		t.Error("Should not return true for removing non-existent backend")
	}
}

func TestClear(t *testing.T) {
	registry := NewRegistry()

	// Register multiple backends
	registry.Register(&mockBackend{name: "backend1"})
	registry.Register(&mockBackend{name: "backend2"})
	registry.Register(&mockBackend{name: "backend3"})

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Expected 0 backends after clear, got %d", registry.Count())
	}
}

func TestFindByModel(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	// Register backends with different model support
	registry.Register(&mockBackend{name: "backend1", supportsModel: true})
	registry.Register(&mockBackend{name: "backend2", supportsModel: false})

	// Test finding backend that supports model
	backend, err := registry.FindByModel(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if backend.Info().Name != "backend1" {
		t.Errorf("Expected 'backend1', got '%s'", backend.Info().Name)
	}

	// Test finding backend for unsupported model
	_, err = registry.FindByModel(ctx, "unsupported-model")
	if err == nil {
		t.Error("Expected error for unsupported model")
	}
}

func TestGetWithModelControl(t *testing.T) {
	registry := NewRegistry()

	// Register backends with different model control support
	registry.Register(&mockBackend{name: "backend1", modelControlSupport: true})
	registry.Register(&mockBackend{name: "backend2", modelControlSupport: false})
	registry.Register(&mockBackend{name: "backend3", modelControlSupport: true})

	backends := registry.GetWithModelControl()
	if len(backends) != 2 {
		t.Errorf("Expected 2 backends with model control, got %d", len(backends))
	}

	// Verify the backends
	for _, b := range backends {
		if !b.SupportsModelControl() {
			t.Errorf("Backend '%s' should support model control", b.Info().Name)
		}
	}
}

func TestGetWithoutModelControl(t *testing.T) {
	registry := NewRegistry()

	// Register backends with different model control support
	registry.Register(&mockBackend{name: "backend1", modelControlSupport: true})
	registry.Register(&mockBackend{name: "backend2", modelControlSupport: false})
	registry.Register(&mockBackend{name: "backend3", modelControlSupport: false})

	backends := registry.GetWithoutModelControl()
	if len(backends) != 2 {
		t.Errorf("Expected 2 backends without model control, got %d", len(backends))
	}

	// Verify the backends
	for _, b := range backends {
		if b.SupportsModelControl() {
			t.Errorf("Backend '%s' should not support model control", b.Info().Name)
		}
	}
}

func TestConcurrency(t *testing.T) {
	registry := NewRegistry()

	// Test concurrent registrations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			backend := &mockBackend{name: fmt.Sprintf("backend%d", id)}
			registry.Register(backend)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if registry.Count() != 10 {
		t.Errorf("Expected 10 backends after concurrent registration, got %d", registry.Count())
	}
}
