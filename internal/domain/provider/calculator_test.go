package provider

import (
	"sync"
	"testing"
)

func TestNewCostCalculator(t *testing.T) {
	calc := NewCostCalculator()

	if calc == nil {
		t.Fatal("NewCostCalculator returned nil")
	}

	if calc.models == nil {
		t.Error("models map should be initialized")
	}

	if calc.ModelCount() != 0 {
		t.Error("new calculator should have no models")
	}
}

func TestNewCostCalculatorFromModels(t *testing.T) {
	models := []*Model{
		NewModel("gpt-4", "GPT-4", ProviderOpenAI).WithCosts(0.03, 0.06),
		NewModel("claude-3", "Claude 3", ProviderAnthropic).WithCosts(0.015, 0.075),
		nil, // should be ignored
	}

	calc := NewCostCalculatorFromModels(models)

	if calc.ModelCount() != 2 {
		t.Errorf("expected 2 models, got %d", calc.ModelCount())
	}

	if !calc.HasModel("gpt-4") {
		t.Error("gpt-4 should be registered")
	}

	if !calc.HasModel("claude-3") {
		t.Error("claude-3 should be registered")
	}
}

func TestRegisterModel(t *testing.T) {
	calc := NewCostCalculator()

	calc.RegisterModel("test-model", 0.01, 0.02)

	if !calc.HasModel("test-model") {
		t.Error("model should be registered")
	}

	rate := calc.GetModelCost("test-model")
	if rate == nil {
		t.Fatal("GetModelCost returned nil for registered model")
	}

	if rate.InputRate != 0.01 {
		t.Errorf("expected input rate 0.01, got %f", rate.InputRate)
	}

	if rate.OutputRate != 0.02 {
		t.Errorf("expected output rate 0.02, got %f", rate.OutputRate)
	}
}

func TestRegisterModelWithProvider(t *testing.T) {
	calc := NewCostCalculator()

	calc.RegisterModelWithProvider("gpt-4", ProviderOpenAI, 0.03, 0.06)

	rate := calc.GetModelCost("gpt-4")
	if rate == nil {
		t.Fatal("GetModelCost returned nil")
	}

	if rate.Provider != ProviderOpenAI {
		t.Errorf("expected provider %s, got %s", ProviderOpenAI, rate.Provider)
	}

	if rate.IsLocal {
		t.Error("OpenAI model should not be marked as local")
	}
}

func TestRegisterLocalModel(t *testing.T) {
	calc := NewCostCalculator()

	calc.RegisterLocalModel("llama3")

	rate := calc.GetModelCost("llama3")
	if rate == nil {
		t.Fatal("GetModelCost returned nil")
	}

	if rate.Provider != ProviderOllama {
		t.Errorf("expected provider %s, got %s", ProviderOllama, rate.Provider)
	}

	if !rate.IsLocal {
		t.Error("Ollama model should be marked as local")
	}

	if rate.InputRate != 0 || rate.OutputRate != 0 {
		t.Error("local model should have zero cost rates")
	}
}

func TestUnregisterModel(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModel("test", 0.01, 0.02)

	// Unregister existing model
	if !calc.UnregisterModel("test") {
		t.Error("UnregisterModel should return true for existing model")
	}

	if calc.HasModel("test") {
		t.Error("model should be unregistered")
	}

	// Unregister non-existent model
	if calc.UnregisterModel("nonexistent") {
		t.Error("UnregisterModel should return false for non-existent model")
	}
}

func TestGetModelCost(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModelWithProvider("test", ProviderAnthropic, 0.015, 0.075)

	// Get existing model
	rate := calc.GetModelCost("test")
	if rate == nil {
		t.Fatal("GetModelCost returned nil for existing model")
	}

	// Verify it's a copy (modification shouldn't affect original)
	rate.InputRate = 999
	original := calc.GetModelCost("test")
	if original.InputRate == 999 {
		t.Error("GetModelCost should return a copy, not reference")
	}

	// Get non-existent model
	if calc.GetModelCost("nonexistent") != nil {
		t.Error("GetModelCost should return nil for non-existent model")
	}
}

func TestCalculate(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModelWithProvider("gpt-4", ProviderOpenAI, 0.03, 0.06)

	breakdown, err := calc.Calculate("gpt-4", 1000, 500)
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}

	if breakdown == nil {
		t.Fatal("Calculate returned nil breakdown")
	}

	// Input: 1000 tokens * $0.03/1K = $0.03
	expectedInputCost := 0.03
	if breakdown.InputCost != expectedInputCost {
		t.Errorf("expected input cost %f, got %f", expectedInputCost, breakdown.InputCost)
	}

	// Output: 500 tokens * $0.06/1K = $0.03
	expectedOutputCost := 0.03
	if breakdown.OutputCost != expectedOutputCost {
		t.Errorf("expected output cost %f, got %f", expectedOutputCost, breakdown.OutputCost)
	}

	// Total: $0.03 + $0.03 = $0.06
	expectedTotalCost := 0.06
	if breakdown.TotalCost != expectedTotalCost {
		t.Errorf("expected total cost %f, got %f", expectedTotalCost, breakdown.TotalCost)
	}

	if breakdown.InputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", breakdown.InputTokens)
	}

	if breakdown.OutputTokens != 500 {
		t.Errorf("expected 500 output tokens, got %d", breakdown.OutputTokens)
	}

	if breakdown.Model != "gpt-4" {
		t.Errorf("expected model gpt-4, got %s", breakdown.Model)
	}

	if breakdown.Provider != ProviderOpenAI {
		t.Errorf("expected provider %s, got %s", ProviderOpenAI, breakdown.Provider)
	}
}

func TestCalculateLocalModel(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterLocalModel("llama3")

	breakdown, err := calc.Calculate("llama3", 1000, 500)
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}

	if breakdown.TotalCost != 0 {
		t.Errorf("local model should have zero cost, got %f", breakdown.TotalCost)
	}

	// Token counts should still be tracked
	if breakdown.InputTokens != 1000 || breakdown.OutputTokens != 500 {
		t.Error("token counts should be tracked for local models")
	}
}

func TestCalculateNotFound(t *testing.T) {
	calc := NewCostCalculator()

	_, err := calc.Calculate("nonexistent", 1000, 500)
	if err != ErrModelNotFound {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestCalculateOrZero(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModel("known", 0.01, 0.02)

	// Known model
	breakdown := calc.CalculateOrZero("known", 1000, 500)
	if breakdown.TotalCost == 0 {
		t.Error("known model should have non-zero cost")
	}

	// Unknown model
	breakdown = calc.CalculateOrZero("unknown", 1000, 500)
	if breakdown.TotalCost != 0 {
		t.Errorf("unknown model should have zero cost, got %f", breakdown.TotalCost)
	}

	// Token counts should still be tracked
	if breakdown.InputTokens != 1000 || breakdown.OutputTokens != 500 {
		t.Error("token counts should be tracked even for unknown models")
	}

	if breakdown.Model != "unknown" {
		t.Errorf("model ID should be set, got %s", breakdown.Model)
	}
}

func TestCalculatorEstimateCost(t *testing.T) {
	calc := NewCostCalculator()
	// $0.03/1K input, $0.06/1K output
	calc.RegisterModelWithProvider("gpt-4", ProviderOpenAI, 0.03, 0.06)

	// Estimate for 2000 tokens (1000 input, 1000 output)
	cost, err := calc.EstimateCost("gpt-4", 2000)
	if err != nil {
		t.Fatalf("EstimateCost failed: %v", err)
	}

	// 1000 input tokens * $0.03/1K = $0.03
	// 1000 output tokens * $0.06/1K = $0.06
	// Total = $0.09
	expectedCost := 0.09
	if cost != expectedCost {
		t.Errorf("expected estimate %f, got %f", expectedCost, cost)
	}
}

func TestCalculatorEstimateCostLocalModel(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterLocalModel("llama3")

	cost, err := calc.EstimateCost("llama3", 2000)
	if err != nil {
		t.Fatalf("EstimateCost failed: %v", err)
	}

	if cost != 0 {
		t.Errorf("local model estimate should be zero, got %f", cost)
	}
}

func TestCalculatorEstimateCostNotFound(t *testing.T) {
	calc := NewCostCalculator()

	_, err := calc.EstimateCost("nonexistent", 2000)
	if err != ErrModelNotFound {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestCalculatorEstimateInputOutputCost(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModel("test", 0.01, 0.02)

	cost, err := calc.EstimateInputOutputCost("test", 2000, 1000)
	if err != nil {
		t.Fatalf("EstimateInputOutputCost failed: %v", err)
	}

	// 2000 input * $0.01/1K = $0.02
	// 1000 output * $0.02/1K = $0.02
	// Total = $0.04
	expectedCost := 0.04
	if cost != expectedCost {
		t.Errorf("expected %f, got %f", expectedCost, cost)
	}
}

func TestCalculatorEstimateInputOutputCostNotFound(t *testing.T) {
	calc := NewCostCalculator()

	_, err := calc.EstimateInputOutputCost("nonexistent", 1000, 500)
	if err != ErrModelNotFound {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestGetCheapestModel(t *testing.T) {
	calc := NewCostCalculator()

	// Empty calculator
	if calc.GetCheapestModel() != "" {
		t.Error("empty calculator should return empty string")
	}

	// Add models
	calc.RegisterModel("expensive", 0.10, 0.20)
	calc.RegisterModel("cheap", 0.01, 0.02)
	calc.RegisterModel("medium", 0.05, 0.10)

	cheapest := calc.GetCheapestModel()
	if cheapest != "cheap" {
		t.Errorf("expected 'cheap' as cheapest model, got %s", cheapest)
	}
}

func TestGetModelsByProvider(t *testing.T) {
	calc := NewCostCalculator()

	calc.RegisterModelWithProvider("gpt-4", ProviderOpenAI, 0.03, 0.06)
	calc.RegisterModelWithProvider("gpt-3.5", ProviderOpenAI, 0.001, 0.002)
	calc.RegisterModelWithProvider("claude-3", ProviderAnthropic, 0.015, 0.075)
	calc.RegisterLocalModel("llama3")

	openaiModels := calc.GetModelsByProvider(ProviderOpenAI)
	if len(openaiModels) != 2 {
		t.Errorf("expected 2 OpenAI models, got %d", len(openaiModels))
	}

	anthropicModels := calc.GetModelsByProvider(ProviderAnthropic)
	if len(anthropicModels) != 1 {
		t.Errorf("expected 1 Anthropic model, got %d", len(anthropicModels))
	}

	ollamaModels := calc.GetModelsByProvider(ProviderOllama)
	if len(ollamaModels) != 1 {
		t.Errorf("expected 1 Ollama model, got %d", len(ollamaModels))
	}

	groqModels := calc.GetModelsByProvider(ProviderGroq)
	if len(groqModels) != 0 {
		t.Errorf("expected 0 Groq models, got %d", len(groqModels))
	}
}

func TestClear(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModel("test1", 0.01, 0.02)
	calc.RegisterModel("test2", 0.01, 0.02)

	if calc.ModelCount() != 2 {
		t.Fatal("setup failed")
	}

	calc.Clear()

	if calc.ModelCount() != 0 {
		t.Error("Clear should remove all models")
	}
}

func TestClone(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModelWithProvider("gpt-4", ProviderOpenAI, 0.03, 0.06)

	clone := calc.Clone()

	// Verify clone has same data
	if clone.ModelCount() != calc.ModelCount() {
		t.Error("clone should have same number of models")
	}

	rate := clone.GetModelCost("gpt-4")
	if rate == nil {
		t.Fatal("clone should have gpt-4 model")
	}

	if rate.InputRate != 0.03 || rate.OutputRate != 0.06 {
		t.Error("clone should have same rates")
	}

	// Verify independence
	clone.RegisterModel("new-model", 0.01, 0.02)
	if calc.HasModel("new-model") {
		t.Error("modifying clone should not affect original")
	}
}

func TestRegisterModelUpdate(t *testing.T) {
	calc := NewCostCalculator()

	// Initial registration
	calc.RegisterModel("test", 0.01, 0.02)

	// Update with new rates
	calc.RegisterModel("test", 0.05, 0.10)

	rate := calc.GetModelCost("test")
	if rate.InputRate != 0.05 || rate.OutputRate != 0.10 {
		t.Error("RegisterModel should update existing model rates")
	}
}

func TestConcurrency(t *testing.T) {
	calc := NewCostCalculator()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			calc.RegisterModel("model"+string(rune(n)), float64(n)*0.01, float64(n)*0.02)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			calc.ModelCount()
			calc.GetCheapestModel()
		}()
	}

	// Concurrent calculations
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			calc.CalculateOrZero("model0", 100, 50)
		}()
	}

	wg.Wait()

	// If we got here without deadlock or panic, concurrency is working
}

func TestHasModel(t *testing.T) {
	calc := NewCostCalculator()

	if calc.HasModel("test") {
		t.Error("HasModel should return false for unregistered model")
	}

	calc.RegisterModel("test", 0.01, 0.02)

	if !calc.HasModel("test") {
		t.Error("HasModel should return true for registered model")
	}
}

func TestModelCount(t *testing.T) {
	calc := NewCostCalculator()

	if calc.ModelCount() != 0 {
		t.Error("new calculator should have 0 models")
	}

	calc.RegisterModel("test1", 0.01, 0.02)
	if calc.ModelCount() != 1 {
		t.Error("expected 1 model")
	}

	calc.RegisterModel("test2", 0.01, 0.02)
	if calc.ModelCount() != 2 {
		t.Error("expected 2 models")
	}

	calc.UnregisterModel("test1")
	if calc.ModelCount() != 1 {
		t.Error("expected 1 model after unregister")
	}
}

func TestCalculateZeroTokens(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModel("test", 0.01, 0.02)

	breakdown, err := calc.Calculate("test", 0, 0)
	if err != nil {
		t.Fatalf("Calculate with zero tokens failed: %v", err)
	}

	if breakdown.TotalCost != 0 {
		t.Error("zero tokens should result in zero cost")
	}

	if breakdown.InputTokens != 0 || breakdown.OutputTokens != 0 {
		t.Error("token counts should be zero")
	}
}

func TestCalculateLargeTokenCounts(t *testing.T) {
	calc := NewCostCalculator()
	calc.RegisterModel("test", 0.03, 0.06)

	// 1 million input tokens, 500K output tokens
	breakdown, err := calc.Calculate("test", 1000000, 500000)
	if err != nil {
		t.Fatalf("Calculate with large tokens failed: %v", err)
	}

	// 1M input * $0.03/1K = $30
	expectedInputCost := 30.0
	if breakdown.InputCost != expectedInputCost {
		t.Errorf("expected input cost %f, got %f", expectedInputCost, breakdown.InputCost)
	}

	// 500K output * $0.06/1K = $30
	expectedOutputCost := 30.0
	if breakdown.OutputCost != expectedOutputCost {
		t.Errorf("expected output cost %f, got %f", expectedOutputCost, breakdown.OutputCost)
	}

	expectedTotalCost := 60.0
	if breakdown.TotalCost != expectedTotalCost {
		t.Errorf("expected total cost %f, got %f", expectedTotalCost, breakdown.TotalCost)
	}
}
