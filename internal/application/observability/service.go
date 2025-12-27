// Package observability provides observability services for workflow execution.
// It integrates structured logging, metrics collection, and distributed tracing
// into the workflow execution pipeline.
package observability

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/metrics"
	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/logging"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/tracing"
)

// Service provides observability features for workflow execution.
// It coordinates logging, metrics collection, and tracing.
type Service struct {
	logger         *logging.Logger
	tracer         *tracing.Tracer
	metricsStorage ports.MetricsStoragePort
	costCalc       *provider.CostCalculator
}

// ServiceConfig holds configuration for the observability service.
type ServiceConfig struct {
	Logger         *logging.Logger
	Tracer         *tracing.Tracer
	MetricsStorage ports.MetricsStoragePort
	CostCalculator *provider.CostCalculator
}

// NewService creates a new observability service.
func NewService(cfg ServiceConfig) *Service {
	logger := cfg.Logger
	if logger == nil {
		logger = logging.Default()
	}

	tracer := cfg.Tracer
	if tracer == nil {
		tracer = tracing.Default()
	}

	return &Service{
		logger:         logger,
		tracer:         tracer,
		metricsStorage: cfg.MetricsStorage,
		costCalc:       cfg.CostCalculator,
	}
}

// WorkflowObserver provides observability for a single workflow execution.
type WorkflowObserver struct {
	service       *Service
	executionID   string
	skillID       string
	skillName     string
	correlationID string
	startTime     time.Time
	workflowSpan  *tracing.WorkflowSpan
	phaseRecords  []*metrics.PhaseExecutionRecord
	totalCost     float64
	cacheHits     int
	cacheMisses   int
}

// StartWorkflow begins observing a workflow execution.
func (s *Service) StartWorkflow(ctx context.Context, skillID, skillName string) (context.Context, *WorkflowObserver) {
	executionID := uuid.New().String()
	correlationID := logging.CorrelationIDFromContext(ctx)
	if correlationID == "" {
		correlationID = uuid.New().String()
		ctx = logging.WithCorrelationID(ctx, correlationID)
	}

	// Log workflow start
	logging.LogWorkflowStart(ctx, s.logger, skillID, skillName)

	// Start tracing span
	ctx, workflowSpan := s.tracer.StartWorkflowSpan(ctx, skillID, skillName)

	return ctx, &WorkflowObserver{
		service:       s,
		executionID:   executionID,
		skillID:       skillID,
		skillName:     skillName,
		correlationID: correlationID,
		startTime:     time.Now(),
		workflowSpan:  workflowSpan,
		phaseRecords:  make([]*metrics.PhaseExecutionRecord, 0),
	}
}

// PhaseObserver provides observability for a single phase execution.
type PhaseObserver struct {
	service      *Service
	workflowObs  *WorkflowObserver
	phaseID      string
	phaseName    string
	startTime    time.Time
	phaseSpan    *tracing.PhaseSpan
	providerSpan *tracing.ProviderSpan
}

// StartPhase begins observing a phase execution.
func (wo *WorkflowObserver) StartPhase(ctx context.Context, phaseID, phaseName string) (context.Context, *PhaseObserver) {
	// Log phase start
	wo.service.logger.Debug("phase started",
		"phase_id", phaseID,
		"phase_name", phaseName,
		"skill_id", wo.skillID,
	)

	// Start tracing span
	ctx, phaseSpan := wo.service.tracer.StartPhaseSpan(ctx, phaseID, phaseName)

	return ctx, &PhaseObserver{
		service:     wo.service,
		workflowObs: wo,
		phaseID:     phaseID,
		phaseName:   phaseName,
		startTime:   time.Now(),
		phaseSpan:   phaseSpan,
	}
}

// StartProviderCall begins tracing a provider call within the phase.
func (po *PhaseObserver) StartProviderCall(ctx context.Context, providerName, model string) context.Context {
	ctx, po.providerSpan = po.service.tracer.StartProviderSpan(ctx, providerName, model)
	po.phaseSpan.SetProvider(providerName, model)
	return ctx
}

// EndProviderCall ends the provider call span with results.
func (po *PhaseObserver) EndProviderCall(outputTokens int, finishReason string, err error) {
	if po.providerSpan == nil {
		return
	}

	po.providerSpan.SetResponse(outputTokens, finishReason)

	if err != nil {
		po.providerSpan.EndWithError(err)
	} else {
		po.providerSpan.End()
	}
}

// CompletePhase ends the phase observation with success.
func (po *PhaseObserver) CompletePhase(ctx context.Context, inputTokens, outputTokens int, providerName, model string, cacheHit bool) {
	duration := time.Since(po.startTime)

	// Calculate cost
	var cost float64
	if po.service.costCalc != nil && !cacheHit {
		breakdown := po.service.costCalc.CalculateOrZero(model, inputTokens, outputTokens)
		cost = breakdown.TotalCost
	}

	// Log phase completion
	logging.LogPhaseComplete(ctx, po.service.logger, po.phaseID, inputTokens, outputTokens, duration, cacheHit)

	// Log cost if applicable
	if cost > 0 {
		logging.LogCostIncurred(ctx, po.service.logger, providerName, model, cost, inputTokens, outputTokens)
	}

	// Update phase span
	po.phaseSpan.SetTokens(inputTokens, outputTokens)
	po.phaseSpan.SetCost(cost)
	po.phaseSpan.SetCacheHit(cacheHit)
	po.phaseSpan.End()

	// Track cache stats
	if cacheHit {
		po.workflowObs.cacheHits++
	} else {
		po.workflowObs.cacheMisses++
	}
	po.workflowObs.totalCost += cost

	// Create phase execution record
	phaseRecord := &metrics.PhaseExecutionRecord{
		ID:           uuid.New().String(),
		ExecutionID:  po.workflowObs.executionID,
		PhaseID:      po.phaseID,
		PhaseName:    po.phaseName,
		Status:       "completed",
		Provider:     providerName,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Cost:         cost,
		Duration:     duration,
		CacheHit:     cacheHit,
		StartedAt:    po.startTime,
		CompletedAt:  time.Now(),
	}

	po.workflowObs.phaseRecords = append(po.workflowObs.phaseRecords, phaseRecord)
}

// FailPhase ends the phase observation with an error.
func (po *PhaseObserver) FailPhase(ctx context.Context, err error, providerName, model string) {
	duration := time.Since(po.startTime)

	// Log phase failure
	logging.LogPhaseError(ctx, po.service.logger, po.phaseID, err)

	// End phase span with error
	po.phaseSpan.EndWithError(err)

	// Track cache miss (errors are always cache misses)
	po.workflowObs.cacheMisses++

	// Create phase execution record
	phaseRecord := &metrics.PhaseExecutionRecord{
		ID:           uuid.New().String(),
		ExecutionID:  po.workflowObs.executionID,
		PhaseID:      po.phaseID,
		PhaseName:    po.phaseName,
		Status:       "failed",
		Provider:     providerName,
		Model:        model,
		Duration:     duration,
		StartedAt:    po.startTime,
		CompletedAt:  time.Now(),
		ErrorMessage: err.Error(),
	}

	po.workflowObs.phaseRecords = append(po.workflowObs.phaseRecords, phaseRecord)
}

// CompleteWorkflow ends the workflow observation with success.
func (wo *WorkflowObserver) CompleteWorkflow(ctx context.Context, inputTokens, outputTokens int, primaryModel string) error {
	duration := time.Since(wo.startTime)

	// Log workflow completion
	logging.LogWorkflowComplete(ctx, wo.service.logger, wo.skillID, duration, inputTokens+outputTokens)

	// Update workflow span
	wo.workflowSpan.SetPhaseCount(len(wo.phaseRecords))
	wo.workflowSpan.SetTotalTokens(inputTokens, outputTokens)
	wo.workflowSpan.SetCost(wo.totalCost)
	wo.workflowSpan.SetCacheStats(wo.cacheHits, wo.cacheMisses)
	wo.workflowSpan.End()

	// Save metrics if storage is configured
	if wo.service.metricsStorage != nil {
		return wo.saveMetrics(ctx, "completed", inputTokens, outputTokens, primaryModel, nil)
	}

	return nil
}

// FailWorkflow ends the workflow observation with an error.
func (wo *WorkflowObserver) FailWorkflow(ctx context.Context, err error, inputTokens, outputTokens int, primaryModel string) error {
	duration := time.Since(wo.startTime)

	// Log workflow failure
	logging.LogWorkflowError(ctx, wo.service.logger, wo.skillID, err, duration)

	// End workflow span with error
	wo.workflowSpan.EndWithError(err)

	// Save metrics if storage is configured
	if wo.service.metricsStorage != nil {
		return wo.saveMetrics(ctx, "failed", inputTokens, outputTokens, primaryModel, err)
	}

	return nil
}

// saveMetrics persists the execution and phase metrics.
func (wo *WorkflowObserver) saveMetrics(ctx context.Context, status string, inputTokens, outputTokens int, primaryModel string, _ error) error {
	now := time.Now()

	// Create execution record
	execRecord := &metrics.ExecutionRecord{
		ID:            wo.executionID,
		SkillID:       wo.skillID,
		SkillName:     wo.skillName,
		Status:        status,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		TotalCost:     wo.totalCost,
		Duration:      now.Sub(wo.startTime),
		PhaseCount:    len(wo.phaseRecords),
		CacheHits:     wo.cacheHits,
		CacheMisses:   wo.cacheMisses,
		PrimaryModel:  primaryModel,
		StartedAt:     wo.startTime,
		CompletedAt:   now,
		CorrelationID: wo.correlationID,
	}

	// Save execution record
	if err := wo.service.metricsStorage.SaveExecution(ctx, execRecord); err != nil {
		wo.service.logger.Error("failed to save execution record",
			"error", err,
			"execution_id", wo.executionID,
		)
		return err
	}

	// Save phase records
	for _, phaseRecord := range wo.phaseRecords {
		if err := wo.service.metricsStorage.SavePhaseExecution(ctx, phaseRecord); err != nil {
			wo.service.logger.Error("failed to save phase execution record",
				"error", err,
				"phase_id", phaseRecord.PhaseID,
				"execution_id", wo.executionID,
			)
			// Continue saving other records even if one fails
		}
	}

	return nil
}

// GetExecutionID returns the execution ID for the current workflow.
func (wo *WorkflowObserver) GetExecutionID() string {
	return wo.executionID
}

// GetCorrelationID returns the correlation ID for the current workflow.
func (wo *WorkflowObserver) GetCorrelationID() string {
	return wo.correlationID
}
