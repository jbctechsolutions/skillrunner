package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	baseURL string
	client  *Client
}

// NewOllamaProvider creates a new Ollama provider
// apiKeyEnv is ignored for Ollama (no API key needed)
func NewOllamaProvider(apiKeyEnv string) (*OllamaProvider, error) {
	// Get Ollama URL from environment or use default
	baseURL := os.Getenv("OLLAMA_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaProvider{
		baseURL: baseURL,
		client:  NewClient(),
	}, nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Chat sends a chat request to Ollama
func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Convert model name to Ollama format (remove "ollama/" prefix if present)
	modelName := strings.TrimPrefix(req.Model, "ollama/")

	// Convert messages to a single prompt (simple concatenation)
	// For better results, you might want to format this better
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

	// Use the client's Ollama completion
	// Note: The client uses hardcoded localhost:11434, but we'll use it for now
	// The model name should include "ollama/" prefix for the client to route correctly
	completionReq := CompletionRequest{
		Model:       "ollama/" + modelName,
		Prompt:      prompt.String(),
		MaxTokens:   req.MaxTokens,
		Temperature: float64(req.Temperature),
		Stream:      false,
	}

	resp, err := p.client.Complete(ctx, completionReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("ollama completion failed: %w", err)
	}

	return ChatResponse{
		Content: resp.Content,
		Usage: Usage{
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
		},
	}, nil
}
