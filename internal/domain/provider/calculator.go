// Package provider contains domain types for AI provider and model management.
package provider

import (
	"errors"
	"sync"
)

// ErrModelNotFound is returned when a model is not registered in the calculator.
var ErrModelNotFound = errors.New("model not found in cost calculator")

// ModelCostRate represents the cost rates for a specific model.
type ModelCostRate struct {
	ModelID    string  // unique identifier for the model
	Provider   string  // provider name (ollama, anthropic, openai, groq)
	InputRate  float64 // cost per 1000 input tokens
	OutputRate float64 // cost per 1000 output tokens
	IsLocal    bool    // whether this is a local model (zero cost)
}

// CostCalculator manages cost calculations for AI model invocations.
// It maintains a registry of model cost rates and provides methods
// for calculating costs based on token usage.
type CostCalculator struct {
	mu     sync.RWMutex
	models map[string]*ModelCostRate
}

// NewCostCalculator creates a new CostCalculator with an empty model registry.
func NewCostCalculator() *CostCalculator {
	return &CostCalculator{
		models: make(map[string]*ModelCostRate),
	}
}

// NewCostCalculatorFromModels creates a CostCalculator pre-populated with
// cost rates from the given Model slice.
func NewCostCalculatorFromModels(models []*Model) *CostCalculator {
	calc := NewCostCalculator()
	for _, m := range models {
		if m != nil {
			calc.RegisterModelWithProvider(m.ID, m.Provider, m.InputCostPer1K, m.OutputCostPer1K)
		}
	}
	return calc
}

// RegisterModel registers a model with its cost rates.
// If the model already exists, its rates are updated.
func (c *CostCalculator) RegisterModel(modelID string, inputRate, outputRate float64) {
	c.RegisterModelWithProvider(modelID, "", inputRate, outputRate)
}

// RegisterModelWithProvider registers a model with its provider and cost rates.
// If the model already exists, its rates are updated.
func (c *CostCalculator) RegisterModelWithProvider(modelID, provider string, inputRate, outputRate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	isLocal := provider == ProviderOllama
	c.models[modelID] = &ModelCostRate{
		ModelID:    modelID,
		Provider:   provider,
		InputRate:  inputRate,
		OutputRate: outputRate,
		IsLocal:    isLocal,
	}
}

// RegisterLocalModel registers a local model with zero cost rates.
// Local models (e.g., Ollama models) don't incur API costs.
func (c *CostCalculator) RegisterLocalModel(modelID string) {
	c.RegisterModelWithProvider(modelID, ProviderOllama, 0, 0)
}

// UnregisterModel removes a model from the calculator.
// Returns true if the model was found and removed, false otherwise.
func (c *CostCalculator) UnregisterModel(modelID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.models[modelID]; exists {
		delete(c.models, modelID)
		return true
	}
	return false
}

// GetModelCost retrieves the cost rates for a specific model.
// Returns nil if the model is not registered.
func (c *CostCalculator) GetModelCost(modelID string) *ModelCostRate {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rate, exists := c.models[modelID]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	return &ModelCostRate{
		ModelID:    rate.ModelID,
		Provider:   rate.Provider,
		InputRate:  rate.InputRate,
		OutputRate: rate.OutputRate,
		IsLocal:    rate.IsLocal,
	}
}

// HasModel checks if a model is registered in the calculator.
func (c *CostCalculator) HasModel(modelID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.models[modelID]
	return exists
}

// Calculate computes the cost breakdown for a model invocation.
// Returns an error if the model is not registered.
func (c *CostCalculator) Calculate(modelID string, inputTokens, outputTokens int) (*CostBreakdown, error) {
	c.mu.RLock()
	rate, exists := c.models[modelID]
	c.mu.RUnlock()

	if !exists {
		return nil, ErrModelNotFound
	}

	// Local models have zero cost
	if rate.IsLocal {
		return &CostBreakdown{
			InputCost:    0,
			OutputCost:   0,
			TotalCost:    0,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Model:        modelID,
			Provider:     rate.Provider,
		}, nil
	}

	// Calculate costs based on per-1K token rates
	inputCost := (float64(inputTokens) / 1000.0) * rate.InputRate
	outputCost := (float64(outputTokens) / 1000.0) * rate.OutputRate

	return &CostBreakdown{
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    inputCost + outputCost,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        modelID,
		Provider:     rate.Provider,
	}, nil
}

// CalculateOrZero computes the cost breakdown, returning zero cost if model not found.
// This is useful when you want to track token usage even for unknown models.
func (c *CostCalculator) CalculateOrZero(modelID string, inputTokens, outputTokens int) *CostBreakdown {
	breakdown, err := c.Calculate(modelID, inputTokens, outputTokens)
	if err != nil {
		return &CostBreakdown{
			InputCost:    0,
			OutputCost:   0,
			TotalCost:    0,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Model:        modelID,
			Provider:     "",
		}
	}
	return breakdown
}

// EstimateCost provides a pre-request cost estimate assuming a balanced
// token distribution (e.g., for chat interactions where input ≈ output).
// estimatedTokens is split evenly between input and output.
func (c *CostCalculator) EstimateCost(modelID string, estimatedTokens int) (float64, error) {
	c.mu.RLock()
	rate, exists := c.models[modelID]
	c.mu.RUnlock()

	if !exists {
		return 0, ErrModelNotFound
	}

	if rate.IsLocal {
		return 0, nil
	}

	// Split tokens evenly between input and output
	halfTokens := float64(estimatedTokens) / 2.0
	inputCost := (halfTokens / 1000.0) * rate.InputRate
	outputCost := (halfTokens / 1000.0) * rate.OutputRate

	return inputCost + outputCost, nil
}

// EstimateInputOutputCost provides a more accurate cost estimate when
// you know the expected input and output token counts separately.
func (c *CostCalculator) EstimateInputOutputCost(modelID string, inputTokens, outputTokens int) (float64, error) {
	breakdown, err := c.Calculate(modelID, inputTokens, outputTokens)
	if err != nil {
		return 0, err
	}
	return breakdown.TotalCost, nil
}

// GetCheapestModel returns the model ID with the lowest combined cost rate
// from the registered models. Returns empty string if no models registered.
func (c *CostCalculator) GetCheapestModel() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var cheapestID string
	var cheapestRate float64 = -1

	for id, rate := range c.models {
		combinedRate := rate.InputRate + rate.OutputRate
		if cheapestRate < 0 || combinedRate < cheapestRate {
			cheapestRate = combinedRate
			cheapestID = id
		}
	}

	return cheapestID
}

// GetModelsByProvider returns all model IDs registered for a specific provider.
func (c *CostCalculator) GetModelsByProvider(provider string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var models []string
	for id, rate := range c.models {
		if rate.Provider == provider {
			models = append(models, id)
		}
	}
	return models
}

// ModelCount returns the number of registered models.
func (c *CostCalculator) ModelCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.models)
}

// Clear removes all registered models from the calculator.
func (c *CostCalculator) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.models = make(map[string]*ModelCostRate)
}

// Clone creates a deep copy of the CostCalculator with all registered models.
func (c *CostCalculator) Clone() *CostCalculator {
	c.mu.RLock()
	defer c.mu.RUnlock()

	clone := NewCostCalculator()
	for id, rate := range c.models {
		clone.models[id] = &ModelCostRate{
			ModelID:    rate.ModelID,
			Provider:   rate.Provider,
			InputRate:  rate.InputRate,
			OutputRate: rate.OutputRate,
			IsLocal:    rate.IsLocal,
		}
	}
	return clone
}
