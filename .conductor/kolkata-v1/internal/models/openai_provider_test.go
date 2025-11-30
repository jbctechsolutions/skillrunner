package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIProviderBasicFlow(t *testing.T) {
	modelList := openaiModelList{
		Data: []openaiModel{
			{ID: "gpt-4.1"},
			{ID: "gpt-3.5-turbo"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			if r.Header.Get("Authorization") != "Bearer test-key" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if err := json.NewEncoder(w).Encode(modelList); err != nil {
				t.Fatalf("encode model list: %v", err)
			}
		case "/v1/chat/completions":
			if r.Header.Get("Authorization") != "Bearer test-key" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("test-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewOpenAIProvider error: %v", err)
	}

	ctx := context.Background()

	models, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	ok, err := provider.SupportsModel(ctx, "gpt-4.1")
	if err != nil {
		t.Fatalf("SupportsModel error: %v", err)
	}
	if !ok {
		t.Fatal("expected support for gpt-4.1")
	}

	meta, err := provider.ModelMetadata(ctx, "gpt-4.1")
	if err != nil {
		t.Fatalf("ModelMetadata error: %v", err)
	}
	if meta.Tier != AgentTierPowerful {
		t.Fatalf("expected powerful tier, got %v", meta.Tier)
	}

	resolved, err := provider.ResolveModel(ctx, "gpt-4.1")
	if err != nil {
		t.Fatalf("ResolveModel error: %v", err)
	}
	if resolved.Provider.Name != "openai-live" {
		t.Fatalf("unexpected provider: %s", resolved.Provider.Name)
	}

	resp, err := provider.Generate(ctx, "gpt-4.1", "say hello", false, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp != "hello" {
		t.Fatalf("unexpected response: %s", resp)
	}
}

func TestOpenAIProviderIsModelAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(openaiModelList{
				Data: []openaiModel{
					{ID: "gpt-4"},
				},
			})
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	available, err := provider.IsModelAvailable(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if !available {
		t.Fatal("expected gpt-4 to be available")
	}

	available, err = provider.IsModelAvailable(ctx, "missing")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if available {
		t.Fatal("expected missing model to be unavailable")
	}
}

func TestOpenAIProviderModelMetadataNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(openaiModelList{Data: []openaiModel{}})
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.ModelMetadata(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestOpenAIProviderResolveModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(openaiModelList{Data: []openaiModel{}})
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.ResolveModel(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestOpenAIProviderCheckModelHealthNoAPIKey(t *testing.T) {
	// Create provider with empty API key to test the check
	provider := &OpenAIProvider{
		apiKey:      "",
		baseURL:     "https://api.openai.com",
		client:      http.DefaultClient,
		info:        ProviderInfo{Name: "openai-live", Type: ProviderTypeCloud},
		modelsCache: make(map[string]openaiModel),
		cacheTTL:    2 * time.Minute,
	}
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("CheckModelHealth should not return error, got: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected unhealthy status when API key is missing")
	}

	if health.Message != "OpenAI API key not configured" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions, got %d", len(health.Suggestions))
	}

	expectedSuggestion := "Set OPENAI_API_KEY environment variable"
	if health.Suggestions[0] != expectedSuggestion {
		t.Errorf("expected first suggestion to be %q, got %q", expectedSuggestion, health.Suggestions[0])
	}
}

func TestOpenAIProviderCheckModelHealthModelFound(t *testing.T) {
	modelList := openaiModelList{
		Data: []openaiModel{
			{ID: "gpt-4"},
			{ID: "gpt-3.5-turbo"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "gpt-4")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if !health.Healthy {
		t.Fatal("expected model to be healthy")
	}

	if health.Message != "Model 'gpt-4' is available" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if health.Suggestions != nil {
		t.Error("expected no suggestions for healthy model")
	}

	if health.Details["model_id"] != "gpt-4" {
		t.Errorf("expected model_id in details")
	}

	if health.Details["provider"] != "openai" {
		t.Errorf("expected provider in details")
	}
}

func TestOpenAIProviderCheckModelHealthModelNotFound(t *testing.T) {
	modelList := openaiModelList{
		Data: []openaiModel{
			{ID: "gpt-4"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "gpt-5-unknown")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected model to be unhealthy")
	}

	if health.Message != "Model 'gpt-5-unknown' not found in OpenAI catalog" {
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

func TestOpenAIProviderCheckModelHealthAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			http.Error(w, "API error", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "any-model")
	if err != nil {
		t.Fatalf("CheckModelHealth should not return error, got: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected unhealthy status for API error")
	}

	if health.Message != "Unable to fetch models from OpenAI API" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) < 3 {
		t.Errorf("expected at least 3 suggestions, got %d", len(health.Suggestions))
	}

	if _, ok := health.Details["error"]; !ok {
		t.Error("expected error in details")
	}
}
