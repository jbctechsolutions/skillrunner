package observability

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/metrics"
	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/logging"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/tracing"
)

// mockMetricsStorage implements ports.MetricsStoragePort for testing.
type mockMetricsStorage struct {
	executions []metrics.ExecutionRecord
	phases     []metrics.PhaseExecutionRecord
}

func newMockMetricsStorage() *mockMetricsStorage {
	return &mockMetricsStorage{
		executions: make([]metrics.ExecutionRecord, 0),
		phases:     make([]metrics.PhaseExecutionRecord, 0),
	}
}

func (m *mockMetricsStorage) SaveExecution(ctx context.Context, exec *metrics.ExecutionRecord) error {
	m.executions = append(m.executions, *exec)
	return nil
}

func (m *mockMetricsStorage) SavePhaseExecution(ctx context.Context, phase *metrics.PhaseExecutionRecord) error {
	m.phases = append(m.phases, *phase)
	return nil
}

func (m *mockMetricsStorage) GetExecutions(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.ExecutionRecord, error) {
	return m.executions, nil
}

func (m *mockMetricsStorage) GetAggregatedMetrics(ctx context.Context, filter metrics.MetricsFilter) (*metrics.AggregatedMetrics, error) {
	return nil, nil
}

func (m *mockMetricsStorage) GetProviderMetrics(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.ProviderMetrics, error) {
	return nil, nil
}

func (m *mockMetricsStorage) GetSkillMetrics(ctx context.Context, filter metrics.MetricsFilter) ([]metrics.SkillMetrics, error) {
	return nil, nil
}

func (m *mockMetricsStorage) GetCostSummary(ctx context.Context, filter metrics.MetricsFilter) (*metrics.CostSummary, error) {
	return nil, nil
}

func TestNewService(t *testing.T) {
	service := NewService(ServiceConfig{})

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	if service.logger == nil {
		t.Error("expected non-nil logger")
	}

	if service.tracer == nil {
		t.Error("expected non-nil tracer")
	}
}

func TestStartWorkflow(t *testing.T) {
	ctx := context.Background()
	logBuf := &bytes.Buffer{}
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
		Output: logBuf,
	})

	service := NewService(ServiceConfig{
		Logger: logger,
	})

	ctx, observer := service.StartWorkflow(ctx, "test-skill", "Test Skill")

	if observer == nil {
		t.Fatal("expected non-nil observer")
	}

	if observer.executionID == "" {
		t.Error("expected non-empty execution ID")
	}

	if observer.skillID != "test-skill" {
		t.Errorf("expected skill ID 'test-skill', got %s", observer.skillID)
	}

	if observer.skillName != "Test Skill" {
		t.Errorf("expected skill name 'Test Skill', got %s", observer.skillName)
	}

	if observer.correlationID == "" {
		t.Error("expected non-empty correlation ID")
	}

	// Check that log was written
	if logBuf.Len() == 0 {
		t.Error("expected log output")
	}

	_ = ctx // Use ctx
}

func TestStartPhase(t *testing.T) {
	ctx := context.Background()
	logBuf := &bytes.Buffer{}
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
		Output: logBuf,
	})

	service := NewService(ServiceConfig{
		Logger: logger,
	})

	ctx, workflowObs := service.StartWorkflow(ctx, "test-skill", "Test Skill")
	ctx, phaseObs := workflowObs.StartPhase(ctx, "phase-1", "Analysis Phase")

	if phaseObs == nil {
		t.Fatal("expected non-nil phase observer")
	}

	if phaseObs.phaseID != "phase-1" {
		t.Errorf("expected phase ID 'phase-1', got %s", phaseObs.phaseID)
	}

	if phaseObs.phaseName != "Analysis Phase" {
		t.Errorf("expected phase name 'Analysis Phase', got %s", phaseObs.phaseName)
	}

	_ = ctx
}

func TestCompletePhaseWithMetrics(t *testing.T) {
	ctx := context.Background()
	logBuf := &bytes.Buffer{}
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
		Output: logBuf,
	})

	mockStorage := newMockMetricsStorage()

	// Set up cost calculator with a test model
	costCalc := provider.NewCostCalculator()
	costCalc.RegisterModelWithProvider("claude-3-5-sonnet", "anthropic", 0.003, 0.015)

	service := NewService(ServiceConfig{
		Logger:         logger,
		MetricsStorage: mockStorage,
		CostCalculator: costCalc,
	})

	ctx, workflowObs := service.StartWorkflow(ctx, "test-skill", "Test Skill")
	ctx, phaseObs := workflowObs.StartPhase(ctx, "phase-1", "Analysis Phase")

	// Simulate some processing time
	time.Sleep(10 * time.Millisecond)

	// Complete the phase
	phaseObs.CompletePhase(ctx, 1000, 500, "anthropic", "claude-3-5-sonnet", false)

	// Check that phase record was created
	if len(workflowObs.phaseRecords) != 1 {
		t.Fatalf("expected 1 phase record, got %d", len(workflowObs.phaseRecords))
	}

	record := workflowObs.phaseRecords[0]
	if record.PhaseID != "phase-1" {
		t.Errorf("expected phase ID 'phase-1', got %s", record.PhaseID)
	}
	if record.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", record.Status)
	}
	if record.InputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", record.InputTokens)
	}
	if record.OutputTokens != 500 {
		t.Errorf("expected 500 output tokens, got %d", record.OutputTokens)
	}
	if record.CacheHit {
		t.Error("expected cache hit to be false")
	}

	// Cost should be calculated: (1000/1000)*0.003 + (500/1000)*0.015 = 0.003 + 0.0075 = 0.0105
	expectedCost := 0.0105
	if record.Cost < expectedCost-0.001 || record.Cost > expectedCost+0.001 {
		t.Errorf("expected cost ~%f, got %f", expectedCost, record.Cost)
	}

	// Check cache stats
	if workflowObs.cacheMisses != 1 {
		t.Errorf("expected 1 cache miss, got %d", workflowObs.cacheMisses)
	}
}

func TestCompletePhaseWithCacheHit(t *testing.T) {
	ctx := context.Background()
	logBuf := &bytes.Buffer{}
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
		Output: logBuf,
	})

	costCalc := provider.NewCostCalculator()
	costCalc.RegisterModelWithProvider("claude-3-5-sonnet", "anthropic", 0.003, 0.015)

	service := NewService(ServiceConfig{
		Logger:         logger,
		CostCalculator: costCalc,
	})

	ctx, workflowObs := service.StartWorkflow(ctx, "test-skill", "Test Skill")
	ctx, phaseObs := workflowObs.StartPhase(ctx, "phase-1", "Analysis Phase")

	// Complete with cache hit - should have zero cost
	phaseObs.CompletePhase(ctx, 1000, 500, "anthropic", "claude-3-5-sonnet", true)

	record := workflowObs.phaseRecords[0]
	if record.Cost != 0 {
		t.Errorf("expected zero cost for cache hit, got %f", record.Cost)
	}

	if !record.CacheHit {
		t.Error("expected cache hit to be true")
	}

	if workflowObs.cacheHits != 1 {
		t.Errorf("expected 1 cache hit, got %d", workflowObs.cacheHits)
	}

	_ = ctx
}

func TestCompleteWorkflow(t *testing.T) {
	ctx := context.Background()
	logBuf := &bytes.Buffer{}
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
		Output: logBuf,
	})

	mockStorage := newMockMetricsStorage()

	service := NewService(ServiceConfig{
		Logger:         logger,
		MetricsStorage: mockStorage,
	})

	ctx, workflowObs := service.StartWorkflow(ctx, "test-skill", "Test Skill")
	ctx, phaseObs := workflowObs.StartPhase(ctx, "phase-1", "Analysis Phase")
	phaseObs.CompletePhase(ctx, 1000, 500, "anthropic", "claude-3-5-sonnet", false)

	// Complete workflow
	err := workflowObs.CompleteWorkflow(ctx, 1000, 500, "claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that execution was saved
	if len(mockStorage.executions) != 1 {
		t.Fatalf("expected 1 execution record, got %d", len(mockStorage.executions))
	}

	exec := mockStorage.executions[0]
	if exec.SkillID != "test-skill" {
		t.Errorf("expected skill ID 'test-skill', got %s", exec.SkillID)
	}
	if exec.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", exec.Status)
	}
	if exec.PhaseCount != 1 {
		t.Errorf("expected phase count 1, got %d", exec.PhaseCount)
	}

	// Check that phase was saved
	if len(mockStorage.phases) != 1 {
		t.Fatalf("expected 1 phase record, got %d", len(mockStorage.phases))
	}
}

func TestFailWorkflow(t *testing.T) {
	ctx := context.Background()
	logBuf := &bytes.Buffer{}
	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
		Output: logBuf,
	})

	mockStorage := newMockMetricsStorage()

	service := NewService(ServiceConfig{
		Logger:         logger,
		MetricsStorage: mockStorage,
	})

	ctx, workflowObs := service.StartWorkflow(ctx, "test-skill", "Test Skill")

	// Fail workflow
	testErr := provider.ErrModelNotFound
	err := workflowObs.FailWorkflow(ctx, testErr, 500, 0, "claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that execution was saved with failed status
	if len(mockStorage.executions) != 1 {
		t.Fatalf("expected 1 execution record, got %d", len(mockStorage.executions))
	}

	exec := mockStorage.executions[0]
	if exec.Status != "failed" {
		t.Errorf("expected status 'failed', got %s", exec.Status)
	}
}

func TestWithTracing(t *testing.T) {
	ctx := context.Background()
	logBuf := &bytes.Buffer{}
	traceBuf := &bytes.Buffer{}

	logger := logging.New(logging.Config{
		Level:  logging.LevelDebug,
		Format: logging.FormatText,
		Output: logBuf,
	})

	tracer, err := tracing.New(ctx, tracing.Config{
		Enabled:      true,
		ExporterType: tracing.ExporterStdout,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Output:       traceBuf,
	})
	if err != nil {
		t.Fatalf("failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(ctx)

	service := NewService(ServiceConfig{
		Logger: logger,
		Tracer: tracer,
	})

	ctx, workflowObs := service.StartWorkflow(ctx, "test-skill", "Test Skill")
	ctx, phaseObs := workflowObs.StartPhase(ctx, "phase-1", "Analysis Phase")
	phaseObs.CompletePhase(ctx, 1000, 500, "anthropic", "claude-3-5-sonnet", false)
	_ = workflowObs.CompleteWorkflow(ctx, 1000, 500, "claude-3-5-sonnet")

	// Shutdown tracer to flush spans
	tracer.Shutdown(ctx)

	// Check that traces were written
	if traceBuf.Len() == 0 {
		t.Error("expected trace output")
	}
}

func TestGetExecutionID(t *testing.T) {
	ctx := context.Background()
	service := NewService(ServiceConfig{})

	_, workflowObs := service.StartWorkflow(ctx, "test-skill", "Test Skill")

	executionID := workflowObs.GetExecutionID()
	if executionID == "" {
		t.Error("expected non-empty execution ID")
	}

	// Should be a valid UUID
	if len(executionID) != 36 {
		t.Errorf("expected UUID format (36 chars), got %d chars", len(executionID))
	}
}

func TestGetCorrelationID(t *testing.T) {
	ctx := context.Background()
	service := NewService(ServiceConfig{})

	// Test without pre-existing correlation ID
	_, workflowObs := service.StartWorkflow(ctx, "test-skill", "Test Skill")

	correlationID := workflowObs.GetCorrelationID()
	if correlationID == "" {
		t.Error("expected non-empty correlation ID")
	}

	// Test with pre-existing correlation ID
	existingCorrelationID := "existing-correlation-id"
	ctx = logging.WithCorrelationID(context.Background(), existingCorrelationID)
	_, workflowObs2 := service.StartWorkflow(ctx, "test-skill", "Test Skill")

	if workflowObs2.GetCorrelationID() != existingCorrelationID {
		t.Errorf("expected correlation ID %s, got %s", existingCorrelationID, workflowObs2.GetCorrelationID())
	}
}
