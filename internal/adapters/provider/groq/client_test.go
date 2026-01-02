package groq

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

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default config", func(t *testing.T) {
		client := NewClient("test-api-key")

		if client.config.APIKey != "test-api-key" {
			t.Errorf("expected API key 'test-api-key', got '%s'", client.config.APIKey)
		}
		if client.config.BaseURL != DefaultBaseURL {
			t.Errorf("expected base URL '%s', got '%s'", DefaultBaseURL, client.config.BaseURL)
		}
		if client.config.MaxRetries != 3 {
			t.Errorf("expected 3 max retries, got %d", client.config.MaxRetries)
		}
	})

	t.Run("applies functional options", func(t *testing.T) {
		customTimeout := 30 * time.Second
		customBaseURL := "https://custom.api.groq.com"
		customMaxRetries := 5

		client := NewClient("test-api-key",
			WithTimeout(customTimeout),
			WithBaseURL(customBaseURL),
			WithMaxRetries(customMaxRetries),
		)

		if client.config.Timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, client.config.Timeout)
		}
		if client.config.BaseURL != customBaseURL {
			t.Errorf("expected base URL '%s', got '%s'", customBaseURL, client.config.BaseURL)
		}
		if client.config.MaxRetries != customMaxRetries {
			t.Errorf("expected %d max retries, got %d", customMaxRetries, client.config.MaxRetries)
		}
	})

	t.Run("applies custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{Timeout: 10 * time.Second}
		client := NewClient("test-api-key", WithHTTPClient(customClient))

		if client.httpClient != customClient {
			t.Error("expected custom HTTP client to be set")
		}
	})
}

func TestClient_Chat(t *testing.T) {
	t.Run("successful chat completion", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != EndpointChatCompletions {
				t.Errorf("expected path %s, got %s", EndpointChatCompletions, r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-api-key" {
				t.Errorf("expected Authorization header 'Bearer test-api-key', got '%s'", r.Header.Get("Authorization"))
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
			}

			// Verify request body
			var req ChatCompletionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req.Model != ModelLlama31_70BVersatile {
				t.Errorf("expected model %s, got %s", ModelLlama31_70BVersatile, req.Model)
			}

			// Send response
			resp := ChatCompletionResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   ModelLlama31_70BVersatile,
				Choices: []Choice{
					{
						Index:        0,
						Message:      Message{Role: RoleAssistant, Content: "Hello! How can I help you?"},
						FinishReason: FinishReasonStop,
					},
				},
				Usage: Usage{
					PromptTokens:     10,
					CompletionTokens: 15,
					TotalTokens:      25,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		req := &ChatCompletionRequest{
			Model: ModelLlama31_70BVersatile,
			Messages: []Message{
				{Role: RoleUser, Content: "Hello!"},
			},
		}

		resp, err := client.Chat(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.ID != "chatcmpl-123" {
			t.Errorf("expected ID 'chatcmpl-123', got '%s'", resp.ID)
		}
		if len(resp.Choices) != 1 {
			t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
		}
		if resp.Choices[0].Message.Content != "Hello! How can I help you?" {
			t.Errorf("unexpected response content: %s", resp.Choices[0].Message.Content)
		}
	})

	t.Run("handles API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorInfo{
					Message: "Invalid model",
					Type:    "invalid_request_error",
					Code:    "model_not_found",
				},
			})
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))

		req := &ChatCompletionRequest{
			Model:    "invalid-model",
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		_, err := client.Chat(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var skillErr *errors.SkillrunnerError
		if !errors.As(err, &skillErr) {
			t.Fatalf("expected SkillrunnerError, got %T", err)
		}
		if skillErr.Code != errors.CodeValidation {
			t.Errorf("expected error code %s, got %s", errors.CodeValidation, skillErr.Code)
		}
	})

	t.Run("handles unauthorized error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorInfo{
					Message: "Invalid API key",
					Type:    "authentication_error",
				},
			})
		}))
		defer server.Close()

		client := NewClient("invalid-key", WithBaseURL(server.URL), WithMaxRetries(0))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		_, err := client.Chat(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var skillErr *errors.SkillrunnerError
		if !errors.As(err, &skillErr) {
			t.Fatalf("expected SkillrunnerError, got %T", err)
		}
		if skillErr.Code != errors.CodeConfiguration {
			t.Errorf("expected error code %s, got %s", errors.CodeConfiguration, skillErr.Code)
		}
	})

	t.Run("retries on server error", func(t *testing.T) {
		var attempts int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := atomic.AddInt32(&attempts, 1)
			if current <= 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Third attempt succeeds
			resp := ChatCompletionResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   ModelLlama31_70BVersatile,
				Choices: []Choice{
					{
						Index:        0,
						Message:      Message{Role: RoleAssistant, Content: "Success after retry"},
						FinishReason: FinishReasonStop,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(3))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		resp, err := client.Chat(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if atomic.LoadInt32(&attempts) != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
		if resp.Choices[0].Message.Content != "Success after retry" {
			t.Errorf("unexpected response content: %s", resp.Choices[0].Message.Content)
		}
	})

	t.Run("retries on rate limit", func(t *testing.T) {
		var attempts int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := atomic.AddInt32(&attempts, 1)
			if current == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			resp := ChatCompletionResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   ModelLlama31_70BVersatile,
				Choices: []Choice{{Index: 0, Message: Message{Role: RoleAssistant, Content: "OK"}, FinishReason: FinishReasonStop}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(2))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		resp, err := client.Chat(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if atomic.LoadInt32(&attempts) != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
		if resp.Choices[0].Message.Content != "OK" {
			t.Errorf("unexpected response content: %s", resp.Choices[0].Message.Content)
		}
	})

	t.Run("fails after max retries", func(t *testing.T) {
		var attempts int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(2))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		_, err := client.Chat(context.Background(), req)
		if err == nil {
			t.Fatal("expected error after max retries")
		}

		// Initial attempt + 2 retries = 3 attempts
		if atomic.LoadInt32(&attempts) != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("respects context cancellation during retry", func(t *testing.T) {
		var attempts int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(5))

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after first attempt
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		_, err := client.Chat(ctx, req)
		if err == nil {
			t.Fatal("expected context error")
		}
	})
}

func TestClient_ChatStream(t *testing.T) {
	t.Run("successful streaming response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify stream is set to true in request
			var req ChatCompletionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if !req.Stream {
				t.Error("expected stream to be true")
			}

			// Send SSE response
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected ResponseWriter to be Flusher")
			}

			chunks := []string{"Hello", " World", "!"}
			for i, chunk := range chunks {
				data := ChatCompletionChunk{
					ID:      "chatcmpl-123",
					Object:  "chat.completion.chunk",
					Created: 1677652288,
					Model:   ModelLlama31_70BVersatile,
					Choices: []StreamChoice{
						{
							Index: 0,
							Delta: Message{Content: chunk},
						},
					},
				}
				if i == len(chunks)-1 {
					data.Choices[0].FinishReason = FinishReasonStop
				}

				jsonData, _ := json.Marshal(data)
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				flusher.Flush()
			}

			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		var receivedContent strings.Builder
		var chunkCount int

		err := client.ChatStream(context.Background(), req, func(chunk *ChatCompletionChunk) error {
			chunkCount++
			if len(chunk.Choices) > 0 {
				receivedContent.WriteString(chunk.Choices[0].Delta.Content)
			}
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if chunkCount != 3 {
			t.Errorf("expected 3 chunks, got %d", chunkCount)
		}

		if receivedContent.String() != "Hello World!" {
			t.Errorf("expected 'Hello World!', got '%s'", receivedContent.String())
		}
	})

	t.Run("handles stream error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorInfo{
					Message: "Invalid request",
					Type:    "invalid_request_error",
				},
			})
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		err := client.ChatStream(context.Background(), req, func(chunk *ChatCompletionChunk) error {
			return nil
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("callback error stops stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)

			for i := 0; i < 10; i++ {
				data := ChatCompletionChunk{
					ID:      "chatcmpl-123",
					Object:  "chat.completion.chunk",
					Created: 1677652288,
					Model:   ModelLlama31_70BVersatile,
					Choices: []StreamChoice{{Index: 0, Delta: Message{Content: "chunk"}}},
				}
				jsonData, _ := json.Marshal(data)
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				flusher.Flush()
			}
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		callbackErr := fmt.Errorf("callback error")
		var chunkCount int

		err := client.ChatStream(context.Background(), req, func(chunk *ChatCompletionChunk) error {
			chunkCount++
			if chunkCount >= 3 {
				return callbackErr
			}
			return nil
		})

		if err != callbackErr {
			t.Errorf("expected callback error, got: %v", err)
		}
		if chunkCount != 3 {
			t.Errorf("expected 3 chunks before error, got %d", chunkCount)
		}
	})

	t.Run("handles malformed JSON in stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {invalid json}\n\n")
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		err := client.ChatStream(context.Background(), req, func(chunk *ChatCompletionChunk) error {
			return nil
		})

		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})
}

func TestClient_ListModels(t *testing.T) {
	t.Run("successful models list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != EndpointModels {
				t.Errorf("expected path %s, got %s", EndpointModels, r.URL.Path)
			}

			resp := ModelsResponse{
				Object: "list",
				Data: []Model{
					{ID: ModelLlama31_70BVersatile, Object: "model", Created: 1677652288, OwnedBy: "groq"},
					{ID: ModelLlama31_8BInstant, Object: "model", Created: 1677652288, OwnedBy: "groq"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		resp, err := client.ListModels(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Data) != 2 {
			t.Errorf("expected 2 models, got %d", len(resp.Data))
		}
		if resp.Data[0].ID != ModelLlama31_70BVersatile {
			t.Errorf("expected model %s, got %s", ModelLlama31_70BVersatile, resp.Data[0].ID)
		}
	})

	t.Run("handles error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorInfo{
					Message: "Not found",
					Type:    "not_found_error",
				},
			})
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))

		_, err := client.ListModels(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_HealthCheck(t *testing.T) {
	t.Run("successful health check", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ModelsResponse{
				Object: "list",
				Data:   []Model{{ID: ModelLlama31_70BVersatile, Object: "model", Created: 1677652288, OwnedBy: "groq"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL))

		err := client.HealthCheck(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("health check fails on error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))

		err := client.HealthCheck(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClient_handleErrorResponse(t *testing.T) {
	t.Run("handles non-JSON error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use 400 Bad Request since 5xx codes trigger retries
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request Text"))
		}))
		defer server.Close()

		client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))

		req := &ChatCompletionRequest{
			Model:    ModelLlama31_70BVersatile,
			Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
		}

		_, err := client.Chat(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Should contain raw response text since it's not valid JSON
		if !strings.Contains(err.Error(), "Bad Request Text") {
			t.Errorf("expected error to contain 'Bad Request Text', got: %s", err.Error())
		}
	})

	t.Run("maps status codes to error codes", func(t *testing.T) {
		tests := []struct {
			statusCode   int
			expectedCode errors.ErrorCode
		}{
			{http.StatusUnauthorized, errors.CodeConfiguration},
			{http.StatusForbidden, errors.CodeConfiguration},
			{http.StatusNotFound, errors.CodeNotFound},
			{http.StatusBadRequest, errors.CodeValidation},
			{http.StatusUnprocessableEntity, errors.CodeValidation},
			{http.StatusTeapot, errors.CodeProvider}, // default for other codes
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					json.NewEncoder(w).Encode(ErrorResponse{
						Error: ErrorInfo{
							Message: "Error message",
							Type:    "error_type",
						},
					})
				}))
				defer server.Close()

				client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))

				req := &ChatCompletionRequest{
					Model:    ModelLlama31_70BVersatile,
					Messages: []Message{{Role: RoleUser, Content: "Hello!"}},
				}

				_, err := client.Chat(context.Background(), req)
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				var skillErr *errors.SkillrunnerError
				if !errors.As(err, &skillErr) {
					t.Fatalf("expected SkillrunnerError, got %T", err)
				}
				if skillErr.Code != tt.expectedCode {
					t.Errorf("expected error code %s, got %s", tt.expectedCode, skillErr.Code)
				}
			})
		}
	})
}
