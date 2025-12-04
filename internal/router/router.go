package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/types"
	"gopkg.in/yaml.v3"
)

// ModelRouter handles intelligent model selection and routing
type ModelRouter struct {
	models      map[string]*types.ModelInfo
	routing     *RoutingConfig
	healthCache map[string]time.Time
	preferLocal bool
	ollamaURL   string // Default Ollama endpoint for dynamic models
}

// RoutingConfig contains routing rules from config file
type RoutingConfig struct {
	PreferLocal          bool                                `yaml:"prefer_local"`
	LocalTimeoutSeconds  int                                 `yaml:"local_timeout_seconds"`
	CloudFallbackEnabled bool                                `yaml:"cloud_fallback_enabled"`
	ByTaskType           map[types.TaskType]*TaskTypeRouting `yaml:"by_task_type"`
	CostOptimization     *CostOptimizationConfig             `yaml:"cost_optimization"`
	QualityRequirements  *QualityRequirementsConfig          `yaml:"quality_requirements"`
}

// TaskTypeRouting contains routing for a specific task type
type TaskTypeRouting struct {
	Preferred []string `yaml:"preferred"`
	Fallback  []string `yaml:"fallback"`
}

// CostOptimizationConfig contains cost optimization settings
type CostOptimizationConfig struct {
	PreferLocalUnderTokens int `yaml:"prefer_local_under_tokens"`
	RequireCloudOverTokens int `yaml:"require_cloud_over_tokens"`
	MaxLocalAttempts       int `yaml:"max_local_attempts"`
}

// QualityRequirementsConfig contains quality thresholds
type QualityRequirementsConfig struct {
	MinReviewScore       float64 `yaml:"min_review_score"`
	EscalateToCloudBelow float64 `yaml:"escalate_to_cloud_below"`
}

// Config represents the full configuration file structure
type Config struct {
	Models  map[string]*types.ModelInfo `yaml:"models"`
	Routing *RoutingConfig              `yaml:"routing"`
	Router  *RouterConfig               `yaml:"router"` // Router settings including ollama_url
}

// RouterConfig contains router-level settings
type RouterConfig struct {
	OllamaURL string `yaml:"ollama_url"`
	AutoStart bool   `yaml:"auto_start"`
}

// NewModelRouter creates a new model router from config file
func NewModelRouter(configPath string) (*ModelRouter, error) {
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Note: This router is deprecated. Use internal/routing for profile-based routing instead.
	// This code is kept for backward compatibility but should not be used in new code.
	if config.Routing == nil {
		return nil, fmt.Errorf("legacy routing configuration not found - use profile-based routing instead")
	}

	// Get Ollama URL from config, use default if not specified
	ollamaURL := "http://localhost:11434" // Default
	if config.Router != nil && config.Router.OllamaURL != "" {
		ollamaURL = config.Router.OllamaURL
	}

	// Initialize models map if nil
	if config.Models == nil {
		config.Models = make(map[string]*types.ModelInfo)
	}

	return &ModelRouter{
		models:      config.Models,
		routing:     config.Routing,
		healthCache: make(map[string]time.Time),
		preferLocal: config.Routing.PreferLocal,
		ollamaURL:   ollamaURL,
	}, nil
}

// SelectModel selects the best model for a task type
func (mr *ModelRouter) SelectModel(
	ctx context.Context,
	taskType types.TaskType,
) (string, error) {
	// Get routing for task type
	taskRouting := mr.routing.ByTaskType[taskType]
	if taskRouting == nil {
		return "", fmt.Errorf("no routing configuration for task type: %s", taskType)
	}

	// Build candidate list
	candidates := append([]string{}, taskRouting.Preferred...)
	if mr.routing.CloudFallbackEnabled {
		candidates = append(candidates, taskRouting.Fallback...)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no candidate models for task type: %s", taskType)
	}

	// Filter by health
	healthyCandidates := []string{}
	for _, modelName := range candidates {
		if mr.isHealthy(ctx, modelName) {
			healthyCandidates = append(healthyCandidates, modelName)
		}
	}

	if len(healthyCandidates) == 0 {
		return "", fmt.Errorf("no healthy models available for task type: %s", taskType)
	}

	// Apply local preference if enabled
	if mr.preferLocal {
		// Try to find a local model first
		for _, modelName := range healthyCandidates {
			model := mr.models[modelName]
			if model != nil && model.Provider == types.ModelProviderOllama {
				return modelName, nil
			}
		}
	}

	// Return first healthy model
	return healthyCandidates[0], nil
}

// CheckHealth checks if a model is healthy and available (public method)
func (mr *ModelRouter) CheckHealth(ctx context.Context, modelName string) bool {
	return mr.isHealthy(ctx, modelName)
}

// isHealthy checks if a model is healthy and available
func (mr *ModelRouter) isHealthy(ctx context.Context, modelName string) bool {
	// Check cache first (5 minute TTL)
	if lastCheck, ok := mr.healthCache[modelName]; ok {
		if time.Since(lastCheck) < 5*time.Minute {
			return true
		}
	}

	model := mr.models[modelName]

	// Handle Ollama models dynamically if not in config
	if model == nil && isOllamaModel(modelName) {
		// Extract model name without provider prefix
		modelID := extractModelID(modelName)
		model = &types.ModelInfo{
			Name:     modelID,
			Provider: types.ModelProviderOllama,
			Endpoint: mr.ollamaURL,
		}
	}

	if model == nil {
		return false
	}

	// Perform health check based on provider
	// Supported providers: Ollama (local), Anthropic (cloud), OpenAI (cloud)
	healthy := false
	switch model.Provider {
	case types.ModelProviderOllama:
		healthy = mr.checkOllamaHealth(ctx, model)
	case types.ModelProviderAnthropic:
		healthy = mr.checkAnthropicHealth(model)
	case types.ModelProviderOpenAI:
		healthy = mr.checkOpenAIHealth(model)
	default:
		// Unknown provider, assume unhealthy
		healthy = false
	}

	// Update cache if healthy
	if healthy {
		mr.healthCache[modelName] = time.Now()
	}

	return healthy
}

// isOllamaModel checks if a model name refers to an Ollama model
func isOllamaModel(modelName string) bool {
	return len(modelName) > 7 && modelName[:7] == "ollama/"
}

// extractModelID extracts the model ID from a fully qualified model name
// e.g., "ollama/qwen2.5:14b" -> "qwen2.5:14b"
func extractModelID(modelName string) string {
	if isOllamaModel(modelName) {
		return modelName[7:] // Remove "ollama/" prefix
	}
	return modelName
}

// checkOllamaHealth checks if Ollama model is available
func (mr *ModelRouter) checkOllamaHealth(ctx context.Context, model *types.ModelInfo) bool {
	if model.Endpoint == "" {
		return false
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Check Ollama /api/tags endpoint to get list of available models
	req, err := http.NewRequestWithContext(ctx, "GET", model.Endpoint+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Parse response to check if specific model exists
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If we can't parse, fall back to just checking service availability
		return true
	}

	// Check if the specific model is in the list
	modelName := model.Name
	for _, m := range result.Models {
		if m.Name == modelName {
			return true
		}
	}

	// Model not found in Ollama
	return false
}

// checkAnthropicHealth checks if Anthropic API key is configured
func (mr *ModelRouter) checkAnthropicHealth(model *types.ModelInfo) bool {
	if model.APIKeyEnv == "" {
		return false
	}
	apiKey := os.Getenv(model.APIKeyEnv)
	return apiKey != ""
}

// checkOpenAIHealth checks if OpenAI API key is configured
func (mr *ModelRouter) checkOpenAIHealth(model *types.ModelInfo) bool {
	if model.APIKeyEnv == "" {
		return false
	}
	apiKey := os.Getenv(model.APIKeyEnv)
	return apiKey != ""
}

// EstimateCost estimates the cost of using a model
func (mr *ModelRouter) EstimateCost(
	modelName string,
	inputTokens int,
	outputTokens int,
) float64 {
	model := mr.models[modelName]
	if model == nil {
		return 0.0
	}

	inputCost := (float64(inputTokens) / 1000.0) * model.CostPer1KInput
	outputCost := (float64(outputTokens) / 1000.0) * model.CostPer1KOutput

	return inputCost + outputCost
}

// GetModelInfo returns information about a model
func (mr *ModelRouter) GetModelInfo(modelName string) (*types.ModelInfo, error) {
	model := mr.models[modelName]
	if model == nil {
		return nil, fmt.Errorf("model not found: %s", modelName)
	}
	return model, nil
}

// ListModels returns all available models
func (mr *ModelRouter) ListModels() []*types.ModelInfo {
	models := make([]*types.ModelInfo, 0, len(mr.models))
	for _, model := range mr.models {
		models = append(models, model)
	}
	return models
}

// SetPreferLocal sets the local preference
func (mr *ModelRouter) SetPreferLocal(prefer bool) {
	mr.preferLocal = prefer
}

// GetPreferLocal returns the local preference
func (mr *ModelRouter) GetPreferLocal() bool {
	return mr.preferLocal
}
