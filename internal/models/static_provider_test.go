package models

import (
	"context"
	"errors"
	"testing"
)

func TestStaticProviderSetAvailability(t *testing.T) {
	ctx := context.Background()
	provider := NewStaticProvider(ProviderInfo{
		Name: "test",
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

	// Initially available
	available, err := provider.IsModelAvailable(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if !available {
		t.Fatal("expected model to be available")
	}

	// Set to unavailable
	provider.SetAvailability("gpt-4", false)
	available, err = provider.IsModelAvailable(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if available {
		t.Fatal("expected model to be unavailable after SetAvailability(false)")
	}

	// Set back to available
	provider.SetAvailability("gpt-4", true)
	available, err = provider.IsModelAvailable(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if !available {
		t.Fatal("expected model to be available after SetAvailability(true)")
	}
}

func TestStaticProviderSetAvailabilityNonexistentModel(t *testing.T) {
	provider := NewStaticProvider(ProviderInfo{
		Name: "test",
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

	// Setting availability on nonexistent model should not panic
	provider.SetAvailability("nonexistent", false)
	provider.SetAvailability("nonexistent", true)
}

func TestStaticProviderIsModelAvailableNotFound(t *testing.T) {
	ctx := context.Background()
	provider := NewStaticProvider(ProviderInfo{
		Name: "test",
		Type: ProviderTypeCloud,
	}, nil)

	_, err := provider.IsModelAvailable(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestStaticProviderModelMetadataNotFound(t *testing.T) {
	ctx := context.Background()
	provider := NewStaticProvider(ProviderInfo{
		Name: "test",
		Type: ProviderTypeCloud,
	}, nil)

	_, err := provider.ModelMetadata(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestStaticProviderResolveModelUnavailable(t *testing.T) {
	ctx := context.Background()
	provider := NewStaticProvider(ProviderInfo{
		Name: "test",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "GPT-4",
			Available:       false,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	_, err := provider.ResolveModel(ctx, "gpt-4")
	if err == nil {
		t.Fatal("expected error for unavailable model")
	}
	if !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("expected ErrModelUnavailable, got %v", err)
	}
}

func TestStaticProviderResolveModelNotFound(t *testing.T) {
	ctx := context.Background()
	provider := NewStaticProvider(ProviderInfo{
		Name: "test",
		Type: ProviderTypeCloud,
	}, nil)

	_, err := provider.ResolveModel(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestStaticProviderModels(t *testing.T) {
	ctx := context.Background()
	provider := NewStaticProvider(ProviderInfo{
		Name: "test",
		Type: ProviderTypeCloud,
	}, []StaticModel{
		{
			Name:            "gpt-4",
			Route:           "gpt-4",
			Description:     "GPT-4 Model",
			Available:       true,
			Tier:            AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
		{
			Name:            "gpt-3.5",
			Route:           "gpt-3.5-turbo",
			Description:     "GPT-3.5 Model",
			Available:       true,
			Tier:            AgentTierBalanced,
			CostPer1KTokens: 0.015,
		},
	})

	models, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	found := make(map[string]bool)
	for _, m := range models {
		found[m.Name] = true
	}

	if !found["gpt-4"] || !found["gpt-3.5"] {
		t.Fatal("expected both models to be listed")
	}
}

func TestStaticProviderSupportsModel(t *testing.T) {
	ctx := context.Background()
	provider := NewStaticProvider(ProviderInfo{
		Name: "test",
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

	supports, err := provider.SupportsModel(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("SupportsModel returned error: %v", err)
	}
	if !supports {
		t.Fatal("expected provider to support gpt-4")
	}

	supports, err = provider.SupportsModel(ctx, "missing")
	if err != nil {
		t.Fatalf("SupportsModel returned error: %v", err)
	}
	if supports {
		t.Fatal("expected provider not to support missing model")
	}
}
