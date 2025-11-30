package metrics

// PhaseCost represents the cost for a single phase
type PhaseCost struct {
	PhaseID      string  `json:"phase_id"`
	ModelID      string  `json:"model_id"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	ActualCost   float64 `json:"actual_cost"`
}

// RunCostSummary represents the cost summary for an entire run
type RunCostSummary struct {
	ActualCost      float64     `json:"actual_cost"`
	PremiumOnlyCost float64     `json:"premium_only_cost"`
	CheapOnlyCost   float64     `json:"cheap_only_cost"`
	PhaseCosts      []PhaseCost `json:"phase_costs"`
}

// CostComputer computes costs for phases and runs
type CostComputer struct {
	premiumModelID string
	cheapModelID   string
	models         map[string]ModelPricing
}

// ModelPricing contains pricing information for a model
type ModelPricing struct {
	CostPer1KInput  float64
	CostPer1KOutput float64
}

// NewCostComputer creates a new cost computer
func NewCostComputer(premiumModelID, cheapModelID string, models map[string]ModelPricing) *CostComputer {
	return &CostComputer{
		premiumModelID: premiumModelID,
		cheapModelID:   cheapModelID,
		models:         models,
	}
}

// ComputePhaseCost computes the cost for a single phase
func (cc *CostComputer) ComputePhaseCost(modelID string, inputTokens, outputTokens int) float64 {
	pricing, ok := cc.models[modelID]
	if !ok {
		return 0.0
	}

	inputCost := (float64(inputTokens) / 1000.0) * pricing.CostPer1KInput
	outputCost := (float64(outputTokens) / 1000.0) * pricing.CostPer1KOutput

	return inputCost + outputCost
}

// SummarizeRun computes the cost summary for an entire run
func (cc *CostComputer) SummarizeRun(phaseCosts []PhaseCost) RunCostSummary {
	var actualCost float64
	var premiumOnlyCost float64
	var cheapOnlyCost float64

	premiumPricing, hasPremium := cc.models[cc.premiumModelID]
	cheapPricing, hasCheap := cc.models[cc.cheapModelID]

	for _, phaseCost := range phaseCosts {
		// Actual cost
		actualCost += phaseCost.ActualCost

		// Premium-only cost (what if we used premium model for this phase)
		if hasPremium {
			premiumInputCost := (float64(phaseCost.InputTokens) / 1000.0) * premiumPricing.CostPer1KInput
			premiumOutputCost := (float64(phaseCost.OutputTokens) / 1000.0) * premiumPricing.CostPer1KOutput
			premiumOnlyCost += premiumInputCost + premiumOutputCost
		}

		// Cheap-only cost (what if we used cheap model for this phase)
		if hasCheap {
			cheapInputCost := (float64(phaseCost.InputTokens) / 1000.0) * cheapPricing.CostPer1KInput
			cheapOutputCost := (float64(phaseCost.OutputTokens) / 1000.0) * cheapPricing.CostPer1KOutput
			cheapOnlyCost += cheapInputCost + cheapOutputCost
		}
	}

	return RunCostSummary{
		ActualCost:      actualCost,
		PremiumOnlyCost: premiumOnlyCost,
		CheapOnlyCost:   cheapOnlyCost,
		PhaseCosts:      phaseCosts,
	}
}
