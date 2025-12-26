package provider

import (
	"testing"
)

// Note: floatEquals is defined in model_test.go with signature floatEquals(a, b float64) bool
// We reuse that function for consistency across the package tests

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name         string
		model        *Model
		inputTokens  int
		outputTokens int
		wantNil      bool
		wantInput    float64
		wantOutput   float64
		wantTotal    float64
	}{
		{
			name:         "nil model returns nil",
			model:        nil,
			inputTokens:  1000,
			outputTokens: 500,
			wantNil:      true,
		},
		{
			name: "claude-3-opus pricing",
			model: NewModel("claude-3-opus", "Claude 3 Opus", ProviderAnthropic).
				WithCosts(15.0, 75.0), // $15 per 1K input, $75 per 1K output
			inputTokens:  1000,
			outputTokens: 500,
			wantInput:    15.0, // 1000/1000 * 15
			wantOutput:   37.5, // 500/1000 * 75
			wantTotal:    52.5,
		},
		{
			name: "gpt-4 pricing",
			model: NewModel("gpt-4", "GPT-4", ProviderOpenAI).
				WithCosts(30.0, 60.0), // $30 per 1K input, $60 per 1K output
			inputTokens:  2000,
			outputTokens: 1000,
			wantInput:    60.0, // 2000/1000 * 30
			wantOutput:   60.0, // 1000/1000 * 60
			wantTotal:    120.0,
		},
		{
			name: "local model (zero cost)",
			model: NewModel("llama3", "Llama 3", ProviderOllama).
				WithCosts(0, 0),
			inputTokens:  5000,
			outputTokens: 2000,
			wantInput:    0,
			wantOutput:   0,
			wantTotal:    0,
		},
		{
			name: "zero tokens",
			model: NewModel("test-model", "Test", ProviderOpenAI).
				WithCosts(10.0, 20.0),
			inputTokens:  0,
			outputTokens: 0,
			wantInput:    0,
			wantOutput:   0,
			wantTotal:    0,
		},
		{
			name: "fractional token counts",
			model: NewModel("test-model", "Test", ProviderOpenAI).
				WithCosts(10.0, 20.0),
			inputTokens:  150, // 0.15 * 10 = 1.5
			outputTokens: 250, // 0.25 * 20 = 5.0
			wantInput:    1.5,
			wantOutput:   5.0,
			wantTotal:    6.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCost(tt.model, tt.inputTokens, tt.outputTokens)

			if tt.wantNil {
				if got != nil {
					t.Errorf("CalculateCost() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("CalculateCost() returned nil, want non-nil")
			}

			if !floatEquals(got.InputCost, tt.wantInput) {
				t.Errorf("InputCost = %v, want %v", got.InputCost, tt.wantInput)
			}
			if !floatEquals(got.OutputCost, tt.wantOutput) {
				t.Errorf("OutputCost = %v, want %v", got.OutputCost, tt.wantOutput)
			}
			if !floatEquals(got.TotalCost, tt.wantTotal) {
				t.Errorf("TotalCost = %v, want %v", got.TotalCost, tt.wantTotal)
			}
			if got.InputTokens != tt.inputTokens {
				t.Errorf("InputTokens = %v, want %v", got.InputTokens, tt.inputTokens)
			}
			if got.OutputTokens != tt.outputTokens {
				t.Errorf("OutputTokens = %v, want %v", got.OutputTokens, tt.outputTokens)
			}
			if got.Model != tt.model.ID {
				t.Errorf("Model = %v, want %v", got.Model, tt.model.ID)
			}
			if got.Provider != tt.model.Provider {
				t.Errorf("Provider = %v, want %v", got.Provider, tt.model.Provider)
			}
		})
	}
}

func TestNewCostSummary(t *testing.T) {
	summary := NewCostSummary()

	if summary == nil {
		t.Fatal("NewCostSummary() returned nil")
	}
	if summary.TotalCost != 0 {
		t.Errorf("TotalCost = %v, want 0", summary.TotalCost)
	}
	if summary.TotalInputCost != 0 {
		t.Errorf("TotalInputCost = %v, want 0", summary.TotalInputCost)
	}
	if summary.TotalOutputCost != 0 {
		t.Errorf("TotalOutputCost = %v, want 0", summary.TotalOutputCost)
	}
	if summary.TotalInputTokens != 0 {
		t.Errorf("TotalInputTokens = %v, want 0", summary.TotalInputTokens)
	}
	if summary.TotalOutputTokens != 0 {
		t.Errorf("TotalOutputTokens = %v, want 0", summary.TotalOutputTokens)
	}
	if summary.ByProvider == nil {
		t.Error("ByProvider is nil, want initialized map")
	}
	if summary.ByModel == nil {
		t.Error("ByModel is nil, want initialized map")
	}
	if summary.LocalSavings != 0 {
		t.Errorf("LocalSavings = %v, want 0", summary.LocalSavings)
	}
}

func TestCostSummary_Add(t *testing.T) {
	t.Run("add nil breakdown is no-op", func(t *testing.T) {
		summary := NewCostSummary()
		summary.Add(nil)

		if summary.TotalCost != 0 {
			t.Errorf("TotalCost = %v after adding nil, want 0", summary.TotalCost)
		}
	})

	t.Run("add single breakdown", func(t *testing.T) {
		summary := NewCostSummary()
		breakdown := &CostBreakdown{
			InputCost:    10.0,
			OutputCost:   20.0,
			TotalCost:    30.0,
			InputTokens:  1000,
			OutputTokens: 500,
			Model:        "gpt-4",
			Provider:     ProviderOpenAI,
		}

		summary.Add(breakdown)

		if !floatEquals(summary.TotalCost, 30.0) {
			t.Errorf("TotalCost = %v, want 30.0", summary.TotalCost)
		}
		if !floatEquals(summary.TotalInputCost, 10.0) {
			t.Errorf("TotalInputCost = %v, want 10.0", summary.TotalInputCost)
		}
		if !floatEquals(summary.TotalOutputCost, 20.0) {
			t.Errorf("TotalOutputCost = %v, want 20.0", summary.TotalOutputCost)
		}
		if summary.TotalInputTokens != 1000 {
			t.Errorf("TotalInputTokens = %v, want 1000", summary.TotalInputTokens)
		}
		if summary.TotalOutputTokens != 500 {
			t.Errorf("TotalOutputTokens = %v, want 500", summary.TotalOutputTokens)
		}
		if !floatEquals(summary.ByProvider[ProviderOpenAI], 30.0) {
			t.Errorf("ByProvider[openai] = %v, want 30.0", summary.ByProvider[ProviderOpenAI])
		}
		if !floatEquals(summary.ByModel["gpt-4"], 30.0) {
			t.Errorf("ByModel[gpt-4] = %v, want 30.0", summary.ByModel["gpt-4"])
		}
	})

	t.Run("add multiple breakdowns", func(t *testing.T) {
		summary := NewCostSummary()

		breakdown1 := &CostBreakdown{
			InputCost:    10.0,
			OutputCost:   20.0,
			TotalCost:    30.0,
			InputTokens:  1000,
			OutputTokens: 500,
			Model:        "gpt-4",
			Provider:     ProviderOpenAI,
		}
		breakdown2 := &CostBreakdown{
			InputCost:    5.0,
			OutputCost:   15.0,
			TotalCost:    20.0,
			InputTokens:  500,
			OutputTokens: 300,
			Model:        "claude-3-sonnet",
			Provider:     ProviderAnthropic,
		}
		breakdown3 := &CostBreakdown{
			InputCost:    0,
			OutputCost:   0,
			TotalCost:    0,
			InputTokens:  2000,
			OutputTokens: 1000,
			Model:        "llama3",
			Provider:     ProviderOllama,
		}

		summary.Add(breakdown1)
		summary.Add(breakdown2)
		summary.Add(breakdown3)

		if !floatEquals(summary.TotalCost, 50.0) {
			t.Errorf("TotalCost = %v, want 50.0", summary.TotalCost)
		}
		if !floatEquals(summary.TotalInputCost, 15.0) {
			t.Errorf("TotalInputCost = %v, want 15.0", summary.TotalInputCost)
		}
		if !floatEquals(summary.TotalOutputCost, 35.0) {
			t.Errorf("TotalOutputCost = %v, want 35.0", summary.TotalOutputCost)
		}
		if summary.TotalInputTokens != 3500 {
			t.Errorf("TotalInputTokens = %v, want 3500", summary.TotalInputTokens)
		}
		if summary.TotalOutputTokens != 1800 {
			t.Errorf("TotalOutputTokens = %v, want 1800", summary.TotalOutputTokens)
		}
		if !floatEquals(summary.ByProvider[ProviderOpenAI], 30.0) {
			t.Errorf("ByProvider[openai] = %v, want 30.0", summary.ByProvider[ProviderOpenAI])
		}
		if !floatEquals(summary.ByProvider[ProviderAnthropic], 20.0) {
			t.Errorf("ByProvider[anthropic] = %v, want 20.0", summary.ByProvider[ProviderAnthropic])
		}
		if !floatEquals(summary.ByProvider[ProviderOllama], 0.0) {
			t.Errorf("ByProvider[ollama] = %v, want 0.0", summary.ByProvider[ProviderOllama])
		}
	})

	t.Run("empty provider and model", func(t *testing.T) {
		summary := NewCostSummary()
		breakdown := &CostBreakdown{
			InputCost:    10.0,
			OutputCost:   20.0,
			TotalCost:    30.0,
			InputTokens:  1000,
			OutputTokens: 500,
			Model:        "",
			Provider:     "",
		}

		summary.Add(breakdown)

		if len(summary.ByProvider) != 0 {
			t.Errorf("ByProvider has %d entries, want 0", len(summary.ByProvider))
		}
		if len(summary.ByModel) != 0 {
			t.Errorf("ByModel has %d entries, want 0", len(summary.ByModel))
		}
	})
}

func TestCostSummary_CalculateSavings(t *testing.T) {
	t.Run("nil premium model", func(t *testing.T) {
		summary := NewCostSummary()
		summary.TotalCost = 50.0
		summary.TotalInputTokens = 1000
		summary.TotalOutputTokens = 500

		summary.CalculateSavings(nil)

		if summary.LocalSavings != 0 {
			t.Errorf("LocalSavings = %v, want 0 with nil model", summary.LocalSavings)
		}
	})

	t.Run("savings from using local model", func(t *testing.T) {
		summary := NewCostSummary()
		// Simulate using a local model (no cost)
		summary.TotalCost = 0
		summary.TotalInputTokens = 1000
		summary.TotalOutputTokens = 500

		// Compare against premium model: $15/1K input, $75/1K output
		premiumModel := NewModel("claude-3-opus", "Claude 3 Opus", ProviderAnthropic).
			WithCosts(15.0, 75.0)

		summary.CalculateSavings(premiumModel)

		// Expected premium cost: (1000/1000 * 15) + (500/1000 * 75) = 15 + 37.5 = 52.5
		expectedSavings := 52.5
		if !floatEquals(summary.LocalSavings, expectedSavings) {
			t.Errorf("LocalSavings = %v, want %v", summary.LocalSavings, expectedSavings)
		}
	})

	t.Run("mixed models partial savings", func(t *testing.T) {
		summary := NewCostSummary()
		// Actual cost was $20 (maybe used cheaper models)
		summary.TotalCost = 20.0
		summary.TotalInputTokens = 2000
		summary.TotalOutputTokens = 1000

		// Compare against premium model: $10/1K input, $30/1K output
		premiumModel := NewModel("premium-model", "Premium", ProviderOpenAI).
			WithCosts(10.0, 30.0)

		summary.CalculateSavings(premiumModel)

		// Expected premium cost: (2000/1000 * 10) + (1000/1000 * 30) = 20 + 30 = 50
		// Savings: 50 - 20 = 30
		expectedSavings := 30.0
		if !floatEquals(summary.LocalSavings, expectedSavings) {
			t.Errorf("LocalSavings = %v, want %v", summary.LocalSavings, expectedSavings)
		}
	})

	t.Run("no savings when premium is cheaper", func(t *testing.T) {
		summary := NewCostSummary()
		// Actual cost was high
		summary.TotalCost = 100.0
		summary.TotalInputTokens = 1000
		summary.TotalOutputTokens = 500

		// Premium model is cheaper: $1/1K input, $2/1K output
		cheapPremium := NewModel("cheap-premium", "Cheap Premium", ProviderOpenAI).
			WithCosts(1.0, 2.0)

		summary.CalculateSavings(cheapPremium)

		// Premium cost would be: (1000/1000 * 1) + (500/1000 * 2) = 1 + 1 = 2
		// Since actual cost (100) > premium cost (2), savings should be 0
		if summary.LocalSavings != 0 {
			t.Errorf("LocalSavings = %v, want 0 (no savings)", summary.LocalSavings)
		}
	})
}

func TestCostSummary_InvocationCount(t *testing.T) {
	summary := NewCostSummary()

	if count := summary.InvocationCount(); count != 0 {
		t.Errorf("InvocationCount() = %v for empty summary, want 0", count)
	}

	summary.ByModel["model1"] = 10.0
	if count := summary.InvocationCount(); count != 1 {
		t.Errorf("InvocationCount() = %v after 1 model, want 1", count)
	}

	summary.ByModel["model2"] = 20.0
	summary.ByModel["model3"] = 30.0
	if count := summary.InvocationCount(); count != 3 {
		t.Errorf("InvocationCount() = %v after 3 models, want 3", count)
	}
}

func TestCostSummary_AverageCostPerToken(t *testing.T) {
	t.Run("zero tokens", func(t *testing.T) {
		summary := NewCostSummary()
		avg := summary.AverageCostPerToken()
		if avg != 0 {
			t.Errorf("AverageCostPerToken() = %v for zero tokens, want 0", avg)
		}
	})

	t.Run("calculate average", func(t *testing.T) {
		summary := NewCostSummary()
		summary.TotalCost = 100.0
		summary.TotalInputTokens = 8000
		summary.TotalOutputTokens = 2000

		// Average: 100 / (8000 + 2000) = 100 / 10000 = 0.01
		expectedAvg := 0.01
		avg := summary.AverageCostPerToken()
		if !floatEquals(avg, expectedAvg) {
			t.Errorf("AverageCostPerToken() = %v, want %v", avg, expectedAvg)
		}
	})
}

func TestCostSummary_Clone(t *testing.T) {
	original := NewCostSummary()
	original.TotalCost = 100.0
	original.TotalInputCost = 30.0
	original.TotalOutputCost = 70.0
	original.TotalInputTokens = 5000
	original.TotalOutputTokens = 2000
	original.LocalSavings = 50.0
	original.ByProvider["openai"] = 60.0
	original.ByProvider["anthropic"] = 40.0
	original.ByModel["gpt-4"] = 60.0
	original.ByModel["claude-3"] = 40.0

	clone := original.Clone()

	// Verify all fields are copied
	if clone.TotalCost != original.TotalCost {
		t.Errorf("Clone TotalCost = %v, want %v", clone.TotalCost, original.TotalCost)
	}
	if clone.TotalInputCost != original.TotalInputCost {
		t.Errorf("Clone TotalInputCost = %v, want %v", clone.TotalInputCost, original.TotalInputCost)
	}
	if clone.TotalOutputCost != original.TotalOutputCost {
		t.Errorf("Clone TotalOutputCost = %v, want %v", clone.TotalOutputCost, original.TotalOutputCost)
	}
	if clone.TotalInputTokens != original.TotalInputTokens {
		t.Errorf("Clone TotalInputTokens = %v, want %v", clone.TotalInputTokens, original.TotalInputTokens)
	}
	if clone.TotalOutputTokens != original.TotalOutputTokens {
		t.Errorf("Clone TotalOutputTokens = %v, want %v", clone.TotalOutputTokens, original.TotalOutputTokens)
	}
	if clone.LocalSavings != original.LocalSavings {
		t.Errorf("Clone LocalSavings = %v, want %v", clone.LocalSavings, original.LocalSavings)
	}

	// Verify maps are deep copied
	if clone.ByProvider["openai"] != 60.0 {
		t.Errorf("Clone ByProvider[openai] = %v, want 60.0", clone.ByProvider["openai"])
	}
	if clone.ByModel["gpt-4"] != 60.0 {
		t.Errorf("Clone ByModel[gpt-4] = %v, want 60.0", clone.ByModel["gpt-4"])
	}

	// Verify independence - modifying clone doesn't affect original
	clone.TotalCost = 999.0
	clone.ByProvider["openai"] = 999.0
	clone.ByModel["gpt-4"] = 999.0

	if original.TotalCost != 100.0 {
		t.Error("Modifying clone affected original TotalCost")
	}
	if original.ByProvider["openai"] != 60.0 {
		t.Error("Modifying clone affected original ByProvider")
	}
	if original.ByModel["gpt-4"] != 60.0 {
		t.Error("Modifying clone affected original ByModel")
	}
}

func TestIntegration_FullCostTracking(t *testing.T) {
	// Create models
	gpt4 := NewModel("gpt-4", "GPT-4", ProviderOpenAI).
		WithCosts(30.0, 60.0) // $30 per 1K input, $60 per 1K output

	claude := NewModel("claude-3-sonnet", "Claude 3 Sonnet", ProviderAnthropic).
		WithCosts(3.0, 15.0) // $3 per 1K input, $15 per 1K output

	llama := NewModel("llama3", "Llama 3", ProviderOllama).
		WithCosts(0, 0) // Free local model

	opus := NewModel("claude-3-opus", "Claude 3 Opus", ProviderAnthropic).
		WithCosts(15.0, 75.0) // Premium model for savings comparison

	// Create summary and track costs
	summary := NewCostSummary()

	// Phase 1: Initial generation with GPT-4 (1000 in, 500 out)
	cost1 := CalculateCost(gpt4, 1000, 500)
	summary.Add(cost1)

	// Phase 2: Review with Claude (500 in, 200 out)
	cost2 := CalculateCost(claude, 500, 200)
	summary.Add(cost2)

	// Phase 3: Local processing with Llama (2000 in, 1000 out)
	cost3 := CalculateCost(llama, 2000, 1000)
	summary.Add(cost3)

	// Verify totals
	// GPT-4: (1000/1000 * 30) + (500/1000 * 60) = 30 + 30 = 60
	// Claude: (500/1000 * 3) + (200/1000 * 15) = 1.5 + 3 = 4.5
	// Llama: 0
	// Total: 64.5
	expectedTotal := 64.5

	if !floatEquals(summary.TotalCost, expectedTotal) {
		t.Errorf("TotalCost = %v, want %v", summary.TotalCost, expectedTotal)
	}

	// Verify token totals
	if summary.TotalInputTokens != 3500 {
		t.Errorf("TotalInputTokens = %v, want 3500", summary.TotalInputTokens)
	}
	if summary.TotalOutputTokens != 1700 {
		t.Errorf("TotalOutputTokens = %v, want 1700", summary.TotalOutputTokens)
	}

	// Calculate savings compared to using Opus for everything
	summary.CalculateSavings(opus)

	// If all tokens used Opus:
	// (3500/1000 * 15) + (1700/1000 * 75) = 52.5 + 127.5 = 180
	// Savings: 180 - 64.5 = 115.5
	expectedSavings := 115.5
	if !floatEquals(summary.LocalSavings, expectedSavings) {
		t.Errorf("LocalSavings = %v, want %v", summary.LocalSavings, expectedSavings)
	}

	// Verify provider breakdown
	if !floatEquals(summary.ByProvider[ProviderOpenAI], 60.0) {
		t.Errorf("ByProvider[openai] = %v, want 60.0", summary.ByProvider[ProviderOpenAI])
	}
	if !floatEquals(summary.ByProvider[ProviderAnthropic], 4.5) {
		t.Errorf("ByProvider[anthropic] = %v, want 4.5", summary.ByProvider[ProviderAnthropic])
	}
}
