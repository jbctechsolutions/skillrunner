package workflow

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// mockStreamingProvider implements ports.ProviderPort for testing streaming.
type mockStreamingProvider struct {
	streamChunks []string
	streamDelay  time.Duration
	inputTokens  int
	outputTokens int
	modelUsed    string
	shouldError  bool
	errorMessage string
}

func newMockStreamingProvider(chunks []string) *mockStreamingProvider {
	return &mockStreamingProvider{
		streamChunks: chunks,
		streamDelay:  10 * time.Millisecond,
		inputTokens:  100,
		outputTokens: 50,
		modelUsed:    "test-model",
	}
}

func (m *mockStreamingProvider) Info() ports.ProviderInfo {
	return ports.ProviderInfo{
		Name:        "mock-streaming",
		Description: "Mock provider for streaming tests",
		BaseURL:     "http://localhost:0",
		IsLocal:     true,
	}
}

func (m *mockStreamingProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"test-model"}, nil
}

func (m *mockStreamingProvider) SupportsModel(ctx context.Context, modelID string) (bool, error) {
	return modelID == "test-model", nil
}

func (m *mockStreamingProvider) IsAvailable(ctx context.Context, modelID string) (bool, error) {
	return true, nil
}

func (m *mockStreamingProvider) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	var content string
	for _, chunk := range m.streamChunks {
		content += chunk
	}

	return &ports.CompletionResponse{
		Content:      content,
		InputTokens:  m.inputTokens,
		OutputTokens: m.outputTokens,
		FinishReason: "stop",
		ModelUsed:    m.modelUsed,
		Duration:     100 * time.Millisecond,
	}, nil
}

func (m *mockStreamingProvider) Stream(ctx context.Context, req ports.CompletionRequest, cb ports.StreamCallback) (*ports.CompletionResponse, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	var content string
	for _, chunk := range m.streamChunks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		content += chunk
		if err := cb(chunk); err != nil {
			return nil, err
		}

		if m.streamDelay > 0 {
			time.Sleep(m.streamDelay)
		}
	}

	return &ports.CompletionResponse{
		Content:      content,
		InputTokens:  m.inputTokens,
		OutputTokens: m.outputTokens,
		FinishReason: "stop",
		ModelUsed:    m.modelUsed,
		Duration:     time.Duration(len(m.streamChunks)) * m.streamDelay,
	}, nil
}

func (m *mockStreamingProvider) HealthCheck(ctx context.Context, modelID string) (*ports.HealthStatus, error) {
	return &ports.HealthStatus{
		Healthy:     true,
		Message:     "OK",
		Latency:     10 * time.Millisecond,
		LastChecked: time.Now(),
	}, nil
}

func TestStreamingExecutor_ExecuteWithStreaming(t *testing.T) {
	chunks := []string{"Hello", " ", "World", "!"}
	provider := newMockStreamingProvider(chunks)

	config := ExecutorConfig{
		MaxParallel: 2,
		Timeout:     10 * time.Second,
	}
	executor := NewStreamingExecutor(provider, config)

	// Create a simple skill with one phase
	sk, err := skill.NewSkill(
		"test-skill",
		"Test Skill",
		"1.0.0",
		[]skill.Phase{
			{
				ID:             "phase1",
				Name:           "Test Phase",
				RoutingProfile: skill.RoutingProfileBalanced,
				PromptTemplate: "{{._input}}",
				MaxTokens:      100,
				Temperature:    0.7,
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to create skill: %v", err)
	}

	var receivedEvents []StreamEvent
	var mu sync.Mutex

	callback := func(event StreamEvent) error {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		return nil
	}

	ctx := context.Background()
	result, err := executor.ExecuteWithStreaming(ctx, sk, "test input", callback)
	if err != nil {
		t.Fatalf("ExecuteWithStreaming failed: %v", err)
	}

	// Verify result
	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	expectedOutput := "Hello World!"
	if result.FinalOutput != expectedOutput {
		t.Errorf("expected output %q, got %q", expectedOutput, result.FinalOutput)
	}

	// Verify events were received
	if len(receivedEvents) == 0 {
		t.Error("expected to receive streaming events")
	}

	// Check for workflow started event
	hasWorkflowStarted := false
	hasPhaseStarted := false
	hasPhaseCompleted := false
	hasWorkflowCompleted := false
	progressCount := 0

	for _, event := range receivedEvents {
		switch event.Type {
		case EventWorkflowStarted:
			hasWorkflowStarted = true
		case EventPhaseStarted:
			hasPhaseStarted = true
		case EventPhaseProgress:
			progressCount++
		case EventPhaseCompleted:
			hasPhaseCompleted = true
		case EventWorkflowCompleted:
			hasWorkflowCompleted = true
		}
	}

	if !hasWorkflowStarted {
		t.Error("expected WorkflowStarted event")
	}
	if !hasPhaseStarted {
		t.Error("expected PhaseStarted event")
	}
	if !hasPhaseCompleted {
		t.Error("expected PhaseCompleted event")
	}
	if !hasWorkflowCompleted {
		t.Error("expected WorkflowCompleted event")
	}
	if progressCount < len(chunks) {
		t.Errorf("expected at least %d progress events, got %d", len(chunks), progressCount)
	}
}

func TestStreamingExecutor_MultiPhase(t *testing.T) {
	chunks := []string{"Result", " ", "text"}
	provider := newMockStreamingProvider(chunks)

	config := ExecutorConfig{
		MaxParallel: 2,
		Timeout:     10 * time.Second,
	}
	executor := NewStreamingExecutor(provider, config)

	// Create a skill with multiple phases
	sk, err := skill.NewSkill(
		"multi-phase-skill",
		"Multi Phase Skill",
		"1.0.0",
		[]skill.Phase{
			{
				ID:             "phase1",
				Name:           "Phase 1",
				RoutingProfile: skill.RoutingProfileBalanced,
				PromptTemplate: "{{._input}}",
				MaxTokens:      100,
				Temperature:    0.7,
			},
			{
				ID:             "phase2",
				Name:           "Phase 2",
				RoutingProfile: skill.RoutingProfileBalanced,
				DependsOn:      []string{"phase1"},
				PromptTemplate: "Process: {{.phase1}}",
				MaxTokens:      100,
				Temperature:    0.7,
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to create skill: %v", err)
	}

	phaseStartedCount := 0
	phaseCompletedCount := 0
	var mu sync.Mutex

	callback := func(event StreamEvent) error {
		mu.Lock()
		defer mu.Unlock()
		switch event.Type {
		case EventPhaseStarted:
			phaseStartedCount++
		case EventPhaseCompleted:
			phaseCompletedCount++
		}
		return nil
	}

	ctx := context.Background()
	result, err := executor.ExecuteWithStreaming(ctx, sk, "test input", callback)
	if err != nil {
		t.Fatalf("ExecuteWithStreaming failed: %v", err)
	}

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	if phaseStartedCount != 2 {
		t.Errorf("expected 2 phase started events, got %d", phaseStartedCount)
	}

	if phaseCompletedCount != 2 {
		t.Errorf("expected 2 phase completed events, got %d", phaseCompletedCount)
	}
}

func TestStreamingExecutor_TokenCounting(t *testing.T) {
	chunks := []string{"Token", " ", "count", " ", "test"}
	provider := newMockStreamingProvider(chunks)
	provider.inputTokens = 50
	provider.outputTokens = 25

	config := ExecutorConfig{
		MaxParallel: 1,
		Timeout:     10 * time.Second,
	}
	executor := NewStreamingExecutor(provider, config)

	sk, err := skill.NewSkill(
		"token-test",
		"Token Test",
		"1.0.0",
		[]skill.Phase{
			{
				ID:             "phase1",
				Name:           "Phase 1",
				RoutingProfile: skill.RoutingProfileBalanced,
				PromptTemplate: "{{._input}}",
				MaxTokens:      100,
				Temperature:    0.7,
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to create skill: %v", err)
	}

	var tokenUpdates []StreamEvent
	var mu sync.Mutex

	callback := func(event StreamEvent) error {
		mu.Lock()
		defer mu.Unlock()
		if event.Type == EventTokenUpdate || event.Type == EventWorkflowCompleted {
			tokenUpdates = append(tokenUpdates, event)
		}
		return nil
	}

	ctx := context.Background()
	result, err := executor.ExecuteWithStreaming(ctx, sk, "test", callback)
	if err != nil {
		t.Fatalf("ExecuteWithStreaming failed: %v", err)
	}

	// Check total tokens in result
	expectedTotal := 50 + 25 // input + output from mock
	if result.TotalTokens != expectedTotal {
		t.Errorf("expected total tokens %d, got %d", expectedTotal, result.TotalTokens)
	}

	// Check we got token updates
	if len(tokenUpdates) == 0 {
		t.Error("expected token update events")
	}
}

func TestStreamingExecutor_Cancellation(t *testing.T) {
	// Create provider with slow streaming
	chunks := make([]string, 100)
	for i := range chunks {
		chunks[i] = "chunk "
	}
	provider := newMockStreamingProvider(chunks)
	provider.streamDelay = 100 * time.Millisecond

	config := ExecutorConfig{
		MaxParallel: 1,
		Timeout:     10 * time.Second,
	}
	executor := NewStreamingExecutor(provider, config)

	sk, err := skill.NewSkill(
		"cancel-test",
		"Cancel Test",
		"1.0.0",
		[]skill.Phase{
			{
				ID:             "phase1",
				Name:           "Phase 1",
				RoutingProfile: skill.RoutingProfileBalanced,
				PromptTemplate: "{{._input}}",
				MaxTokens:      100,
				Temperature:    0.7,
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to create skill: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	_, err = executor.ExecuteWithStreaming(ctx, sk, "test", nil)
	if err == nil {
		t.Error("expected error due to cancellation")
	}
}

func TestStreamingExecutor_NilSkill(t *testing.T) {
	provider := newMockStreamingProvider([]string{"test"})
	executor := NewStreamingExecutor(provider, DefaultExecutorConfig())

	_, err := executor.ExecuteWithStreaming(context.Background(), nil, "test", nil)
	if err == nil {
		t.Error("expected error for nil skill")
	}
}

func TestStreamingPhaseExecutor_Execute(t *testing.T) {
	chunks := []string{"Hello", " ", "streaming", " ", "world"}
	provider := newMockStreamingProvider(chunks)

	executor := newStreamingPhaseExecutor(provider, "")

	phase := &skill.Phase{
		ID:             "test-phase",
		Name:           "Test Phase",
		PromptTemplate: "Input: {{._input}}",
		MaxTokens:      100,
		Temperature:    0.5,
	}

	deps := map[string]string{
		"_input": "test input",
	}

	var receivedChunks []string
	var mu sync.Mutex

	callback := func(chunk string, inputTokens, outputTokens int) error {
		if chunk != "" {
			mu.Lock()
			receivedChunks = append(receivedChunks, chunk)
			mu.Unlock()
		}
		return nil
	}

	result := executor.ExecuteWithStreaming(context.Background(), phase, deps, callback)

	if result.Status != PhaseStatusCompleted {
		t.Errorf("expected status Completed, got %s", result.Status)
	}

	expectedOutput := "Hello streaming world"
	if result.Output != expectedOutput {
		t.Errorf("expected output %q, got %q", expectedOutput, result.Output)
	}

	if len(receivedChunks) != len(chunks) {
		t.Errorf("expected %d chunks, got %d", len(chunks), len(receivedChunks))
	}
}

func TestStreamingPhaseExecutor_Error(t *testing.T) {
	provider := newMockStreamingProvider([]string{"test"})
	provider.shouldError = true
	provider.errorMessage = "streaming failed"

	executor := newStreamingPhaseExecutor(provider, "")

	phase := &skill.Phase{
		ID:             "error-phase",
		Name:           "Error Phase",
		PromptTemplate: "{{._input}}",
		MaxTokens:      100,
		Temperature:    0.5,
	}

	deps := map[string]string{"_input": "test"}

	result := executor.ExecuteWithStreaming(context.Background(), phase, deps, nil)

	if result.Status != PhaseStatusFailed {
		t.Errorf("expected status Failed, got %s", result.Status)
	}

	if result.Error == nil {
		t.Error("expected error in result")
	}
}
