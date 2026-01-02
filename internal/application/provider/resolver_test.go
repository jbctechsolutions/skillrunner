package provider

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
)

// resolverMockProvider implements ports.ProviderPort for resolver testing.
type resolverMockProvider struct {
	name            string
	isLocal         bool
	models          []string
	supportedModels map[string]bool
	availableModels map[string]bool
	healthStatus    *ports.HealthStatus
	completionResp  *ports.CompletionResponse
	completionErr   error
}

func newResolverMockProvider(name string) *resolverMockProvider {
	return &resolverMockProvider{
		name:            name,
		models:          []string{},
		supportedModels: make(map[string]bool),
		availableModels: make(map[string]bool),
		healthStatus: &ports.HealthStatus{
			Healthy: true,
			Message: "OK",
			Latency: 10 * time.Millisecond,
		},
	}
}

func (m *resolverMockProvider) withModel(modelID string, supported, available bool) *resolverMockProvider {
	m.models = append(m.models, modelID)
	m.supportedModels[modelID] = supported
	m.availableModels[modelID] = available
	return m
}

func (m *resolverMockProvider) withLocal(isLocal bool) *resolverMockProvider {
	m.isLocal = isLocal
	return m
}

func (m *resolverMockProvider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:    m.name,
		IsLocal: m.isLocal,
	}
}

func (m *resolverMockProvider) ListModels(ctx context.Context) ([]string, error) {
	return m.models, nil
}

func (m *resolverMockProvider) SupportsModel(ctx context.Context, modelID string) (bool, error) {
	return m.supportedModels[modelID], nil
}

func (m *resolverMockProvider) IsAvailable(ctx context.Context, modelID string) (bool, error) {
	return m.availableModels[modelID], nil
}

func (m *resolverMockProvider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	if m.completionErr != nil {
		return nil, m.completionErr
	}
	if m.completionResp != nil {
		return m.completionResp, nil
	}
	return &ports.CompletionResponse{
		Content:      "response",
		InputTokens:  100,
		OutputTokens: 50,
	}, nil
}

func (m *resolverMockProvider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	return m.Complete(ctx, req)
}

func (m *resolverMockProvider) HealthCheck(ctx context.Context, modelID string) (*ports.HealthStatus, error) {
	return m.healthStatus, nil
}

// Helper to create a test routing configuration for resolver tests
func createResolverTestRoutingConfig() *config.RoutingConfiguration {
	cfg := config.NewRoutingConfiguration()
	cfg.Providers = map[string]*config.ProviderConfiguration{
		"ollama": {
			Enabled:  true,
			Priority: 1,
			Models: map[string]*config.ModelConfiguration{
				"llama3.2:3b": {
					Enabled:            true,
					Tier:               "cheap",
					CostPerInputToken:  0.0,
					CostPerOutputToken: 0.0,
					MaxTokens:          4096,
					ContextWindow:      8192,
				},
				"llama3.2:8b": {
					Enabled:            true,
					Tier:               "balanced",
					CostPerInputToken:  0.0,
					CostPerOutputToken: 0.0,
					MaxTokens:          4096,
					ContextWindow:      8192,
				},
			},
		},
		"anthropic": {
			Enabled:  true,
			Priority: 2,
			Models: map[string]*config.ModelConfiguration{
				"claude-3-5-sonnet-20241022": {
					Enabled:            true,
					Tier:               "premium",
					CostPerInputToken:  0.003,
					CostPerOutputToken: 0.015,
					MaxTokens:          8192,
					ContextWindow:      200000,
					Capabilities:       []string{"vision", "function_calling"},
				},
			},
		},
		"openai": {
			Enabled:  true,
			Priority: 3,
			Models: map[string]*config.ModelConfiguration{
				"gpt-4o": {
					Enabled:            true,
					Tier:               "premium",
					CostPerInputToken:  0.005,
					CostPerOutputToken: 0.015,
					MaxTokens:          4096,
					ContextWindow:      128000,
					Capabilities:       []string{"vision", "function_calling"},
				},
			},
		},
	}
	return cfg
}

// Helper to create a test registry with mock providers for resolver tests
func createResolverTestRegistry() *adapterProvider.Registry {
	registry := adapterProvider.NewRegistry()

	ollamaProvider := newResolverMockProvider("ollama").
		withLocal(true).
		withModel("llama3.2:3b", true, true).
		withModel("llama3.2:8b", true, true).
		withModel("llama3.2:1b", true, true)
	registry.Register(ollamaProvider)

	anthropicProvider := newResolverMockProvider("anthropic").
		withModel("claude-3-5-sonnet-20241022", true, true)
	registry.Register(anthropicProvider)

	openaiProvider := newResolverMockProvider("openai").
		withModel("gpt-4o", true, true)
	registry.Register(openaiProvider)

	return registry
}

// TestNewResolver tests the NewResolver constructor.
func TestNewResolver(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, err := NewRouter(cfg, registry)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	tests := []struct {
		name        string
		router      *Router
		registry    *adapterProvider.Registry
		config      *config.RoutingConfiguration
		expectError error
	}{
		{
			name:        "valid inputs",
			router:      router,
			registry:    registry,
			config:      cfg,
			expectError: nil,
		},
		{
			name:        "nil router",
			router:      nil,
			registry:    registry,
			config:      cfg,
			expectError: ErrResolverRouterNil,
		},
		{
			name:        "nil registry",
			router:      router,
			registry:    nil,
			config:      cfg,
			expectError: ErrResolverRegistryNil,
		},
		{
			name:        "nil config",
			router:      router,
			registry:    registry,
			config:      nil,
			expectError: ErrResolverConfigNil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewResolver(tt.router, tt.registry, tt.config)
			if tt.expectError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectError)
				} else if !errors.Is(err, tt.expectError) {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resolver == nil {
					t.Error("expected non-nil resolver")
				}
			}
		})
	}
}

// TestResolverResolve tests the Resolver.Resolve method.
func TestResolverResolve(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	ctx := context.Background()

	tests := []struct {
		name           string
		profile        string
		expectError    bool
		expectFallback bool
		expectModelID  string
	}{
		{
			name:           "cheap profile",
			profile:        skill.ProfileCheap,
			expectError:    false,
			expectFallback: false,
			expectModelID:  "llama3.2:3b",
		},
		{
			name:           "balanced profile",
			profile:        skill.ProfileBalanced,
			expectError:    false,
			expectFallback: false,
			expectModelID:  "llama3.2:8b",
		},
		{
			name:           "premium profile",
			profile:        skill.ProfilePremium,
			expectError:    false,
			expectFallback: false,
			expectModelID:  "claude-3-5-sonnet-20241022",
		},
		{
			name:        "invalid profile",
			profile:     "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := resolver.Resolve(ctx, tt.profile)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resolution == nil {
					t.Fatal("expected non-nil resolution")
				}
				if resolution.ModelID != tt.expectModelID {
					t.Errorf("expected model %s, got %s", tt.expectModelID, resolution.ModelID)
				}
				if resolution.IsFallback != tt.expectFallback {
					t.Errorf("expected fallback=%v, got %v", tt.expectFallback, resolution.IsFallback)
				}
			}
		})
	}
}

// TestResolverResolveForPhase tests the Resolver.ResolveForPhase method.
func TestResolverResolveForPhase(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	ctx := context.Background()

	tests := []struct {
		name          string
		phase         *skill.Phase
		expectError   bool
		expectModelID string
	}{
		{
			name:        "nil phase",
			phase:       nil,
			expectError: true,
		},
		{
			name: "generation phase with balanced profile",
			phase: func() *skill.Phase {
				p, _ := skill.NewPhase("gen-1", "Generation", "Generate code")
				p.RoutingProfile = skill.ProfileBalanced
				return p
			}(),
			expectError:   false,
			expectModelID: "llama3.2:8b",
		},
		{
			name: "generation phase with cheap profile",
			phase: func() *skill.Phase {
				p, _ := skill.NewPhase("gen-2", "Generation", "Generate code")
				p.RoutingProfile = skill.ProfileCheap
				return p
			}(),
			expectError:   false,
			expectModelID: "llama3.2:3b",
		},
		{
			name: "review phase with premium profile",
			phase: func() *skill.Phase {
				p, _ := skill.NewPhase("review-1", "Code Review", "Review the code")
				p.RoutingProfile = skill.ProfilePremium
				return p
			}(),
			expectError:   false,
			expectModelID: "gpt-4o", // review model for premium
		},
		{
			name: "validate phase (review indicator) with balanced profile",
			phase: func() *skill.Phase {
				p, _ := skill.NewPhase("validate-1", "Validate Output", "Validate")
				p.RoutingProfile = skill.ProfileBalanced
				return p
			}(),
			expectError:   false,
			expectModelID: "llama3.2:8b", // review model same as gen for balanced
		},
		{
			name: "phase with empty routing profile defaults to balanced",
			phase: func() *skill.Phase {
				p, _ := skill.NewPhase("default-1", "Default Phase", "Default prompt")
				p.RoutingProfile = ""
				return p
			}(),
			expectError:   false,
			expectModelID: "llama3.2:8b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := resolver.ResolveForPhase(ctx, tt.phase)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resolution == nil {
					t.Fatal("expected non-nil resolution")
				}
				if resolution.ModelID != tt.expectModelID {
					t.Errorf("expected model %s, got %s", tt.expectModelID, resolution.ModelID)
				}
			}
		})
	}
}

// TestResolverResolveWithCapabilities tests the Resolver.ResolveWithCapabilities method.
func TestResolverResolveWithCapabilities(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	ctx := context.Background()

	tests := []struct {
		name           string
		profile        string
		capabilities   []string
		expectError    bool
		expectFallback bool
	}{
		{
			name:           "premium with vision capability",
			profile:        skill.ProfilePremium,
			capabilities:   []string{"vision"},
			expectError:    false,
			expectFallback: false,
		},
		{
			name:           "premium with function_calling capability",
			profile:        skill.ProfilePremium,
			capabilities:   []string{"function_calling"},
			expectError:    false,
			expectFallback: false,
		},
		{
			name:           "premium with both capabilities",
			profile:        skill.ProfilePremium,
			capabilities:   []string{"vision", "function_calling"},
			expectError:    false,
			expectFallback: false,
		},
		{
			name:           "cheap profile with no capabilities",
			profile:        skill.ProfileCheap,
			capabilities:   nil,
			expectError:    false,
			expectFallback: false,
		},
		{
			name:         "invalid profile",
			profile:      "invalid",
			capabilities: []string{"vision"},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := resolver.ResolveWithCapabilities(ctx, tt.profile, tt.capabilities)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resolution == nil {
					t.Fatal("expected non-nil resolution")
				}
			}
		})
	}
}

// TestResolverCostTracking tests the cost tracking functionality.
func TestResolverCostTracking(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	// Initial state should have zero costs
	summary := resolver.GetCostSummary()
	if summary.TotalCost != 0 {
		t.Errorf("expected initial total cost 0, got %f", summary.TotalCost)
	}

	// Track a cost for anthropic model
	breakdown := resolver.TrackCost("claude-3-5-sonnet-20241022", "anthropic", 1000, 500)
	if breakdown == nil {
		t.Fatal("expected non-nil breakdown")
	}

	// Cost should be: input = 1000 * 0.003 = 3.0, output = 500 * 0.015 = 7.5, total = 10.5
	expectedInputCost := 1000 * 0.003
	expectedOutputCost := 500 * 0.015
	expectedTotalCost := expectedInputCost + expectedOutputCost

	if breakdown.InputCost != expectedInputCost {
		t.Errorf("expected input cost %f, got %f", expectedInputCost, breakdown.InputCost)
	}
	if breakdown.OutputCost != expectedOutputCost {
		t.Errorf("expected output cost %f, got %f", expectedOutputCost, breakdown.OutputCost)
	}
	if breakdown.TotalCost != expectedTotalCost {
		t.Errorf("expected total cost %f, got %f", expectedTotalCost, breakdown.TotalCost)
	}

	// Verify summary is updated
	summary = resolver.GetCostSummary()
	if summary.TotalCost != expectedTotalCost {
		t.Errorf("expected summary total cost %f, got %f", expectedTotalCost, summary.TotalCost)
	}
	if summary.TotalInputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", summary.TotalInputTokens)
	}
	if summary.TotalOutputTokens != 500 {
		t.Errorf("expected 500 output tokens, got %d", summary.TotalOutputTokens)
	}

	// Track another cost
	resolver.TrackCost("gpt-4o", "openai", 500, 200)

	summary = resolver.GetCostSummary()
	if summary.TotalInputTokens != 1500 {
		t.Errorf("expected 1500 total input tokens, got %d", summary.TotalInputTokens)
	}
	if summary.TotalOutputTokens != 700 {
		t.Errorf("expected 700 total output tokens, got %d", summary.TotalOutputTokens)
	}

	// Verify by-provider breakdown
	if _, ok := summary.ByProvider["anthropic"]; !ok {
		t.Error("expected anthropic in ByProvider map")
	}
	if _, ok := summary.ByProvider["openai"]; !ok {
		t.Error("expected openai in ByProvider map")
	}

	// Test reset
	resolver.ResetCostTracking()
	summary = resolver.GetCostSummary()
	if summary.TotalCost != 0 {
		t.Errorf("expected reset total cost 0, got %f", summary.TotalCost)
	}
	if summary.TotalInputTokens != 0 {
		t.Errorf("expected reset input tokens 0, got %d", summary.TotalInputTokens)
	}
}

// TestResolverEstimateCost tests cost estimation without tracking.
func TestResolverEstimateCost(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	// Estimate cost for anthropic model
	breakdown := resolver.EstimateCost("claude-3-5-sonnet-20241022", "anthropic", 1000, 500)
	if breakdown == nil {
		t.Fatal("expected non-nil breakdown")
	}

	expectedInputCost := 1000 * 0.003
	expectedOutputCost := 500 * 0.015
	expectedTotalCost := expectedInputCost + expectedOutputCost

	if breakdown.TotalCost != expectedTotalCost {
		t.Errorf("expected total cost %f, got %f", expectedTotalCost, breakdown.TotalCost)
	}

	// Verify that EstimateCost doesn't affect the summary
	summary := resolver.GetCostSummary()
	if summary.TotalCost != 0 {
		t.Errorf("expected summary total cost 0 after estimate, got %f", summary.TotalCost)
	}
}

// TestResolverCostTrackingWithUnknownModel tests cost tracking for unknown models.
func TestResolverCostTrackingWithUnknownModel(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	// Track cost for unknown model - should use zero costs
	breakdown := resolver.TrackCost("unknown-model", "unknown-provider", 1000, 500)
	if breakdown == nil {
		t.Fatal("expected non-nil breakdown")
	}

	// Unknown models should have zero costs
	if breakdown.TotalCost != 0 {
		t.Errorf("expected zero cost for unknown model, got %f", breakdown.TotalCost)
	}

	summary := resolver.GetCostSummary()
	if summary.TotalInputTokens != 1000 {
		t.Errorf("expected 1000 input tokens tracked, got %d", summary.TotalInputTokens)
	}
}

// TestResolverGetProvider tests the GetProvider method.
func TestResolverGetProvider(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	provider := resolver.GetProvider("ollama")
	if provider == nil {
		t.Error("expected non-nil provider for ollama")
	}

	provider = resolver.GetProvider("nonexistent")
	if provider != nil {
		t.Error("expected nil provider for nonexistent")
	}
}

// TestResolverIsModelAvailable tests the IsModelAvailable method.
func TestResolverIsModelAvailable(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	ctx := context.Background()

	if !resolver.IsModelAvailable(ctx, "llama3.2:3b") {
		t.Error("expected llama3.2:3b to be available")
	}

	if resolver.IsModelAvailable(ctx, "nonexistent-model") {
		t.Error("expected nonexistent-model to not be available")
	}
}

// TestResolverGetEnabledProviders tests the GetEnabledProviders method.
func TestResolverGetEnabledProviders(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	providers := resolver.GetEnabledProviders()
	if len(providers) == 0 {
		t.Error("expected at least one enabled provider")
	}

	// Verify order by priority
	// ollama has priority 1, anthropic has 2, openai has 3
	expectedOrder := []string{"ollama", "anthropic", "openai"}
	for i, expected := range expectedOrder {
		if i < len(providers) && providers[i] != expected {
			t.Errorf("expected provider at index %d to be %s, got %s", i, expected, providers[i])
		}
	}
}

// TestResolverGetDefaultProvider tests the GetDefaultProvider method.
func TestResolverGetDefaultProvider(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	defaultProvider := resolver.GetDefaultProvider()
	if defaultProvider != "ollama" {
		t.Errorf("expected default provider 'ollama', got '%s'", defaultProvider)
	}
}

// TestResolverGetProfileConfig tests the GetProfileConfig method.
func TestResolverGetProfileConfig(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	profileConfig := resolver.GetProfileConfig(skill.ProfileBalanced)
	if profileConfig == nil {
		t.Fatal("expected non-nil profile config for balanced")
	}

	if profileConfig.GenerationModel != "llama3.2:8b" {
		t.Errorf("expected generation model 'llama3.2:8b', got '%s'", profileConfig.GenerationModel)
	}

	nilConfig := resolver.GetProfileConfig("invalid")
	if nilConfig != nil {
		t.Error("expected nil profile config for invalid profile")
	}
}

// TestResolutionModelConfig tests that Resolution includes model configuration.
func TestResolutionModelConfig(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	ctx := context.Background()

	resolution, err := resolver.Resolve(ctx, skill.ProfilePremium)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolution.ModelConfig == nil {
		t.Fatal("expected non-nil ModelConfig")
	}

	if resolution.ModelConfig.Tier != "premium" {
		t.Errorf("expected tier 'premium', got '%s'", resolution.ModelConfig.Tier)
	}

	if !resolution.ModelConfig.HasCapability("vision") {
		t.Error("expected claude model to have vision capability")
	}
}

// TestResolverConcurrentAccess tests thread safety of the resolver.
func TestResolverConcurrentAccess(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent resolves
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = resolver.Resolve(ctx, skill.ProfileBalanced)
			}
		}()
	}

	// Concurrent cost tracking
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				resolver.TrackCost("llama3.2:3b", "ollama", 100, 50)
			}
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = resolver.GetCostSummary()
				_ = resolver.GetEnabledProviders()
				_ = resolver.GetDefaultProvider()
			}
		}()
	}

	// Wait for all goroutines
	wg.Wait()

	// Verify state is consistent
	summary := resolver.GetCostSummary()
	if summary.TotalInputTokens < 1000 {
		t.Error("expected accumulated input tokens from concurrent tracking")
	}
}

// TestResolverFallbackResolution tests that fallback is correctly indicated.
func TestResolverFallbackResolution(t *testing.T) {
	// Create a registry where primary models are unavailable
	registry := adapterProvider.NewRegistry()
	ollamaProvider := newResolverMockProvider("ollama").
		withLocal(true).
		withModel("llama3.2:3b", true, false). // supported but NOT available
		withModel("llama3.2:1b", true, true)   // fallback available
	registry.Register(ollamaProvider)

	cfg := config.NewRoutingConfiguration()
	cfg.Profiles[skill.ProfileCheap].GenerationModel = "llama3.2:3b"
	cfg.Profiles[skill.ProfileCheap].FallbackModel = "llama3.2:1b"
	cfg.FallbackChain = []string{"ollama"}

	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	ctx := context.Background()

	resolution, err := resolver.Resolve(ctx, skill.ProfileCheap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resolution.IsFallback {
		t.Error("expected resolution to indicate fallback was used")
	}

	if resolution.ModelID != "llama3.2:1b" {
		t.Errorf("expected fallback model 'llama3.2:1b', got '%s'", resolution.ModelID)
	}
}

// TestResolverLocalModelCostTracking tests that local models have zero costs.
func TestResolverLocalModelCostTracking(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	// Local models (ollama) have zero costs
	breakdown := resolver.TrackCost("llama3.2:3b", "ollama", 10000, 5000)
	if breakdown.TotalCost != 0 {
		t.Errorf("expected zero cost for local model, got %f", breakdown.TotalCost)
	}

	// But tokens should still be tracked
	summary := resolver.GetCostSummary()
	if summary.TotalInputTokens != 10000 {
		t.Errorf("expected 10000 input tokens, got %d", summary.TotalInputTokens)
	}
}

// TestResolverMultipleProviderCostTracking tests tracking costs across multiple providers.
func TestResolverMultipleProviderCostTracking(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	// Track multiple costs
	resolver.TrackCost("claude-3-5-sonnet-20241022", "anthropic", 1000, 500)
	resolver.TrackCost("gpt-4o", "openai", 2000, 1000)
	resolver.TrackCost("llama3.2:3b", "ollama", 5000, 2500)

	summary := resolver.GetCostSummary()

	// Check total tokens
	expectedInputTokens := 1000 + 2000 + 5000
	expectedOutputTokens := 500 + 1000 + 2500
	if summary.TotalInputTokens != expectedInputTokens {
		t.Errorf("expected %d input tokens, got %d", expectedInputTokens, summary.TotalInputTokens)
	}
	if summary.TotalOutputTokens != expectedOutputTokens {
		t.Errorf("expected %d output tokens, got %d", expectedOutputTokens, summary.TotalOutputTokens)
	}

	// Check by-provider breakdown
	if len(summary.ByProvider) != 3 {
		t.Errorf("expected 3 providers in breakdown, got %d", len(summary.ByProvider))
	}

	// Check by-model breakdown
	if len(summary.ByModel) != 3 {
		t.Errorf("expected 3 models in breakdown, got %d", len(summary.ByModel))
	}
}

// TestResolverResolutionFields tests all fields of the Resolution struct.
func TestResolverResolutionFields(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	ctx := context.Background()

	resolution, err := resolver.Resolve(ctx, skill.ProfilePremium)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check all fields are populated
	if resolution.ModelID == "" {
		t.Error("expected non-empty ModelID")
	}
	if resolution.ProviderName == "" {
		t.Error("expected non-empty ProviderName")
	}
	// IsFallback should be false for primary resolution
	if resolution.IsFallback {
		t.Error("expected IsFallback to be false for primary resolution")
	}
	// ModelConfig should be populated
	if resolution.ModelConfig == nil {
		t.Error("expected non-nil ModelConfig")
	}
}

// TestResolverCostSummaryClone tests that GetCostSummary returns a clone.
func TestResolverCostSummaryClone(t *testing.T) {
	cfg := createResolverTestRoutingConfig()
	registry := createResolverTestRegistry()
	router, _ := NewRouter(cfg, registry)
	resolver, _ := NewResolver(router, registry, cfg)

	// Track a cost
	resolver.TrackCost("claude-3-5-sonnet-20241022", "anthropic", 1000, 500)

	// Get summary
	summary1 := resolver.GetCostSummary()

	// Track another cost
	resolver.TrackCost("gpt-4o", "openai", 500, 250)

	// Get new summary
	summary2 := resolver.GetCostSummary()

	// Original summary should not be affected
	if summary1.TotalInputTokens != 1000 {
		t.Errorf("expected original summary to have 1000 input tokens, got %d", summary1.TotalInputTokens)
	}

	// New summary should have updated values
	if summary2.TotalInputTokens != 1500 {
		t.Errorf("expected new summary to have 1500 input tokens, got %d", summary2.TotalInputTokens)
	}
}
