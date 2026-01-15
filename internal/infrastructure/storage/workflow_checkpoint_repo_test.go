package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

func setupWorkflowCheckpointTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// Create the required table
	_, err = db.Exec(`
		CREATE TABLE workflow_checkpoints (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			skill_id TEXT NOT NULL,
			skill_name TEXT NOT NULL,
			input TEXT NOT NULL,
			input_hash TEXT NOT NULL,
			completed_batch INTEGER DEFAULT -1,
			total_batches INTEGER NOT NULL,
			phase_results TEXT,
			phase_outputs TEXT,
			status TEXT NOT NULL DEFAULT 'in_progress',
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			machine_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_wf_checkpoint_skill_input ON workflow_checkpoints(skill_id, input_hash);
		CREATE INDEX idx_wf_checkpoint_status ON workflow_checkpoints(status);
		CREATE INDEX idx_wf_checkpoint_machine ON workflow_checkpoints(machine_id);
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return db
}

func createTestCheckpoint(t *testing.T, id string) *workflow.WorkflowCheckpoint {
	t.Helper()
	cp, err := workflow.NewWorkflowCheckpoint(id, "exec-"+id, "skill-1", "Test Skill", "test input", 3)
	if err != nil {
		t.Fatalf("failed to create checkpoint: %v", err)
	}
	cp.SetMachineID("machine-1")
	return cp
}

func TestWorkflowCheckpointRepository_Create(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	cp := createTestCheckpoint(t, "cp-1")

	err := repo.Create(ctx, cp)
	if err != nil {
		t.Fatalf("failed to create checkpoint: %v", err)
	}

	// Verify it was saved
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM workflow_checkpoints WHERE id = ?", cp.ID()).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestWorkflowCheckpointRepository_Create_Duplicate(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	cp := createTestCheckpoint(t, "cp-1")

	err := repo.Create(ctx, cp)
	if err != nil {
		t.Fatalf("failed to create first checkpoint: %v", err)
	}

	// Try to create duplicate
	err = repo.Create(ctx, cp)
	if err == nil {
		t.Error("expected error for duplicate, got nil")
	}
}

func TestWorkflowCheckpointRepository_Get(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Create checkpoint
	cp := createTestCheckpoint(t, "cp-1")
	cp.AddPhaseResult("phase-1", &workflow.PhaseResultData{
		PhaseID:   "phase-1",
		PhaseName: "Phase One",
		Status:    "completed",
		Output:    "output1",
	})
	cp.AddPhaseOutput("phase-1", "output1")
	_ = cp.UpdateBatch(0)
	cp.UpdateTokens(100, 50)

	if err := repo.Create(ctx, cp); err != nil {
		t.Fatalf("failed to create checkpoint: %v", err)
	}

	// Get checkpoint
	got, err := repo.Get(ctx, "cp-1")
	if err != nil {
		t.Fatalf("failed to get checkpoint: %v", err)
	}

	// Verify fields
	if got.ID() != cp.ID() {
		t.Errorf("expected ID %s, got %s", cp.ID(), got.ID())
	}
	if got.ExecutionID() != cp.ExecutionID() {
		t.Errorf("expected ExecutionID %s, got %s", cp.ExecutionID(), got.ExecutionID())
	}
	if got.SkillID() != cp.SkillID() {
		t.Errorf("expected SkillID %s, got %s", cp.SkillID(), got.SkillID())
	}
	if got.SkillName() != cp.SkillName() {
		t.Errorf("expected SkillName %s, got %s", cp.SkillName(), got.SkillName())
	}
	if got.Input() != cp.Input() {
		t.Errorf("expected Input %s, got %s", cp.Input(), got.Input())
	}
	if got.CompletedBatch() != cp.CompletedBatch() {
		t.Errorf("expected CompletedBatch %d, got %d", cp.CompletedBatch(), got.CompletedBatch())
	}
	if got.TotalBatches() != cp.TotalBatches() {
		t.Errorf("expected TotalBatches %d, got %d", cp.TotalBatches(), got.TotalBatches())
	}
	if got.InputTokens() != cp.InputTokens() {
		t.Errorf("expected InputTokens %d, got %d", cp.InputTokens(), got.InputTokens())
	}
	if len(got.PhaseResults()) != 1 {
		t.Errorf("expected 1 phase result, got %d", len(got.PhaseResults()))
	}
	if len(got.PhaseOutputs()) != 1 {
		t.Errorf("expected 1 phase output, got %d", len(got.PhaseOutputs()))
	}
}

func TestWorkflowCheckpointRepository_Get_NotFound(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	_, err := repo.Get(ctx, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent checkpoint, got nil")
	}
}

func TestWorkflowCheckpointRepository_GetLatestInProgress(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Insert checkpoints directly with explicit timestamps to ensure ordering
	inputHash := workflow.HashInput("input")
	oldTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	newTime := time.Now().Format(time.RFC3339)

	// Insert older checkpoint
	_, err := db.Exec(`
		INSERT INTO workflow_checkpoints
		(id, execution_id, skill_id, skill_name, input, input_hash, total_batches, status, machine_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "cp-1", "exec-1", "skill-1", "Skill", "input", inputHash, 3, "in_progress", "machine-1", oldTime, oldTime)
	if err != nil {
		t.Fatalf("failed to insert cp-1: %v", err)
	}

	// Insert newer checkpoint
	_, err = db.Exec(`
		INSERT INTO workflow_checkpoints
		(id, execution_id, skill_id, skill_name, input, input_hash, total_batches, status, machine_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "cp-2", "exec-2", "skill-1", "Skill", "input", inputHash, 3, "in_progress", "machine-1", newTime, newTime)
	if err != nil {
		t.Fatalf("failed to insert cp-2: %v", err)
	}

	// Should return the most recent (cp2)
	got, err := repo.GetLatestInProgress(ctx, "skill-1", inputHash)
	if err != nil {
		t.Fatalf("failed to get latest: %v", err)
	}
	if got == nil {
		t.Fatal("expected checkpoint, got nil")
	}
	if got.ID() != "cp-2" {
		t.Errorf("expected cp-2, got %s", got.ID())
	}
}

func TestWorkflowCheckpointRepository_GetLatestInProgress_NoMatch(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	got, err := repo.GetLatestInProgress(ctx, "skill-1", "nonexistent-hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got checkpoint %s", got.ID())
	}
}

func TestWorkflowCheckpointRepository_GetLatestInProgress_OnlyInProgress(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Create completed checkpoint
	cp1, _ := workflow.NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)
	cp1.MarkCompleted()
	_ = repo.Create(ctx, cp1)

	// Should return nil (no in-progress checkpoints)
	got, err := repo.GetLatestInProgress(ctx, "skill-1", workflow.HashInput("input"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for completed checkpoint, got %s", got.ID())
	}
}

func TestWorkflowCheckpointRepository_GetByExecutionID(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Create checkpoints with same execution ID
	cp1, _ := workflow.NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)
	_ = repo.Create(ctx, cp1)

	cp2, _ := workflow.NewWorkflowCheckpoint("cp-2", "exec-1", "skill-1", "Skill", "input", 3)
	_ = repo.Create(ctx, cp2)

	cp3, _ := workflow.NewWorkflowCheckpoint("cp-3", "exec-2", "skill-1", "Skill", "input", 3)
	_ = repo.Create(ctx, cp3)

	checkpoints, err := repo.GetByExecutionID(ctx, "exec-1")
	if err != nil {
		t.Fatalf("failed to get by execution ID: %v", err)
	}
	if len(checkpoints) != 2 {
		t.Errorf("expected 2 checkpoints, got %d", len(checkpoints))
	}
}

func TestWorkflowCheckpointRepository_Update(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Create checkpoint
	cp := createTestCheckpoint(t, "cp-1")
	if err := repo.Create(ctx, cp); err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Update checkpoint
	_ = cp.UpdateBatch(1)
	cp.AddPhaseResult("phase-1", &workflow.PhaseResultData{PhaseID: "phase-1", Status: "completed"})
	cp.AddPhaseOutput("phase-1", "output1")
	cp.UpdateTokens(200, 100)

	if err := repo.Update(ctx, cp); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify update
	got, _ := repo.Get(ctx, "cp-1")
	if got.CompletedBatch() != 1 {
		t.Errorf("expected CompletedBatch 1, got %d", got.CompletedBatch())
	}
	if got.InputTokens() != 200 {
		t.Errorf("expected InputTokens 200, got %d", got.InputTokens())
	}
}

func TestWorkflowCheckpointRepository_Update_NotFound(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	cp := createTestCheckpoint(t, "cp-nonexistent")

	err := repo.Update(ctx, cp)
	if err == nil {
		t.Error("expected error for non-existent checkpoint, got nil")
	}
}

func TestWorkflowCheckpointRepository_List(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Create checkpoints
	cp1, _ := workflow.NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill A", "input", 3)
	cp1.SetMachineID("machine-1")
	_ = repo.Create(ctx, cp1)

	cp2, _ := workflow.NewWorkflowCheckpoint("cp-2", "exec-2", "skill-2", "Skill B", "input", 3)
	cp2.SetMachineID("machine-1")
	_ = repo.Create(ctx, cp2)

	cp3, _ := workflow.NewWorkflowCheckpoint("cp-3", "exec-3", "skill-1", "Skill A", "input", 3)
	cp3.SetMachineID("machine-2")
	cp3.MarkCompleted()
	_ = repo.Create(ctx, cp3)

	// List all
	all, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 checkpoints, got %d", len(all))
	}

	// Filter by skill
	filtered, err := repo.List(ctx, &ports.WorkflowCheckpointFilter{SkillID: "skill-1"})
	if err != nil {
		t.Fatalf("failed to list with filter: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 checkpoints for skill-1, got %d", len(filtered))
	}

	// Filter by status
	filtered, err = repo.List(ctx, &ports.WorkflowCheckpointFilter{
		Status: []workflow.CheckpointStatus{workflow.CheckpointStatusInProgress},
	})
	if err != nil {
		t.Fatalf("failed to list with status filter: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 in-progress checkpoints, got %d", len(filtered))
	}

	// Filter by machine
	filtered, err = repo.List(ctx, &ports.WorkflowCheckpointFilter{MachineID: "machine-1"})
	if err != nil {
		t.Fatalf("failed to list with machine filter: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 checkpoints for machine-1, got %d", len(filtered))
	}

	// Test limit
	filtered, err = repo.List(ctx, &ports.WorkflowCheckpointFilter{Limit: 2})
	if err != nil {
		t.Fatalf("failed to list with limit: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 checkpoints with limit, got %d", len(filtered))
	}
}

func TestWorkflowCheckpointRepository_Delete(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	cp := createTestCheckpoint(t, "cp-1")
	_ = repo.Create(ctx, cp)

	err := repo.Delete(ctx, "cp-1")
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify deleted
	_, err = repo.Get(ctx, "cp-1")
	if err == nil {
		t.Error("expected error for deleted checkpoint, got nil")
	}
}

func TestWorkflowCheckpointRepository_Delete_NotFound(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent checkpoint, got nil")
	}
}

func TestWorkflowCheckpointRepository_DeleteByExecutionID(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Create checkpoints
	cp1, _ := workflow.NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)
	_ = repo.Create(ctx, cp1)

	cp2, _ := workflow.NewWorkflowCheckpoint("cp-2", "exec-1", "skill-1", "Skill", "input", 3)
	_ = repo.Create(ctx, cp2)

	cp3, _ := workflow.NewWorkflowCheckpoint("cp-3", "exec-2", "skill-1", "Skill", "input", 3)
	_ = repo.Create(ctx, cp3)

	count, err := repo.DeleteByExecutionID(ctx, "exec-1")
	if err != nil {
		t.Fatalf("failed to delete by execution ID: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 deleted, got %d", count)
	}

	// Verify only cp3 remains
	all, _ := repo.List(ctx, nil)
	if len(all) != 1 {
		t.Errorf("expected 1 remaining, got %d", len(all))
	}
}

func TestWorkflowCheckpointRepository_MarkAbandoned(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Create checkpoints
	cp1, _ := workflow.NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)
	cp1.SetMachineID("machine-1")
	_ = repo.Create(ctx, cp1)

	cp2, _ := workflow.NewWorkflowCheckpoint("cp-2", "exec-2", "skill-1", "Skill", "input", 3)
	cp2.SetMachineID("machine-1")
	_ = repo.Create(ctx, cp2)

	cp3, _ := workflow.NewWorkflowCheckpoint("cp-3", "exec-3", "skill-1", "Skill", "input", 3)
	cp3.SetMachineID("machine-2")
	_ = repo.Create(ctx, cp3)

	count, err := repo.MarkAbandoned(ctx, "machine-1")
	if err != nil {
		t.Fatalf("failed to mark abandoned: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 marked abandoned, got %d", count)
	}

	// Verify status changes
	got1, _ := repo.Get(ctx, "cp-1")
	if got1.Status() != workflow.CheckpointStatusAbandoned {
		t.Errorf("expected abandoned status, got %s", got1.Status())
	}

	got3, _ := repo.Get(ctx, "cp-3")
	if got3.Status() != workflow.CheckpointStatusInProgress {
		t.Errorf("expected in_progress status for machine-2 checkpoint, got %s", got3.Status())
	}
}

func TestWorkflowCheckpointRepository_Cleanup(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Insert checkpoints directly with old timestamps
	oldTime := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	newTime := time.Now().Format(time.RFC3339)

	// Old completed checkpoint (should be cleaned)
	_, _ = db.Exec(`
		INSERT INTO workflow_checkpoints
		(id, execution_id, skill_id, skill_name, input, input_hash, total_batches, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "cp-old-completed", "exec-1", "skill-1", "Skill", "input", "hash", 3, "completed", oldTime, oldTime)

	// Old in-progress checkpoint (should NOT be cleaned)
	_, _ = db.Exec(`
		INSERT INTO workflow_checkpoints
		(id, execution_id, skill_id, skill_name, input, input_hash, total_batches, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "cp-old-inprogress", "exec-2", "skill-1", "Skill", "input", "hash", 3, "in_progress", oldTime, oldTime)

	// New completed checkpoint (should NOT be cleaned)
	_, _ = db.Exec(`
		INSERT INTO workflow_checkpoints
		(id, execution_id, skill_id, skill_name, input, input_hash, total_batches, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "cp-new-completed", "exec-3", "skill-1", "Skill", "input", "hash", 3, "completed", newTime, newTime)

	count, err := repo.Cleanup(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to cleanup: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 cleaned up, got %d", count)
	}

	// Verify correct checkpoint was removed
	all, _ := repo.List(ctx, nil)
	if len(all) != 2 {
		t.Errorf("expected 2 remaining, got %d", len(all))
	}
}

func TestWorkflowCheckpointRepository_EmptyDatabase(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// List on empty database
	checkpoints, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list empty database: %v", err)
	}
	if len(checkpoints) != 0 {
		t.Errorf("expected 0 checkpoints, got %d", len(checkpoints))
	}

	// GetLatestInProgress on empty database
	got, err := repo.GetLatestInProgress(ctx, "skill-1", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for empty database")
	}

	// DeleteByExecutionID on empty database
	count, err := repo.DeleteByExecutionID(ctx, "exec-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 deleted, got %d", count)
	}

	// MarkAbandoned on empty database
	count, err = repo.MarkAbandoned(ctx, "machine-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 marked, got %d", count)
	}

	// Cleanup on empty database
	count, err = repo.Cleanup(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cleaned, got %d", count)
	}
}

func TestWorkflowCheckpointRepository_RoundTrip(t *testing.T) {
	db := setupWorkflowCheckpointTestDB(t)
	defer db.Close()

	repo := NewWorkflowCheckpointRepository(db)
	ctx := context.Background()

	// Create a fully populated checkpoint
	cp, _ := workflow.NewWorkflowCheckpoint("cp-full", "exec-full", "skill-test", "Test Skill", "complex input with unicode: 日本語", 5)
	cp.SetMachineID("machine-test")

	// Add phase results
	cp.AddPhaseResult("phase-1", &workflow.PhaseResultData{
		PhaseID:      "phase-1",
		PhaseName:    "Phase One",
		Status:       "completed",
		Output:       "Long output with special chars: <>&\"'",
		InputTokens:  500,
		OutputTokens: 250,
		ModelUsed:    "claude-3",
		CacheHit:     true,
	})
	cp.AddPhaseResult("phase-2", &workflow.PhaseResultData{
		PhaseID:      "phase-2",
		PhaseName:    "Phase Two",
		Status:       "completed",
		Output:       "Another output",
		InputTokens:  300,
		OutputTokens: 150,
	})

	// Add phase outputs
	cp.AddPhaseOutput("_input", "complex input with unicode: 日本語")
	cp.AddPhaseOutput("phase-1", "Long output with special chars: <>&\"'")
	cp.AddPhaseOutput("phase-2", "Another output")

	// Update tokens and batch
	cp.UpdateTokens(800, 400)
	_ = cp.UpdateBatch(1)

	// Save
	if err := repo.Create(ctx, cp); err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Retrieve
	got, err := repo.Get(ctx, "cp-full")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	// Verify all fields roundtrip correctly
	if got.Input() != "complex input with unicode: 日本語" {
		t.Errorf("input mismatch: got %q", got.Input())
	}
	if got.MachineID() != "machine-test" {
		t.Errorf("machineID mismatch: got %q", got.MachineID())
	}
	if got.CompletedBatch() != 1 {
		t.Errorf("completedBatch mismatch: got %d", got.CompletedBatch())
	}
	if got.InputTokens() != 800 {
		t.Errorf("inputTokens mismatch: got %d", got.InputTokens())
	}

	// Check phase results
	results := got.PhaseResults()
	if len(results) != 2 {
		t.Fatalf("expected 2 phase results, got %d", len(results))
	}
	if results["phase-1"].Output != "Long output with special chars: <>&\"'" {
		t.Errorf("phase-1 output mismatch: got %q", results["phase-1"].Output)
	}
	if !results["phase-1"].CacheHit {
		t.Error("expected phase-1 cache hit to be true")
	}

	// Check phase outputs
	outputs := got.PhaseOutputs()
	if len(outputs) != 3 {
		t.Errorf("expected 3 phase outputs, got %d", len(outputs))
	}
	if outputs["_input"] != "complex input with unicode: 日本語" {
		t.Errorf("_input output mismatch: got %q", outputs["_input"])
	}
}
