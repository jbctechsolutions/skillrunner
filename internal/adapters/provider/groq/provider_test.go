package groq

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// newTestServer creates a test HTTP server with the given handler.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Provider) {
	t.Helper()
	server := httptest.NewServer(handler)
	config := Config{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 2,
	}
	return server, NewProvider(config, WithBaseURL(server.URL))
}

func TestProvider_Info(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")
	info := provider.Info()

	if info.Name != "groq" {
		t.Errorf("expected name 'groq', got %q", info.Name)
	}
	if info.IsLocal {
		t.Error("expected IsLocal to be false")
	}
	if !strings.Contains(info.Description, "Groq") {
		t.Errorf("expected description to mention Groq, got %q", info.Description)
	}
	if info.BaseURL == "" {
		t.Error("expected BaseURL to be set")
	}
}

func TestProvider_ListModels(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")
	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Verify known models are present
	expectedModels := []string{
		ModelLlama31_70BVersatile,
		ModelLlama31_8BInstant,
		ModelMixtral8x7B_32768,
	}
	for _, expected := range expectedModels {
		if !slices.Contains(models, expected) {
			t.Errorf("expected model %q not found in list", expected)
		}
	}
}

func TestProvider_SupportsModel(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")
	ctx := context.Background()

	tests := []struct {
		modelID  string
		expected bool
	}{
		{ModelLlama31_70BVersatile, true},
		{ModelMixtral8x7B_32768, true},
		{ModelGemma2_9BIt, true},
		{"unknown-model", false},
		{"gpt-4", false},
		{"claude-3-5-sonnet", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			supported, err := provider.SupportsModel(ctx, tt.modelID)
			if err != nil {
				t.Fatalf("SupportsModel failed: %v", err)
			}
			if supported != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, supported)
			}
		})
	}
}

func TestProvider_IsAvailable(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")
	ctx := context.Background()

	tests := []struct {
		name     string
		modelID  string
		expected bool
	}{
		{"supported model", ModelLlama31_70BVersatile, true},
		{"unsupported model", "unknown-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			available, err := provider.IsAvailable(ctx, tt.modelID)
			if err != nil {
				t.Fatalf("IsAvailable failed: %v", err)
			}
			if available != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, available)
			}
		})
	}
}

func TestProvider_Complete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != EndpointChatCompletions {
			t.Errorf("expected %s, got %s", EndpointChatCompletions, r.URL.Path)
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("missing or incorrect Authorization header: %s", authHeader)
		}

		// Decode request to verify structure
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Send response
		resp := ChatCompletionResponse{
			ID:      "chatcmpl_123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: "Hello! How can I help you today?",
					},
					FinishReason: FinishReasonStop,
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 1024,
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Content != "Hello! How can I help you today?" {
		t.Errorf("unexpected content: %q", resp.Content)
	}
	if resp.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 8 {
		t.Errorf("expected 8 output tokens, got %d", resp.OutputTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish reason 'stop', got %q", resp.FinishReason)
	}
	if resp.ModelUsed != ModelLlama31_70BVersatile {
		t.Errorf("expected model %q, got %q", ModelLlama31_70BVersatile, resp.ModelUsed)
	}
	if resp.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedReq ChatCompletionRequest

	handler := func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := ChatCompletionResponse{
			ID:      "chatcmpl_123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   ModelLlama31_70BVersatile,
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: "I am a helpful assistant.",
					},
					FinishReason: FinishReasonStop,
				},
			},
			Usage: Usage{PromptTokens: 20, CompletionTokens: 5, TotalTokens: 25},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:      ModelLlama31_70BVersatile,
		MaxTokens:    100,
		SystemPrompt: "You are a helpful assistant.",
		Messages: []ports.Message{
			{Role: "user", Content: "Who are you?"},
		},
	}

	_, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Verify system message was included
	if len(receivedReq.Messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(receivedReq.Messages))
	}
	if receivedReq.Messages[0].Role != RoleSystem {
		t.Errorf("expected first message to be system, got %v", receivedReq.Messages[0].Role)
	}
	if receivedReq.Messages[0].Content != "You are a helpful assistant." {
		t.Errorf("system message content incorrect: %q", receivedReq.Messages[0].Content)
	}
}

func TestProvider_Complete_WithTemperature(t *testing.T) {
	var receivedReq ChatCompletionRequest

	handler := func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := ChatCompletionResponse{
			ID:      "chatcmpl_123",
			Object:  "chat.completion",
			Model:   ModelLlama31_70BVersatile,
			Choices: []Choice{{Message: Message{Content: "Response"}, FinishReason: FinishReasonStop}},
			Usage:   Usage{PromptTokens: 5, CompletionTokens: 1},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:     ModelLlama31_70BVersatile,
		MaxTokens:   100,
		Temperature: 0.8,
		Messages:    []ports.Message{{Role: "user", Content: "Test"}},
	}

	_, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if receivedReq.Temperature == nil {
		t.Fatal("expected temperature to be set")
	}
	if *receivedReq.Temperature != 0.8 {
		t.Errorf("expected temperature 0.8, got %f", *receivedReq.Temperature)
	}
}

func TestProvider_Complete_ErrorResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorInfo{
				Type:    "invalid_request_error",
				Message: "max_tokens must be greater than 0",
			},
		})
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 0,
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := provider.Complete(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "max_tokens") {
		t.Errorf("error should mention max_tokens: %v", err)
	}
}

func TestProvider_Complete_RateLimitRetry(t *testing.T) {
	var attempts atomic.Int32

	handler := func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		resp := ChatCompletionResponse{
			ID:      "chatcmpl_123",
			Object:  "chat.completion",
			Model:   ModelLlama31_70BVersatile,
			Choices: []Choice{{Message: Message{Content: "Success after retry"}, FinishReason: FinishReasonStop}},
			Usage:   Usage{PromptTokens: 5, CompletionTokens: 3},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Test"}},
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}

	if resp.Content != "Success after retry" {
		t.Errorf("unexpected content: %q", resp.Content)
	}
}

func TestProvider_Stream(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming is requested
		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("expected stream=true in request")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		// Send SSE events (OpenAI-compatible format)
		events := []string{
			`data: {"id":"chatcmpl_123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl_123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl_123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{"content":" World"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl_123","object":"chat.completion.chunk","created":1234567890,"model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
			`data: [DONE]`,
		}

		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Hello"}},
	}

	var chunks []string
	resp, err := provider.Stream(context.Background(), req, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})

	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}

	expectedContent := "Hello World"
	if resp.Content != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, resp.Content)
	}

	if resp.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.InputTokens)
	}

	if resp.OutputTokens != 5 {
		t.Errorf("expected 5 output tokens, got %d", resp.OutputTokens)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("expected finish reason 'stop', got %q", resp.FinishReason)
	}

	if resp.ModelUsed != ModelLlama31_70BVersatile {
		t.Errorf("expected model %q, got %q", ModelLlama31_70BVersatile, resp.ModelUsed)
	}
}

func TestProvider_Stream_CallbackError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		events := []string{
			`data: {"id":"chatcmpl_123","object":"chat.completion.chunk","model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		}

		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
		}
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Hello"}},
	}

	callbackErr := fmt.Errorf("callback error")
	_, err := provider.Stream(context.Background(), req, func(chunk string) error {
		return callbackErr
	})

	if err != callbackErr {
		t.Errorf("expected callback error, got: %v", err)
	}
}

func TestProvider_Stream_ErrorResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorInfo{
				Type:    "invalid_request_error",
				Message: "invalid model",
			},
		})
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   "invalid-model",
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := provider.Stream(context.Background(), req, func(chunk string) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProvider_HealthCheck(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			ID:      "chatcmpl_health",
			Object:  "chat.completion",
			Model:   ModelLlama31_8BInstant,
			Choices: []Choice{{Message: Message{Content: "Hi"}, FinishReason: FinishReasonStop}},
			Usage:   Usage{PromptTokens: 2, CompletionTokens: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	status, err := provider.HealthCheck(context.Background(), ModelLlama31_8BInstant)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if !status.Healthy {
		t.Errorf("expected healthy=true, got false: %s", status.Message)
	}

	if status.Message != "OK" {
		t.Errorf("expected message 'OK', got %q", status.Message)
	}

	if status.Latency <= 0 {
		t.Error("expected positive latency")
	}

	if status.LastChecked.IsZero() {
		t.Error("expected LastChecked to be set")
	}
}

func TestProvider_HealthCheck_Unhealthy(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	status, err := provider.HealthCheck(context.Background(), ModelLlama31_8BInstant)
	if err != nil {
		t.Fatalf("HealthCheck should not return error: %v", err)
	}

	if status.Healthy {
		t.Error("expected healthy=false")
	}

	if status.Message == "" {
		t.Error("expected error message to be set")
	}
}

func TestProvider_Complete_ContextCancellation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		resp := ChatCompletionResponse{
			ID:      "chatcmpl_123",
			Object:  "chat.completion",
			Model:   ModelLlama31_70BVersatile,
			Choices: []Choice{{Message: Message{Content: "Response"}, FinishReason: FinishReasonStop}},
			Usage:   Usage{PromptTokens: 5, CompletionTokens: 1},
		}
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := provider.Complete(ctx, req)
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}

func TestBuildRequest_MessageRoles(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 100,
		Messages: []ports.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
		Temperature: 0.7,
	}

	groqReq := provider.buildRequest(req)

	// Should have 4 messages (system at front)
	if len(groqReq.Messages) != 4 {
		t.Errorf("expected 4 messages, got %d", len(groqReq.Messages))
	}

	// First should be system
	if groqReq.Messages[0].Role != RoleSystem {
		t.Errorf("expected first message to be system, got %v", groqReq.Messages[0].Role)
	}

	// Temperature should be set
	if groqReq.Temperature == nil || *groqReq.Temperature != 0.7 {
		t.Error("temperature not set correctly")
	}
}

func TestBuildRequest_SystemPromptPriority(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")

	// When SystemPrompt is provided, it should take priority over system message in Messages
	req := ports.CompletionRequest{
		ModelID:      ModelLlama31_70BVersatile,
		MaxTokens:    100,
		SystemPrompt: "Priority system prompt",
		Messages: []ports.Message{
			{Role: "system", Content: "This should be ignored"},
			{Role: "user", Content: "Hello"},
		},
	}

	groqReq := provider.buildRequest(req)

	// Should have 2 messages (system from SystemPrompt + user)
	if len(groqReq.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(groqReq.Messages))
	}

	// First should be system with priority content
	if groqReq.Messages[0].Content != "Priority system prompt" {
		t.Errorf("expected system prompt content, got %q", groqReq.Messages[0].Content)
	}
}

func TestClient_HandleErrorResponse(t *testing.T) {
	tests := []struct {
		statusCode int
		errType    string
		errMessage string
	}{
		{401, "authentication_error", "Invalid API key"},
		{403, "permission_error", "Access denied"},
		{404, "not_found_error", "Model not found"},
		{422, "invalid_request_error", "Invalid parameters"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("HTTP_%d", tt.statusCode), func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: ErrorInfo{
						Type:    tt.errType,
						Message: tt.errMessage,
					},
				})
			}

			server, provider := newTestServer(t, handler)
			defer server.Close()

			req := ports.CompletionRequest{
				ModelID:   ModelLlama31_70BVersatile,
				MaxTokens: 100,
				Messages:  []ports.Message{{Role: "user", Content: "Test"}},
			}

			_, err := provider.Complete(context.Background(), req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tt.errMessage) {
				t.Errorf("error should contain message %q: %v", tt.errMessage, err)
			}
		})
	}
}

func TestProvider_Complete_EmptyChoices(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			ID:      "chatcmpl_123",
			Object:  "chat.completion",
			Model:   ModelLlama31_70BVersatile,
			Choices: []Choice{}, // Empty choices
			Usage:   Usage{PromptTokens: 5, CompletionTokens: 0},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Content != "" {
		t.Errorf("expected empty content, got %q", resp.Content)
	}
}

func TestNewProviderWithAPIKey(t *testing.T) {
	provider := NewProviderWithAPIKey("my-api-key")

	if provider == nil {
		t.Fatal("expected provider to be created")
	}

	info := provider.Info()
	if info.Name != "groq" {
		t.Errorf("expected name 'groq', got %q", info.Name)
	}
}

func TestNewProvider_WithOptions(t *testing.T) {
	config := Config{
		APIKey:     "test-key",
		BaseURL:    "https://custom.groq.com",
		Timeout:    30 * time.Second,
		MaxRetries: 5,
	}

	provider := NewProvider(config, WithTimeout(10*time.Second))

	if provider == nil {
		t.Fatal("expected provider to be created")
	}

	info := provider.Info()
	if info.BaseURL != "https://custom.groq.com" {
		t.Errorf("expected custom base URL, got %q", info.BaseURL)
	}
}

func TestProvider_Stream_NoUsageInChunks(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Send chunks without usage info (some providers don't include it)
		events := []string{
			`data: {"id":"chatcmpl_123","object":"chat.completion.chunk","model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl_123","object":"chat.completion.chunk","model":"llama-3.1-70b-versatile","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}

		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := provider.Stream(context.Background(), req, func(chunk string) error {
		return nil
	})

	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Usage should be 0 when not provided
	if resp.InputTokens != 0 {
		t.Errorf("expected 0 input tokens when not provided, got %d", resp.InputTokens)
	}
}

func TestProvider_Complete_FinishReasonLength(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			ID:     "chatcmpl_123",
			Object: "chat.completion",
			Model:  ModelLlama31_70BVersatile,
			Choices: []Choice{
				{
					Message:      Message{Content: "Truncated response..."},
					FinishReason: FinishReasonLength,
				},
			},
			Usage: Usage{PromptTokens: 100, CompletionTokens: 1000},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelLlama31_70BVersatile,
		MaxTokens: 1000,
		Messages:  []ports.Message{{Role: "user", Content: "Write a long story"}},
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.FinishReason != "length" {
		t.Errorf("expected finish reason 'length', got %q", resp.FinishReason)
	}
}
