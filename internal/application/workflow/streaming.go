// Package workflow provides the workflow executor for skill execution.
package workflow

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// StreamEventType represents the type of streaming event.
type StreamEventType string

const (
	// EventPhaseStarted indicates a phase has begun execution.
	EventPhaseStarted StreamEventType = "phase_started"
	// EventPhaseProgress indicates streaming progress for a phase.
	EventPhaseProgress StreamEventType = "phase_progress"
	// EventPhaseCompleted indicates a phase has finished successfully.
	EventPhaseCompleted StreamEventType = "phase_completed"
	// EventPhaseFailed indicates a phase has failed.
	EventPhaseFailed StreamEventType = "phase_failed"
	// EventTokenUpdate indicates a token count update.
	EventTokenUpdate StreamEventType = "token_update"
	// EventWorkflowStarted indicates the workflow has begun.
	EventWorkflowStarted StreamEventType = "workflow_started"
	// EventWorkflowCompleted indicates the workflow has finished.
	EventWorkflowCompleted StreamEventType = "workflow_completed"
)

// StreamEvent represents a real-time update during workflow execution.
type StreamEvent struct {
	Type         StreamEventType
	PhaseID      string
	PhaseName    string
	Content      string // For progress events, the streamed chunk
	Error        error
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Timestamp    time.Time
	PhaseIndex   int // Current phase index (1-based)
	TotalPhases  int // Total number of phases
}

// StreamCallback is called for each streaming event during execution.
type StreamCallback func(event StreamEvent) error

// StreamingExecutor orchestrates skill execution with real-time streaming support.
type StreamingExecutor interface {
	// ExecuteWithStreaming runs all phases with streaming callbacks.
	ExecuteWithStreaming(ctx context.Context, skill *skill.Skill, input string, callback StreamCallback) (*ExecutionResult, error)
}

// streamingExecutor is the default implementation of StreamingExecutor.
type streamingExecutor struct {
	provider               ports.ProviderPort
	config                 ExecutorConfig
	streamingPhaseExecutor *streamingPhaseExecutor
}

// NewStreamingExecutor creates a new streaming workflow executor.
func NewStreamingExecutor(provider ports.ProviderPort, config ExecutorConfig) StreamingExecutor {
	if config.MaxParallel <= 0 {
		config.MaxParallel = DefaultExecutorConfig().MaxParallel
	}
	if config.Timeout <= 0 {
		config.Timeout = DefaultExecutorConfig().Timeout
	}

	return &streamingExecutor{
		provider:               provider,
		config:                 config,
		streamingPhaseExecutor: newStreamingPhaseExecutor(provider, config.MemoryContent),
	}
}

// ExecuteWithStreaming runs all phases of a skill with streaming callbacks.
func (e *streamingExecutor) ExecuteWithStreaming(ctx context.Context, s *skill.Skill, input string, callback StreamCallback) (*ExecutionResult, error) {
	if s == nil {
		return nil, errors.NewError(errors.CodeValidation, "skill is required", nil)
	}

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

	// Emit workflow started event
	if callback != nil {
		_ = callback(StreamEvent{
			Type:        EventWorkflowStarted,
			TotalPhases: len(phases),
			Timestamp:   time.Now(),
		})
	}

	// Track outputs from previous phases and token counts
	phaseOutputs := make(map[string]string)
	phaseOutputs["_input"] = input
	var totalInputTokens, totalOutputTokens int64
	phaseCounter := 0

	// Execute batches sequentially, phases within each batch in parallel
	for _, batch := range batches {
		if err := e.executeBatchWithStreaming(ctx, dag, batch, result, phaseOutputs, callback, &totalInputTokens, &totalOutputTokens, &phaseCounter, len(phases)); err != nil {
			result.Status = PhaseStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)

			// Mark remaining phases as skipped
			e.markRemainingAsSkipped(result)

			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			return result, nil
		}

		if ctx.Err() != nil {
			result.Status = PhaseStatusFailed
			result.Error = ctx.Err()
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			e.markRemainingAsSkipped(result)
			return result, ctx.Err()
		}
	}

	// Find the final output
	result.FinalOutput = e.determineFinalOutput(dag, phases, phaseOutputs)

	// Aggregate token counts
	result.TotalTokens = int(atomic.LoadInt64(&totalInputTokens) + atomic.LoadInt64(&totalOutputTokens))

	result.Status = PhaseStatusCompleted
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Emit workflow completed event
	if callback != nil {
		_ = callback(StreamEvent{
			Type:         EventWorkflowCompleted,
			TotalPhases:  len(phases),
			TotalTokens:  result.TotalTokens,
			InputTokens:  int(totalInputTokens),
			OutputTokens: int(totalOutputTokens),
			Timestamp:    time.Now(),
		})
	}

	return result, nil
}

// executeBatchWithStreaming executes a batch of phases with streaming support.
func (e *streamingExecutor) executeBatchWithStreaming(
	ctx context.Context,
	dag *workflow.DAG,
	batch []string,
	result *ExecutionResult,
	phaseOutputs map[string]string,
	callback StreamCallback,
	totalInputTokens, totalOutputTokens *int64,
	phaseCounter *int,
	totalPhases int,
) error {
	if len(batch) == 0 {
		return nil
	}

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

			// Get phase index
			mu.Lock()
			*phaseCounter++
			currentPhaseIndex := *phaseCounter
			dependencyOutputs := e.gatherDependencyOutputs(dag, p.ID, phaseOutputs)
			result.PhaseResults[p.ID].Status = PhaseStatusRunning
			result.PhaseResults[p.ID].StartTime = time.Now()
			mu.Unlock()

			// Emit phase started event
			if callback != nil {
				_ = callback(StreamEvent{
					Type:        EventPhaseStarted,
					PhaseID:     p.ID,
					PhaseName:   p.Name,
					PhaseIndex:  currentPhaseIndex,
					TotalPhases: totalPhases,
					Timestamp:   time.Now(),
				})
			}

			// Create streaming callback for this phase
			phaseCallback := func(chunk string, inputToks, outputToks int) error {
				if callback != nil {
					// Report progress with current phase token estimates
					// Final token counts are updated when phase completes
					currentTotal := atomic.LoadInt64(totalInputTokens) + atomic.LoadInt64(totalOutputTokens)

					return callback(StreamEvent{
						Type:         EventPhaseProgress,
						PhaseID:      p.ID,
						PhaseName:    p.Name,
						Content:      chunk,
						InputTokens:  inputToks,
						OutputTokens: outputToks,
						TotalTokens:  int(currentTotal) + inputToks + outputToks,
						PhaseIndex:   currentPhaseIndex,
						TotalPhases:  totalPhases,
						Timestamp:    time.Now(),
					})
				}
				return nil
			}

			// Execute the phase with streaming
			phaseResult := e.streamingPhaseExecutor.ExecuteWithStreaming(ctx, p, dependencyOutputs, phaseCallback)

			// Store result
			mu.Lock()
			result.PhaseResults[p.ID] = phaseResult

			if phaseResult.Status == PhaseStatusCompleted {
				phaseOutputs[p.ID] = phaseResult.Output
				atomic.AddInt64(totalInputTokens, int64(phaseResult.InputTokens))
				atomic.AddInt64(totalOutputTokens, int64(phaseResult.OutputTokens))

				// Emit phase completed event
				if callback != nil {
					_ = callback(StreamEvent{
						Type:         EventPhaseCompleted,
						PhaseID:      p.ID,
						PhaseName:    p.Name,
						InputTokens:  phaseResult.InputTokens,
						OutputTokens: phaseResult.OutputTokens,
						TotalTokens:  int(atomic.LoadInt64(totalInputTokens) + atomic.LoadInt64(totalOutputTokens)),
						PhaseIndex:   currentPhaseIndex,
						TotalPhases:  totalPhases,
						Timestamp:    time.Now(),
					})

					// Emit token update event
					_ = callback(StreamEvent{
						Type:         EventTokenUpdate,
						InputTokens:  int(atomic.LoadInt64(totalInputTokens)),
						OutputTokens: int(atomic.LoadInt64(totalOutputTokens)),
						TotalTokens:  int(atomic.LoadInt64(totalInputTokens) + atomic.LoadInt64(totalOutputTokens)),
						Timestamp:    time.Now(),
					})
				}
			} else if phaseResult.Error != nil {
				if firstErr == nil {
					firstErr = phaseResult.Error
				}

				// Emit phase failed event
				if callback != nil {
					_ = callback(StreamEvent{
						Type:        EventPhaseFailed,
						PhaseID:     p.ID,
						PhaseName:   p.Name,
						Error:       phaseResult.Error,
						PhaseIndex:  currentPhaseIndex,
						TotalPhases: totalPhases,
						Timestamp:   time.Now(),
					})
				}
			}
			mu.Unlock()
		}(phase)
	}

	wg.Wait()

	return firstErr
}

// gatherDependencyOutputs collects outputs from dependent phases.
func (e *streamingExecutor) gatherDependencyOutputs(dag *workflow.DAG, phaseID string, phaseOutputs map[string]string) map[string]string {
	deps := dag.GetDependencies(phaseID)
	outputs := make(map[string]string, len(deps)+1)

	if input, ok := phaseOutputs["_input"]; ok {
		outputs["_input"] = input
	}

	for _, depID := range deps {
		if output, ok := phaseOutputs[depID]; ok {
			outputs[depID] = output
		}
	}

	return outputs
}

// markRemainingAsSkipped marks all pending phases as skipped.
func (e *streamingExecutor) markRemainingAsSkipped(result *ExecutionResult) {
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

// determineFinalOutput determines the final output from terminal phases.
func (e *streamingExecutor) determineFinalOutput(dag *workflow.DAG, phases []skill.Phase, phaseOutputs map[string]string) string {
	var terminalPhases []string
	for _, phase := range phases {
		dependents := dag.GetDependents(phase.ID)
		if len(dependents) == 0 {
			terminalPhases = append(terminalPhases, phase.ID)
		}
	}

	if len(terminalPhases) == 0 {
		return ""
	}

	if len(terminalPhases) == 1 {
		return phaseOutputs[terminalPhases[0]]
	}

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
