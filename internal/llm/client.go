package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// Client provides a unified interface for calling different LLM providers
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new LLM client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Minute, // Long timeout for complex tasks with large models
		},
	}
}

// CompletionRequest represents a request to generate text
type CompletionRequest struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float64
	Stream      bool
}

// CompletionResponse represents the response from a completion
type CompletionResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Model        string
	Provider     string
	Duration     time.Duration
}

// Complete generates a completion using the specified model
func (c *Client) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// If streaming is requested, use streaming method
	if req.Stream {
		return c.completeStreaming(ctx, req)
	}

	// Determine provider from model name
	provider := getProvider(req.Model)

	switch provider {
	case types.ModelProviderOllama:
		return c.completeOllama(ctx, req)
	case types.ModelProviderAnthropic:
		return c.completeAnthropic(ctx, req)
	case types.ModelProviderGroq:
		return c.completeGroq(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// StreamCompletion streams a completion and calls callback for each chunk
func (c *Client) StreamCompletion(ctx context.Context, req CompletionRequest, callback func(string) error) (*CompletionResponse, error) {
	provider := getProvider(req.Model)
	if provider != types.ModelProviderOllama {
		// For non-Ollama providers, fall back to non-streaming
		return c.Complete(ctx, CompletionRequest{
			Model:       req.Model,
			Prompt:      req.Prompt,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			Stream:      false,
		})
	}

	return c.streamOllama(ctx, req, callback)
}

// completeOllama calls Ollama API
func (c *Client) completeOllama(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	startTime := time.Now()

	// Extract model name (remove "ollama/" prefix)
	modelName := strings.TrimPrefix(req.Model, "ollama/")

	// Build Ollama API request
	ollamaReq := map[string]interface{}{
		"model":  modelName,
		"prompt": req.Prompt,
		"stream": req.Stream,
	}

	if req.MaxTokens > 0 {
		ollamaReq["options"] = map[string]interface{}{
			"num_predict": req.MaxTokens,
		}
	}
	if req.Temperature > 0 {
		if options, ok := ollamaReq["options"].(map[string]interface{}); ok {
			options["temperature"] = req.Temperature
		} else {
			ollamaReq["options"] = map[string]interface{}{
				"temperature": req.Temperature,
			}
		}
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Call Ollama API
	endpoint := "http://localhost:11434/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Handle streaming vs non-streaming
	if req.Stream {
		return c.handleOllamaStream(resp.Body, req.Model, startTime)
	}

	// Parse non-streaming response
	var ollamaResp struct {
		Model           string `json:"model"`
		Response        string `json:"response"`
		Done            bool   `json:"done"`
		Context         []int  `json:"context"`
		TotalDuration   int64  `json:"total_duration"`
		LoadDuration    int64  `json:"load_duration"`
		PromptEvalCount int    `json:"prompt_eval_count"`
		EvalCount       int    `json:"eval_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &CompletionResponse{
		Content:      ollamaResp.Response,
		InputTokens:  ollamaResp.PromptEvalCount,
		OutputTokens: ollamaResp.EvalCount,
		Model:        req.Model,
		Provider:     string(types.ModelProviderOllama),
		Duration:     time.Since(startTime),
	}, nil
}

// handleOllamaStream handles streaming response from Ollama
func (c *Client) handleOllamaStream(body io.ReadCloser, model string, startTime time.Time) (*CompletionResponse, error) {
	defer body.Close()

	var fullContent strings.Builder
	var inputTokens, outputTokens int
	decoder := json.NewDecoder(body)

	for {
		var chunk struct {
			Response        string `json:"response"`
			Done            bool   `json:"done"`
			PromptEvalCount int    `json:"prompt_eval_count"`
			EvalCount       int    `json:"eval_count"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode stream chunk: %w", err)
		}

		if chunk.Response != "" {
			fullContent.WriteString(chunk.Response)
		}

		if chunk.PromptEvalCount > 0 {
			inputTokens = chunk.PromptEvalCount
		}
		if chunk.EvalCount > 0 {
			outputTokens = chunk.EvalCount
		}

		if chunk.Done {
			break
		}
	}

	return &CompletionResponse{
		Content:      fullContent.String(),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        model,
		Provider:     string(types.ModelProviderOllama),
		Duration:     time.Since(startTime),
	}, nil
}

// streamOllama streams Ollama responses and calls callback for each chunk
func (c *Client) streamOllama(ctx context.Context, req CompletionRequest, callback func(string) error) (*CompletionResponse, error) {
	startTime := time.Now()
	modelName := strings.TrimPrefix(req.Model, "ollama/")

	ollamaReq := map[string]interface{}{
		"model":  modelName,
		"prompt": req.Prompt,
		"stream": true,
	}

	if req.MaxTokens > 0 {
		ollamaReq["options"] = map[string]interface{}{
			"num_predict": req.MaxTokens,
		}
	}
	if req.Temperature > 0 {
		if options, ok := ollamaReq["options"].(map[string]interface{}); ok {
			options["temperature"] = req.Temperature
		} else {
			ollamaReq["options"] = map[string]interface{}{
				"temperature": req.Temperature,
			}
		}
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := "http://localhost:11434/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var fullContent strings.Builder
	var inputTokens, outputTokens int
	decoder := json.NewDecoder(resp.Body)

	for {
		var chunk struct {
			Response        string `json:"response"`
			Done            bool   `json:"done"`
			PromptEvalCount int    `json:"prompt_eval_count"`
			EvalCount       int    `json:"eval_count"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode stream chunk: %w", err)
		}

		if chunk.Response != "" {
			fullContent.WriteString(chunk.Response)
			// Call callback for each chunk
			if callback != nil {
				if err := callback(chunk.Response); err != nil {
					return nil, fmt.Errorf("callback error: %w", err)
				}
			}
		}

		if chunk.PromptEvalCount > 0 {
			inputTokens = chunk.PromptEvalCount
		}
		if chunk.EvalCount > 0 {
			outputTokens = chunk.EvalCount
		}

		if chunk.Done {
			break
		}
	}

	return &CompletionResponse{
		Content:      fullContent.String(),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        req.Model,
		Provider:     string(types.ModelProviderOllama),
		Duration:     time.Since(startTime),
	}, nil
}

// completeStreaming handles streaming completion (aggregates stream into single response)
func (c *Client) completeStreaming(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	provider := getProvider(req.Model)
	if provider != types.ModelProviderOllama {
		// For non-Ollama, disable streaming and use regular completion
		req.Stream = false
		return c.Complete(ctx, req)
	}

	// For Ollama, use streaming but aggregate
	return c.streamOllama(ctx, req, nil)
}

// completeAnthropic calls Anthropic API
func (c *Client) completeAnthropic(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	startTime := time.Now()

	// Get API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Extract model name (remove "anthropic/" prefix)
	modelName := strings.TrimPrefix(req.Model, "anthropic/")

	// Build Anthropic API request
	anthropicReq := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": req.Prompt,
			},
		},
		"max_tokens": req.MaxTokens,
	}

	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}

	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Call Anthropic API
	endpoint := "https://api.anthropic.com/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Extract text from content blocks
	var content string
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &CompletionResponse{
		Content:      content,
		InputTokens:  anthropicResp.Usage.InputTokens,
		OutputTokens: anthropicResp.Usage.OutputTokens,
		Model:        req.Model,
		Provider:     string(types.ModelProviderAnthropic),
		Duration:     time.Since(startTime),
	}, nil
}

// completeGroq calls Groq API for ultra-fast inference using GroqProvider
func (c *Client) completeGroq(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	startTime := time.Now()

	// Create Groq provider (uses GROQ_API_KEY env var by default)
	provider, err := NewGroqProvider("")
	if err != nil {
		return nil, err
	}

	// Extract model name (remove "groq/" prefix)
	modelName := strings.TrimPrefix(req.Model, "groq/")

	// Build chat request
	chatReq := ChatRequest{
		Model: modelName,
		Messages: []ChatMessage{
			{Role: RoleUser, Content: req.Prompt},
		},
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature),
	}

	// Call Groq via provider
	chatResp, err := provider.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	return &CompletionResponse{
		Content:      chatResp.Content,
		InputTokens:  chatResp.Usage.InputTokens,
		OutputTokens: chatResp.Usage.OutputTokens,
		Model:        req.Model,
		Provider:     string(types.ModelProviderGroq),
		Duration:     time.Since(startTime),
	}, nil
}

// getProvider determines the provider from model name.
// Supported providers: Ollama (local), Anthropic (cloud), Groq (cloud, ultra-fast).
func getProvider(model string) types.ModelProvider {
	if strings.HasPrefix(model, "ollama/") {
		return types.ModelProviderOllama
	}
	if strings.HasPrefix(model, "anthropic/") {
		return types.ModelProviderAnthropic
	}
	if strings.HasPrefix(model, "groq/") {
		return types.ModelProviderGroq
	}
	// Default to Ollama for bare model names (local-first approach)
	return types.ModelProviderOllama
}
