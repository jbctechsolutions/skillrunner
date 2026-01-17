// Package workflow provides the workflow executor for skill execution.
package workflow

import (
	"context"
	"sync"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// PhaseStatus represents the execution state of a phase.
type PhaseStatus string

const (
	PhaseStatusPending   PhaseStatus = "pending"
	PhaseStatusRunning   PhaseStatus = "running"
	PhaseStatusCompleted PhaseStatus = "completed"
	PhaseStatusFailed    PhaseStatus = "failed"
	PhaseStatusSkipped   PhaseStatus = "skipped"
)

// PhaseResult contains the result of executing a single phase.
type PhaseResult struct {
	PhaseID      string
	PhaseName    string
	Status       PhaseStatus
	Output       string
	Error        error
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	InputTokens  int
	OutputTokens int
	ModelUsed    string
	CacheHit     bool    // Wave 10: Whether the result was served from cache
	Cost         float64 // Cost in USD for this phase execution
}

// ExecutionResult contains the aggregated results of executing a skill.
type ExecutionResult struct {
	SkillID      string
	SkillName    string
	Status       PhaseStatus
	PhaseResults map[string]*PhaseResult
	FinalOutput  string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	TotalTokens  int
	TotalCost    float64 // Total cost in USD across all phases
	Error        error
	// Wave 10: Cache statistics
	CacheHits   int // Number of phases served from cache
	CacheMisses int // Number of phases that required provider calls
}

// ExecutorConfig contains configuration options for the executor.
type ExecutorConfig struct {
	MaxParallel   int           // Maximum number of phases to execute in parallel
	Timeout       time.Duration // Overall timeout for skill execution
	MemoryContent string        // Memory content to inject into prompts (from MEMORY.md/CLAUDE.md)
}

// DefaultExecutorConfig returns the default executor configuration.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		MaxParallel:   4,
		Timeout:       5 * time.Minute,
		MemoryContent: "",
	}
}

// Executor orchestrates the execution of skill phases.
type Executor interface {
	Execute(ctx context.Context, skill *skill.Skill, input string) (*ExecutionResult, error)
}

// executor is the default implementation of Executor.
type executor struct {
	provider      ports.ProviderPort
	config        ExecutorConfig
	phaseExecutor *phaseExecutor
}

// NewExecutor creates a new workflow executor with the given provider and configuration.
func NewExecutor(provider ports.ProviderPort, config ExecutorConfig) Executor {
	if config.MaxParallel <= 0 {
		config.MaxParallel = DefaultExecutorConfig().MaxParallel
	}
	if config.Timeout <= 0 {
		config.Timeout = DefaultExecutorConfig().Timeout
	}

	return &executor{
		provider:      provider,
		config:        config,
		phaseExecutor: newPhaseExecutor(provider, config.MemoryContent),
	}
}

// Execute runs all phases of a skill in DAG order, executing parallel batches concurrently.
func (e *executor) Execute(ctx context.Context, s *skill.Skill, input string) (*ExecutionResult, error) {
	if s == nil {
		return nil, errors.NewError(errors.CodeValidation, "skill is required", nil)
	}

	// Validate the skill
	if err := s.Validate(); err != nil {
		return nil, errors.NewError(errors.CodeValidation, "invalid skill", err)
	}

	// Apply timeout to context
	ctx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	result := &ExecutionResult{
		SkillID:      s.ID(),
		SkillName:    s.Name(),
		Status:       PhaseStatusRunning,
		PhaseResults: make(map[string]*PhaseResult),
		StartTime:    time.Now(),
	}

	// Build DAG from phases
	phases := s.Phases()
	dag, err := workflow.NewDAG(phases)
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// Get parallel batches for execution
	batches, err := dag.GetParallelBatches()
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// Initialize all phases as pending
	for _, phase := range phases {
		result.PhaseResults[phase.ID] = &PhaseResult{
			PhaseID:   phase.ID,
			PhaseName: phase.Name,
			Status:    PhaseStatusPending,
		}
	}

	// Track outputs from previous phases for context
	phaseOutputs := make(map[string]string)
	phaseOutputs["_input"] = input

	// Execute batches sequentially, phases within each batch in parallel
	for _, batch := range batches {
		if err := e.executeBatch(ctx, dag, batch, result, phaseOutputs); err != nil {
			result.Status = PhaseStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)

			// Mark remaining phases as skipped
			e.markRemainingAsSkipped(result)

			// Return context errors to caller, but not phase errors
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			return result, nil
		}

		// Check for context cancellation
		if ctx.Err() != nil {
			result.Status = PhaseStatusFailed
			result.Error = ctx.Err()
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)

			// Mark remaining phases as skipped
			e.markRemainingAsSkipped(result)
			return result, ctx.Err()
		}
	}

	// Find the final output (from the last executed phase)
	result.FinalOutput = e.determineFinalOutput(dag, phases, phaseOutputs)

	// Aggregate token counts and costs
	for _, phaseResult := range result.PhaseResults {
		result.TotalTokens += phaseResult.InputTokens + phaseResult.OutputTokens
		result.TotalCost += phaseResult.Cost
	}

	result.Status = PhaseStatusCompleted
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// executeBatch executes a batch of phases in parallel with a concurrency limit.
func (e *executor) executeBatch(
	ctx context.Context,
	dag *workflow.DAG,
	batch []string,
	result *ExecutionResult,
	phaseOutputs map[string]string,
) error {
	if len(batch) == 0 {
		return nil
	}

	// Create a semaphore for limiting parallelism
	sem := make(chan struct{}, e.config.MaxParallel)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, phaseID := range batch {
		phase := dag.GetPhase(phaseID)
		if phase == nil {
			continue
		}

		wg.Add(1)
		go func(p *skill.Phase) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				if firstErr == nil {
					firstErr = ctx.Err()
				}
				mu.Unlock()
				return
			}

			// Build context from dependent phases (with lock to prevent data race)
			mu.Lock()
			dependencyOutputs := e.gatherDependencyOutputs(dag, p.ID, phaseOutputs)
			mu.Unlock()

			// Update status to running
			mu.Lock()
			result.PhaseResults[p.ID].Status = PhaseStatusRunning
			result.PhaseResults[p.ID].StartTime = time.Now()
			mu.Unlock()

			// Execute the phase
			phaseResult := e.phaseExecutor.Execute(ctx, p, dependencyOutputs)

			// Store result
			mu.Lock()
			result.PhaseResults[p.ID] = phaseResult
			if phaseResult.Status == PhaseStatusCompleted {
				phaseOutputs[p.ID] = phaseResult.Output
			} else if phaseResult.Error != nil && firstErr == nil {
				firstErr = phaseResult.Error
			}
			mu.Unlock()
		}(phase)
	}

	wg.Wait()

	return firstErr
}

// gatherDependencyOutputs collects outputs from all phases this phase depends on.
func (e *executor) gatherDependencyOutputs(dag *workflow.DAG, phaseID string, phaseOutputs map[string]string) map[string]string {
	deps := dag.GetDependencies(phaseID)
	outputs := make(map[string]string, len(deps)+1)

	// Always include the original input
	if input, ok := phaseOutputs["_input"]; ok {
		outputs["_input"] = input
	}

	// Add outputs from dependencies
	for _, depID := range deps {
		if output, ok := phaseOutputs[depID]; ok {
			outputs[depID] = output
		}
	}

	return outputs
}

// markRemainingAsSkipped marks all pending phases as skipped.
func (e *executor) markRemainingAsSkipped(result *ExecutionResult) {
	now := time.Now()
	for _, phaseResult := range result.PhaseResults {
		if phaseResult.Status == PhaseStatusPending || phaseResult.Status == PhaseStatusRunning {
			phaseResult.Status = PhaseStatusSkipped
			phaseResult.EndTime = now
			if phaseResult.StartTime.IsZero() {
				phaseResult.StartTime = now
			}
			phaseResult.Duration = phaseResult.EndTime.Sub(phaseResult.StartTime)
		}
	}
}

// determineFinalOutput determines the final output from the last phase(s) in the DAG.
// If there are multiple terminal phases, it concatenates their outputs.
func (e *executor) determineFinalOutput(dag *workflow.DAG, phases []skill.Phase, phaseOutputs map[string]string) string {
	// Find terminal phases (phases with no dependents)
	var terminalPhases []string
	for _, phase := range phases {
		dependents := dag.GetDependents(phase.ID)
		if len(dependents) == 0 {
			terminalPhases = append(terminalPhases, phase.ID)
		}
	}

	// If no terminal phases found, return empty
	if len(terminalPhases) == 0 {
		return ""
	}

	// If single terminal phase, return its output
	if len(terminalPhases) == 1 {
		return phaseOutputs[terminalPhases[0]]
	}

	// Multiple terminal phases - concatenate outputs
	var finalOutput string
	for i, phaseID := range terminalPhases {
		if output, ok := phaseOutputs[phaseID]; ok {
			if i > 0 {
				finalOutput += "\n\n"
			}
			finalOutput += output
		}
	}

	return finalOutput
}
