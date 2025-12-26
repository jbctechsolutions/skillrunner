// Package workflow provides workflow orchestration types for skill execution.
package workflow

import "time"

// ExecutionStatus represents the status of a phase or skill execution.
type ExecutionStatus string

const (
	// StatusPending indicates execution has not started.
	StatusPending ExecutionStatus = "pending"
	// StatusRunning indicates execution is in progress.
	StatusRunning ExecutionStatus = "running"
	// StatusCompleted indicates execution finished successfully.
	StatusCompleted ExecutionStatus = "completed"
	// StatusFailed indicates execution failed with an error.
	StatusFailed ExecutionStatus = "failed"
	// StatusSkipped indicates execution was skipped.
	StatusSkipped ExecutionStatus = "skipped"
)

// PhaseResult captures the outcome of executing a single phase.
type PhaseResult struct {
	PhaseID      string          // ID of the executed phase
	Status       ExecutionStatus // Current status of the phase
	Output       string          // Output produced by the phase
	Error        error           // Error if the phase failed
	InputTokens  int             // Number of input tokens consumed
	OutputTokens int             // Number of output tokens generated
	ModelUsed    string          // Model used for execution
	Duration     time.Duration   // Time taken to execute
	StartedAt    time.Time       // When execution started
	CompletedAt  time.Time       // When execution completed
}

// ExecutionResult captures the outcome of executing an entire skill.
type ExecutionResult struct {
	SkillID      string          // ID of the executed skill
	Status       ExecutionStatus // Overall status of the execution
	PhaseResults []PhaseResult   // Results from each phase
	TotalCost    float64         // Total cost of execution
	StartedAt    time.Time       // When execution started
	CompletedAt  time.Time       // When execution completed
}

// NewPhaseResult creates a new PhaseResult with the given phase ID in pending status.
func NewPhaseResult(phaseID string) *PhaseResult {
	return &PhaseResult{
		PhaseID: phaseID,
		Status:  StatusPending,
	}
}

// MarkRunning transitions the phase result to running status and records the start time.
func (r *PhaseResult) MarkRunning() {
	r.Status = StatusRunning
	r.StartedAt = time.Now()
}

// MarkCompleted transitions the phase result to completed status with output and token counts.
func (r *PhaseResult) MarkCompleted(output string, inputTokens, outputTokens int, model string) {
	r.Status = StatusCompleted
	r.Output = output
	r.InputTokens = inputTokens
	r.OutputTokens = outputTokens
	r.ModelUsed = model
	r.CompletedAt = time.Now()
	if !r.StartedAt.IsZero() {
		r.Duration = r.CompletedAt.Sub(r.StartedAt)
	}
}

// MarkFailed transitions the phase result to failed status with an error.
func (r *PhaseResult) MarkFailed(err error) {
	r.Status = StatusFailed
	r.Error = err
	r.CompletedAt = time.Now()
	if !r.StartedAt.IsZero() {
		r.Duration = r.CompletedAt.Sub(r.StartedAt)
	}
}

// NewExecutionResult creates a new ExecutionResult with the given skill ID in pending status.
func NewExecutionResult(skillID string) *ExecutionResult {
	return &ExecutionResult{
		SkillID:      skillID,
		Status:       StatusPending,
		PhaseResults: make([]PhaseResult, 0),
		StartedAt:    time.Now(),
	}
}

// AddPhaseResult appends a phase result and updates the overall status.
func (r *ExecutionResult) AddPhaseResult(pr *PhaseResult) {
	if pr == nil {
		return
	}
	r.PhaseResults = append(r.PhaseResults, *pr)

	// Update overall status based on phase result
	switch pr.Status {
	case StatusFailed:
		r.Status = StatusFailed
		r.CompletedAt = time.Now()
	case StatusRunning:
		if r.Status == StatusPending {
			r.Status = StatusRunning
		}
	case StatusCompleted:
		// Only mark as completed if all phases are done
		// For now, just update to running if we were pending
		if r.Status == StatusPending {
			r.Status = StatusRunning
		}
	}
}

// IsSuccess returns true if the overall execution completed successfully.
func (r *ExecutionResult) IsSuccess() bool {
	return r.Status == StatusCompleted
}

// TotalDuration returns the total duration of the execution.
func (r *ExecutionResult) TotalDuration() time.Duration {
	if r.CompletedAt.IsZero() {
		if r.StartedAt.IsZero() {
			return 0
		}
		return time.Since(r.StartedAt)
	}
	return r.CompletedAt.Sub(r.StartedAt)
}

// TotalTokens returns the total input and output tokens across all phases.
func (r *ExecutionResult) TotalTokens() (input, output int) {
	for _, pr := range r.PhaseResults {
		input += pr.InputTokens
		output += pr.OutputTokens
	}
	return input, output
}

// Complete marks the execution as completed if no phase failed.
func (r *ExecutionResult) Complete() {
	if r.Status != StatusFailed {
		r.Status = StatusCompleted
	}
	r.CompletedAt = time.Now()
}
