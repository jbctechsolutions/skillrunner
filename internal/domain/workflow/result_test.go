package workflow

import (
	"errors"
	"testing"
	"time"
)

func TestExecutionStatus_Constants(t *testing.T) {
	tests := []struct {
		status   ExecutionStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusSkipped, "skipped"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}

func TestNewPhaseResult(t *testing.T) {
	phaseID := "test-phase"
	pr := NewPhaseResult(phaseID)

	if pr == nil {
		t.Fatal("expected non-nil PhaseResult")
	}
	if pr.PhaseID != phaseID {
		t.Errorf("expected PhaseID %s, got %s", phaseID, pr.PhaseID)
	}
	if pr.Status != StatusPending {
		t.Errorf("expected status %s, got %s", StatusPending, pr.Status)
	}
	if pr.Output != "" {
		t.Errorf("expected empty output, got %s", pr.Output)
	}
	if pr.Error != nil {
		t.Errorf("expected nil error, got %v", pr.Error)
	}
}

func TestPhaseResult_MarkRunning(t *testing.T) {
	pr := NewPhaseResult("test-phase")

	before := time.Now()
	pr.MarkRunning()
	after := time.Now()

	if pr.Status != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, pr.Status)
	}
	if pr.StartedAt.Before(before) || pr.StartedAt.After(after) {
		t.Errorf("StartedAt should be between before and after")
	}
}

func TestPhaseResult_MarkCompleted(t *testing.T) {
	pr := NewPhaseResult("test-phase")
	pr.MarkRunning()

	// Small delay to ensure duration is measurable
	time.Sleep(time.Millisecond)

	output := "test output"
	inputTokens := 100
	outputTokens := 50
	model := "claude-3-opus"

	before := time.Now()
	pr.MarkCompleted(output, inputTokens, outputTokens, model)
	after := time.Now()

	if pr.Status != StatusCompleted {
		t.Errorf("expected status %s, got %s", StatusCompleted, pr.Status)
	}
	if pr.Output != output {
		t.Errorf("expected output %s, got %s", output, pr.Output)
	}
	if pr.InputTokens != inputTokens {
		t.Errorf("expected input tokens %d, got %d", inputTokens, pr.InputTokens)
	}
	if pr.OutputTokens != outputTokens {
		t.Errorf("expected output tokens %d, got %d", outputTokens, pr.OutputTokens)
	}
	if pr.ModelUsed != model {
		t.Errorf("expected model %s, got %s", model, pr.ModelUsed)
	}
	if pr.CompletedAt.Before(before) || pr.CompletedAt.After(after) {
		t.Errorf("CompletedAt should be between before and after")
	}
	if pr.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", pr.Duration)
	}
}

func TestPhaseResult_MarkCompleted_WithoutRunning(t *testing.T) {
	pr := NewPhaseResult("test-phase")

	// Mark completed without first marking running
	pr.MarkCompleted("output", 100, 50, "model")

	if pr.Status != StatusCompleted {
		t.Errorf("expected status %s, got %s", StatusCompleted, pr.Status)
	}
	// Duration should be 0 since StartedAt was never set
	if pr.Duration != 0 {
		t.Errorf("expected 0 duration when not started, got %v", pr.Duration)
	}
}

func TestPhaseResult_MarkFailed(t *testing.T) {
	pr := NewPhaseResult("test-phase")
	pr.MarkRunning()

	time.Sleep(time.Millisecond)

	testErr := errors.New("test error")
	before := time.Now()
	pr.MarkFailed(testErr)
	after := time.Now()

	if pr.Status != StatusFailed {
		t.Errorf("expected status %s, got %s", StatusFailed, pr.Status)
	}
	if pr.Error != testErr {
		t.Errorf("expected error %v, got %v", testErr, pr.Error)
	}
	if pr.CompletedAt.Before(before) || pr.CompletedAt.After(after) {
		t.Errorf("CompletedAt should be between before and after")
	}
	if pr.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", pr.Duration)
	}
}

func TestPhaseResult_MarkFailed_WithoutRunning(t *testing.T) {
	pr := NewPhaseResult("test-phase")

	testErr := errors.New("test error")
	pr.MarkFailed(testErr)

	if pr.Status != StatusFailed {
		t.Errorf("expected status %s, got %s", StatusFailed, pr.Status)
	}
	if pr.Duration != 0 {
		t.Errorf("expected 0 duration when not started, got %v", pr.Duration)
	}
}

func TestNewExecutionResult(t *testing.T) {
	skillID := "test-skill"

	before := time.Now()
	er := NewExecutionResult(skillID)
	after := time.Now()

	if er == nil {
		t.Fatal("expected non-nil ExecutionResult")
	}
	if er.SkillID != skillID {
		t.Errorf("expected SkillID %s, got %s", skillID, er.SkillID)
	}
	if er.Status != StatusPending {
		t.Errorf("expected status %s, got %s", StatusPending, er.Status)
	}
	if er.PhaseResults == nil {
		t.Error("expected non-nil PhaseResults slice")
	}
	if len(er.PhaseResults) != 0 {
		t.Errorf("expected empty PhaseResults, got %d", len(er.PhaseResults))
	}
	if er.StartedAt.Before(before) || er.StartedAt.After(after) {
		t.Errorf("StartedAt should be between before and after")
	}
}

func TestExecutionResult_AddPhaseResult(t *testing.T) {
	er := NewExecutionResult("test-skill")

	// Add a running phase
	pr1 := NewPhaseResult("phase-1")
	pr1.MarkRunning()
	er.AddPhaseResult(pr1)

	if len(er.PhaseResults) != 1 {
		t.Errorf("expected 1 phase result, got %d", len(er.PhaseResults))
	}
	if er.Status != StatusRunning {
		t.Errorf("expected status %s after running phase, got %s", StatusRunning, er.Status)
	}

	// Add a completed phase
	pr2 := NewPhaseResult("phase-2")
	pr2.MarkRunning()
	pr2.MarkCompleted("output", 100, 50, "model")
	er.AddPhaseResult(pr2)

	if len(er.PhaseResults) != 2 {
		t.Errorf("expected 2 phase results, got %d", len(er.PhaseResults))
	}
}

func TestExecutionResult_AddPhaseResult_Nil(t *testing.T) {
	er := NewExecutionResult("test-skill")
	initialLen := len(er.PhaseResults)

	er.AddPhaseResult(nil)

	if len(er.PhaseResults) != initialLen {
		t.Errorf("expected no change when adding nil, got %d results", len(er.PhaseResults))
	}
}

func TestExecutionResult_AddPhaseResult_Failed(t *testing.T) {
	er := NewExecutionResult("test-skill")

	pr := NewPhaseResult("phase-1")
	pr.MarkRunning()
	pr.MarkFailed(errors.New("test error"))
	er.AddPhaseResult(pr)

	if er.Status != StatusFailed {
		t.Errorf("expected status %s after failed phase, got %s", StatusFailed, er.Status)
	}
	if er.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set when phase fails")
	}
}

func TestExecutionResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		expected bool
	}{
		{"pending", StatusPending, false},
		{"running", StatusRunning, false},
		{"completed", StatusCompleted, true},
		{"failed", StatusFailed, false},
		{"skipped", StatusSkipped, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := NewExecutionResult("test-skill")
			er.Status = tt.status

			if got := er.IsSuccess(); got != tt.expected {
				t.Errorf("IsSuccess() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestExecutionResult_TotalDuration(t *testing.T) {
	t.Run("with completed execution", func(t *testing.T) {
		er := NewExecutionResult("test-skill")
		time.Sleep(10 * time.Millisecond)
		er.Complete()

		duration := er.TotalDuration()
		if duration < 10*time.Millisecond {
			t.Errorf("expected duration >= 10ms, got %v", duration)
		}
	})

	t.Run("with ongoing execution", func(t *testing.T) {
		er := NewExecutionResult("test-skill")
		time.Sleep(10 * time.Millisecond)

		duration := er.TotalDuration()
		if duration < 10*time.Millisecond {
			t.Errorf("expected duration >= 10ms for ongoing execution, got %v", duration)
		}
	})

	t.Run("with no start time", func(t *testing.T) {
		er := &ExecutionResult{
			SkillID: "test-skill",
			Status:  StatusPending,
		}

		duration := er.TotalDuration()
		if duration != 0 {
			t.Errorf("expected 0 duration with no start time, got %v", duration)
		}
	})
}

func TestExecutionResult_TotalTokens(t *testing.T) {
	er := NewExecutionResult("test-skill")

	// Add some phase results with token counts
	pr1 := NewPhaseResult("phase-1")
	pr1.MarkRunning()
	pr1.MarkCompleted("output1", 100, 50, "model1")
	er.AddPhaseResult(pr1)

	pr2 := NewPhaseResult("phase-2")
	pr2.MarkRunning()
	pr2.MarkCompleted("output2", 200, 75, "model2")
	er.AddPhaseResult(pr2)

	pr3 := NewPhaseResult("phase-3")
	pr3.MarkRunning()
	pr3.MarkCompleted("output3", 150, 100, "model3")
	er.AddPhaseResult(pr3)

	inputTokens, outputTokens := er.TotalTokens()

	expectedInput := 100 + 200 + 150
	expectedOutput := 50 + 75 + 100

	if inputTokens != expectedInput {
		t.Errorf("expected input tokens %d, got %d", expectedInput, inputTokens)
	}
	if outputTokens != expectedOutput {
		t.Errorf("expected output tokens %d, got %d", expectedOutput, outputTokens)
	}
}

func TestExecutionResult_TotalTokens_Empty(t *testing.T) {
	er := NewExecutionResult("test-skill")

	inputTokens, outputTokens := er.TotalTokens()

	if inputTokens != 0 {
		t.Errorf("expected 0 input tokens for empty result, got %d", inputTokens)
	}
	if outputTokens != 0 {
		t.Errorf("expected 0 output tokens for empty result, got %d", outputTokens)
	}
}

func TestExecutionResult_Complete(t *testing.T) {
	t.Run("marks as completed when not failed", func(t *testing.T) {
		er := NewExecutionResult("test-skill")
		er.Status = StatusRunning

		before := time.Now()
		er.Complete()
		after := time.Now()

		if er.Status != StatusCompleted {
			t.Errorf("expected status %s, got %s", StatusCompleted, er.Status)
		}
		if er.CompletedAt.Before(before) || er.CompletedAt.After(after) {
			t.Errorf("CompletedAt should be between before and after")
		}
	})

	t.Run("keeps failed status", func(t *testing.T) {
		er := NewExecutionResult("test-skill")
		er.Status = StatusFailed

		er.Complete()

		if er.Status != StatusFailed {
			t.Errorf("expected status to remain %s, got %s", StatusFailed, er.Status)
		}
	})
}

func TestPhaseResult_Lifecycle(t *testing.T) {
	// Test complete lifecycle: pending -> running -> completed
	pr := NewPhaseResult("lifecycle-phase")

	if pr.Status != StatusPending {
		t.Fatalf("expected initial status %s", StatusPending)
	}

	pr.MarkRunning()
	if pr.Status != StatusRunning {
		t.Fatalf("expected status %s after MarkRunning", StatusRunning)
	}
	if pr.StartedAt.IsZero() {
		t.Fatal("expected StartedAt to be set")
	}

	time.Sleep(time.Millisecond)
	pr.MarkCompleted("final output", 500, 250, "claude-3-sonnet")

	if pr.Status != StatusCompleted {
		t.Fatalf("expected status %s after MarkCompleted", StatusCompleted)
	}
	if pr.Output != "final output" {
		t.Errorf("expected output 'final output', got %s", pr.Output)
	}
	if pr.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", pr.Duration)
	}
}

func TestExecutionResult_Lifecycle(t *testing.T) {
	// Test complete lifecycle with multiple phases
	er := NewExecutionResult("lifecycle-skill")

	if er.Status != StatusPending {
		t.Fatalf("expected initial status %s", StatusPending)
	}

	// Phase 1: runs and completes
	pr1 := NewPhaseResult("phase-1")
	pr1.MarkRunning()
	pr1.MarkCompleted("output-1", 100, 50, "model")
	er.AddPhaseResult(pr1)

	if er.Status != StatusRunning {
		t.Errorf("expected status %s after adding phase", StatusRunning)
	}

	// Phase 2: runs and completes
	pr2 := NewPhaseResult("phase-2")
	pr2.MarkRunning()
	pr2.MarkCompleted("output-2", 200, 100, "model")
	er.AddPhaseResult(pr2)

	// Complete the execution
	er.Complete()

	if er.Status != StatusCompleted {
		t.Errorf("expected status %s after Complete", StatusCompleted)
	}
	if !er.IsSuccess() {
		t.Error("expected IsSuccess to be true")
	}

	input, output := er.TotalTokens()
	if input != 300 || output != 150 {
		t.Errorf("expected tokens (300, 150), got (%d, %d)", input, output)
	}
}
