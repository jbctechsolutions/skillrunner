package ollama

import "time"

// DefaultBaseURL is the default Ollama API endpoint
const DefaultBaseURL = "http://localhost:11434"

// API endpoints
const (
	EndpointTags     = "/api/tags"
	EndpointChat     = "/api/chat"
	EndpointGenerate = "/api/generate"
)

// TagsResponse represents the response from GET /api/tags
type TagsResponse struct {
	Models []ModelInfo `json:"models"`
}

// ModelInfo contains information about a model
type ModelInfo struct {
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    Details   `json:"details"`
}

// Details contains model details
type Details struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  *Options      `json:"options,omitempty"`
}

// ChatMessage represents a message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Options for model configuration
type Options struct {
	Temperature float32 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Model              string      `json:"model"`
	CreatedAt          time.Time   `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	DoneReason         string      `json:"done_reason,omitempty"`
	TotalDuration      int64       `json:"total_duration,omitempty"`
	LoadDuration       int64       `json:"load_duration,omitempty"`
	PromptEvalCount    int         `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64       `json:"prompt_eval_duration,omitempty"`
	EvalCount          int         `json:"eval_count,omitempty"`
	EvalDuration       int64       `json:"eval_duration,omitempty"`
}

// GenerateRequest represents a text generation request
type GenerateRequest struct {
	Model    string   `json:"model"`
	Prompt   string   `json:"prompt"`
	System   string   `json:"system,omitempty"`
	Stream   bool     `json:"stream"`
	Options  *Options `json:"options,omitempty"`
	Template string   `json:"template,omitempty"`
	Context  []int    `json:"context,omitempty"`
}

// GenerateResponse represents a text generation response
type GenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	DoneReason         string    `json:"done_reason,omitempty"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error string `json:"error"`
}
