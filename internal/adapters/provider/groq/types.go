// Package groq provides an adapter for the Groq API.
// Groq uses an OpenAI-compatible API format.
package groq

import "time"

// DefaultBaseURL is the default Groq API endpoint.
const DefaultBaseURL = "https://api.groq.com/openai/v1"

// API endpoints
const (
	EndpointChatCompletions = "/chat/completions"
	EndpointModels          = "/models"
)

// MessageRole represents the role of a message participant.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// Message represents a single message in the chat conversation.
type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

// ChatCompletionRequest is the request body for Groq chat completions.
type ChatCompletionRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	MaxTokens        int       `json:"max_tokens,omitempty"`
	Temperature      *float32  `json:"temperature,omitempty"`
	TopP             *float32  `json:"top_p,omitempty"`
	N                *int      `json:"n,omitempty"`
	Stream           bool      `json:"stream,omitempty"`
	Stop             []string  `json:"stop,omitempty"`
	PresencePenalty  *float32  `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32  `json:"frequency_penalty,omitempty"`
	User             string    `json:"user,omitempty"`
}

// Usage contains token usage information from the response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
	FinishReasonStop   FinishReason = "stop"
	FinishReasonLength FinishReason = "length"
)

// Choice represents a single completion choice in the response.
type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	Delta        *Message     `json:"delta,omitempty"`
	FinishReason FinishReason `json:"finish_reason,omitempty"`
}

// ChatCompletionResponse is the response body from Groq chat completions.
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// StreamChoice represents a choice in a streaming response chunk.
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        Message      `json:"delta"`
	FinishReason FinishReason `json:"finish_reason,omitempty"`
}

// ChatCompletionChunk represents a streaming response chunk.
type ChatCompletionChunk struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []StreamChoice `json:"choices"`
	Usage             *Usage         `json:"usage,omitempty"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
}

// ErrorResponse represents an error from the Groq API.
type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

// ErrorInfo contains detailed error information.
type ErrorInfo struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// Model represents model information from the /models endpoint.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse is the response from the /models endpoint.
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Config contains configuration for the Groq client.
type Config struct {
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

// DefaultConfig returns a Config with default values.
func DefaultConfig(apiKey string) Config {
	return Config{
		APIKey:     apiKey,
		BaseURL:    DefaultBaseURL,
		Timeout:    60 * time.Second,
		MaxRetries: 3,
	}
}

// Available Groq models.
const (
	ModelLlama31_70BVersatile = "llama-3.1-70b-versatile"
	ModelLlama31_8BInstant    = "llama-3.1-8b-instant"
	ModelMixtral8x7B_32768    = "mixtral-8x7b-32768"
	ModelGemma2_9BIt          = "gemma2-9b-it"
	ModelLlama3_70B_8192      = "llama3-70b-8192"
	ModelLlama3_8B_8192       = "llama3-8b-8192"
)

// SupportedModels returns the list of models supported by this adapter.
func SupportedModels() []string {
	return []string{
		ModelLlama31_70BVersatile,
		ModelLlama31_8BInstant,
		ModelMixtral8x7B_32768,
		ModelGemma2_9BIt,
		ModelLlama3_70B_8192,
		ModelLlama3_8B_8192,
	}
}
