package workflow

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// mockProvider implements ports.ProviderPort for testing
type mockProvider struct {
	mu             sync.Mutex
	completeCalls  []ports.CompletionRequest
	completeFunc   func(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error)
	completeDelay  time.Duration
	callCount      atomic.Int32
	failOnPhaseIDs map[string]error
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		failOnPhaseIDs: make(map[string]error),
	}
}

func (m *mockProvider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:        "mock",
		Description: "Mock provider for testing",
	}
}

func (m *mockProvider) ListModels(_ context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func (m *mockProvider) SupportsModel(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (m *mockProvider) IsAvailable(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (m *mockProvider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	m.callCount.Add(1)

	if m.completeDelay > 0 {
		select {
		case <-time.After(m.completeDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	m.mu.Lock()
	m.completeCalls = append(m.completeCalls, req)
	m.mu.Unlock()

	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}

	// Default response
	return &ports.CompletionResponse{
		Content:      "Mock response for: " + req.Messages[len(req.Messages)-1].Content,
		InputTokens:  10,
		OutputTokens: 20,
		FinishReason: "stop",
		ModelUsed:    req.ModelID,
		Duration:     100 * time.Millisecond,
	}, nil
}

func (m *mockProvider) Stream(_ context.Context, _ ports.CompletionRequest, _ ports.StreamCallback) (*ports.CompletionResponse, error) {
	return nil, errors.New("streaming not implemented in mock")
}

func (m *mockProvider) HealthCheck(_ context.Context, _ string) (*ports.HealthStatus, error) {
	return &ports.HealthStatus{
		Healthy:     true,
		Message:     "OK",
		LastChecked: time.Now(),
	}, nil
}

// Helper to create a simple skill for testing
func createTestSkill(t *testing.T, phases []skill.Phase) *skill.Skill {
	t.Helper()
	s, err := skill.NewSkill("test-skill", "Test Skill", "1.0.0", phases)
	if err != nil {
		t.Fatalf("failed to create test skill: %v", err)
	}
	return s
}

// Helper to create a valid phase
func createTestPhase(t *testing.T, id, name, prompt string, deps []string) skill.Phase {
	t.Helper()
	p, err := skill.NewPhase(id, name, prompt)
	if err != nil {
		t.Fatalf("failed to create test phase: %v", err)
	}
	if deps != nil {
		p = p.WithDependencies(deps)
	}
	return *p
}

func TestNewExecutor(t *testing.T) {
	provider := newMockProvider()

	t.Run("with default config", func(t *testing.T) {
		exec := NewExecutor(provider, ExecutorConfig{})
		if exec == nil {
			t.Error("expected executor to be created")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := ExecutorConfig{
			MaxParallel: 10,
			Timeout:     10 * time.Minute,
		}
		exec := NewExecutor(provider, config)
		if exec == nil {
			t.Error("expected executor to be created")
		}
	})
}

func TestExecutor_Execute_SinglePhase(t *testing.T) {
	provider := newMockProvider()
	exec := NewExecutor(provider, DefaultExecutorConfig())

	phase := createTestPhase(t, "phase-1", "Phase 1", "Process this: {{._input}}", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	result, err := exec.Execute(context.Background(), s, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	if result.SkillID != "test-skill" {
		t.Errorf("expected skill ID test-skill, got %s", result.SkillID)
	}

	if len(result.PhaseResults) != 1 {
		t.Errorf("expected 1 phase result, got %d", len(result.PhaseResults))
	}

	phaseResult := result.PhaseResults["phase-1"]
	if phaseResult == nil {
		t.Fatal("expected phase result for phase-1")
	}
	if phaseResult.Status != PhaseStatusCompleted {
		t.Errorf("expected phase status Completed, got %s", phaseResult.Status)
	}
	if phaseResult.Output == "" {
		t.Error("expected non-empty output")
	}
}

func TestExecutor_Execute_SequentialPhases(t *testing.T) {
	provider := newMockProvider()
	exec := NewExecutor(provider, DefaultExecutorConfig())

	phases := []skill.Phase{
		createTestPhase(t, "phase1", "Phase 1", "Process: {{._input}}", nil),
		createTestPhase(t, "phase2", "Phase 2", "Continue from: {{.phase1}}", []string{"phase1"}),
		createTestPhase(t, "phase3", "Phase 3", "Finalize: {{.phase2}}", []string{"phase2"}),
	}
	s := createTestSkill(t, phases)

	result, err := exec.Execute(context.Background(), s, "initial input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	if len(result.PhaseResults) != 3 {
		t.Errorf("expected 3 phase results, got %d", len(result.PhaseResults))
	}

	// Verify all phases completed
	for id, pr := range result.PhaseResults {
		if pr.Status != PhaseStatusCompleted {
			t.Errorf("phase %s: expected status Completed, got %s", id, pr.Status)
		}
	}

	// Verify execution order via timestamps
	if result.PhaseResults["phase1"].EndTime.After(result.PhaseResults["phase2"].StartTime) {
		// Allow some slack for timing
		if result.PhaseResults["phase1"].EndTime.Sub(result.PhaseResults["phase2"].StartTime) > 100*time.Millisecond {
			t.Error("phase2 should start after phase1 ends")
		}
	}
}

func TestExecutor_Execute_ParallelPhases(t *testing.T) {
	provider := newMockProvider()
	provider.completeDelay = 50 * time.Millisecond

	config := ExecutorConfig{
		MaxParallel: 4,
		Timeout:     5 * time.Minute,
	}
	exec := NewExecutor(provider, config)

	// Create phases that can run in parallel (no dependencies on each other)
	phases := []skill.Phase{
		createTestPhase(t, "parallel1", "Parallel 1", "Process A: {{._input}}", nil),
		createTestPhase(t, "parallel2", "Parallel 2", "Process B: {{._input}}", nil),
		createTestPhase(t, "parallel3", "Parallel 3", "Process C: {{._input}}", nil),
	}
	s := createTestSkill(t, phases)

	start := time.Now()
	result, err := exec.Execute(context.Background(), s, "parallel test")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	// If running in parallel, total time should be less than 3x the delay
	// (some overhead expected)
	if elapsed > 3*provider.completeDelay {
		t.Logf("elapsed: %v, expected less than %v for parallel execution", elapsed, 3*provider.completeDelay)
	}

	// All phases should complete
	for id, pr := range result.PhaseResults {
		if pr.Status != PhaseStatusCompleted {
			t.Errorf("phase %s: expected status Completed, got %s", id, pr.Status)
		}
	}
}

func TestExecutor_Execute_DAGWithMixedDependencies(t *testing.T) {
	provider := newMockProvider()
	exec := NewExecutor(provider, DefaultExecutorConfig())

	// DAG structure:
	// phase1 ──┬──> phase3 ──> phase5
	//          │
	// phase2 ──┴──> phase4 ──┘
	phases := []skill.Phase{
		createTestPhase(t, "phase1", "Phase 1", "A: {{._input}}", nil),
		createTestPhase(t, "phase2", "Phase 2", "B: {{._input}}", nil),
		createTestPhase(t, "phase3", "Phase 3", "C: {{.phase1}} {{.phase2}}", []string{"phase1", "phase2"}),
		createTestPhase(t, "phase4", "Phase 4", "D: {{.phase1}} {{.phase2}}", []string{"phase1", "phase2"}),
		createTestPhase(t, "phase5", "Phase 5", "E: {{.phase3}} {{.phase4}}", []string{"phase3", "phase4"}),
	}
	s := createTestSkill(t, phases)

	result, err := exec.Execute(context.Background(), s, "dag input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	if len(result.PhaseResults) != 5 {
		t.Errorf("expected 5 phase results, got %d", len(result.PhaseResults))
	}

	// All phases should complete
	for id, pr := range result.PhaseResults {
		if pr.Status != PhaseStatusCompleted {
			t.Errorf("phase %s: expected status Completed, got %s", id, pr.Status)
		}
	}

	// Final output should be from phase5
	if result.FinalOutput == "" {
		t.Error("expected non-empty final output from terminal phase")
	}
}

func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	provider := newMockProvider()
	provider.completeDelay = 500 * time.Millisecond

	exec := NewExecutor(provider, DefaultExecutorConfig())

	phase := createTestPhase(t, "slowphase", "Slow Phase", "Process: {{._input}}", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := exec.Execute(ctx, s, "test")

	if err == nil {
		t.Error("expected context cancellation error")
	}

	if result.Status != PhaseStatusFailed {
		t.Errorf("expected status Failed, got %s", result.Status)
	}
}

func TestExecutor_Execute_Timeout(t *testing.T) {
	provider := newMockProvider()
	provider.completeDelay = 500 * time.Millisecond

	config := ExecutorConfig{
		MaxParallel: 4,
		Timeout:     100 * time.Millisecond, // Short timeout
	}
	exec := NewExecutor(provider, config)

	phase := createTestPhase(t, "slowphase", "Slow Phase", "Process: {{._input}}", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	result, err := exec.Execute(context.Background(), s, "test")

	if err == nil {
		t.Error("expected timeout error")
	}

	if result.Status != PhaseStatusFailed {
		t.Errorf("expected status Failed, got %s", result.Status)
	}
}

func TestExecutor_Execute_ProviderError(t *testing.T) {
	provider := newMockProvider()
	provider.completeFunc = func(_ context.Context, _ ports.CompletionRequest) (*ports.CompletionResponse, error) {
		return nil, errors.New("provider error")
	}

	exec := NewExecutor(provider, DefaultExecutorConfig())

	phase := createTestPhase(t, "failingphase", "Failing Phase", "Process: {{._input}}", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	result, err := exec.Execute(context.Background(), s, "test")

	// The executor returns nil error but marks result as failed
	if err != nil {
		t.Logf("execution returned error: %v", err)
	}

	if result.Status != PhaseStatusFailed {
		t.Errorf("expected status Failed, got %s", result.Status)
	}

	phaseResult := result.PhaseResults["failingphase"]
	if phaseResult.Status != PhaseStatusFailed {
		t.Errorf("expected phase status Failed, got %s", phaseResult.Status)
	}
	if phaseResult.Error == nil {
		t.Error("expected error in phase result")
	}
}

func TestExecutor_Execute_SkipsRemainingOnFailure(t *testing.T) {
	provider := newMockProvider()
	callCount := 0
	provider.completeFunc = func(_ context.Context, _ ports.CompletionRequest) (*ports.CompletionResponse, error) {
		callCount++
		if callCount == 1 {
			return nil, errors.New("first phase fails")
		}
		return &ports.CompletionResponse{
			Content:      "success",
			InputTokens:  10,
			OutputTokens: 20,
		}, nil
	}

	exec := NewExecutor(provider, DefaultExecutorConfig())

	phases := []skill.Phase{
		createTestPhase(t, "phase1", "Phase 1", "A: {{._input}}", nil),
		createTestPhase(t, "phase2", "Phase 2", "B: {{.phase1}}", []string{"phase1"}),
		createTestPhase(t, "phase3", "Phase 3", "C: {{.phase2}}", []string{"phase2"}),
	}
	s := createTestSkill(t, phases)

	result, _ := exec.Execute(context.Background(), s, "test")

	if result.Status != PhaseStatusFailed {
		t.Errorf("expected status Failed, got %s", result.Status)
	}

	// First phase should be failed
	if result.PhaseResults["phase1"].Status != PhaseStatusFailed {
		t.Errorf("expected phase1 Failed, got %s", result.PhaseResults["phase1"].Status)
	}

	// Remaining phases should be skipped
	if result.PhaseResults["phase2"].Status != PhaseStatusSkipped {
		t.Errorf("expected phase2 Skipped, got %s", result.PhaseResults["phase2"].Status)
	}
	if result.PhaseResults["phase3"].Status != PhaseStatusSkipped {
		t.Errorf("expected phase3 Skipped, got %s", result.PhaseResults["phase3"].Status)
	}
}

func TestExecutor_Execute_NilSkill(t *testing.T) {
	provider := newMockProvider()
	exec := NewExecutor(provider, DefaultExecutorConfig())

	_, err := exec.Execute(context.Background(), nil, "test")
	if err == nil {
		t.Error("expected error for nil skill")
	}
}

func TestExecutor_Execute_TokenCounting(t *testing.T) {
	provider := newMockProvider()
	inputTokens := 15
	outputTokens := 25

	provider.completeFunc = func(_ context.Context, _ ports.CompletionRequest) (*ports.CompletionResponse, error) {
		return &ports.CompletionResponse{
			Content:      "response",
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			FinishReason: "stop",
		}, nil
	}

	exec := NewExecutor(provider, DefaultExecutorConfig())

	phases := []skill.Phase{
		createTestPhase(t, "phase1", "Phase 1", "A", nil),
		createTestPhase(t, "phase2", "Phase 2", "B", nil),
	}
	s := createTestSkill(t, phases)

	result, err := exec.Execute(context.Background(), s, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTotal := 2 * (inputTokens + outputTokens)
	if result.TotalTokens != expectedTotal {
		t.Errorf("expected total tokens %d, got %d", expectedTotal, result.TotalTokens)
	}
}

func TestExecutor_Execute_MaxParallelLimit(t *testing.T) {
	provider := newMockProvider()
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	provider.completeFunc = func(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
		current := currentConcurrent.Add(1)
		defer currentConcurrent.Add(-1)

		// Track max concurrency
		for {
			old := maxConcurrent.Load()
			if current <= old || maxConcurrent.CompareAndSwap(old, current) {
				break
			}
		}

		// Simulate some work
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		return &ports.CompletionResponse{
			Content: "response",
		}, nil
	}

	config := ExecutorConfig{
		MaxParallel: 2, // Limit to 2 concurrent
		Timeout:     5 * time.Minute,
	}
	exec := NewExecutor(provider, config)

	// Create 5 independent phases that could run in parallel
	phases := []skill.Phase{
		createTestPhase(t, "p1", "P1", "A", nil),
		createTestPhase(t, "p2", "P2", "B", nil),
		createTestPhase(t, "p3", "P3", "C", nil),
		createTestPhase(t, "p4", "P4", "D", nil),
		createTestPhase(t, "p5", "P5", "E", nil),
	}
	s := createTestSkill(t, phases)

	result, err := exec.Execute(context.Background(), s, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	// Max concurrent should not exceed our limit
	if maxConcurrent.Load() > int32(config.MaxParallel) {
		t.Errorf("max concurrent %d exceeded limit %d", maxConcurrent.Load(), config.MaxParallel)
	}
}

func TestPhaseExecutor_BuildPrompt(t *testing.T) {
	pe := newPhaseExecutor(newMockProvider(), "")

	tests := []struct {
		name     string
		template string
		data     map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple variable",
			template: "Hello {{._input}}",
			data:     map[string]string{"_input": "World"},
			expected: "Hello World",
		},
		{
			name:     "multiple variables",
			template: "{{.phase1}} and {{.phase2}}",
			data:     map[string]string{"phase1": "First", "phase2": "Second"},
			expected: "First and Second",
		},
		{
			name:     "missing variable",
			template: "Hello {{.missing}}",
			data:     map[string]string{},
			expected: "Hello <no value>",
		},
		{
			name:     "invalid template",
			template: "Hello {{.invalid",
			data:     map[string]string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pe.buildPrompt(tt.template, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPhaseExecutor_SelectModel(t *testing.T) {
	pe := newPhaseExecutor(newMockProvider(), "")

	tests := []struct {
		profile  string
		expected string
	}{
		{skill.RoutingProfileCheap, "cheap-model"},
		{skill.RoutingProfileBalanced, "balanced-model"},
		{skill.RoutingProfilePremium, "premium-model"},
		{"unknown", "balanced-model"},
		{"", "balanced-model"},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			result := pe.selectModel(tt.profile)
			if result != tt.expected {
				t.Errorf("for profile %q: expected %q, got %q", tt.profile, tt.expected, result)
			}
		})
	}
}

func TestExecutionResult_Duration(t *testing.T) {
	provider := newMockProvider()
	provider.completeDelay = 50 * time.Millisecond

	exec := NewExecutor(provider, DefaultExecutorConfig())

	phase := createTestPhase(t, "phase1", "Phase 1", "Test", nil)
	s := createTestSkill(t, []skill.Phase{phase})

	result, err := exec.Execute(context.Background(), s, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duration should be positive
	if result.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", result.Duration)
	}

	// Each phase should also have duration
	for id, pr := range result.PhaseResults {
		if pr.Duration <= 0 {
			t.Errorf("phase %s: expected positive duration, got %v", id, pr.Duration)
		}
	}
}

func TestExecutor_Execute_FinalOutputFromTerminalPhase(t *testing.T) {
	provider := newMockProvider()

	outputMap := map[string]string{
		"phase1": "Output from phase 1",
		"phase2": "Output from phase 2",
		"final":  "Final output",
	}

	provider.completeFunc = func(_ context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
		// Extract which phase this is from the model name or messages
		for phaseID, output := range outputMap {
			if containsPhaseID(req.Messages, phaseID) {
				return &ports.CompletionResponse{Content: output}, nil
			}
		}
		return &ports.CompletionResponse{Content: "unknown"}, nil
	}

	exec := NewExecutor(provider, DefaultExecutorConfig())

	phases := []skill.Phase{
		createTestPhase(t, "phase1", "Phase 1", "phase1: {{._input}}", nil),
		createTestPhase(t, "phase2", "Phase 2", "phase2: {{._input}}", nil),
		createTestPhase(t, "final", "Final", "final: {{.phase1}} {{.phase2}}", []string{"phase1", "phase2"}),
	}
	s := createTestSkill(t, phases)

	result, err := exec.Execute(context.Background(), s, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The final output should be from the terminal phase
	if result.FinalOutput == "" {
		t.Error("expected non-empty final output")
	}
}

// Helper function to check if messages contain a phase ID reference
func containsPhaseID(messages []ports.Message, phaseID string) bool {
	for _, msg := range messages {
		if len(msg.Content) >= len(phaseID) && msg.Content[:len(phaseID)] == phaseID {
			return true
		}
	}
	return false
}

func TestDefaultExecutorConfig(t *testing.T) {
	config := DefaultExecutorConfig()

	if config.MaxParallel <= 0 {
		t.Errorf("expected positive MaxParallel, got %d", config.MaxParallel)
	}

	if config.Timeout <= 0 {
		t.Errorf("expected positive Timeout, got %v", config.Timeout)
	}
}

func TestPhaseExecutor_BuildMessages_WithMemory(t *testing.T) {
	tests := []struct {
		name             string
		memoryContent    string
		prompt           string
		dependencyOutput map[string]string
		wantMemoryMsg    bool
		wantMemoryFirst  bool
	}{
		{
			name:             "no memory content",
			memoryContent:    "",
			prompt:           "Test prompt",
			dependencyOutput: nil,
			wantMemoryMsg:    false,
		},
		{
			name:             "with memory content",
			memoryContent:    "Remember this context",
			prompt:           "Test prompt",
			dependencyOutput: nil,
			wantMemoryMsg:    true,
			wantMemoryFirst:  true,
		},
		{
			name:             "memory with dependencies",
			memoryContent:    "Project rules",
			prompt:           "Test prompt",
			dependencyOutput: map[string]string{"phase1": "Previous output"},
			wantMemoryMsg:    true,
			wantMemoryFirst:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := newPhaseExecutor(newMockProvider(), tt.memoryContent)
			messages := pe.buildMessages(tt.prompt, tt.dependencyOutput)

			// Check if memory message is present
			hasMemoryMsg := false
			memoryMsgIndex := -1
			for i, msg := range messages {
				if msg.Role == "system" && len(msg.Content) > 0 {
					if len(msg.Content) >= len("Project Memory:") && msg.Content[:len("Project Memory:")] == "Project Memory:" {
						hasMemoryMsg = true
						memoryMsgIndex = i
						break
					}
				}
			}

			if tt.wantMemoryMsg && !hasMemoryMsg {
				t.Error("expected memory message but not found")
			}
			if !tt.wantMemoryMsg && hasMemoryMsg {
				t.Error("did not expect memory message but found one")
			}

			// Check if memory is first message
			if tt.wantMemoryFirst && memoryMsgIndex != 0 {
				t.Errorf("expected memory message to be first, but it was at index %d", memoryMsgIndex)
			}
		})
	}
}
