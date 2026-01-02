package openai

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// Provider implements the ports.ProviderPort interface for OpenAI.
type Provider struct {
	client *Client
	config Config
}

// Ensure Provider implements ProviderPort at compile time.
var _ ports.ProviderPort = (*Provider)(nil)

// NewProvider creates a new OpenAI provider with the given configuration.
func NewProvider(config Config) *Provider {
	return &Provider{
		client: NewClient(config),
		config: config,
	}
}

// NewProviderWithAPIKey creates a new OpenAI provider with default configuration.
func NewProviderWithAPIKey(apiKey string) *Provider {
	return NewProvider(DefaultConfig(apiKey))
}

// Info returns metadata about this provider.
func (p *Provider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:        "openai",
		Description: "OpenAI API provider for GPT-4, GPT-3.5, and O1 models",
		BaseURL:     p.config.BaseURL,
		IsLocal:     false,
	}
}

// ListModels returns the list of available models.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	// Return the statically known supported models
	// We could use client.ListModels() but it returns ALL models including deprecated ones
	return SupportedModels(), nil
}

// SupportsModel checks if this provider supports the given model.
func (p *Provider) SupportsModel(ctx context.Context, modelID string) (bool, error) {
	return slices.Contains(SupportedModels(), modelID), nil
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

	// For cloud providers, if we support the model, it's available
	return true, nil
}

// Complete sends a completion request and returns the response.
func (p *Provider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	startTime := time.Now()

	openaiReq := p.buildRequest(req)

	resp, _, err := p.client.Chat(ctx, openaiReq)
	if err != nil {
		return nil, err
	}

	return p.buildResponse(resp, startTime), nil
}

// Stream sends a streaming completion request and calls the callback for each chunk.
func (p *Provider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	startTime := time.Now()

	openaiReq := p.buildRequest(req)

	var fullContent strings.Builder
	var inputTokens, outputTokens int
	var finishReason string
	var modelUsed string

	_, err := p.client.ChatStream(ctx, openaiReq, func(chunk *StreamChunk) error {
		// Capture model from first chunk
		if modelUsed == "" && chunk.Model != "" {
			modelUsed = chunk.Model
		}

		// Process choices
		for _, choice := range chunk.Choices {
			// Accumulate content
			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				if err := cb(choice.Delta.Content); err != nil {
					return err
				}
			}

			// Capture finish reason
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				finishReason = string(*choice.FinishReason)
			}
		}

		// Capture usage from final chunk (when stream_options.include_usage is true)
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

// buildRequest converts a ports.CompletionRequest to an OpenAI ChatCompletionRequest.
func (p *Provider) buildRequest(req ports.CompletionRequest) *ChatCompletionRequest {
	messages := make([]Message, 0, len(req.Messages)+1)

	// Add system prompt as first message if provided
	if req.SystemPrompt != "" {
		messages = append(messages, Message{
			Role:    RoleSystem,
			Content: req.SystemPrompt,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		// If there's a system message in messages and we already added SystemPrompt, skip it
		if msg.Role == "system" && req.SystemPrompt != "" {
			continue
		}

		var role MessageRole
		switch msg.Role {
		case "system":
			role = RoleSystem
		case "user":
			role = RoleUser
		case "assistant":
			role = RoleAssistant
		default:
			role = MessageRole(msg.Role)
		}

		messages = append(messages, Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	openaiReq := &ChatCompletionRequest{
		Model:    req.ModelID,
		Messages: messages,
	}

	// Add max tokens if specified
	if req.MaxTokens > 0 {
		openaiReq.MaxTokens = &req.MaxTokens
	}

	// Add temperature if non-zero
	if req.Temperature > 0 {
		openaiReq.Temperature = &req.Temperature
	}

	return openaiReq
}

// buildResponse converts an OpenAI ChatCompletionResponse to a ports.CompletionResponse.
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
