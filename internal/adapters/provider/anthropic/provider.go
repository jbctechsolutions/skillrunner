package anthropic

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// Provider implements the ports.ProviderPort interface for Anthropic Claude.
type Provider struct {
	client *Client
	config Config
}

// Ensure Provider implements ProviderPort at compile time.
var _ ports.ProviderPort = (*Provider)(nil)

// NewProvider creates a new Anthropic provider with the given configuration.
func NewProvider(config Config) *Provider {
	return &Provider{
		client: NewClient(config),
		config: config,
	}
}

// NewProviderWithAPIKey creates a new Anthropic provider with default configuration.
func NewProviderWithAPIKey(apiKey string) *Provider {
	return NewProvider(DefaultConfig(apiKey))
}

// Info returns metadata about this provider.
func (p *Provider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:        "anthropic",
		Description: "Anthropic Claude API provider for Claude 3 and Claude 3.5 models",
		BaseURL:     p.config.BaseURL,
		IsLocal:     false,
	}
}

// ListModels returns the list of available models.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	// Anthropic doesn't have a list models endpoint, so we return the known models
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

	anthropicReq := p.buildRequest(req)

	resp, err := p.client.SendMessage(ctx, anthropicReq)
	if err != nil {
		return nil, err
	}

	return p.buildResponse(resp, startTime), nil
}

// Stream sends a streaming completion request and calls the callback for each chunk.
func (p *Provider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	startTime := time.Now()

	anthropicReq := p.buildRequest(req)

	var fullContent strings.Builder
	var inputTokens, outputTokens int
	var finishReason string
	var modelUsed string

	err := p.client.StreamMessage(ctx, anthropicReq, func(event *StreamEvent) error {
		switch event.Type {
		case EventMessageStart:
			if event.Message != nil {
				modelUsed = event.Message.Model
				if event.Message.Usage != nil {
					inputTokens = event.Message.Usage.InputTokens
				}
			}
		case EventContentBlockDelta:
			if event.Delta != nil && event.Delta.Text != "" {
				fullContent.WriteString(event.Delta.Text)
				if err := cb(event.Delta.Text); err != nil {
					return err
				}
			}
		case EventMessageDelta:
			if event.Delta != nil && event.Delta.StopReason != nil {
				finishReason = string(*event.Delta.StopReason)
			}
			if event.Usage != nil {
				outputTokens = event.Usage.OutputTokens
			}
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

// buildRequest converts a ports.CompletionRequest to an Anthropic MessagesRequest.
func (p *Provider) buildRequest(req ports.CompletionRequest) *MessagesRequest {
	messages := make([]Message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		// Skip system messages as they go in the system field
		if msg.Role == "system" {
			continue
		}
		messages = append(messages, Message{
			Role: MessageRole(msg.Role),
			Content: MessageContent{
				{Type: "text", Text: msg.Content},
			},
		})
	}

	anthropicReq := &MessagesRequest{
		Model:     req.ModelID,
		MaxTokens: req.MaxTokens,
		Messages:  messages,
	}

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		anthropicReq.System = req.SystemPrompt
	} else {
		// Check for system message in the messages
		for _, msg := range req.Messages {
			if msg.Role == "system" {
				anthropicReq.System = msg.Content
				break
			}
		}
	}

	// Add temperature if non-zero
	if req.Temperature > 0 {
		temp := req.Temperature
		anthropicReq.Temperature = &temp
	}

	return anthropicReq
}

// buildResponse converts an Anthropic MessagesResponse to a ports.CompletionResponse.
func (p *Provider) buildResponse(resp *MessagesResponse, startTime time.Time) *ports.CompletionResponse {
	var content strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}

	return &ports.CompletionResponse{
		Content:      content.String(),
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		FinishReason: string(resp.StopReason),
		ModelUsed:    resp.Model,
		Duration:     time.Since(startTime),
	}
}
