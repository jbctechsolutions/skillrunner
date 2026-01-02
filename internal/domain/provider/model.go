// Package provider contains domain types for AI provider and model management.
package provider

import "slices"

// Provider names
const (
	ProviderOllama    = "ollama"
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	ProviderGroq      = "groq"
)

// Common capability identifiers
const (
	CapabilityVision          = "vision"
	CapabilityFunctionCalling = "function_calling"
	CapabilityStreaming       = "streaming"
)

// Model represents metadata about an AI model from any provider.
// It includes information about capabilities, costs, and context limits.
type Model struct {
	ID                  string    // unique identifier for the model
	Name                string    // human-readable name
	Provider            string    // ollama, anthropic, openai, groq
	ContextWindow       int       // max tokens the model can handle
	InputCostPer1K      float64   // cost per 1000 input tokens
	OutputCostPer1K     float64   // cost per 1000 output tokens
	InputPricePerToken  float64   // cost per single input token (for cost.go compatibility)
	OutputPricePerToken float64   // cost per single output token (for cost.go compatibility)
	Capabilities        []string  // vision, function_calling, streaming
	Tier                AgentTier // cheap, balanced, premium
}

// NewModel creates a new Model with the required fields.
// The model starts with empty capabilities and balanced tier.
func NewModel(id, name, provider string) *Model {
	return &Model{
		ID:           id,
		Name:         name,
		Provider:     provider,
		Capabilities: []string{},
		Tier:         TierBalanced,
	}
}

// WithContextWindow sets the context window size for the model.
// Returns the model for fluent chaining.
func (m *Model) WithContextWindow(size int) *Model {
	m.ContextWindow = size
	return m
}

// WithCosts sets the input and output token costs per 1000 tokens.
// Also automatically sets the per-token prices for compatibility with cost calculations.
// Returns the model for fluent chaining.
func (m *Model) WithCosts(inputCost, outputCost float64) *Model {
	m.InputCostPer1K = inputCost
	m.OutputCostPer1K = outputCost
	// Also set per-token prices for compatibility with cost.go
	m.InputPricePerToken = inputCost / 1000.0
	m.OutputPricePerToken = outputCost / 1000.0
	return m
}

// WithCapabilities sets the model's capabilities.
// Returns the model for fluent chaining.
func (m *Model) WithCapabilities(caps ...string) *Model {
	m.Capabilities = make([]string, len(caps))
	copy(m.Capabilities, caps)
	return m
}

// WithTier sets the model's agent tier.
// Returns the model for fluent chaining.
func (m *Model) WithTier(tier AgentTier) *Model {
	m.Tier = tier
	return m
}

// HasCapability returns true if the model has the specified capability.
func (m *Model) HasCapability(cap string) bool {
	return slices.Contains(m.Capabilities, cap)
}

// EstimateCost calculates the estimated cost for a given number of
// input and output tokens. Returns the total cost in the same currency
// unit as the per-1K costs.
func (m *Model) EstimateCost(inputTokens, outputTokens int) float64 {
	inputCost := (float64(inputTokens) / 1000.0) * m.InputCostPer1K
	outputCost := (float64(outputTokens) / 1000.0) * m.OutputCostPer1K
	return inputCost + outputCost
}

// IsLocal returns true if the model runs locally (i.e., provider is ollama).
func (m *Model) IsLocal() bool {
	return m.Provider == ProviderOllama
}
