package workflow

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// mockCheckpointPort implements ports.WorkflowCheckpointPort for testing
type mockCheckpointPort struct {
	mu                    sync.Mutex
	checkpoints           map[string]*workflow.WorkflowCheckpoint
	createCalls           int
	updateCalls           int
	getLatestInProgressFn func(skillID, inputHash string) *workflow.WorkflowCheckpoint
	createError           error
	updateError           error
}

func newMockCheckpointPort() *mockCheckpointPort {
	return &mockCheckpointPort{
		checkpoints: make(map[string]*workflow.WorkflowCheckpoint),
	}
}

func (m *mockCheckpointPort) Create(ctx context.Context, checkpoint *workflow.WorkflowCheckpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalls++
	if m.createError != nil {
		return m.createError
	}
	m.checkpoints[checkpoint.ID()] = checkpoint
	return nil
}

func (m *mockCheckpointPort) Get(ctx context.Context, id string) (*workflow.WorkflowCheckpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cp, ok := m.checkpoints[id]; ok {
		return cp, nil
	}
	return nil, nil
}

func (m *mockCheckpointPort) GetLatestInProgress(ctx context.Context, skillID string, inputHash string) (*workflow.WorkflowCheckpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getLatestInProgressFn != nil {
		return m.getLatestInProgressFn(skillID, inputHash), nil
	}
	// Find an in-progress checkpoint matching skill and input hash
	for _, cp := range m.checkpoints {
		if cp.SkillID() == skillID && cp.InputHash() == inputHash && cp.Status() == workflow.CheckpointStatusInProgress {
			return cp, nil
		}
	}
	return nil, nil
}

func (m *mockCheckpointPort) GetByExecutionID(ctx context.Context, executionID string) ([]*workflow.WorkflowCheckpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*workflow.WorkflowCheckpoint
	for _, cp := range m.checkpoints {
		if cp.ExecutionID() == executionID {
			result = append(result, cp)
		}
	}
	return result, nil
}

func (m *mockCheckpointPort) Update(ctx context.Context, checkpoint *workflow.WorkflowCheckpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++
	if m.updateError != nil {
		return m.updateError
	}
	m.checkpoints[checkpoint.ID()] = checkpoint
	return nil
}

func (m *mockCheckpointPort) List(ctx context.Context, filter *ports.WorkflowCheckpointFilter) ([]*workflow.WorkflowCheckpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*workflow.WorkflowCheckpoint
	for _, cp := range m.checkpoints {
		result = append(result, cp)
	}
	return result, nil
}

func (m *mockCheckpointPort) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.checkpoints, id)
	return nil
}

func (m *mockCheckpointPort) DeleteByExecutionID(ctx context.Context, executionID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for id, cp := range m.checkpoints {
		if cp.ExecutionID() == executionID {
			delete(m.checkpoints, id)
			count++
		}
	}
	return count, nil
}

func (m *mockCheckpointPort) MarkAbandoned(ctx context.Context, machineID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, cp := range m.checkpoints {
		if cp.MachineID() == machineID && cp.Status() == workflow.CheckpointStatusInProgress {
			cp.MarkAbandoned()
			count++
		}
	}
	return count, nil
}

func (m *mockCheckpointPort) Cleanup(ctx context.Context, olderThan time.Duration) (int, error) {
	return 0, nil
}

func TestNewCheckpointingExecutor(t *testing.T) {
	provider := newMockProvider()
	cpPort := newMockCheckpointPort()

	exec := NewCheckpointingExecutor(
		provider,
		DefaultExecutorConfig(),
		CheckpointConfig{
			Enabled:   true,
			Port:      cpPort,
			MachineID: "test-machine",
		},
	)

	if exec == nil {
		t.Error("expected executor to be created")
	}
}

func TestCheckpointingExecutor_Execute_WithCheckpointDisabled(t *testing.T) {
	provider := newMockProvider()
	cpPort := newMockCheckpointPort()

	exec := NewCheckpointingExecutor(
		provider,
		DefaultExecutorConfig(),
		CheckpointConfig{
			Enabled: false,
			Port:    cpPort,
		},
	)

	phase := createTestPhase(t, "phase1", "Phase 1", "Process: {{._input}}", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	result, err := exec.Execute(context.Background(), s, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	// No checkpoints should be created
	if cpPort.createCalls > 0 {
		t.Errorf("expected no checkpoint creates, got %d", cpPort.createCalls)
	}
}

func TestCheckpointingExecutor_Execute_CreatesCheckpoint(t *testing.T) {
	provider := newMockProvider()
	cpPort := newMockCheckpointPort()

	exec := NewCheckpointingExecutor(
		provider,
		DefaultExecutorConfig(),
		CheckpointConfig{
			Enabled:   true,
			Port:      cpPort,
			MachineID: "test-machine",
		},
	)

	phase := createTestPhase(t, "phase1", "Phase 1", "Process: {{._input}}", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	result, err := exec.Execute(context.Background(), s, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	// Checkpoint should be created
	if cpPort.createCalls != 1 {
		t.Errorf("expected 1 checkpoint create, got %d", cpPort.createCalls)
	}

	// Check that checkpoint was updated (once per batch + final completion)
	if cpPort.updateCalls < 1 {
		t.Errorf("expected at least 1 update call, got %d", cpPort.updateCalls)
	}

	// Verify checkpoint was marked completed
	cpPort.mu.Lock()
	defer cpPort.mu.Unlock()
	completedFound := false
	for _, cp := range cpPort.checkpoints {
		if cp.Status() == workflow.CheckpointStatusCompleted {
			completedFound = true
			break
		}
	}
	if !completedFound {
		t.Error("expected checkpoint to be marked as completed")
	}
}

func TestCheckpointingExecutor_Execute_MultiPhase(t *testing.T) {
	provider := newMockProvider()
	cpPort := newMockCheckpointPort()

	exec := NewCheckpointingExecutor(
		provider,
		DefaultExecutorConfig(),
		CheckpointConfig{
			Enabled:   true,
			Port:      cpPort,
			MachineID: "test-machine",
		},
	)

	// Create a 3-phase sequential skill
	phase1 := createTestPhase(t, "phase1", "Phase 1", "Process: {{._input}}", nil)
	phase2 := createTestPhase(t, "phase2", "Phase 2", "Continue: {{.phase1}}", []string{"phase1"})
	phase3 := createTestPhase(t, "phase3", "Phase 3", "Finalize: {{.phase2}}", []string{"phase2"})
	s := createTestSkill(t, []skill.Phase{phase1, phase2, phase3})

	result, err := exec.Execute(context.Background(), s, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	// All phases should be completed
	for phaseID, pr := range result.PhaseResults {
		if pr.Status != PhaseStatusCompleted {
			t.Errorf("expected phase %s to be Completed, got %s", phaseID, pr.Status)
		}
	}

	// Should have 3 batch updates + completion
	if cpPort.updateCalls < 3 {
		t.Errorf("expected at least 3 update calls for 3 batches, got %d", cpPort.updateCalls)
	}
}

func TestCheckpointingExecutor_Execute_Resume(t *testing.T) {
	provider := newMockProvider()
	cpPort := newMockCheckpointPort()

	// Create an existing in-progress checkpoint with batch 0 completed
	existingCP, _ := workflow.NewWorkflowCheckpoint(
		"existing-cp",
		"exec-123",
		"test-skill",
		"Test Skill",
		"test input",
		3,
	)
	existingCP.AddPhaseOutput("_input", "test input")
	existingCP.AddPhaseOutput("phase1", "Phase 1 output")
	existingCP.AddPhaseResult("phase1", &workflow.PhaseResultData{
		PhaseID:   "phase1",
		PhaseName: "Phase 1",
		Status:    "completed",
		Output:    "Phase 1 output",
	})
	_ = existingCP.UpdateBatch(0)
	cpPort.checkpoints["existing-cp"] = existingCP

	exec := NewCheckpointingExecutor(
		provider,
		DefaultExecutorConfig(),
		CheckpointConfig{
			Enabled:   true,
			Port:      cpPort,
			Resume:    true,
			MachineID: "test-machine",
		},
	)

	// Create a 3-phase sequential skill
	phase1 := createTestPhase(t, "phase1", "Phase 1", "Process: {{._input}}", nil)
	phase2 := createTestPhase(t, "phase2", "Phase 2", "Continue: {{.phase1}}", []string{"phase1"})
	phase3 := createTestPhase(t, "phase3", "Phase 3", "Finalize: {{.phase2}}", []string{"phase2"})
	s := createTestSkill(t, []skill.Phase{phase1, phase2, phase3})

	// Track provider calls to verify phase-1 was skipped
	initialCallCount := provider.callCount.Load()

	result, err := exec.Execute(context.Background(), s, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	// Provider should have been called only for phases 2 and 3 (not phase1)
	finalCallCount := provider.callCount.Load()
	callsDuringExec := finalCallCount - initialCallCount
	if callsDuringExec != 2 {
		t.Errorf("expected 2 provider calls (skipping resumed phase1), got %d", callsDuringExec)
	}
}

func TestCheckpointingExecutor_Execute_NilSkill(t *testing.T) {
	provider := newMockProvider()
	cpPort := newMockCheckpointPort()

	exec := NewCheckpointingExecutor(
		provider,
		DefaultExecutorConfig(),
		CheckpointConfig{
			Enabled:   true,
			Port:      cpPort,
			MachineID: "test-machine",
		},
	)

	_, err := exec.Execute(context.Background(), nil, "test input")
	if err == nil {
		t.Error("expected error for nil skill")
	}
}

func TestCheckpointingExecutor_Execute_CheckpointCreateError(t *testing.T) {
	provider := newMockProvider()
	cpPort := newMockCheckpointPort()
	cpPort.createError = context.DeadlineExceeded // Simulate error

	exec := NewCheckpointingExecutor(
		provider,
		DefaultExecutorConfig(),
		CheckpointConfig{
			Enabled:   true,
			Port:      cpPort,
			MachineID: "test-machine",
		},
	)

	phase := createTestPhase(t, "phase1", "Phase 1", "Process: {{._input}}", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	// Execution should still succeed even if checkpoint creation fails
	result, err := exec.Execute(context.Background(), s, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}
}

func TestCheckpointingExecutor_Execute_CheckpointUpdateError(t *testing.T) {
	provider := newMockProvider()
	cpPort := newMockCheckpointPort()

	// Allow create but fail on update
	cpPort.updateError = context.DeadlineExceeded

	exec := NewCheckpointingExecutor(
		provider,
		DefaultExecutorConfig(),
		CheckpointConfig{
			Enabled:   true,
			Port:      cpPort,
			MachineID: "test-machine",
		},
	)

	phase := createTestPhase(t, "phase1", "Phase 1", "Process: {{._input}}", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	// Execution should still succeed even if checkpoint update fails
	result, err := exec.Execute(context.Background(), s, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}
}

func TestGetExistingCheckpoint(t *testing.T) {
	cpPort := newMockCheckpointPort()

	// Create an in-progress checkpoint
	cp, _ := workflow.NewWorkflowCheckpoint(
		"cp-1",
		"exec-1",
		"skill-1",
		"Skill",
		"test input",
		3,
	)
	cpPort.checkpoints["cp-1"] = cp

	// Should find the checkpoint
	found, err := GetExistingCheckpoint(context.Background(), cpPort, "skill-1", "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil {
		t.Error("expected to find existing checkpoint")
	}

	// Should not find checkpoint for different input
	found, err = GetExistingCheckpoint(context.Background(), cpPort, "skill-1", "different input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Error("expected no checkpoint for different input")
	}
}
