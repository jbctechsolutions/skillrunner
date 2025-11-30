package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnthropicProviderBasicFlow(t *testing.T) {
	modelList := anthropicModelList{
		Data: []struct {
			ID string `json:"id"`
		}{
			{ID: "claude-3-opus-20240229"},
			{ID: "claude-3-sonnet-20240229"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			if r.Header.Get("x-api-key") != "anthropic-key" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if err := json.NewEncoder(w).Encode(modelList); err != nil {
				t.Fatalf("encode models: %v", err)
			}
		case "/v1/messages":
			if r.Header.Get("x-api-key") != "anthropic-key" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"content":[{"text":"hello from anthropic"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := NewAnthropicProvider("anthropic-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewAnthropicProvider error: %v", err)
	}

	ctx := context.Background()

	models, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("Models error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	ok, err := provider.SupportsModel(ctx, "claude-3-opus-20240229")
	if err != nil {
		t.Fatalf("SupportsModel error: %v", err)
	}
	if !ok {
		t.Fatal("expected support for claude-3-opus-20240229")
	}

	meta, err := provider.ModelMetadata(ctx, "claude-3-opus-20240229")
	if err != nil {
		t.Fatalf("ModelMetadata error: %v", err)
	}
	if meta.Tier != AgentTierPowerful {
		t.Fatalf("expected powerful tier, got %v", meta.Tier)
	}

	resolved, err := provider.ResolveModel(ctx, "claude-3-opus-20240229")
	if err != nil {
		t.Fatalf("ResolveModel error: %v", err)
	}
	if resolved.Provider.Name != "anthropic-live" {
		t.Fatalf("unexpected provider: %s", resolved.Provider.Name)
	}

	resp, err := provider.Generate(ctx, "claude-3-opus-20240229", "hello", false, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp != "hello from anthropic" {
		t.Fatalf("unexpected response: %s", resp)
	}
}

func TestAnthropicProviderModelMetadataNotFound(t *testing.T) {
	provider, _ := NewAnthropicProvider("test-key", "", nil)
	ctx := context.Background()

	_, err := provider.ModelMetadata(ctx, "nonexistent-model")
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestAnthropicProviderIsModelAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(anthropicModelList{
				Data: []struct {
					ID string `json:"id"`
				}{
					{ID: "claude-3-sonnet-20240229"},
				},
			})
		}
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	available, err := provider.IsModelAvailable(ctx, "claude-3-sonnet-20240229")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if !available {
		t.Fatal("expected claude-3-sonnet to be available")
	}

	available, err = provider.IsModelAvailable(ctx, "unknown-model")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if available {
		t.Fatal("expected unknown model to be unavailable")
	}
}

func TestAnthropicProviderResolveModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(anthropicModelList{
				Data: []struct {
					ID string `json:"id"`
				}{},
			})
		}
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.ResolveModel(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestAnthropicProviderCheckModelHealthNoAPIKey(t *testing.T) {
	// Create provider with empty API key to test the check
	provider := &AnthropicProvider{
		apiKey:      "",
		baseURL:     "https://api.anthropic.com",
		client:      http.DefaultClient,
		info:        ProviderInfo{Name: "anthropic-live", Type: ProviderTypeCloud},
		modelsCache: make(map[string]struct{}),
		cacheTTL:    2 * time.Minute,
	}
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "claude-3-sonnet-20240229")
	if err != nil {
		t.Fatalf("CheckModelHealth should not return error, got: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected unhealthy status when API key is missing")
	}

	if health.Message != "Anthropic API key not configured" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions, got %d", len(health.Suggestions))
	}

	expectedSuggestion := "Set ANTHROPIC_API_KEY environment variable"
	if health.Suggestions[0] != expectedSuggestion {
		t.Errorf("expected first suggestion to be %q, got %q", expectedSuggestion, health.Suggestions[0])
	}
}

func TestAnthropicProviderCheckModelHealthModelFound(t *testing.T) {
	modelList := anthropicModelList{
		Data: []struct {
			ID string `json:"id"`
		}{
			{ID: "claude-3-opus-20240229"},
			{ID: "claude-3-sonnet-20240229"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "claude-3-sonnet-20240229")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if !health.Healthy {
		t.Fatal("expected model to be healthy")
	}

	if health.Message != "Model 'claude-3-sonnet-20240229' is available" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if health.Suggestions != nil {
		t.Error("expected no suggestions for healthy model")
	}

	if health.Details["model_id"] != "claude-3-sonnet-20240229" {
		t.Errorf("expected model_id in details")
	}

	if health.Details["provider"] != "anthropic" {
		t.Errorf("expected provider in details")
	}
}

func TestAnthropicProviderCheckModelHealthModelNotFound(t *testing.T) {
	modelList := anthropicModelList{
		Data: []struct {
			ID string `json:"id"`
		}{
			{ID: "claude-3-opus-20240229"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "claude-3-unknown-model")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected model to be unhealthy")
	}

	if health.Message != "Model 'claude-3-unknown-model' not found in Anthropic catalog" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) != 3 {
		t.Errorf("expected 3 suggestions, got %d", len(health.Suggestions))
	}

	knownModels, ok := health.Details["known_models"].([]string)
	if !ok {
		t.Fatal("expected known_models in details")
	}

	if len(knownModels) != 1 {
		t.Errorf("expected 1 known model, got %d", len(knownModels))
	}
}

func TestAnthropicProviderCheckModelHealthAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			http.Error(w, "API error", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "any-model")
	if err != nil {
		t.Fatalf("CheckModelHealth should not return error, got: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected unhealthy status for API error")
	}

	if health.Message != "Unable to fetch models from Anthropic API" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) < 3 {
		t.Errorf("expected at least 3 suggestions, got %d", len(health.Suggestions))
	}

	if _, ok := health.Details["error"]; !ok {
		t.Error("expected error in details")
	}
}
