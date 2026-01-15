package workflow

import (
	"context"
	"testing"

	domainProvider "github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// mockTokenEstimator is a simple mock for testing.
type mockTokenEstimator struct {
	tokensPerChar float64
}

func (m *mockTokenEstimator) CountTokens(text string) int {
	if m.tokensPerChar <= 0 {
		m.tokensPerChar = 0.25 // ~4 chars per token
	}
	return int(float64(len(text)) * m.tokensPerChar)
}

func TestDefaultPlannerConfig(t *testing.T) {
	config := DefaultPlannerConfig()

	if config.OutputTokenFraction != 0.5 {
		t.Errorf("expected OutputTokenFraction 0.5, got %f", config.OutputTokenFraction)
	}
	if config.DefaultOutputTokens != 500 {
		t.Errorf("expected DefaultOutputTokens 500, got %d", config.DefaultOutputTokens)
	}
}

func TestNewPlanner(t *testing.T) {
	estimator := &mockTokenEstimator{}
	config := DefaultPlannerConfig()

	planner := NewPlanner(nil, nil, estimator, config)

	if planner == nil {
		t.Fatal("expected non-nil Planner")
	}
	if planner.tokenEstimator != estimator {
		t.Error("tokenEstimator not set correctly")
	}
}

func TestNewPlanner_InvalidConfig(t *testing.T) {
	tests := []struct {
		name                  string
		fraction              float64
		defaultTokens         int
		expectedFraction      float64
		expectedDefaultTokens int
	}{
		{
			name:                  "zero fraction",
			fraction:              0,
			defaultTokens:         500,
			expectedFraction:      0.5,
			expectedDefaultTokens: 500,
		},
		{
			name:                  "negative fraction",
			fraction:              -0.5,
			defaultTokens:         500,
			expectedFraction:      0.5,
			expectedDefaultTokens: 500,
		},
		{
			name:                  "fraction > 1",
			fraction:              1.5,
			defaultTokens:         500,
			expectedFraction:      0.5,
			expectedDefaultTokens: 500,
		},
		{
			name:                  "zero default tokens",
			fraction:              0.5,
			defaultTokens:         0,
			expectedFraction:      0.5,
			expectedDefaultTokens: 500,
		},
		{
			name:                  "negative default tokens",
			fraction:              0.5,
			defaultTokens:         -100,
			expectedFraction:      0.5,
			expectedDefaultTokens: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := PlannerConfig{
				OutputTokenFraction: tt.fraction,
				DefaultOutputTokens: tt.defaultTokens,
			}
			planner := NewPlanner(nil, nil, nil, config)

			if planner.config.OutputTokenFraction != tt.expectedFraction {
				t.Errorf("expected OutputTokenFraction %f, got %f",
					tt.expectedFraction, planner.config.OutputTokenFraction)
			}
			if planner.config.DefaultOutputTokens != tt.expectedDefaultTokens {
				t.Errorf("expected DefaultOutputTokens %d, got %d",
					tt.expectedDefaultTokens, planner.config.DefaultOutputTokens)
			}
		})
	}
}

func TestPlanner_GeneratePlan_NilSkill(t *testing.T) {
	planner := NewPlanner(nil, nil, nil, DefaultPlannerConfig())

	_, err := planner.GeneratePlan(context.Background(), nil, "input", "")
	if err == nil {
		t.Error("expected error for nil skill")
	}
}

func TestPlanner_GeneratePlan_SinglePhase(t *testing.T) {
	estimator := &mockTokenEstimator{tokensPerChar: 0.25}
	planner := NewPlanner(nil, nil, estimator, DefaultPlannerConfig())

	phase, err := skill.NewPhase("phase-1", "Phase 1", "Process this: {{.input}}")
	if err != nil {
		t.Fatalf("NewPhase error: %v", err)
	}
	phase.WithRoutingProfile(skill.ProfileBalanced).WithMaxTokens(1000)

	sk, err := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})
	if err != nil {
		t.Fatalf("NewSkill error: %v", err)
	}

	plan, err := planner.GeneratePlan(context.Background(), sk, "test input", "")
	if err != nil {
		t.Fatalf("GeneratePlan() error: %v", err)
	}

	if plan.SkillID != "test-skill" {
		t.Errorf("expected SkillID 'test-skill', got %s", plan.SkillID)
	}
	if plan.SkillName != "Test Skill" {
		t.Errorf("expected SkillName 'Test Skill', got %s", plan.SkillName)
	}
	if plan.Input != "test input" {
		t.Errorf("expected Input 'test input', got %s", plan.Input)
	}
	if plan.PhaseCount() != 1 {
		t.Errorf("expected 1 phase, got %d", plan.PhaseCount())
	}
	if plan.BatchCount() != 1 {
		t.Errorf("expected 1 batch, got %d", plan.BatchCount())
	}

	// Check phase plan
	phasePlan := plan.GetPhase("phase-1")
	if phasePlan == nil {
		t.Fatal("expected to find phase-1")
	}
	if phasePlan.PhaseName != "Phase 1" {
		t.Errorf("expected PhaseName 'Phase 1', got %s", phasePlan.PhaseName)
	}
	if phasePlan.RoutingProfile != skill.ProfileBalanced {
		t.Errorf("expected RoutingProfile 'balanced', got %s", phasePlan.RoutingProfile)
	}
	// Without router, should use placeholder
	if phasePlan.ResolvedModel != "balanced-model" {
		t.Errorf("expected ResolvedModel 'balanced-model', got %s", phasePlan.ResolvedModel)
	}
	if phasePlan.EstimatedInputTokens <= 0 {
		t.Errorf("expected positive EstimatedInputTokens, got %d", phasePlan.EstimatedInputTokens)
	}
	// Output tokens should be 50% of MaxTokens (1000 * 0.5 = 500)
	if phasePlan.EstimatedOutputTokens != 500 {
		t.Errorf("expected EstimatedOutputTokens 500, got %d", phasePlan.EstimatedOutputTokens)
	}
}

func TestPlanner_GeneratePlan_MultiplePhases(t *testing.T) {
	estimator := &mockTokenEstimator{tokensPerChar: 0.25}
	planner := NewPlanner(nil, nil, estimator, DefaultPlannerConfig())

	phase1, err := skill.NewPhase("phase-1", "Phase 1", "Analyze: {{.input}}")
	if err != nil {
		t.Fatalf("NewPhase error: %v", err)
	}
	phase1.WithRoutingProfile(skill.ProfileCheap).WithMaxTokens(500)

	phase2, err := skill.NewPhase("phase-2", "Phase 2", "Review: {{.phase-1}}")
	if err != nil {
		t.Fatalf("NewPhase error: %v", err)
	}
	phase2.WithRoutingProfile(skill.ProfilePremium).WithMaxTokens(1000).WithDependencies([]string{"phase-1"})

	sk, err := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase1, *phase2})
	if err != nil {
		t.Fatalf("NewSkill error: %v", err)
	}

	plan, err := planner.GeneratePlan(context.Background(), sk, "test input", "")
	if err != nil {
		t.Fatalf("GeneratePlan() error: %v", err)
	}

	if plan.PhaseCount() != 2 {
		t.Errorf("expected 2 phases, got %d", plan.PhaseCount())
	}
	if plan.BatchCount() != 2 {
		t.Errorf("expected 2 batches (sequential execution), got %d", plan.BatchCount())
	}

	// Check phase 1 (cheap profile)
	p1 := plan.GetPhase("phase-1")
	if p1 == nil {
		t.Fatal("expected to find phase-1")
	}
	if p1.ResolvedModel != "local-model" {
		t.Errorf("expected ResolvedModel 'local-model' for cheap profile, got %s", p1.ResolvedModel)
	}
	if p1.ResolvedProvider != "ollama" {
		t.Errorf("expected ResolvedProvider 'ollama' for cheap profile, got %s", p1.ResolvedProvider)
	}
	if p1.BatchIndex != 0 {
		t.Errorf("expected BatchIndex 0, got %d", p1.BatchIndex)
	}

	// Check phase 2 (premium profile, depends on phase-1)
	p2 := plan.GetPhase("phase-2")
	if p2 == nil {
		t.Fatal("expected to find phase-2")
	}
	if p2.ResolvedModel != "premium-model" {
		t.Errorf("expected ResolvedModel 'premium-model' for premium profile, got %s", p2.ResolvedModel)
	}
	if p2.ResolvedProvider != "anthropic" {
		t.Errorf("expected ResolvedProvider 'anthropic' for premium profile, got %s", p2.ResolvedProvider)
	}
	if len(p2.DependsOn) != 1 || p2.DependsOn[0] != "phase-1" {
		t.Errorf("expected DependsOn [phase-1], got %v", p2.DependsOn)
	}
	if p2.BatchIndex != 1 {
		t.Errorf("expected BatchIndex 1, got %d", p2.BatchIndex)
	}
}

func TestPlanner_GeneratePlan_ParallelPhases(t *testing.T) {
	estimator := &mockTokenEstimator{tokensPerChar: 0.25}
	planner := NewPlanner(nil, nil, estimator, DefaultPlannerConfig())

	// Create three phases that can run in parallel (no dependencies between first two)
	phaseA, _ := skill.NewPhase("phase-a", "Phase A", "Process A: {{.input}}")
	phaseA.WithRoutingProfile(skill.ProfileBalanced).WithMaxTokens(500)

	phaseB, _ := skill.NewPhase("phase-b", "Phase B", "Process B: {{.input}}")
	phaseB.WithRoutingProfile(skill.ProfileBalanced).WithMaxTokens(500)

	phaseC, _ := skill.NewPhase("phase-c", "Phase C", "Combine: {{.phase-a}} {{.phase-b}}")
	phaseC.WithRoutingProfile(skill.ProfileBalanced).WithMaxTokens(500).WithDependencies([]string{"phase-a", "phase-b"})

	sk, _ := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phaseA, *phaseB, *phaseC})

	plan, err := planner.GeneratePlan(context.Background(), sk, "test input", "")
	if err != nil {
		t.Fatalf("GeneratePlan() error: %v", err)
	}

	if plan.PhaseCount() != 3 {
		t.Errorf("expected 3 phases, got %d", plan.PhaseCount())
	}
	if plan.BatchCount() != 2 {
		t.Errorf("expected 2 batches (A,B parallel then C), got %d", plan.BatchCount())
	}

	// Check that phase A and B are in batch 0
	pA := plan.GetPhase("phase-a")
	pB := plan.GetPhase("phase-b")
	pC := plan.GetPhase("phase-c")

	if pA.BatchIndex != 0 {
		t.Errorf("expected phase-a BatchIndex 0, got %d", pA.BatchIndex)
	}
	if pB.BatchIndex != 0 {
		t.Errorf("expected phase-b BatchIndex 0, got %d", pB.BatchIndex)
	}
	if pC.BatchIndex != 1 {
		t.Errorf("expected phase-c BatchIndex 1, got %d", pC.BatchIndex)
	}
}

func TestPlanner_GeneratePlan_WithMemoryContent(t *testing.T) {
	estimator := &mockTokenEstimator{tokensPerChar: 0.25}
	planner := NewPlanner(nil, nil, estimator, DefaultPlannerConfig())

	phase, _ := skill.NewPhase("phase-1", "Phase 1", "Process: {{.input}}")
	phase.WithRoutingProfile(skill.ProfileBalanced).WithMaxTokens(1000)

	sk, _ := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})

	memoryContent := "This is the project memory content with some context about the codebase."

	planWithoutMemory, _ := planner.GeneratePlan(context.Background(), sk, "test input", "")
	planWithMemory, _ := planner.GeneratePlan(context.Background(), sk, "test input", memoryContent)

	// Plan with memory should have more input tokens
	p1NoMem := planWithoutMemory.GetPhase("phase-1")
	p1WithMem := planWithMemory.GetPhase("phase-1")

	if p1WithMem.EstimatedInputTokens <= p1NoMem.EstimatedInputTokens {
		t.Errorf("expected more tokens with memory (%d) than without (%d)",
			p1WithMem.EstimatedInputTokens, p1NoMem.EstimatedInputTokens)
	}
}

func TestPlanner_GeneratePlan_WithCostCalculator(t *testing.T) {
	estimator := &mockTokenEstimator{tokensPerChar: 0.25}
	calculator := domainProvider.NewCostCalculator()

	// Register models with costs
	calculator.RegisterModelWithProvider("balanced-model", "anthropic", 3.0, 15.0) // $3/1K input, $15/1K output

	planner := NewPlanner(nil, calculator, estimator, DefaultPlannerConfig())

	phase, _ := skill.NewPhase("phase-1", "Phase 1", "Process: {{.input}}")
	phase.WithRoutingProfile(skill.ProfileBalanced).WithMaxTokens(1000)

	sk, _ := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})

	plan, err := planner.GeneratePlan(context.Background(), sk, "test input", "")
	if err != nil {
		t.Fatalf("GeneratePlan() error: %v", err)
	}

	p1 := plan.GetPhase("phase-1")
	if p1.EstimatedCost <= 0 {
		t.Errorf("expected positive EstimatedCost, got %f", p1.EstimatedCost)
	}
	if plan.TotalEstimatedCost <= 0 {
		t.Errorf("expected positive TotalEstimatedCost, got %f", plan.TotalEstimatedCost)
	}
}

func TestPlanner_GeneratePlan_NoTokenEstimator(t *testing.T) {
	planner := NewPlanner(nil, nil, nil, DefaultPlannerConfig())

	phase, _ := skill.NewPhase("phase-1", "Phase 1", "Process: {{.input}}")
	phase.WithRoutingProfile(skill.ProfileBalanced).WithMaxTokens(1000)

	sk, _ := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})

	plan, err := planner.GeneratePlan(context.Background(), sk, "test input", "")
	if err != nil {
		t.Fatalf("GeneratePlan() error: %v", err)
	}

	p1 := plan.GetPhase("phase-1")
	// Without token estimator, input tokens should be 0
	if p1.EstimatedInputTokens != 0 {
		t.Errorf("expected 0 EstimatedInputTokens without estimator, got %d", p1.EstimatedInputTokens)
	}
	// Output tokens should still use the default calculation
	if p1.EstimatedOutputTokens != 500 {
		t.Errorf("expected 500 EstimatedOutputTokens (50%% of 1000), got %d", p1.EstimatedOutputTokens)
	}
}

func TestPlanner_EstimateOutputTokens_DefaultMaxTokens(t *testing.T) {
	config := DefaultPlannerConfig()
	planner := NewPlanner(nil, nil, nil, config)

	// Create phase with default MaxTokens (which is 4096)
	phase, _ := skill.NewPhase("phase-1", "Phase 1", "Process: {{.input}}")

	tokens := planner.estimateOutputTokens(phase)
	// 4096 * 0.5 = 2048
	expected := int(float64(skill.DefaultMaxTokens) * config.OutputTokenFraction)
	if tokens != expected {
		t.Errorf("expected %d tokens, got %d", expected, tokens)
	}
}

func TestPlanner_RenderTemplate(t *testing.T) {
	planner := NewPlanner(nil, nil, nil, DefaultPlannerConfig())

	tests := []struct {
		name     string
		template string
		input    string
		contains string
	}{
		{
			name:     "simple template",
			template: "Process: {{.input}}",
			input:    "hello world",
			contains: "hello world",
		},
		{
			name:     "template with _input",
			template: "Data: {{._input}}",
			input:    "test data",
			contains: "test data",
		},
		{
			name:     "invalid template returns original",
			template: "Invalid: {{.missing}",
			input:    "input",
			contains: "{{.missing}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := planner.renderTemplate(tt.template, tt.input)
			if !containsString(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
