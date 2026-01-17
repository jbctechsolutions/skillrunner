// Package provider contains domain types for AI provider and model management.
package provider

// DefaultModelPricing returns the default pricing for well-known models.
// Prices are per 1000 tokens in USD.
// To convert from provider pricing (typically per million tokens):
//
//	rate_per_1k = price_per_million / 1000
//
// Example: Claude Sonnet at $3/MTok input = 0.003 per 1K tokens
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
		// Opus 4.5: $5/MTok input, $25/MTok output
		{ModelID: "claude-opus-4-5-20251101", Provider: ProviderAnthropic, InputRate: 0.005, OutputRate: 0.025, IsLocal: false},
		// Sonnet 4.5: $3/MTok input, $15/MTok output
		{ModelID: "claude-sonnet-4-5-20251101", Provider: ProviderAnthropic, InputRate: 0.003, OutputRate: 0.015, IsLocal: false},
		// Haiku 4.5: $1/MTok input, $5/MTok output
		{ModelID: "claude-haiku-4-5-20251101", Provider: ProviderAnthropic, InputRate: 0.001, OutputRate: 0.005, IsLocal: false},

		// Claude 4 Series
		// Opus 4: $15/MTok input, $75/MTok output
		{ModelID: "claude-opus-4-20250514", Provider: ProviderAnthropic, InputRate: 0.015, OutputRate: 0.075, IsLocal: false},
		// Sonnet 4: $3/MTok input, $15/MTok output
		{ModelID: "claude-sonnet-4-20250514", Provider: ProviderAnthropic, InputRate: 0.003, OutputRate: 0.015, IsLocal: false},

		// Claude 3.5 Series (still widely used)
		// Sonnet 3.5: $3/MTok input, $15/MTok output
		{ModelID: "claude-3-5-sonnet-20241022", Provider: ProviderAnthropic, InputRate: 0.003, OutputRate: 0.015, IsLocal: false},
		{ModelID: "claude-3-5-sonnet-latest", Provider: ProviderAnthropic, InputRate: 0.003, OutputRate: 0.015, IsLocal: false},
		// Haiku 3.5: $0.80/MTok input, $4/MTok output
		{ModelID: "claude-3-5-haiku-20241022", Provider: ProviderAnthropic, InputRate: 0.0008, OutputRate: 0.004, IsLocal: false},
		{ModelID: "claude-3-5-haiku-latest", Provider: ProviderAnthropic, InputRate: 0.0008, OutputRate: 0.004, IsLocal: false},

		// Claude 3 Series (legacy)
		// Opus 3: $15/MTok input, $75/MTok output
		{ModelID: "claude-3-opus-20240229", Provider: ProviderAnthropic, InputRate: 0.015, OutputRate: 0.075, IsLocal: false},
		// Sonnet 3: $3/MTok input, $15/MTok output
		{ModelID: "claude-3-sonnet-20240229", Provider: ProviderAnthropic, InputRate: 0.003, OutputRate: 0.015, IsLocal: false},
		// Haiku 3: $0.25/MTok input, $1.25/MTok output
		{ModelID: "claude-3-haiku-20240307", Provider: ProviderAnthropic, InputRate: 0.00025, OutputRate: 0.00125, IsLocal: false},

		// ============================================
		// OpenAI GPT models
		// https://openai.com/api/pricing/
		// ============================================

		// GPT-4o Series
		// GPT-4o: $2.50/MTok input, $10/MTok output
		{ModelID: "gpt-4o", Provider: ProviderOpenAI, InputRate: 0.0025, OutputRate: 0.01, IsLocal: false},
		{ModelID: "gpt-4o-2024-11-20", Provider: ProviderOpenAI, InputRate: 0.0025, OutputRate: 0.01, IsLocal: false},
		// ChatGPT-4o: $5/MTok input, $15/MTok output
		{ModelID: "chatgpt-4o-latest", Provider: ProviderOpenAI, InputRate: 0.005, OutputRate: 0.015, IsLocal: false},
		// GPT-4o mini: $0.15/MTok input, $0.60/MTok output
		{ModelID: "gpt-4o-mini", Provider: ProviderOpenAI, InputRate: 0.00015, OutputRate: 0.0006, IsLocal: false},
		{ModelID: "gpt-4o-mini-2024-07-18", Provider: ProviderOpenAI, InputRate: 0.00015, OutputRate: 0.0006, IsLocal: false},

		// O-Series (reasoning models)
		// o1: $15/MTok input, $60/MTok output
		{ModelID: "o1", Provider: ProviderOpenAI, InputRate: 0.015, OutputRate: 0.06, IsLocal: false},
		{ModelID: "o1-preview", Provider: ProviderOpenAI, InputRate: 0.015, OutputRate: 0.06, IsLocal: false},
		// o1-mini: $3/MTok input, $12/MTok output
		{ModelID: "o1-mini", Provider: ProviderOpenAI, InputRate: 0.003, OutputRate: 0.012, IsLocal: false},
		// o3-mini: $1.10/MTok input, $4.40/MTok output
		{ModelID: "o3-mini", Provider: ProviderOpenAI, InputRate: 0.0011, OutputRate: 0.0044, IsLocal: false},

		// GPT-4 Legacy
		// GPT-4 Turbo: $10/MTok input, $30/MTok output
		{ModelID: "gpt-4-turbo", Provider: ProviderOpenAI, InputRate: 0.01, OutputRate: 0.03, IsLocal: false},
		{ModelID: "gpt-4-turbo-2024-04-09", Provider: ProviderOpenAI, InputRate: 0.01, OutputRate: 0.03, IsLocal: false},
		// GPT-4: $30/MTok input, $60/MTok output
		{ModelID: "gpt-4", Provider: ProviderOpenAI, InputRate: 0.03, OutputRate: 0.06, IsLocal: false},
		// GPT-3.5 Turbo: $0.50/MTok input, $1.50/MTok output
		{ModelID: "gpt-3.5-turbo", Provider: ProviderOpenAI, InputRate: 0.0005, OutputRate: 0.0015, IsLocal: false},

		// ============================================
		// Groq models
		// https://groq.com/pricing/
		// ============================================

		// Llama 4 Series
		// Llama 4 Scout: $0.11/MTok input, $0.34/MTok output
		{ModelID: "llama-4-scout-17b-16e-instruct", Provider: ProviderGroq, InputRate: 0.00011, OutputRate: 0.00034, IsLocal: false},

		// Llama 3.3 Series
		// Llama 3.3 70B: $0.59/MTok input, $0.79/MTok output
		{ModelID: "llama-3.3-70b-versatile", Provider: ProviderGroq, InputRate: 0.00059, OutputRate: 0.00079, IsLocal: false},
		{ModelID: "llama-3.3-70b-specdec", Provider: ProviderGroq, InputRate: 0.00059, OutputRate: 0.00099, IsLocal: false},

		// Llama 3.1 Series
		{ModelID: "llama-3.1-70b-versatile", Provider: ProviderGroq, InputRate: 0.00059, OutputRate: 0.00079, IsLocal: false},
		// Llama 3.1 8B: $0.05/MTok input, $0.08/MTok output
		{ModelID: "llama-3.1-8b-instant", Provider: ProviderGroq, InputRate: 0.00005, OutputRate: 0.00008, IsLocal: false},

		// Llama 3 Series (legacy)
		{ModelID: "llama3-70b-8192", Provider: ProviderGroq, InputRate: 0.00059, OutputRate: 0.00079, IsLocal: false},
		{ModelID: "llama3-8b-8192", Provider: ProviderGroq, InputRate: 0.00005, OutputRate: 0.00008, IsLocal: false},

		// Other Groq models
		// Mixtral: $0.24/MTok input, $0.24/MTok output
		{ModelID: "mixtral-8x7b-32768", Provider: ProviderGroq, InputRate: 0.00024, OutputRate: 0.00024, IsLocal: false},
		// Gemma2: $0.20/MTok input, $0.20/MTok output
		{ModelID: "gemma2-9b-it", Provider: ProviderGroq, InputRate: 0.0002, OutputRate: 0.0002, IsLocal: false},
		// DeepSeek R1: $0.75/MTok input, $0.99/MTok output
		{ModelID: "deepseek-r1-distill-llama-70b", Provider: ProviderGroq, InputRate: 0.00075, OutputRate: 0.00099, IsLocal: false},

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
