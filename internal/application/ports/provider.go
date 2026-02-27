package ports

import (
	"context"
	"encoding/json"
	"time"
)

// ProviderInfo contains provider metadata
type ProviderInfo struct {
	Name        string
	Description string
	BaseURL     string
	IsLocal     bool
}

// ToolCall represents a tool invocation returned by the LLM.
type ToolCall struct {
	ID        string         // tool_use block id from the provider
	Name      string         // full tool name (mcp__server__tool format)
	Arguments map[string]any // parsed input arguments
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	ToolCallID string // matches ToolCall.ID
	Content    string // serialised result
	IsError    bool   // true if the tool returned an error
}

// Message represents a chat message.
// For assistant messages that invoked tools, ToolCalls is populated and Content may be empty.
// For user messages that carry tool results, ToolResults is populated and Content may be empty.
type Message struct {
	Role        string       // system, user, assistant
	Content     string       // text content
	ToolCalls   []ToolCall   // populated for assistant tool_use messages
	ToolResults []ToolResult // populated for user tool_result messages
}

// Tool represents a tool that can be called by the LLM.
type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"input_schema"`
	DeferLoading bool            `json:"defer_loading,omitempty"` // For Tool Search Tool support
}

// CompletionRequest is the input for LLM completion
type CompletionRequest struct {
	ModelID      string
	Messages     []Message
	MaxTokens    int
	Temperature  float32
	SystemPrompt string
	Tools        []Tool // Optional tools for function calling
}

// FinishReason constants for LLM completion stop reasons.
const (
	FinishReasonEndTurn   = "end_turn"
	FinishReasonToolUse   = "tool_use"
	FinishReasonMaxTokens = "max_tokens"
)

// CompletionResponse is the output from LLM completion
type CompletionResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	FinishReason string
	ModelUsed    string
	Duration     time.Duration
	ToolCalls    []ToolCall // populated when FinishReason == FinishReasonToolUse
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
