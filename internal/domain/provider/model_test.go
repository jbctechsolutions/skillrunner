package provider

import (
	"math"
	"testing"
)

const floatTolerance = 1e-9

// floatEquals compares two floats with an optional tolerance.
// If tolerance is not provided (called with 2 args), uses 1e-9 as default.
// This function is used across the package's test files.
func floatEquals(a, b float64, tolerance ...float64) bool {
	tol := floatTolerance
	if len(tolerance) > 0 {
		tol = tolerance[0]
	}
	return math.Abs(a-b) < tol
}

func TestNewModel(t *testing.T) {
	t.Run("creates model with required fields", func(t *testing.T) {
		m := NewModel("gpt-4", "GPT-4", ProviderOpenAI)

		if m.ID != "gpt-4" {
			t.Errorf("expected ID 'gpt-4', got %q", m.ID)
		}
		if m.Name != "GPT-4" {
			t.Errorf("expected Name 'GPT-4', got %q", m.Name)
		}
		if m.Provider != ProviderOpenAI {
			t.Errorf("expected Provider 'openai', got %q", m.Provider)
		}
	})

	t.Run("initializes with empty capabilities", func(t *testing.T) {
		m := NewModel("gpt-4", "GPT-4", ProviderOpenAI)

		if m.Capabilities == nil {
			t.Error("expected Capabilities to be initialized, got nil")
		}
		if len(m.Capabilities) != 0 {
			t.Errorf("expected 0 capabilities, got %d", len(m.Capabilities))
		}
	})

	t.Run("defaults to balanced tier", func(t *testing.T) {
		m := NewModel("gpt-4", "GPT-4", ProviderOpenAI)

		if m.Tier != TierBalanced {
			t.Errorf("expected Tier 'balanced', got %q", m.Tier)
		}
	})
}

func TestWithContextWindow(t *testing.T) {
	m := NewModel("gpt-4", "GPT-4", ProviderOpenAI).
		WithContextWindow(128000)

	if m.ContextWindow != 128000 {
		t.Errorf("expected ContextWindow 128000, got %d", m.ContextWindow)
	}
}

func TestWithCosts(t *testing.T) {
	m := NewModel("gpt-4", "GPT-4", ProviderOpenAI).
		WithCosts(0.03, 0.06)

	if m.InputCostPer1K != 0.03 {
		t.Errorf("expected InputCostPer1K 0.03, got %f", m.InputCostPer1K)
	}
	if m.OutputCostPer1K != 0.06 {
		t.Errorf("expected OutputCostPer1K 0.06, got %f", m.OutputCostPer1K)
	}

	// Check per-token prices are also set
	expectedInputPerToken := 0.03 / 1000.0
	if !floatEquals(m.InputPricePerToken, expectedInputPerToken, floatTolerance) {
		t.Errorf("expected InputPricePerToken %e, got %e", expectedInputPerToken, m.InputPricePerToken)
	}

	expectedOutputPerToken := 0.06 / 1000.0
	if !floatEquals(m.OutputPricePerToken, expectedOutputPerToken, floatTolerance) {
		t.Errorf("expected OutputPricePerToken %e, got %e", expectedOutputPerToken, m.OutputPricePerToken)
	}
}

func TestWithCapabilities(t *testing.T) {
	t.Run("sets multiple capabilities", func(t *testing.T) {
		m := NewModel("gpt-4-vision", "GPT-4 Vision", ProviderOpenAI).
			WithCapabilities(CapabilityVision, CapabilityFunctionCalling, CapabilityStreaming)

		if len(m.Capabilities) != 3 {
			t.Errorf("expected 3 capabilities, got %d", len(m.Capabilities))
		}
	})

	t.Run("replaces existing capabilities", func(t *testing.T) {
		m := NewModel("gpt-4", "GPT-4", ProviderOpenAI).
			WithCapabilities(CapabilityVision).
			WithCapabilities(CapabilityStreaming)

		if len(m.Capabilities) != 1 {
			t.Errorf("expected 1 capability after replacement, got %d", len(m.Capabilities))
		}
		if m.Capabilities[0] != CapabilityStreaming {
			t.Errorf("expected capability 'streaming', got %q", m.Capabilities[0])
		}
	})

	t.Run("creates a copy of input slice", func(t *testing.T) {
		caps := []string{CapabilityVision}
		m := NewModel("gpt-4", "GPT-4", ProviderOpenAI).
			WithCapabilities(caps...)

		caps[0] = "modified"
		if m.Capabilities[0] == "modified" {
			t.Error("expected Capabilities to be independent of input slice")
		}
	})
}

func TestWithTier(t *testing.T) {
	testCases := []struct {
		tier     AgentTier
		expected AgentTier
	}{
		{TierCheap, TierCheap},
		{TierBalanced, TierBalanced},
		{TierPremium, TierPremium},
	}

	for _, tc := range testCases {
		t.Run(string(tc.tier), func(t *testing.T) {
			m := NewModel("gpt-4", "GPT-4", ProviderOpenAI).
				WithTier(tc.tier)

			if m.Tier != tc.expected {
				t.Errorf("expected Tier %q, got %q", tc.expected, m.Tier)
			}
		})
	}
}

func TestHasCapability(t *testing.T) {
	m := NewModel("gpt-4", "GPT-4", ProviderOpenAI).
		WithCapabilities(CapabilityVision, CapabilityStreaming)

	t.Run("returns true for existing capability", func(t *testing.T) {
		if !m.HasCapability(CapabilityVision) {
			t.Error("expected HasCapability to return true for 'vision'")
		}
		if !m.HasCapability(CapabilityStreaming) {
			t.Error("expected HasCapability to return true for 'streaming'")
		}
	})

	t.Run("returns false for missing capability", func(t *testing.T) {
		if m.HasCapability(CapabilityFunctionCalling) {
			t.Error("expected HasCapability to return false for 'function_calling'")
		}
	})

	t.Run("returns false for empty model", func(t *testing.T) {
		empty := NewModel("test", "Test", ProviderOpenAI)
		if empty.HasCapability(CapabilityVision) {
			t.Error("expected HasCapability to return false for empty model")
		}
	})
}

func TestEstimateCost(t *testing.T) {
	// Model costs: $0.03 per 1K input, $0.06 per 1K output
	m := NewModel("gpt-4", "GPT-4", ProviderOpenAI).
		WithCosts(0.03, 0.06)

	t.Run("calculates cost for given tokens", func(t *testing.T) {
		// 1000 input tokens = $0.03, 500 output tokens = $0.03
		cost := m.EstimateCost(1000, 500)
		expected := 0.06

		if !floatEquals(cost, expected, floatTolerance) {
			t.Errorf("expected cost %f, got %f", expected, cost)
		}
	})

	t.Run("handles zero tokens", func(t *testing.T) {
		cost := m.EstimateCost(0, 0)
		if cost != 0 {
			t.Errorf("expected cost 0 for zero tokens, got %f", cost)
		}
	})

	t.Run("handles large token counts", func(t *testing.T) {
		// 1M input tokens = $30, 500K output tokens = $30
		cost := m.EstimateCost(1000000, 500000)
		expected := 60.0

		if !floatEquals(cost, expected, floatTolerance) {
			t.Errorf("expected cost %f, got %f", expected, cost)
		}
	})

	t.Run("handles fractional calculations", func(t *testing.T) {
		// 100 input tokens = $0.003, 100 output tokens = $0.006
		cost := m.EstimateCost(100, 100)
		expected := 0.009

		if !floatEquals(cost, expected, floatTolerance) {
			t.Errorf("expected cost %f, got %f", expected, cost)
		}
	})
}

func TestIsLocal(t *testing.T) {
	testCases := []struct {
		provider string
		expected bool
	}{
		{ProviderOllama, true},
		{ProviderAnthropic, false},
		{ProviderOpenAI, false},
		{ProviderGroq, false},
		{"unknown", false},
	}

	for _, tc := range testCases {
		t.Run(tc.provider, func(t *testing.T) {
			m := NewModel("test", "Test", tc.provider)
			if m.IsLocal() != tc.expected {
				t.Errorf("expected IsLocal() = %v for provider %q", tc.expected, tc.provider)
			}
		})
	}
}

func TestFluentChaining(t *testing.T) {
	m := NewModel("llama2", "Llama 2", ProviderOllama).
		WithContextWindow(4096).
		WithCosts(0, 0).
		WithCapabilities(CapabilityStreaming).
		WithTier(TierCheap)

	if m.ID != "llama2" {
		t.Errorf("expected ID 'llama2', got %q", m.ID)
	}
	if m.Name != "Llama 2" {
		t.Errorf("expected Name 'Llama 2', got %q", m.Name)
	}
	if m.Provider != ProviderOllama {
		t.Errorf("expected Provider 'ollama', got %q", m.Provider)
	}
	if m.ContextWindow != 4096 {
		t.Errorf("expected ContextWindow 4096, got %d", m.ContextWindow)
	}
	if m.InputCostPer1K != 0 {
		t.Errorf("expected InputCostPer1K 0, got %f", m.InputCostPer1K)
	}
	if m.OutputCostPer1K != 0 {
		t.Errorf("expected OutputCostPer1K 0, got %f", m.OutputCostPer1K)
	}
	if !m.HasCapability(CapabilityStreaming) {
		t.Error("expected model to have streaming capability")
	}
	if m.Tier != TierCheap {
		t.Errorf("expected Tier 'cheap', got %q", m.Tier)
	}
	if !m.IsLocal() {
		t.Error("expected Ollama model to be local")
	}
}
