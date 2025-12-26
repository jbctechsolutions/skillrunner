// Package openai provides an adapter for the OpenAI Chat Completions API.
package openai

import "time"

// MessageRole represents the role of a message participant.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
)

// Message represents a single message in the chat conversation.
type Message struct {
	Role       MessageRole `json:"role"`
	Content    string      `json:"content,omitempty"`
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool/function call requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatCompletionRequest is the request body for the OpenAI Chat Completions API.
type ChatCompletionRequest struct {
	Model            string          `json:"model"`
	Messages         []Message       `json:"messages"`
	MaxTokens        *int            `json:"max_tokens,omitempty"`
	Temperature      *float32        `json:"temperature,omitempty"`
	TopP             *float32        `json:"top_p,omitempty"`
	N                *int            `json:"n,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	StreamOptions    *StreamOptions  `json:"stream_options,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	PresencePenalty  *float32        `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32        `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int  `json:"logit_bias,omitempty"`
	User             string          `json:"user,omitempty"`
	Seed             *int            `json:"seed,omitempty"`
	Tools            []Tool          `json:"tools,omitempty"`
	ToolChoice       any             `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormat `json:"response_format,omitempty"`
}

// StreamOptions contains options for streaming responses.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Tool represents a tool available to the model.
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function describes a function that can be called.
type Function struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// ResponseFormat specifies the format of the response.
type ResponseFormat struct {
	Type string `json:"type"` // "text" or "json_object"
}

// ChatCompletionResponse is the response body from the OpenAI Chat Completions API.
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finish_reason"`
	LogProbs     *LogProbs    `json:"logprobs,omitempty"`
}

// LogProbs contains log probability information.
type LogProbs struct {
	Content []TokenLogProb `json:"content,omitempty"`
}

// TokenLogProb contains log probability for a token.
type TokenLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

// Usage contains token usage information from the response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorResponse represents an error from the OpenAI API.
type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

// ErrorInfo contains detailed error information.
type ErrorInfo struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param,omitempty"`
	Code    *string `json:"code,omitempty"`
}

// StreamChunk represents a chunk in a streaming response.
type StreamChunk struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []StreamChoice `json:"choices"`
	Usage             *Usage         `json:"usage,omitempty"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
}

// StreamChoice represents a choice in a streaming response.
type StreamChoice struct {
	Index        int           `json:"index"`
	Delta        StreamDelta   `json:"delta"`
	FinishReason *FinishReason `json:"finish_reason,omitempty"`
	LogProbs     *LogProbs     `json:"logprobs,omitempty"`
}

// StreamDelta contains the incremental content in streaming responses.
type StreamDelta struct {
	Role      MessageRole `json:"role,omitempty"`
	Content   string      `json:"content,omitempty"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"`
}

// ModelsResponse is the response from the OpenAI models list endpoint.
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model represents an OpenAI model.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// Config contains configuration for the OpenAI client.
type Config struct {
	APIKey         string
	BaseURL        string
	Organization   string
	Timeout        time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

// DefaultConfig returns a Config with default values.
func DefaultConfig(apiKey string) Config {
	return Config{
		APIKey:         apiKey,
		BaseURL:        "https://api.openai.com/v1",
		Organization:   "",
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Second,
		RetryMaxDelay:  30 * time.Second,
	}
}

// Available OpenAI models.
const (
	// GPT-4 family
	ModelGPT4             = "gpt-4"
	ModelGPT4Turbo        = "gpt-4-turbo"
	ModelGPT4TurboPreview = "gpt-4-turbo-preview"
	ModelGPT4o            = "gpt-4o"
	ModelGPT4oMini        = "gpt-4o-mini"
	ModelGPT432k          = "gpt-4-32k"

	// GPT-3.5 family
	ModelGPT35Turbo         = "gpt-3.5-turbo"
	ModelGPT35Turbo16k      = "gpt-3.5-turbo-16k"
	ModelGPT35TurboInstruct = "gpt-3.5-turbo-instruct"

	// O1 family (reasoning models)
	ModelO1        = "o1"
	ModelO1Preview = "o1-preview"
	ModelO1Mini    = "o1-mini"
)

// SupportedModels returns the list of models supported by this adapter.
func SupportedModels() []string {
	return []string{
		ModelGPT4,
		ModelGPT4Turbo,
		ModelGPT4TurboPreview,
		ModelGPT4o,
		ModelGPT4oMini,
		ModelGPT432k,
		ModelGPT35Turbo,
		ModelGPT35Turbo16k,
		ModelGPT35TurboInstruct,
		ModelO1,
		ModelO1Preview,
		ModelO1Mini,
	}
}

// IsSupportedModel checks if a model ID is in the list of supported models.
func IsSupportedModel(modelID string) bool {
	for _, m := range SupportedModels() {
		if m == modelID {
			return true
		}
	}
	return false
}

// RateLimitInfo contains rate limit information from response headers.
type RateLimitInfo struct {
	LimitRequests     int       // x-ratelimit-limit-requests
	LimitTokens       int       // x-ratelimit-limit-tokens
	RemainingRequests int       // x-ratelimit-remaining-requests
	RemainingTokens   int       // x-ratelimit-remaining-tokens
	ResetRequests     time.Time // x-ratelimit-reset-requests
	ResetTokens       time.Time // x-ratelimit-reset-tokens
}
