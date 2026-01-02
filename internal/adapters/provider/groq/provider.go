package groq

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// Provider implements the ports.ProviderPort interface for Groq.
type Provider struct {
	client *Client
	config Config
}

// Ensure Provider implements ProviderPort at compile time.
var _ ports.ProviderPort = (*Provider)(nil)

// NewProvider creates a new Groq provider with the given configuration.
func NewProvider(config Config, opts ...ClientOption) *Provider {
	return &Provider{
		client: NewClient(config.APIKey, opts...),
		config: config,
	}
}

// NewProviderWithAPIKey creates a new Groq provider with default configuration.
func NewProviderWithAPIKey(apiKey string, opts ...ClientOption) *Provider {
	return NewProvider(DefaultConfig(apiKey), opts...)
}

// Info returns metadata about this provider.
func (p *Provider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:        "groq",
		Description: "Groq API provider for fast LLM inference (Llama, Mixtral, Gemma models)",
		BaseURL:     p.config.BaseURL,
		IsLocal:     false,
	}
}

// ListModels returns the list of available models.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	// Return the statically defined supported models
	// This is more reliable than querying the API since Groq may return models we don't support
	return SupportedModels(), nil
}

// SupportsModel checks if this provider supports the given model.
func (p *Provider) SupportsModel(ctx context.Context, modelID string) (bool, error) {
	models := SupportedModels()
	return slices.Contains(models, modelID), nil
}

// IsAvailable checks if a model is currently available.
func (p *Provider) IsAvailable(ctx context.Context, modelID string) (bool, error) {
	supported, err := p.SupportsModel(ctx, modelID)
	if err != nil {
		return false, err
	}
	if !supported {
		return false, nil
	}

	// For cloud providers, if we can reach the API, the model is available
	return true, nil
}

// Complete sends a completion request and returns the response.
func (p *Provider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	startTime := time.Now()

	groqReq := p.buildRequest(req)

	resp, err := p.client.Chat(ctx, groqReq)
	if err != nil {
		return nil, err
	}

	return p.buildResponse(resp, startTime), nil
}

// Stream sends a streaming completion request and calls the callback for each chunk.
func (p *Provider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	startTime := time.Now()

	groqReq := p.buildRequest(req)

	var fullContent strings.Builder
	var inputTokens, outputTokens int
	var finishReason string
	var modelUsed string

	err := p.client.ChatStream(ctx, groqReq, func(chunk *ChatCompletionChunk) error {
		modelUsed = chunk.Model

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				if err := cb(choice.Delta.Content); err != nil {
					return err
				}
			}
			if choice.FinishReason != "" {
				finishReason = string(choice.FinishReason)
			}
		}

		// Usage may be included in the final chunk
		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &ports.CompletionResponse{
		Content:      fullContent.String(),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		FinishReason: finishReason,
		ModelUsed:    modelUsed,
		Duration:     time.Since(startTime),
	}, nil
}

// HealthCheck verifies the provider is healthy and responsive.
func (p *Provider) HealthCheck(ctx context.Context, modelID string) (*ports.HealthStatus, error) {
	startTime := time.Now()

	// Send a minimal request to check health
	req := ports.CompletionRequest{
		ModelID:   modelID,
		MaxTokens: 1,
		Messages: []ports.Message{
			{Role: "user", Content: "Hi"},
		},
	}

	_, err := p.Complete(ctx, req)
	latency := time.Since(startTime)

	if err != nil {
		return &ports.HealthStatus{
			Healthy:     false,
			Message:     err.Error(),
			Latency:     latency,
			LastChecked: time.Now(),
		}, nil
	}

	return &ports.HealthStatus{
		Healthy:     true,
		Message:     "OK",
		Latency:     latency,
		LastChecked: time.Now(),
	}, nil
}

// buildRequest converts a ports.CompletionRequest to a Groq ChatCompletionRequest.
func (p *Provider) buildRequest(req ports.CompletionRequest) *ChatCompletionRequest {
	messages := make([]Message, 0, len(req.Messages)+1)

	// Add system prompt as the first message if provided
	if req.SystemPrompt != "" {
		messages = append(messages, Message{
			Role:    RoleSystem,
			Content: req.SystemPrompt,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		// Skip system messages if we already added a system prompt
		if msg.Role == "system" && req.SystemPrompt != "" {
			continue
		}

		var role MessageRole
		switch msg.Role {
		case "system":
			role = RoleSystem
		case "assistant":
			role = RoleAssistant
		default:
			role = RoleUser
		}

		messages = append(messages, Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	groqReq := &ChatCompletionRequest{
		Model:     req.ModelID,
		MaxTokens: req.MaxTokens,
		Messages:  messages,
	}

	// Add temperature if non-zero
	if req.Temperature > 0 {
		temp := req.Temperature
		groqReq.Temperature = &temp
	}

	return groqReq
}

// buildResponse converts a Groq ChatCompletionResponse to a ports.CompletionResponse.
func (p *Provider) buildResponse(resp *ChatCompletionResponse, startTime time.Time) *ports.CompletionResponse {
	var content string
	var finishReason string

	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
		finishReason = string(resp.Choices[0].FinishReason)
	}

	return &ports.CompletionResponse{
		Content:      content,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		FinishReason: finishReason,
		ModelUsed:    resp.Model,
		Duration:     time.Since(startTime),
	}
}
