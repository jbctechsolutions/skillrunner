package sync

import (
	"context"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// mockSyncBackend is a mock implementation of SyncBackendPort for testing
type mockSyncBackend struct {
	available bool
	pushError error
	pullError error
	state     *ports.SyncState
}

func (m *mockSyncBackend) Push(ctx context.Context, state *ports.SyncState) error {
	if m.pushError != nil {
		return m.pushError
	}
	m.state = state
	return nil
}

func (m *mockSyncBackend) Pull(ctx context.Context) (*ports.SyncState, error) {
	if m.pullError != nil {
		return nil, m.pullError
	}
	return m.state, nil
}

func (m *mockSyncBackend) HasUpdates(ctx context.Context, since time.Time) (bool, error) {
	return false, nil
}

func (m *mockSyncBackend) IsAvailable(ctx context.Context) (bool, error) {
	return m.available, nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if registry.Count() != 0 {
		t.Errorf("expected count 0, got %d", registry.Count())
	}
}

func TestRegister(t *testing.T) {
	registry := NewRegistry()
	backend := &mockSyncBackend{available: true}

	// Test successful registration
	err := registry.Register("test", backend)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("expected count 1, got %d", registry.Count())
	}

	// Test nil backend
	err = registry.Register("nil", nil)
	if err == nil {
		t.Error("expected error for nil backend")
	}

	// Test empty name
	err = registry.Register("", backend)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestGet(t *testing.T) {
	registry := NewRegistry()
	backend := &mockSyncBackend{available: true}

	registry.Register("test", backend)

	// Test successful get
	retrieved := registry.Get("test")
	if retrieved == nil {
		t.Error("expected backend, got nil")
	}

	// Test non-existent backend
	retrieved = registry.Get("nonexistent")
	if retrieved != nil {
		t.Error("expected nil, got backend")
	}
}

func TestGetRequired(t *testing.T) {
	registry := NewRegistry()
	backend := &mockSyncBackend{available: true}

	registry.Register("test", backend)

	// Test successful get
	retrieved, err := registry.GetRequired("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if retrieved == nil {
		t.Error("expected backend, got nil")
	}

	// Test non-existent backend
	_, err = registry.GetRequired("nonexistent")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestList(t *testing.T) {
	registry := NewRegistry()
	backend1 := &mockSyncBackend{available: true}
	backend2 := &mockSyncBackend{available: true}

	registry.Register("test1", backend1)
	registry.Register("test2", backend2)

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("expected 2 backends, got %d", len(list))
	}

	// Check order
	if list[0] != "test1" || list[1] != "test2" {
		t.Error("backends not in registration order")
	}
}

func TestListBackends(t *testing.T) {
	registry := NewRegistry()
	backend1 := &mockSyncBackend{available: true}
	backend2 := &mockSyncBackend{available: true}

	registry.Register("test1", backend1)
	registry.Register("test2", backend2)

	backends := registry.ListBackends()
	if len(backends) != 2 {
		t.Errorf("expected 2 backends, got %d", len(backends))
	}
}

func TestRemove(t *testing.T) {
	registry := NewRegistry()
	backend := &mockSyncBackend{available: true}

	registry.Register("test", backend)

	// Test successful removal
	removed := registry.Remove("test")
	if !removed {
		t.Error("expected true, got false")
	}

	if registry.Count() != 0 {
		t.Errorf("expected count 0, got %d", registry.Count())
	}

	// Test removing non-existent backend
	removed = registry.Remove("nonexistent")
	if removed {
		t.Error("expected false, got true")
	}
}

func TestClear(t *testing.T) {
	registry := NewRegistry()
	backend1 := &mockSyncBackend{available: true}
	backend2 := &mockSyncBackend{available: true}

	registry.Register("test1", backend1)
	registry.Register("test2", backend2)

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("expected count 0, got %d", registry.Count())
	}

	if len(registry.List()) != 0 {
		t.Error("expected empty list after clear")
	}
}

func TestFindAvailable(t *testing.T) {
	registry := NewRegistry()
	available := &mockSyncBackend{available: true}
	unavailable := &mockSyncBackend{available: false}

	registry.Register("available", available)
	registry.Register("unavailable", unavailable)

	ctx := context.Background()
	backends := registry.FindAvailable(ctx)

	if len(backends) != 1 {
		t.Errorf("expected 1 available backend, got %d", len(backends))
	}
}

func TestGetPrimary(t *testing.T) {
	registry := NewRegistry()
	backend1 := &mockSyncBackend{available: true}
	backend2 := &mockSyncBackend{available: true}

	registry.Register("first", backend1)
	registry.Register("second", backend2)

	ctx := context.Background()
	primary, err := registry.GetPrimary(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if primary != backend1 {
		t.Error("expected first backend as primary")
	}

	// Test with no available backends
	emptyRegistry := NewRegistry()
	_, err = emptyRegistry.GetPrimary(ctx)
	if err == nil {
		t.Error("expected error for empty registry")
	}
}

func TestRegistryThreadSafety(t *testing.T) {
	registry := NewRegistry()
	backend := &mockSyncBackend{available: true}

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			name := "backend"
			registry.Register(name, backend)
			registry.Get(name)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have registered the backend
	if registry.Count() == 0 {
		t.Error("expected at least one backend registered")
	}
}
