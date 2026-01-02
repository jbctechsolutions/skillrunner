package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Error("NewLoader() returned nil")
	}
}

func TestLoadSkill_ValidFile(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create a valid skill YAML file
	validYAML := `
id: test-skill
name: Test Skill
version: "1.0.0"
description: A test skill for unit tests
phases:
  - id: analyze
    name: Analysis Phase
    prompt_template: |
      Analyze the following input: {{.input}}
    routing_profile: balanced
    max_tokens: 2048
    temperature: 0.5
  - id: generate
    name: Generation Phase
    prompt_template: |
      Based on the analysis: {{.analyze_output}}
      Generate a response.
    routing_profile: premium
    depends_on:
      - analyze
    max_tokens: 4096
    temperature: 0.7
routing:
  default_profile: balanced
  generation_model: gpt-4
  fallback_model: gpt-3.5-turbo
  max_context_tokens: 8192
metadata:
  author: test
  tags:
    - testing
    - example
`
	skillPath := filepath.Join(tmpDir, "test-skill.yaml")
	if err := os.WriteFile(skillPath, []byte(validYAML), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	s, err := loader.LoadSkill(skillPath)
	if err != nil {
		t.Fatalf("LoadSkill() error = %v", err)
	}

	// Verify skill properties
	if s.ID() != "test-skill" {
		t.Errorf("ID() = %q, want %q", s.ID(), "test-skill")
	}
	if s.Name() != "Test Skill" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Test Skill")
	}
	if s.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want %q", s.Version(), "1.0.0")
	}
	if s.Description() != "A test skill for unit tests" {
		t.Errorf("Description() = %q, want %q", s.Description(), "A test skill for unit tests")
	}

	// Verify phases
	phases := s.Phases()
	if len(phases) != 2 {
		t.Fatalf("len(Phases()) = %d, want 2", len(phases))
	}

	if phases[0].ID != "analyze" {
		t.Errorf("phases[0].ID = %q, want %q", phases[0].ID, "analyze")
	}
	if phases[1].ID != "generate" {
		t.Errorf("phases[1].ID = %q, want %q", phases[1].ID, "generate")
	}
	if len(phases[1].DependsOn) != 1 || phases[1].DependsOn[0] != "analyze" {
		t.Errorf("phases[1].DependsOn = %v, want [analyze]", phases[1].DependsOn)
	}

	// Verify routing
	routing := s.Routing()
	if routing.DefaultProfile != "balanced" {
		t.Errorf("routing.DefaultProfile = %q, want %q", routing.DefaultProfile, "balanced")
	}
	if routing.GenerationModel != "gpt-4" {
		t.Errorf("routing.GenerationModel = %q, want %q", routing.GenerationModel, "gpt-4")
	}
	if routing.FallbackModel != "gpt-3.5-turbo" {
		t.Errorf("routing.FallbackModel = %q, want %q", routing.FallbackModel, "gpt-3.5-turbo")
	}
	if routing.MaxContextTokens != 8192 {
		t.Errorf("routing.MaxContextTokens = %d, want %d", routing.MaxContextTokens, 8192)
	}

	// Verify metadata
	meta := s.Metadata()
	if meta["author"] != "test" {
		t.Errorf("metadata[author] = %v, want %q", meta["author"], "test")
	}
}

func TestLoadSkill_MinimalValidFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Minimal valid skill
	minimalYAML := `
id: minimal-skill
name: Minimal Skill
phases:
  - id: main
    name: Main Phase
    prompt_template: Process this input
`
	skillPath := filepath.Join(tmpDir, "minimal.yaml")
	if err := os.WriteFile(skillPath, []byte(minimalYAML), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	s, err := loader.LoadSkill(skillPath)
	if err != nil {
		t.Fatalf("LoadSkill() error = %v", err)
	}

	if s.ID() != "minimal-skill" {
		t.Errorf("ID() = %q, want %q", s.ID(), "minimal-skill")
	}
	if len(s.Phases()) != 1 {
		t.Errorf("len(Phases()) = %d, want 1", len(s.Phases()))
	}

	// Defaults should be applied
	phase := s.Phases()[0]
	if phase.RoutingProfile != skill.DefaultRoutingProfile {
		t.Errorf("phase.RoutingProfile = %q, want %q", phase.RoutingProfile, skill.DefaultRoutingProfile)
	}
	if phase.MaxTokens != skill.DefaultMaxTokens {
		t.Errorf("phase.MaxTokens = %d, want %d", phase.MaxTokens, skill.DefaultMaxTokens)
	}
}

func TestLoadSkill_YMLExtension(t *testing.T) {
	tmpDir := t.TempDir()

	yaml := `
id: yml-skill
name: YML Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
`
	// Test .yml extension
	skillPath := filepath.Join(tmpDir, "skill.yml")
	if err := os.WriteFile(skillPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	s, err := loader.LoadSkill(skillPath)
	if err != nil {
		t.Fatalf("LoadSkill() with .yml extension error = %v", err)
	}
	if s.ID() != "yml-skill" {
		t.Errorf("ID() = %q, want %q", s.ID(), "yml-skill")
	}
}

func TestLoadSkill_InvalidPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: "invalid file path",
		},
		{
			name:    "non-yaml extension",
			path:    "/path/to/skill.json",
			wantErr: "file is not a YAML file",
		},
		{
			name:    "txt extension",
			path:    "/path/to/skill.txt",
			wantErr: "file is not a YAML file",
		},
	}

	loader := NewLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loader.LoadSkill(tt.path)
			if err == nil {
				t.Errorf("LoadSkill() error = nil, want error containing %q", tt.wantErr)
				return
			}
			if !containsString(err.Error(), tt.wantErr) {
				t.Errorf("LoadSkill() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadSkill_FileNotFound(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadSkill("/nonexistent/path/skill.yaml")
	if err == nil {
		t.Error("LoadSkill() error = nil, want error")
	}
	if !containsString(err.Error(), "failed to read file") {
		t.Errorf("LoadSkill() error = %v, want error about reading file", err)
	}
}

func TestLoadSkill_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyPath := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	_, err := loader.LoadSkill(emptyPath)
	if err == nil {
		t.Error("LoadSkill() error = nil, want error")
	}
	if !containsString(err.Error(), "file is empty") {
		t.Errorf("LoadSkill() error = %v, want error about empty file", err)
	}
}

func TestLoadSkill_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	invalidYAML := `
id: test
name: [invalid yaml
  - broken structure
`
	if err := os.WriteFile(invalidPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	_, err := loader.LoadSkill(invalidPath)
	if err == nil {
		t.Error("LoadSkill() error = nil, want error")
	}
	if !containsString(err.Error(), "failed to parse YAML") {
		t.Errorf("LoadSkill() error = %v, want error about YAML parsing", err)
	}
}

func TestLoadSkill_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "missing skill id",
			yaml: `
name: Test Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
`,
			wantErr: "skill id is required",
		},
		{
			name: "missing skill name",
			yaml: `
id: test-skill
phases:
  - id: main
    name: Main
    prompt_template: Test
`,
			wantErr: "skill name is required",
		},
		{
			name: "no phases",
			yaml: `
id: test-skill
name: Test Skill
phases: []
`,
			wantErr: "at least one phase is required",
		},
		{
			name: "missing phase id",
			yaml: `
id: test-skill
name: Test Skill
phases:
  - name: Main
    prompt_template: Test
`,
			wantErr: "id is required",
		},
		{
			name: "missing phase name",
			yaml: `
id: test-skill
name: Test Skill
phases:
  - id: main
    prompt_template: Test
`,
			wantErr: "name is required",
		},
		{
			name: "missing prompt template",
			yaml: `
id: test-skill
name: Test Skill
phases:
  - id: main
    name: Main
`,
			wantErr: "prompt_template is required",
		},
		{
			name: "invalid routing profile",
			yaml: `
id: test-skill
name: Test Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
    routing_profile: invalid
`,
			wantErr: "invalid routing_profile",
		},
		{
			name: "unknown dependency",
			yaml: `
id: test-skill
name: Test Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
    depends_on:
      - nonexistent
`,
			wantErr: "unknown dependency",
		},
		{
			name: "duplicate phase id",
			yaml: `
id: test-skill
name: Test Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
  - id: main
    name: Another Main
    prompt_template: Test 2
`,
			wantErr: "duplicate phase id",
		},
	}

	loader := NewLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			skillPath := filepath.Join(tmpDir, "skill.yaml")
			if err := os.WriteFile(skillPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			_, err := loader.LoadSkill(skillPath)
			if err == nil {
				t.Errorf("LoadSkill() error = nil, want error containing %q", tt.wantErr)
				return
			}
			if !containsString(err.Error(), tt.wantErr) {
				t.Errorf("LoadSkill() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadSkillsDir_ValidDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple skill files
	skill1 := `
id: skill-1
name: Skill One
phases:
  - id: main
    name: Main
    prompt_template: Process input
`
	skill2 := `
id: skill-2
name: Skill Two
phases:
  - id: analyze
    name: Analyze
    prompt_template: Analyze input
  - id: output
    name: Output
    prompt_template: Generate output
    depends_on:
      - analyze
`

	if err := os.WriteFile(filepath.Join(tmpDir, "skill1.yaml"), []byte(skill1), 0644); err != nil {
		t.Fatalf("failed to write skill1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skill2.yml"), []byte(skill2), 0644); err != nil {
		t.Fatalf("failed to write skill2: %v", err)
	}

	loader := NewLoader()
	skills, err := loader.LoadSkillsDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillsDir() error = %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("len(skills) = %d, want 2", len(skills))
	}

	if _, ok := skills["skill-1"]; !ok {
		t.Error("skills[skill-1] not found")
	}
	if _, ok := skills["skill-2"]; !ok {
		t.Error("skills[skill-2] not found")
	}
}

func TestLoadSkillsDir_Subdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	skill1 := `
id: root-skill
name: Root Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
`
	skill2 := `
id: nested-skill
name: Nested Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
`

	if err := os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(skill1), 0644); err != nil {
		t.Fatalf("failed to write root skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.yaml"), []byte(skill2), 0644); err != nil {
		t.Fatalf("failed to write nested skill: %v", err)
	}

	loader := NewLoader()
	skills, err := loader.LoadSkillsDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillsDir() error = %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("len(skills) = %d, want 2", len(skills))
	}

	if _, ok := skills["root-skill"]; !ok {
		t.Error("skills[root-skill] not found")
	}
	if _, ok := skills["nested-skill"]; !ok {
		t.Error("skills[nested-skill] not found")
	}
}

func TestLoadSkillsDir_IgnoresNonYAMLFiles(t *testing.T) {
	tmpDir := t.TempDir()

	skill := `
id: only-skill
name: Only Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
`

	if err := os.WriteFile(filepath.Join(tmpDir, "skill.yaml"), []byte(skill), 0644); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Readme"), 0644); err != nil {
		t.Fatalf("failed to write readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	loader := NewLoader()
	skills, err := loader.LoadSkillsDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillsDir() error = %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("len(skills) = %d, want 1", len(skills))
	}
	if _, ok := skills["only-skill"]; !ok {
		t.Error("skills[only-skill] not found")
	}
}

func TestLoadSkillsDir_DuplicateIDs(t *testing.T) {
	tmpDir := t.TempDir()

	// Two skills with the same ID
	skill := `
id: duplicate-id
name: Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
`

	if err := os.WriteFile(filepath.Join(tmpDir, "skill1.yaml"), []byte(skill), 0644); err != nil {
		t.Fatalf("failed to write skill1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skill2.yaml"), []byte(skill), 0644); err != nil {
		t.Fatalf("failed to write skill2: %v", err)
	}

	loader := NewLoader()
	skills, err := loader.LoadSkillsDir(tmpDir)

	// Should still return the first skill but report an error
	if len(skills) != 1 {
		t.Errorf("len(skills) = %d, want 1", len(skills))
	}

	if err == nil {
		t.Error("LoadSkillsDir() error = nil, want error about duplicate IDs")
		return
	}
	if !containsString(err.Error(), "duplicate skill ID") {
		t.Errorf("LoadSkillsDir() error = %v, want error about duplicate IDs", err)
	}
}

func TestLoadSkillsDir_InvalidDirectory(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadSkillsDir("/nonexistent/directory")
	if err == nil {
		t.Error("LoadSkillsDir() error = nil, want error")
	}
	if !containsString(err.Error(), "failed to access directory") {
		t.Errorf("LoadSkillsDir() error = %v, want error about directory access", err)
	}
}

func TestLoadSkillsDir_FileNotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loader := NewLoader()
	_, err := loader.LoadSkillsDir(filePath)
	if err == nil {
		t.Error("LoadSkillsDir() error = nil, want error")
	}
	if !containsString(err.Error(), "is not a directory") {
		t.Errorf("LoadSkillsDir() error = %v, want error about not being a directory", err)
	}
}

func TestLoadSkillsDir_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	loader := NewLoader()
	skills, err := loader.LoadSkillsDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadSkillsDir() error = %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("len(skills) = %d, want 0", len(skills))
	}
}

func TestLoadSkillsDir_PartialLoadWithErrors(t *testing.T) {
	tmpDir := t.TempDir()

	validSkill := `
id: valid-skill
name: Valid Skill
phases:
  - id: main
    name: Main
    prompt_template: Test
`
	invalidSkill := `
id: invalid-skill
phases: []
`

	if err := os.WriteFile(filepath.Join(tmpDir, "valid.yaml"), []byte(validSkill), 0644); err != nil {
		t.Fatalf("failed to write valid skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidSkill), 0644); err != nil {
		t.Fatalf("failed to write invalid skill: %v", err)
	}

	loader := NewLoader()
	skills, err := loader.LoadSkillsDir(tmpDir)

	// Should return the valid skill
	if len(skills) != 1 {
		t.Errorf("len(skills) = %d, want 1", len(skills))
	}
	if _, ok := skills["valid-skill"]; !ok {
		t.Error("skills[valid-skill] not found")
	}

	// Should also report the error
	if err == nil {
		t.Error("LoadSkillsDir() error = nil, want error about invalid skill")
	}
}

func TestLoadSkill_CyclicDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	cyclicYAML := `
id: cyclic-skill
name: Cyclic Skill
phases:
  - id: phase-a
    name: Phase A
    prompt_template: Test A
    depends_on:
      - phase-b
  - id: phase-b
    name: Phase B
    prompt_template: Test B
    depends_on:
      - phase-a
`
	skillPath := filepath.Join(tmpDir, "cyclic.yaml")
	if err := os.WriteFile(skillPath, []byte(cyclicYAML), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	_, err := loader.LoadSkill(skillPath)
	if err == nil {
		t.Error("LoadSkill() error = nil, want error about cyclic dependencies")
		return
	}
	if !containsString(err.Error(), "cycle") {
		t.Errorf("LoadSkill() error = %v, want error about cycle", err)
	}
}

func TestIsValidRoutingProfile(t *testing.T) {
	tests := []struct {
		profile string
		valid   bool
	}{
		{"cheap", true},
		{"balanced", true},
		{"premium", true},
		{"Cheap", false},
		{"BALANCED", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			if got := isValidRoutingProfile(tt.profile); got != tt.valid {
				t.Errorf("isValidRoutingProfile(%q) = %v, want %v", tt.profile, got, tt.valid)
			}
		})
	}
}

func TestLoadSkill_DemoDocGenSkill(t *testing.T) {
	// Test that the doc-gen.yaml demo skill can be loaded successfully
	loader := NewLoader()
	s, err := loader.LoadSkill("../../../skills/doc-gen.yaml")
	if err != nil {
		t.Fatalf("LoadSkill(doc-gen.yaml) error = %v", err)
	}

	// Verify expected values
	if s.ID() != "doc-gen" {
		t.Errorf("ID() = %q, want %q", s.ID(), "doc-gen")
	}
	if s.Name() != "Documentation Generator" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Documentation Generator")
	}
	if s.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want %q", s.Version(), "1.0.0")
	}
	if s.Description() == "" {
		t.Error("Description() is empty, expected a description")
	}

	// Verify phases - should have analyze and generate
	phases := s.Phases()
	if len(phases) != 2 {
		t.Fatalf("len(Phases()) = %d, want 2", len(phases))
	}

	// First phase: analyze
	if phases[0].ID != "analyze" {
		t.Errorf("phases[0].ID = %q, want %q", phases[0].ID, "analyze")
	}
	if phases[0].Name != "Code Structure Analysis" {
		t.Errorf("phases[0].Name = %q, want %q", phases[0].Name, "Code Structure Analysis")
	}
	if phases[0].RoutingProfile != "cheap" {
		t.Errorf("phases[0].RoutingProfile = %q, want %q", phases[0].RoutingProfile, "cheap")
	}

	// Second phase: generate (depends on analyze)
	if phases[1].ID != "generate" {
		t.Errorf("phases[1].ID = %q, want %q", phases[1].ID, "generate")
	}
	if phases[1].Name != "Documentation Generation" {
		t.Errorf("phases[1].Name = %q, want %q", phases[1].Name, "Documentation Generation")
	}
	if phases[1].RoutingProfile != "cheap" {
		t.Errorf("phases[1].RoutingProfile = %q, want %q", phases[1].RoutingProfile, "cheap")
	}
	if len(phases[1].DependsOn) != 1 || phases[1].DependsOn[0] != "analyze" {
		t.Errorf("phases[1].DependsOn = %v, want [analyze]", phases[1].DependsOn)
	}

	// Verify routing is cost-aware (cheap profile)
	routing := s.Routing()
	if routing.DefaultProfile != "cheap" {
		t.Errorf("routing.DefaultProfile = %q, want %q", routing.DefaultProfile, "cheap")
	}
	if routing.MaxContextTokens != 8192 {
		t.Errorf("routing.MaxContextTokens = %d, want %d", routing.MaxContextTokens, 8192)
	}

	// Verify metadata
	meta := s.Metadata()
	if meta["category"] != "documentation" {
		t.Errorf("metadata[category] = %v, want %q", meta["category"], "documentation")
	}
	if meta["cost_tier"] != "low" {
		t.Errorf("metadata[cost_tier] = %v, want %q", meta["cost_tier"], "low")
	}
}

// containsString checks if s contains substr (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && contains(s, substr)))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
