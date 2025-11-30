package models

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestResolvePrefersLocalProvider(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	cloud := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "OpenAI GPT-4",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	local := NewStaticProvider(ProviderInfo{
		Name: "ollama",
		Type: ProviderTypeLocal,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4:local",
			Description:     "Local GPT-4 mirror",
			Available:       true,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.012,
		},
	})

	orchestrator.RegisterProvider(cloud)
	orchestrator.RegisterProvider(local)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model:             "gpt-4",
		PreferredProvider: ProviderTypeLocal,
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if result.Provider.Type != ProviderTypeLocal {
		t.Fatalf("expected local provider, got %s", result.Provider.Type)
	}
	if result.Route != "gpt-4:local" {
		t.Fatalf("expected local route, got %s", result.Route)
	}
	if result.Tier != AgentTierBalanced {
		t.Fatalf("expected balanced tier, got %v", result.Tier)
	}
	if result.CostPer1KTokens != 0.012 {
		t.Fatalf("expected cost 0.012, got %f", result.CostPer1KTokens)
	}
}

func TestResolveFallsBackToCloudWhenLocalUnavailable(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	local := NewStaticProvider(ProviderInfo{
		Name: "ollama",
		Type: ProviderTypeLocal,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4:local",
			Description:     "Local GPT-4 mirror",
			Available:       false,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.012,
		},
	})

	cloud := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "OpenAI GPT-4",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	orchestrator.RegisterProvider(local)
	orchestrator.RegisterProvider(cloud)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model:             "gpt-4",
		PreferredProvider: ProviderTypeLocal,
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if result.Provider.Type != ProviderTypeCloud {
		t.Fatalf("expected cloud provider fallback, got %s", result.Provider.Type)
	}
	if result.Route != "gpt-4" {
		t.Fatalf("expected cloud route, got %s", result.Route)
	}
	if result.Tier != AgentTierPowerful {
		t.Fatalf("expected powerful tier, got %v", result.Tier)
	}
}

func TestResolveUsesFallbackModels(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	cloud := NewStaticProvider(ProviderInfo{
		Name: "anthropic",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "claude-3-sonnet",
			Route:           "claude-3-sonnet",
			Description:     "Anthropic Claude 3 Sonnet",
			Available:       true,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.015,
		},
	})

	orchestrator.RegisterProvider(cloud)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model:          "nonexistent",
		FallbackModels: []string{"claude-3-sonnet"},
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if result.Name != "claude-3-sonnet" {
		t.Fatalf("expected fallback model, got %s", result.Name)
	}
	if result.Tier != AgentTierBalanced {
		t.Fatalf("expected balanced tier, got %v", result.Tier)
	}
	if result.CostPer1KTokens != 0.015 {
		t.Fatalf("expected cost 0.015, got %f", result.CostPer1KTokens)
	}
}

func TestResolveErrorsWhenModelMissing(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	empty := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, nil)
	orchestrator.RegisterProvider(empty)

	_, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model: "missing",
	})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestListModelsReportsAvailability(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	local := NewStaticProvider(ProviderInfo{
		Name: "ollama",
		Type: ProviderTypeLocal,
	}, []StaticModel{
		{
			Name:            "llama3",
			Route:           "llama3",
			Description:     "Ollama LLaMA 3",
			Available:       true,
			Tier:            AgentTierFast,
			CostPer1KTokens: 0.002,
		},
	})

	cloud := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "OpenAI GPT-4",
			Available:       false,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	orchestrator.RegisterProvider(local)
	orchestrator.RegisterProvider(cloud)

	results, err := orchestrator.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 models, got %d", len(results))
	}

	for _, r := range results {
		switch r.Name {
		case "llama3":
			if !r.Available {
				t.Fatal("expected llama3 to be available")
			}
			if r.Tier != AgentTierFast {
				t.Fatalf("expected llama3 tier fast, got %v", r.Tier)
			}
			if r.CostPer1KTokens != 0.002 {
				t.Fatalf("expected llama3 cost 0.002, got %f", r.CostPer1KTokens)
			}
		case "gpt-4":
			if r.Available {
				t.Fatal("expected gpt-4 to be unavailable")
			}
			if r.Tier != AgentTierPowerful {
				t.Fatalf("expected gpt-4 tier powerful, got %v", r.Tier)
			}
			if r.CostPer1KTokens != 0.03 {
				t.Fatalf("expected gpt-4 cost 0.03, got %f", r.CostPer1KTokens)
			}
		default:
			t.Fatalf("unexpected model %s", r.Name)
		}
	}
}

func TestIsAvailable(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	cloud := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "OpenAI GPT-4",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})
	orchestrator.RegisterProvider(cloud)

	available, err := orchestrator.IsAvailable(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("IsAvailable returned error: %v", err)
	}
	if !available {
		t.Fatal("expected gpt-4 to be available")
	}

	available, err = orchestrator.IsAvailable(ctx, "missing")
	if err != nil {
		t.Fatalf("IsAvailable returned error: %v", err)
	}
	if available {
		t.Fatal("expected missing model to be unavailable")
	}
}

func TestResolvePrefersHigherSuccessRateDespiteCost(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	expensive := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "Premium model",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	cheap := NewStaticProvider(ProviderInfo{
		Name: "anthropic",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4-anthropic",
			Description:     "Lower cost alternative",
			Available:       true,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.01,
		},
	})

	orchestrator.RegisterProvider(expensive)
	orchestrator.RegisterProvider(cheap)

	orchestrator.metrics["openai"] = &ProviderMetrics{
		TotalCalls:    20,
		SuccessCalls:  19,
		FailureCalls:  1,
		TotalLatency:  0,
		TotalCostHint: 0.0,
	}
	orchestrator.metrics["anthropic"] = &ProviderMetrics{
		TotalCalls:    20,
		SuccessCalls:  10,
		FailureCalls:  10,
		TotalLatency:  0,
		TotalCostHint: 0.0,
	}

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model: "gpt-4",
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if result.Provider.Name != "openai" {
		t.Fatalf("expected openai provider due to success rate, got %s", result.Provider.Name)
	}
}

func TestResolvePrefersLowerCostWhenSuccessEqual(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	highCost := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "Higher cost",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	lowCost := NewStaticProvider(ProviderInfo{
		Name: "mistral",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4-mistral",
			Description:     "Lower cost equivalent",
			Available:       true,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.005,
		},
	})

	orchestrator.RegisterProvider(highCost)
	orchestrator.RegisterProvider(lowCost)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model: "gpt-4",
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if result.Provider.Name != "mistral" {
		t.Fatalf("expected mistral provider due to lower cost, got %s", result.Provider.Name)
	}
}

func TestResolvePolicyLocalFirst(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	cloud := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "Cloud GPT-4",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.015,
		},
	})

	local := NewStaticProvider(ProviderInfo{
		Name: "ollama",
		Type: ProviderTypeLocal,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4:local",
			Description:     "Local GPT-4",
			Available:       true,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.02,
		},
	})

	orchestrator.RegisterProvider(cloud)
	orchestrator.RegisterProvider(local)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model:  "gpt-4",
		Policy: ResolvePolicyLocalFirst,
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}
	if result.Provider.Type != ProviderTypeLocal {
		t.Fatalf("expected local provider, got %s", result.Provider.Type)
	}
}

func TestResolvePolicyPerformanceFirst(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	balanced := NewStaticProvider(ProviderInfo{
		Name: "anthropic",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "claude-sonnet",
			Route:           "claude-sonnet",
			Description:     "Claude Sonnet",
			Available:       true,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.02,
		},
	})

	powerful := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "GPT-4",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	orchestrator.RegisterProvider(balanced)
	orchestrator.RegisterProvider(powerful)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model:  "gpt-4",
		Policy: ResolvePolicyPerformanceFirst,
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}
	if result.Provider.Name != "openai" {
		t.Fatalf("expected openai provider, got %s", result.Provider.Name)
	}
}

func TestResolvePolicyCostOptimized(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	cheap := NewStaticProvider(ProviderInfo{
		Name: "mistral",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4-mistral",
			Description:     "Budget GPT",
			Available:       true,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.004,
		},
	})

	premium := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "Premium GPT",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	orchestrator.RegisterProvider(cheap)
	orchestrator.RegisterProvider(premium)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model:  "gpt-4",
		Policy: ResolvePolicyCostOptimized,
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}
	if result.Provider.Name != "mistral" {
		t.Fatalf("expected mistral provider for cost optimized, got %s", result.Provider.Name)
	}
}

func TestResolveRetriesAndUpdatesMetrics(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	baseProvider := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "Flaky model",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	flaky := &flakyProvider{
		StaticProvider: baseProvider,
		failUntil:      1,
	}

	orchestrator.RegisterProvider(flaky)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model: "gpt-4",
	})
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if result.Provider.Name != "openai" {
		t.Fatalf("expected openai provider, got %s", result.Provider.Name)
	}

	metrics := orchestrator.Metrics("openai")
	if metrics.TotalCalls != 2 {
		t.Fatalf("expected 2 total calls (1 retry), got %d", metrics.TotalCalls)
	}
	if metrics.SuccessCalls != 1 || metrics.FailureCalls != 1 {
		t.Fatalf("expected 1 success and 1 failure, got success=%d failure=%d", metrics.SuccessCalls, metrics.FailureCalls)
	}
}

type flakyProvider struct {
	*StaticProvider
	failUntil int
	calls     int
}

func (f *flakyProvider) ResolveModel(ctx context.Context, model string) (*ResolvedModel, error) {
	if f.calls < f.failUntil {
		f.calls++
		return nil, fmt.Errorf("%w: transient failure", ErrModelUnavailable)
	}
	f.calls++
	return f.StaticProvider.ResolveModel(ctx, model)
}

func TestResolveNoProvidersRegistered(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	_, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model: "gpt-4",
	})

	if err == nil {
		t.Fatal("expected error when no providers registered")
	}
}

func TestResolveWithMultipleFallbacks(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	provider := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4-turbo",
			Route:           "gpt-4-turbo",
			Description:     "GPT-4 Turbo",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.01,
		},
	})

	orchestrator.RegisterProvider(provider)

	result, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model:          "nonexistent-1",
		FallbackModels: []string{"nonexistent-2", "gpt-4-turbo"},
	})

	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if result.Name != "gpt-4-turbo" {
		t.Fatalf("expected gpt-4-turbo from fallbacks, got %s", result.Name)
	}
}

func TestResolveAllFallbacksFail(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewOrchestrator()

	provider := NewStaticProvider(ProviderInfo{
		Name: "openai",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "GPT-4",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	orchestrator.RegisterProvider(provider)

	_, err := orchestrator.ResolveModel(ctx, ResolveRequest{
		Model:          "nonexistent-1",
		FallbackModels: []string{"nonexistent-2", "nonexistent-3"},
	})

	if err == nil {
		t.Fatal("expected error when all fallbacks fail")
	}

	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestTierBonus(t *testing.T) {
	tests := []struct {
		tier     AgentTier
		expected float64
	}{
		{AgentTierFast, fastTierBonus},
		{AgentTierBalanced, balancedTierBonus},
		{AgentTierPowerful, powerfulTierBonus},
		{AgentTier(999), 0.0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.tier), func(t *testing.T) {
			result := tierBonus(tt.tier)
			if result != tt.expected {
				t.Errorf("tierBonus(%v) = %f, want %f", tt.tier, result, tt.expected)
			}
		})
	}
}

func TestComputeCostScore(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		minCost  float64
		maxCost  float64
		expected float64
	}{
		{"Low cost in range", 0.01, 0.01, 0.03, 1.0},
		{"High cost in range", 0.03, 0.01, 0.03, 0.0},
		{"Mid cost in range", 0.02, 0.01, 0.03, 0.5},
		{"Zero cost", 0.0, 0.01, 0.03, defaultCostScore},
		{"Equal min and max", 0.02, 0.02, 0.02, defaultCostScore},
		{"Cost below min", 0.005, 0.01, 0.03, 1.0},
		{"Cost above max", 0.05, 0.01, 0.03, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeCostScore(tt.cost, tt.minCost, tt.maxCost)
			// Use approximate comparison for floating point
			const epsilon = 1e-9
			if diff := result - tt.expected; diff < -epsilon || diff > epsilon {
				t.Errorf("computeCostScore(%f, %f, %f) = %f, want %f (diff: %e)",
					tt.cost, tt.minCost, tt.maxCost, result, tt.expected, diff)
			}
		})
	}
}

func TestHealthStatusStructure(t *testing.T) {
	// Test that HealthStatus can be created and used properly
	health := &HealthStatus{
		Healthy: true,
		Message: "Model is available",
		Suggestions: []string{
			"First suggestion",
			"Second suggestion",
		},
		Details: map[string]interface{}{
			"model_name": "test-model",
			"provider":   "test-provider",
			"size_gb":    10.5,
		},
	}

	if !health.Healthy {
		t.Error("expected Healthy to be true")
	}

	if health.Message != "Model is available" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(health.Suggestions))
	}

	if health.Details["model_name"] != "test-model" {
		t.Errorf("unexpected model_name in details: %v", health.Details["model_name"])
	}

	if sizeGB, ok := health.Details["size_gb"].(float64); !ok || sizeGB != 10.5 {
		t.Errorf("unexpected size_gb in details: %v", health.Details["size_gb"])
	}

	// Test unhealthy status with no suggestions
	unhealthy := &HealthStatus{
		Healthy:     false,
		Message:     "Model not found",
		Suggestions: nil,
		Details: map[string]interface{}{
			"error": "model does not exist",
		},
	}

	if unhealthy.Healthy {
		t.Error("expected Healthy to be false")
	}

	if unhealthy.Suggestions != nil {
		t.Error("expected nil suggestions")
	}
}
