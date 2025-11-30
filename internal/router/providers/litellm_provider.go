package providers

import (
	"context"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/router/litellm"
	"github.com/jbctechsolutions/skillrunner/internal/router/types"
)

// LiteLLMProvider implements the Provider interface for LiteLLM
type LiteLLMProvider struct {
	client *litellm.Client
}

// NewLiteLLMProvider creates a new LiteLLM provider
func NewLiteLLMProvider(baseURL, apiKey string) *LiteLLMProvider {
	return &LiteLLMProvider{
		client: litellm.NewClient(baseURL, apiKey),
	}
}

// Complete sends a completion request through LiteLLM
func (p *LiteLLMProvider) Complete(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	return p.client.Complete(ctx, req)
}

// Name returns the name of the provider
func (p *LiteLLMProvider) Name() string {
	return "litellm"
}

// IsAvailable checks if the LiteLLM provider is available
func (p *LiteLLMProvider) IsAvailable(ctx context.Context) bool {
	// Try a simple health check by making a minimal request
	// For now, we'll do a simple check - in production, you might want
	// to check a health endpoint or make a minimal API call
	req := &types.CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "test",
			},
		},
	}

	_, err := p.client.Complete(ctx, req)
	// If we get a non-network error (like auth error), the service is available
	// If we get a network error, it's not available
	return err == nil || !isNetworkError(err)
}

// isNetworkError checks if an error is a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	networkErrors := []string{
		"connection refused",
		"no such host",
		"timeout",
		"network",
		"dial tcp",
		"context deadline exceeded",
	}
	for _, ne := range networkErrors {
		if strings.Contains(errStr, ne) {
			return true
		}
	}
	return false
}
