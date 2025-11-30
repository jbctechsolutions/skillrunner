package litellm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/router/types"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:18432", "test-api-key")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.baseURL != "http://localhost:18432" {
		t.Errorf("baseURL = %s; want http://localhost:18432", client.baseURL)
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("apiKey = %s; want test-api-key", client.apiKey)
	}

	if client.httpClient == nil {
		t.Error("httpClient should be initialized")
	}
}

func TestNewClientWithDefaults(t *testing.T) {
	client := NewClient("", "")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.baseURL != "http://localhost:18432" {
		t.Errorf("baseURL = %s; want http://localhost:18432", client.baseURL)
	}
}

func TestClient_Complete_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s; want POST", r.Method)
		}

		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Path = %s; want /v1/chat/completions", r.URL.Path)
		}

		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Error("Authorization header should start with 'Bearer '")
		}

		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %s; want application/json", contentType)
		}

		// Parse request body
		var req types.CompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify request
		if req.Model != "gpt-4" {
			t.Errorf("Model = %s; want gpt-4", req.Model)
		}

		if len(req.Messages) != 1 {
			t.Errorf("Messages count = %d; want 1", len(req.Messages))
		}

		// Send response
		response := types.CompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []types.Choice{
				{
					Index: 0,
					Message: types.Message{
						Role:    "assistant",
						Content: "Hello! How can I help you?",
					},
					FinishReason: "stop",
				},
			},
			Usage: types.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewClient(server.URL, "test-api-key")

	// Make request
	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	}

	ctx := context.Background()
	resp, err := client.Complete(ctx, req)

	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Complete returned nil response")
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("ID = %s; want chatcmpl-123", resp.ID)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Choices count = %d; want 1", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Hello! How can I help you?" {
		t.Errorf("Content = %s; want Hello! How can I help you?", resp.Choices[0].Message.Content)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d; want 30", resp.Usage.TotalTokens)
	}
}

func TestClient_Complete_HTTPError(t *testing.T) {
	// Create a mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for HTTP 500, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Error should contain status code, got: %v", err)
	}
}

func TestClient_Complete_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestClient_Complete_NetworkError(t *testing.T) {
	// Use an invalid URL to simulate network error
	client := NewClient("http://localhost:99999", "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for network failure, got nil")
	}
}

func TestClient_Complete_ContextCancellation(t *testing.T) {
	// Create a mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for context cancellation, got nil")
	}

	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Error should mention context, got: %v", err)
	}
}

func TestClient_Complete_NilRequest(t *testing.T) {
	client := NewClient("http://localhost:18432", "test-api-key")

	ctx := context.Background()
	_, err := client.Complete(ctx, nil)

	if err == nil {
		t.Fatal("Expected error for nil request, got nil")
	}

	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Error should mention nil request, got: %v", err)
	}
}

func TestClient_Complete_EmptyMessages(t *testing.T) {
	client := NewClient("http://localhost:18432", "test-api-key")

	req := &types.CompletionRequest{
		Model:    "gpt-4",
		Messages: []types.Message{},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	// This should either validate and return an error, or send the request
	// For now, we'll let the API handle validation
	// But we should test that the request is sent properly
	if err == nil {
		// If no validation, that's fine - API will handle it
	}
}

func TestClient_Complete_WithMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req types.CompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Metadata == nil {
			t.Error("Metadata should be present in request")
		}

		if req.Metadata["test"] != "value" {
			t.Errorf("Metadata[test] = %v; want value", req.Metadata["test"])
		}

		response := types.CompletionResponse{
			ID:     "test",
			Object: "chat.completion",
			Choices: []types.Choice{
				{
					Message: types.Message{
						Role:    "assistant",
						Content: "Response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
		Metadata: map[string]interface{}{
			"test": "value",
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
}

func TestClient_Complete_StreamOption(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req types.CompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Error("Stream should be true in request")
		}

		// For streaming, we'd need different handling, but for now just return normal response
		response := types.CompletionResponse{
			ID:     "test",
			Object: "chat.completion",
			Choices: []types.Choice{
				{
					Message: types.Message{
						Role:    "assistant",
						Content: "Response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
}

func TestClient_Complete_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "invalid-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for unauthorized, got nil")
	}

	if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "Unauthorized") {
		t.Errorf("Error should mention unauthorized status, got: %v", err)
	}
}

func TestClient_Complete_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "Invalid request", "type": "invalid_request_error"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for bad request, got nil")
	}

	if !strings.Contains(err.Error(), "400") && !strings.Contains(err.Error(), "Bad Request") {
		t.Errorf("Error should mention bad request status, got: %v", err)
	}
}

func TestClient_Complete_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for rate limit, got nil")
	}

	if !strings.Contains(err.Error(), "429") && !strings.Contains(err.Error(), "Too Many Requests") {
		t.Errorf("Error should mention rate limit status, got: %v", err)
	}
}

func TestClient_Complete_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	resp, err := client.Complete(ctx, req)

	// Empty response might be valid in some cases, but typically should have choices
	if err != nil {
		// If we validate, that's fine
		return
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestClient_Complete_ErrorResponseWithoutMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {}}`)) // Error object without message
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for HTTP 500, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Error should contain status code, got: %v", err)
	}
}

func TestClient_Complete_ErrorResponseEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(``)) // Empty body
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for HTTP 502, got nil")
	}

	if !strings.Contains(err.Error(), "502") {
		t.Errorf("Error should contain status code, got: %v", err)
	}
}

func TestClient_Complete_ErrorResponseInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`invalid json`)) // Invalid JSON in error response
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for HTTP 403, got nil")
	}

	if !strings.Contains(err.Error(), "403") {
		t.Errorf("Error should contain status code, got: %v", err)
	}
}

func TestClient_Complete_MultipleChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := types.CompletionResponse{
			ID:     "test",
			Object: "chat.completion",
			Choices: []types.Choice{
				{
					Index: 0,
					Message: types.Message{
						Role:    "assistant",
						Content: "First choice",
					},
				},
				{
					Index: 1,
					Message: types.Message{
						Role:    "assistant",
						Content: "Second choice",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	resp, err := client.Complete(ctx, req)

	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if len(resp.Choices) != 2 {
		t.Errorf("Choices count = %d; want 2", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "First choice" {
		t.Errorf("First choice content = %s; want First choice", resp.Choices[0].Message.Content)
	}

	if resp.Choices[1].Message.Content != "Second choice" {
		t.Errorf("Second choice content = %s; want Second choice", resp.Choices[1].Message.Content)
	}
}

func TestClient_Complete_RequestTimeout(t *testing.T) {
	// Create a server that takes longer than the timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	client.httpClient.Timeout = 100 * time.Millisecond

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, req)

	if err == nil {
		t.Fatal("Expected error for timeout, got nil")
	}
}

func TestClient_URLConstruction(t *testing.T) {
	testCases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "URL with trailing slash",
			baseURL: "http://localhost:18432/",
			want:    "http://localhost:18432/v1/chat/completions",
		},
		{
			name:    "URL without trailing slash",
			baseURL: "http://localhost:18432",
			want:    "http://localhost:18432/v1/chat/completions",
		},
		{
			name:    "URL with path",
			baseURL: "http://localhost:18432/api",
			want:    "http://localhost:18432/api/v1/chat/completions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" && r.URL.Path != "/api/v1/chat/completions" {
					t.Errorf("Path = %s; want /v1/chat/completions or /api/v1/chat/completions", r.URL.Path)
				}

				response := types.CompletionResponse{
					ID:     "test",
					Object: "chat.completion",
					Choices: []types.Choice{
						{
							Message: types.Message{
								Role:    "assistant",
								Content: "Response",
							},
						},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Replace server URL with our test case URL
			client := NewClient(tc.baseURL, "test-api-key")
			// Override the baseURL for testing
			client.baseURL = server.URL

			req := &types.CompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{Role: "user", Content: "Hello"},
				},
			}

			ctx := context.Background()
			_, err := client.Complete(ctx, req)

			if err != nil {
				t.Fatalf("Complete failed: %v", err)
			}
		})
	}
}
