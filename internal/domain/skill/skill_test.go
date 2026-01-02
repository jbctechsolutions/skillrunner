package skill

import (
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// Helper to create a valid phase for testing
func validPhase(id, name string) Phase {
	p, _ := NewPhase(id, name, "test prompt")
	return *p
}

func TestNewSkill(t *testing.T) {
	t.Run("creates skill with valid inputs", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, err := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if skill == nil {
			t.Fatal("expected skill to be created")
		}
		if skill.ID() != "skill-1" {
			t.Errorf("expected id 'skill-1', got '%s'", skill.ID())
		}
		if skill.Name() != "Test Skill" {
			t.Errorf("expected name 'Test Skill', got '%s'", skill.Name())
		}
		if skill.Version() != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", skill.Version())
		}
	})

	t.Run("trims whitespace from inputs", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, err := NewSkill("  skill-1  ", "  Test Skill  ", "  1.0.0  ", phases)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if skill.ID() != "skill-1" {
			t.Errorf("expected trimmed id 'skill-1', got '%s'", skill.ID())
		}
		if skill.Name() != "Test Skill" {
			t.Errorf("expected trimmed name 'Test Skill', got '%s'", skill.Name())
		}
		if skill.Version() != "1.0.0" {
			t.Errorf("expected trimmed version '1.0.0', got '%s'", skill.Version())
		}
	})

	t.Run("returns error when id is empty", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		_, err := NewSkill("", "Test Skill", "1.0.0", phases)

		if err == nil {
			t.Fatal("expected error for empty id")
		}
		if !errors.Is(err, errors.ErrSkillIDRequired) {
			t.Errorf("expected ErrSkillIDRequired, got %v", err)
		}
	})

	t.Run("returns error when id is only whitespace", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		_, err := NewSkill("   ", "Test Skill", "1.0.0", phases)

		if err == nil {
			t.Fatal("expected error for whitespace-only id")
		}
		if !errors.Is(err, errors.ErrSkillIDRequired) {
			t.Errorf("expected ErrSkillIDRequired, got %v", err)
		}
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		_, err := NewSkill("skill-1", "", "1.0.0", phases)

		if err == nil {
			t.Fatal("expected error for empty name")
		}
		if !errors.Is(err, errors.ErrSkillNameRequired) {
			t.Errorf("expected ErrSkillNameRequired, got %v", err)
		}
	})

	t.Run("returns error when phases is empty", func(t *testing.T) {
		_, err := NewSkill("skill-1", "Test Skill", "1.0.0", []Phase{})

		if err == nil {
			t.Fatal("expected error for empty phases")
		}
		if !errors.Is(err, errors.ErrNoPhasesDefied) {
			t.Errorf("expected ErrNoPhasesDefied, got %v", err)
		}
	})

	t.Run("returns error when phases is nil", func(t *testing.T) {
		_, err := NewSkill("skill-1", "Test Skill", "1.0.0", nil)

		if err == nil {
			t.Fatal("expected error for nil phases")
		}
		if !errors.Is(err, errors.ErrNoPhasesDefied) {
			t.Errorf("expected ErrNoPhasesDefied, got %v", err)
		}
	})

	t.Run("creates defensive copy of phases", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		// Modify original phases
		phases[0].ID = "modified"

		// Skill should still have original phase
		skillPhases := skill.Phases()
		if skillPhases[0].ID != "phase-1" {
			t.Error("skill phases should not be affected by external mutation")
		}
	})

	t.Run("sets default routing config", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		routing := skill.Routing()
		if routing.DefaultProfile != ProfileBalanced {
			t.Errorf("expected default profile 'balanced', got '%s'", routing.DefaultProfile)
		}
	})

	t.Run("initializes empty metadata", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		metadata := skill.Metadata()
		if metadata == nil {
			t.Error("expected metadata to be initialized")
		}
		if len(metadata) != 0 {
			t.Error("expected metadata to be empty")
		}
	})
}

func TestSkillGetters(t *testing.T) {
	t.Run("Phases returns defensive copy", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		returnedPhases := skill.Phases()
		returnedPhases[0].ID = "modified"

		// Original skill phases should be unchanged
		originalPhases := skill.Phases()
		if originalPhases[0].ID != "phase-1" {
			t.Error("Phases() should return a defensive copy")
		}
	})

	t.Run("Metadata returns defensive copy", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)
		skill.SetMetadata("key", "value")

		returnedMeta := skill.Metadata()
		returnedMeta["key"] = "modified"

		// Original metadata should be unchanged
		originalMeta := skill.Metadata()
		if originalMeta["key"] != "value" {
			t.Error("Metadata() should return a defensive copy")
		}
	})
}

func TestSkillSetters(t *testing.T) {
	t.Run("SetDescription sets the description", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		skill.SetDescription("A test skill")

		if skill.Description() != "A test skill" {
			t.Errorf("expected description 'A test skill', got '%s'", skill.Description())
		}
	})

	t.Run("SetRouting sets the routing configuration", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		newRouting := *NewRoutingConfig().WithDefaultProfile(ProfilePremium)
		skill.SetRouting(newRouting)

		routing := skill.Routing()
		if routing.DefaultProfile != ProfilePremium {
			t.Errorf("expected profile 'premium', got '%s'", routing.DefaultProfile)
		}
	})

	t.Run("SetMetadata sets metadata values", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		skill.SetMetadata("author", "test")
		skill.SetMetadata("count", 42)

		metadata := skill.Metadata()
		if metadata["author"] != "test" {
			t.Errorf("expected author 'test', got '%v'", metadata["author"])
		}
		if metadata["count"] != 42 {
			t.Errorf("expected count 42, got '%v'", metadata["count"])
		}
	})
}

func TestSkillGetPhase(t *testing.T) {
	t.Run("returns phase when found", func(t *testing.T) {
		phases := []Phase{
			validPhase("phase-1", "Phase 1"),
			validPhase("phase-2", "Phase 2"),
		}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		phase, err := skill.GetPhase("phase-2")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if phase == nil {
			t.Fatal("expected phase to be returned")
		}
		if phase.Name != "Phase 2" {
			t.Errorf("expected phase name 'Phase 2', got '%s'", phase.Name)
		}
	})

	t.Run("returns error when phase not found", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		_, err := skill.GetPhase("nonexistent")

		if err == nil {
			t.Fatal("expected error for nonexistent phase")
		}
		if !errors.Is(err, errors.ErrPhaseNotFound) {
			t.Errorf("expected ErrPhaseNotFound, got %v", err)
		}
	})
}

func TestSkillValidate(t *testing.T) {
	t.Run("valid skill passes validation", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		err := skill.Validate()

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("validates phases", func(t *testing.T) {
		phase, _ := NewPhase("phase-1", "Phase 1", "prompt")
		phase.RoutingProfile = "invalid"
		phases := []Phase{*phase}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		err := skill.Validate()

		if err == nil {
			t.Fatal("expected error for invalid phase")
		}
	})

	t.Run("validates routing configuration", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)
		skill.routing.DefaultProfile = "invalid"

		err := skill.Validate()

		if err == nil {
			t.Fatal("expected error for invalid routing config")
		}
	})

	t.Run("detects missing dependency", func(t *testing.T) {
		phase, _ := NewPhase("phase-1", "Phase 1", "prompt")
		phase = phase.WithDependencies([]string{"nonexistent"})
		phases := []Phase{*phase}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		err := skill.Validate()

		if err == nil {
			t.Fatal("expected error for missing dependency")
		}
		if !errors.Is(err, errors.ErrDependencyNotFound) {
			t.Errorf("expected ErrDependencyNotFound, got %v", err)
		}
	})

	t.Run("detects simple cycle", func(t *testing.T) {
		phase1, _ := NewPhase("phase-1", "Phase 1", "prompt")
		phase1 = phase1.WithDependencies([]string{"phase-2"})
		phase2, _ := NewPhase("phase-2", "Phase 2", "prompt")
		phase2 = phase2.WithDependencies([]string{"phase-1"})
		phases := []Phase{*phase1, *phase2}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		err := skill.Validate()

		if err == nil {
			t.Fatal("expected error for cycle")
		}
		if !errors.Is(err, errors.ErrCycleDetected) {
			t.Errorf("expected ErrCycleDetected, got %v", err)
		}
	})

	t.Run("detects self-referential cycle", func(t *testing.T) {
		phase, _ := NewPhase("phase-1", "Phase 1", "prompt")
		phase = phase.WithDependencies([]string{"phase-1"})
		phases := []Phase{*phase}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		err := skill.Validate()

		if err == nil {
			t.Fatal("expected error for self-referential cycle")
		}
		if !errors.Is(err, errors.ErrCycleDetected) {
			t.Errorf("expected ErrCycleDetected, got %v", err)
		}
	})

	t.Run("detects complex cycle", func(t *testing.T) {
		// A -> B -> C -> A
		phaseA, _ := NewPhase("a", "Phase A", "prompt")
		phaseA = phaseA.WithDependencies([]string{"c"})
		phaseB, _ := NewPhase("b", "Phase B", "prompt")
		phaseB = phaseB.WithDependencies([]string{"a"})
		phaseC, _ := NewPhase("c", "Phase C", "prompt")
		phaseC = phaseC.WithDependencies([]string{"b"})
		phases := []Phase{*phaseA, *phaseB, *phaseC}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		err := skill.Validate()

		if err == nil {
			t.Fatal("expected error for complex cycle")
		}
		if !errors.Is(err, errors.ErrCycleDetected) {
			t.Errorf("expected ErrCycleDetected, got %v", err)
		}
	})

	t.Run("valid DAG passes validation", func(t *testing.T) {
		// A -> B -> C (no cycle)
		phaseA, _ := NewPhase("a", "Phase A", "prompt")
		phaseB, _ := NewPhase("b", "Phase B", "prompt")
		phaseB = phaseB.WithDependencies([]string{"a"})
		phaseC, _ := NewPhase("c", "Phase C", "prompt")
		phaseC = phaseC.WithDependencies([]string{"b"})
		phases := []Phase{*phaseA, *phaseB, *phaseC}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		err := skill.Validate()

		if err != nil {
			t.Fatalf("expected no error for valid DAG, got %v", err)
		}
	})

	t.Run("diamond dependency passes validation", func(t *testing.T) {
		// A -> B, A -> C, B -> D, C -> D (diamond, no cycle)
		phaseA, _ := NewPhase("a", "Phase A", "prompt")
		phaseB, _ := NewPhase("b", "Phase B", "prompt")
		phaseB = phaseB.WithDependencies([]string{"a"})
		phaseC, _ := NewPhase("c", "Phase C", "prompt")
		phaseC = phaseC.WithDependencies([]string{"a"})
		phaseD, _ := NewPhase("d", "Phase D", "prompt")
		phaseD = phaseD.WithDependencies([]string{"b", "c"})
		phases := []Phase{*phaseA, *phaseB, *phaseC, *phaseD}
		skill, _ := NewSkill("skill-1", "Test Skill", "1.0.0", phases)

		err := skill.Validate()

		if err != nil {
			t.Fatalf("expected no error for diamond dependency, got %v", err)
		}
	})
}

func TestHasCycle(t *testing.T) {
	t.Run("returns false for empty phases", func(t *testing.T) {
		if hasCycle([]Phase{}) {
			t.Error("expected no cycle for empty phases")
		}
	})

	t.Run("returns false for single phase without dependencies", func(t *testing.T) {
		phases := []Phase{validPhase("phase-1", "Phase 1")}
		if hasCycle(phases) {
			t.Error("expected no cycle for single phase")
		}
	})

	t.Run("returns true for self-referential phase", func(t *testing.T) {
		phase, _ := NewPhase("phase-1", "Phase 1", "prompt")
		phase = phase.WithDependencies([]string{"phase-1"})
		phases := []Phase{*phase}
		if !hasCycle(phases) {
			t.Error("expected cycle for self-referential phase")
		}
	})
}
