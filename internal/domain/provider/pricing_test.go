package provider

import "testing"

func TestDefaultModelPricing(t *testing.T) {
	pricing := DefaultModelPricing()

	if len(pricing) == 0 {
		t.Fatal("DefaultModelPricing should return non-empty pricing list")
	}

	// Check for some expected models
	foundAnthropic := false
	foundOpenAI := false
	foundOllama := false
	foundGroq := false

	for _, rate := range pricing {
		switch rate.Provider {
		case ProviderAnthropic:
			foundAnthropic = true
		case ProviderOpenAI:
			foundOpenAI = true
		case ProviderOllama:
			foundOllama = true
			// Ollama models should be local with zero cost
			if !rate.IsLocal {
				t.Errorf("Ollama model %s should be local", rate.ModelID)
			}
			if rate.InputRate != 0 || rate.OutputRate != 0 {
				t.Errorf("Ollama model %s should have zero cost, got input=%f output=%f",
					rate.ModelID, rate.InputRate, rate.OutputRate)
			}
		case ProviderGroq:
			foundGroq = true
		}
	}

	if !foundAnthropic {
		t.Error("DefaultModelPricing should include Anthropic models")
	}
	if !foundOpenAI {
		t.Error("DefaultModelPricing should include OpenAI models")
	}
	if !foundOllama {
		t.Error("DefaultModelPricing should include Ollama models")
	}
	if !foundGroq {
		t.Error("DefaultModelPricing should include Groq models")
	}
}

func TestPopulateCostCalculator(t *testing.T) {
	calc := NewCostCalculator()

	// Should start empty
	if calc.ModelCount() != 0 {
		t.Errorf("expected 0 models, got %d", calc.ModelCount())
	}

	// Populate with defaults
	PopulateCostCalculator(calc)

	// Should now have models
	if calc.ModelCount() == 0 {
		t.Error("PopulateCostCalculator should add models")
	}

	// Check Claude Opus 4.5 (latest flagship model)
	// Pricing: $5/MTok input, $25/MTok output = 0.005, 0.025 per 1K tokens
	rate := calc.GetModelCost("claude-opus-4-5-20251101")
	if rate == nil {
		t.Fatal("expected claude-opus-4-5-20251101 to be registered")
	}
	if rate.Provider != ProviderAnthropic {
		t.Errorf("expected provider %s, got %s", ProviderAnthropic, rate.Provider)
	}
	if !floatEquals(rate.InputRate, 0.005) {
		t.Errorf("expected input rate 0.005, got %f", rate.InputRate)
	}
	if !floatEquals(rate.OutputRate, 0.025) {
		t.Errorf("expected output rate 0.025, got %f", rate.OutputRate)
	}

	// Check Claude 3.5 Sonnet (still widely used)
	// Pricing: $3/MTok input, $15/MTok output = 0.003, 0.015 per 1K tokens
	rate = calc.GetModelCost("claude-3-5-sonnet-20241022")
	if rate == nil {
		t.Fatal("expected claude-3-5-sonnet-20241022 to be registered")
	}
	if !floatEquals(rate.InputRate, 0.003) {
		t.Errorf("expected input rate 0.003, got %f", rate.InputRate)
	}
}

func TestPopulateCostCalculator_NilCalc(t *testing.T) {
	// Should not panic with nil calculator
	PopulateCostCalculator(nil)
}

func TestPopulateCostCalculator_CostCalculation(t *testing.T) {
	calc := NewCostCalculator()
	PopulateCostCalculator(calc)

	// Test cost calculation for Claude Opus 4.5
	// Pricing: $5/MTok input, $25/MTok output = 0.005, 0.025 per 1K tokens
	// 1000 input tokens + 500 output tokens
	// Input: (1000/1000) * 0.005 = $0.005
	// Output: (500/1000) * 0.025 = $0.0125
	// Total: $0.0175
	breakdown, err := calc.Calculate("claude-opus-4-5-20251101", 1000, 500)
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}

	expectedInput := 0.005
	expectedOutput := 0.0125
	expectedTotal := 0.0175

	if !floatEquals(breakdown.InputCost, expectedInput) {
		t.Errorf("expected input cost %f, got %f", expectedInput, breakdown.InputCost)
	}
	if !floatEquals(breakdown.OutputCost, expectedOutput) {
		t.Errorf("expected output cost %f, got %f", expectedOutput, breakdown.OutputCost)
	}
	if !floatEquals(breakdown.TotalCost, expectedTotal) {
		t.Errorf("expected total cost %f, got %f", expectedTotal, breakdown.TotalCost)
	}

	// Test GPT-4o mini (budget option)
	// Pricing: $0.15/MTok input, $0.60/MTok output = 0.00015, 0.0006 per 1K tokens
	// 1000 input tokens + 500 output tokens
	// Input: (1000/1000) * 0.00015 = $0.00015
	// Output: (500/1000) * 0.0006 = $0.0003
	// Total: $0.00045
	breakdown, err = calc.Calculate("gpt-4o-mini", 1000, 500)
	if err != nil {
		t.Fatalf("Calculate for GPT-4o mini failed: %v", err)
	}
	if !floatEquals(breakdown.TotalCost, 0.00045) {
		t.Errorf("expected total cost 0.00045, got %f", breakdown.TotalCost)
	}

	// Test local model (should be free)
	breakdown, err = calc.Calculate("llama3.2:3b", 1000, 500)
	if err != nil {
		t.Fatalf("Calculate for local model failed: %v", err)
	}
	if breakdown.TotalCost != 0 {
		t.Errorf("expected zero cost for local model, got %f", breakdown.TotalCost)
	}
}
