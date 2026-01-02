package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		APIKey:         "test-api-key",
		BaseURL:        server.URL,
		Organization:   "",
		Timeout:        5 * time.Second,
		MaxRetries:     2,
		RetryBaseDelay: 10 * time.Millisecond,
		RetryMaxDelay:  50 * time.Millisecond,
	}
	return server, NewProvider(config)
}

func TestProvider_Info(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")
	info := provider.Info()

	if info.Name != "openai" {
		t.Errorf("expected name 'openai', got %q", info.Name)
	}
	if info.IsLocal {
		t.Error("expected IsLocal to be false")
	}
	if !strings.Contains(info.Description, "OpenAI") {
		t.Errorf("expected description to mention OpenAI, got %q", info.Description)
	}
	if !strings.Contains(info.Description, "GPT") {
		t.Errorf("expected description to mention GPT, got %q", info.Description)
	}
	if info.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default BaseURL, got %q", info.BaseURL)
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
		ModelGPT4,
		ModelGPT4o,
		ModelGPT4oMini,
		ModelGPT35Turbo,
		ModelO1,
	}
	for _, expected := range expectedModels {
		found := false
		for _, model := range models {
			if model == expected {
				found = true
				break
			}
		}
		if !found {
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
		{ModelGPT4, true},
		{ModelGPT4o, true},
		{ModelGPT4oMini, true},
		{ModelGPT35Turbo, true},
		{ModelO1, true},
		{ModelO1Mini, true},
		{"unknown-model", false},
		{"claude-3-sonnet", false},
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

	// Supported model should be available
	available, err := provider.IsAvailable(ctx, ModelGPT4o)
	if err != nil {
		t.Fatalf("IsAvailable failed: %v", err)
	}
	if !available {
		t.Error("expected GPT-4o to be available")
	}

	// Unsupported model should not be available
	available, err = provider.IsAvailable(ctx, "unknown-model")
	if err != nil {
		t.Fatalf("IsAvailable failed: %v", err)
	}
	if available {
		t.Error("expected unknown model to not be available")
	}
}

func TestProvider_Complete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing or incorrect Authorization header")
		}

		// Decode request to verify structure
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.Stream {
			t.Error("expected stream=false for non-streaming request")
		}

		// Send response
		resp := ChatCompletionResponse{
			ID:      "chatcmpl-123",
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
		ModelID:   ModelGPT4o,
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
	if resp.ModelUsed != ModelGPT4o {
		t.Errorf("expected model %q, got %q", ModelGPT4o, resp.ModelUsed)
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
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   ModelGPT4o,
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
		ModelID:      ModelGPT4o,
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

	// First message should be system prompt
	if len(receivedReq.Messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(receivedReq.Messages))
	}
	if receivedReq.Messages[0].Role != RoleSystem {
		t.Errorf("expected first message to be system, got %s", receivedReq.Messages[0].Role)
	}
	if receivedReq.Messages[0].Content != "You are a helpful assistant." {
		t.Errorf("system prompt not set correctly: %q", receivedReq.Messages[0].Content)
	}
}

func TestProvider_Complete_WithTemperature(t *testing.T) {
	var receivedReq ChatCompletionRequest

	handler := func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := ChatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   ModelGPT4o,
			Choices: []Choice{
				{
					Index:        0,
					Message:      Message{Role: RoleAssistant, Content: "Response"},
					FinishReason: FinishReasonStop,
				},
			},
			Usage: Usage{PromptTokens: 5, CompletionTokens: 1, TotalTokens: 6},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:     ModelGPT4o,
		MaxTokens:   100,
		Temperature: 0.7,
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if receivedReq.Temperature == nil {
		t.Fatal("expected temperature to be set")
	}
	if *receivedReq.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", *receivedReq.Temperature)
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
		ModelID:   ModelGPT4o,
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
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   ModelGPT4o,
			Choices: []Choice{
				{
					Index:        0,
					Message:      Message{Role: RoleAssistant, Content: "Success after retry"},
					FinishReason: FinishReasonStop,
				},
			},
			Usage: Usage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelGPT4o,
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

		// Send SSE events
		events := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":" World"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
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
		ModelID:   ModelGPT4o,
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

	if resp.ModelUsed != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", resp.ModelUsed)
	}
}

func TestProvider_Stream_CallbackError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		events := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
		}

		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
		}
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelGPT4o,
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

func TestProvider_HealthCheck(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			ID:      "chatcmpl-health",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   ModelGPT4oMini,
			Choices: []Choice{
				{
					Index:        0,
					Message:      Message{Role: RoleAssistant, Content: "Hi"},
					FinishReason: FinishReasonStop,
				},
			},
			Usage: Usage{PromptTokens: 2, CompletionTokens: 1, TotalTokens: 3},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	status, err := provider.HealthCheck(context.Background(), ModelGPT4oMini)
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

	status, err := provider.HealthCheck(context.Background(), ModelGPT4oMini)
	if err != nil {
		t.Fatalf("HealthCheck should not return error: %v", err)
	}

	if status.Healthy {
		t.Error("expected healthy=false")
	}

	if status.Message == "" {
		t.Error("expected error message in unhealthy status")
	}
}

func TestProvider_Complete_ContextCancellation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		resp := ChatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   ModelGPT4o,
			Choices: []Choice{
				{
					Index:        0,
					Message:      Message{Role: RoleAssistant, Content: "Response"},
					FinishReason: FinishReasonStop,
				},
			},
			Usage: Usage{PromptTokens: 5, CompletionTokens: 1, TotalTokens: 6},
		}
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := ports.CompletionRequest{
		ModelID:   ModelGPT4o,
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
		ModelID:   ModelGPT4o,
		MaxTokens: 100,
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
		Temperature: 0.7,
	}

	openaiReq := provider.buildRequest(req)

	// Should have 3 messages
	if len(openaiReq.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(openaiReq.Messages))
	}

	// Verify roles
	expectedRoles := []MessageRole{RoleUser, RoleAssistant, RoleUser}
	for i, msg := range openaiReq.Messages {
		if msg.Role != expectedRoles[i] {
			t.Errorf("message %d: expected role %s, got %s", i, expectedRoles[i], msg.Role)
		}
	}

	// Temperature should be set
	if openaiReq.Temperature == nil || *openaiReq.Temperature != 0.7 {
		t.Error("temperature not set correctly")
	}

	// MaxTokens should be set
	if openaiReq.MaxTokens == nil || *openaiReq.MaxTokens != 100 {
		t.Error("max_tokens not set correctly")
	}
}

func TestBuildRequest_SystemPromptSkipsDuplicateSystem(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")

	req := ports.CompletionRequest{
		ModelID:      ModelGPT4o,
		MaxTokens:    100,
		SystemPrompt: "You are a helpful assistant.",
		Messages: []ports.Message{
			{Role: "system", Content: "Ignored system message"},
			{Role: "user", Content: "Hello"},
		},
	}

	openaiReq := provider.buildRequest(req)

	// Should have 2 messages: system prompt + user message
	// The system message in Messages should be skipped since SystemPrompt is set
	if len(openaiReq.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(openaiReq.Messages))
	}

	if openaiReq.Messages[0].Role != RoleSystem {
		t.Errorf("first message should be system, got %s", openaiReq.Messages[0].Role)
	}
	if openaiReq.Messages[0].Content != "You are a helpful assistant." {
		t.Errorf("system message content incorrect: %q", openaiReq.Messages[0].Content)
	}
}

func TestClient_HandleErrorResponse(t *testing.T) {
	tests := []struct {
		statusCode int
		errType    string
		errMessage string
	}{
		{401, "invalid_api_key", "Invalid API key"},
		{403, "access_denied", "Access denied"},
		{404, "model_not_found", "Model not found"},
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
				ModelID:   ModelGPT4o,
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
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   ModelGPT4o,
			Choices: []Choice{}, // Empty choices
			Usage:   Usage{PromptTokens: 5, CompletionTokens: 0, TotalTokens: 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelGPT4o,
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// With empty choices, content and finish reason should be empty
	if resp.Content != "" {
		t.Errorf("expected empty content, got %q", resp.Content)
	}
	if resp.FinishReason != "" {
		t.Errorf("expected empty finish reason, got %q", resp.FinishReason)
	}
}

func TestNewProvider(t *testing.T) {
	config := Config{
		APIKey:         "test-api-key",
		BaseURL:        "https://custom.openai.com/v1",
		Organization:   "org-123",
		Timeout:        10 * time.Second,
		MaxRetries:     5,
		RetryBaseDelay: 2 * time.Second,
		RetryMaxDelay:  60 * time.Second,
	}

	provider := NewProvider(config)

	if provider.client == nil {
		t.Error("expected client to be initialized")
	}
	if provider.config.APIKey != "test-api-key" {
		t.Errorf("expected API key to be set, got %q", provider.config.APIKey)
	}
	if provider.config.BaseURL != "https://custom.openai.com/v1" {
		t.Errorf("expected custom BaseURL, got %q", provider.config.BaseURL)
	}
}

func TestNewProviderWithAPIKey(t *testing.T) {
	provider := NewProviderWithAPIKey("my-api-key")

	if provider.config.APIKey != "my-api-key" {
		t.Errorf("expected API key 'my-api-key', got %q", provider.config.APIKey)
	}
	if provider.config.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default BaseURL, got %q", provider.config.BaseURL)
	}
}

func TestProvider_Stream_NoUsageInChunks(t *testing.T) {
	// Test when usage is not included in stream chunks
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		events := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
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
		ModelID:   ModelGPT4o,
		MaxTokens: 100,
		Messages:  []ports.Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := provider.Stream(context.Background(), req, func(chunk string) error {
		return nil
	})

	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Usage should be 0 when not included in chunks
	if resp.InputTokens != 0 || resp.OutputTokens != 0 {
		t.Errorf("expected 0 tokens when usage not in stream, got input=%d output=%d",
			resp.InputTokens, resp.OutputTokens)
	}

	if resp.Content != "Hi" {
		t.Errorf("expected content 'Hi', got %q", resp.Content)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("expected finish reason 'stop', got %q", resp.FinishReason)
	}
}

func TestProvider_Stream_ErrorResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorInfo{
				Type:    "invalid_request_error",
				Message: "Invalid model specified",
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

	if !strings.Contains(err.Error(), "Invalid model") {
		t.Errorf("error should mention invalid model: %v", err)
	}
}

func TestProvider_Complete_FinishReasonLength(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   ModelGPT4o,
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: "Truncated response due to token limit...",
					},
					FinishReason: FinishReasonLength,
				},
			},
			Usage: Usage{PromptTokens: 10, CompletionTokens: 100, TotalTokens: 110},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelGPT4o,
		MaxTokens: 100,
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

func TestBuildRequest_ZeroMaxTokens(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")

	req := ports.CompletionRequest{
		ModelID:   ModelGPT4o,
		MaxTokens: 0, // Zero max tokens
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	openaiReq := provider.buildRequest(req)

	// MaxTokens should be nil when 0
	if openaiReq.MaxTokens != nil {
		t.Errorf("expected nil MaxTokens, got %d", *openaiReq.MaxTokens)
	}
}

func TestBuildRequest_ZeroTemperature(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")

	req := ports.CompletionRequest{
		ModelID:     ModelGPT4o,
		MaxTokens:   100,
		Temperature: 0, // Zero temperature
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	openaiReq := provider.buildRequest(req)

	// Temperature should be nil when 0
	if openaiReq.Temperature != nil {
		t.Errorf("expected nil Temperature, got %f", *openaiReq.Temperature)
	}
}
