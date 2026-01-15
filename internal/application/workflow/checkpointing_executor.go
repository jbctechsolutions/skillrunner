// Package workflow provides the workflow executor for skill execution.
package workflow

import (
	"context"
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
	domainSkill "github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// CheckpointConfig contains configuration for checkpoint-aware execution.
type CheckpointConfig struct {
	// Enabled enables checkpoint persistence.
	Enabled bool

	// Port is the storage port for checkpoints (required if Enabled).
	Port ports.WorkflowCheckpointPort

	// Resume attempts to resume from an existing checkpoint if available.
	Resume bool

	// ExecutionID is the correlation ID for this execution.
	// If empty and Resume is true, will try to find existing checkpoint.
	// If empty and Resume is false, a new UUID will be generated.
	ExecutionID string

	// MachineID identifies the machine running this execution.
	MachineID string

	// Logger for checkpoint operations (optional).
	Logger *slog.Logger
}

// CheckpointingExecutor wraps an Executor with checkpoint support for crash recovery.
type CheckpointingExecutor struct {
	executor Executor
	provider ports.ProviderPort
	config   ExecutorConfig
	cpConfig CheckpointConfig
}

// NewCheckpointingExecutor creates a new executor with checkpoint support.
func NewCheckpointingExecutor(
	provider ports.ProviderPort,
	config ExecutorConfig,
	cpConfig CheckpointConfig,
) *CheckpointingExecutor {
	return &CheckpointingExecutor{
		executor: NewExecutor(provider, config),
		provider: provider,
		config:   config,
		cpConfig: cpConfig,
	}
}

// Execute runs the skill with checkpoint support for crash recovery.
func (e *CheckpointingExecutor) Execute(ctx context.Context, s *domainSkill.Skill, input string) (*ExecutionResult, error) {
	// If checkpointing is disabled, delegate to base executor
	if !e.cpConfig.Enabled || e.cpConfig.Port == nil {
		return e.executor.Execute(ctx, s, input)
	}

	return e.executeWithCheckpoint(ctx, s, input)
}

// executeWithCheckpoint handles checkpoint creation, updates, and resume.
func (e *CheckpointingExecutor) executeWithCheckpoint(ctx context.Context, s *domainSkill.Skill, input string) (*ExecutionResult, error) {
	if s == nil {
		return nil, errors.NewError(errors.CodeValidation, "skill is required", nil)
	}

	if err := s.Validate(); err != nil {
		return nil, errors.NewError(errors.CodeValidation, "invalid skill", err)
	}

	// Apply timeout to context
	ctx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// Build DAG from phases
	phases := s.Phases()
	dag, err := workflow.NewDAG(phases)
	if err != nil {
		return &ExecutionResult{
			SkillID:   s.ID(),
			SkillName: s.Name(),
			Status:    PhaseStatusFailed,
			Error:     err,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}, err
	}

	// Get parallel batches for execution
	batches, err := dag.GetParallelBatches()
	if err != nil {
		return &ExecutionResult{
			SkillID:   s.ID(),
			SkillName: s.Name(),
			Status:    PhaseStatusFailed,
			Error:     err,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}, err
	}

	// Try to resume from checkpoint if requested
	var checkpoint *workflow.WorkflowCheckpoint
	var startBatchIndex int
	phaseOutputs := make(map[string]string)
	phaseOutputs["_input"] = input

	result := &ExecutionResult{
		SkillID:      s.ID(),
		SkillName:    s.Name(),
		Status:       PhaseStatusRunning,
		PhaseResults: make(map[string]*PhaseResult),
		StartTime:    time.Now(),
	}

	// Initialize all phases as pending
	for _, phase := range phases {
		result.PhaseResults[phase.ID] = &PhaseResult{
			PhaseID:   phase.ID,
			PhaseName: phase.Name,
			Status:    PhaseStatusPending,
		}
	}

	if e.cpConfig.Resume {
		checkpoint, err = e.tryResume(ctx, s, input, result, phaseOutputs)
		if err != nil {
			e.log("warn", "failed to resume from checkpoint", "error", err)
		}
		if checkpoint != nil {
			startBatchIndex = checkpoint.CompletedBatch() + 1
			e.log("info", "resuming from checkpoint",
				"checkpoint_id", checkpoint.ID(),
				"start_batch", startBatchIndex,
				"total_batches", len(batches))
		}
	}

	// Create new checkpoint if not resuming or no checkpoint found
	if checkpoint == nil {
		checkpoint, err = e.createCheckpoint(ctx, s, input, len(batches))
		if err != nil {
			e.log("warn", "failed to create checkpoint", "error", err)
			// Continue without checkpointing
			checkpoint = nil
		}
	}

	// Execute batches
	for batchIndex := startBatchIndex; batchIndex < len(batches); batchIndex++ {
		batch := batches[batchIndex]

		if err := e.executeBatch(ctx, dag, batch, result, phaseOutputs); err != nil {
			result.Status = PhaseStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)

			// Mark remaining phases as skipped
			e.markRemainingAsSkipped(result)

			// Update checkpoint to failed if we have one
			if checkpoint != nil {
				checkpoint.MarkFailed()
				if updateErr := e.cpConfig.Port.Update(ctx, checkpoint); updateErr != nil {
					e.log("warn", "failed to update checkpoint status to failed", "error", updateErr)
				}
			}

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
			e.markRemainingAsSkipped(result)
			return result, ctx.Err()
		}

		// Update checkpoint after successful batch
		if checkpoint != nil {
			e.updateCheckpoint(ctx, checkpoint, batchIndex, result, phaseOutputs)
		}
	}

	// Determine final output
	result.FinalOutput = e.determineFinalOutput(dag, phases, phaseOutputs)

	// Aggregate token counts
	for _, phaseResult := range result.PhaseResults {
		result.TotalTokens += phaseResult.InputTokens + phaseResult.OutputTokens
	}

	result.Status = PhaseStatusCompleted
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Mark checkpoint as completed
	if checkpoint != nil {
		checkpoint.MarkCompleted()
		if err := e.cpConfig.Port.Update(ctx, checkpoint); err != nil {
			e.log("warn", "failed to mark checkpoint completed", "error", err)
		}
	}

	return result, nil
}

// tryResume attempts to resume from an existing checkpoint.
func (e *CheckpointingExecutor) tryResume(
	ctx context.Context,
	s *domainSkill.Skill,
	input string,
	result *ExecutionResult,
	phaseOutputs map[string]string,
) (*workflow.WorkflowCheckpoint, error) {
	inputHash := workflow.HashInput(input)

	checkpoint, err := e.cpConfig.Port.GetLatestInProgress(ctx, s.ID(), inputHash)
	if err != nil {
		return nil, err
	}
	if checkpoint == nil {
		return nil, nil
	}

	// Restore phase outputs
	maps.Copy(phaseOutputs, checkpoint.PhaseOutputs())

	// Restore phase results
	for phaseID, data := range checkpoint.PhaseResults() {
		if pr, ok := result.PhaseResults[phaseID]; ok {
			pr.Status = PhaseStatus(data.Status)
			pr.Output = data.Output
			if data.ErrorMessage != "" {
				pr.Error = errors.New("checkpoint", data.ErrorMessage)
			}
			pr.StartTime = time.Unix(0, data.StartTime)
			pr.EndTime = time.Unix(0, data.EndTime)
			pr.Duration = time.Duration(data.DurationNs)
			pr.InputTokens = data.InputTokens
			pr.OutputTokens = data.OutputTokens
			pr.ModelUsed = data.ModelUsed
			pr.CacheHit = data.CacheHit
		}
	}

	return checkpoint, nil
}

// createCheckpoint creates a new checkpoint for this execution.
func (e *CheckpointingExecutor) createCheckpoint(
	ctx context.Context,
	s *domainSkill.Skill,
	input string,
	totalBatches int,
) (*workflow.WorkflowCheckpoint, error) {
	executionID := e.cpConfig.ExecutionID
	if executionID == "" {
		executionID = uuid.New().String()
	}

	checkpoint, err := workflow.NewWorkflowCheckpoint(
		uuid.New().String(),
		executionID,
		s.ID(),
		s.Name(),
		input,
		totalBatches,
	)
	if err != nil {
		return nil, err
	}

	checkpoint.SetMachineID(e.cpConfig.MachineID)
	checkpoint.AddPhaseOutput("_input", input)

	if err := e.cpConfig.Port.Create(ctx, checkpoint); err != nil {
		return nil, err
	}

	return checkpoint, nil
}

// updateCheckpoint updates the checkpoint after a batch completes.
func (e *CheckpointingExecutor) updateCheckpoint(
	ctx context.Context,
	checkpoint *workflow.WorkflowCheckpoint,
	batchIndex int,
	result *ExecutionResult,
	phaseOutputs map[string]string,
) {
	// Update completed batch
	if err := checkpoint.UpdateBatch(batchIndex); err != nil {
		e.log("warn", "failed to update batch in checkpoint", "error", err)
		return
	}

	// Update phase results
	for phaseID, pr := range result.PhaseResults {
		if pr.Status == PhaseStatusCompleted || pr.Status == PhaseStatusFailed {
			data := &workflow.PhaseResultData{
				PhaseID:      pr.PhaseID,
				PhaseName:    pr.PhaseName,
				Status:       string(pr.Status),
				Output:       pr.Output,
				StartTime:    pr.StartTime.UnixNano(),
				EndTime:      pr.EndTime.UnixNano(),
				DurationNs:   pr.Duration.Nanoseconds(),
				InputTokens:  pr.InputTokens,
				OutputTokens: pr.OutputTokens,
				ModelUsed:    pr.ModelUsed,
				CacheHit:     pr.CacheHit,
			}
			if pr.Error != nil {
				data.ErrorMessage = pr.Error.Error()
			}
			checkpoint.AddPhaseResult(phaseID, data)
		}
	}

	// Update phase outputs
	for k, v := range phaseOutputs {
		checkpoint.AddPhaseOutput(k, v)
	}

	// Calculate total tokens
	var inputTokens, outputTokens int
	for _, pr := range result.PhaseResults {
		inputTokens += pr.InputTokens
		outputTokens += pr.OutputTokens
	}
	checkpoint.UpdateTokens(inputTokens, outputTokens)

	// Save checkpoint
	if err := e.cpConfig.Port.Update(ctx, checkpoint); err != nil {
		e.log("warn", "failed to update checkpoint", "error", err)
	}
}

// executeBatch executes a batch of phases in parallel.
func (e *CheckpointingExecutor) executeBatch(
	ctx context.Context,
	dag *workflow.DAG,
	batch []string,
	result *ExecutionResult,
	phaseOutputs map[string]string,
) error {
	if len(batch) == 0 {
		return nil
	}

	// Create phase executor
	phaseExecutor := newPhaseExecutor(e.provider, e.config.MemoryContent)

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

		// Skip already completed phases (from checkpoint restore)
		if pr, ok := result.PhaseResults[phaseID]; ok && pr.Status == PhaseStatusCompleted {
			e.log("debug", "skipping already completed phase", "phase_id", phaseID)
			continue
		}

		wg.Add(1)
		go func(p *domainSkill.Phase) {
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

			// Build context from dependent phases
			mu.Lock()
			dependencyOutputs := e.gatherDependencyOutputs(dag, p.ID, phaseOutputs)
			mu.Unlock()

			// Update status to running
			mu.Lock()
			result.PhaseResults[p.ID].Status = PhaseStatusRunning
			result.PhaseResults[p.ID].StartTime = time.Now()
			mu.Unlock()

			// Execute the phase
			phaseResult := phaseExecutor.Execute(ctx, p, dependencyOutputs)

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
func (e *CheckpointingExecutor) gatherDependencyOutputs(dag *workflow.DAG, phaseID string, phaseOutputs map[string]string) map[string]string {
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
func (e *CheckpointingExecutor) markRemainingAsSkipped(result *ExecutionResult) {
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
func (e *CheckpointingExecutor) determineFinalOutput(dag *workflow.DAG, phases []domainSkill.Phase, phaseOutputs map[string]string) string {
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

// log logs a message if a logger is configured.
func (e *CheckpointingExecutor) log(level string, msg string, args ...any) {
	if e.cpConfig.Logger == nil {
		return
	}

	switch level {
	case "debug":
		e.cpConfig.Logger.Debug(msg, args...)
	case "info":
		e.cpConfig.Logger.Info(msg, args...)
	case "warn":
		e.cpConfig.Logger.Warn(msg, args...)
	case "error":
		e.cpConfig.Logger.Error(msg, args...)
	}
}

// GetExistingCheckpoint checks if there's an existing in-progress checkpoint for this skill/input.
// This can be used to warn users before starting a new execution.
func GetExistingCheckpoint(ctx context.Context, port ports.WorkflowCheckpointPort, skillID, input string) (*workflow.WorkflowCheckpoint, error) {
	return port.GetLatestInProgress(ctx, skillID, workflow.HashInput(input))
}
