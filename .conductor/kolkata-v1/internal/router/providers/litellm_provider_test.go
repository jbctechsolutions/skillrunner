package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/router/litellm"
	"github.com/jbctechsolutions/skillrunner/internal/router/types"
)

func TestNewLiteLLMProvider(t *testing.T) {
	provider := NewLiteLLMProvider("http://localhost:18432", "test-key")
	if provider == nil {
		t.Fatal("NewLiteLLMProvider returned nil")
	}

	if provider.client == nil {
		t.Error("client should be initialized")
	}
}

func TestLiteLLMProvider_Name(t *testing.T) {
	provider := NewLiteLLMProvider("", "")
	if provider.Name() != "litellm" {
		t.Errorf("Name() = %s; want litellm", provider.Name())
	}
}

func TestLiteLLMProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := types.CompletionResponse{
			ID:     "test-id",
			Object: "chat.completion",
			Choices: []types.Choice{
				{
					Message: types.Message{
						Role:    "assistant",
						Content: "Test response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server
	provider := &LiteLLMProvider{
		client: litellm.NewClient(server.URL, "test-key"),
	}

	req := &types.CompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	resp, err := provider.Complete(ctx, req)

	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Complete returned nil response")
	}

	if resp.ID != "test-id" {
		t.Errorf("ID = %s; want test-id", resp.ID)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Choices count = %d; want 1", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Test response" {
		t.Errorf("Content = %s; want Test response", resp.Choices[0].Message.Content)
	}
}

func TestLiteLLMProvider_IsAvailable_Available(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	provider := &LiteLLMProvider{
		client: litellm.NewClient(server.URL, "test-key"),
	}

	ctx := context.Background()
	available := provider.IsAvailable(ctx)

	if !available {
		t.Error("IsAvailable() = false; want true")
	}
}

func TestLiteLLMProvider_IsAvailable_Unauthorized(t *testing.T) {
	// Server returns 401, but service is available
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	provider := &LiteLLMProvider{
		client: litellm.NewClient(server.URL, "invalid-key"),
	}

	ctx := context.Background()
	available := provider.IsAvailable(ctx)

	if !available {
		t.Error("IsAvailable() = false; want true (service is available, just unauthorized)")
	}
}

func TestLiteLLMProvider_IsAvailable_NotAvailable(t *testing.T) {
	// Use invalid port to simulate network error
	provider := &LiteLLMProvider{
		client: litellm.NewClient("http://localhost:99999", "test-key"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	available := provider.IsAvailable(ctx)

	if available {
		t.Error("IsAvailable() = true; want false (service not available)")
	}
}

func TestLiteLLMProvider_IsAvailable_Timeout(t *testing.T) {
	// Server that times out
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &LiteLLMProvider{
		client: litellm.NewClient(server.URL, "test-key"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	available := provider.IsAvailable(ctx)

	if available {
		t.Error("IsAvailable() = true; want false (timeout)")
	}
}

func TestIsNetworkError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection refused",
			err:  &testError{msg: "connection refused"},
			want: true,
		},
		{
			name: "no such host",
			err:  &testError{msg: "no such host"},
			want: true,
		},
		{
			name: "timeout",
			err:  &testError{msg: "timeout"},
			want: true,
		},
		{
			name: "network error",
			err:  &testError{msg: "network error"},
			want: true,
		},
		{
			name: "dial tcp",
			err:  &testError{msg: "dial tcp: connection refused"},
			want: true,
		},
		{
			name: "context deadline exceeded",
			err:  &testError{msg: "context deadline exceeded"},
			want: true,
		},
		{
			name: "HTTP error",
			err:  &testError{msg: "HTTP 500: Internal Server Error"},
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isNetworkError(tc.err)
			if got != tc.want {
				t.Errorf("isNetworkError() = %v; want %v", got, tc.want)
			}
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
