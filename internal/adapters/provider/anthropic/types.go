// Package anthropic provides an adapter for the Anthropic Claude API.
package anthropic

import "time"

// MessageRole represents the role of a message participant.
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// ContentType represents the type of content in a message.
type ContentType string

const (
	ContentTypeText ContentType = "text"
)

// ContentBlock represents a content block in a message.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MessageContent can be either a string or an array of content blocks.
// For simplicity, we use content blocks array format.
type MessageContent []ContentBlock

// Message represents a single message in the conversation.
type Message struct {
	Role    MessageRole    `json:"role"`
	Content MessageContent `json:"content"`
}

// MessagesRequest is the request body for the Anthropic Messages API.
type MessagesRequest struct {
	Model         string    `json:"model"`
	Messages      []Message `json:"messages"`
	MaxTokens     int       `json:"max_tokens"`
	System        string    `json:"system,omitempty"`
	Temperature   *float32  `json:"temperature,omitempty"`
	TopP          *float32  `json:"top_p,omitempty"`
	TopK          *int      `json:"top_k,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Stream        bool      `json:"stream,omitempty"`
}

// Usage contains token usage information from the response.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StopReason indicates why the model stopped generating.
type StopReason string

const (
	StopReasonEndTurn      StopReason = "end_turn"
	StopReasonMaxTokens    StopReason = "max_tokens"
	StopReasonStopSequence StopReason = "stop_sequence"
)

// MessagesResponse is the response body from the Anthropic Messages API.
type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         MessageRole    `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   StopReason     `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence,omitempty"`
	Usage        Usage          `json:"usage"`
}

// ErrorResponse represents an error from the Anthropic API.
type ErrorResponse struct {
	Type  string    `json:"type"`
	Error ErrorInfo `json:"error"`
}

// ErrorInfo contains detailed error information.
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StreamEventType represents the type of SSE event in streaming.
type StreamEventType string

const (
	EventMessageStart      StreamEventType = "message_start"
	EventContentBlockStart StreamEventType = "content_block_start"
	EventPing              StreamEventType = "ping"
	EventContentBlockDelta StreamEventType = "content_block_delta"
	EventContentBlockStop  StreamEventType = "content_block_stop"
	EventMessageDelta      StreamEventType = "message_delta"
	EventMessageStop       StreamEventType = "message_stop"
	EventError             StreamEventType = "error"
)

// StreamEvent represents a server-sent event from the streaming API.
type StreamEvent struct {
	Type         StreamEventType `json:"type"`
	Message      *StreamMessage  `json:"message,omitempty"`
	Index        *int            `json:"index,omitempty"`
	ContentBlock *ContentBlock   `json:"content_block,omitempty"`
	Delta        *StreamDelta    `json:"delta,omitempty"`
	Usage        *StreamUsage    `json:"usage,omitempty"`
}

// StreamMessage contains the message metadata in streaming responses.
type StreamMessage struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	Role         MessageRole `json:"role"`
	Content      []any       `json:"content"`
	Model        string      `json:"model"`
	StopReason   *StopReason `json:"stop_reason,omitempty"`
	StopSequence *string     `json:"stop_sequence,omitempty"`
	Usage        *Usage      `json:"usage,omitempty"`
}

// StreamDelta contains the incremental content in streaming responses.
type StreamDelta struct {
	Type         string      `json:"type,omitempty"`
	Text         string      `json:"text,omitempty"`
	StopReason   *StopReason `json:"stop_reason,omitempty"`
	StopSequence *string     `json:"stop_sequence,omitempty"`
}

// StreamUsage contains token usage in streaming message_delta events.
type StreamUsage struct {
	OutputTokens int `json:"output_tokens"`
}

// Config contains configuration for the Anthropic client.
type Config struct {
	APIKey     string
	BaseURL    string
	Version    string
	Timeout    time.Duration
	MaxRetries int
}

// DefaultConfig returns a Config with default values.
func DefaultConfig(apiKey string) Config {
	return Config{
		APIKey:     apiKey,
		BaseURL:    "https://api.anthropic.com/v1",
		Version:    "2023-06-01",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}

// Available Claude models.
const (
	ModelClaude3Opus    = "claude-3-opus-20240229"
	ModelClaude35Sonnet = "claude-3-5-sonnet-20241022"
	ModelClaude35Haiku  = "claude-3-5-haiku-20241022"
	ModelClaude3Sonnet  = "claude-3-sonnet-20240229"
	ModelClaude3Haiku   = "claude-3-haiku-20240307"
)

// SupportedModels returns the list of models supported by this adapter.
func SupportedModels() []string {
	return []string{
		ModelClaude3Opus,
		ModelClaude35Sonnet,
		ModelClaude35Haiku,
		ModelClaude3Sonnet,
		ModelClaude3Haiku,
	}
}
