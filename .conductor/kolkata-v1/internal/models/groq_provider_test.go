package models

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewGroqProvider(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		envKey    string
		wantErr   bool
		errString string
	}{
		{
			name:    "valid api key provided",
			apiKey:  "test-key",
			wantErr: false,
		},
		{
			name:    "api key from environment",
			apiKey:  "",
			envKey:  "env-test-key",
			wantErr: false,
		},
		{
			name:      "no api key",
			apiKey:    "",
			wantErr:   true,
			errString: "groq api key is required (set GROQ_API_KEY environment variable)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variable if needed
			if tt.envKey != "" {
				os.Setenv("GROQ_API_KEY", tt.envKey)
				defer os.Unsetenv("GROQ_API_KEY")
			}

			provider, err := NewGroqProvider(tt.apiKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewGroqProvider() expected error containing %q, got nil", tt.errString)
					return
				}
				if tt.errString != "" && err.Error() != tt.errString {
					t.Errorf("NewGroqProvider() error = %v, want %v", err, tt.errString)
				}
				return
			}

			if err != nil {
				t.Errorf("NewGroqProvider() unexpected error = %v", err)
				return
			}

			if provider == nil {
				t.Error("NewGroqProvider() returned nil provider")
				return
			}

			// Verify provider configuration
			if provider.Info().Name != "groq" {
				t.Errorf("provider name = %v, want %v", provider.Info().Name, "groq")
			}
			if provider.Info().Type != ProviderTypeCloud {
				t.Errorf("provider type = %v, want %v", provider.Info().Type, ProviderTypeCloud)
			}
			if provider.baseURL != "https://api.groq.com/openai/v1" {
				t.Errorf("base URL = %v, want %v", provider.baseURL, "https://api.groq.com/openai/v1")
			}
		})
	}
}

func TestGroqProvider_Info(t *testing.T) {
	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}

	info := provider.Info()
	if info.Name != "groq" {
		t.Errorf("Info().Name = %v, want %v", info.Name, "groq")
	}
	if info.Type != ProviderTypeCloud {
		t.Errorf("Info().Type = %v, want %v", info.Type, ProviderTypeCloud)
	}
}

func TestGroqProvider_Models(t *testing.T) {
	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}

	ctx := context.Background()
	models, err := provider.Models(ctx)
	if err != nil {
		t.Errorf("Models() error = %v", err)
		return
	}

	// Verify we get the expected number of models
	expectedCount := len(groqModels)
	if len(models) != expectedCount {
		t.Errorf("Models() returned %d models, want %d", len(models), expectedCount)
	}

	// Verify known models are present
	modelMap := make(map[string]ModelRef)
	for _, model := range models {
		modelMap[model.Name] = model
	}

	expectedModels := []string{
		"llama-3.3-70b-versatile",
		"llama-3.1-70b-versatile",
		"llama-3.1-8b-instant",
		"mixtral-8x7b-32768",
		"gemma2-9b-it",
	}

	for _, name := range expectedModels {
		if _, ok := modelMap[name]; !ok {
			t.Errorf("Models() missing expected model: %s", name)
		}
	}
}

func TestGroqProvider_SupportsModel(t *testing.T) {
	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name      string
		model     string
		supported bool
	}{
		{
			name:      "supported model - llama 3.3",
			model:     "llama-3.3-70b-versatile",
			supported: true,
		},
		{
			name:      "supported model - llama 3.1",
			model:     "llama-3.1-70b-versatile",
			supported: true,
		},
		{
			name:      "supported model - mixtral",
			model:     "mixtral-8x7b-32768",
			supported: true,
		},
		{
			name:      "unsupported model",
			model:     "gpt-4",
			supported: false,
		},
		{
			name:      "unknown model",
			model:     "unknown-model",
			supported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supported, err := provider.SupportsModel(ctx, tt.model)
			if err != nil {
				t.Errorf("SupportsModel() error = %v", err)
				return
			}
			if supported != tt.supported {
				t.Errorf("SupportsModel(%s) = %v, want %v", tt.model, supported, tt.supported)
			}
		})
	}
}

func TestGroqProvider_ModelMetadata(t *testing.T) {
	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{
			name:    "valid model - llama 3.3",
			model:   "llama-3.3-70b-versatile",
			wantErr: false,
		},
		{
			name:    "valid model - gemma2",
			model:   "gemma2-9b-it",
			wantErr: false,
		},
		{
			name:    "invalid model",
			model:   "unknown-model",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := provider.ModelMetadata(ctx, tt.model)

			if tt.wantErr {
				if err == nil {
					t.Error("ModelMetadata() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ModelMetadata() error = %v", err)
				return
			}

			// All Groq models should be AgentTierFast
			if metadata.Tier != AgentTierFast {
				t.Errorf("ModelMetadata().Tier = %v, want %v", metadata.Tier, AgentTierFast)
			}

			// All Groq models should have zero or low cost
			if metadata.CostPer1KTokens != 0 {
				t.Errorf("ModelMetadata().CostPer1KTokens = %v, want 0", metadata.CostPer1KTokens)
			}

			// Description should be set
			if metadata.Description == "" {
				t.Error("ModelMetadata().Description is empty")
			}
		})
	}
}

func TestGroqProvider_ResolveModel(t *testing.T) {
	// Create a test server to mock API validation
	validationCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validationCalled = true
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}
	provider.baseURL = server.URL

	ctx := context.Background()

	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{
			name:    "valid model",
			model:   "llama-3.3-70b-versatile",
			wantErr: false,
		},
		{
			name:    "invalid model",
			model:   "unknown-model",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationCalled = false
			resolved, err := provider.ResolveModel(ctx, tt.model)

			if tt.wantErr {
				if err == nil {
					t.Error("ResolveModel() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ResolveModel() error = %v", err)
				return
			}

			// Verify API validation was called
			if !validationCalled {
				t.Error("ResolveModel() did not validate API key")
			}

			// Verify resolved model properties
			if resolved.Name != tt.model {
				t.Errorf("ResolvedModel.Name = %v, want %v", resolved.Name, tt.model)
			}
			if resolved.Provider.Name != "groq" {
				t.Errorf("ResolvedModel.Provider.Name = %v, want groq", resolved.Provider.Name)
			}
			if resolved.Tier != AgentTierFast {
				t.Errorf("ResolvedModel.Tier = %v, want %v", resolved.Tier, AgentTierFast)
			}
			if resolved.CostPer1KTokens != 0 {
				t.Errorf("ResolvedModel.CostPer1KTokens = %v, want 0", resolved.CostPer1KTokens)
			}
			if resolved.Route == "" {
				t.Error("ResolvedModel.Route is empty")
			}
		})
	}
}

func TestGroqProvider_IsModelAvailable(t *testing.T) {
	// Create a test server to mock API validation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}
	provider.baseURL = server.URL

	ctx := context.Background()

	tests := []struct {
		name      string
		model     string
		available bool
	}{
		{
			name:      "supported model",
			model:     "llama-3.3-70b-versatile",
			available: true,
		},
		{
			name:      "unsupported model",
			model:     "gpt-4",
			available: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			available, err := provider.IsModelAvailable(ctx, tt.model)
			if err != nil {
				t.Errorf("IsModelAvailable() error = %v", err)
				return
			}
			if available != tt.available {
				t.Errorf("IsModelAvailable(%s) = %v, want %v", tt.model, available, tt.available)
			}
		})
	}
}

func TestGroqProvider_Generate(t *testing.T) {
	// Create a test server to mock the Groq API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid api key",
			})
			return
		}

		// Parse request
		var req groqChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Return mock response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(groqChatResponse{
			Choices: []groqChatChoice{
				{
					Message: groqChatMessage{
						Role:    "assistant",
						Content: "This is a test response",
					},
					FinishReason: "stop",
				},
			},
		})
	}))
	defer server.Close()

	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}
	provider.baseURL = server.URL

	ctx := context.Background()

	tests := []struct {
		name    string
		model   string
		prompt  string
		opts    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "successful generation",
			model:   "llama-3.3-70b-versatile",
			prompt:  "Hello, world!",
			wantErr: false,
		},
		{
			name:   "with options",
			model:  "llama-3.3-70b-versatile",
			prompt: "Test prompt",
			opts: map[string]interface{}{
				"temperature": 0.7,
				"max_tokens":  100,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := provider.Generate(ctx, tt.model, tt.prompt, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("Generate() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Generate() error = %v", err)
				return
			}

			if response == "" {
				t.Error("Generate() returned empty response")
			}

			if response != "This is a test response" {
				t.Errorf("Generate() response = %v, want 'This is a test response'", response)
			}
		})
	}
}

func TestGroqProvider_ValidateAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		errString  string
	}{
		{
			name:       "valid api key",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "invalid api key",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
			errString:  "invalid groq api key",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"data": []map[string]string{},
					})
				}
			}))
			defer server.Close()

			provider, err := NewGroqProvider("test-key")
			if err != nil {
				t.Fatalf("NewGroqProvider() error = %v", err)
			}
			provider.baseURL = server.URL

			ctx := context.Background()
			err = provider.validateAPIKey(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("validateAPIKey() expected error, got nil")
					return
				}
				if tt.errString != "" && err.Error() != tt.errString {
					t.Errorf("validateAPIKey() error = %v, want %v", err.Error(), tt.errString)
				}
				return
			}

			if err != nil {
				t.Errorf("validateAPIKey() unexpected error = %v", err)
			}

			// Verify caching - second call should not hit the server
			callCount := 0
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				w.WriteHeader(http.StatusOK)
			})

			err = provider.validateAPIKey(ctx)
			if err != nil {
				t.Errorf("validateAPIKey() second call error = %v", err)
			}
			if callCount > 0 {
				t.Error("validateAPIKey() should use cached result")
			}
		})
	}
}

func TestGroqProvider_ClientTimeout(t *testing.T) {
	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}

	if provider.client.Timeout != 30*time.Second {
		t.Errorf("client timeout = %v, want 30s", provider.client.Timeout)
	}
}

func TestGroqProvider_ContextWindow(t *testing.T) {
	expectedWindows := map[string]int{
		"llama-3.3-70b-versatile": 128000,
		"llama-3.1-70b-versatile": 128000,
		"llama-3.1-8b-instant":    128000,
		"mixtral-8x7b-32768":      32768,
		"gemma2-9b-it":            8192,
	}

	provider, err := NewGroqProvider("test-key")
	if err != nil {
		t.Fatalf("NewGroqProvider() error = %v", err)
	}

	// Mock server for validation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]string{},
		})
	}))
	defer server.Close()
	provider.baseURL = server.URL

	ctx := context.Background()

	for model, expectedWindow := range expectedWindows {
		t.Run(model, func(t *testing.T) {
			resolved, err := provider.ResolveModel(ctx, model)
			if err != nil {
				t.Errorf("ResolveModel() error = %v", err)
				return
			}

			windowStr := resolved.Metadata["context_window"]
			if windowStr != fmt.Sprintf("%d", expectedWindow) {
				t.Errorf("context_window = %v, want %d", windowStr, expectedWindow)
			}
		})
	}
}
