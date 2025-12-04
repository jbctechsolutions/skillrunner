package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGroqProviderBasicFlow(t *testing.T) {
	modelList := groqModelListResponse{
		Object: "list",
		Data: []struct {
			ID            string `json:"id"`
			Object        string `json:"object"`
			Created       int64  `json:"created"`
			OwnedBy       string `json:"owned_by"`
			Active        bool   `json:"active"`
			ContextWindow int    `json:"context_window"`
		}{
			{ID: "llama-3.1-70b-versatile", Active: true, ContextWindow: 131072},
			{ID: "llama-3.1-8b-instant", Active: true, ContextWindow: 131072},
			{ID: "mixtral-8x7b-32768", Active: true, ContextWindow: 32768},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer groq-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/models":
			if err := json.NewEncoder(w).Encode(modelList); err != nil {
				t.Fatalf("encode models: %v", err)
			}
		case "/v1/chat/completions":
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello from groq"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := NewGroqProvider("groq-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewGroqProvider error: %v", err)
	}

	ctx := context.Background()

	models, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("Models error: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}

	ok, err := provider.SupportsModel(ctx, "llama-3.1-70b-versatile")
	if err != nil {
		t.Fatalf("SupportsModel error: %v", err)
	}
	if !ok {
		t.Fatal("expected support for llama-3.1-70b-versatile")
	}

	meta, err := provider.ModelMetadata(ctx, "llama-3.1-70b-versatile")
	if err != nil {
		t.Fatalf("ModelMetadata error: %v", err)
	}
	if meta.Tier != AgentTierPowerful {
		t.Fatalf("expected powerful tier for 70b model, got %v", meta.Tier)
	}

	resolved, err := provider.ResolveModel(ctx, "llama-3.1-70b-versatile")
	if err != nil {
		t.Fatalf("ResolveModel error: %v", err)
	}
	if resolved.Provider.Name != "groq" {
		t.Fatalf("unexpected provider: %s", resolved.Provider.Name)
	}

	output, err := provider.Generate(ctx, "llama-3.1-70b-versatile", "hello", false, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	// Collect output from channel
	var response string
	for chunk := range output {
		response += chunk
	}
	if response != "hello from groq" {
		t.Fatalf("unexpected response: %s", response)
	}
}

func TestGroqProviderModelMetadataTiers(t *testing.T) {
	modelList := groqModelListResponse{
		Object: "list",
		Data: []struct {
			ID            string `json:"id"`
			Object        string `json:"object"`
			Created       int64  `json:"created"`
			OwnedBy       string `json:"owned_by"`
			Active        bool   `json:"active"`
			ContextWindow int    `json:"context_window"`
		}{
			{ID: "llama-3.1-70b-versatile", Active: true, ContextWindow: 131072},
			{ID: "llama-3.1-8b-instant", Active: true, ContextWindow: 131072},
			{ID: "mixtral-8x7b-32768", Active: true, ContextWindow: 32768},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	// Test 70b model -> Powerful tier
	meta, err := provider.ModelMetadata(ctx, "llama-3.1-70b-versatile")
	if err != nil {
		t.Fatalf("ModelMetadata error: %v", err)
	}
	if meta.Tier != AgentTierPowerful {
		t.Errorf("expected Powerful tier for 70b model, got %v", meta.Tier)
	}

	// Test 8b model -> Fast tier
	meta, err = provider.ModelMetadata(ctx, "llama-3.1-8b-instant")
	if err != nil {
		t.Fatalf("ModelMetadata error: %v", err)
	}
	if meta.Tier != AgentTierFast {
		t.Errorf("expected Fast tier for 8b model, got %v", meta.Tier)
	}

	// Test mixtral model -> Balanced tier
	meta, err = provider.ModelMetadata(ctx, "mixtral-8x7b-32768")
	if err != nil {
		t.Fatalf("ModelMetadata error: %v", err)
	}
	if meta.Tier != AgentTierBalanced {
		t.Errorf("expected Balanced tier for mixtral model, got %v", meta.Tier)
	}
}

func TestGroqProviderModelMetadataNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(groqModelListResponse{Object: "list", Data: nil})
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.ModelMetadata(ctx, "nonexistent-model")
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestGroqProviderIsModelAvailable(t *testing.T) {
	modelList := groqModelListResponse{
		Object: "list",
		Data: []struct {
			ID            string `json:"id"`
			Object        string `json:"object"`
			Created       int64  `json:"created"`
			OwnedBy       string `json:"owned_by"`
			Active        bool   `json:"active"`
			ContextWindow int    `json:"context_window"`
		}{
			{ID: "llama-3.1-8b-instant", Active: true, ContextWindow: 131072},
			{ID: "inactive-model", Active: false, ContextWindow: 4096},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	// Active model should be available
	available, err := provider.IsModelAvailable(ctx, "llama-3.1-8b-instant")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if !available {
		t.Fatal("expected llama-3.1-8b-instant to be available")
	}

	// Inactive model should not be available
	available, err = provider.IsModelAvailable(ctx, "inactive-model")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if available {
		t.Fatal("expected inactive-model to be unavailable")
	}

	// Unknown model should not be available
	available, err = provider.IsModelAvailable(ctx, "unknown-model")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if available {
		t.Fatal("expected unknown model to be unavailable")
	}
}

func TestGroqProviderResolveModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(groqModelListResponse{Object: "list", Data: nil})
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.ResolveModel(ctx, "missing")
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestGroqProviderCheckModelHealthNoAPIKey(t *testing.T) {
	provider := &GroqProvider{
		apiKey:      "",
		baseURL:     "https://api.groq.com/openai",
		client:      http.DefaultClient,
		info:        ProviderInfo{Name: "groq", Type: ProviderTypeCloud},
		modelsCache: make(map[string]groqModelInfo),
		cacheTTL:    5 * time.Minute,
	}
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "llama-3.1-8b-instant")
	if err != nil {
		t.Fatalf("CheckModelHealth should not return error, got: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected unhealthy status when API key is missing")
	}

	if health.Message != "Groq API key not configured" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions, got %d", len(health.Suggestions))
	}

	expectedSuggestion := "Set GROQ_API_KEY environment variable"
	if health.Suggestions[0] != expectedSuggestion {
		t.Errorf("expected first suggestion to be %q, got %q", expectedSuggestion, health.Suggestions[0])
	}
}

func TestGroqProviderCheckModelHealthModelFound(t *testing.T) {
	modelList := groqModelListResponse{
		Object: "list",
		Data: []struct {
			ID            string `json:"id"`
			Object        string `json:"object"`
			Created       int64  `json:"created"`
			OwnedBy       string `json:"owned_by"`
			Active        bool   `json:"active"`
			ContextWindow int    `json:"context_window"`
		}{
			{ID: "llama-3.1-70b-versatile", Active: true, ContextWindow: 131072},
			{ID: "llama-3.1-8b-instant", Active: true, ContextWindow: 131072},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "llama-3.1-8b-instant")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if !health.Healthy {
		t.Fatal("expected model to be healthy")
	}

	if health.Message != "Model 'llama-3.1-8b-instant' is available" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if health.Suggestions != nil {
		t.Error("expected no suggestions for healthy model")
	}

	if health.Details["model_id"] != "llama-3.1-8b-instant" {
		t.Errorf("expected model_id in details")
	}

	if health.Details["provider"] != "groq" {
		t.Errorf("expected provider in details")
	}
}

func TestGroqProviderCheckModelHealthModelNotFound(t *testing.T) {
	modelList := groqModelListResponse{
		Object: "list",
		Data: []struct {
			ID            string `json:"id"`
			Object        string `json:"object"`
			Created       int64  `json:"created"`
			OwnedBy       string `json:"owned_by"`
			Active        bool   `json:"active"`
			ContextWindow int    `json:"context_window"`
		}{
			{ID: "llama-3.1-70b-versatile", Active: true, ContextWindow: 131072},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "unknown-model")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected model to be unhealthy")
	}

	if health.Message != "Model 'unknown-model' not found in Groq catalog" {
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

func TestGroqProviderCheckModelHealthInactiveModel(t *testing.T) {
	modelList := groqModelListResponse{
		Object: "list",
		Data: []struct {
			ID            string `json:"id"`
			Object        string `json:"object"`
			Created       int64  `json:"created"`
			OwnedBy       string `json:"owned_by"`
			Active        bool   `json:"active"`
			ContextWindow int    `json:"context_window"`
		}{
			{ID: "inactive-model", Active: false, ContextWindow: 4096},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "inactive-model")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected model to be unhealthy")
	}

	if health.Message != "Model 'inactive-model' is not currently active" {
		t.Errorf("unexpected message: %s", health.Message)
	}
}

func TestGroqProviderCheckModelHealthAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			http.Error(w, "API error", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "any-model")
	if err != nil {
		t.Fatalf("CheckModelHealth should not return error, got: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected unhealthy status for API error")
	}

	if health.Message != "Unable to fetch models from Groq API" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if len(health.Suggestions) < 3 {
		t.Errorf("expected at least 3 suggestions, got %d", len(health.Suggestions))
	}

	if _, ok := health.Details["error"]; !ok {
		t.Error("expected error in details")
	}
}

func TestGroqProviderStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			json.NewEncoder(w).Encode(groqModelListResponse{
				Object: "list",
				Data: []struct {
					ID            string `json:"id"`
					Object        string `json:"object"`
					Created       int64  `json:"created"`
					OwnedBy       string `json:"owned_by"`
					Active        bool   `json:"active"`
					ContextWindow int    `json:"context_window"`
				}{
					{ID: "llama-3.1-8b-instant", Active: true, ContextWindow: 131072},
				},
			})
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
			flusher.Flush()
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n"))
			flusher.Flush()
			w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	provider, _ := NewGroqProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	output, err := provider.Generate(ctx, "llama-3.1-8b-instant", "hello", true, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var response string
	for chunk := range output {
		response += chunk
	}

	if response != "Hello world" {
		t.Fatalf("unexpected response: %s", response)
	}
}

func TestNewGroqProviderRequiresAPIKey(t *testing.T) {
	_, err := NewGroqProvider("", "", nil)
	if err == nil {
		t.Fatal("expected error when API key is empty")
	}
	if err.Error() != "groq api key is required" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}
