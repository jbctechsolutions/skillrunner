package anthropic

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
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		Version:    "2023-06-01",
		Timeout:    5 * time.Second,
		MaxRetries: 2,
	}
	return server, NewProvider(config)
}

func TestProvider_Info(t *testing.T) {
	provider := NewProviderWithAPIKey("test-key")
	info := provider.Info()

	if info.Name != "anthropic" {
		t.Errorf("expected name 'anthropic', got %q", info.Name)
	}
	if info.IsLocal {
		t.Error("expected IsLocal to be false")
	}
	if !strings.Contains(info.Description, "Claude") {
		t.Errorf("expected description to mention Claude, got %q", info.Description)
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
		ModelClaude35Sonnet,
		ModelClaude35Haiku,
		ModelClaude3Opus,
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
		{ModelClaude35Sonnet, true},
		{ModelClaude3Opus, true},
		{"unknown-model", false},
		{"gpt-4", false},
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

func TestProvider_Complete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("expected /messages, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-api-key" {
			t.Error("missing or incorrect API key")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("missing or incorrect anthropic-version header")
		}

		// Decode request to verify structure
		var req MessagesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Send response
		resp := MessagesResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       RoleAssistant,
			Model:      req.Model,
			StopReason: StopReasonEndTurn,
			Content: []ContentBlock{
				{Type: "text", Text: "Hello! How can I help you today?"},
			},
			Usage: Usage{
				InputTokens:  10,
				OutputTokens: 8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelClaude35Sonnet,
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
	if resp.FinishReason != "end_turn" {
		t.Errorf("expected finish reason 'end_turn', got %q", resp.FinishReason)
	}
	if resp.ModelUsed != ModelClaude35Sonnet {
		t.Errorf("expected model %q, got %q", ModelClaude35Sonnet, resp.ModelUsed)
	}
}

func TestProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedReq MessagesRequest

	handler := func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := MessagesResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       RoleAssistant,
			Model:      ModelClaude35Sonnet,
			StopReason: StopReasonEndTurn,
			Content: []ContentBlock{
				{Type: "text", Text: "I am a helpful assistant."},
			},
			Usage: Usage{InputTokens: 20, OutputTokens: 5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:      ModelClaude35Sonnet,
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

	if receivedReq.System != "You are a helpful assistant." {
		t.Errorf("system prompt not set correctly: %q", receivedReq.System)
	}
}

func TestProvider_Complete_ErrorResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Type: "error",
			Error: ErrorInfo{
				Type:    "invalid_request_error",
				Message: "max_tokens must be greater than 0",
			},
		})
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelClaude35Sonnet,
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

		resp := MessagesResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       RoleAssistant,
			Model:      ModelClaude35Sonnet,
			StopReason: StopReasonEndTurn,
			Content:    []ContentBlock{{Type: "text", Text: "Success after retry"}},
			Usage:      Usage{InputTokens: 5, OutputTokens: 3},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelClaude35Sonnet,
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
		var req MessagesRequest
		json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("expected stream=true in request")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		// Send SSE events
		events := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-5-sonnet-20241022","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" World"}}`,
			`event: content_block_stop
data: {"type":"content_block_stop","index":0}`,
			`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}`,
			`event: message_stop
data: {"type":"message_stop"}`,
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
		ModelID:   ModelClaude35Sonnet,
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

	if resp.FinishReason != "end_turn" {
		t.Errorf("expected finish reason 'end_turn', got %q", resp.FinishReason)
	}
}

func TestProvider_Stream_CallbackError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		events := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-5-sonnet-20241022"}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
		}

		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
		}
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	req := ports.CompletionRequest{
		ModelID:   ModelClaude35Sonnet,
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
		resp := MessagesResponse{
			ID:         "msg_health",
			Type:       "message",
			Role:       RoleAssistant,
			Model:      ModelClaude35Haiku,
			StopReason: StopReasonMaxTokens,
			Content:    []ContentBlock{{Type: "text", Text: "Hi"}},
			Usage:      Usage{InputTokens: 2, OutputTokens: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	status, err := provider.HealthCheck(context.Background(), ModelClaude35Haiku)
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
}

func TestProvider_HealthCheck_Unhealthy(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	status, err := provider.HealthCheck(context.Background(), ModelClaude35Haiku)
	if err != nil {
		t.Fatalf("HealthCheck should not return error: %v", err)
	}

	if status.Healthy {
		t.Error("expected healthy=false")
	}
}

func TestProvider_Complete_ContextCancellation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		resp := MessagesResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       RoleAssistant,
			Model:      ModelClaude35Sonnet,
			StopReason: StopReasonEndTurn,
			Content:    []ContentBlock{{Type: "text", Text: "Response"}},
			Usage:      Usage{InputTokens: 5, OutputTokens: 1},
		}
		json.NewEncoder(w).Encode(resp)
	}

	server, provider := newTestServer(t, handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := ports.CompletionRequest{
		ModelID:   ModelClaude35Sonnet,
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
		ModelID:   ModelClaude35Sonnet,
		MaxTokens: 100,
		Messages: []ports.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
		Temperature: 0.7,
	}

	anthropicReq := provider.buildRequest(req)

	// System message should be in system field, not messages
	if anthropicReq.System != "You are helpful." {
		t.Errorf("expected system field to be set, got %q", anthropicReq.System)
	}

	// Should have 3 messages (excluding system)
	if len(anthropicReq.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(anthropicReq.Messages))
	}

	// Temperature should be set
	if anthropicReq.Temperature == nil || *anthropicReq.Temperature != 0.7 {
		t.Error("temperature not set correctly")
	}
}

func TestClient_HandleErrorResponse(t *testing.T) {
	// Note: 5xx errors are retried, so we only test non-retryable errors here
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
					Type: "error",
					Error: ErrorInfo{
						Type:    tt.errType,
						Message: tt.errMessage,
					},
				})
			}

			server, provider := newTestServer(t, handler)
			defer server.Close()

			req := ports.CompletionRequest{
				ModelID:   ModelClaude35Sonnet,
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
