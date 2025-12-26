package ports

import (
	"context"
	"time"
)

// ProviderInfo contains provider metadata
type ProviderInfo struct {
	Name        string
	Description string
	BaseURL     string
	IsLocal     bool
}

// Message represents a chat message
type Message struct {
	Role    string // system, user, assistant
	Content string
}

// CompletionRequest is the input for LLM completion
type CompletionRequest struct {
	ModelID      string
	Messages     []Message
	MaxTokens    int
	Temperature  float32
	SystemPrompt string
}

// CompletionResponse is the output from LLM completion
type CompletionResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	FinishReason string
	ModelUsed    string
	Duration     time.Duration
}

// StreamCallback for streaming responses
type StreamCallback func(chunk string) error

// HealthStatus for provider health checks
type HealthStatus struct {
	Healthy     bool
	Message     string
	Latency     time.Duration
	LastChecked time.Time
}

// ProviderPort is the main interface for LLM providers
type ProviderPort interface {
	Info() ProviderInfo
	ListModels(ctx context.Context) ([]string, error)
	SupportsModel(ctx context.Context, modelID string) (bool, error)
	IsAvailable(ctx context.Context, modelID string) (bool, error)
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest, cb StreamCallback) (*CompletionResponse, error)
	HealthCheck(ctx context.Context, modelID string) (*HealthStatus, error)
}
