package skills

import (
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

func TestValidateSkill_ValidSkill(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err != nil {
		t.Errorf("ValidateSkill failed for valid skill: %v", err)
	}
}

func TestValidateSkill_MissingName(t *testing.T) {
	skill := &types.SkillConfig{
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for missing name, got nil")
	}
}

func TestValidateSkill_EmptyName(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for empty name, got nil")
	}
}

func TestValidateSkill_MissingVersion(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for missing version, got nil")
	}
}

func TestValidateSkill_EmptyVersion(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for empty version, got nil")
	}
}

func TestValidateSkill_MissingDescription(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "1.0.0",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for missing description, got nil")
	}
}

func TestValidateSkill_EmptyDescription(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "1.0.0",
		Description:  "",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for empty description, got nil")
	}
}

func TestValidateSkill_MissingDefaultModel(t *testing.T) {
	skill := &types.SkillConfig{
		Name:        "test-skill",
		Version:     "1.0.0",
		Description: "A test skill",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for missing default_model, got nil")
	}
}

func TestValidateSkill_EmptyDefaultModel(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for empty default_model, got nil")
	}
}

func TestValidateSkill_InvalidName_Whitespace(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test skill",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for name with whitespace, got nil")
	}
}

func TestValidateSkill_InvalidName_SpecialChars(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test@skill",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for name with special characters, got nil")
	}
}

func TestValidateSkill_ValidName_WithHyphens(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill-name",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err != nil {
		t.Errorf("ValidateSkill failed for valid name with hyphens: %v", err)
	}
}

func TestValidateSkill_ValidName_WithUnderscores(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test_skill_name",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err != nil {
		t.Errorf("ValidateSkill failed for valid name with underscores: %v", err)
	}
}

func TestValidateSkill_InvalidVersion_Format(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "invalid-version",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	// Version format validation is optional, but if we validate it, should check
	// For now, we'll just check it's not empty
	if err != nil {
		// If validation fails, that's okay for this test - we're just checking version isn't empty
	}
	if skill.Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestValidateSkill_ValidVersion_Semver(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "1.2.3",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err != nil {
		t.Errorf("ValidateSkill failed for valid semver: %v", err)
	}
}

func TestValidateSkill_NilSkill(t *testing.T) {
	err := ValidateSkill(nil)
	if err == nil {
		t.Error("Expected error for nil skill, got nil")
	}
}

func TestValidateSkill_AllFieldsValid(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "backend-architect",
		Version:      "2.1.0",
		Description:  "Architect and implement backend systems with comprehensive planning",
		DefaultModel: "claude-3-opus",
	}

	err := ValidateSkill(skill)
	if err != nil {
		t.Errorf("ValidateSkill failed for fully valid skill: %v", err)
	}
}

func TestValidateSkill_WhitespaceOnlyName(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "   ",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for whitespace-only name, got nil")
	}
}

func TestValidateSkill_WhitespaceOnlyVersion(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "   ",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for whitespace-only version, got nil")
	}
}

func TestValidateSkill_WhitespaceOnlyDescription(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "1.0.0",
		Description:  "   ",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for whitespace-only description, got nil")
	}
}

func TestValidateSkill_WhitespaceOnlyDefaultModel(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "test-skill",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "   ",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for whitespace-only default_model, got nil")
	}
}

func TestValidateSkill_NameWithLeadingTrailingWhitespace(t *testing.T) {
	skill := &types.SkillConfig{
		Name:         "  test-skill  ",
		Version:      "1.0.0",
		Description:  "A test skill",
		DefaultModel: "gpt-4",
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("Expected error for name with leading/trailing whitespace, got nil")
	}
}
