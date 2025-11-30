package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaProviderBasicOperations(t *testing.T) {
	tagsResponse := ollamaTagsResponse{
		Models: []struct {
			Name       string                 `json:"name"`
			Model      string                 `json:"model"`
			Digest     string                 `json:"digest"`
			ModifiedAt string                 `json:"modified_at"`
			Size       int64                  `json:"size"`
			Details    map[string]interface{} `json:"details"`
		}{
			{
				Name:       "llama3",
				Model:      "Meta/llama3",
				Digest:     "digest-123",
				ModifiedAt: "2024-01-02T03:04:05Z",
				Size:       123456789,
				Details: map[string]interface{}{
					"description": "Meta llama3 local",
				},
			},
			{
				Name:       "codellama",
				Model:      "Meta/codellama",
				Digest:     "digest-456",
				ModifiedAt: "2024-02-02T03:04:05Z",
				Size:       987654321,
				Details:    map[string]interface{}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			if err := json.NewEncoder(w).Encode(tagsResponse); err != nil {
				t.Fatalf("encode tags response: %v", err)
			}
		case "/api/generate":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"model":"llama3","created_at":"now","response":"hello","done":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewOllamaProvider returned error: %v", err)
	}

	ctx := context.Background()

	models, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	supports, err := provider.SupportsModel(ctx, "llama3")
	if err != nil {
		t.Fatalf("SupportsModel returned error: %v", err)
	}
	if !supports {
		t.Fatal("expected provider to support llama3")
	}

	metadata, err := provider.ModelMetadata(ctx, "llama3")
	if err != nil {
		t.Fatalf("ModelMetadata returned error: %v", err)
	}
	if metadata.Tier != AgentTierFast {
		t.Fatalf("expected tier fast, got %v", metadata.Tier)
	}

	resolved, err := provider.ResolveModel(ctx, "llama3")
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}
	if resolved.Provider.Name != "ollama" {
		t.Fatalf("expected provider name ollama, got %s", resolved.Provider.Name)
	}
	if resolved.Route != server.URL+"/api/generate" {
		t.Fatalf("unexpected route: %s", resolved.Route)
	}

	stream, err := provider.Generate(ctx, "llama3", "hi", false, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	response := <-stream
	if response != "hello" {
		t.Fatalf("expected response 'hello', got %s", response)
	}
}

func TestOllamaProviderInfo(t *testing.T) {
	provider, err := NewOllamaProvider("http://localhost:11434", nil)
	if err != nil {
		t.Fatalf("NewOllamaProvider returned error: %v", err)
	}

	info := provider.Info()
	if info.Name != "ollama" {
		t.Errorf("expected provider name 'ollama', got %s", info.Name)
	}
	if info.Type != ProviderTypeLocal {
		t.Errorf("expected provider type local, got %s", info.Type)
	}
}

func TestOllamaProviderIsModelAvailable(t *testing.T) {
	tagsResponse := ollamaTagsResponse{
		Models: []struct {
			Name       string                 `json:"name"`
			Model      string                 `json:"model"`
			Digest     string                 `json:"digest"`
			ModifiedAt string                 `json:"modified_at"`
			Size       int64                  `json:"size"`
			Details    map[string]interface{} `json:"details"`
		}{
			{Name: "llama3"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			json.NewEncoder(w).Encode(tagsResponse)
		}
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(server.URL, server.Client())
	ctx := context.Background()

	available, err := provider.IsModelAvailable(ctx, "llama3")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if !available {
		t.Fatal("expected llama3 to be available")
	}

	available, err = provider.IsModelAvailable(ctx, "missing")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if available {
		t.Fatal("expected missing model to be unavailable")
	}
}

func TestOllamaProviderModelMetadataNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			json.NewEncoder(w).Encode(ollamaTagsResponse{Models: []struct {
				Name       string                 `json:"name"`
				Model      string                 `json:"model"`
				Digest     string                 `json:"digest"`
				ModifiedAt string                 `json:"modified_at"`
				Size       int64                  `json:"size"`
				Details    map[string]interface{} `json:"details"`
			}{}})
		}
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.ModelMetadata(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestOllamaProviderResolveModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			json.NewEncoder(w).Encode(ollamaTagsResponse{Models: []struct {
				Name       string                 `json:"name"`
				Model      string                 `json:"model"`
				Digest     string                 `json:"digest"`
				ModifiedAt string                 `json:"modified_at"`
				Size       int64                  `json:"size"`
				Details    map[string]interface{} `json:"details"`
			}{}})
		}
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.ResolveModel(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestOllamaProviderCheckModelHealthModelFound(t *testing.T) {
	tagsResponse := ollamaTagsResponse{
		Models: []struct {
			Name       string                 `json:"name"`
			Model      string                 `json:"model"`
			Digest     string                 `json:"digest"`
			ModifiedAt string                 `json:"modified_at"`
			Size       int64                  `json:"size"`
			Details    map[string]interface{} `json:"details"`
		}{
			{
				Name:       "qwen2.5:14b",
				Model:      "qwen2.5",
				Digest:     "abc123",
				ModifiedAt: "2024-01-15T10:30:00Z",
				Size:       9663676416, // ~9GB
				Details: map[string]interface{}{
					"parameter_size": "14B",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			json.NewEncoder(w).Encode(tagsResponse)
		}
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "qwen2.5:14b")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if !health.Healthy {
		t.Fatal("expected model to be healthy")
	}

	if health.Message != "Model 'qwen2.5:14b' is available" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if health.Suggestions != nil {
		t.Error("expected no suggestions for healthy model")
	}

	if health.Details["model_name"] != "qwen2.5:14b" {
		t.Errorf("expected model_name in details, got: %v", health.Details)
	}

	if health.Details["size_gb"] != "9.0" {
		t.Errorf("expected size_gb 9.0, got: %v", health.Details["size_gb"])
	}
}

func TestOllamaProviderCheckModelHealthModelNotFound(t *testing.T) {
	tagsResponse := ollamaTagsResponse{
		Models: []struct {
			Name       string                 `json:"name"`
			Model      string                 `json:"model"`
			Digest     string                 `json:"digest"`
			ModifiedAt string                 `json:"modified_at"`
			Size       int64                  `json:"size"`
			Details    map[string]interface{} `json:"details"`
		}{
			{
				Name:       "llama3:8b",
				Model:      "llama3",
				Digest:     "xyz789",
				ModifiedAt: "2024-01-10T08:00:00Z",
				Size:       4661209088, // ~4.3GB
				Details:    map[string]interface{}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			json.NewEncoder(w).Encode(tagsResponse)
		}
	}))
	defer server.Close()

	provider, _ := NewOllamaProvider(server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "missing-model:7b")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected model to be unhealthy")
	}

	if health.Message != "Model 'missing-model:7b' not found in Ollama" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) != 3 {
		t.Errorf("expected 3 suggestions, got %d", len(health.Suggestions))
	}

	expectedSuggestions := []string{
		"Pull the model: ollama pull missing-model:7b",
		"Check model name spelling",
		"List available models: ollama list",
	}

	for i, expected := range expectedSuggestions {
		if health.Suggestions[i] != expected {
			t.Errorf("suggestion %d: expected %q, got %q", i, expected, health.Suggestions[i])
		}
	}

	availableModels, ok := health.Details["available_models"].([]string)
	if !ok {
		t.Fatal("expected available_models in details")
	}

	if len(availableModels) != 1 {
		t.Errorf("expected 1 available model, got %d", len(availableModels))
	}
}

func TestOllamaProviderCheckModelHealthConnectionError(t *testing.T) {
	// Create a server that immediately closes to simulate connection error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return error
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	server.Close() // Close immediately to force connection error

	provider, _ := NewOllamaProvider(server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "any-model")
	if err != nil {
		t.Fatalf("CheckModelHealth should not return error, got: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected unhealthy status for connection error")
	}

	if len(health.Suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions, got %d", len(health.Suggestions))
	}

	if health.Details["ollama_url"] != server.URL {
		t.Errorf("expected ollama_url in details")
	}

	if _, ok := health.Details["error"]; !ok {
		t.Error("expected error in details")
	}
}
