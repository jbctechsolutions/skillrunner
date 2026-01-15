package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	infraSkills "github.com/jbctechsolutions/skillrunner/internal/infrastructure/skills"
)

func TestNewRegistry(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	if registry.Count() != 0 {
		t.Errorf("expected empty registry, got %d skills", registry.Count())
	}

	if registry.IsLoaded() {
		t.Error("expected IsLoaded to return false for new registry")
	}
}

func TestRegisterAndGetSkill(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Create a test skill
	phase, err := skill.NewPhase("test-phase", "Test Phase", "Do something: {{.input}}")
	if err != nil {
		t.Fatalf("failed to create phase: %v", err)
	}

	testSkill, err := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})
	if err != nil {
		t.Fatalf("failed to create skill: %v", err)
	}
	testSkill.SetDescription("A test skill for unit testing")

	// Register the skill
	err = registry.Register(testSkill)
	if err != nil {
		t.Fatalf("failed to register skill: %v", err)
	}

	// Verify count
	if registry.Count() != 1 {
		t.Errorf("expected count 1, got %d", registry.Count())
	}

	// Get by ID
	retrieved := registry.GetSkill("test-skill")
	if retrieved == nil {
		t.Fatal("expected to retrieve skill by ID")
	}
	if retrieved.ID() != "test-skill" {
		t.Errorf("expected ID 'test-skill', got %s", retrieved.ID())
	}
	if retrieved.Name() != "Test Skill" {
		t.Errorf("expected name 'Test Skill', got %s", retrieved.Name())
	}

	// Get by name
	byName := registry.GetSkillByName("Test Skill")
	if byName == nil {
		t.Fatal("expected to retrieve skill by name")
	}
	if byName.ID() != "test-skill" {
		t.Errorf("expected ID 'test-skill', got %s", byName.ID())
	}
}

func TestGetNonExistentSkill(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Get non-existent skill by ID
	s := registry.GetSkill("non-existent")
	if s != nil {
		t.Error("expected nil for non-existent skill ID")
	}

	// Get non-existent skill by name
	s = registry.GetSkillByName("Non Existent")
	if s != nil {
		t.Error("expected nil for non-existent skill name")
	}
}

func TestListSkills(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Empty registry
	skills := registry.ListSkills()
	if len(skills) != 0 {
		t.Errorf("expected empty list, got %d skills", len(skills))
	}

	// Add some skills
	for i := 0; i < 3; i++ {
		phase, _ := skill.NewPhase("phase", "Phase", "template")
		s, _ := skill.NewSkill(
			"skill-"+string(rune('a'+i)),
			"Skill "+string(rune('A'+i)),
			"1.0.0",
			[]skill.Phase{*phase},
		)
		registry.Register(s)
	}

	skills = registry.ListSkills()
	if len(skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(skills))
	}
}

func TestUnregister(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Register a skill
	phase, _ := skill.NewPhase("phase", "Phase", "template")
	s, _ := skill.NewSkill("to-remove", "To Remove", "1.0.0", []skill.Phase{*phase})
	registry.Register(s)

	if registry.Count() != 1 {
		t.Errorf("expected 1 skill, got %d", registry.Count())
	}

	// Unregister it
	registry.Unregister("to-remove")

	if registry.Count() != 0 {
		t.Errorf("expected 0 skills after unregister, got %d", registry.Count())
	}

	if registry.GetSkill("to-remove") != nil {
		t.Error("expected nil after unregister")
	}
}

func TestClear(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Add some skills
	for i := 0; i < 5; i++ {
		phase, _ := skill.NewPhase("phase", "Phase", "template")
		s, _ := skill.NewSkill(
			"skill-"+string(rune('a'+i)),
			"Skill",
			"1.0.0",
			[]skill.Phase{*phase},
		)
		registry.Register(s)
	}

	if registry.Count() != 5 {
		t.Errorf("expected 5 skills, got %d", registry.Count())
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("expected 0 skills after clear, got %d", registry.Count())
	}

	if registry.IsLoaded() {
		t.Error("expected IsLoaded to return false after clear")
	}
}

func TestRegisterNilSkill(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	err := registry.Register(nil)
	if err == nil {
		t.Error("expected error when registering nil skill")
	}
}

func TestLoadBuiltInSkillsNonExistentDir(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	registry.SetBuiltInDir("/non/existent/path")

	// Should not error for non-existent directory
	err := registry.LoadBuiltInSkills()
	if err != nil {
		t.Errorf("expected no error for non-existent built-in dir, got: %v", err)
	}
}

func TestLoadUserSkillsNonExistentDir(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Should not error for non-existent directory
	err := registry.LoadUserSkills("/non/existent/path")
	if err != nil {
		t.Errorf("expected no error for non-existent user dir, got: %v", err)
	}
}

func TestLoadSkillsFromDirectory(t *testing.T) {
	// Create a temp directory with a test skill file
	tmpDir, err := os.MkdirTemp("", "skillrunner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a test skill YAML file
	skillYAML := `
id: test-skill
name: Test Skill
version: "1.0.0"
description: A test skill
phases:
  - id: main
    name: Main Phase
    prompt_template: "Process: {{.input}}"
routing:
  default_profile: balanced
`
	skillPath := filepath.Join(tmpDir, "test-skill.yaml")
	if err := os.WriteFile(skillPath, []byte(skillYAML), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	// Load skills from the temp directory
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	registry.SetBuiltInDir(tmpDir)

	err = registry.LoadBuiltInSkills()
	if err != nil {
		t.Fatalf("failed to load built-in skills: %v", err)
	}

	// Verify the skill was loaded
	if registry.Count() != 1 {
		t.Errorf("expected 1 skill, got %d", registry.Count())
	}

	s := registry.GetSkill("test-skill")
	if s == nil {
		t.Fatal("expected to find test-skill")
	}
	if s.Name() != "Test Skill" {
		t.Errorf("expected name 'Test Skill', got %s", s.Name())
	}
	if s.Description() != "A test skill" {
		t.Errorf("expected description 'A test skill', got %s", s.Description())
	}
}

func TestLoadAll(t *testing.T) {
	// Create temp directories
	builtInDir, err := os.MkdirTemp("", "skillrunner-builtin-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(builtInDir)

	userDir, err := os.MkdirTemp("", "skillrunner-user-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(userDir)

	// Write built-in skill
	builtInYAML := `
id: builtin-skill
name: Built-in Skill
version: "1.0.0"
phases:
  - id: main
    name: Main
    prompt_template: "Built-in: {{.input}}"
`
	if err := os.WriteFile(filepath.Join(builtInDir, "builtin.yaml"), []byte(builtInYAML), 0644); err != nil {
		t.Fatalf("failed to write built-in skill: %v", err)
	}

	// Write user skill
	userYAML := `
id: user-skill
name: User Skill
version: "1.0.0"
phases:
  - id: main
    name: Main
    prompt_template: "User: {{.input}}"
`
	if err := os.WriteFile(filepath.Join(userDir, "user.yaml"), []byte(userYAML), 0644); err != nil {
		t.Fatalf("failed to write user skill: %v", err)
	}

	// Load all skills
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	registry.SetBuiltInDir(builtInDir)
	registry.SetUserDir(userDir)

	err = registry.LoadAll()
	if err != nil {
		t.Fatalf("failed to load all skills: %v", err)
	}

	// Verify both skills were loaded
	if registry.Count() != 2 {
		t.Errorf("expected 2 skills, got %d", registry.Count())
	}

	if !registry.IsLoaded() {
		t.Error("expected IsLoaded to return true after LoadAll")
	}

	if registry.GetSkill("builtin-skill") == nil {
		t.Error("expected to find builtin-skill")
	}
	if registry.GetSkill("user-skill") == nil {
		t.Error("expected to find user-skill")
	}
}

func TestUserSkillsOverrideBuiltIn(t *testing.T) {
	// Create temp directories
	builtInDir, err := os.MkdirTemp("", "skillrunner-builtin-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(builtInDir)

	userDir, err := os.MkdirTemp("", "skillrunner-user-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(userDir)

	// Write built-in skill with same ID
	builtInYAML := `
id: same-skill
name: Built-in Version
version: "1.0.0"
description: Original built-in version
phases:
  - id: main
    name: Main
    prompt_template: "Built-in: {{.input}}"
`
	if err := os.WriteFile(filepath.Join(builtInDir, "skill.yaml"), []byte(builtInYAML), 0644); err != nil {
		t.Fatalf("failed to write built-in skill: %v", err)
	}

	// Write user skill with same ID (should override)
	userYAML := `
id: same-skill
name: User Version
version: "2.0.0"
description: User-customized version
phases:
  - id: main
    name: Main
    prompt_template: "User: {{.input}}"
`
	if err := os.WriteFile(filepath.Join(userDir, "skill.yaml"), []byte(userYAML), 0644); err != nil {
		t.Fatalf("failed to write user skill: %v", err)
	}

	// Load all skills
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	registry.SetBuiltInDir(builtInDir)
	registry.SetUserDir(userDir)

	err = registry.LoadAll()
	if err != nil {
		t.Fatalf("failed to load all skills: %v", err)
	}

	// Verify only one skill exists (user overrides built-in)
	if registry.Count() != 1 {
		t.Errorf("expected 1 skill, got %d", registry.Count())
	}

	s := registry.GetSkill("same-skill")
	if s == nil {
		t.Fatal("expected to find same-skill")
	}

	// Verify it's the user version
	if s.Name() != "User Version" {
		t.Errorf("expected user version with name 'User Version', got %s", s.Name())
	}
	if s.Version() != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %s", s.Version())
	}
}

func TestRegisterWithSource(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Create a test skill
	phase, err := skill.NewPhase("test-phase", "Test Phase", "Do something: {{.input}}")
	if err != nil {
		t.Fatalf("failed to create phase: %v", err)
	}

	testSkill, err := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})
	if err != nil {
		t.Fatalf("failed to create skill: %v", err)
	}

	t.Run("registers skill with source", func(t *testing.T) {
		err := registry.RegisterWithSource(testSkill, "/path/to/skill.yaml", skill.SourceUser)
		if err != nil {
			t.Fatalf("failed to register skill with source: %v", err)
		}

		// Verify skill is registered
		s := registry.GetSkill("test-skill")
		if s == nil {
			t.Fatal("expected to find skill")
		}

		// Verify source tracking
		source := registry.GetSource("test-skill")
		if source == nil {
			t.Fatal("expected to find source")
		}
		if source.SkillID() != "test-skill" {
			t.Errorf("expected skill ID 'test-skill', got %q", source.SkillID())
		}
		if source.FilePath() != "/path/to/skill.yaml" {
			t.Errorf("expected file path '/path/to/skill.yaml', got %q", source.FilePath())
		}
		if source.Source() != skill.SourceUser {
			t.Errorf("expected source type User, got %q", source.Source())
		}
	})

	t.Run("path mapping works", func(t *testing.T) {
		skillID := registry.GetSkillIDByPath("/path/to/skill.yaml")
		if skillID != "test-skill" {
			t.Errorf("expected skill ID 'test-skill', got %q", skillID)
		}
	})

	t.Run("returns empty string for unknown path", func(t *testing.T) {
		skillID := registry.GetSkillIDByPath("/unknown/path.yaml")
		if skillID != "" {
			t.Errorf("expected empty string, got %q", skillID)
		}
	})

	t.Run("returns error for nil skill", func(t *testing.T) {
		err := registry.RegisterWithSource(nil, "/path/to/skill.yaml", skill.SourceUser)
		if err == nil {
			t.Error("expected error for nil skill")
		}
	})

	t.Run("returns error for empty file path", func(t *testing.T) {
		err := registry.RegisterWithSource(testSkill, "", skill.SourceUser)
		if err == nil {
			t.Error("expected error for empty file path")
		}
	})
}

func TestUnregisterByPath(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Create and register a test skill
	phase, _ := skill.NewPhase("test-phase", "Test Phase", "template")
	testSkill, _ := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})

	err := registry.RegisterWithSource(testSkill, "/path/to/skill.yaml", skill.SourceUser)
	if err != nil {
		t.Fatalf("failed to register skill: %v", err)
	}

	t.Run("removes skill by path", func(t *testing.T) {
		skillID, found := registry.UnregisterByPath("/path/to/skill.yaml")
		if !found {
			t.Fatal("expected to find skill by path")
		}
		if skillID != "test-skill" {
			t.Errorf("expected skill ID 'test-skill', got %q", skillID)
		}

		// Verify skill is removed
		s := registry.GetSkill("test-skill")
		if s != nil {
			t.Error("expected skill to be removed")
		}

		// Verify source tracking is cleaned up
		source := registry.GetSource("test-skill")
		if source != nil {
			t.Error("expected source to be removed")
		}

		// Verify path mapping is cleaned up
		pathSkillID := registry.GetSkillIDByPath("/path/to/skill.yaml")
		if pathSkillID != "" {
			t.Error("expected path mapping to be removed")
		}
	})

	t.Run("returns false for unknown path", func(t *testing.T) {
		_, found := registry.UnregisterByPath("/unknown/path.yaml")
		if found {
			t.Error("expected not to find skill for unknown path")
		}
	})
}

func TestSourcePriorityOverride(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	// Create test skills
	phase, _ := skill.NewPhase("test-phase", "Test Phase", "template")
	builtInSkill, _ := skill.NewSkill("same-skill", "Built-in Version", "1.0.0", []skill.Phase{*phase})
	userSkill, _ := skill.NewSkill("same-skill", "User Version", "2.0.0", []skill.Phase{*phase})
	projectSkill, _ := skill.NewSkill("same-skill", "Project Version", "3.0.0", []skill.Phase{*phase})

	t.Run("higher priority overrides lower priority", func(t *testing.T) {
		// Register built-in skill first
		err := registry.RegisterWithSource(builtInSkill, "/builtin/skill.yaml", skill.SourceBuiltIn)
		if err != nil {
			t.Fatalf("failed to register built-in skill: %v", err)
		}

		// User skill should override built-in
		err = registry.RegisterWithSource(userSkill, "/user/skill.yaml", skill.SourceUser)
		if err != nil {
			t.Fatalf("failed to register user skill: %v", err)
		}

		s := registry.GetSkill("same-skill")
		if s.Name() != "User Version" {
			t.Errorf("expected User Version, got %q", s.Name())
		}

		// Project skill should override user
		err = registry.RegisterWithSource(projectSkill, "/project/skill.yaml", skill.SourceProject)
		if err != nil {
			t.Fatalf("failed to register project skill: %v", err)
		}

		s = registry.GetSkill("same-skill")
		if s.Name() != "Project Version" {
			t.Errorf("expected Project Version, got %q", s.Name())
		}
	})

	t.Run("lower priority does not override higher priority", func(t *testing.T) {
		registry.Clear()

		// Register project skill first
		err := registry.RegisterWithSource(projectSkill, "/project/skill.yaml", skill.SourceProject)
		if err != nil {
			t.Fatalf("failed to register project skill: %v", err)
		}

		// Try to register user skill - should not override
		err = registry.RegisterWithSource(userSkill, "/user/skill.yaml", skill.SourceUser)
		if err != nil {
			t.Fatalf("failed to register user skill: %v", err)
		}

		// Should still be project version
		s := registry.GetSkill("same-skill")
		if s.Name() != "Project Version" {
			t.Errorf("expected Project Version, got %q", s.Name())
		}

		// Path mapping should still point to project file
		source := registry.GetSource("same-skill")
		if source.FilePath() != "/project/skill.yaml" {
			t.Errorf("expected project path, got %q", source.FilePath())
		}
	})
}

func TestHasSourceTracking(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	t.Run("returns false when empty", func(t *testing.T) {
		if registry.HasSourceTracking() {
			t.Error("expected no source tracking for empty registry")
		}
	})

	t.Run("returns true after RegisterWithSource", func(t *testing.T) {
		phase, _ := skill.NewPhase("test-phase", "Test Phase", "template")
		testSkill, _ := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})

		registry.RegisterWithSource(testSkill, "/path/to/skill.yaml", skill.SourceUser)

		if !registry.HasSourceTracking() {
			t.Error("expected source tracking after RegisterWithSource")
		}
	})

	t.Run("returns false after Register (without source)", func(t *testing.T) {
		registry.Clear()

		phase, _ := skill.NewPhase("test-phase", "Test Phase", "template")
		testSkill, _ := skill.NewSkill("test-skill", "Test Skill", "1.0.0", []skill.Phase{*phase})

		registry.Register(testSkill)

		if registry.HasSourceTracking() {
			t.Error("expected no source tracking after Register")
		}
	})
}

func TestDirectoryAccessors(t *testing.T) {
	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)

	t.Run("ProjectDir returns set value", func(t *testing.T) {
		registry.SetProjectDir("/project/skills")
		if registry.ProjectDir() != "/project/skills" {
			t.Errorf("expected '/project/skills', got %q", registry.ProjectDir())
		}
	})

	t.Run("UserDir returns set value", func(t *testing.T) {
		registry.SetUserDir("/user/skills")
		if registry.UserDir() != "/user/skills" {
			t.Errorf("expected '/user/skills', got %q", registry.UserDir())
		}
	})

	t.Run("BuiltInDir returns set value", func(t *testing.T) {
		registry.SetBuiltInDir("/builtin/skills")
		if registry.BuiltInDir() != "/builtin/skills" {
			t.Errorf("expected '/builtin/skills', got %q", registry.BuiltInDir())
		}
	})
}

func TestReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skillrunner-reload-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write initial skill
	skillYAML := `
id: initial-skill
name: Initial Skill
version: "1.0.0"
phases:
  - id: main
    name: Main
    prompt_template: "Initial: {{.input}}"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "skill.yaml"), []byte(skillYAML), 0644); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}

	loader := infraSkills.NewLoader()
	registry := NewRegistry(loader)
	registry.SetBuiltInDir(tmpDir)
	// Set user dir to non-existent path to isolate test from environment
	registry.SetUserDir("/non/existent/user/dir")

	// Initial load
	if err := registry.LoadAll(); err != nil {
		t.Fatalf("failed initial load: %v", err)
	}
	if registry.Count() != 1 {
		t.Errorf("expected 1 skill after initial load, got %d", registry.Count())
	}

	// Add a new skill file
	newYAML := `
id: new-skill
name: New Skill
version: "1.0.0"
phases:
  - id: main
    name: Main
    prompt_template: "New: {{.input}}"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "new.yaml"), []byte(newYAML), 0644); err != nil {
		t.Fatalf("failed to write new skill: %v", err)
	}

	// Reload
	if err := registry.Reload(); err != nil {
		t.Fatalf("failed to reload: %v", err)
	}

	// Should now have both skills
	if registry.Count() != 2 {
		t.Errorf("expected 2 skills after reload, got %d", registry.Count())
	}
	if registry.GetSkill("new-skill") == nil {
		t.Error("expected to find new-skill after reload")
	}
}
