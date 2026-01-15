// Package workflow provides workflow orchestration types for skill execution.
package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// CheckpointStatus represents the state of a workflow checkpoint.
type CheckpointStatus string

const (
	// CheckpointStatusInProgress indicates execution is ongoing.
	CheckpointStatusInProgress CheckpointStatus = "in_progress"
	// CheckpointStatusCompleted indicates execution finished successfully.
	CheckpointStatusCompleted CheckpointStatus = "completed"
	// CheckpointStatusFailed indicates execution failed.
	CheckpointStatusFailed CheckpointStatus = "failed"
	// CheckpointStatusAbandoned indicates the checkpoint was abandoned (e.g., process crashed).
	CheckpointStatusAbandoned CheckpointStatus = "abandoned"
)

// MaxInputSize is the maximum allowed size for checkpoint input (1MB).
const MaxInputSize = 1 << 20 // 1MB

// ValidCheckpointStatuses contains all valid checkpoint status values.
var ValidCheckpointStatuses = []CheckpointStatus{
	CheckpointStatusInProgress,
	CheckpointStatusCompleted,
	CheckpointStatusFailed,
	CheckpointStatusAbandoned,
}

// IsValidStatus checks if a status string is a valid checkpoint status.
func IsValidStatus(status string) bool {
	for _, s := range ValidCheckpointStatuses {
		if string(s) == status {
			return true
		}
	}
	return false
}

// PhaseResultData is a JSON-serializable version of PhaseResult for checkpoint storage.
// Unlike PhaseResult, it stores error as a string since error types cannot be serialized.
type PhaseResultData struct {
	PhaseID      string `json:"phase_id"`
	PhaseName    string `json:"phase_name"`
	Status       string `json:"status"`
	Output       string `json:"output"`
	ErrorMessage string `json:"error_message,omitempty"`
	StartTime    int64  `json:"start_time_unix"`
	EndTime      int64  `json:"end_time_unix"`
	DurationNs   int64  `json:"duration_ns"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	ModelUsed    string `json:"model_used"`
	CacheHit     bool   `json:"cache_hit"`
}

// WorkflowCheckpoint captures the state of a workflow execution for crash recovery.
// It stores all data needed to resume execution from the last completed batch.
type WorkflowCheckpoint struct {
	id             string                      // Unique checkpoint ID (UUID)
	executionID    string                      // Correlates checkpoints for the same execution
	skillID        string                      // Skill being executed
	skillName      string                      // Human-readable skill name
	input          string                      // Original input to the skill
	inputHash      string                      // Hash of input for matching
	completedBatch int                         // Last completed batch index (-1 if none)
	totalBatches   int                         // Total number of batches in the DAG
	phaseResults   map[string]*PhaseResultData // Results from completed phases (phaseID -> result)
	phaseOutputs   map[string]string           // Outputs for template interpolation
	status         CheckpointStatus            // Current checkpoint status
	inputTokens    int                         // Total input tokens consumed
	outputTokens   int                         // Total output tokens consumed
	machineID      string                      // Machine where execution started
	createdAt      time.Time
	updatedAt      time.Time
}

// NewWorkflowCheckpoint creates a new WorkflowCheckpoint with the required fields.
// Returns an error if validation fails.
func NewWorkflowCheckpoint(id, executionID, skillID, skillName, input string, totalBatches int) (*WorkflowCheckpoint, error) {
	id = strings.TrimSpace(id)
	executionID = strings.TrimSpace(executionID)
	skillID = strings.TrimSpace(skillID)
	skillName = strings.TrimSpace(skillName)

	if id == "" {
		return nil, errors.New("workflow_checkpoint", "checkpoint ID is required")
	}
	if executionID == "" {
		return nil, errors.New("workflow_checkpoint", "execution ID is required")
	}
	if skillID == "" {
		return nil, errors.New("workflow_checkpoint", "skill ID is required")
	}
	if skillName == "" {
		return nil, errors.New("workflow_checkpoint", "skill name is required")
	}
	if totalBatches < 1 {
		return nil, errors.New("workflow_checkpoint", "total batches must be at least 1")
	}
	if len(input) > MaxInputSize {
		return nil, errors.New("workflow_checkpoint", "input exceeds maximum size limit")
	}

	now := time.Now()
	return &WorkflowCheckpoint{
		id:             id,
		executionID:    executionID,
		skillID:        skillID,
		skillName:      skillName,
		input:          input,
		inputHash:      HashInput(input),
		completedBatch: -1, // No batches completed yet
		totalBatches:   totalBatches,
		phaseResults:   make(map[string]*PhaseResultData),
		phaseOutputs:   make(map[string]string),
		status:         CheckpointStatusInProgress,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// HashInput generates a deterministic hash of the input for checkpoint matching.
func HashInput(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:16]) // First 128 bits is sufficient
}

// ID returns the checkpoint's unique identifier.
func (c *WorkflowCheckpoint) ID() string {
	return c.id
}

// ExecutionID returns the execution correlation ID.
func (c *WorkflowCheckpoint) ExecutionID() string {
	return c.executionID
}

// SkillID returns the skill being executed.
func (c *WorkflowCheckpoint) SkillID() string {
	return c.skillID
}

// SkillName returns the human-readable skill name.
func (c *WorkflowCheckpoint) SkillName() string {
	return c.skillName
}

// Input returns the original input to the skill.
func (c *WorkflowCheckpoint) Input() string {
	return c.input
}

// InputHash returns the hash of the input for matching.
func (c *WorkflowCheckpoint) InputHash() string {
	return c.inputHash
}

// CompletedBatch returns the index of the last completed batch (-1 if none).
func (c *WorkflowCheckpoint) CompletedBatch() int {
	return c.completedBatch
}

// TotalBatches returns the total number of batches in the workflow.
func (c *WorkflowCheckpoint) TotalBatches() int {
	return c.totalBatches
}

// PhaseResults returns a copy of the phase results map.
func (c *WorkflowCheckpoint) PhaseResults() map[string]*PhaseResultData {
	results := make(map[string]*PhaseResultData, len(c.phaseResults))
	for k, v := range c.phaseResults {
		// Deep copy the PhaseResultData
		copied := *v
		results[k] = &copied
	}
	return results
}

// PhaseOutputs returns a copy of the phase outputs map.
func (c *WorkflowCheckpoint) PhaseOutputs() map[string]string {
	outputs := make(map[string]string, len(c.phaseOutputs))
	maps.Copy(outputs, c.phaseOutputs)
	return outputs
}

// Status returns the current checkpoint status.
func (c *WorkflowCheckpoint) Status() CheckpointStatus {
	return c.status
}

// InputTokens returns the total input tokens consumed.
func (c *WorkflowCheckpoint) InputTokens() int {
	return c.inputTokens
}

// OutputTokens returns the total output tokens consumed.
func (c *WorkflowCheckpoint) OutputTokens() int {
	return c.outputTokens
}

// TotalTokens returns the total tokens (input + output) consumed.
func (c *WorkflowCheckpoint) TotalTokens() int {
	return c.inputTokens + c.outputTokens
}

// MachineID returns the machine ID where execution started.
func (c *WorkflowCheckpoint) MachineID() string {
	return c.machineID
}

// CreatedAt returns when the checkpoint was created.
func (c *WorkflowCheckpoint) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns when the checkpoint was last updated.
func (c *WorkflowCheckpoint) UpdatedAt() time.Time {
	return c.updatedAt
}

// SetMachineID sets the machine ID for the checkpoint.
func (c *WorkflowCheckpoint) SetMachineID(machineID string) {
	c.machineID = strings.TrimSpace(machineID)
	c.updatedAt = time.Now()
}

// UpdateBatch updates the completed batch index and records progress.
func (c *WorkflowCheckpoint) UpdateBatch(batchIndex int) error {
	if batchIndex < 0 {
		return errors.New("workflow_checkpoint", "batch index cannot be negative")
	}
	if batchIndex >= c.totalBatches {
		return errors.New("workflow_checkpoint", "batch index exceeds total batches")
	}
	c.completedBatch = batchIndex
	c.updatedAt = time.Now()
	return nil
}

// AddPhaseResult adds or updates a phase result in the checkpoint.
func (c *WorkflowCheckpoint) AddPhaseResult(phaseID string, result *PhaseResultData) {
	if phaseID == "" || result == nil {
		return
	}
	c.phaseResults[phaseID] = result
	c.updatedAt = time.Now()
}

// SetPhaseResults replaces all phase results.
func (c *WorkflowCheckpoint) SetPhaseResults(results map[string]*PhaseResultData) {
	c.phaseResults = make(map[string]*PhaseResultData, len(results))
	for k, v := range results {
		if v != nil {
			copied := *v
			c.phaseResults[k] = &copied
		}
	}
	c.updatedAt = time.Now()
}

// AddPhaseOutput adds or updates a phase output in the checkpoint.
func (c *WorkflowCheckpoint) AddPhaseOutput(phaseID, output string) {
	if phaseID == "" {
		return
	}
	c.phaseOutputs[phaseID] = output
	c.updatedAt = time.Now()
}

// SetPhaseOutputs replaces all phase outputs.
func (c *WorkflowCheckpoint) SetPhaseOutputs(outputs map[string]string) {
	c.phaseOutputs = make(map[string]string, len(outputs))
	maps.Copy(c.phaseOutputs, outputs)
	c.updatedAt = time.Now()
}

// UpdateTokens updates the token counts.
func (c *WorkflowCheckpoint) UpdateTokens(inputTokens, outputTokens int) {
	c.inputTokens = inputTokens
	c.outputTokens = outputTokens
	c.updatedAt = time.Now()
}

// AddTokens adds to the existing token counts.
func (c *WorkflowCheckpoint) AddTokens(inputTokens, outputTokens int) {
	c.inputTokens += inputTokens
	c.outputTokens += outputTokens
	c.updatedAt = time.Now()
}

// MarkCompleted marks the checkpoint as completed.
func (c *WorkflowCheckpoint) MarkCompleted() {
	c.status = CheckpointStatusCompleted
	c.updatedAt = time.Now()
}

// MarkFailed marks the checkpoint as failed.
func (c *WorkflowCheckpoint) MarkFailed() {
	c.status = CheckpointStatusFailed
	c.updatedAt = time.Now()
}

// MarkAbandoned marks the checkpoint as abandoned.
func (c *WorkflowCheckpoint) MarkAbandoned() {
	c.status = CheckpointStatusAbandoned
	c.updatedAt = time.Now()
}

// IsResumable returns true if the checkpoint can be resumed.
func (c *WorkflowCheckpoint) IsResumable() bool {
	return c.status == CheckpointStatusInProgress
}

// Progress returns a string describing the checkpoint progress (e.g., "2/5").
func (c *WorkflowCheckpoint) Progress() string {
	completed := c.completedBatch + 1
	return fmt.Sprintf("%d/%d", completed, c.totalBatches)
}

// Validate checks if the checkpoint is in a valid state.
func (c *WorkflowCheckpoint) Validate() error {
	if strings.TrimSpace(c.id) == "" {
		return errors.New("workflow_checkpoint", "checkpoint ID is required")
	}
	if strings.TrimSpace(c.executionID) == "" {
		return errors.New("workflow_checkpoint", "execution ID is required")
	}
	if strings.TrimSpace(c.skillID) == "" {
		return errors.New("workflow_checkpoint", "skill ID is required")
	}
	if strings.TrimSpace(c.skillName) == "" {
		return errors.New("workflow_checkpoint", "skill name is required")
	}
	if c.totalBatches < 1 {
		return errors.New("workflow_checkpoint", "total batches must be at least 1")
	}
	if c.completedBatch >= c.totalBatches {
		return errors.New("workflow_checkpoint", "completed batch exceeds total batches")
	}
	return nil
}

// ReconstructCheckpoint creates a checkpoint from stored data (used by repository).
// This bypasses normal validation to allow loading existing checkpoints.
func ReconstructCheckpoint(
	id, executionID, skillID, skillName, input, inputHash string,
	completedBatch, totalBatches int,
	phaseResults map[string]*PhaseResultData,
	phaseOutputs map[string]string,
	status CheckpointStatus,
	inputTokens, outputTokens int,
	machineID string,
	createdAt, updatedAt time.Time,
) *WorkflowCheckpoint {
	if phaseResults == nil {
		phaseResults = make(map[string]*PhaseResultData)
	}
	if phaseOutputs == nil {
		phaseOutputs = make(map[string]string)
	}
	return &WorkflowCheckpoint{
		id:             id,
		executionID:    executionID,
		skillID:        skillID,
		skillName:      skillName,
		input:          input,
		inputHash:      inputHash,
		completedBatch: completedBatch,
		totalBatches:   totalBatches,
		phaseResults:   phaseResults,
		phaseOutputs:   phaseOutputs,
		status:         status,
		inputTokens:    inputTokens,
		outputTokens:   outputTokens,
		machineID:      machineID,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}
