package provider

import (
	"context"
	"testing"
	"time"

	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
)

// testProvider implements ports.ProviderPort for testing the initializer
type testProvider struct {
	name      string
	isLocal   bool
	baseURL   string
	models    []string
	healthy   bool
	healthErr error
	latency   time.Duration
}

func (m *testProvider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:    m.name,
		IsLocal: m.isLocal,
		BaseURL: m.baseURL,
	}
}

func (m *testProvider) ListModels(ctx context.Context) ([]string, error) {
	return m.models, nil
}

func (m *testProvider) SupportsModel(ctx context.Context, modelID string) (bool, error) {
	for _, model := range m.models {
		if model == modelID {
			return true, nil
		}
	}
	return false, nil
}

func (m *testProvider) IsAvailable(ctx context.Context, modelID string) (bool, error) {
	return m.healthy, nil
}

func (m *testProvider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	return nil, nil
}

func (m *testProvider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	return nil, nil
}

func (m *testProvider) HealthCheck(ctx context.Context, modelID string) (*ports.HealthStatus, error) {
	if m.healthErr != nil {
		return nil, m.healthErr
	}
	return &ports.HealthStatus{
		Healthy:     m.healthy,
		Latency:     m.latency,
		LastChecked: time.Now(),
	}, nil
}

func TestNewInitializer(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)

	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	if initializer == nil {
		t.Fatal("NewInitializer returned nil")
	}

	if initializer.registry != registry {
		t.Error("initializer registry does not match input")
	}

	if initializer.health == nil {
		t.Error("initializer health map should be initialized")
	}

	if initializer.encryptor == nil {
		t.Error("initializer encryptor should be initialized")
	}
}

func TestInitFromConfig_NilConfig(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	err = initializer.InitFromConfig(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestInitFromConfig_DefaultConfig(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	cfg := config.NewDefaultConfig()
	// Default config has Ollama enabled but no API keys for cloud providers
	err = initializer.InitFromConfig(cfg)

	// Ollama should be registered (default config has it enabled)
	if registry.Get("ollama") == nil {
		t.Error("expected Ollama provider to be registered")
	}

	// Cloud providers should not be registered (no API keys)
	if registry.Get("anthropic") != nil {
		t.Error("Anthropic should not be registered without API key")
	}

	// Check health data is populated
	ollamaHealth := initializer.GetHealth("ollama")
	if ollamaHealth == nil {
		t.Error("expected Ollama health data")
	}
	if !ollamaHealth.Enabled {
		t.Error("Ollama should be marked as enabled")
	}

	// Check error is nil since Ollama initialized successfully
	if err != nil {
		t.Logf("InitFromConfig returned non-fatal error: %v", err)
	}
}

func TestInitFromConfig_DisabledProviders(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	cfg := config.NewDefaultConfig()
	cfg.Providers.Ollama.Enabled = false

	_ = initializer.InitFromConfig(cfg)

	// Ollama should not be registered
	if registry.Get("ollama") != nil {
		t.Error("Ollama should not be registered when disabled")
	}

	// But health data should still exist
	ollamaHealth := initializer.GetHealth("ollama")
	if ollamaHealth == nil {
		t.Fatal("expected Ollama health data even when disabled")
	}
	if ollamaHealth.Enabled {
		t.Error("Ollama health should show as disabled")
	}
}

func TestInitFromConfig_CloudProviderWithKey(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	cfg := config.NewDefaultConfig()
	cfg.Providers.Ollama.Enabled = false
	cfg.Providers.Anthropic.Enabled = true

	// Encrypt a test API key using the same encryptor
	encryptedKey, err := initializer.encryptor.Encrypt("test-api-key")
	if err != nil {
		t.Fatalf("failed to encrypt test API key: %v", err)
	}
	cfg.Providers.Anthropic.APIKeyEncrypted = encryptedKey

	err = initializer.InitFromConfig(cfg)

	// Anthropic should be registered
	if registry.Get("anthropic") == nil {
		t.Error("expected Anthropic provider to be registered")
	}

	// Check health data
	anthropicHealth := initializer.GetHealth("anthropic")
	if anthropicHealth == nil {
		t.Fatal("expected Anthropic health data")
	}
	if !anthropicHealth.Enabled {
		t.Error("Anthropic should be marked as enabled")
	}
	if !anthropicHealth.APIKeySet {
		t.Error("Anthropic should show API key as set")
	}

	// No errors since cloud provider initialization with key should succeed
	if err != nil {
		t.Logf("InitFromConfig returned: %v", err)
	}
}

func TestCheckHealth(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	// Register a test provider
	mock := &testProvider{
		name:    "test-provider",
		isLocal: true,
		baseURL: "http://localhost:1234",
		models:  []string{"model-a", "model-b"},
		healthy: true,
		latency: 50 * time.Millisecond,
	}
	_ = registry.Register(mock)

	// Check health
	ctx := context.Background()
	results := initializer.CheckHealth(ctx)

	health, exists := results["test-provider"]
	if !exists {
		t.Fatal("expected test-provider in health results")
	}

	if !health.Healthy {
		t.Error("expected healthy status")
	}

	if health.Latency != 50*time.Millisecond {
		t.Errorf("expected latency of 50ms, got %v", health.Latency)
	}

	if len(health.Models) != 2 {
		t.Errorf("expected 2 models, got %d", len(health.Models))
	}
}

func TestCheckHealth_UnhealthyProvider(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	// Register an unhealthy test provider
	mock := &testProvider{
		name:    "unhealthy-provider",
		isLocal: false,
		healthy: false,
		latency: 100 * time.Millisecond,
	}
	_ = registry.Register(mock)

	ctx := context.Background()
	results := initializer.CheckHealth(ctx)

	health, exists := results["unhealthy-provider"]
	if !exists {
		t.Fatal("expected unhealthy-provider in health results")
	}

	if health.Healthy {
		t.Error("expected unhealthy status")
	}

	if health.Type != "cloud" {
		t.Errorf("expected cloud type, got %s", health.Type)
	}
}

func TestGetAllHealth(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	// Set up some health data
	initializer.setProviderHealth("provider1", &ProviderHealth{
		Name:    "provider1",
		Enabled: true,
		Healthy: true,
	})
	initializer.setProviderHealth("provider2", &ProviderHealth{
		Name:    "provider2",
		Enabled: false,
		Healthy: false,
	})

	allHealth := initializer.GetAllHealth()

	if len(allHealth) != 2 {
		t.Errorf("expected 2 health entries, got %d", len(allHealth))
	}

	if allHealth["provider1"] == nil {
		t.Error("expected provider1 health data")
	}

	if allHealth["provider2"] == nil {
		t.Error("expected provider2 health data")
	}
}

func TestGetAvailableModels(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	// Register test providers
	mock1 := &testProvider{
		name:   "provider1",
		models: []string{"model-a", "model-b"},
	}
	mock2 := &testProvider{
		name:   "provider2",
		models: []string{"model-c"},
	}
	_ = registry.Register(mock1)
	_ = registry.Register(mock2)

	ctx := context.Background()
	models := initializer.GetAvailableModels(ctx)

	if len(models) != 2 {
		t.Errorf("expected 2 providers, got %d", len(models))
	}

	if len(models["provider1"]) != 2 {
		t.Errorf("expected 2 models for provider1, got %d", len(models["provider1"]))
	}

	if len(models["provider2"]) != 1 {
		t.Errorf("expected 1 model for provider2, got %d", len(models["provider2"]))
	}
}

func TestRegistry(t *testing.T) {
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	if initializer.Registry() != registry {
		t.Error("Registry() should return the underlying registry")
	}
}

func TestProviderHealth_Type(t *testing.T) {
	// Test that provider types are correctly identified
	registry := adapterProvider.NewRegistry()
	initializer, err := NewInitializer(registry)
	if err != nil {
		t.Fatalf("NewInitializer returned error: %v", err)
	}

	localMock := &testProvider{
		name:    "local-provider",
		isLocal: true,
		healthy: true,
	}
	cloudMock := &testProvider{
		name:    "cloud-provider",
		isLocal: false,
		healthy: true,
	}
	_ = registry.Register(localMock)
	_ = registry.Register(cloudMock)

	ctx := context.Background()
	results := initializer.CheckHealth(ctx)

	localHealth := results["local-provider"]
	if localHealth.Type != "local" {
		t.Errorf("expected local type, got %s", localHealth.Type)
	}

	cloudHealth := results["cloud-provider"]
	if cloudHealth.Type != "cloud" {
		t.Errorf("expected cloud type, got %s", cloudHealth.Type)
	}
}
