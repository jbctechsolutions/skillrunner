// Package provider contains domain types for AI provider and model management.
package provider

// DefaultModelPricing returns the default pricing for well-known models.
// Prices are per 1000 tokens in USD (i.e., per-million-token price divided by 1000).
// Last updated: January 2026
// Sources:
//   - Anthropic: https://docs.anthropic.com/en/docs/about-claude/models
//   - OpenAI: https://openai.com/api/pricing/
//   - Groq: https://groq.com/pricing/
func DefaultModelPricing() []ModelCostRate {
	return []ModelCostRate{
		// ============================================
		// Anthropic Claude models
		// https://docs.anthropic.com/en/docs/about-claude/models
		// ============================================

		// Claude 4.5 Series (Latest - November 2025)
		{ModelID: "claude-opus-4-5-20251101", Provider: ProviderAnthropic, InputRate: 5.0, OutputRate: 25.0, IsLocal: false},
		{ModelID: "claude-sonnet-4-5-20251101", Provider: ProviderAnthropic, InputRate: 3.0, OutputRate: 15.0, IsLocal: false},
		{ModelID: "claude-haiku-4-5-20251101", Provider: ProviderAnthropic, InputRate: 1.0, OutputRate: 5.0, IsLocal: false},

		// Claude 4 Series
		{ModelID: "claude-opus-4-20250514", Provider: ProviderAnthropic, InputRate: 15.0, OutputRate: 75.0, IsLocal: false},
		{ModelID: "claude-sonnet-4-20250514", Provider: ProviderAnthropic, InputRate: 3.0, OutputRate: 15.0, IsLocal: false},

		// Claude 3.5 Series (still widely used)
		{ModelID: "claude-3-5-sonnet-20241022", Provider: ProviderAnthropic, InputRate: 3.0, OutputRate: 15.0, IsLocal: false},
		{ModelID: "claude-3-5-sonnet-latest", Provider: ProviderAnthropic, InputRate: 3.0, OutputRate: 15.0, IsLocal: false},
		{ModelID: "claude-3-5-haiku-20241022", Provider: ProviderAnthropic, InputRate: 0.80, OutputRate: 4.0, IsLocal: false},
		{ModelID: "claude-3-5-haiku-latest", Provider: ProviderAnthropic, InputRate: 0.80, OutputRate: 4.0, IsLocal: false},

		// Claude 3 Series (legacy)
		{ModelID: "claude-3-opus-20240229", Provider: ProviderAnthropic, InputRate: 15.0, OutputRate: 75.0, IsLocal: false},
		{ModelID: "claude-3-sonnet-20240229", Provider: ProviderAnthropic, InputRate: 3.0, OutputRate: 15.0, IsLocal: false},
		{ModelID: "claude-3-haiku-20240307", Provider: ProviderAnthropic, InputRate: 0.25, OutputRate: 1.25, IsLocal: false},

		// ============================================
		// OpenAI GPT models
		// https://openai.com/api/pricing/
		// ============================================

		// GPT-4o Series
		{ModelID: "gpt-4o", Provider: ProviderOpenAI, InputRate: 2.50, OutputRate: 10.0, IsLocal: false},
		{ModelID: "gpt-4o-2024-11-20", Provider: ProviderOpenAI, InputRate: 2.50, OutputRate: 10.0, IsLocal: false},
		{ModelID: "chatgpt-4o-latest", Provider: ProviderOpenAI, InputRate: 5.0, OutputRate: 15.0, IsLocal: false},
		{ModelID: "gpt-4o-mini", Provider: ProviderOpenAI, InputRate: 0.15, OutputRate: 0.60, IsLocal: false},
		{ModelID: "gpt-4o-mini-2024-07-18", Provider: ProviderOpenAI, InputRate: 0.15, OutputRate: 0.60, IsLocal: false},

		// O-Series (reasoning models)
		{ModelID: "o1", Provider: ProviderOpenAI, InputRate: 15.0, OutputRate: 60.0, IsLocal: false},
		{ModelID: "o1-preview", Provider: ProviderOpenAI, InputRate: 15.0, OutputRate: 60.0, IsLocal: false},
		{ModelID: "o1-mini", Provider: ProviderOpenAI, InputRate: 3.0, OutputRate: 12.0, IsLocal: false},
		{ModelID: "o3-mini", Provider: ProviderOpenAI, InputRate: 1.10, OutputRate: 4.40, IsLocal: false},

		// GPT-4 Legacy
		{ModelID: "gpt-4-turbo", Provider: ProviderOpenAI, InputRate: 10.0, OutputRate: 30.0, IsLocal: false},
		{ModelID: "gpt-4-turbo-2024-04-09", Provider: ProviderOpenAI, InputRate: 10.0, OutputRate: 30.0, IsLocal: false},
		{ModelID: "gpt-4", Provider: ProviderOpenAI, InputRate: 30.0, OutputRate: 60.0, IsLocal: false},
		{ModelID: "gpt-3.5-turbo", Provider: ProviderOpenAI, InputRate: 0.50, OutputRate: 1.50, IsLocal: false},

		// ============================================
		// Groq models
		// https://groq.com/pricing/
		// ============================================

		// Llama 4 Series
		{ModelID: "llama-4-scout-17b-16e-instruct", Provider: ProviderGroq, InputRate: 0.11, OutputRate: 0.34, IsLocal: false},

		// Llama 3.3 Series
		{ModelID: "llama-3.3-70b-versatile", Provider: ProviderGroq, InputRate: 0.59, OutputRate: 0.79, IsLocal: false},
		{ModelID: "llama-3.3-70b-specdec", Provider: ProviderGroq, InputRate: 0.59, OutputRate: 0.99, IsLocal: false},

		// Llama 3.1 Series
		{ModelID: "llama-3.1-70b-versatile", Provider: ProviderGroq, InputRate: 0.59, OutputRate: 0.79, IsLocal: false},
		{ModelID: "llama-3.1-8b-instant", Provider: ProviderGroq, InputRate: 0.05, OutputRate: 0.08, IsLocal: false},

		// Llama 3 Series (legacy)
		{ModelID: "llama3-70b-8192", Provider: ProviderGroq, InputRate: 0.59, OutputRate: 0.79, IsLocal: false},
		{ModelID: "llama3-8b-8192", Provider: ProviderGroq, InputRate: 0.05, OutputRate: 0.08, IsLocal: false},

		// Other Groq models
		{ModelID: "mixtral-8x7b-32768", Provider: ProviderGroq, InputRate: 0.24, OutputRate: 0.24, IsLocal: false},
		{ModelID: "gemma2-9b-it", Provider: ProviderGroq, InputRate: 0.20, OutputRate: 0.20, IsLocal: false},
		{ModelID: "deepseek-r1-distill-llama-70b", Provider: ProviderGroq, InputRate: 0.75, OutputRate: 0.99, IsLocal: false},

		// ============================================
		// Ollama models (local, zero cost)
		// All local models are free to run
		// ============================================

		// Llama 4 Series
		{ModelID: "llama4:scout", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "llama4:maverick", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// Llama 3.2 Series
		{ModelID: "llama3.2:1b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "llama3.2:3b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "llama3.2:8b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "llama3.2:70b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// Llama 3.1 Series
		{ModelID: "llama3.1:8b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "llama3.1:70b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "llama3.1:405b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// Llama 3 Series
		{ModelID: "llama3:8b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "llama3:70b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// Qwen 2.5 Series
		{ModelID: "qwen2.5:3b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "qwen2.5:7b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "qwen2.5:14b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "qwen2.5:32b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "qwen2.5:72b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "qwen2.5-coder:7b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "qwen2.5-coder:32b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// DeepSeek Series
		{ModelID: "deepseek-r1:7b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "deepseek-r1:14b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "deepseek-r1:32b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "deepseek-r1:70b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "deepseek-coder:6.7b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "deepseek-coder:33b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// Mistral Series
		{ModelID: "mistral:7b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "mixtral:8x7b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "mistral-small:22b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// CodeLlama Series
		{ModelID: "codellama:7b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "codellama:13b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "codellama:34b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "codellama:70b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// Phi Series
		{ModelID: "phi3:mini", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "phi3:medium", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "phi4:14b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},

		// Gemma Series
		{ModelID: "gemma:2b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "gemma:7b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "gemma2:9b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
		{ModelID: "gemma2:27b", Provider: ProviderOllama, InputRate: 0, OutputRate: 0, IsLocal: true},
	}
}

// PopulateCostCalculator adds default model pricing to a CostCalculator.
func PopulateCostCalculator(calc *CostCalculator) {
	if calc == nil {
		return
	}

	for _, rate := range DefaultModelPricing() {
		calc.RegisterModelWithProvider(rate.ModelID, rate.Provider, rate.InputRate, rate.OutputRate)
	}
}
