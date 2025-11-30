package providers

import (
	"context"

	"github.com/jbctechsolutions/skillrunner/internal/router/types"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Complete sends a completion request and returns the response
	Complete(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error)

	// Name returns the name of the provider
	Name() string

	// IsAvailable checks if the provider is available
	IsAvailable(ctx context.Context) bool
}
