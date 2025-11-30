package types

import "time"

// Intent represents the type of action for a workflow step
type Intent string

const (
	IntentPlan Intent = "plan"
	IntentEdit Intent = "edit"
	IntentRun  Intent = "run"
)

// SkillType represents the execution mode of a skill
type SkillType string

const (
	SkillTypeMarketplace  SkillType = "marketplace"
	SkillTypeOrchestrated SkillType = "orchestrated"
	SkillTypeSimple       SkillType = "simple" // Legacy/backward compatibility
)

// PhaseStatus represents the execution status of a phase
type PhaseStatus string

const (
	PhaseStatusPending PhaseStatus = "pending"
	PhaseStatusRunning PhaseStatus = "running"
	PhaseStatusSuccess PhaseStatus = "success"
	PhaseStatusFailed  PhaseStatus = "failed"
	PhaseStatusSkipped PhaseStatus = "skipped"
	PhaseStatusCached  PhaseStatus = "cached"
)

// TaskType represents the type of task for model routing
type TaskType string

const (
	TaskTypeSummarization   TaskType = "summarization"
	TaskTypeExtraction      TaskType = "extraction"
	TaskTypeAnalysis        TaskType = "analysis"
	TaskTypeGeneration      TaskType = "generation"
	TaskTypeVerification    TaskType = "verification"
	TaskTypeReview          TaskType = "review"
	TaskTypeArchitecture    TaskType = "architecture"
	TaskTypeComplexAnalysis TaskType = "complex_analysis"
)

// SelectionStrategy represents model selection strategy
type SelectionStrategy string

const (
	SelectionStrategyCheapest    SelectionStrategy = "cheapest"
	SelectionStrategyFastest     SelectionStrategy = "fastest"
	SelectionStrategyBestQuality SelectionStrategy = "best_quality"
)

// FileOperation represents a file operation in a step
type FileOperation struct {
	Op      string `json:"op"`                // write, edit, delete
	Path    string `json:"path"`              // path to the file
	Content string `json:"content,omitempty"` // content for write operations
	Pattern string `json:"pattern,omitempty"` // pattern for search operations
}

// ContextItem represents a context item for a step
type ContextItem struct {
	Type   string `json:"type"`             // folder, file, mcp
	Source string `json:"source"`           // source identifier
	Filter string `json:"filter,omitempty"` // optional filter or pattern
}

// Step represents a single step in a skill workflow
type Step struct {
	Intent   Intent                 `json:"intent"`
	Model    string                 `json:"model"`
	Prompt   string                 `json:"prompt,omitempty"`
	Context  []ContextItem          `json:"context"`
	FileOps  []FileOperation        `json:"file_ops"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Envelope is the main structure for Skillrunner output
type Envelope struct {
	Version  string                 `json:"version"`
	Skill    string                 `json:"skill"`
	Request  string                 `json:"request"`
	Steps    []Step                 `json:"steps"`
	Metadata map[string]interface{} `json:"metadata"`
}

// RoutingConfig represents model routing configuration
type RoutingConfig struct {
	GenerationModel  string `yaml:"generation_model" json:"generation_model"`
	ReviewModel      string `yaml:"review_model" json:"review_model"`
	FallbackModel    string `yaml:"fallback_model" json:"fallback_model"`
	MaxContextTokens int    `yaml:"max_context_tokens" json:"max_context_tokens"`
	ChunkStrategy    string `yaml:"chunk_strategy" json:"chunk_strategy"`
}

// ContextStrategy represents context chunking strategy
type ContextStrategy struct {
	Type               string `yaml:"type" json:"type"`
	ChunkSize          int    `yaml:"chunk_size" json:"chunk_size"`
	Overlap            int    `yaml:"overlap" json:"overlap"`
	SummarizationModel string `yaml:"summarization_model" json:"summarization_model"`
}

// SkillConfig represents configuration for a skill
type SkillConfig struct {
	Name            string           `yaml:"name" json:"name"`
	Version         string           `yaml:"version" json:"version"`
	Description     string           `yaml:"description" json:"description"`
	DefaultModel    string           `yaml:"default_model" json:"default_model"`
	Routing         *RoutingConfig   `yaml:"routing,omitempty" json:"routing,omitempty"`
	ContextStrategy *ContextStrategy `yaml:"context_strategy,omitempty" json:"context_strategy,omitempty"`
}

// SystemStatus represents system status information
type SystemStatus struct {
	Version          string   `json:"version"`
	SkillCount       int      `json:"skill_count"`
	Workspace        string   `json:"workspace"`
	Ready            bool     `json:"ready"`
	ConfiguredModels []string `json:"configured_models"`
}

// RouterMetrics tracks routing performance and costs
type RouterMetrics struct {
	LocalCalls    int       `json:"local_calls"`
	CloudCalls    int       `json:"cloud_calls"`
	TotalTokens   int       `json:"total_tokens"`
	EstimatedCost float64   `json:"estimated_cost"`
	ElapsedTime   float64   `json:"elapsed_time"` // in seconds
	StartTime     time.Time `json:"start_time"`
}

// RouterResult represents the result of a routing operation
type RouterResult struct {
	Generation string        `json:"generation"`
	Review     string        `json:"review,omitempty"`
	Metrics    RouterMetrics `json:"metrics"`
}

// ModelProvider represents the model provider
type ModelProvider string

const (
	ModelProviderOllama    ModelProvider = "ollama"
	ModelProviderAnthropic ModelProvider = "anthropic"
	ModelProviderOpenAI    ModelProvider = "openai"
)

// ModelInfo contains information about a model
type ModelInfo struct {
	Name            string           `yaml:"name" json:"name"`
	Provider        ModelProvider    `yaml:"provider" json:"provider"`
	Endpoint        string           `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`       // For Ollama
	APIKeyEnv       string           `yaml:"api_key_env,omitempty" json:"api_key_env,omitempty"` // For cloud providers
	ContextWindow   int              `yaml:"context_window" json:"context_window"`
	CostPer1KInput  float64          `yaml:"cost_per_1k_input_tokens" json:"cost_per_1k_input_tokens"`
	CostPer1KOutput float64          `yaml:"cost_per_1k_output_tokens" json:"cost_per_1k_output_tokens"`
	Capabilities    []string         `yaml:"capabilities" json:"capabilities"`
	TaskTypes       []TaskType       `yaml:"task_types" json:"task_types"`
	Performance     ModelPerformance `yaml:"performance" json:"performance"`
}

// ModelPerformance contains performance metrics for a model
type ModelPerformance struct {
	TokensPerSecond  int `yaml:"tokens_per_second" json:"tokens_per_second"`
	TypicalLatencyMs int `yaml:"typical_latency_ms" json:"typical_latency_ms"`
}

// MarketplaceSkill represents a skill from the HuggingFace marketplace
type MarketplaceSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	SDK         string   `json:"sdk"`
	Likes       int      `json:"likes"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	URL         string   `json:"url,omitempty"`
	Runtime     string   `json:"runtime,omitempty"`
}

// MarketplaceResult represents the result of executing a marketplace skill
type MarketplaceResult struct {
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output"`
	Error   string                 `json:"error,omitempty"`
}

// Phase represents a single phase in an orchestrated skill
type Phase struct {
	ID             string                 `yaml:"id" json:"id"`
	Name           string                 `yaml:"name" json:"name"`
	TaskType       TaskType               `yaml:"task_type" json:"task_type"`
	DependsOn      []string               `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	PromptTemplate string                 `yaml:"prompt_template" json:"prompt_template"`
	PromptFile     string                 `yaml:"prompt_file,omitempty" json:"prompt_file,omitempty"`
	InputRequired  []string               `yaml:"input_required,omitempty" json:"input_required,omitempty"`
	InputOptional  []string               `yaml:"input_optional,omitempty" json:"input_optional,omitempty"`
	OutputKey      string                 `yaml:"output_key" json:"output_key"`
	Routing        *PhaseRouting          `yaml:"routing,omitempty" json:"routing,omitempty"`
	Condition      string                 `yaml:"condition,omitempty" json:"condition,omitempty"`
	MaxTokens      int                    `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
	Temperature    float64                `yaml:"temperature,omitempty" json:"temperature,omitempty"`
	Metadata       map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// PhaseRouting contains routing preferences for a phase
type PhaseRouting struct {
	PreferredModels    []string          `yaml:"preferred_models,omitempty" json:"preferred_models,omitempty"`
	FallbackModels     []string          `yaml:"fallback_models,omitempty" json:"fallback_models,omitempty"`
	SelectionStrategy  SelectionStrategy `yaml:"selection_strategy,omitempty" json:"selection_strategy,omitempty"`
	RequiresCapability []string          `yaml:"requires_capability,omitempty" json:"requires_capability,omitempty"`
}

// PhaseResult represents the result of executing a phase
type PhaseResult struct {
	PhaseID      string                 `json:"phase_id"`
	Status       PhaseStatus            `json:"status"`
	Output       string                 `json:"output"`
	ModelUsed    string                 `json:"model_used"`
	InputTokens  int                    `json:"input_tokens"`
	OutputTokens int                    `json:"output_tokens"`
	DurationMs   int64                  `json:"duration_ms"`
	Error        string                 `json:"error,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// OrchestratedSkill represents a multi-phase orchestrated skill
type OrchestratedSkill struct {
	Name        string                 `yaml:"name" json:"name"`
	Version     string                 `yaml:"version" json:"version"`
	Description string                 `yaml:"description" json:"description"`
	Type        SkillType              `yaml:"type" json:"type"`
	Phases      []Phase                `yaml:"phases" json:"phases"`
	Context     map[string]interface{} `yaml:"context,omitempty" json:"context,omitempty"`
	Routing     *RoutingConfig         `yaml:"routing,omitempty" json:"routing,omitempty"`
	Output      *OutputConfig          `yaml:"output,omitempty" json:"output,omitempty"`
}

// OutputConfig defines how skill output should be formatted
type OutputConfig struct {
	Format      string   `yaml:"format" json:"format"` // markdown, json, yaml
	IncludeKeys []string `yaml:"include_keys,omitempty" json:"include_keys,omitempty"`
	Template    string   `yaml:"template,omitempty" json:"template,omitempty"`
}
