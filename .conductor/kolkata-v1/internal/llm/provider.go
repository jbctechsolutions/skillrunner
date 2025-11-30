package llm

import "context"

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleSystem    MessageRole = "system"
	RoleAssistant MessageRole = "assistant"
)

type ChatMessage struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

type ChatRequest struct {
	Model       string
	Messages    []ChatMessage
	MaxTokens   int
	Temperature float32
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type ChatResponse struct {
	Content string
	Usage   Usage
}

type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
