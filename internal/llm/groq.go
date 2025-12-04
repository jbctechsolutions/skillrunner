package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// GroqProvider implements the Provider interface for Groq's ultra-fast inference API.
// Groq uses an OpenAI-compatible API format.
type GroqProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewGroqProvider creates a new Groq provider.
// apiKeyEnv is the name of the environment variable containing the API key (default: GROQ_API_KEY).
func NewGroqProvider(apiKeyEnv string) (*GroqProvider, error) {
	if apiKeyEnv == "" {
		apiKeyEnv = "GROQ_API_KEY"
	}
	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("%s environment variable not set", apiKeyEnv)
	}

	return &GroqProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

// Name returns the provider name.
func (p *GroqProvider) Name() string {
	return "groq"
}

// Chat sends a chat request to Groq's API using the OpenAI-compatible format.
func (p *GroqProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Convert ChatMessage to OpenAI-compatible format
	messages := make([]map[string]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    string(msg.Role),
			"content": msg.Content,
		})
	}

	// Build Groq API request (OpenAI-compatible format)
	groqReq := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
	}

	if req.MaxTokens > 0 {
		groqReq["max_tokens"] = req.MaxTokens
	}

	if req.Temperature > 0 {
		groqReq["temperature"] = req.Temperature
	}

	reqBody, err := json.Marshal(groqReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	// Call Groq API
	endpoint := "https://api.groq.com/openai/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("groq request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Handle rate limiting specifically
		if resp.StatusCode == http.StatusTooManyRequests {
			return ChatResponse{}, fmt.Errorf("groq rate limit exceeded: %s", string(body))
		}
		return ChatResponse{}, fmt.Errorf("groq API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse OpenAI-compatible response
	var groqResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return ChatResponse{}, fmt.Errorf("decode response: %w", err)
	}

	// Extract content from first choice
	var content string
	if len(groqResp.Choices) > 0 {
		content = groqResp.Choices[0].Message.Content
	}

	return ChatResponse{
		Content: content,
		Usage: Usage{
			InputTokens:  groqResp.Usage.PromptTokens,
			OutputTokens: groqResp.Usage.CompletionTokens,
		},
	}, nil
}
