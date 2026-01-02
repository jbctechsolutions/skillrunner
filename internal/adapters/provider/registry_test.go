package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// mockProvider implements ports.ProviderPort for testing
type mockProvider struct {
	name            string
	isLocal         bool
	healthy         bool
	supportedModels []string
}

func newMockProvider(name string, isLocal bool) *mockProvider {
	return &mockProvider{
		name:            name,
		isLocal:         isLocal,
		healthy:         true,
		supportedModels: []string{"model-1", "model-2"},
	}
}

func (m *mockProvider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:    m.name,
		IsLocal: m.isLocal,
	}
}

func (m *mockProvider) ListModels(_ context.Context) ([]string, error) {
	return m.supportedModels, nil
}

func (m *mockProvider) SupportsModel(_ context.Context, modelID string) (bool, error) {
	for _, model := range m.supportedModels {
		if model == modelID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockProvider) IsAvailable(_ context.Context, _ string) (bool, error) {
	return m.healthy, nil
}

func (m *mockProvider) Complete(_ context.Context, _ ports.CompletionRequest) (*ports.CompletionResponse, error) {
	return &ports.CompletionResponse{Content: "mock response"}, nil
}

func (m *mockProvider) Stream(_ context.Context, _ ports.CompletionRequest, _ ports.StreamCallback) (*ports.CompletionResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProvider) HealthCheck(_ context.Context, _ string) (*ports.HealthStatus, error) {
	return &ports.HealthStatus{
		Healthy:     m.healthy,
		Message:     "ok",
		LastChecked: time.Now(),
	}, nil
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.Count() != 0 {
		t.Errorf("expected empty registry, got count %d", r.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	t.Run("register valid provider", func(t *testing.T) {
		p := newMockProvider("test-provider", false)
		err := r.Register(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Count() != 1 {
			t.Errorf("expected count 1, got %d", r.Count())
		}
	})

	t.Run("register nil provider", func(t *testing.T) {
		err := r.Register(nil)
		if err == nil {
			t.Error("expected error for nil provider")
		}
	})

	t.Run("register duplicate replaces", func(t *testing.T) {
		p1 := newMockProvider("duplicate", false)
		p2 := newMockProvider("duplicate", true)

		r.Register(p1)
		initialCount := r.Count()

		r.Register(p2)
		if r.Count() != initialCount {
			t.Error("registering duplicate should not increase count")
		}

		// Verify the second one is used
		got := r.Get("duplicate")
		if got.Info().IsLocal != true {
			t.Error("expected second provider to replace first")
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	p := newMockProvider("test", false)
	r.Register(p)

	t.Run("get existing provider", func(t *testing.T) {
		got := r.Get("test")
		if got == nil {
			t.Fatal("expected to find provider")
		}
		if got.Info().Name != "test" {
			t.Errorf("expected name 'test', got %s", got.Info().Name)
		}
	})

	t.Run("get non-existent provider", func(t *testing.T) {
		got := r.Get("nonexistent")
		if got != nil {
			t.Error("expected nil for non-existent provider")
		}
	})
}

func TestRegistry_GetRequired(t *testing.T) {
	r := NewRegistry()
	p := newMockProvider("test", false)
	r.Register(p)

	t.Run("get existing provider", func(t *testing.T) {
		got, err := r.GetRequired("test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected to find provider")
		}
	})

	t.Run("get non-existent provider returns error", func(t *testing.T) {
		_, err := r.GetRequired("nonexistent")
		if err == nil {
			t.Error("expected error for non-existent provider")
		}
	})
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	// Register in specific order
	r.Register(newMockProvider("alpha", false))
	r.Register(newMockProvider("beta", false))
	r.Register(newMockProvider("gamma", false))

	names := r.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}

	// Verify order is maintained
	expected := []string{"alpha", "beta", "gamma"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], name)
		}
	}
}

func TestRegistry_ListProviders(t *testing.T) {
	r := NewRegistry()

	r.Register(newMockProvider("one", false))
	r.Register(newMockProvider("two", true))

	providers := r.ListProviders()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
}

func TestRegistry_Remove(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockProvider("test", false))

	t.Run("remove existing provider", func(t *testing.T) {
		removed := r.Remove("test")
		if !removed {
			t.Error("expected true for removed provider")
		}
		if r.Count() != 0 {
			t.Error("expected count 0 after removal")
		}
		if r.Get("test") != nil {
			t.Error("expected nil after removal")
		}
	})

	t.Run("remove non-existent provider", func(t *testing.T) {
		removed := r.Remove("nonexistent")
		if removed {
			t.Error("expected false for non-existent provider")
		}
	})
}

func TestRegistry_Clear(t *testing.T) {
	r := NewRegistry()
	r.Register(newMockProvider("one", false))
	r.Register(newMockProvider("two", false))

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("expected count 0 after clear, got %d", r.Count())
	}
	if len(r.List()) != 0 {
		t.Error("expected empty list after clear")
	}
}

func TestRegistry_FindByModel(t *testing.T) {
	r := NewRegistry()

	p1 := newMockProvider("provider1", false)
	p1.supportedModels = []string{"gpt-4", "gpt-3.5"}

	p2 := newMockProvider("provider2", true)
	p2.supportedModels = []string{"llama2", "mistral"}

	r.Register(p1)
	r.Register(p2)

	ctx := context.Background()

	t.Run("find provider for supported model", func(t *testing.T) {
		provider, err := r.FindByModel(ctx, "llama2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if provider.Info().Name != "provider2" {
			t.Errorf("expected provider2, got %s", provider.Info().Name)
		}
	})

	t.Run("find returns first match in registration order", func(t *testing.T) {
		// Add model to both providers
		p1.supportedModels = append(p1.supportedModels, "shared-model")
		p2.supportedModels = append(p2.supportedModels, "shared-model")

		provider, err := r.FindByModel(ctx, "shared-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return first registered (provider1)
		if provider.Info().Name != "provider1" {
			t.Errorf("expected provider1 (first registered), got %s", provider.Info().Name)
		}
	})

	t.Run("error for unsupported model", func(t *testing.T) {
		_, err := r.FindByModel(ctx, "unknown-model")
		if err == nil {
			t.Error("expected error for unsupported model")
		}
	})
}

func TestRegistry_FindAvailable(t *testing.T) {
	r := NewRegistry()

	p1 := newMockProvider("healthy", false)
	p1.healthy = true

	p2 := newMockProvider("unhealthy", false)
	p2.healthy = false

	r.Register(p1)
	r.Register(p2)

	available := r.FindAvailable(context.Background())
	if len(available) != 1 {
		t.Fatalf("expected 1 available provider, got %d", len(available))
	}
	if available[0].Info().Name != "healthy" {
		t.Errorf("expected healthy provider, got %s", available[0].Info().Name)
	}
}

func TestRegistry_GetLocalProviders(t *testing.T) {
	r := NewRegistry()

	r.Register(newMockProvider("cloud1", false))
	r.Register(newMockProvider("local1", true))
	r.Register(newMockProvider("cloud2", false))
	r.Register(newMockProvider("local2", true))

	local := r.GetLocalProviders()
	if len(local) != 2 {
		t.Fatalf("expected 2 local providers, got %d", len(local))
	}

	for _, p := range local {
		if !p.Info().IsLocal {
			t.Errorf("expected local provider, got %s", p.Info().Name)
		}
	}
}

func TestRegistry_GetCloudProviders(t *testing.T) {
	r := NewRegistry()

	r.Register(newMockProvider("cloud1", false))
	r.Register(newMockProvider("local1", true))
	r.Register(newMockProvider("cloud2", false))

	cloud := r.GetCloudProviders()
	if len(cloud) != 2 {
		t.Fatalf("expected 2 cloud providers, got %d", len(cloud))
	}

	for _, p := range cloud {
		if p.Info().IsLocal {
			t.Errorf("expected cloud provider, got %s", p.Info().Name)
		}
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	// Test concurrent registration and access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			r.Register(newMockProvider("concurrent", false))
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			r.Get("concurrent")
			r.List()
			r.Count()
		}
		done <- true
	}()

	<-done
	<-done
}
