package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

func TestProvider_Info(t *testing.T) {
	p := NewProvider()
	info := p.Info()

	if info.Name != "ollama" {
		t.Errorf("expected name 'ollama', got '%s'", info.Name)
	}
	if !info.IsLocal {
		t.Error("expected IsLocal to be true")
	}
	if info.BaseURL != DefaultBaseURL {
		t.Errorf("expected base URL '%s', got '%s'", DefaultBaseURL, info.BaseURL)
	}
}

func TestProvider_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != EndpointTags {
			t.Errorf("expected path '%s', got '%s'", EndpointTags, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected method GET, got '%s'", r.Method)
		}

		resp := TagsResponse{
			Models: []ModelInfo{
				{Name: "llama2:latest", Model: "llama2"},
				{Name: "mistral:latest", Model: "mistral"},
				{Name: "codellama:7b", Model: "codellama"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProviderWithURL(server.URL)
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(models) != 3 {
		t.Errorf("expected 3 models, got %d", len(models))
	}

	expectedModels := []string{"llama2:latest", "mistral:latest", "codellama:7b"}
	for i, expected := range expectedModels {
		if models[i] != expected {
			t.Errorf("expected model '%s', got '%s'", expected, models[i])
		}
	}
}

func TestProvider_ListModels_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "internal server error"})
	}))
	defer server.Close()

	p := NewProviderWithURL(server.URL)
	_, err := p.ListModels(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestProvider_SupportsModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TagsResponse{
			Models: []ModelInfo{
				{Name: "llama2:latest"},
				{Name: "mistral:7b"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProviderWithURL(server.URL)

	tests := []struct {
		modelID  string
		expected bool
	}{
		{"llama2:latest", true},
		{"llama2", true},
		{"LLAMA2", true},
		{"mistral:7b", true},
		{"gpt-4", false},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			supported, err := p.SupportsModel(context.Background(), tt.modelID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if supported != tt.expected {
				t.Errorf("SupportsModel(%s) = %v, expected %v", tt.modelID, supported, tt.expected)
			}
		})
	}
}

func TestProvider_IsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TagsResponse{
			Models: []ModelInfo{
				{Name: "llama2:latest"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProviderWithURL(server.URL)

	available, err := p.IsAvailable(context.Background(), "llama2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !available {
		t.Error("expected model to be available")
	}
}

func TestProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != EndpointChat {
			t.Errorf("expected path '%s', got '%s'", EndpointChat, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got '%s'", r.Method)
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Model != "llama2" {
			t.Errorf("expected model 'llama2', got '%s'", req.Model)
		}
		if req.Stream {
			t.Error("expected stream to be false for Complete")
		}
		if len(req.Messages) != 2 {
			t.Errorf("expected 2 messages (system + user), got %d", len(req.Messages))
		}

		resp := ChatResponse{
			Model:     "llama2",
			CreatedAt: time.Now(),
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Hello! How can I help you?",
			},
			Done:            true,
			DoneReason:      "stop",
			PromptEvalCount: 10,
			EvalCount:       6,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProviderWithURL(server.URL)

	req := ports.CompletionRequest{
		ModelID: "llama2",
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:    100,
		Temperature:  0.7,
		SystemPrompt: "You are a helpful assistant.",
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("unexpected content: %s", resp.Content)
	}
	if resp.InputTokens != 10 {
		t.Errorf("expected input tokens 10, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 6 {
		t.Errorf("expected output tokens 6, got %d", resp.OutputTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish reason 'stop', got '%s'", resp.FinishReason)
	}
	if resp.ModelUsed != "llama2" {
		t.Errorf("expected model 'llama2', got '%s'", resp.ModelUsed)
	}
}

func TestProvider_Complete_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "model not found"})
	}))
	defer server.Close()

	p := NewProviderWithURL(server.URL)

	req := ports.CompletionRequest{
		ModelID: "nonexistent",
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("expected error to contain 'model not found', got '%s'", err.Error())
	}
}

func TestProvider_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != EndpointChat {
			t.Errorf("expected path '%s', got '%s'", EndpointChat, r.URL.Path)
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Error("expected stream to be true for Stream")
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected ResponseWriter to be a Flusher")
		}

		chunks := []ChatResponse{
			{Model: "llama2", Message: ChatMessage{Role: "assistant", Content: "Hello"}, Done: false},
			{Model: "llama2", Message: ChatMessage{Role: "assistant", Content: " there"}, Done: false},
			{Model: "llama2", Message: ChatMessage{Role: "assistant", Content: "!"}, Done: true, DoneReason: "stop", PromptEvalCount: 5, EvalCount: 3},
		}

		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			w.Write(data)
			w.Write([]byte("\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := NewProviderWithURL(server.URL)

	req := ports.CompletionRequest{
		ModelID: "llama2",
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	var chunks []string
	resp, err := p.Stream(context.Background(), req, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedChunks := []string{"Hello", " there", "!"}
	if len(chunks) != len(expectedChunks) {
		t.Errorf("expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}
	for i, expected := range expectedChunks {
		if i < len(chunks) && chunks[i] != expected {
			t.Errorf("chunk %d: expected '%s', got '%s'", i, expected, chunks[i])
		}
	}

	if resp.Content != "Hello there!" {
		t.Errorf("expected full content 'Hello there!', got '%s'", resp.Content)
	}
}

func TestProvider_HealthCheck(t *testing.T) {
	tests := []struct {
		name            string
		modelID         string
		serverHandler   http.HandlerFunc
		expectedHealthy bool
		expectedMessage string
	}{
		{
			name:    "healthy server no model",
			modelID: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := TagsResponse{Models: []ModelInfo{{Name: "llama2:latest"}}}
				json.NewEncoder(w).Encode(resp)
			},
			expectedHealthy: true,
			expectedMessage: "ok",
		},
		{
			name:    "healthy server with available model",
			modelID: "llama2",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := TagsResponse{Models: []ModelInfo{{Name: "llama2:latest"}}}
				json.NewEncoder(w).Encode(resp)
			},
			expectedHealthy: true,
			expectedMessage: "ok",
		},
		{
			name:    "healthy server with unavailable model",
			modelID: "gpt-4",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				resp := TagsResponse{Models: []ModelInfo{{Name: "llama2:latest"}}}
				json.NewEncoder(w).Encode(resp)
			},
			expectedHealthy: false,
			expectedMessage: "server healthy but model not found: gpt-4",
		},
		{
			name:    "unhealthy server",
			modelID: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedHealthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			p := NewProviderWithURL(server.URL)
			status, err := p.HealthCheck(context.Background(), tt.modelID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if status.Healthy != tt.expectedHealthy {
				t.Errorf("expected healthy=%v, got %v", tt.expectedHealthy, status.Healthy)
			}
			if tt.expectedMessage != "" && status.Message != tt.expectedMessage {
				t.Errorf("expected message '%s', got '%s'", tt.expectedMessage, status.Message)
			}
			if status.Latency <= 0 {
				t.Error("expected positive latency")
			}
			if status.LastChecked.IsZero() {
				t.Error("expected non-zero LastChecked")
			}
		})
	}
}

func TestProvider_HealthCheck_ServerDown(t *testing.T) {
	p := NewProviderWithURL("http://localhost:99999")

	status, err := p.HealthCheck(context.Background(), "")
	if err != nil {
		t.Fatalf("HealthCheck should not return error, got: %v", err)
	}

	if status.Healthy {
		t.Error("expected healthy=false for unreachable server")
	}
}

func TestNormalizeModelID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"llama2", "llama2:latest"},
		{"llama2:latest", "llama2:latest"},
		{"LLAMA2", "llama2:latest"},
		{"mistral:7b", "mistral:7b"},
		{"  llama2  ", "llama2:latest"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeModelID(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeModelID(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertMessages(t *testing.T) {
	messages := []ports.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result := convertMessages(messages, "You are helpful.")
	if len(result) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result))
	}

	if result[0].Role != "system" || result[0].Content != "You are helpful." {
		t.Errorf("expected system message first, got %+v", result[0])
	}
	if result[1].Role != "user" || result[1].Content != "Hello" {
		t.Errorf("unexpected second message: %+v", result[1])
	}
	if result[2].Role != "assistant" || result[2].Content != "Hi there!" {
		t.Errorf("unexpected third message: %+v", result[2])
	}
}

func TestConvertMessages_NoSystemPrompt(t *testing.T) {
	messages := []ports.Message{
		{Role: "user", Content: "Hello"},
	}

	result := convertMessages(messages, "")
	if len(result) != 1 {
		t.Errorf("expected 1 message, got %d", len(result))
	}

	if result[0].Role != "user" {
		t.Errorf("expected user message, got %s", result[0].Role)
	}
}

func TestClient_Options(t *testing.T) {
	client := NewClient(
		WithBaseURL("http://custom:8080"),
		WithTimeout(60*time.Second),
	)

	if client.baseURL != "http://custom:8080" {
		t.Errorf("expected custom base URL, got %s", client.baseURL)
	}
	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("expected 60s timeout, got %v", client.httpClient.Timeout)
	}
}

func TestProvider_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		json.NewEncoder(w).Encode(TagsResponse{})
	}))
	defer server.Close()

	p := NewProviderWithURL(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := p.ListModels(ctx)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestWithClient(t *testing.T) {
	customClient := NewClient(WithBaseURL("http://custom:11434"))
	p := NewProvider(WithClient(customClient))

	info := p.Info()
	if info.BaseURL != "http://custom:11434" {
		t.Errorf("expected custom base URL, got %s", info.BaseURL)
	}
}

func TestWithHTTPClient(t *testing.T) {
	customHTTP := &http.Client{Timeout: 99 * time.Second}
	client := NewClient(WithHTTPClient(customHTTP))

	if client.httpClient.Timeout != 99*time.Second {
		t.Errorf("expected 99s timeout, got %v", client.httpClient.Timeout)
	}
}

func TestClient_Generate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != EndpointGenerate {
			t.Errorf("expected path '%s', got '%s'", EndpointGenerate, r.URL.Path)
		}

		resp := GenerateResponse{
			Model:     "llama2",
			Response:  "Generated response",
			Done:      true,
			EvalCount: 50,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	req := &GenerateRequest{
		Model:  "llama2",
		Prompt: "Hello",
	}

	resp, err := client.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Response != "Generated response" {
		t.Errorf("expected 'Generated response', got '%s'", resp.Response)
	}
	if !resp.Done {
		t.Error("expected Done to be true")
	}
}

func TestClient_GenerateStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != EndpointGenerate {
			t.Errorf("expected path '%s', got '%s'", EndpointGenerate, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, _ := w.(http.Flusher)

		chunks := []GenerateResponse{
			{Response: "Hello ", Done: false},
			{Response: "World", Done: false},
			{Response: "", Done: true, EvalCount: 10},
		}

		for _, chunk := range chunks {
			json.NewEncoder(w).Encode(chunk)
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	req := &GenerateRequest{
		Model:  "llama2",
		Prompt: "Hello",
	}

	var chunks []string
	resp, err := client.GenerateStream(context.Background(), req, func(chunk *GenerateResponse) error {
		if chunk.Response != "" {
			chunks = append(chunks, chunk.Response)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}

	if resp == nil || !resp.Done {
		t.Error("expected final response with Done=true")
	}
}

func TestClient_Generate_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid model"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	req := &GenerateRequest{
		Model:  "invalid",
		Prompt: "Hello",
	}

	_, err := client.Generate(context.Background(), req)
	if err == nil {
		t.Error("expected error for bad request")
	}
}

func TestClient_ParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found - plain text"))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	req := &GenerateRequest{Model: "test", Prompt: "test"}

	_, err := client.Generate(context.Background(), req)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
}
