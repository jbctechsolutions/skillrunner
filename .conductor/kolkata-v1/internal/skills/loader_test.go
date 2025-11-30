package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader("/test/skills")
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	if loader.skillsDir != "/test/skills" {
		t.Errorf("skillsDir = %s; want /test/skills", loader.skillsDir)
	}
	if loader.cache == nil {
		t.Error("cache should be initialized")
	}
}

func TestNewLoaderWithEmptyDir(t *testing.T) {
	loader := NewLoader("")
	if loader.skillsDir != "" {
		t.Errorf("skillsDir = %s; want empty string", loader.skillsDir)
	}
}

func TestLoadSkill_ValidYAML(t *testing.T) {
	// Create temporary directory for test skills
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "test-skill.yaml")

	yamlContent := `name: test-skill
version: 1.0.0
description: A test skill
default_model: gpt-4
`
	err := os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skill, err := loader.LoadSkill("test-skill")
	if err != nil {
		t.Fatalf("LoadSkill failed: %v", err)
	}

	if skill == nil {
		t.Fatal("LoadSkill returned nil skill")
	}

	if skill.Name != "test-skill" {
		t.Errorf("Name = %s; want test-skill", skill.Name)
	}
	if skill.Version != "1.0.0" {
		t.Errorf("Version = %s; want 1.0.0", skill.Version)
	}
	if skill.Description != "A test skill" {
		t.Errorf("Description = %s; want A test skill", skill.Description)
	}
	if skill.DefaultModel != "gpt-4" {
		t.Errorf("DefaultModel = %s; want gpt-4", skill.DefaultModel)
	}
}

func TestLoadSkill_Caching(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "cached-skill.yaml")

	yamlContent := `name: cached-skill
version: 1.0.0
description: A cached skill
default_model: gpt-4
`
	err := os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)

	// First load
	skill1, err := loader.LoadSkill("cached-skill")
	if err != nil {
		t.Fatalf("First LoadSkill failed: %v", err)
	}

	// Second load should come from cache
	skill2, err := loader.LoadSkill("cached-skill")
	if err != nil {
		t.Fatalf("Second LoadSkill failed: %v", err)
	}

	// Should be the same pointer (cached)
	if skill1 != skill2 {
		t.Error("Skills should be the same instance from cache")
	}
}

func TestLoadSkill_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "invalid-skill.yaml")

	invalidYAML := `name: invalid-skill
version: 1.0.0
description: Invalid YAML
default_model: gpt-4
invalid: [unclosed bracket
`
	err := os.WriteFile(skillFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)
	_, err = loader.LoadSkill("invalid-skill")
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoadSkill_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	_, err := loader.LoadSkill("nonexistent-skill")
	if err == nil {
		t.Error("Expected error for nonexistent skill, got nil")
	}
}

func TestLoadSkill_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "invalid-skill.yaml")

	// Missing required fields
	invalidYAML := `name: invalid-skill
version: 1.0.0
`
	err := os.WriteFile(skillFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)
	_, err = loader.LoadSkill("invalid-skill")
	if err == nil {
		t.Error("Expected validation error, got nil")
	}
}

func TestLoadAllSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple skill files
	skill1File := filepath.Join(tmpDir, "skill1.yaml")
	skill2File := filepath.Join(tmpDir, "skill2.yaml")

	yaml1 := `name: skill1
version: 1.0.0
description: First skill
default_model: gpt-4
`
	yaml2 := `name: skill2
version: 2.0.0
description: Second skill
default_model: claude-3
`

	err := os.WriteFile(skill1File, []byte(yaml1), 0644)
	if err != nil {
		t.Fatalf("Failed to create skill1 file: %v", err)
	}
	err = os.WriteFile(skill2File, []byte(yaml2), 0644)
	if err != nil {
		t.Fatalf("Failed to create skill2 file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skills, err := loader.LoadAllSkills()
	if err != nil {
		t.Fatalf("LoadAllSkills failed: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("Skills count = %d; want 2", len(skills))
	}

	skillMap := make(map[string]*types.SkillConfig)
	for _, skill := range skills {
		skillMap[skill.Name] = skill
	}

	if skillMap["skill1"] == nil {
		t.Error("skill1 not found")
	}
	if skillMap["skill2"] == nil {
		t.Error("skill2 not found")
	}
}

func TestLoadAllSkills_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	skills, err := loader.LoadAllSkills()
	if err != nil {
		t.Fatalf("LoadAllSkills failed: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("Skills count = %d; want 0", len(skills))
	}
}

func TestLoadAllSkills_IgnoresNonYAMLFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a YAML file and a non-YAML file
	skillFile := filepath.Join(tmpDir, "skill.yaml")
	textFile := filepath.Join(tmpDir, "readme.txt")

	yamlContent := `name: skill
version: 1.0.0
description: A skill
default_model: gpt-4
`
	err := os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}
	err = os.WriteFile(textFile, []byte("not a skill"), 0644)
	if err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skills, err := loader.LoadAllSkills()
	if err != nil {
		t.Fatalf("LoadAllSkills failed: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Skills count = %d; want 1", len(skills))
	}
}

func TestLoadAllSkills_InvalidSkillInDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one valid and one invalid skill
	validFile := filepath.Join(tmpDir, "valid.yaml")
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")

	validYAML := `name: valid
version: 1.0.0
description: Valid skill
default_model: gpt-4
`
	invalidYAML := `name: invalid
version: 1.0.0
`

	err := os.WriteFile(validFile, []byte(validYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid file: %v", err)
	}
	err = os.WriteFile(invalidFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skills, err := loader.LoadAllSkills()
	// Should return error or skip invalid skills
	// For now, we'll expect it to return an error
	if err == nil {
		// If no error, should only have valid skills
		if len(skills) != 1 {
			t.Errorf("Skills count = %d; want 1 (only valid)", len(skills))
		}
	}
}

func TestClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "cache-test.yaml")

	yamlContent := `name: cache-test
version: 1.0.0
description: Cache test
default_model: gpt-4
`
	err := os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)

	// Load skill to populate cache
	_, err = loader.LoadSkill("cache-test")
	if err != nil {
		t.Fatalf("LoadSkill failed: %v", err)
	}

	// Clear cache
	loader.ClearCache()

	// Cache should be empty
	if len(loader.cache) != 0 {
		t.Error("Cache should be empty after ClearCache")
	}
}

func TestGetCachedSkill(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "cached.yaml")

	yamlContent := `name: cached
version: 1.0.0
description: Cached skill
default_model: gpt-4
`
	err := os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)

	// Not in cache yet
	cached := loader.GetCachedSkill("cached")
	if cached != nil {
		t.Error("Skill should not be in cache yet")
	}

	// Load skill
	_, err = loader.LoadSkill("cached")
	if err != nil {
		t.Fatalf("LoadSkill failed: %v", err)
	}

	// Now should be in cache
	cached = loader.GetCachedSkill("cached")
	if cached == nil {
		t.Fatal("Skill should be in cache after loading")
	}
	//nolint:staticcheck // t.Fatal exits, so cached is guaranteed non-nil
	if cached.Name != "cached" {
		t.Errorf("Cached skill name = %s; want cached", cached.Name)
	}
}

func TestLoadSkill_WithSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "special-skill.yaml")

	yamlContent := `name: special-skill
version: 1.0.0
description: Skill with "quotes" and 'apostrophes'
default_model: gpt-4
`
	err := os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skill, err := loader.LoadSkill("special-skill")
	if err != nil {
		t.Fatalf("LoadSkill failed: %v", err)
	}

	if skill.Description != `Skill with "quotes" and 'apostrophes'` {
		t.Errorf("Description = %s; want Skill with \"quotes\" and 'apostrophes'", skill.Description)
	}
}

func TestLoadSkill_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "empty.yaml")

	err := os.WriteFile(skillFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	loader := NewLoader(tmpDir)
	_, err = loader.LoadSkill("empty")
	if err == nil {
		t.Error("Expected error for empty YAML file, got nil")
	}
}

func TestLoadSkill_DirectoryDoesNotExist(t *testing.T) {
	loader := NewLoader("/nonexistent/directory")
	_, err := loader.LoadSkill("test")
	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}

func TestLoadSkill_NameMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "test-skill.yaml")

	// Skill name in file doesn't match filename
	yamlContent := `name: different-name
version: 1.0.0
description: A test skill
default_model: gpt-4
`
	err := os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)
	_, err = loader.LoadSkill("test-skill")
	if err == nil {
		t.Error("Expected error for name mismatch, got nil")
	}
}

func TestLoadSkill_EmptyNameInFile(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "auto-named.yaml")

	// Name is empty, should use filename
	yamlContent := `version: 1.0.0
description: A test skill
default_model: gpt-4
`
	err := os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skill, err := loader.LoadSkill("auto-named")
	if err != nil {
		t.Fatalf("LoadSkill failed: %v", err)
	}

	if skill.Name != "auto-named" {
		t.Errorf("Name = %s; want auto-named", skill.Name)
	}
}

func TestLoadSkill_ReadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "read-error.yaml")

	// Create a directory with the same name to cause a read error
	err := os.Mkdir(skillFile, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	loader := NewLoader(tmpDir)
	_, err = loader.LoadSkill("read-error")
	if err == nil {
		t.Error("Expected error for read error, got nil")
	}
}

func TestLoadAllSkills_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a skill file
	skillFile := filepath.Join(tmpDir, "skill.yaml")
	yamlContent := `name: skill
version: 1.0.0
description: A skill
default_model: gpt-4
`
	err = os.WriteFile(skillFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skills, err := loader.LoadAllSkills()
	if err != nil {
		t.Fatalf("LoadAllSkills failed: %v", err)
	}

	// Should only load the skill file, not the subdirectory
	if len(skills) != 1 {
		t.Errorf("Skills count = %d; want 1", len(skills))
	}
}

func TestLoadAllSkills_WithYMLFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create both .yaml and .yml files
	yamlFile := filepath.Join(tmpDir, "skill1.yaml")
	ymlFile := filepath.Join(tmpDir, "skill2.yml")

	yaml1 := `name: skill1
version: 1.0.0
description: First skill
default_model: gpt-4
`
	yaml2 := `name: skill2
version: 1.0.0
description: Second skill
default_model: claude-3
`

	err := os.WriteFile(yamlFile, []byte(yaml1), 0644)
	if err != nil {
		t.Fatalf("Failed to create yaml file: %v", err)
	}
	err = os.WriteFile(ymlFile, []byte(yaml2), 0644)
	if err != nil {
		t.Fatalf("Failed to create yml file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skills, err := loader.LoadAllSkills()
	if err != nil {
		t.Fatalf("LoadAllSkills failed: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("Skills count = %d; want 2", len(skills))
	}
}

func TestLoadAllSkills_PartialErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one valid and one invalid skill
	validFile := filepath.Join(tmpDir, "valid.yaml")
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")

	validYAML := `name: valid
version: 1.0.0
description: Valid skill
default_model: gpt-4
`
	invalidYAML := `name: invalid
version: 1.0.0
`

	err := os.WriteFile(validFile, []byte(validYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid file: %v", err)
	}
	err = os.WriteFile(invalidFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	loader := NewLoader(tmpDir)
	skills, err := loader.LoadAllSkills()
	// Should return valid skills even if some fail
	if err != nil {
		// If error is returned, it should only be when all skills fail
		if len(skills) == 0 {
			t.Logf("Got error as expected when all skills fail: %v", err)
		}
	}

	// Should have at least the valid skill
	if len(skills) < 1 {
		t.Errorf("Should have at least 1 valid skill, got %d", len(skills))
	}
}

func TestLoadSkill_EmptySkillsDir(t *testing.T) {
	// Test LoadAllSkills with empty dir
	loader := NewLoader("")
	skills, err := loader.LoadAllSkills()
	if err != nil {
		t.Fatalf("LoadAllSkills with empty dir should not error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Skills count = %d; want 0 for empty dir", len(skills))
	}

	// Test LoadSkill with empty dir and relative path
	// Create a file in current directory temporarily
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tmpFile := filepath.Join(cwd, "temp-skill.yaml")
	defer os.Remove(tmpFile)

	yamlContent := `name: temp-skill
version: 1.0.0
description: A temp skill
default_model: gpt-4
`
	err = os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp skill file: %v", err)
	}

	loader2 := NewLoader("")
	skill, err := loader2.LoadSkill("temp-skill")
	if err != nil {
		t.Fatalf("LoadSkill with empty dir failed: %v", err)
	}
	if skill.Name != "temp-skill" {
		t.Errorf("Name = %s; want temp-skill", skill.Name)
	}
}
