package workflow

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPhasePlan_TotalEstimatedTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		output   int
		expected int
	}{
		{"both zero", 0, 0, 0},
		{"only input", 100, 0, 100},
		{"only output", 0, 50, 50},
		{"both set", 100, 50, 150},
		{"large values", 10000, 5000, 15000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PhasePlan{
				EstimatedInputTokens:  tt.input,
				EstimatedOutputTokens: tt.output,
			}
			if got := p.TotalEstimatedTokens(); got != tt.expected {
				t.Errorf("TotalEstimatedTokens() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestNewExecutionPlan(t *testing.T) {
	before := time.Now()
	plan := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")
	after := time.Now()

	if plan == nil {
		t.Fatal("expected non-nil ExecutionPlan")
	}
	if plan.SkillID != "skill-1" {
		t.Errorf("expected SkillID 'skill-1', got %s", plan.SkillID)
	}
	if plan.SkillName != "Test Skill" {
		t.Errorf("expected SkillName 'Test Skill', got %s", plan.SkillName)
	}
	if plan.SkillVersion != "1.0.0" {
		t.Errorf("expected SkillVersion '1.0.0', got %s", plan.SkillVersion)
	}
	if plan.Input != "test input" {
		t.Errorf("expected Input 'test input', got %s", plan.Input)
	}
	if plan.Phases == nil {
		t.Error("expected non-nil Phases slice")
	}
	if len(plan.Phases) != 0 {
		t.Errorf("expected empty Phases, got %d", len(plan.Phases))
	}
	if plan.Batches == nil {
		t.Error("expected non-nil Batches slice")
	}
	if len(plan.Batches) != 0 {
		t.Errorf("expected empty Batches, got %d", len(plan.Batches))
	}
	if plan.CreatedAt.Before(before) || plan.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after")
	}
	if plan.TotalEstimatedInputTokens != 0 {
		t.Errorf("expected TotalEstimatedInputTokens 0, got %d", plan.TotalEstimatedInputTokens)
	}
	if plan.TotalEstimatedOutputTokens != 0 {
		t.Errorf("expected TotalEstimatedOutputTokens 0, got %d", plan.TotalEstimatedOutputTokens)
	}
	if plan.TotalEstimatedCost != 0 {
		t.Errorf("expected TotalEstimatedCost 0, got %f", plan.TotalEstimatedCost)
	}
}

func TestExecutionPlan_AddPhasePlan(t *testing.T) {
	plan := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")

	phase1 := PhasePlan{
		PhaseID:               "phase-1",
		PhaseName:             "Phase 1",
		RoutingProfile:        "balanced",
		ResolvedModel:         "claude-3-sonnet",
		ResolvedProvider:      "anthropic",
		EstimatedInputTokens:  100,
		EstimatedOutputTokens: 50,
		EstimatedCost:         0.01,
		BatchIndex:            0,
	}

	plan.AddPhasePlan(phase1)

	if len(plan.Phases) != 1 {
		t.Errorf("expected 1 phase, got %d", len(plan.Phases))
	}
	if plan.TotalEstimatedInputTokens != 100 {
		t.Errorf("expected TotalEstimatedInputTokens 100, got %d", plan.TotalEstimatedInputTokens)
	}
	if plan.TotalEstimatedOutputTokens != 50 {
		t.Errorf("expected TotalEstimatedOutputTokens 50, got %d", plan.TotalEstimatedOutputTokens)
	}
	if plan.TotalEstimatedCost != 0.01 {
		t.Errorf("expected TotalEstimatedCost 0.01, got %f", plan.TotalEstimatedCost)
	}

	// Add another phase
	phase2 := PhasePlan{
		PhaseID:               "phase-2",
		PhaseName:             "Phase 2",
		RoutingProfile:        "premium",
		ResolvedModel:         "claude-3-opus",
		ResolvedProvider:      "anthropic",
		DependsOn:             []string{"phase-1"},
		EstimatedInputTokens:  200,
		EstimatedOutputTokens: 100,
		EstimatedCost:         0.05,
		BatchIndex:            1,
	}

	plan.AddPhasePlan(phase2)

	if len(plan.Phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(plan.Phases))
	}
	if plan.TotalEstimatedInputTokens != 300 {
		t.Errorf("expected TotalEstimatedInputTokens 300, got %d", plan.TotalEstimatedInputTokens)
	}
	if plan.TotalEstimatedOutputTokens != 150 {
		t.Errorf("expected TotalEstimatedOutputTokens 150, got %d", plan.TotalEstimatedOutputTokens)
	}
	// Use tolerance for floating point comparison
	expectedCost := 0.06
	if diff := plan.TotalEstimatedCost - expectedCost; diff < -0.0001 || diff > 0.0001 {
		t.Errorf("expected TotalEstimatedCost ~0.06, got %f", plan.TotalEstimatedCost)
	}
}

func TestExecutionPlan_SetBatches(t *testing.T) {
	plan := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")

	batches := [][]string{
		{"phase-1", "phase-2"},
		{"phase-3"},
		{"phase-4", "phase-5"},
	}

	plan.SetBatches(batches)

	if len(plan.Batches) != 3 {
		t.Errorf("expected 3 batches, got %d", len(plan.Batches))
	}
	if len(plan.Batches[0]) != 2 {
		t.Errorf("expected 2 phases in batch 0, got %d", len(plan.Batches[0]))
	}
	if len(plan.Batches[1]) != 1 {
		t.Errorf("expected 1 phase in batch 1, got %d", len(plan.Batches[1]))
	}
	if len(plan.Batches[2]) != 2 {
		t.Errorf("expected 2 phases in batch 2, got %d", len(plan.Batches[2]))
	}
}

func TestExecutionPlan_TotalEstimatedTokens(t *testing.T) {
	plan := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")

	plan.AddPhasePlan(PhasePlan{
		PhaseID:               "phase-1",
		EstimatedInputTokens:  100,
		EstimatedOutputTokens: 50,
	})
	plan.AddPhasePlan(PhasePlan{
		PhaseID:               "phase-2",
		EstimatedInputTokens:  200,
		EstimatedOutputTokens: 100,
	})

	if got := plan.TotalEstimatedTokens(); got != 450 {
		t.Errorf("TotalEstimatedTokens() = %d, expected 450", got)
	}
}

func TestExecutionPlan_PhaseCount(t *testing.T) {
	plan := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")

	if plan.PhaseCount() != 0 {
		t.Errorf("expected PhaseCount 0, got %d", plan.PhaseCount())
	}

	plan.AddPhasePlan(PhasePlan{PhaseID: "phase-1"})
	plan.AddPhasePlan(PhasePlan{PhaseID: "phase-2"})
	plan.AddPhasePlan(PhasePlan{PhaseID: "phase-3"})

	if plan.PhaseCount() != 3 {
		t.Errorf("expected PhaseCount 3, got %d", plan.PhaseCount())
	}
}

func TestExecutionPlan_BatchCount(t *testing.T) {
	plan := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")

	if plan.BatchCount() != 0 {
		t.Errorf("expected BatchCount 0, got %d", plan.BatchCount())
	}

	plan.SetBatches([][]string{
		{"phase-1"},
		{"phase-2", "phase-3"},
	})

	if plan.BatchCount() != 2 {
		t.Errorf("expected BatchCount 2, got %d", plan.BatchCount())
	}
}

func TestExecutionPlan_GetPhase(t *testing.T) {
	plan := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")

	plan.AddPhasePlan(PhasePlan{PhaseID: "phase-1", PhaseName: "Phase 1"})
	plan.AddPhasePlan(PhasePlan{PhaseID: "phase-2", PhaseName: "Phase 2"})
	plan.AddPhasePlan(PhasePlan{PhaseID: "phase-3", PhaseName: "Phase 3"})

	// Find existing phase
	phase := plan.GetPhase("phase-2")
	if phase == nil {
		t.Fatal("expected to find phase-2")
	}
	if phase.PhaseName != "Phase 2" {
		t.Errorf("expected PhaseName 'Phase 2', got %s", phase.PhaseName)
	}

	// Find non-existing phase
	phase = plan.GetPhase("phase-99")
	if phase != nil {
		t.Error("expected nil for non-existing phase")
	}
}

func TestExecutionPlan_ToJSON(t *testing.T) {
	plan := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")
	plan.AddPhasePlan(PhasePlan{
		PhaseID:               "phase-1",
		PhaseName:             "Phase 1",
		RoutingProfile:        "balanced",
		ResolvedModel:         "claude-3-sonnet",
		ResolvedProvider:      "anthropic",
		EstimatedInputTokens:  100,
		EstimatedOutputTokens: 50,
		EstimatedCost:         0.01,
		BatchIndex:            0,
	})
	plan.SetBatches([][]string{{"phase-1"}})

	jsonData, err := plan.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify key fields
	if parsed["skill_id"] != "skill-1" {
		t.Errorf("expected skill_id 'skill-1', got %v", parsed["skill_id"])
	}
	if parsed["skill_name"] != "Test Skill" {
		t.Errorf("expected skill_name 'Test Skill', got %v", parsed["skill_name"])
	}
}

func TestExecutionPlanFromJSON(t *testing.T) {
	original := NewExecutionPlan("skill-1", "Test Skill", "1.0.0", "test input")
	original.AddPhasePlan(PhasePlan{
		PhaseID:               "phase-1",
		PhaseName:             "Phase 1",
		RoutingProfile:        "balanced",
		ResolvedModel:         "claude-3-sonnet",
		ResolvedProvider:      "anthropic",
		DependsOn:             []string{},
		EstimatedInputTokens:  100,
		EstimatedOutputTokens: 50,
		EstimatedCost:         0.01,
		BatchIndex:            0,
	})
	original.SetBatches([][]string{{"phase-1"}})

	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	parsed, err := ExecutionPlanFromJSON(jsonData)
	if err != nil {
		t.Fatalf("ExecutionPlanFromJSON() error: %v", err)
	}

	if parsed.SkillID != original.SkillID {
		t.Errorf("expected SkillID %s, got %s", original.SkillID, parsed.SkillID)
	}
	if parsed.SkillName != original.SkillName {
		t.Errorf("expected SkillName %s, got %s", original.SkillName, parsed.SkillName)
	}
	if parsed.Input != original.Input {
		t.Errorf("expected Input %s, got %s", original.Input, parsed.Input)
	}
	if len(parsed.Phases) != len(original.Phases) {
		t.Errorf("expected %d phases, got %d", len(original.Phases), len(parsed.Phases))
	}
	if len(parsed.Batches) != len(original.Batches) {
		t.Errorf("expected %d batches, got %d", len(original.Batches), len(parsed.Batches))
	}
}

func TestExecutionPlanFromJSON_Invalid(t *testing.T) {
	_, err := ExecutionPlanFromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestExecutionPlan_HasLocalModels(t *testing.T) {
	tests := []struct {
		name     string
		phases   []PhasePlan
		expected bool
	}{
		{
			name:     "empty plan",
			phases:   []PhasePlan{},
			expected: false,
		},
		{
			name: "only cloud models",
			phases: []PhasePlan{
				{PhaseID: "1", EstimatedCost: 0.01, ResolvedProvider: "anthropic"},
				{PhaseID: "2", EstimatedCost: 0.02, ResolvedProvider: "openai"},
			},
			expected: false,
		},
		{
			name: "only local models",
			phases: []PhasePlan{
				{PhaseID: "1", EstimatedCost: 0, ResolvedProvider: "ollama"},
				{PhaseID: "2", EstimatedCost: 0, ResolvedProvider: "ollama"},
			},
			expected: true,
		},
		{
			name: "mixed models",
			phases: []PhasePlan{
				{PhaseID: "1", EstimatedCost: 0, ResolvedProvider: "ollama"},
				{PhaseID: "2", EstimatedCost: 0.01, ResolvedProvider: "anthropic"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := NewExecutionPlan("skill-1", "Test", "1.0", "input")
			for _, phase := range tt.phases {
				plan.AddPhasePlan(phase)
			}
			if got := plan.HasLocalModels(); got != tt.expected {
				t.Errorf("HasLocalModels() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestExecutionPlan_HasCloudModels(t *testing.T) {
	tests := []struct {
		name     string
		phases   []PhasePlan
		expected bool
	}{
		{
			name:     "empty plan",
			phases:   []PhasePlan{},
			expected: false,
		},
		{
			name: "only local models",
			phases: []PhasePlan{
				{PhaseID: "1", EstimatedCost: 0, ResolvedProvider: "ollama"},
				{PhaseID: "2", EstimatedCost: 0, ResolvedProvider: "ollama"},
			},
			expected: false,
		},
		{
			name: "only cloud models",
			phases: []PhasePlan{
				{PhaseID: "1", EstimatedCost: 0.01, ResolvedProvider: "anthropic"},
				{PhaseID: "2", EstimatedCost: 0.02, ResolvedProvider: "openai"},
			},
			expected: true,
		},
		{
			name: "mixed models",
			phases: []PhasePlan{
				{PhaseID: "1", EstimatedCost: 0, ResolvedProvider: "ollama"},
				{PhaseID: "2", EstimatedCost: 0.01, ResolvedProvider: "anthropic"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := NewExecutionPlan("skill-1", "Test", "1.0", "input")
			for _, phase := range tt.phases {
				plan.AddPhasePlan(phase)
			}
			if got := plan.HasCloudModels(); got != tt.expected {
				t.Errorf("HasCloudModels() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestPhasePlan_JSONSerialization(t *testing.T) {
	original := PhasePlan{
		PhaseID:               "phase-1",
		PhaseName:             "Test Phase",
		RoutingProfile:        "premium",
		ResolvedModel:         "claude-3-opus",
		ResolvedProvider:      "anthropic",
		DependsOn:             []string{"phase-0"},
		EstimatedInputTokens:  1000,
		EstimatedOutputTokens: 500,
		EstimatedCost:         0.05,
		BatchIndex:            1,
	}

	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	var parsed PhasePlan
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if parsed.PhaseID != original.PhaseID {
		t.Errorf("expected PhaseID %s, got %s", original.PhaseID, parsed.PhaseID)
	}
	if parsed.PhaseName != original.PhaseName {
		t.Errorf("expected PhaseName %s, got %s", original.PhaseName, parsed.PhaseName)
	}
	if parsed.RoutingProfile != original.RoutingProfile {
		t.Errorf("expected RoutingProfile %s, got %s", original.RoutingProfile, parsed.RoutingProfile)
	}
	if parsed.ResolvedModel != original.ResolvedModel {
		t.Errorf("expected ResolvedModel %s, got %s", original.ResolvedModel, parsed.ResolvedModel)
	}
	if parsed.ResolvedProvider != original.ResolvedProvider {
		t.Errorf("expected ResolvedProvider %s, got %s", original.ResolvedProvider, parsed.ResolvedProvider)
	}
	if len(parsed.DependsOn) != 1 || parsed.DependsOn[0] != "phase-0" {
		t.Errorf("expected DependsOn [phase-0], got %v", parsed.DependsOn)
	}
	if parsed.EstimatedInputTokens != original.EstimatedInputTokens {
		t.Errorf("expected EstimatedInputTokens %d, got %d", original.EstimatedInputTokens, parsed.EstimatedInputTokens)
	}
	if parsed.EstimatedOutputTokens != original.EstimatedOutputTokens {
		t.Errorf("expected EstimatedOutputTokens %d, got %d", original.EstimatedOutputTokens, parsed.EstimatedOutputTokens)
	}
	if parsed.EstimatedCost != original.EstimatedCost {
		t.Errorf("expected EstimatedCost %f, got %f", original.EstimatedCost, parsed.EstimatedCost)
	}
	if parsed.BatchIndex != original.BatchIndex {
		t.Errorf("expected BatchIndex %d, got %d", original.BatchIndex, parsed.BatchIndex)
	}
}
