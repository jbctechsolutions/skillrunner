package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenRouterProviderBasicFlow(t *testing.T) {
	modelList := openRouterModelsResponse{
		Data: []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
			Pricing       struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		}{
			{
				ID:            "anthropic/claude-3.5-sonnet",
				Name:          "Claude 3.5 Sonnet",
				Description:   "Anthropic's Claude 3.5 Sonnet",
				ContextLength: 200000,
				Pricing: struct {
					Prompt     string `json:"prompt"`
					Completion string `json:"completion"`
				}{
					Prompt:     "0.000003",
					Completion: "0.000015",
				},
			},
			{
				ID:            "openai/gpt-4-turbo",
				Name:          "GPT-4 Turbo",
				Description:   "OpenAI's GPT-4 Turbo",
				ContextLength: 128000,
				Pricing: struct {
					Prompt     string `json:"prompt"`
					Completion string `json:"completion"`
				}{
					Prompt:     "0.00001",
					Completion: "0.00003",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			if r.Header.Get("Authorization") != "Bearer test-key" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if err := json.NewEncoder(w).Encode(modelList); err != nil {
				t.Fatalf("encode models: %v", err)
			}
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := NewOpenRouterProvider("test-key", server.URL)
	if err != nil {
		t.Fatalf("NewOpenRouterProvider error: %v", err)
	}
	provider.client = server.Client()

	ctx := context.Background()

	// Test Models()
	models, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("Models error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	// Test SupportsModel()
	ok, err := provider.SupportsModel(ctx, "anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("SupportsModel error: %v", err)
	}
	if !ok {
		t.Fatal("expected support for anthropic/claude-3.5-sonnet")
	}

	// Test IsModelAvailable()
	available, err := provider.IsModelAvailable(ctx, "openai/gpt-4-turbo")
	if err != nil {
		t.Fatalf("IsModelAvailable error: %v", err)
	}
	if !available {
		t.Fatal("expected gpt-4-turbo to be available")
	}

	// Test ModelMetadata()
	meta, err := provider.ModelMetadata(ctx, "anthropic/claude-3.5-sonnet")
	if err != nil {
		t.Fatalf("ModelMetadata error: %v", err)
	}
	if meta.Tier != AgentTierPowerful {
		t.Fatalf("expected powerful tier for Claude Sonnet, got %v", meta.Tier)
	}
	// Cost should be average of (3 + 15) / 2 = 9 per million tokens = 0.009 per 1K tokens
	expectedCost := (3.0 + 15.0) / 2000.0
	if meta.CostPer1KTokens != expectedCost {
		t.Fatalf("expected cost %f, got %f", expectedCost, meta.CostPer1KTokens)
	}

	// Test ResolveModel()
	resolved, err := provider.ResolveModel(ctx, "openai/gpt-4-turbo")
	if err != nil {
		t.Fatalf("ResolveModel error: %v", err)
	}
	if resolved.Provider.Name != "openrouter" {
		t.Fatalf("unexpected provider: %s", resolved.Provider.Name)
	}
	if resolved.Provider.Type != ProviderTypeCloud {
		t.Fatalf("expected cloud provider type, got %s", resolved.Provider.Type)
	}
	if resolved.Metadata["context_length"] != "128000" {
		t.Fatalf("unexpected context length: %s", resolved.Metadata["context_length"])
	}
}

func TestOpenRouterProviderCaching(t *testing.T) {
	requestCount := 0
	modelList := openRouterModelsResponse{
		Data: []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
			Pricing       struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		}{
			{
				ID:            "test/model",
				Name:          "Test Model",
				Description:   "A test model",
				ContextLength: 4096,
				Pricing: struct {
					Prompt     string `json:"prompt"`
					Completion string `json:"completion"`
				}{
					Prompt:     "0.000001",
					Completion: "0.000002",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			requestCount++
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewOpenRouterProvider("test-key", server.URL)
	provider.client = server.Client()
	ctx := context.Background()

	// First call - should hit API
	_, err := provider.Models(ctx)
	if err != nil {
		t.Fatalf("first Models call error: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected 1 request, got %d", requestCount)
	}

	// Second call within TTL - should use cache
	_, err = provider.Models(ctx)
	if err != nil {
		t.Fatalf("second Models call error: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected still 1 request (cached), got %d", requestCount)
	}

	// Third call - still cached
	_, err = provider.SupportsModel(ctx, "test/model")
	if err != nil {
		t.Fatalf("SupportsModel call error: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected still 1 request (cached), got %d", requestCount)
	}
}

func TestOpenRouterProviderCacheExpiry(t *testing.T) {
	requestCount := 0
	modelList := openRouterModelsResponse{
		Data: []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
			Pricing       struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		}{
			{
				ID:   "test/model",
				Name: "Test Model",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			requestCount++
			json.NewEncoder(w).Encode(modelList)
		}
	}))
	defer server.Close()

	provider, _ := NewOpenRouterProvider("test-key", server.URL)
	provider.client = server.Client()
	provider.cacheTTL = 50 * time.Millisecond // Short TTL for testing
	ctx := context.Background()

	// First call
	_, _ = provider.Models(ctx)
	firstCount := requestCount

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second call - should hit API again
	_, _ = provider.Models(ctx)
	if requestCount <= firstCount {
		t.Fatalf("expected cache to expire and new request to be made, got %d requests", requestCount)
	}
}

func TestOpenRouterProviderModelTiers(t *testing.T) {
	tests := []struct {
		modelID      string
		expectedTier AgentTier
	}{
		{"openai/gpt-4-turbo", AgentTierPowerful},
		{"anthropic/claude-3-opus", AgentTierPowerful},
		{"anthropic/claude-3-5-sonnet", AgentTierPowerful},
		{"google/gemini-pro", AgentTierPowerful},
		{"openai/gpt-3.5-turbo", AgentTierFast},
		{"anthropic/claude-3-haiku", AgentTierFast},
		{"meta/llama-3.1-8b", AgentTierFast},
		{"google/gemini-flash", AgentTierFast},
		{"mistralai/mistral-medium", AgentTierBalanced},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			modelList := openRouterModelsResponse{
				Data: []struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					Description   string `json:"description"`
					ContextLength int    `json:"context_length"`
					Pricing       struct {
						Prompt     string `json:"prompt"`
						Completion string `json:"completion"`
					} `json:"pricing"`
				}{
					{
						ID:   tt.modelID,
						Name: tt.modelID,
						Pricing: struct {
							Prompt     string `json:"prompt"`
							Completion string `json:"completion"`
						}{
							Prompt:     "0.000001",
							Completion: "0.000002",
						},
					},
				},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(modelList)
			}))
			defer server.Close()

			provider, _ := NewOpenRouterProvider("test-key", server.URL)
			provider.client = server.Client()

			meta, err := provider.ModelMetadata(context.Background(), tt.modelID)
			if err != nil {
				t.Fatalf("ModelMetadata error: %v", err)
			}
			if meta.Tier != tt.expectedTier {
				t.Fatalf("expected tier %v, got %v", tt.expectedTier, meta.Tier)
			}
		})
	}
}

func TestOpenRouterProviderErrors(t *testing.T) {
	t.Run("empty API key", func(t *testing.T) {
		_, err := NewOpenRouterProvider("", "https://example.com")
		if err == nil {
			t.Fatal("expected error for empty API key")
		}
	})

	t.Run("network error", func(t *testing.T) {
		provider, _ := NewOpenRouterProvider("test-key", "http://localhost:1")
		provider.client = &http.Client{Timeout: 10 * time.Millisecond}

		_, err := provider.Models(context.Background())
		if err == nil {
			t.Fatal("expected error for network failure")
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}))
		defer server.Close()

		provider, _ := NewOpenRouterProvider("bad-key", server.URL)
		provider.client = server.Client()

		_, err := provider.Models(context.Background())
		if err == nil {
			t.Fatal("expected error for unauthorized request")
		}
	})

	t.Run("model not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(openRouterModelsResponse{Data: []struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				Description   string `json:"description"`
				ContextLength int    `json:"context_length"`
				Pricing       struct {
					Prompt     string `json:"prompt"`
					Completion string `json:"completion"`
				} `json:"pricing"`
			}{}})
		}))
		defer server.Close()

		provider, _ := NewOpenRouterProvider("test-key", server.URL)
		provider.client = server.Client()

		_, err := provider.ModelMetadata(context.Background(), "nonexistent/model")
		if err == nil {
			t.Fatal("expected error for nonexistent model")
		}

		_, err = provider.ResolveModel(context.Background(), "nonexistent/model")
		if err == nil {
			t.Fatal("expected error for nonexistent model")
		}
	})
}

func TestOpenRouterProviderParsePricing(t *testing.T) {
	tests := []struct {
		name            string
		promptPrice     string
		completionPrice string
		expectedCost    float64 // per 1K tokens
	}{
		{
			name:            "typical pricing",
			promptPrice:     "0.000003", // $3 per million
			completionPrice: "0.000015", // $15 per million
			expectedCost:    0.009,      // avg = (3 + 15) / 2000
		},
		{
			name:            "free model",
			promptPrice:     "0",
			completionPrice: "0",
			expectedCost:    0,
		},
		{
			name:            "expensive model",
			promptPrice:     "0.00006", // $60 per million
			completionPrice: "0.00012", // $120 per million
			expectedCost:    0.09,      // avg = (60 + 120) / 2000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelList := openRouterModelsResponse{
				Data: []struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					Description   string `json:"description"`
					ContextLength int    `json:"context_length"`
					Pricing       struct {
						Prompt     string `json:"prompt"`
						Completion string `json:"completion"`
					} `json:"pricing"`
				}{
					{
						ID:   "test/model",
						Name: "Test Model",
						Pricing: struct {
							Prompt     string `json:"prompt"`
							Completion string `json:"completion"`
						}{
							Prompt:     tt.promptPrice,
							Completion: tt.completionPrice,
						},
					},
				},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(modelList)
			}))
			defer server.Close()

			provider, _ := NewOpenRouterProvider("test-key", server.URL)
			provider.client = server.Client()

			meta, err := provider.ModelMetadata(context.Background(), "test/model")
			if err != nil {
				t.Fatalf("ModelMetadata error: %v", err)
			}

			// Allow small floating point differences
			if diff := meta.CostPer1KTokens - tt.expectedCost; diff < -0.0001 || diff > 0.0001 {
				t.Fatalf("expected cost %f, got %f", tt.expectedCost, meta.CostPer1KTokens)
			}
		})
	}
}

func TestOpenRouterProviderDefaultBaseURL(t *testing.T) {
	provider, err := NewOpenRouterProvider("test-key", "")
	if err != nil {
		t.Fatalf("NewOpenRouterProvider error: %v", err)
	}
	if provider.baseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("expected default base URL, got %s", provider.baseURL)
	}
}

func TestOpenRouterProviderInfo(t *testing.T) {
	provider, _ := NewOpenRouterProvider("test-key", "")
	info := provider.Info()

	if info.Name != "openrouter" {
		t.Fatalf("expected provider name 'openrouter', got '%s'", info.Name)
	}
	if info.Type != ProviderTypeCloud {
		t.Fatalf("expected cloud provider type, got %s", info.Type)
	}
}
