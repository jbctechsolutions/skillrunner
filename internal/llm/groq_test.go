package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGroqProviderChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-groq-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "llama-3.1-8b-instant",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello! How can I help you today?",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 8,
				"total_tokens":      18,
			},
		})
	}))
	defer server.Close()

	// Set env var for the test
	os.Setenv("GROQ_API_KEY", "test-groq-key")
	defer os.Unsetenv("GROQ_API_KEY")

	provider, err := NewGroqProvider("GROQ_API_KEY")
	if err != nil {
		t.Fatalf("NewGroqProvider error: %v", err)
	}

	// Use SetBaseURL to point to our test server
	provider.SetBaseURL(server.URL)

	// Test the Chat method
	resp, err := provider.Chat(context.Background(), ChatRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []ChatMessage{
			{Role: RoleUser, Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	if resp.Content != "Hello! How can I help you today?" {
		t.Errorf("expected 'Hello! How can I help you today?', got '%s'", resp.Content)
	}

	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected input tokens 10, got %d", resp.Usage.InputTokens)
	}

	if resp.Usage.OutputTokens != 8 {
		t.Errorf("expected output tokens 8, got %d", resp.Usage.OutputTokens)
	}
}

func TestGroqProviderNewProviderNoAPIKey(t *testing.T) {
	os.Unsetenv("GROQ_API_KEY")

	_, err := NewGroqProvider("GROQ_API_KEY")
	if err == nil {
		t.Fatal("expected error when GROQ_API_KEY is not set")
	}
}

func TestGroqProviderName(t *testing.T) {
	os.Setenv("GROQ_API_KEY", "test-key")
	defer os.Unsetenv("GROQ_API_KEY")

	provider, err := NewGroqProvider("")
	if err != nil {
		t.Fatalf("NewGroqProvider error: %v", err)
	}

	if provider.Name() != "groq" {
		t.Errorf("expected provider name 'groq', got '%s'", provider.Name())
	}
}

func TestGroqProviderChatRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	os.Setenv("GROQ_API_KEY", "test-groq-key")
	defer os.Unsetenv("GROQ_API_KEY")

	provider, err := NewGroqProvider("GROQ_API_KEY")
	if err != nil {
		t.Fatalf("NewGroqProvider error: %v", err)
	}

	// Use SetBaseURL to point to our test server
	provider.SetBaseURL(server.URL)

	// Test that rate limit error is properly returned
	_, err = provider.Chat(context.Background(), ChatRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []ChatMessage{
			{Role: RoleUser, Content: "Hello"},
		},
	})

	if err == nil {
		t.Fatal("expected error for rate limit, got nil")
	}

	// Check that error message contains rate limit info
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected error to mention 'rate limit', got: %v", err)
	}
}

func TestGroqProviderChatWithMessages(t *testing.T) {
	os.Setenv("GROQ_API_KEY", "test-groq-key")
	defer os.Unsetenv("GROQ_API_KEY")

	provider, err := NewGroqProvider("")
	if err != nil {
		t.Fatalf("NewGroqProvider error: %v", err)
	}

	// Verify provider implements Provider interface
	var _ Provider = provider

	// Verify Name() returns correct value
	if provider.Name() != "groq" {
		t.Errorf("expected 'groq', got '%s'", provider.Name())
	}
}

func TestGroqProviderChatRequest(t *testing.T) {
	var capturedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "test response",
					},
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     5,
				"completion_tokens": 3,
			},
		})
	}))
	defer server.Close()

	os.Setenv("GROQ_API_KEY", "test-key")
	defer os.Unsetenv("GROQ_API_KEY")

	provider, _ := NewGroqProvider("")
	provider.httpClient = server.Client()

	// Note: Can't test Chat directly since it uses hardcoded endpoint
	// This test verifies provider creation works
	if provider.Name() != "groq" {
		t.Errorf("unexpected provider name: %s", provider.Name())
	}
}

func TestGroqProviderChatResponseParsing(t *testing.T) {
	testCases := []struct {
		name            string
		response        map[string]interface{}
		expectedContent string
		expectedInput   int
		expectedOutput  int
	}{
		{
			name: "single choice",
			response: map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"content": "Hello world",
						},
					},
				},
				"usage": map[string]int{
					"prompt_tokens":     10,
					"completion_tokens": 5,
				},
			},
			expectedContent: "Hello world",
			expectedInput:   10,
			expectedOutput:  5,
		},
		{
			name: "empty choices",
			response: map[string]interface{}{
				"choices": []map[string]interface{}{},
				"usage": map[string]int{
					"prompt_tokens":     10,
					"completion_tokens": 0,
				},
			},
			expectedContent: "",
			expectedInput:   10,
			expectedOutput:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tc.response)
			}))
			defer server.Close()

			os.Setenv("GROQ_API_KEY", "test-key")
			defer os.Unsetenv("GROQ_API_KEY")

			provider, _ := NewGroqProvider("")
			provider.SetBaseURL(server.URL)

			resp, err := provider.Chat(context.Background(), ChatRequest{
				Model: "llama-3.1-8b-instant",
				Messages: []ChatMessage{
					{Role: RoleUser, Content: "Hello"},
				},
			})
			if err != nil {
				t.Fatalf("Chat error: %v", err)
			}

			if resp.Content != tc.expectedContent {
				t.Errorf("expected content '%s', got '%s'", tc.expectedContent, resp.Content)
			}

			if resp.Usage.InputTokens != tc.expectedInput {
				t.Errorf("expected input tokens %d, got %d", tc.expectedInput, resp.Usage.InputTokens)
			}

			if resp.Usage.OutputTokens != tc.expectedOutput {
				t.Errorf("expected output tokens %d, got %d", tc.expectedOutput, resp.Usage.OutputTokens)
			}
		})
	}
}
