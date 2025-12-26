// Package provider contains domain types for AI provider and model management.
package provider

// CostBreakdown represents the cost breakdown for a single model invocation.
type CostBreakdown struct {
	InputCost    float64 // cost for input tokens
	OutputCost   float64 // cost for output tokens
	TotalCost    float64 // total cost (InputCost + OutputCost)
	InputTokens  int     // number of input tokens
	OutputTokens int     // number of output tokens
	Model        string  // model identifier
	Provider     string  // provider name
}

// CalculateCost calculates the cost breakdown for a model invocation.
// Returns nil if model is nil.
// Cost is calculated based on the model's per-1000-token pricing.
func CalculateCost(model *Model, inputTokens, outputTokens int) *CostBreakdown {
	if model == nil {
		return nil
	}

	// Convert per-1K pricing to actual cost
	inputCost := (float64(inputTokens) / 1000.0) * model.InputCostPer1K
	outputCost := (float64(outputTokens) / 1000.0) * model.OutputCostPer1K

	return &CostBreakdown{
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    inputCost + outputCost,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        model.ID,
		Provider:     model.Provider,
	}
}

// CostSummary aggregates costs across multiple phases or invocations.
type CostSummary struct {
	TotalCost         float64            // total cost across all invocations
	TotalInputCost    float64            // total input token costs
	TotalOutputCost   float64            // total output token costs
	TotalInputTokens  int                // total input tokens used
	TotalOutputTokens int                // total output tokens used
	ByProvider        map[string]float64 // cost breakdown by provider
	ByModel           map[string]float64 // cost breakdown by model
	LocalSavings      float64            // estimated savings from using local models
}

// NewCostSummary creates a new empty CostSummary.
func NewCostSummary() *CostSummary {
	return &CostSummary{
		ByProvider: make(map[string]float64),
		ByModel:    make(map[string]float64),
	}
}

// Add adds a CostBreakdown to the summary.
// If breakdown is nil, this is a no-op.
func (s *CostSummary) Add(breakdown *CostBreakdown) {
	if breakdown == nil {
		return
	}

	s.TotalCost += breakdown.TotalCost
	s.TotalInputCost += breakdown.InputCost
	s.TotalOutputCost += breakdown.OutputCost
	s.TotalInputTokens += breakdown.InputTokens
	s.TotalOutputTokens += breakdown.OutputTokens

	if breakdown.Provider != "" {
		s.ByProvider[breakdown.Provider] += breakdown.TotalCost
	}
	if breakdown.Model != "" {
		s.ByModel[breakdown.Model] += breakdown.TotalCost
	}
}

// CalculateSavings calculates the estimated savings from using local models
// by comparing actual cost against what it would cost if all tokens
// were processed by the given premium model.
func (s *CostSummary) CalculateSavings(premiumModel *Model) {
	if premiumModel == nil {
		s.LocalSavings = 0
		return
	}

	// Calculate what the cost would be if all tokens used the premium model
	premiumInputCost := (float64(s.TotalInputTokens) / 1000.0) * premiumModel.InputCostPer1K
	premiumOutputCost := (float64(s.TotalOutputTokens) / 1000.0) * premiumModel.OutputCostPer1K
	premiumTotalCost := premiumInputCost + premiumOutputCost

	// Savings is the difference between premium cost and actual cost
	s.LocalSavings = premiumTotalCost - s.TotalCost
	if s.LocalSavings < 0 {
		s.LocalSavings = 0
	}
}

// InvocationCount returns the total number of distinct models used in this summary.
func (s *CostSummary) InvocationCount() int {
	return len(s.ByModel)
}

// AverageCostPerToken returns the average cost per token (input + output).
// Returns 0 if no tokens have been processed.
func (s *CostSummary) AverageCostPerToken() float64 {
	totalTokens := s.TotalInputTokens + s.TotalOutputTokens
	if totalTokens == 0 {
		return 0
	}
	return s.TotalCost / float64(totalTokens)
}

// Clone creates a deep copy of the CostSummary.
func (s *CostSummary) Clone() *CostSummary {
	clone := &CostSummary{
		TotalCost:         s.TotalCost,
		TotalInputCost:    s.TotalInputCost,
		TotalOutputCost:   s.TotalOutputCost,
		TotalInputTokens:  s.TotalInputTokens,
		TotalOutputTokens: s.TotalOutputTokens,
		LocalSavings:      s.LocalSavings,
		ByProvider:        make(map[string]float64, len(s.ByProvider)),
		ByModel:           make(map[string]float64, len(s.ByModel)),
	}

	for k, v := range s.ByProvider {
		clone.ByProvider[k] = v
	}
	for k, v := range s.ByModel {
		clone.ByModel[k] = v
	}

	return clone
}
