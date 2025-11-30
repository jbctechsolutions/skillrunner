package router

import (
	"fmt"
	"regexp"
	"strings"
)

// ModelValidator validates model names and identifiers
type ModelValidator struct {
	allowedModels map[string]bool
	modelPattern  *regexp.Regexp
}

// NewModelValidator creates a new model validator
func NewModelValidator(allowedModels []string) *ModelValidator {
	allowedMap := make(map[string]bool)
	for _, model := range allowedModels {
		allowedMap[strings.ToLower(model)] = true
	}

	// Pattern: alphanumeric, hyphens, underscores, dots, and slashes
	// Examples: gpt-4, claude-3, gpt-3.5-turbo, openai/gpt-4
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

	return &ModelValidator{
		allowedModels: allowedMap,
		modelPattern:  pattern,
	}
}

// DefaultModelValidator creates a validator with default allowed models
func DefaultModelValidator() *ModelValidator {
	return NewModelValidator([]string{
		"gpt-4",
		"gpt-4-turbo",
		"gpt-3.5-turbo",
		"claude-3",
		"claude-3-opus",
		"claude-3-sonnet",
		"claude-3-haiku",
		"claude-2",
		"claude-2.1",
	})
}

// Validate validates a model name
func (v *ModelValidator) Validate(model string) error {
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	// Check format
	if !v.modelPattern.MatchString(model) {
		return fmt.Errorf("invalid model name format: %s (must contain only alphanumeric characters, hyphens, underscores, dots, and slashes)", model)
	}

	// Check if model is in allowed list (if list is not empty)
	if len(v.allowedModels) > 0 {
		modelLower := strings.ToLower(model)
		if !v.allowedModels[modelLower] {
			return fmt.Errorf("model '%s' is not in the allowed list", model)
		}
	}

	return nil
}

// ValidateWithSuggestions validates a model and returns suggestions if invalid
func (v *ModelValidator) ValidateWithSuggestions(model string) (error, []string) {
	err := v.Validate(model)
	if err == nil {
		return nil, nil
	}

	// Generate suggestions for similar model names
	suggestions := v.generateSuggestions(model)
	return err, suggestions
}

// IsValid checks if a model name is valid without returning an error
func (v *ModelValidator) IsValid(model string) bool {
	return v.Validate(model) == nil
}

// AddAllowedModel adds a model to the allowed list
func (v *ModelValidator) AddAllowedModel(model string) {
	if model != "" {
		v.allowedModels[strings.ToLower(model)] = true
	}
}

// RemoveAllowedModel removes a model from the allowed list
func (v *ModelValidator) RemoveAllowedModel(model string) {
	delete(v.allowedModels, strings.ToLower(model))
}

// GetAllowedModels returns the list of allowed models
func (v *ModelValidator) GetAllowedModels() []string {
	models := make([]string, 0, len(v.allowedModels))
	for model := range v.allowedModels {
		models = append(models, model)
	}
	return models
}

// generateSuggestions generates suggestions for similar model names
func (v *ModelValidator) generateSuggestions(model string) []string {
	if len(v.allowedModels) == 0 {
		return nil
	}

	modelLower := strings.ToLower(model)
	suggestions := make([]string, 0)

	// Find models that contain the input as a substring
	for allowed := range v.allowedModels {
		if strings.Contains(allowed, modelLower) || strings.Contains(modelLower, allowed) {
			suggestions = append(suggestions, allowed)
		}
	}

	// If no substring matches, find models with similar prefixes
	if len(suggestions) == 0 {
		prefix := strings.Split(modelLower, "-")[0]
		for allowed := range v.allowedModels {
			if strings.HasPrefix(allowed, prefix) {
				suggestions = append(suggestions, allowed)
			}
		}
	}

	// Limit to 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// ValidateModelList validates a list of model names
func (v *ModelValidator) ValidateModelList(models []string) []error {
	var errors []error
	for i, model := range models {
		if err := v.Validate(model); err != nil {
			errors = append(errors, fmt.Errorf("model[%d] '%s': %w", i, model, err))
		}
	}
	return errors
}

// ValidateModelMap validates model names in a map
func (v *ModelValidator) ValidateModelMap(modelMap map[string]string) []error {
	var errors []error
	for key, model := range modelMap {
		if err := v.Validate(model); err != nil {
			errors = append(errors, fmt.Errorf("key '%s': %w", key, err))
		}
	}
	return errors
}
