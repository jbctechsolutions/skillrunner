package config

// ModelDefaults defines default models for different task types
type ModelDefaults struct {
	CodeGeneration ModelConfig `yaml:"code_generation"`
	CodeReview     ModelConfig `yaml:"code_review"`
	Architecture   ModelConfig `yaml:"architecture"`
	Documentation  ModelConfig `yaml:"documentation"`
	Analysis       ModelConfig `yaml:"analysis"`
	Refactoring    ModelConfig `yaml:"refactoring"`
}

// ModelConfig defines model configuration for a task type
type ModelConfig struct {
	GenerationModel string `yaml:"generation_model"`
	ReviewModel     string `yaml:"review_model"`
	FallbackModel   string `yaml:"fallback_model"`
}

// DefaultModelDefaults returns sensible defaults for different task types
func DefaultModelDefaults() *ModelDefaults {
	return &ModelDefaults{
		CodeGeneration: ModelConfig{
			GenerationModel: "ollama/deepseek-coder-v2:16b",
			ReviewModel:     "anthropic/claude-3-5-sonnet-20241022",
			FallbackModel:   "anthropic/claude-3-haiku-20240307",
		},
		CodeReview: ModelConfig{
			GenerationModel: "ollama/deepseek-coder-v2:16b",
			ReviewModel:     "anthropic/claude-3-5-sonnet-20241022",
			FallbackModel:   "anthropic/claude-3-haiku-20240307",
		},
		Architecture: ModelConfig{
			GenerationModel: "ollama/qwen2.5-coder:32b",
			ReviewModel:     "anthropic/claude-3-5-sonnet-20241022",
			FallbackModel:   "anthropic/claude-3-opus-20240229",
		},
		Documentation: ModelConfig{
			GenerationModel: "ollama/llama3.2",
			ReviewModel:     "anthropic/claude-3-haiku-20240307",
			FallbackModel:   "anthropic/claude-3-haiku-20240307",
		},
		Analysis: ModelConfig{
			GenerationModel: "ollama/qwen2.5-coder:32b",
			ReviewModel:     "anthropic/claude-3-5-sonnet-20241022",
			FallbackModel:   "anthropic/claude-3-haiku-20240307",
		},
		Refactoring: ModelConfig{
			GenerationModel: "ollama/deepseek-coder-v2:16b",
			ReviewModel:     "anthropic/claude-3-5-sonnet-20241022",
			FallbackModel:   "anthropic/claude-3-haiku-20240307",
		},
	}
}

// GetModelConfig returns model config for a task type
func (m *ModelDefaults) GetModelConfig(taskType string) ModelConfig {
	switch taskType {
	case "code-generation", "code-gen":
		return m.CodeGeneration
	case "code-review", "review":
		return m.CodeReview
	case "architecture", "arch":
		return m.Architecture
	case "documentation", "docs":
		return m.Documentation
	case "analysis":
		return m.Analysis
	case "refactoring", "refactor":
		return m.Refactoring
	default:
		// Default to code generation
		return m.CodeGeneration
	}
}
