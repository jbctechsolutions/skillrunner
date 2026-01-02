package ollama

import (
	"context"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

// Provider implements the ProviderPort interface for Ollama
type Provider struct {
	client *Client
}

// ProviderOption is a functional option for configuring the Provider
type ProviderOption func(*Provider)

// WithClient sets a custom client for the provider
func WithClient(client *Client) ProviderOption {
	return func(p *Provider) {
		p.client = client
	}
}

// NewProvider creates a new Ollama provider
func NewProvider(opts ...ProviderOption) *Provider {
	p := &Provider{
		client: NewClient(),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// NewProviderWithURL creates a new Ollama provider with a custom base URL
func NewProviderWithURL(baseURL string) *Provider {
	return &Provider{
		client: NewClient(WithBaseURL(baseURL)),
	}
}

// Info returns provider metadata
func (p *Provider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:        "ollama",
		Description: "Ollama - Run large language models locally",
		BaseURL:     p.client.baseURL,
		IsLocal:     true,
	}
}

// ListModels returns all available models
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	tagsResp, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]string, len(tagsResp.Models))
	for i, model := range tagsResp.Models {
		models[i] = model.Name
	}

	return models, nil
}

// SupportsModel checks if a model is available in Ollama
func (p *Provider) SupportsModel(ctx context.Context, modelID string) (bool, error) {
	models, err := p.ListModels(ctx)
	if err != nil {
		return false, err
	}

	normalizedID := normalizeModelID(modelID)
	for _, model := range models {
		if normalizeModelID(model) == normalizedID {
			return true, nil
		}
	}

	return false, nil
}

// IsAvailable checks if a specific model is available and ready
func (p *Provider) IsAvailable(ctx context.Context, modelID string) (bool, error) {
	return p.SupportsModel(ctx, modelID)
}

// Complete performs a synchronous completion request
func (p *Provider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	startTime := time.Now()

	messages := convertMessages(req.Messages, req.SystemPrompt)

	chatReq := &ChatRequest{
		Model:    req.ModelID,
		Messages: messages,
		Options: &Options{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	chatResp, err := p.client.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	return &ports.CompletionResponse{
		Content:      chatResp.Message.Content,
		InputTokens:  chatResp.PromptEvalCount,
		OutputTokens: chatResp.EvalCount,
		FinishReason: chatResp.DoneReason,
		ModelUsed:    chatResp.Model,
		Duration:     time.Since(startTime),
	}, nil
}

// Stream performs a streaming completion request
func (p *Provider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	startTime := time.Now()

	messages := convertMessages(req.Messages, req.SystemPrompt)

	chatReq := &ChatRequest{
		Model:    req.ModelID,
		Messages: messages,
		Options: &Options{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	var fullContent strings.Builder

	finalResp, err := p.client.ChatStream(ctx, chatReq, func(chunk *ChatResponse) error {
		fullContent.WriteString(chunk.Message.Content)
		if cb != nil {
			return cb(chunk.Message.Content)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if finalResp == nil {
		return &ports.CompletionResponse{
			Content:   fullContent.String(),
			ModelUsed: req.ModelID,
			Duration:  time.Since(startTime),
		}, nil
	}

	return &ports.CompletionResponse{
		Content:      fullContent.String(),
		InputTokens:  finalResp.PromptEvalCount,
		OutputTokens: finalResp.EvalCount,
		FinishReason: finalResp.DoneReason,
		ModelUsed:    finalResp.Model,
		Duration:     time.Since(startTime),
	}, nil
}

// HealthCheck checks the health of the provider for a specific model
func (p *Provider) HealthCheck(ctx context.Context, modelID string) (*ports.HealthStatus, error) {
	startTime := time.Now()

	err := p.client.Ping(ctx)
	latency := time.Since(startTime)
	now := time.Now()

	if err != nil {
		return &ports.HealthStatus{
			Healthy:     false,
			Message:     err.Error(),
			Latency:     latency,
			LastChecked: now,
		}, nil
	}

	if modelID != "" {
		available, err := p.SupportsModel(ctx, modelID)
		if err != nil {
			return &ports.HealthStatus{
				Healthy:     false,
				Message:     "server healthy but model check failed: " + err.Error(),
				Latency:     latency,
				LastChecked: now,
			}, nil
		}
		if !available {
			return &ports.HealthStatus{
				Healthy:     false,
				Message:     "server healthy but model not found: " + modelID,
				Latency:     latency,
				LastChecked: now,
			}, nil
		}
	}

	return &ports.HealthStatus{
		Healthy:     true,
		Message:     "ok",
		Latency:     latency,
		LastChecked: now,
	}, nil
}

// convertMessages converts port messages to Ollama chat messages
func convertMessages(messages []ports.Message, systemPrompt string) []ChatMessage {
	result := make([]ChatMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		result = append(result, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		result = append(result, ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return result
}

// normalizeModelID normalizes model IDs for comparison
// Ollama models can have tags like "llama2:latest" or just "llama2"
func normalizeModelID(modelID string) string {
	id := strings.ToLower(strings.TrimSpace(modelID))
	if !strings.Contains(id, ":") {
		id += ":latest"
	}
	return id
}

// Ensure Provider implements ProviderPort
var _ ports.ProviderPort = (*Provider)(nil)
