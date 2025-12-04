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
	modelList := openAIModelList{
		Data: []struct {
			ID string `json:"id"`
		}{
			{ID: "gpt-4o"},
			{ID: "gpt-4o-mini"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer openai-key" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if err := json.NewEncoder(w).Encode(modelList); err != nil {
				t.Fatalf("encode models: %v", err)
			}
		case "/v1/chat/completions":
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer openai-key" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			resp := openAIChatResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   "gpt-4o",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Index: 0,
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    "assistant",
							Content: "hello from openai",
						},
						FinishReason: "stop",
					},
				},
				Usage: struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				}{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			}
			json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("openai-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewOpenAIProvider error: %v", err)
	}

	ctx := context.Background()

	models, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("Models error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	ok, err := provider.SupportsModel(ctx, "gpt-4o")
	if err != nil {
		t.Fatalf("SupportsModel error: %v", err)
	}
	if !ok {
		t.Fatal("expected support for gpt-4o")
	}

	meta, err := provider.ModelMetadata(ctx, "gpt-4o")
	if err != nil {
		t.Fatalf("ModelMetadata error: %v", err)
	}
	if meta.Tier != AgentTierBalanced {
		t.Fatalf("expected balanced tier for gpt-4o, got %v", meta.Tier)
	}

	resolved, err := provider.ResolveModel(ctx, "gpt-4o")
	if err != nil {
		t.Fatalf("ResolveModel error: %v", err)
	}
	if resolved.Provider.Name != "openai-live" {
		t.Fatalf("unexpected provider: %s", resolved.Provider.Name)
	}

	resp, err := provider.Generate(ctx, "gpt-4o", "hello", false, nil)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp != "hello from openai" {
		t.Fatalf("unexpected response: %s", resp)
	}
}

func TestOpenAIProviderModelMetadataTiers(t *testing.T) {
	modelList := openAIModelList{
		Data: []struct {
			ID string `json:"id"`
		}{
			{ID: "gpt-4o"},
			{ID: "gpt-4o-mini"},
			{ID: "gpt-4-turbo"},
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

	tests := []struct {
		model        string
		expectedTier AgentTier
	}{
		{"gpt-4o", AgentTierBalanced},
		{"gpt-4o-mini", AgentTierFast},
		{"gpt-4-turbo", AgentTierPowerful},
		{"gpt-4", AgentTierPowerful},
		{"gpt-3.5-turbo", AgentTierFast},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			meta, err := provider.ModelMetadata(ctx, tt.model)
			if err != nil {
				t.Fatalf("ModelMetadata error for %s: %v", tt.model, err)
			}
			if meta.Tier != tt.expectedTier {
				t.Errorf("expected tier %v for %s, got %v", tt.expectedTier, tt.model, meta.Tier)
			}
		})
	}
}

func TestOpenAIProviderModelMetadataNotFound(t *testing.T) {
	provider, _ := NewOpenAIProvider("test-key", "", nil)
	ctx := context.Background()

	_, err := provider.ModelMetadata(ctx, "nonexistent-model")
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestOpenAIProviderIsModelAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(openAIModelList{
				Data: []struct {
					ID string `json:"id"`
				}{
					{ID: "gpt-4o"},
				},
			})
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	available, err := provider.IsModelAvailable(ctx, "gpt-4o")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if !available {
		t.Fatal("expected gpt-4o to be available")
	}

	available, err = provider.IsModelAvailable(ctx, "unknown-model")
	if err != nil {
		t.Fatalf("IsModelAvailable returned error: %v", err)
	}
	if available {
		t.Fatal("expected unknown model to be unavailable")
	}
}

func TestOpenAIProviderResolveModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			json.NewEncoder(w).Encode(openAIModelList{
				Data: []struct {
					ID string `json:"id"`
				}{},
			})
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
	provider := &OpenAIProvider{
		apiKey:      "",
		baseURL:     "https://api.openai.com",
		client:      http.DefaultClient,
		info:        ProviderInfo{Name: "openai-live", Type: ProviderTypeCloud},
		modelsCache: make(map[string]struct{}),
		cacheTTL:    2 * time.Minute,
	}
	ctx := context.Background()

	health, err := provider.CheckModelHealth(ctx, "gpt-4o")
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
	modelList := openAIModelList{
		Data: []struct {
			ID string `json:"id"`
		}{
			{ID: "gpt-4o"},
			{ID: "gpt-4o-mini"},
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

	health, err := provider.CheckModelHealth(ctx, "gpt-4o")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if !health.Healthy {
		t.Fatal("expected model to be healthy")
	}

	if health.Message != "Model 'gpt-4o' is available" {
		t.Errorf("unexpected message: %s", health.Message)
	}

	if health.Suggestions != nil {
		t.Error("expected no suggestions for healthy model")
	}

	if health.Details["model_id"] != "gpt-4o" {
		t.Errorf("expected model_id in details")
	}

	if health.Details["provider"] != "openai" {
		t.Errorf("expected provider in details")
	}
}

func TestOpenAIProviderCheckModelHealthModelNotFound(t *testing.T) {
	modelList := openAIModelList{
		Data: []struct {
			ID string `json:"id"`
		}{
			{ID: "gpt-4o"},
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

	health, err := provider.CheckModelHealth(ctx, "gpt-unknown-model")
	if err != nil {
		t.Fatalf("CheckModelHealth returned error: %v", err)
	}

	if health.Healthy {
		t.Fatal("expected model to be unhealthy")
	}

	if health.Message != "Model 'gpt-unknown-model' not found in OpenAI catalog" {
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

func TestOpenAIProviderGenerateWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			json.NewEncoder(w).Encode(openAIModelList{
				Data: []struct {
					ID string `json:"id"`
				}{
					{ID: "gpt-4o"},
				},
			})
		case "/v1/chat/completions":
			var req openAIChatRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}

			// Verify options were passed
			if req.MaxTokens != 100 {
				t.Errorf("expected max_tokens 100, got %d", req.MaxTokens)
			}
			if req.Temperature != 0.7 {
				t.Errorf("expected temperature 0.7, got %f", req.Temperature)
			}

			resp := openAIChatResponse{
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Message: struct {
							Role    string `json:"role"`
							Content string `json:"content"`
						}{
							Role:    "assistant",
							Content: "response with options",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	opts := map[string]interface{}{
		"max_tokens":  100,
		"temperature": 0.7,
	}

	resp, err := provider.Generate(ctx, "gpt-4o", "hello", false, opts)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp != "response with options" {
		t.Fatalf("unexpected response: %s", resp)
	}
}

func TestOpenAIProviderGenerateStreamingNotSupported(t *testing.T) {
	provider, _ := NewOpenAIProvider("test-key", "", nil)
	ctx := context.Background()

	_, err := provider.Generate(ctx, "gpt-4o", "hello", true, nil)
	if err == nil {
		t.Fatal("expected error for streaming")
	}
	if err.Error() != "openai streaming not implemented" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderInfo(t *testing.T) {
	provider, _ := NewOpenAIProvider("test-key", "", nil)

	info := provider.Info()
	if info.Name != "openai-live" {
		t.Errorf("expected name 'openai-live', got %s", info.Name)
	}
	if info.Type != ProviderTypeCloud {
		t.Errorf("expected cloud provider type, got %v", info.Type)
	}
}

func TestNewOpenAIProviderValidation(t *testing.T) {
	// Empty API key should fail
	_, err := NewOpenAIProvider("", "", nil)
	if err == nil {
		t.Fatal("expected error for empty API key")
	}

	// Valid API key should succeed
	provider, err := NewOpenAIProvider("valid-key", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.baseURL != "https://api.openai.com" {
		t.Errorf("expected default base URL, got %s", provider.baseURL)
	}

	// Custom base URL should be used
	provider, err = NewOpenAIProvider("valid-key", "https://custom.api.com/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.baseURL != "https://custom.api.com" {
		t.Errorf("expected custom base URL without trailing slash, got %s", provider.baseURL)
	}
}

func TestOpenAIProviderGenerateEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			json.NewEncoder(w).Encode(openAIModelList{
				Data: []struct {
					ID string `json:"id"`
				}{
					{ID: "gpt-4o"},
				},
			})
		case "/v1/chat/completions":
			// Return empty choices
			resp := openAIChatResponse{
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.Generate(ctx, "gpt-4o", "hello", false, nil)
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
	if err.Error() != "openai response contained no choices" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderGenerateAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			json.NewEncoder(w).Encode(openAIModelList{
				Data: []struct {
					ID string `json:"id"`
				}{
					{ID: "gpt-4o"},
				},
			})
		case "/v1/chat/completions":
			http.Error(w, `{"error": {"message": "rate limit exceeded"}}`, http.StatusTooManyRequests)
		}
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider("test-key", server.URL, server.Client())
	ctx := context.Background()

	_, err := provider.Generate(ctx, "gpt-4o", "hello", false, nil)
	if err == nil {
		t.Fatal("expected error for API error")
	}
}
