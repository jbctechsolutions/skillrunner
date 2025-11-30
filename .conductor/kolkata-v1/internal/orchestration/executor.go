package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	contextmgr "github.com/jbctechsolutions/skillrunner/internal/context"
	srerrors "github.com/jbctechsolutions/skillrunner/internal/errors"
	"github.com/jbctechsolutions/skillrunner/internal/llm"
	"github.com/jbctechsolutions/skillrunner/internal/metrics"
	profilerouter "github.com/jbctechsolutions/skillrunner/internal/routing"
	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// StreamCallback is called for each chunk of streaming output
type StreamCallback func(phaseID string, chunk string) error

// PhaseExecutor executes skill phases with dependency management
type PhaseExecutor struct {
	skill               *types.OrchestratedSkill
	dag                 *DAG
	profileRouter       *profilerouter.Router
	routingConfig       *profilerouter.RoutingConfig
	llmClient           *llm.Client
	contextManager      *contextmgr.Manager
	metricsStorage      *metrics.Storage
	results             map[string]*types.PhaseResult
	outputKeyMap        map[string]string // Maps output_key to phase ID
	userContext         map[string]interface{}
	configPath          string
	profile             string
	globalModelOverride string // Model override from CLI --model flag
	phaseCosts          []metrics.PhaseCost
	costComputer        *metrics.CostComputer
	streamCallback      StreamCallback
}

// NewPhaseExecutor creates a new phase executor
func NewPhaseExecutor(
	skill *types.OrchestratedSkill,
	userContext map[string]interface{},
	configPath string,
	profile string,
	streamCallback StreamCallback,
	modelOverride string, // Optional model override from CLI
) (*PhaseExecutor, error) {
	// Build DAG from phases
	dag, err := NewDAG(skill.Phases)
	if err != nil {
		return nil, srerrors.Wrap(err, srerrors.ErrorCodeSkillInvalid, "failed to build execution DAG").
			WithContext("skill", skill.Name)
	}

	// Default to balanced profile if not specified
	if profile == "" {
		profile = "balanced"
	}

	// Create profile router
	routingConfig, err := profilerouter.LoadRoutingConfig(configPath)
	if err != nil {
		return nil, srerrors.Wrap(err, srerrors.ErrorCodeConfigInvalid, "failed to load routing config").
			WithContext("config_path", configPath)
	}

	profileRouter, err := profilerouter.NewRouter(routingConfig)
	if err != nil {
		return nil, srerrors.Wrap(err, srerrors.ErrorCodeConfigInvalid, "failed to create profile router").
			WithContext("config_path", configPath)
	}

	// Build model pricing map for cost computer
	modelPricing := make(map[string]metrics.ModelPricing)
	for _, model := range routingConfig.Models {
		modelPricing[model.ID] = metrics.ModelPricing{
			CostPer1KInput:  model.ProfileCostPer1KInput,
			CostPer1KOutput: model.ProfileCostPer1KOutput,
		}
	}

	costSimConfig := profileRouter.GetCostSimulationConfig()
	costComputer := metrics.NewCostComputer(
		costSimConfig.PremiumModelID,
		costSimConfig.CheapModelID,
		modelPricing,
	)

	// Create LLM client
	llmClient := llm.NewClient()

	// Create context manager
	contextMgr := contextmgr.NewManager(contextmgr.DefaultConfig(), llmClient)

	// Create metrics storage
	metricsStorage, err := metrics.NewStorage()
	if err != nil {
		// Log warning but continue without metrics
		fmt.Printf("Warning: Failed to create metrics storage: %v\n", err)
		metricsStorage = nil
	}

	// Build output_key to phase ID mapping
	outputKeyMap := make(map[string]string)
	for _, phase := range skill.Phases {
		if phase.OutputKey != "" {
			outputKeyMap[phase.OutputKey] = phase.ID
		}
	}

	return &PhaseExecutor{
		skill:               skill,
		dag:                 dag,
		profileRouter:       profileRouter,
		routingConfig:       routingConfig,
		outputKeyMap:        outputKeyMap,
		llmClient:           llmClient,
		contextManager:      contextMgr,
		metricsStorage:      metricsStorage,
		results:             make(map[string]*types.PhaseResult),
		userContext:         userContext,
		configPath:          configPath,
		profile:             profile,
		globalModelOverride: modelOverride,
		phaseCosts:          []metrics.PhaseCost{},
		costComputer:        costComputer,
		streamCallback:      streamCallback,
	}, nil
}

// Execute runs all phases in dependency order
func (pe *PhaseExecutor) Execute(ctx context.Context) (map[string]*types.PhaseResult, error) {
	// Get execution batches
	batches, err := pe.dag.GetBatches()
	if err != nil {
		return nil, fmt.Errorf("get execution batches: %w", err)
	}

	fmt.Printf("Executing %d phases in %d batches\n\n", len(pe.skill.Phases), len(batches))

	// Execute batches sequentially, phases in batch concurrently
	for batchIdx, batch := range batches {
		fmt.Printf("=== Batch %d/%d (%d phases) ===\n", batchIdx+1, len(batches), len(batch))

		// Execute all phases in this batch concurrently
		results := make(chan *types.PhaseResult, len(batch))
		errors := make(chan error, len(batch))

		for _, phaseID := range batch {
			go func(pid string) {
				result, err := pe.executePhase(ctx, pid)
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(phaseID)
		}

		// Collect results
		for i := 0; i < len(batch); i++ {
			select {
			case result := <-results:
				pe.results[result.PhaseID] = result
				pe.printPhaseResult(result)
			case err := <-errors:
				return nil, fmt.Errorf("phase execution failed: %w", err)
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		fmt.Println()
	}

	// Check if any phase failed
	for _, result := range pe.results {
		if result.Status == types.PhaseStatusFailed {
			return nil, fmt.Errorf("phase '%s' failed: %s", result.PhaseID, result.Error)
		}
	}

	return pe.results, nil
}

// executePhase executes a single phase
func (pe *PhaseExecutor) executePhase(ctx context.Context, phaseID string) (*types.PhaseResult, error) {
	node := pe.dag.Nodes[phaseID]
	phase := node.Phase
	startTime := time.Now()

	// Check if phase should be skipped based on condition
	if phase.Condition != "" && !pe.evaluateCondition(phase.Condition) {
		return &types.PhaseResult{
			PhaseID:   phaseID,
			Status:    types.PhaseStatusSkipped,
			Timestamp: startTime,
		}, nil
	}

	// Prepare phase input
	phaseInput, err := pe.preparePhaseInput(phase)
	if err != nil {
		return &types.PhaseResult{
			PhaseID:   phaseID,
			Status:    types.PhaseStatusFailed,
			Error:     fmt.Sprintf("prepare input: %v", err),
			Timestamp: startTime,
		}, nil
	}

	// Render prompt
	prompt, err := pe.renderPrompt(phase, phaseInput)
	if err != nil {
		return &types.PhaseResult{
			PhaseID:   phaseID,
			Status:    types.PhaseStatusFailed,
			Error:     fmt.Sprintf("render prompt: %v", err),
			Timestamp: startTime,
		}, nil
	}

	// Select model for this phase and get provider info
	// Priority: 1) Global override (CLI --model), 2) Phase-specific models, 3) Profile routing
	var modelInfo *types.ModelInfo
	var profileProvider llm.Provider
	var profileModelConfig *profilerouter.ModelConfig
	var modelID string
	var selectedModel string

	// Check for global model override first (from CLI --model flag)
	if pe.globalModelOverride != "" {
		// Try to resolve as model ID first, then as provider/model format
		modelConfig, err := pe.profileRouter.GetModelConfig(pe.globalModelOverride)
		if err == nil {
			// Found as model ID - get provider
			provider, _, err := pe.profileRouter.RouteToModel(pe.globalModelOverride)
			if err == nil {
				profileProvider = provider
				profileModelConfig = modelConfig
				modelID = modelConfig.ID
				selectedModel = fmt.Sprintf("%s/%s", modelConfig.Provider, modelConfig.Model)
				modelInfo = &types.ModelInfo{
					ContextWindow: 200000, // Default large context window for cloud models
				}
			}
		}
		// If not found as model ID, treat as provider/model format
		if selectedModel == "" {
			selectedModel = pe.globalModelOverride
			// Parse provider/model format to get provider
			parts := strings.Split(pe.globalModelOverride, "/")
			if len(parts) == 2 {
				providerName := parts[0]
				// Find provider from routing config
				for _, p := range pe.routingConfig.Providers {
					if p.Name == providerName {
						var err error
						switch p.Type {
						case "anthropic":
							profileProvider, err = llm.NewAnthropicProvider(p.APIKeyEnv)
						case "openai":
							profileProvider, err = llm.NewOpenAIProvider(p.APIKeyEnv)
						case "google":
							profileProvider, err = llm.NewGoogleProvider(p.APIKeyEnv)
						case "ollama":
							profileProvider, err = llm.NewOllamaProvider(p.APIKeyEnv)
						}
						if err == nil {
							modelID = pe.globalModelOverride
							selectedModel = pe.globalModelOverride
							modelInfo = &types.ModelInfo{
								ContextWindow: 200000,
							}
							break
						}
					}
				}
			}
		}
	}

	// If no global override, check for phase-specific models
	if selectedModel == "" && phase.Routing != nil && len(phase.Routing.PreferredModels) > 0 {
		for _, preferredModelID := range phase.Routing.PreferredModels {
			provider, modelConfig, err := pe.profileRouter.RouteToModel(preferredModelID)
			if err == nil {
				profileProvider = provider
				// Store modelConfig in a local variable to avoid dangling pointer
				localModelConfig := modelConfig
				profileModelConfig = &localModelConfig
				modelID = modelConfig.ID
				selectedModel = fmt.Sprintf("%s/%s", modelConfig.Provider, modelConfig.Model)
				modelInfo = &types.ModelInfo{
					ContextWindow: 200000,
				}
				fmt.Printf("  Using phase-specific model: %s\n", selectedModel)
				break
			}
		}
		// If preferred models failed, try fallback models
		if selectedModel == "" && len(phase.Routing.FallbackModels) > 0 {
			for _, fallbackModelID := range phase.Routing.FallbackModels {
				provider, modelConfig, err := pe.profileRouter.RouteToModel(fallbackModelID)
				if err == nil {
					profileProvider = provider
					// Store modelConfig in a local variable to avoid dangling pointer
					localModelConfig := modelConfig
					profileModelConfig = &localModelConfig
					modelID = modelConfig.ID
					selectedModel = fmt.Sprintf("%s/%s", modelConfig.Provider, modelConfig.Model)
					modelInfo = &types.ModelInfo{
						ContextWindow: 200000,
					}
					fmt.Printf("  Using fallback model: %s\n", selectedModel)
					break
				}
			}
		}
	}

	// Fall back to profile-based routing if no override or phase-specific model
	if selectedModel == "" {
		provider, modelConfig, err := pe.profileRouter.Route(pe.profile)
		if err != nil {
			return &types.PhaseResult{
				PhaseID:   phaseID,
				Status:    types.PhaseStatusFailed,
				Error:     fmt.Sprintf("profile router failed: %v", err),
				Timestamp: startTime,
			}, nil
		}
		profileProvider = provider
		// Store modelConfig in a local variable to avoid dangling pointer
		localModelConfig := modelConfig
		profileModelConfig = &localModelConfig
		modelID = modelConfig.ID
		selectedModel = fmt.Sprintf("%s/%s", modelConfig.Provider, modelConfig.Model)
		modelInfo = &types.ModelInfo{
			ContextWindow: 200000, // Default large context window for cloud models
		}
	}

	// Determine max input tokens (reserve space for output)
	maxTokens := phase.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4000
	}

	// Calculate max input tokens: 80% of context window minus output tokens
	maxInputTokens := int(float64(modelInfo.ContextWindow)*0.8) - maxTokens

	// Optimize prompt if it exceeds context window
	optimizedPrompt, wasOptimized, err := pe.contextManager.PrepareContext(ctx, prompt, maxInputTokens)
	if err != nil {
		// Log warning but continue with original prompt
		fmt.Printf("  Warning: context optimization failed: %v\n", err)
		optimizedPrompt = prompt
	} else if wasOptimized {
		fmt.Printf("  Context optimized (exceeded %d tokens)\n", maxInputTokens)
	}

	prompt = optimizedPrompt

	// Execute with LLM
	temperature := phase.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	var resp *llm.CompletionResponse

	if profileProvider != nil {
		// Use profile provider interface
		chatReq := llm.ChatRequest{
			Model: profileModelConfig.Model,
			Messages: []llm.ChatMessage{
				{
					Role:    llm.RoleUser,
					Content: prompt,
				},
			},
			MaxTokens:   maxTokens,
			Temperature: float32(temperature),
		}

		var chatResp llm.ChatResponse
		chatResp, err = profileProvider.Chat(ctx, chatReq)
		if err != nil {
			duration := time.Since(startTime)
			// Record failed execution in metrics
			if pe.metricsStorage != nil {
				record := metrics.ExecutionRecord{
					Timestamp:    startTime,
					Skill:        pe.skill.Name,
					Model:        modelID,
					InputTokens:  0,
					OutputTokens: 0,
					Cost:         0.0,
					DurationMs:   duration.Milliseconds(),
					Success:      false,
					Error:        err.Error(),
				}
				go func() {
					_ = pe.metricsStorage.RecordExecution(record)
				}()
			}

			return &types.PhaseResult{
				PhaseID:   phaseID,
				Status:    types.PhaseStatusFailed,
				Error:     fmt.Sprintf("llm execution: %v", err),
				Timestamp: startTime,
			}, nil
		}

		// Convert ChatResponse to CompletionResponse
		resp = &llm.CompletionResponse{
			Content:      chatResp.Content,
			InputTokens:  chatResp.Usage.InputTokens,
			OutputTokens: chatResp.Usage.OutputTokens,
			Model:        modelID,
			Provider:     profileProvider.Name(),
		}
	} else {
		// Use existing LLM client
		req := llm.CompletionRequest{
			Model:       selectedModel,
			Prompt:      prompt,
			MaxTokens:   maxTokens,
			Temperature: temperature,
			Stream:      false, // Will be set to true if streaming callback is provided
		}

		// Use streaming if callback is provided
		if pe.streamCallback != nil {
			// Create a callback wrapper that includes the phase ID
			wrappedCallback := func(chunk string) error {
				return pe.streamCallback(phaseID, chunk)
			}

			resp, err = pe.llmClient.StreamCompletion(ctx, req, wrappedCallback)
		} else {
			resp, err = pe.llmClient.Complete(ctx, req)
		}

		if err != nil {
			duration := time.Since(startTime)
			// Record failed execution in metrics
			if pe.metricsStorage != nil {
				record := metrics.ExecutionRecord{
					Timestamp:    startTime,
					Skill:        pe.skill.Name,
					Model:        selectedModel,
					InputTokens:  0,
					OutputTokens: 0,
					Cost:         0.0,
					DurationMs:   duration.Milliseconds(),
					Success:      false,
					Error:        err.Error(),
				}
				go func() {
					_ = pe.metricsStorage.RecordExecution(record)
				}()
			}

			return &types.PhaseResult{
				PhaseID:   phaseID,
				Status:    types.PhaseStatusFailed,
				Error:     fmt.Sprintf("llm execution: %v", err),
				Timestamp: startTime,
			}, nil
		}
	}

	duration := time.Since(startTime)
	result := &types.PhaseResult{
		PhaseID:      phaseID,
		Status:       types.PhaseStatusSuccess,
		Output:       resp.Content,
		ModelUsed:    modelID,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		DurationMs:   duration.Milliseconds(),
		Timestamp:    startTime,
	}

	// Track costs for cost simulation if profile routing is enabled
	if pe.costComputer != nil && profileModelConfig != nil {
		actualCost := pe.costComputer.ComputePhaseCost(modelID, resp.InputTokens, resp.OutputTokens)
		phaseCost := metrics.PhaseCost{
			PhaseID:      phaseID,
			ModelID:      modelID,
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
			ActualCost:   actualCost,
		}
		pe.phaseCosts = append(pe.phaseCosts, phaseCost)
	}

	// Record metrics if storage is available
	if pe.metricsStorage != nil {
		var cost float64
		if pe.costComputer != nil && profileModelConfig != nil {
			cost = pe.costComputer.ComputePhaseCost(modelID, resp.InputTokens, resp.OutputTokens)
		}

		record := metrics.ExecutionRecord{
			Timestamp:    startTime,
			Skill:        pe.skill.Name,
			Model:        modelID,
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
			Cost:         cost,
			DurationMs:   duration.Milliseconds(),
			Success:      true,
		}
		// Record asynchronously to avoid blocking
		go func() {
			if err := pe.metricsStorage.RecordExecution(record); err != nil {
				// Silently fail metrics recording to avoid disrupting execution
				_ = err
			}
		}()
	}

	return result, nil
}

// preparePhaseInput prepares input data for a phase
func (pe *PhaseExecutor) preparePhaseInput(phase *types.Phase) (map[string]interface{}, error) {
	input := make(map[string]interface{})

	// Add user context
	for k, v := range pe.userContext {
		input[k] = v
	}

	// Add outputs from dependent phases
	for _, depID := range phase.DependsOn {
		if result, ok := pe.results[depID]; ok {
			// Store dependency output with phase ID as key
			input[depID] = result.Output

			// Also store by output_key if the phase has one
			// Find the phase to get its output_key
			for _, p := range pe.skill.Phases {
				if p.ID == depID && p.OutputKey != "" {
					input[p.OutputKey] = result.Output
					break
				}
			}
		}
	}

	// Check required inputs
	for _, reqKey := range phase.InputRequired {
		if _, ok := input[reqKey]; !ok {
			return nil, fmt.Errorf("required input missing: %s", reqKey)
		}
	}

	return input, nil
}

// renderPrompt renders the phase prompt with input data
func (pe *PhaseExecutor) renderPrompt(phase *types.Phase, input map[string]interface{}) (string, error) {
	template := phase.PromptTemplate

	// Simple template rendering (replace {{key}} with values)
	rendered := template
	for key, value := range input {
		placeholder := fmt.Sprintf("{{%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		rendered = strings.ReplaceAll(rendered, placeholder, valueStr)
	}

	return rendered, nil
}

// evaluateCondition evaluates a phase condition with support for expressions, comparisons, and boolean logic
func (pe *PhaseExecutor) evaluateCondition(condition string) bool {
	if condition == "" {
		return true // Empty condition means always execute
	}

	// Handle simple boolean strings
	condition = strings.TrimSpace(condition)
	if condition == "true" {
		return true
	}
	if condition == "false" {
		return false
	}

	// Parse and evaluate the condition
	result, err := pe.parseCondition(condition)
	if err != nil {
		// If parsing fails, log and default to false for safety
		fmt.Printf("Warning: Failed to parse condition '%s': %v. Phase will be skipped.\n", condition, err)
		return false
	}

	return result
}

// parseCondition parses a condition string and evaluates it
// Supports:
// - Template variables: {{phase1.status}}, {{phase1.output}}, {{user.request}}
// - Comparisons: ==, !=, >, <, >=, <=
// - Boolean operators: &&, ||, !
// - String literals: "success", "failed"
// - Boolean literals: true, false
func (pe *PhaseExecutor) parseCondition(condition string) (bool, error) {
	// Remove whitespace
	condition = strings.TrimSpace(condition)

	// Handle negation
	if strings.HasPrefix(condition, "!") {
		result, err := pe.parseCondition(condition[1:])
		return !result, err
	}

	// Handle parentheses (simple case: wrap entire expression)
	if strings.HasPrefix(condition, "(") && strings.HasSuffix(condition, ")") {
		return pe.parseCondition(condition[1 : len(condition)-1])
	}

	// Handle boolean operators (check for && and ||)
	if idx := strings.Index(condition, " && "); idx > 0 {
		left, err := pe.parseCondition(condition[:idx])
		if err != nil {
			return false, err
		}
		right, err := pe.parseCondition(condition[idx+4:])
		if err != nil {
			return false, err
		}
		return left && right, nil
	}

	if idx := strings.Index(condition, " || "); idx > 0 {
		left, err := pe.parseCondition(condition[:idx])
		if err != nil {
			return false, err
		}
		right, err := pe.parseCondition(condition[idx+4:])
		if err != nil {
			return false, err
		}
		return left || right, nil
	}

	// Handle comparisons
	for _, op := range []string{" == ", " != ", " >= ", " <= ", " > ", " < "} {
		if idx := strings.Index(condition, op); idx > 0 {
			left := strings.TrimSpace(condition[:idx])
			right := strings.TrimSpace(condition[idx+len(op):])

			leftVal := pe.resolveValue(left)
			rightVal := pe.resolveValue(right)

			switch op {
			case " == ":
				return leftVal == rightVal, nil
			case " != ":
				return leftVal != rightVal, nil
			case " > ":
				return pe.compareValues(leftVal, rightVal) > 0, nil
			case " < ":
				return pe.compareValues(leftVal, rightVal) < 0, nil
			case " >= ":
				return pe.compareValues(leftVal, rightVal) >= 0, nil
			case " <= ":
				return pe.compareValues(leftVal, rightVal) <= 0, nil
			}
		}
	}

	// If no operators found, treat as a single value
	value := pe.resolveValue(condition)
	if value == "true" {
		return true, nil
	}
	if value == "false" {
		return false, nil
	}
	// Non-empty string is truthy
	return value != "", nil
}

// resolveValue resolves a template variable or returns the literal value
func (pe *PhaseExecutor) resolveValue(expr string) string {
	expr = strings.TrimSpace(expr)

	// Remove quotes if present
	if (strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`)) ||
		(strings.HasPrefix(expr, `'`) && strings.HasSuffix(expr, `'`)) {
		return expr[1 : len(expr)-1]
	}

	// Check if it's a template variable {{...}}
	if strings.HasPrefix(expr, "{{") && strings.HasSuffix(expr, "}}") {
		path := strings.TrimSpace(expr[2 : len(expr)-2])
		return pe.getVariableValue(path)
	}

	// Return as-is (could be a literal value)
	return expr
}

// getVariableValue gets a value from phase results or user context
// Supports paths like:
// - phase1.status
// - phase1.output
// - user.request
// - phase1.metadata.key
// - greeting (looks up by output_key)
func (pe *PhaseExecutor) getVariableValue(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return ""
	}

	// Check if it's a phase result by phase ID (e.g., "phase1.output")
	if len(parts) >= 2 && strings.HasPrefix(parts[0], "phase") {
		phaseID := parts[0]
		result, ok := pe.results[phaseID]
		if !ok {
			return ""
		}

		switch parts[1] {
		case "status":
			return string(result.Status)
		case "output":
			return result.Output
		case "error":
			return result.Error
		case "model_used":
			return result.ModelUsed
		case "metadata":
			if len(parts) > 2 && result.Metadata != nil {
				if val, ok := result.Metadata[parts[2]]; ok {
					return fmt.Sprintf("%v", val)
				}
			}
		}
		return ""
	}

	// Check if it's a direct output_key lookup (e.g., "greeting", "tip")
	// This handles simple variable names that match output_key values
	if phaseID, ok := pe.outputKeyMap[path]; ok {
		if result, ok := pe.results[phaseID]; ok {
			return result.Output
		}
	}

	// Check user context
	if len(parts) >= 1 && parts[0] == "user" {
		if len(parts) > 1 {
			key := strings.Join(parts[1:], ".")
			if val, ok := pe.userContext[key]; ok {
				return fmt.Sprintf("%v", val)
			}
		}
		return ""
	}

	// Try direct lookup in user context
	if val, ok := pe.userContext[path]; ok {
		return fmt.Sprintf("%v", val)
	}

	return ""
}

// compareValues compares two string values (tries numeric comparison first, then string)
func (pe *PhaseExecutor) compareValues(left, right string) int {
	// Try numeric comparison
	leftNum, leftErr := parseNumber(left)
	rightNum, rightErr := parseNumber(right)
	if leftErr == nil && rightErr == nil {
		if leftNum < rightNum {
			return -1
		}
		if leftNum > rightNum {
			return 1
		}
		return 0
	}

	// Fall back to string comparison
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

// parseNumber attempts to parse a string as a number
func parseNumber(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// printPhaseResult prints the result of a phase execution
func (pe *PhaseExecutor) printPhaseResult(result *types.PhaseResult) {
	statusIcon := "✓"
	if result.Status == types.PhaseStatusFailed {
		statusIcon = "✗"
	} else if result.Status == types.PhaseStatusSkipped {
		statusIcon = "○"
	}

	fmt.Printf("%s Phase: %s\n", statusIcon, result.PhaseID)
	fmt.Printf("  Status:  %s\n", result.Status)

	if result.Status == types.PhaseStatusSuccess {
		fmt.Printf("  Model:   %s\n", result.ModelUsed)
		fmt.Printf("  Tokens:  %d in, %d out\n", result.InputTokens, result.OutputTokens)
		fmt.Printf("  Duration: %dms\n", result.DurationMs)

		// Show truncated output
		output := result.Output
		if len(output) > 200 {
			output = output[:200] + "..."
		}
		fmt.Printf("  Output:  %s\n", strings.ReplaceAll(output, "\n", "\n           "))
	}

	if result.Error != "" {
		fmt.Printf("  Error:   %s\n", result.Error)
	}
}

// GetResults returns all phase results
func (pe *PhaseExecutor) GetResults() map[string]*types.PhaseResult {
	return pe.results
}

// GetFinalOutput returns the output from the final phase(s)
func (pe *PhaseExecutor) GetFinalOutput() string {
	// Get leaf phases (phases with no dependents)
	leaves := pe.dag.GetLeafPhases()

	if len(leaves) == 0 {
		return ""
	}

	var rawOutput string

	// If single leaf, return its output
	if len(leaves) == 1 {
		if result, ok := pe.results[leaves[0]]; ok {
			rawOutput = result.Output
		}
	} else {
		// Multiple leaves - combine outputs
		var outputs []string
		for _, leafID := range leaves {
			if result, ok := pe.results[leafID]; ok && result.Status == types.PhaseStatusSuccess {
				outputs = append(outputs, fmt.Sprintf("=== %s ===\n%s", leafID, result.Output))
			}
		}
		rawOutput = strings.Join(outputs, "\n\n")
	}

	// Process template variables in the output
	return pe.processTemplateVariables(rawOutput)
}

// processTemplateVariables processes template variables like {{key}} in a string
// by replacing them with values from phase results or user context
func (pe *PhaseExecutor) processTemplateVariables(text string) string {
	// Find all template variables in the format {{...}}
	// We'll use a simple approach: find {{...}} patterns and replace them
	result := text
	start := 0

	for {
		// Find the next {{ pattern
		openIdx := strings.Index(result[start:], "{{")
		if openIdx == -1 {
			break
		}
		openIdx += start

		// Find the matching }}
		closeIdx := strings.Index(result[openIdx:], "}}")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx

		// Extract the variable path
		varPath := strings.TrimSpace(result[openIdx+2 : closeIdx])

		// Resolve the variable value
		varValue := pe.getVariableValue(varPath)

		// Replace the template variable with the resolved value
		result = result[:openIdx] + varValue + result[closeIdx+2:]

		// Continue searching from after the replacement
		start = openIdx + len(varValue)
	}

	return result
}

// GetCostSummary returns the cost summary if cost computer is available
func (pe *PhaseExecutor) GetCostSummary() *metrics.RunCostSummary {
	if pe.costComputer == nil {
		return nil
	}
	summary := pe.costComputer.SummarizeRun(pe.phaseCosts)
	return &summary
}
