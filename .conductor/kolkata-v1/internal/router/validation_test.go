package router

import (
	"strings"
	"testing"
)

func TestNewModelValidator(t *testing.T) {
	allowedModels := []string{"gpt-4", "claude-3"}
	validator := NewModelValidator(allowedModels)

	if validator.allowedModels == nil {
		t.Error("allowedModels should not be nil")
	}

	if !validator.allowedModels["gpt-4"] {
		t.Error("gpt-4 should be in allowed models")
	}

	if !validator.allowedModels["claude-3"] {
		t.Error("claude-3 should be in allowed models")
	}

	if validator.modelPattern == nil {
		t.Error("modelPattern should not be nil")
	}
}

func TestNewModelValidator_EmptyList(t *testing.T) {
	validator := NewModelValidator([]string{})
	if len(validator.allowedModels) != 0 {
		t.Error("allowedModels should be empty")
	}
}

func TestDefaultModelValidator(t *testing.T) {
	validator := DefaultModelValidator()

	if validator.allowedModels == nil {
		t.Error("allowedModels should not be nil")
	}

	if len(validator.allowedModels) == 0 {
		t.Error("Default validator should have allowed models")
	}

	// Check some expected models
	expectedModels := []string{"gpt-4", "claude-3", "gpt-3.5-turbo"}
	for _, model := range expectedModels {
		if !validator.allowedModels[strings.ToLower(model)] {
			t.Errorf("Expected model '%s' not found in default validator", model)
		}
	}
}

func TestValidate_EmptyModel(t *testing.T) {
	validator := NewModelValidator([]string{})
	err := validator.Validate("")
	if err == nil {
		t.Error("Validate should return error for empty model")
	}
}

func TestValidate_InvalidFormat(t *testing.T) {
	validator := NewModelValidator([]string{})

	invalidModels := []string{
		"model with spaces",
		"model@invalid",
		"model#invalid",
		"model$invalid",
		"model%invalid",
		"model&invalid",
		"model*invalid",
		"model(invalid)",
		"model[invalid]",
		"model{invalid}",
	}

	for _, model := range invalidModels {
		err := validator.Validate(model)
		if err == nil {
			t.Errorf("Validate should return error for invalid format: %s", model)
		}
	}
}

func TestValidate_ValidFormat(t *testing.T) {
	validator := NewModelValidator([]string{})

	validModels := []string{
		"gpt-4",
		"claude-3",
		"gpt-3.5-turbo",
		"openai/gpt-4",
		"model_name",
		"model.name",
		"model_name-v1",
		"a",
		"123",
		"model-123",
	}

	for _, model := range validModels {
		err := validator.Validate(model)
		if err != nil {
			t.Errorf("Validate should not return error for valid format '%s': %v", model, err)
		}
	}
}

func TestValidate_NotInAllowedList(t *testing.T) {
	allowedModels := []string{"gpt-4", "claude-3"}
	validator := NewModelValidator(allowedModels)

	err := validator.Validate("gpt-3.5-turbo")
	if err == nil {
		t.Error("Validate should return error for model not in allowed list")
	}
}

func TestValidate_InAllowedList(t *testing.T) {
	allowedModels := []string{"gpt-4", "claude-3", "gpt-3.5-turbo"}
	validator := NewModelValidator(allowedModels)

	err := validator.Validate("gpt-4")
	if err != nil {
		t.Errorf("Validate should not return error for allowed model: %v", err)
	}

	err = validator.Validate("GPT-4") // Case insensitive
	if err != nil {
		t.Errorf("Validate should be case insensitive: %v", err)
	}

	err = validator.Validate("Claude-3")
	if err != nil {
		t.Errorf("Validate should be case insensitive: %v", err)
	}
}

func TestValidate_EmptyAllowedList(t *testing.T) {
	validator := NewModelValidator([]string{})

	// With empty allowed list, any valid format should pass
	err := validator.Validate("gpt-4")
	if err != nil {
		t.Errorf("Validate should not return error when allowed list is empty: %v", err)
	}
}

func TestValidateWithSuggestions_ValidModel(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "claude-3"})

	err, suggestions := validator.ValidateWithSuggestions("gpt-4")
	if err != nil {
		t.Errorf("ValidateWithSuggestions should not return error for valid model: %v", err)
	}
	if suggestions != nil {
		t.Error("ValidateWithSuggestions should not return suggestions for valid model")
	}
}

func TestValidateWithSuggestions_InvalidModel(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "gpt-3.5-turbo", "claude-3"})

	err, suggestions := validator.ValidateWithSuggestions("gpt-5")
	if err == nil {
		t.Error("ValidateWithSuggestions should return error for invalid model")
	}
	if suggestions == nil {
		t.Error("ValidateWithSuggestions should return suggestions for invalid model")
	}
}

func TestValidateWithSuggestions_SimilarModels(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "gpt-3.5-turbo", "claude-3"})

	err, suggestions := validator.ValidateWithSuggestions("gpt")
	if err == nil {
		t.Error("ValidateWithSuggestions should return error")
	}
	if len(suggestions) == 0 {
		t.Error("ValidateWithSuggestions should return suggestions for similar models")
	}
}

func TestIsValid(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4"})

	if !validator.IsValid("gpt-4") {
		t.Error("IsValid should return true for valid model")
	}

	if validator.IsValid("invalid-model") {
		t.Error("IsValid should return false for invalid model")
	}

	if validator.IsValid("") {
		t.Error("IsValid should return false for empty model")
	}
}

func TestAddAllowedModel(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4"})

	validator.AddAllowedModel("claude-3")
	if !validator.allowedModels["claude-3"] {
		t.Error("AddAllowedModel should add model to allowed list")
	}

	validator.AddAllowedModel("GPT-4-TURBO")
	if !validator.allowedModels["gpt-4-turbo"] {
		t.Error("AddAllowedModel should be case insensitive")
	}

	validator.AddAllowedModel("")
	if validator.allowedModels[""] {
		t.Error("AddAllowedModel should not add empty model")
	}
}

func TestRemoveAllowedModel(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "claude-3"})

	validator.RemoveAllowedModel("gpt-4")
	if validator.allowedModels["gpt-4"] {
		t.Error("RemoveAllowedModel should remove model from allowed list")
	}

	if !validator.allowedModels["claude-3"] {
		t.Error("RemoveAllowedModel should not remove other models")
	}

	validator.RemoveAllowedModel("GPT-4")
	// Should handle case insensitive removal gracefully
}

func TestGetAllowedModels(t *testing.T) {
	allowedModels := []string{"gpt-4", "claude-3", "gpt-3.5-turbo"}
	validator := NewModelValidator(allowedModels)

	models := validator.GetAllowedModels()
	if len(models) != len(allowedModels) {
		t.Errorf("GetAllowedModels returned %d models; want %d", len(models), len(allowedModels))
	}

	// Check that all models are present (order may vary)
	modelMap := make(map[string]bool)
	for _, model := range models {
		modelMap[model] = true
	}

	for _, expected := range allowedModels {
		if !modelMap[strings.ToLower(expected)] {
			t.Errorf("Expected model '%s' not found in GetAllowedModels", expected)
		}
	}
}

func TestValidateModelList(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "claude-3"})

	models := []string{"gpt-4", "invalid-model", "claude-3", "another-invalid"}
	errors := validator.ValidateModelList(models)

	if len(errors) != 2 {
		t.Errorf("ValidateModelList returned %d errors; want 2", len(errors))
	}
}

func TestValidateModelList_AllValid(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "claude-3"})

	models := []string{"gpt-4", "claude-3"}
	errors := validator.ValidateModelList(models)

	if len(errors) != 0 {
		t.Errorf("ValidateModelList returned %d errors; want 0", len(errors))
	}
}

func TestValidateModelList_EmptyList(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4"})

	errors := validator.ValidateModelList([]string{})
	if len(errors) != 0 {
		t.Errorf("ValidateModelList returned %d errors for empty list; want 0", len(errors))
	}
}

func TestValidateModelMap(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "claude-3"})

	modelMap := map[string]string{
		"step1": "gpt-4",
		"step2": "invalid-model",
		"step3": "claude-3",
	}

	errors := validator.ValidateModelMap(modelMap)
	if len(errors) != 1 {
		t.Errorf("ValidateModelMap returned %d errors; want 1", len(errors))
	}
}

func TestValidateModelMap_AllValid(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "claude-3"})

	modelMap := map[string]string{
		"step1": "gpt-4",
		"step2": "claude-3",
	}

	errors := validator.ValidateModelMap(modelMap)
	if len(errors) != 0 {
		t.Errorf("ValidateModelMap returned %d errors; want 0", len(errors))
	}
}

func TestValidateModelMap_EmptyMap(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4"})

	errors := validator.ValidateModelMap(map[string]string{})
	if len(errors) != 0 {
		t.Errorf("ValidateModelMap returned %d errors for empty map; want 0", len(errors))
	}
}

func TestGenerateSuggestions_SubstringMatch(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "gpt-3.5-turbo", "claude-3"})

	suggestions := validator.generateSuggestions("gpt")
	if len(suggestions) == 0 {
		t.Error("generateSuggestions should return suggestions for substring match")
	}
}

func TestGenerateSuggestions_PrefixMatch(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "gpt-3.5-turbo", "claude-3"})

	suggestions := validator.generateSuggestions("gpt-")
	if len(suggestions) == 0 {
		t.Error("generateSuggestions should return suggestions for prefix match")
	}
}

func TestGenerateSuggestions_NoMatch(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "claude-3"})

	suggestions := validator.generateSuggestions("nonexistent-model-xyz")
	// Should return empty or limited suggestions
	if len(suggestions) > 5 {
		t.Error("generateSuggestions should limit suggestions to 5")
	}
}

func TestGenerateSuggestions_Limit(t *testing.T) {
	validator := NewModelValidator([]string{
		"gpt-4", "gpt-3.5-turbo", "gpt-4-turbo", "gpt-4o", "gpt-4o-mini",
		"gpt-4o-preview", "gpt-4o-2024", "gpt-4o-2025",
	})

	suggestions := validator.generateSuggestions("gpt")
	if len(suggestions) > 5 {
		t.Errorf("generateSuggestions returned %d suggestions; should limit to 5", len(suggestions))
	}
}

func TestValidate_SpecialCharacters(t *testing.T) {
	validator := NewModelValidator([]string{})

	// Test various special characters
	testCases := []struct {
		model   string
		allowed bool
	}{
		{"model-name", true},
		{"model_name", true},
		{"model.name", true},
		{"model/name", true},
		{"model_name-v1.0", true},
		{"model name", false},
		{"model@name", false},
		{"model#name", false},
		{"model$name", false},
	}

	for _, tc := range testCases {
		err := validator.Validate(tc.model)
		if tc.allowed && err != nil {
			t.Errorf("Model '%s' should be allowed but got error: %v", tc.model, err)
		}
		if !tc.allowed && err == nil {
			t.Errorf("Model '%s' should not be allowed", tc.model)
		}
	}
}

func TestValidate_CaseInsensitive(t *testing.T) {
	validator := NewModelValidator([]string{"gpt-4", "Claude-3"})

	err := validator.Validate("GPT-4")
	if err != nil {
		t.Errorf("Validate should be case insensitive: %v", err)
	}

	err = validator.Validate("CLAUDE-3")
	if err != nil {
		t.Errorf("Validate should be case insensitive: %v", err)
	}
}

func TestNewModelValidator_CaseInsensitiveStorage(t *testing.T) {
	allowedModels := []string{"GPT-4", "claude-3", "GPT-3.5-TURBO"}
	validator := NewModelValidator(allowedModels)

	// All should be stored in lowercase
	if !validator.allowedModels["gpt-4"] {
		t.Error("Model should be stored in lowercase")
	}
	if !validator.allowedModels["claude-3"] {
		t.Error("Model should be stored in lowercase")
	}
	if !validator.allowedModels["gpt-3.5-turbo"] {
		t.Error("Model should be stored in lowercase")
	}
}
