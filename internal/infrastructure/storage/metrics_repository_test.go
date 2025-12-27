package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/jbctechsolutions/skillrunner/internal/domain/metrics"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// Create the required tables
	_, err = db.Exec(`
		CREATE TABLE execution_records (
			id TEXT PRIMARY KEY,
			skill_id TEXT NOT NULL,
			skill_name TEXT NOT NULL,
			status TEXT NOT NULL,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			total_cost REAL DEFAULT 0,
			duration_ns INTEGER DEFAULT 0,
			phase_count INTEGER DEFAULT 0,
			cache_hits INTEGER DEFAULT 0,
			cache_misses INTEGER DEFAULT 0,
			primary_model TEXT,
			started_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP NOT NULL,
			correlation_id TEXT
		);

		CREATE TABLE phase_execution_records (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			phase_id TEXT NOT NULL,
			phase_name TEXT NOT NULL,
			status TEXT NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			cost REAL DEFAULT 0,
			duration_ns INTEGER DEFAULT 0,
			cache_hit BOOLEAN DEFAULT 0,
			started_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP NOT NULL,
			error_message TEXT,
			FOREIGN KEY (execution_id) REFERENCES execution_records(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	return db
}

func TestMetricsRepository_SaveExecution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()
	exec := &metrics.ExecutionRecord{
		ID:            "exec-1",
		SkillID:       "code-review",
		SkillName:     "Code Review",
		Status:        "completed",
		InputTokens:   1000,
		OutputTokens:  500,
		TotalCost:     0.015,
		Duration:      5 * time.Second,
		PhaseCount:    3,
		CacheHits:     1,
		CacheMisses:   2,
		PrimaryModel:  "claude-3-5-sonnet",
		StartedAt:     now.Add(-5 * time.Second),
		CompletedAt:   now,
		CorrelationID: "corr-123",
	}

	err := repo.SaveExecution(ctx, exec)
	if err != nil {
		t.Fatalf("failed to save execution: %v", err)
	}

	// Verify it was saved
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM execution_records WHERE id = ?", exec.ID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestMetricsRepository_SaveExecution_NilRecord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	err := repo.SaveExecution(ctx, nil)
	if err == nil {
		t.Error("expected error for nil record, got nil")
	}
}

func TestMetricsRepository_SavePhaseExecution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// First save the parent execution
	exec := &metrics.ExecutionRecord{
		ID:          "exec-1",
		SkillID:     "code-review",
		SkillName:   "Code Review",
		Status:      "completed",
		StartedAt:   now.Add(-5 * time.Second),
		CompletedAt: now,
	}
	if err := repo.SaveExecution(ctx, exec); err != nil {
		t.Fatalf("failed to save execution: %v", err)
	}

	// Now save a phase
	phase := &metrics.PhaseExecutionRecord{
		ID:           "phase-1",
		ExecutionID:  "exec-1",
		PhaseID:      "analysis",
		PhaseName:    "Code Analysis",
		Status:       "completed",
		Provider:     "anthropic",
		Model:        "claude-3-5-sonnet",
		InputTokens:  500,
		OutputTokens: 200,
		Cost:         0.01,
		Duration:     2 * time.Second,
		CacheHit:     false,
		StartedAt:    now.Add(-3 * time.Second),
		CompletedAt:  now.Add(-1 * time.Second),
	}

	err := repo.SavePhaseExecution(ctx, phase)
	if err != nil {
		t.Fatalf("failed to save phase execution: %v", err)
	}

	// Verify it was saved
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM phase_execution_records WHERE id = ?", phase.ID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestMetricsRepository_GetExecutions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Save multiple executions
	executions := []*metrics.ExecutionRecord{
		{
			ID:          "exec-1",
			SkillID:     "code-review",
			SkillName:   "Code Review",
			Status:      "completed",
			StartedAt:   now.Add(-2 * time.Hour),
			CompletedAt: now.Add(-1*time.Hour - 55*time.Minute),
		},
		{
			ID:          "exec-2",
			SkillID:     "test-gen",
			SkillName:   "Test Generator",
			Status:      "completed",
			StartedAt:   now.Add(-1 * time.Hour),
			CompletedAt: now.Add(-55 * time.Minute),
		},
		{
			ID:          "exec-3",
			SkillID:     "code-review",
			SkillName:   "Code Review",
			Status:      "failed",
			StartedAt:   now.Add(-30 * time.Minute),
			CompletedAt: now.Add(-25 * time.Minute),
		},
	}

	for _, exec := range executions {
		if err := repo.SaveExecution(ctx, exec); err != nil {
			t.Fatalf("failed to save execution: %v", err)
		}
	}

	// Test getting all executions
	filter := metrics.MetricsFilter{
		StartDate: now.Add(-3 * time.Hour),
		EndDate:   now,
	}

	results, err := repo.GetExecutions(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 executions, got %d", len(results))
	}

	// Test filtering by skill ID
	filter.SkillID = "code-review"
	results, err = repo.GetExecutions(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 code-review executions, got %d", len(results))
	}

	// Test filtering by status
	filter.SkillID = ""
	filter.Status = "completed"
	results, err = repo.GetExecutions(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 completed executions, got %d", len(results))
	}

	// Test limit
	filter.Status = ""
	filter.Limit = 2
	results, err = repo.GetExecutions(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 executions with limit, got %d", len(results))
	}
}

func TestMetricsRepository_GetAggregatedMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Save test data
	executions := []*metrics.ExecutionRecord{
		{
			ID:           "exec-1",
			SkillID:      "code-review",
			SkillName:    "Code Review",
			Status:       "completed",
			InputTokens:  1000,
			OutputTokens: 500,
			TotalCost:    0.015,
			Duration:     5 * time.Second,
			CacheHits:    2,
			CacheMisses:  1,
			StartedAt:    now.Add(-1 * time.Hour),
			CompletedAt:  now.Add(-55 * time.Minute),
		},
		{
			ID:           "exec-2",
			SkillID:      "test-gen",
			SkillName:    "Test Generator",
			Status:       "completed",
			InputTokens:  800,
			OutputTokens: 400,
			TotalCost:    0.012,
			Duration:     4 * time.Second,
			CacheHits:    1,
			CacheMisses:  1,
			StartedAt:    now.Add(-30 * time.Minute),
			CompletedAt:  now.Add(-26 * time.Minute),
		},
		{
			ID:           "exec-3",
			SkillID:      "code-review",
			SkillName:    "Code Review",
			Status:       "failed",
			InputTokens:  200,
			OutputTokens: 0,
			TotalCost:    0.002,
			Duration:     1 * time.Second,
			CacheHits:    0,
			CacheMisses:  1,
			StartedAt:    now.Add(-10 * time.Minute),
			CompletedAt:  now.Add(-9 * time.Minute),
		},
	}

	for _, exec := range executions {
		if err := repo.SaveExecution(ctx, exec); err != nil {
			t.Fatalf("failed to save execution: %v", err)
		}
	}

	filter := metrics.MetricsFilter{
		StartDate: now.Add(-2 * time.Hour),
		EndDate:   now,
	}

	agg, err := repo.GetAggregatedMetrics(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get aggregated metrics: %v", err)
	}

	// Verify totals
	if agg.TotalExecutions != 3 {
		t.Errorf("expected 3 total executions, got %d", agg.TotalExecutions)
	}
	if agg.SuccessCount != 2 {
		t.Errorf("expected 2 successful executions, got %d", agg.SuccessCount)
	}
	if agg.FailedCount != 1 {
		t.Errorf("expected 1 failed execution, got %d", agg.FailedCount)
	}
	if agg.InputTokens != 2000 {
		t.Errorf("expected 2000 input tokens, got %d", agg.InputTokens)
	}
	if agg.OutputTokens != 900 {
		t.Errorf("expected 900 output tokens, got %d", agg.OutputTokens)
	}

	// Check success rate (approximately 66.67%)
	expectedSuccessRate := 2.0 / 3.0
	if agg.SuccessRate < expectedSuccessRate-0.01 || agg.SuccessRate > expectedSuccessRate+0.01 {
		t.Errorf("expected success rate ~%.2f, got %.2f", expectedSuccessRate, agg.SuccessRate)
	}
}

func TestMetricsRepository_GetProviderMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// First save an execution
	exec := &metrics.ExecutionRecord{
		ID:          "exec-1",
		SkillID:     "code-review",
		SkillName:   "Code Review",
		Status:      "completed",
		StartedAt:   now.Add(-1 * time.Hour),
		CompletedAt: now,
	}
	if err := repo.SaveExecution(ctx, exec); err != nil {
		t.Fatalf("failed to save execution: %v", err)
	}

	// Save phase executions with different providers
	phases := []*metrics.PhaseExecutionRecord{
		{
			ID:           "phase-1",
			ExecutionID:  "exec-1",
			PhaseID:      "analysis",
			PhaseName:    "Analysis",
			Status:       "completed",
			Provider:     "anthropic",
			Model:        "claude-3-5-sonnet",
			InputTokens:  500,
			OutputTokens: 200,
			Cost:         0.01,
			Duration:     2 * time.Second,
			StartedAt:    now.Add(-50 * time.Minute),
			CompletedAt:  now.Add(-48 * time.Minute),
		},
		{
			ID:           "phase-2",
			ExecutionID:  "exec-1",
			PhaseID:      "review",
			PhaseName:    "Review",
			Status:       "completed",
			Provider:     "anthropic",
			Model:        "claude-3-5-sonnet",
			InputTokens:  400,
			OutputTokens: 150,
			Cost:         0.008,
			Duration:     1500 * time.Millisecond,
			StartedAt:    now.Add(-40 * time.Minute),
			CompletedAt:  now.Add(-38 * time.Minute),
		},
		{
			ID:           "phase-3",
			ExecutionID:  "exec-1",
			PhaseID:      "summary",
			PhaseName:    "Summary",
			Status:       "completed",
			Provider:     "ollama",
			Model:        "llama3",
			InputTokens:  300,
			OutputTokens: 100,
			Cost:         0.00,
			Duration:     500 * time.Millisecond,
			CacheHit:     true,
			StartedAt:    now.Add(-30 * time.Minute),
			CompletedAt:  now.Add(-29 * time.Minute),
		},
	}

	for _, phase := range phases {
		if err := repo.SavePhaseExecution(ctx, phase); err != nil {
			t.Fatalf("failed to save phase: %v", err)
		}
	}

	filter := metrics.MetricsFilter{
		StartDate: now.Add(-2 * time.Hour),
		EndDate:   now,
	}

	providerMetrics, err := repo.GetProviderMetrics(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get provider metrics: %v", err)
	}

	if len(providerMetrics) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providerMetrics))
	}

	// Find anthropic metrics
	var anthropicMetrics *metrics.ProviderMetrics
	var ollamaMetrics *metrics.ProviderMetrics
	for i := range providerMetrics {
		if providerMetrics[i].Name == "anthropic" {
			anthropicMetrics = &providerMetrics[i]
		}
		if providerMetrics[i].Name == "ollama" {
			ollamaMetrics = &providerMetrics[i]
		}
	}

	if anthropicMetrics == nil {
		t.Fatal("expected anthropic metrics")
	}
	if anthropicMetrics.TotalRequests != 2 {
		t.Errorf("expected 2 anthropic requests, got %d", anthropicMetrics.TotalRequests)
	}
	if anthropicMetrics.Type != "cloud" {
		t.Errorf("expected anthropic type 'cloud', got '%s'", anthropicMetrics.Type)
	}

	if ollamaMetrics == nil {
		t.Fatal("expected ollama metrics")
	}
	if ollamaMetrics.TotalRequests != 1 {
		t.Errorf("expected 1 ollama request, got %d", ollamaMetrics.TotalRequests)
	}
	if ollamaMetrics.Type != "local" {
		t.Errorf("expected ollama type 'local', got '%s'", ollamaMetrics.Type)
	}
	if ollamaMetrics.CacheHits != 1 {
		t.Errorf("expected 1 cache hit for ollama, got %d", ollamaMetrics.CacheHits)
	}
}

func TestMetricsRepository_GetSkillMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Save test executions
	executions := []*metrics.ExecutionRecord{
		{
			ID:           "exec-1",
			SkillID:      "code-review",
			SkillName:    "Code Review",
			Status:       "completed",
			InputTokens:  1000,
			OutputTokens: 500,
			TotalCost:    0.015,
			Duration:     5 * time.Second,
			StartedAt:    now.Add(-2 * time.Hour),
			CompletedAt:  now.Add(-1*time.Hour - 55*time.Minute),
		},
		{
			ID:           "exec-2",
			SkillID:      "code-review",
			SkillName:    "Code Review",
			Status:       "completed",
			InputTokens:  800,
			OutputTokens: 400,
			TotalCost:    0.012,
			Duration:     4 * time.Second,
			StartedAt:    now.Add(-1 * time.Hour),
			CompletedAt:  now.Add(-56 * time.Minute),
		},
		{
			ID:           "exec-3",
			SkillID:      "test-gen",
			SkillName:    "Test Generator",
			Status:       "failed",
			InputTokens:  200,
			OutputTokens: 0,
			TotalCost:    0.002,
			Duration:     1 * time.Second,
			StartedAt:    now.Add(-30 * time.Minute),
			CompletedAt:  now.Add(-29 * time.Minute),
		},
	}

	for _, exec := range executions {
		if err := repo.SaveExecution(ctx, exec); err != nil {
			t.Fatalf("failed to save execution: %v", err)
		}
	}

	filter := metrics.MetricsFilter{
		StartDate: now.Add(-3 * time.Hour),
		EndDate:   now,
	}

	skillMetrics, err := repo.GetSkillMetrics(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get skill metrics: %v", err)
	}

	if len(skillMetrics) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skillMetrics))
	}

	// Find code-review metrics (should be first due to ORDER BY total_runs DESC)
	var codeReviewMetrics *metrics.SkillMetrics
	for i := range skillMetrics {
		if skillMetrics[i].SkillID == "code-review" {
			codeReviewMetrics = &skillMetrics[i]
			break
		}
	}

	if codeReviewMetrics == nil {
		t.Fatal("expected code-review metrics")
	}
	if codeReviewMetrics.TotalRuns != 2 {
		t.Errorf("expected 2 runs for code-review, got %d", codeReviewMetrics.TotalRuns)
	}
	if codeReviewMetrics.SuccessCount != 2 {
		t.Errorf("expected 2 successful runs, got %d", codeReviewMetrics.SuccessCount)
	}
	if codeReviewMetrics.SuccessRate != 1.0 {
		t.Errorf("expected 100%% success rate, got %.2f", codeReviewMetrics.SuccessRate)
	}
}

func TestMetricsRepository_GetCostSummary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Save execution
	exec := &metrics.ExecutionRecord{
		ID:           "exec-1",
		SkillID:      "code-review",
		SkillName:    "Code Review",
		Status:       "completed",
		InputTokens:  1000,
		OutputTokens: 500,
		TotalCost:    0.025,
		Duration:     5 * time.Second,
		StartedAt:    now.Add(-1 * time.Hour),
		CompletedAt:  now.Add(-55 * time.Minute),
	}
	if err := repo.SaveExecution(ctx, exec); err != nil {
		t.Fatalf("failed to save execution: %v", err)
	}

	// Save phases with costs
	phases := []*metrics.PhaseExecutionRecord{
		{
			ID:           "phase-1",
			ExecutionID:  "exec-1",
			PhaseID:      "analysis",
			PhaseName:    "Analysis",
			Status:       "completed",
			Provider:     "anthropic",
			Model:        "claude-3-5-sonnet",
			InputTokens:  500,
			OutputTokens: 200,
			Cost:         0.015,
			Duration:     2 * time.Second,
			StartedAt:    now.Add(-50 * time.Minute),
			CompletedAt:  now.Add(-48 * time.Minute),
		},
		{
			ID:           "phase-2",
			ExecutionID:  "exec-1",
			PhaseID:      "summary",
			PhaseName:    "Summary",
			Status:       "completed",
			Provider:     "openai",
			Model:        "gpt-4o",
			InputTokens:  500,
			OutputTokens: 300,
			Cost:         0.010,
			Duration:     3 * time.Second,
			StartedAt:    now.Add(-40 * time.Minute),
			CompletedAt:  now.Add(-37 * time.Minute),
		},
	}

	for _, phase := range phases {
		if err := repo.SavePhaseExecution(ctx, phase); err != nil {
			t.Fatalf("failed to save phase: %v", err)
		}
	}

	filter := metrics.MetricsFilter{
		StartDate: now.Add(-2 * time.Hour),
		EndDate:   now,
	}

	summary, err := repo.GetCostSummary(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get cost summary: %v", err)
	}

	if summary.TotalCost != 0.025 {
		t.Errorf("expected total cost 0.025, got %.3f", summary.TotalCost)
	}
	if summary.InputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", summary.InputTokens)
	}
	if summary.OutputTokens != 500 {
		t.Errorf("expected 500 output tokens, got %d", summary.OutputTokens)
	}

	// Check cost by provider
	if len(summary.ByProvider) != 2 {
		t.Errorf("expected 2 providers in cost breakdown, got %d", len(summary.ByProvider))
	}
	if summary.ByProvider["anthropic"] != 0.015 {
		t.Errorf("expected anthropic cost 0.015, got %.3f", summary.ByProvider["anthropic"])
	}
	if summary.ByProvider["openai"] != 0.010 {
		t.Errorf("expected openai cost 0.010, got %.3f", summary.ByProvider["openai"])
	}

	// Check cost by model
	if len(summary.ByModel) != 2 {
		t.Errorf("expected 2 models in cost breakdown, got %d", len(summary.ByModel))
	}
}

func TestMetricsRepository_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	filter := metrics.MetricsFilter{
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now(),
	}

	// Test GetExecutions on empty database
	executions, err := repo.GetExecutions(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get executions from empty db: %v", err)
	}
	if len(executions) != 0 {
		t.Errorf("expected 0 executions, got %d", len(executions))
	}

	// Test GetAggregatedMetrics on empty database
	agg, err := repo.GetAggregatedMetrics(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get aggregated metrics from empty db: %v", err)
	}
	if agg.TotalExecutions != 0 {
		t.Errorf("expected 0 total executions, got %d", agg.TotalExecutions)
	}

	// Test GetCostSummary on empty database
	summary, err := repo.GetCostSummary(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get cost summary from empty db: %v", err)
	}
	if summary.TotalCost != 0 {
		t.Errorf("expected 0 total cost, got %.3f", summary.TotalCost)
	}
}
