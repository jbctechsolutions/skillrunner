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
	rate := calc.GetModelCost("claude-opus-4-5-20251101")
	if rate == nil {
		t.Fatal("expected claude-opus-4-5-20251101 to be registered")
	}
	if rate.Provider != ProviderAnthropic {
		t.Errorf("expected provider %s, got %s", ProviderAnthropic, rate.Provider)
	}
	if rate.InputRate != 5.0 {
		t.Errorf("expected input rate 5.0, got %f", rate.InputRate)
	}
	if rate.OutputRate != 25.0 {
		t.Errorf("expected output rate 25.0, got %f", rate.OutputRate)
	}

	// Check Claude 3.5 Sonnet (still widely used)
	rate = calc.GetModelCost("claude-3-5-sonnet-20241022")
	if rate == nil {
		t.Fatal("expected claude-3-5-sonnet-20241022 to be registered")
	}
	if rate.InputRate != 3.0 {
		t.Errorf("expected input rate 3.0, got %f", rate.InputRate)
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
	// 1000 input tokens + 500 output tokens
	// Input: 1000 * $5.0/1K = $5.00
	// Output: 500 * $25.0/1K = $12.50
	// Total: $17.50
	breakdown, err := calc.Calculate("claude-opus-4-5-20251101", 1000, 500)
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}

	expectedInput := 5.0
	expectedOutput := 12.5
	expectedTotal := 17.5

	if breakdown.InputCost != expectedInput {
		t.Errorf("expected input cost %f, got %f", expectedInput, breakdown.InputCost)
	}
	if breakdown.OutputCost != expectedOutput {
		t.Errorf("expected output cost %f, got %f", expectedOutput, breakdown.OutputCost)
	}
	if breakdown.TotalCost != expectedTotal {
		t.Errorf("expected total cost %f, got %f", expectedTotal, breakdown.TotalCost)
	}

	// Test GPT-4o mini (budget option)
	// 1000 input tokens + 500 output tokens
	// Input: 1000 * $0.15/1K = $0.15
	// Output: 500 * $0.60/1K = $0.30
	// Total: $0.45
	breakdown, err = calc.Calculate("gpt-4o-mini", 1000, 500)
	if err != nil {
		t.Fatalf("Calculate for GPT-4o mini failed: %v", err)
	}
	if !floatEquals(breakdown.TotalCost, 0.45) {
		t.Errorf("expected total cost 0.45, got %f", breakdown.TotalCost)
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
