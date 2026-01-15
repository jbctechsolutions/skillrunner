package workflow

import (
	"testing"
	"time"
)

func TestCheckpointStatus_Constants(t *testing.T) {
	tests := []struct {
		status   CheckpointStatus
		expected string
	}{
		{CheckpointStatusInProgress, "in_progress"},
		{CheckpointStatusCompleted, "completed"},
		{CheckpointStatusFailed, "failed"},
		{CheckpointStatusAbandoned, "abandoned"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}

func TestHashInput(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string // partial match, just check non-empty and consistent
	}{
		{"empty input", "", ""},
		{"simple input", "hello", ""},
		{"unicode input", "hello 世界", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashInput(tt.input)
			hash2 := HashInput(tt.input)

			// Should be deterministic
			if hash1 != hash2 {
				t.Errorf("HashInput not deterministic: %s != %s", hash1, hash2)
			}

			// Should be 32 hex characters (128 bits)
			if len(hash1) != 32 {
				t.Errorf("expected hash length 32, got %d", len(hash1))
			}
		})
	}

	// Different inputs should produce different hashes
	hash1 := HashInput("input1")
	hash2 := HashInput("input2")
	if hash1 == hash2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestNewWorkflowCheckpoint(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		executionID  string
		skillID      string
		skillName    string
		input        string
		totalBatches int
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid checkpoint",
			id:           "cp-123",
			executionID:  "exec-456",
			skillID:      "skill-789",
			skillName:    "Test Skill",
			input:        "test input",
			totalBatches: 3,
			wantErr:      false,
		},
		{
			name:         "empty id",
			id:           "",
			executionID:  "exec-456",
			skillID:      "skill-789",
			skillName:    "Test Skill",
			input:        "test input",
			totalBatches: 3,
			wantErr:      true,
			errContains:  "checkpoint ID is required",
		},
		{
			name:         "whitespace id",
			id:           "   ",
			executionID:  "exec-456",
			skillID:      "skill-789",
			skillName:    "Test Skill",
			input:        "test input",
			totalBatches: 3,
			wantErr:      true,
			errContains:  "checkpoint ID is required",
		},
		{
			name:         "empty execution id",
			id:           "cp-123",
			executionID:  "",
			skillID:      "skill-789",
			skillName:    "Test Skill",
			input:        "test input",
			totalBatches: 3,
			wantErr:      true,
			errContains:  "execution ID is required",
		},
		{
			name:         "empty skill id",
			id:           "cp-123",
			executionID:  "exec-456",
			skillID:      "",
			skillName:    "Test Skill",
			input:        "test input",
			totalBatches: 3,
			wantErr:      true,
			errContains:  "skill ID is required",
		},
		{
			name:         "empty skill name",
			id:           "cp-123",
			executionID:  "exec-456",
			skillID:      "skill-789",
			skillName:    "",
			input:        "test input",
			totalBatches: 3,
			wantErr:      true,
			errContains:  "skill name is required",
		},
		{
			name:         "zero total batches",
			id:           "cp-123",
			executionID:  "exec-456",
			skillID:      "skill-789",
			skillName:    "Test Skill",
			input:        "test input",
			totalBatches: 0,
			wantErr:      true,
			errContains:  "total batches must be at least 1",
		},
		{
			name:         "negative total batches",
			id:           "cp-123",
			executionID:  "exec-456",
			skillID:      "skill-789",
			skillName:    "Test Skill",
			input:        "test input",
			totalBatches: -1,
			wantErr:      true,
			errContains:  "total batches must be at least 1",
		},
		{
			name:         "empty input allowed",
			id:           "cp-123",
			executionID:  "exec-456",
			skillID:      "skill-789",
			skillName:    "Test Skill",
			input:        "",
			totalBatches: 1,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			cp, err := NewWorkflowCheckpoint(tt.id, tt.executionID, tt.skillID, tt.skillName, tt.input, tt.totalBatches)
			after := time.Now()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify all fields
			if cp.ID() != tt.id {
				t.Errorf("expected ID %s, got %s", tt.id, cp.ID())
			}
			if cp.ExecutionID() != tt.executionID {
				t.Errorf("expected ExecutionID %s, got %s", tt.executionID, cp.ExecutionID())
			}
			if cp.SkillID() != tt.skillID {
				t.Errorf("expected SkillID %s, got %s", tt.skillID, cp.SkillID())
			}
			if cp.SkillName() != tt.skillName {
				t.Errorf("expected SkillName %s, got %s", tt.skillName, cp.SkillName())
			}
			if cp.Input() != tt.input {
				t.Errorf("expected Input %s, got %s", tt.input, cp.Input())
			}
			if cp.InputHash() != HashInput(tt.input) {
				t.Errorf("expected InputHash %s, got %s", HashInput(tt.input), cp.InputHash())
			}
			if cp.TotalBatches() != tt.totalBatches {
				t.Errorf("expected TotalBatches %d, got %d", tt.totalBatches, cp.TotalBatches())
			}
			if cp.CompletedBatch() != -1 {
				t.Errorf("expected CompletedBatch -1, got %d", cp.CompletedBatch())
			}
			if cp.Status() != CheckpointStatusInProgress {
				t.Errorf("expected Status %s, got %s", CheckpointStatusInProgress, cp.Status())
			}
			if cp.InputTokens() != 0 {
				t.Errorf("expected InputTokens 0, got %d", cp.InputTokens())
			}
			if cp.OutputTokens() != 0 {
				t.Errorf("expected OutputTokens 0, got %d", cp.OutputTokens())
			}
			if cp.TotalTokens() != 0 {
				t.Errorf("expected TotalTokens 0, got %d", cp.TotalTokens())
			}
			if cp.MachineID() != "" {
				t.Errorf("expected empty MachineID, got %s", cp.MachineID())
			}
			if cp.CreatedAt().Before(before) || cp.CreatedAt().After(after) {
				t.Error("CreatedAt should be between before and after")
			}
			if cp.UpdatedAt().Before(before) || cp.UpdatedAt().After(after) {
				t.Error("UpdatedAt should be between before and after")
			}
			if len(cp.PhaseResults()) != 0 {
				t.Errorf("expected empty PhaseResults, got %d", len(cp.PhaseResults()))
			}
			if len(cp.PhaseOutputs()) != 0 {
				t.Errorf("expected empty PhaseOutputs, got %d", len(cp.PhaseOutputs()))
			}
		})
	}
}

func TestWorkflowCheckpoint_SetMachineID(t *testing.T) {
	cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)
	originalUpdated := cp.UpdatedAt()

	time.Sleep(time.Millisecond)
	cp.SetMachineID("  machine-123  ")

	if cp.MachineID() != "machine-123" {
		t.Errorf("expected MachineID 'machine-123', got %q", cp.MachineID())
	}
	if !cp.UpdatedAt().After(originalUpdated) {
		t.Error("UpdatedAt should be updated")
	}
}

func TestWorkflowCheckpoint_UpdateBatch(t *testing.T) {
	cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 5)

	tests := []struct {
		name        string
		batchIndex  int
		wantErr     bool
		errContains string
	}{
		{"valid batch 0", 0, false, ""},
		{"valid batch 2", 2, false, ""},
		{"valid batch 4", 4, false, ""},
		{"negative batch", -1, true, "cannot be negative"},
		{"batch equals total", 5, true, "exceeds total batches"},
		{"batch exceeds total", 10, true, "exceeds total batches"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cp.UpdateBatch(tt.batchIndex)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cp.CompletedBatch() != tt.batchIndex {
				t.Errorf("expected CompletedBatch %d, got %d", tt.batchIndex, cp.CompletedBatch())
			}
		})
	}
}

func TestWorkflowCheckpoint_PhaseResults(t *testing.T) {
	cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)

	// Add a phase result
	result := &PhaseResultData{
		PhaseID:      "phase-1",
		PhaseName:    "Phase One",
		Status:       "completed",
		Output:       "output text",
		InputTokens:  100,
		OutputTokens: 50,
	}
	cp.AddPhaseResult("phase-1", result)

	// Get and verify
	results := cp.PhaseResults()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results["phase-1"].PhaseID != "phase-1" {
		t.Errorf("expected PhaseID 'phase-1', got %q", results["phase-1"].PhaseID)
	}

	// Verify immutability (modifying returned map doesn't affect internal)
	results["phase-1"].Output = "modified"
	internalResults := cp.PhaseResults()
	if internalResults["phase-1"].Output == "modified" {
		t.Error("modifying returned map should not affect internal state")
	}

	// Test AddPhaseResult with empty key
	cp.AddPhaseResult("", result)
	if len(cp.PhaseResults()) != 1 {
		t.Error("adding with empty key should be ignored")
	}

	// Test AddPhaseResult with nil result
	cp.AddPhaseResult("phase-2", nil)
	if len(cp.PhaseResults()) != 1 {
		t.Error("adding nil result should be ignored")
	}
}

func TestWorkflowCheckpoint_SetPhaseResults(t *testing.T) {
	cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)

	results := map[string]*PhaseResultData{
		"phase-1": {PhaseID: "phase-1", Status: "completed"},
		"phase-2": {PhaseID: "phase-2", Status: "running"},
	}
	cp.SetPhaseResults(results)

	got := cp.PhaseResults()
	if len(got) != 2 {
		t.Errorf("expected 2 results, got %d", len(got))
	}

	// Test with nil values in map
	resultsWithNil := map[string]*PhaseResultData{
		"phase-1": {PhaseID: "phase-1"},
		"phase-2": nil,
	}
	cp.SetPhaseResults(resultsWithNil)

	got = cp.PhaseResults()
	if len(got) != 1 {
		t.Errorf("expected 1 result (nil filtered), got %d", len(got))
	}
}

func TestWorkflowCheckpoint_PhaseOutputs(t *testing.T) {
	cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)

	// Add outputs
	cp.AddPhaseOutput("phase-1", "output1")
	cp.AddPhaseOutput("phase-2", "output2")

	outputs := cp.PhaseOutputs()
	if len(outputs) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(outputs))
	}
	if outputs["phase-1"] != "output1" {
		t.Errorf("expected 'output1', got %q", outputs["phase-1"])
	}

	// Verify immutability
	outputs["phase-1"] = "modified"
	internalOutputs := cp.PhaseOutputs()
	if internalOutputs["phase-1"] == "modified" {
		t.Error("modifying returned map should not affect internal state")
	}

	// Test with empty key
	cp.AddPhaseOutput("", "value")
	if len(cp.PhaseOutputs()) != 2 {
		t.Error("adding with empty key should be ignored")
	}
}

func TestWorkflowCheckpoint_SetPhaseOutputs(t *testing.T) {
	cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)

	outputs := map[string]string{
		"phase-1": "output1",
		"phase-2": "output2",
		"_input":  "original input",
	}
	cp.SetPhaseOutputs(outputs)

	got := cp.PhaseOutputs()
	if len(got) != 3 {
		t.Errorf("expected 3 outputs, got %d", len(got))
	}
	if got["_input"] != "original input" {
		t.Errorf("expected '_input' to be 'original input', got %q", got["_input"])
	}
}

func TestWorkflowCheckpoint_Tokens(t *testing.T) {
	cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)

	// UpdateTokens
	cp.UpdateTokens(100, 50)
	if cp.InputTokens() != 100 {
		t.Errorf("expected InputTokens 100, got %d", cp.InputTokens())
	}
	if cp.OutputTokens() != 50 {
		t.Errorf("expected OutputTokens 50, got %d", cp.OutputTokens())
	}
	if cp.TotalTokens() != 150 {
		t.Errorf("expected TotalTokens 150, got %d", cp.TotalTokens())
	}

	// AddTokens
	cp.AddTokens(50, 25)
	if cp.InputTokens() != 150 {
		t.Errorf("expected InputTokens 150, got %d", cp.InputTokens())
	}
	if cp.OutputTokens() != 75 {
		t.Errorf("expected OutputTokens 75, got %d", cp.OutputTokens())
	}
	if cp.TotalTokens() != 225 {
		t.Errorf("expected TotalTokens 225, got %d", cp.TotalTokens())
	}
}

func TestWorkflowCheckpoint_StatusTransitions(t *testing.T) {
	cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)

	// Initial state
	if cp.Status() != CheckpointStatusInProgress {
		t.Errorf("expected initial status %s, got %s", CheckpointStatusInProgress, cp.Status())
	}
	if !cp.IsResumable() {
		t.Error("in_progress checkpoint should be resumable")
	}

	// Mark completed
	cp.MarkCompleted()
	if cp.Status() != CheckpointStatusCompleted {
		t.Errorf("expected status %s, got %s", CheckpointStatusCompleted, cp.Status())
	}
	if cp.IsResumable() {
		t.Error("completed checkpoint should not be resumable")
	}

	// Create new checkpoint for failed test
	cp2, _ := NewWorkflowCheckpoint("cp-2", "exec-2", "skill-1", "Skill", "input", 3)
	cp2.MarkFailed()
	if cp2.Status() != CheckpointStatusFailed {
		t.Errorf("expected status %s, got %s", CheckpointStatusFailed, cp2.Status())
	}
	if cp2.IsResumable() {
		t.Error("failed checkpoint should not be resumable")
	}

	// Create new checkpoint for abandoned test
	cp3, _ := NewWorkflowCheckpoint("cp-3", "exec-3", "skill-1", "Skill", "input", 3)
	cp3.MarkAbandoned()
	if cp3.Status() != CheckpointStatusAbandoned {
		t.Errorf("expected status %s, got %s", CheckpointStatusAbandoned, cp3.Status())
	}
	if cp3.IsResumable() {
		t.Error("abandoned checkpoint should not be resumable")
	}
}

func TestWorkflowCheckpoint_Progress(t *testing.T) {
	tests := []struct {
		completedBatch int
		totalBatches   int
		expected       string
	}{
		{-1, 5, "0/5"},
		{0, 5, "1/5"},
		{2, 5, "3/5"},
		{4, 5, "5/5"},
		{0, 1, "1/1"},
		{9, 10, "10/10"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", tt.totalBatches)
			if tt.completedBatch >= 0 {
				_ = cp.UpdateBatch(tt.completedBatch)
			}

			progress := cp.Progress()
			if progress != tt.expected {
				t.Errorf("expected progress %q, got %q", tt.expected, progress)
			}
		})
	}
}

func TestWorkflowCheckpoint_Validate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*WorkflowCheckpoint)
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid checkpoint",
			modify:  func(cp *WorkflowCheckpoint) {},
			wantErr: false,
		},
		{
			name:        "empty id",
			modify:      func(cp *WorkflowCheckpoint) { cp.id = "" },
			wantErr:     true,
			errContains: "checkpoint ID is required",
		},
		{
			name:        "whitespace id",
			modify:      func(cp *WorkflowCheckpoint) { cp.id = "   " },
			wantErr:     true,
			errContains: "checkpoint ID is required",
		},
		{
			name:        "empty execution id",
			modify:      func(cp *WorkflowCheckpoint) { cp.executionID = "" },
			wantErr:     true,
			errContains: "execution ID is required",
		},
		{
			name:        "empty skill id",
			modify:      func(cp *WorkflowCheckpoint) { cp.skillID = "" },
			wantErr:     true,
			errContains: "skill ID is required",
		},
		{
			name:        "empty skill name",
			modify:      func(cp *WorkflowCheckpoint) { cp.skillName = "" },
			wantErr:     true,
			errContains: "skill name is required",
		},
		{
			name:        "zero total batches",
			modify:      func(cp *WorkflowCheckpoint) { cp.totalBatches = 0 },
			wantErr:     true,
			errContains: "total batches must be at least 1",
		},
		{
			name: "completed batch exceeds total",
			modify: func(cp *WorkflowCheckpoint) {
				cp.completedBatch = 5
				cp.totalBatches = 3
			},
			wantErr:     true,
			errContains: "completed batch exceeds total batches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp, _ := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Skill", "input", 3)
			tt.modify(cp)

			err := cp.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestReconstructCheckpoint(t *testing.T) {
	now := time.Now()
	phaseResults := map[string]*PhaseResultData{
		"phase-1": {PhaseID: "phase-1", Status: "completed"},
	}
	phaseOutputs := map[string]string{
		"_input":  "original",
		"phase-1": "output1",
	}

	cp := ReconstructCheckpoint(
		"cp-1",
		"exec-1",
		"skill-1",
		"Skill Name",
		"input text",
		"inputhash123",
		2,
		5,
		phaseResults,
		phaseOutputs,
		CheckpointStatusInProgress,
		100,
		50,
		"machine-1",
		now,
		now,
	)

	if cp.ID() != "cp-1" {
		t.Errorf("expected ID 'cp-1', got %q", cp.ID())
	}
	if cp.ExecutionID() != "exec-1" {
		t.Errorf("expected ExecutionID 'exec-1', got %q", cp.ExecutionID())
	}
	if cp.SkillID() != "skill-1" {
		t.Errorf("expected SkillID 'skill-1', got %q", cp.SkillID())
	}
	if cp.SkillName() != "Skill Name" {
		t.Errorf("expected SkillName 'Skill Name', got %q", cp.SkillName())
	}
	if cp.Input() != "input text" {
		t.Errorf("expected Input 'input text', got %q", cp.Input())
	}
	if cp.InputHash() != "inputhash123" {
		t.Errorf("expected InputHash 'inputhash123', got %q", cp.InputHash())
	}
	if cp.CompletedBatch() != 2 {
		t.Errorf("expected CompletedBatch 2, got %d", cp.CompletedBatch())
	}
	if cp.TotalBatches() != 5 {
		t.Errorf("expected TotalBatches 5, got %d", cp.TotalBatches())
	}
	if cp.Status() != CheckpointStatusInProgress {
		t.Errorf("expected Status %s, got %s", CheckpointStatusInProgress, cp.Status())
	}
	if cp.InputTokens() != 100 {
		t.Errorf("expected InputTokens 100, got %d", cp.InputTokens())
	}
	if cp.OutputTokens() != 50 {
		t.Errorf("expected OutputTokens 50, got %d", cp.OutputTokens())
	}
	if cp.MachineID() != "machine-1" {
		t.Errorf("expected MachineID 'machine-1', got %q", cp.MachineID())
	}
	if !cp.CreatedAt().Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, cp.CreatedAt())
	}
	if !cp.UpdatedAt().Equal(now) {
		t.Errorf("expected UpdatedAt %v, got %v", now, cp.UpdatedAt())
	}
	if len(cp.PhaseResults()) != 1 {
		t.Errorf("expected 1 phase result, got %d", len(cp.PhaseResults()))
	}
	if len(cp.PhaseOutputs()) != 2 {
		t.Errorf("expected 2 phase outputs, got %d", len(cp.PhaseOutputs()))
	}
}

func TestReconstructCheckpoint_NilMaps(t *testing.T) {
	now := time.Now()

	cp := ReconstructCheckpoint(
		"cp-1",
		"exec-1",
		"skill-1",
		"Skill",
		"input",
		"hash",
		0,
		1,
		nil, // nil phaseResults
		nil, // nil phaseOutputs
		CheckpointStatusCompleted,
		0,
		0,
		"",
		now,
		now,
	)

	// Should have empty maps, not nil
	if cp.PhaseResults() == nil {
		t.Error("PhaseResults should not be nil")
	}
	if cp.PhaseOutputs() == nil {
		t.Error("PhaseOutputs should not be nil")
	}
}

func TestWorkflowCheckpoint_Lifecycle(t *testing.T) {
	// Test complete lifecycle: create -> update batches -> complete
	cp, err := NewWorkflowCheckpoint("cp-1", "exec-1", "skill-1", "Test Skill", "test input", 3)
	if err != nil {
		t.Fatalf("failed to create checkpoint: %v", err)
	}

	// Set machine ID
	cp.SetMachineID("machine-1")

	// Add _input to phase outputs
	cp.AddPhaseOutput("_input", "test input")

	// Batch 0 completes
	cp.AddPhaseResult("phase-1", &PhaseResultData{
		PhaseID:      "phase-1",
		PhaseName:    "Phase One",
		Status:       "completed",
		Output:       "output1",
		InputTokens:  100,
		OutputTokens: 50,
	})
	cp.AddPhaseOutput("phase-1", "output1")
	cp.AddTokens(100, 50)
	if err := cp.UpdateBatch(0); err != nil {
		t.Fatalf("failed to update batch 0: %v", err)
	}

	if cp.Progress() != "1/3" {
		t.Errorf("expected progress '1/3', got %q", cp.Progress())
	}

	// Batch 1 completes
	cp.AddPhaseResult("phase-2", &PhaseResultData{
		PhaseID:      "phase-2",
		PhaseName:    "Phase Two",
		Status:       "completed",
		Output:       "output2",
		InputTokens:  150,
		OutputTokens: 75,
	})
	cp.AddPhaseOutput("phase-2", "output2")
	cp.AddTokens(150, 75)
	if err := cp.UpdateBatch(1); err != nil {
		t.Fatalf("failed to update batch 1: %v", err)
	}

	if cp.Progress() != "2/3" {
		t.Errorf("expected progress '2/3', got %q", cp.Progress())
	}

	// Batch 2 completes
	cp.AddPhaseResult("phase-3", &PhaseResultData{
		PhaseID:      "phase-3",
		PhaseName:    "Phase Three",
		Status:       "completed",
		Output:       "output3",
		InputTokens:  200,
		OutputTokens: 100,
	})
	cp.AddPhaseOutput("phase-3", "output3")
	cp.AddTokens(200, 100)
	if err := cp.UpdateBatch(2); err != nil {
		t.Fatalf("failed to update batch 2: %v", err)
	}

	// Mark completed
	cp.MarkCompleted()

	// Verify final state
	if cp.Status() != CheckpointStatusCompleted {
		t.Errorf("expected status %s, got %s", CheckpointStatusCompleted, cp.Status())
	}
	if cp.Progress() != "3/3" {
		t.Errorf("expected progress '3/3', got %q", cp.Progress())
	}
	if cp.TotalTokens() != 675 {
		t.Errorf("expected TotalTokens 675, got %d", cp.TotalTokens())
	}
	if len(cp.PhaseResults()) != 3 {
		t.Errorf("expected 3 phase results, got %d", len(cp.PhaseResults()))
	}
	if len(cp.PhaseOutputs()) != 4 { // _input + 3 phases
		t.Errorf("expected 4 phase outputs, got %d", len(cp.PhaseOutputs()))
	}
	if cp.IsResumable() {
		t.Error("completed checkpoint should not be resumable")
	}

	// Validation should pass
	if err := cp.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
