// Package provider provides model routing and provider selection for LLM requests.
package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
)

// mockProvider implements ports.ProviderPort for testing.
type mockProvider struct {
	name           string
	isLocal        bool
	models         []string
	supportedModel map[string]bool
	availableModel map[string]bool
	listModelsErr  error
	supportsErr    error
	availableErr   error
	healthStatus   *ports.HealthStatus
	healthErr      error
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:           name,
		isLocal:        false,
		models:         []string{},
		supportedModel: make(map[string]bool),
		availableModel: make(map[string]bool),
	}
}

func (m *mockProvider) withModels(models ...string) *mockProvider {
	m.models = models
	for _, model := range models {
		m.supportedModel[model] = true
		m.availableModel[model] = true
	}
	return m
}

func (m *mockProvider) withSupportedModel(modelID string, supported bool) *mockProvider {
	m.supportedModel[modelID] = supported
	return m
}

func (m *mockProvider) withAvailableModel(modelID string, available bool) *mockProvider {
	m.availableModel[modelID] = available
	return m
}

// withLocal sets whether the mock provider is local (kept for future use).
var _ = (*mockProvider).withLocal

func (m *mockProvider) withLocal(isLocal bool) *mockProvider {
	m.isLocal = isLocal
	return m
}

func (m *mockProvider) withListModelsError(err error) *mockProvider {
	m.listModelsErr = err
	return m
}

func (m *mockProvider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:    m.name,
		IsLocal: m.isLocal,
	}
}

func (m *mockProvider) ListModels(ctx context.Context) ([]string, error) {
	if m.listModelsErr != nil {
		return nil, m.listModelsErr
	}
	return m.models, nil
}

func (m *mockProvider) SupportsModel(ctx context.Context, modelID string) (bool, error) {
	if m.supportsErr != nil {
		return false, m.supportsErr
	}
	return m.supportedModel[modelID], nil
}

func (m *mockProvider) IsAvailable(ctx context.Context, modelID string) (bool, error) {
	if m.availableErr != nil {
		return false, m.availableErr
	}
	return m.availableModel[modelID], nil
}

func (m *mockProvider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	return &ports.CompletionResponse{
		Content:      "test response",
		InputTokens:  10,
		OutputTokens: 20,
		FinishReason: "stop",
		ModelUsed:    req.ModelID,
		Duration:     100 * time.Millisecond,
	}, nil
}

func (m *mockProvider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	return m.Complete(ctx, req)
}

func (m *mockProvider) HealthCheck(ctx context.Context, modelID string) (*ports.HealthStatus, error) {
	if m.healthErr != nil {
		return nil, m.healthErr
	}
	if m.healthStatus != nil {
		return m.healthStatus, nil
	}
	return &ports.HealthStatus{
		Healthy:     true,
		Message:     "OK",
		Latency:     10 * time.Millisecond,
		LastChecked: time.Now(),
	}, nil
}

// Helper function to create a test routing configuration
func newTestRoutingConfig() *config.RoutingConfiguration {
	cfg := config.NewRoutingConfiguration()
	cfg.Providers = map[string]*config.ProviderConfiguration{
		"ollama": {
			Enabled:  true,
			Priority: 1,
			Models: map[string]*config.ModelConfiguration{
				"llama3.2:3b": {
					Tier:               "cheap",
					CostPerInputToken:  0.0,
					CostPerOutputToken: 0.0,
					MaxTokens:          4096,
					ContextWindow:      8192,
					Enabled:            true,
					Capabilities:       []string{"text"},
				},
				"llama3.2:8b": {
					Tier:               "balanced",
					CostPerInputToken:  0.0,
					CostPerOutputToken: 0.0,
					MaxTokens:          8192,
					ContextWindow:      16384,
					Enabled:            true,
					Capabilities:       []string{"text", "code"},
				},
			},
		},
		"anthropic": {
			Enabled:  true,
			Priority: 2,
			Models: map[string]*config.ModelConfiguration{
				"claude-3-5-sonnet-20241022": {
					Tier:               "premium",
					CostPerInputToken:  0.003,
					CostPerOutputToken: 0.015,
					MaxTokens:          8192,
					ContextWindow:      200000,
					Enabled:            true,
					Capabilities:       []string{"text", "code", "vision"},
				},
			},
		},
		"openai": {
			Enabled:  true,
			Priority: 3,
			Models: map[string]*config.ModelConfiguration{
				"gpt-4o": {
					Tier:               "premium",
					CostPerInputToken:  0.005,
					CostPerOutputToken: 0.015,
					MaxTokens:          4096,
					ContextWindow:      128000,
					Enabled:            true,
					Capabilities:       []string{"text", "code", "vision", "function_calling"},
				},
			},
		},
	}
	cfg.DefaultProvider = "ollama"
	cfg.FallbackChain = []string{"ollama", "anthropic", "openai"}
	return cfg
}

func TestNewRouter(t *testing.T) {
	t.Run("valid config and registry", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v, want nil", err)
		}
		if router == nil {
			t.Fatal("NewRouter() returned nil router")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(nil, registry)
		if err == nil {
			t.Fatal("NewRouter() error = nil, want error")
		}
		if !errors.Is(err, ErrConfigurationNil) {
			t.Errorf("NewRouter() error = %v, want %v", err, ErrConfigurationNil)
		}
		if router != nil {
			t.Error("NewRouter() returned non-nil router with nil config")
		}
	})

	t.Run("nil registry", func(t *testing.T) {
		cfg := newTestRoutingConfig()

		router, err := NewRouter(cfg, nil)
		if err == nil {
			t.Fatal("NewRouter() error = nil, want error")
		}
		if !errors.Is(err, ErrRegistryNil) {
			t.Errorf("NewRouter() error = %v, want %v", err, ErrRegistryNil)
		}
		if router != nil {
			t.Error("NewRouter() returned non-nil router with nil registry")
		}
	})
}

func TestSelectModel(t *testing.T) {
	t.Run("valid profile with available model", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		// Register a mock provider with the model
		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.SelectModel(context.Background(), skill.ProfileBalanced)
		if err != nil {
			t.Fatalf("SelectModel() error = %v", err)
		}

		if selection.ModelID != "llama3.2:8b" {
			t.Errorf("SelectModel() ModelID = %q, want %q", selection.ModelID, "llama3.2:8b")
		}
		if selection.ProviderName != "ollama" {
			t.Errorf("SelectModel() ProviderName = %q, want %q", selection.ProviderName, "ollama")
		}
		if selection.IsFallback {
			t.Error("SelectModel() IsFallback = true, want false")
		}
	})

	t.Run("invalid profile", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		_, err = router.SelectModel(context.Background(), "invalid")
		if err == nil {
			t.Fatal("SelectModel() error = nil, want error")
		}
		if !errors.Is(err, ErrInvalidProfile) {
			t.Errorf("SelectModel() error = %v, want %v", err, ErrInvalidProfile)
		}
	})

	t.Run("profile cheap", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:3b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.SelectModel(context.Background(), skill.ProfileCheap)
		if err != nil {
			t.Fatalf("SelectModel() error = %v", err)
		}

		if selection.ModelID != "llama3.2:3b" {
			t.Errorf("SelectModel() ModelID = %q, want %q", selection.ModelID, "llama3.2:3b")
		}
	})

	t.Run("profile premium", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockAnthropic := newMockProvider("anthropic").withModels("claude-3-5-sonnet-20241022")
		if err := registry.Register(mockAnthropic); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.SelectModel(context.Background(), skill.ProfilePremium)
		if err != nil {
			t.Fatalf("SelectModel() error = %v", err)
		}

		if selection.ModelID != "claude-3-5-sonnet-20241022" {
			t.Errorf("SelectModel() ModelID = %q, want %q", selection.ModelID, "claude-3-5-sonnet-20241022")
		}
	})

	t.Run("primary unavailable uses fallback", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		// Register providers but the primary model is not available
		mockOllama := newMockProvider("ollama").
			withModels("llama3.2:8b", "llama3.2:3b").
			withAvailableModel("llama3.2:8b", false).
			withAvailableModel("llama3.2:3b", true)

		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.SelectModel(context.Background(), skill.ProfileBalanced)
		if err != nil {
			t.Fatalf("SelectModel() error = %v", err)
		}

		// Should fall back to llama3.2:3b
		if selection.ModelID != "llama3.2:3b" {
			t.Errorf("SelectModel() ModelID = %q, want %q", selection.ModelID, "llama3.2:3b")
		}
		if !selection.IsFallback {
			t.Error("SelectModel() IsFallback = false, want true")
		}
	})
}

func TestSelectModelForPhase(t *testing.T) {
	t.Run("nil phase", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		_, err = router.SelectModelForPhase(context.Background(), nil)
		if err == nil {
			t.Fatal("SelectModelForPhase() error = nil, want error")
		}
	})

	t.Run("generation phase uses generation model", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		phase := &skill.Phase{
			ID:             "generate",
			Name:           "Generate Code",
			RoutingProfile: skill.ProfileBalanced,
		}

		selection, err := router.SelectModelForPhase(context.Background(), phase)
		if err != nil {
			t.Fatalf("SelectModelForPhase() error = %v", err)
		}

		if selection.ModelID != "llama3.2:8b" {
			t.Errorf("SelectModelForPhase() ModelID = %q, want %q", selection.ModelID, "llama3.2:8b")
		}
	})

	t.Run("review phase uses review model", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		// Set a different review model
		cfg.Profiles[skill.ProfileBalanced].ReviewModel = "gpt-4o"

		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b")
		mockOpenAI := newMockProvider("openai").withModels("gpt-4o")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register ollama: %v", err)
		}
		if err := registry.Register(mockOpenAI); err != nil {
			t.Fatalf("failed to register openai: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		phase := &skill.Phase{
			ID:             "review",
			Name:           "Code Review",
			RoutingProfile: skill.ProfileBalanced,
		}

		selection, err := router.SelectModelForPhase(context.Background(), phase)
		if err != nil {
			t.Fatalf("SelectModelForPhase() error = %v", err)
		}

		if selection.ModelID != "gpt-4o" {
			t.Errorf("SelectModelForPhase() ModelID = %q, want %q", selection.ModelID, "gpt-4o")
		}
	})

	t.Run("validate phase uses review model", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		cfg.Profiles[skill.ProfileBalanced].ReviewModel = "gpt-4o"

		registry := adapterProvider.NewRegistry()

		mockOpenAI := newMockProvider("openai").withModels("gpt-4o")
		if err := registry.Register(mockOpenAI); err != nil {
			t.Fatalf("failed to register openai: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		phase := &skill.Phase{
			ID:             "validate_output",
			Name:           "Validate Output",
			RoutingProfile: skill.ProfileBalanced,
		}

		selection, err := router.SelectModelForPhase(context.Background(), phase)
		if err != nil {
			t.Fatalf("SelectModelForPhase() error = %v", err)
		}

		if selection.ModelID != "gpt-4o" {
			t.Errorf("SelectModelForPhase() ModelID = %q, want %q", selection.ModelID, "gpt-4o")
		}
	})

	t.Run("check phase uses review model", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		cfg.Profiles[skill.ProfileBalanced].ReviewModel = "gpt-4o"

		registry := adapterProvider.NewRegistry()

		mockOpenAI := newMockProvider("openai").withModels("gpt-4o")
		if err := registry.Register(mockOpenAI); err != nil {
			t.Fatalf("failed to register openai: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		phase := &skill.Phase{
			ID:             "syntax_check",
			Name:           "Syntax Check",
			RoutingProfile: skill.ProfileBalanced,
		}

		selection, err := router.SelectModelForPhase(context.Background(), phase)
		if err != nil {
			t.Fatalf("SelectModelForPhase() error = %v", err)
		}

		if selection.ModelID != "gpt-4o" {
			t.Errorf("SelectModelForPhase() ModelID = %q, want %q", selection.ModelID, "gpt-4o")
		}
	})

	t.Run("invalid routing profile defaults to balanced", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		phase := &skill.Phase{
			ID:             "generate",
			Name:           "Generate",
			RoutingProfile: "invalid_profile", // Invalid profile
		}

		selection, err := router.SelectModelForPhase(context.Background(), phase)
		if err != nil {
			t.Fatalf("SelectModelForPhase() error = %v", err)
		}

		// Should fall back to balanced profile's generation model
		if selection.ModelID != "llama3.2:8b" {
			t.Errorf("SelectModelForPhase() ModelID = %q, want %q", selection.ModelID, "llama3.2:8b")
		}
	})
}

func TestGetFallbackModel(t *testing.T) {
	t.Run("uses profile fallback model", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:3b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.GetFallbackModel(context.Background(), skill.ProfileBalanced)
		if err != nil {
			t.Fatalf("GetFallbackModel() error = %v", err)
		}

		if selection.ModelID != "llama3.2:3b" {
			t.Errorf("GetFallbackModel() ModelID = %q, want %q", selection.ModelID, "llama3.2:3b")
		}
		if !selection.IsFallback {
			t.Error("GetFallbackModel() IsFallback = false, want true")
		}
	})

	t.Run("uses fallback chain when profile fallback unavailable", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		// Profile fallback is not available, but a model from fallback chain is
		mockOllama := newMockProvider("ollama").
			withModels("llama3.2:8b").
			withSupportedModel("llama3.2:3b", false) // Fallback model not supported

		mockAnthropic := newMockProvider("anthropic").withModels("claude-3-5-sonnet-20241022")

		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register ollama: %v", err)
		}
		if err := registry.Register(mockAnthropic); err != nil {
			t.Fatalf("failed to register anthropic: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.GetFallbackModel(context.Background(), skill.ProfileBalanced)
		if err != nil {
			t.Fatalf("GetFallbackModel() error = %v", err)
		}

		// Should use a model from the fallback chain (ollama has a model available)
		if selection.ProviderName != "ollama" {
			t.Errorf("GetFallbackModel() ProviderName = %q, want %q", selection.ProviderName, "ollama")
		}
		if !selection.IsFallback {
			t.Error("GetFallbackModel() IsFallback = false, want true")
		}
	})

	t.Run("invalid profile", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		_, err = router.GetFallbackModel(context.Background(), "invalid")
		if err == nil {
			t.Fatal("GetFallbackModel() error = nil, want error")
		}
		if !errors.Is(err, ErrInvalidProfile) {
			t.Errorf("GetFallbackModel() error = %v, want %v", err, ErrInvalidProfile)
		}
	})

	t.Run("no fallback available", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		cfg.FallbackChain = []string{} // Empty fallback chain
		cfg.Profiles[skill.ProfileBalanced].FallbackModel = ""
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		_, err = router.GetFallbackModel(context.Background(), skill.ProfileBalanced)
		if err == nil {
			t.Fatal("GetFallbackModel() error = nil, want error")
		}
		if !errors.Is(err, ErrNoFallbackModel) {
			t.Errorf("GetFallbackModel() error = %v, want %v", err, ErrNoFallbackModel)
		}
	})

	t.Run("fallback chain provider returns error on ListModels", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		cfg.Profiles[skill.ProfileBalanced].FallbackModel = ""
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").
			withListModelsError(errors.New("connection refused"))

		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		_, err = router.GetFallbackModel(context.Background(), skill.ProfileBalanced)
		if err == nil {
			t.Fatal("GetFallbackModel() error = nil, want error")
		}
	})
}

func TestSelectModelWithCapabilities(t *testing.T) {
	t.Run("selects model with required capabilities", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOpenAI := newMockProvider("openai").withModels("gpt-4o")
		if err := registry.Register(mockOpenAI); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.SelectModelWithCapabilities(
			context.Background(),
			skill.ProfilePremium,
			[]string{"vision", "function_calling"},
		)
		if err != nil {
			t.Fatalf("SelectModelWithCapabilities() error = %v", err)
		}

		if selection.ModelID != "gpt-4o" {
			t.Errorf("SelectModelWithCapabilities() ModelID = %q, want %q", selection.ModelID, "gpt-4o")
		}
	})

	t.Run("falls back when no model has capabilities", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		// Register a provider that will be used for the fallback
		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		// Request capabilities that no model has
		selection, err := router.SelectModelWithCapabilities(
			context.Background(),
			skill.ProfileBalanced,
			[]string{"nonexistent_capability"},
		)
		if err != nil {
			t.Fatalf("SelectModelWithCapabilities() error = %v", err)
		}

		// Should fall back to regular selection
		if selection.ModelID != "llama3.2:8b" {
			t.Errorf("SelectModelWithCapabilities() ModelID = %q, want %q", selection.ModelID, "llama3.2:8b")
		}
	})

	t.Run("invalid profile", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		_, err = router.SelectModelWithCapabilities(
			context.Background(),
			"invalid",
			[]string{"text"},
		)
		if err == nil {
			t.Fatal("SelectModelWithCapabilities() error = nil, want error")
		}
		if !errors.Is(err, ErrInvalidProfile) {
			t.Errorf("SelectModelWithCapabilities() error = %v, want %v", err, ErrInvalidProfile)
		}
	})

	t.Run("empty capabilities list", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.SelectModelWithCapabilities(
			context.Background(),
			skill.ProfileBalanced,
			[]string{},
		)
		if err != nil {
			t.Fatalf("SelectModelWithCapabilities() error = %v", err)
		}

		// With empty capabilities, should match any model
		if selection.ModelID == "" {
			t.Error("SelectModelWithCapabilities() ModelID = empty, want non-empty")
		}
	})

	t.Run("vision capability", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockAnthropic := newMockProvider("anthropic").withModels("claude-3-5-sonnet-20241022")
		if err := registry.Register(mockAnthropic); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		selection, err := router.SelectModelWithCapabilities(
			context.Background(),
			skill.ProfilePremium,
			[]string{"vision"},
		)
		if err != nil {
			t.Fatalf("SelectModelWithCapabilities() error = %v", err)
		}

		if selection.ModelID != "claude-3-5-sonnet-20241022" {
			t.Errorf("SelectModelWithCapabilities() ModelID = %q, want %q", selection.ModelID, "claude-3-5-sonnet-20241022")
		}
	})
}

func TestIsModelAvailable(t *testing.T) {
	t.Run("model is available", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		available := router.IsModelAvailable(context.Background(), "llama3.2:8b")
		if !available {
			t.Error("IsModelAvailable() = false, want true")
		}
	})

	t.Run("model not available", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").
			withModels("llama3.2:8b").
			withAvailableModel("llama3.2:8b", false)

		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		available := router.IsModelAvailable(context.Background(), "llama3.2:8b")
		if available {
			t.Error("IsModelAvailable() = true, want false")
		}
	})

	t.Run("model not found", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		available := router.IsModelAvailable(context.Background(), "nonexistent-model")
		if available {
			t.Error("IsModelAvailable() = true, want false")
		}
	})
}

func TestGetModelConfig(t *testing.T) {
	t.Run("existing model", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		modelCfg := router.GetModelConfig("ollama", "llama3.2:8b")
		if modelCfg == nil {
			t.Fatal("GetModelConfig() = nil, want non-nil")
		}
		if modelCfg.Tier != "balanced" {
			t.Errorf("GetModelConfig() Tier = %q, want %q", modelCfg.Tier, "balanced")
		}
	})

	t.Run("non-existing provider", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		modelCfg := router.GetModelConfig("nonexistent", "model")
		if modelCfg != nil {
			t.Error("GetModelConfig() = non-nil, want nil")
		}
	})

	t.Run("non-existing model", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		modelCfg := router.GetModelConfig("ollama", "nonexistent-model")
		if modelCfg != nil {
			t.Error("GetModelConfig() = non-nil, want nil")
		}
	})
}

func TestGetProfileConfig(t *testing.T) {
	t.Run("existing profile", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		profileCfg := router.GetProfileConfig(skill.ProfileBalanced)
		if profileCfg == nil {
			t.Fatal("GetProfileConfig() = nil, want non-nil")
		}
		if profileCfg.GenerationModel != "llama3.2:8b" {
			t.Errorf("GetProfileConfig() GenerationModel = %q, want %q", profileCfg.GenerationModel, "llama3.2:8b")
		}
	})

	t.Run("non-existing profile", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		profileCfg := router.GetProfileConfig("nonexistent")
		if profileCfg != nil {
			t.Error("GetProfileConfig() = non-nil, want nil")
		}
	})
}

func TestUpdateConfig(t *testing.T) {
	t.Run("update with valid config", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		// Create a new config with different defaults
		newCfg := newTestRoutingConfig()
		newCfg.DefaultProvider = "anthropic"

		err = router.UpdateConfig(newCfg)
		if err != nil {
			t.Fatalf("UpdateConfig() error = %v", err)
		}

		if router.GetDefaultProvider() != "anthropic" {
			t.Errorf("GetDefaultProvider() = %q, want %q", router.GetDefaultProvider(), "anthropic")
		}
	})

	t.Run("update with nil config", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		err = router.UpdateConfig(nil)
		if err == nil {
			t.Fatal("UpdateConfig() error = nil, want error")
		}
		if !errors.Is(err, ErrConfigurationNil) {
			t.Errorf("UpdateConfig() error = %v, want %v", err, ErrConfigurationNil)
		}
	})
}

func TestGetEnabledProviders(t *testing.T) {
	t.Run("returns enabled providers in priority order", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		providers := router.GetEnabledProviders()
		if len(providers) != 3 {
			t.Errorf("GetEnabledProviders() len = %d, want 3", len(providers))
		}

		// Verify order by priority
		if providers[0] != "ollama" {
			t.Errorf("GetEnabledProviders()[0] = %q, want %q", providers[0], "ollama")
		}
		if providers[1] != "anthropic" {
			t.Errorf("GetEnabledProviders()[1] = %q, want %q", providers[1], "anthropic")
		}
		if providers[2] != "openai" {
			t.Errorf("GetEnabledProviders()[2] = %q, want %q", providers[2], "openai")
		}
	})
}

func TestGetDefaultProvider(t *testing.T) {
	t.Run("returns default provider", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		defaultProvider := router.GetDefaultProvider()
		if defaultProvider != "ollama" {
			t.Errorf("GetDefaultProvider() = %q, want %q", defaultProvider, "ollama")
		}
	})
}

func TestIsValidProfile(t *testing.T) {
	tests := []struct {
		profile string
		want    bool
	}{
		{skill.ProfileCheap, true},
		{skill.ProfileBalanced, true},
		{skill.ProfilePremium, true},
		{"invalid", false},
		{"", false},
		{"CHEAP", false},    // Case sensitive
		{"Balanced", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			got := isValidProfile(tt.profile)
			if got != tt.want {
				t.Errorf("isValidProfile(%q) = %v, want %v", tt.profile, got, tt.want)
			}
		})
	}
}

func TestIsReviewPhase(t *testing.T) {
	tests := []struct {
		name  string
		phase *skill.Phase
		want  bool
	}{
		{
			name:  "nil phase",
			phase: nil,
			want:  false,
		},
		{
			name: "review in id",
			phase: &skill.Phase{
				ID:   "code_review",
				Name: "Code Analysis",
			},
			want: true,
		},
		{
			name: "review in name",
			phase: &skill.Phase{
				ID:   "analysis",
				Name: "Code Review",
			},
			want: true,
		},
		{
			name: "validate in id",
			phase: &skill.Phase{
				ID:   "validate_output",
				Name: "Output Processing",
			},
			want: true,
		},
		{
			name: "check in id",
			phase: &skill.Phase{
				ID:   "syntax_check",
				Name: "Syntax Processing",
			},
			want: true,
		},
		{
			name: "verify in name",
			phase: &skill.Phase{
				ID:   "final_step",
				Name: "Verify Results",
			},
			want: true,
		},
		{
			name: "audit in name",
			phase: &skill.Phase{
				ID:   "security",
				Name: "Security Audit",
			},
			want: true,
		},
		{
			name: "generation phase",
			phase: &skill.Phase{
				ID:   "generate",
				Name: "Generate Code",
			},
			want: false,
		},
		{
			name: "transform phase",
			phase: &skill.Phase{
				ID:   "transform",
				Name: "Transform Data",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isReviewPhase(tt.phase)
			if got != tt.want {
				t.Errorf("isReviewPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "World", true},
		{"Hello World", "xyz", false},
		{"review", "REVIEW", true},
		{"CODE_REVIEW", "review", true},
		{"", "test", false},
		{"test", "", true},
		{"ab", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HELLO", "hello"},
		{"Hello World", "hello world"},
		{"already lower", "already lower"},
		{"MiXeD CaSe", "mixed case"},
		{"123ABC", "123abc"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toLower(tt.input)
			if got != tt.want {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasAllCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		model    *config.ModelConfiguration
		required []string
		want     bool
	}{
		{
			name:     "nil model empty required",
			model:    nil,
			required: []string{},
			want:     true,
		},
		{
			name:     "nil model with required",
			model:    nil,
			required: []string{"text"},
			want:     false,
		},
		{
			name: "model with all capabilities",
			model: &config.ModelConfiguration{
				Capabilities: []string{"text", "code", "vision"},
			},
			required: []string{"text", "code"},
			want:     true,
		},
		{
			name: "model missing capability",
			model: &config.ModelConfiguration{
				Capabilities: []string{"text", "code"},
			},
			required: []string{"text", "vision"},
			want:     false,
		},
		{
			name: "empty capabilities list",
			model: &config.ModelConfiguration{
				Capabilities: []string{},
			},
			required: []string{"text"},
			want:     false,
		},
		{
			name: "empty required list",
			model: &config.ModelConfiguration{
				Capabilities: []string{"text", "code"},
			},
			required: []string{},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAllCapabilities(tt.model, tt.required)
			if got != tt.want {
				t.Errorf("hasAllCapabilities() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouterConcurrency(t *testing.T) {
	t.Run("concurrent SelectModel calls", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b", "llama3.2:3b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		// Run concurrent selections
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, err := router.SelectModel(context.Background(), skill.ProfileBalanced)
				if err != nil {
					t.Errorf("concurrent SelectModel() error = %v", err)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent UpdateConfig and SelectModel", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		registry := adapterProvider.NewRegistry()

		mockOllama := newMockProvider("ollama").withModels("llama3.2:8b")
		if err := registry.Register(mockOllama); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		done := make(chan bool, 20)

		// Run concurrent selections
		for i := 0; i < 10; i++ {
			go func() {
				_, _ = router.SelectModel(context.Background(), skill.ProfileBalanced)
				done <- true
			}()
		}

		// Run concurrent updates
		for i := 0; i < 10; i++ {
			go func() {
				newCfg := newTestRoutingConfig()
				_ = router.UpdateConfig(newCfg)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 20; i++ {
			<-done
		}
	})
}

func TestNoProfileConfig(t *testing.T) {
	t.Run("SelectModel with no profile config", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		// Remove the balanced profile
		delete(cfg.Profiles, skill.ProfileBalanced)
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		_, err = router.SelectModel(context.Background(), skill.ProfileBalanced)
		if err == nil {
			t.Fatal("SelectModel() error = nil, want error")
		}
		if !errors.Is(err, ErrNoProfileConfig) {
			t.Errorf("SelectModel() error = %v, want %v", err, ErrNoProfileConfig)
		}
	})

	t.Run("SelectModelForPhase with no profile config", func(t *testing.T) {
		cfg := newTestRoutingConfig()
		delete(cfg.Profiles, skill.ProfileBalanced)
		registry := adapterProvider.NewRegistry()

		router, err := NewRouter(cfg, registry)
		if err != nil {
			t.Fatalf("NewRouter() error = %v", err)
		}

		phase := &skill.Phase{
			ID:             "generate",
			Name:           "Generate",
			RoutingProfile: skill.ProfileBalanced,
		}

		_, err = router.SelectModelForPhase(context.Background(), phase)
		if err == nil {
			t.Fatal("SelectModelForPhase() error = nil, want error")
		}
		if !errors.Is(err, ErrNoProfileConfig) {
			t.Errorf("SelectModelForPhase() error = %v, want %v", err, ErrNoProfileConfig)
		}
	})
}

func TestModelSelection(t *testing.T) {
	t.Run("ModelSelection struct fields", func(t *testing.T) {
		selection := &ModelSelection{
			ModelID:      "test-model",
			ProviderName: "test-provider",
			IsFallback:   true,
		}

		if selection.ModelID != "test-model" {
			t.Errorf("ModelSelection.ModelID = %q, want %q", selection.ModelID, "test-model")
		}
		if selection.ProviderName != "test-provider" {
			t.Errorf("ModelSelection.ProviderName = %q, want %q", selection.ProviderName, "test-provider")
		}
		if !selection.IsFallback {
			t.Error("ModelSelection.IsFallback = false, want true")
		}
	})
}
