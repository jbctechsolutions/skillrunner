package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// OpenAIProvider implements the Provider interface for OpenAI's Chat API
type OpenAIProvider struct {
	apiKey string
	client *Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKeyEnv string) (*OpenAIProvider, error) {
	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("%s environment variable not set", apiKeyEnv)
	}

	return &OpenAIProvider{
		apiKey: apiKey,
		client: NewClient(),
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Chat sends a chat request to OpenAI
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Convert model name to OpenAI format (remove "openai/" prefix if present)
	modelName := strings.TrimPrefix(req.Model, "openai/")

	// Convert messages to a single prompt for now (using completion interface)
	var prompt strings.Builder
	for _, msg := range req.Messages {
		switch msg.Role {
		case RoleSystem:
			prompt.WriteString("System: ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		case RoleUser:
			prompt.WriteString("User: ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		case RoleAssistant:
			prompt.WriteString("Assistant: ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		}
	}
	prompt.WriteString("Assistant: ")

	// Use the client's OpenAI completion
	completionReq := CompletionRequest{
		Model:       "openai/" + modelName,
		Prompt:      prompt.String(),
		MaxTokens:   req.MaxTokens,
		Temperature: float64(req.Temperature),
		Stream:      false,
	}

	resp, err := p.client.completeOpenAI(ctx, completionReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("openai completion failed: %w", err)
	}

	return ChatResponse{
		Content: resp.Content,
		Usage: Usage{
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
		},
	}, nil
}
