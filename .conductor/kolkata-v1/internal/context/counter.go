package context

import (
	"regexp"
	"strings"
)

// TokenCounter provides token counting functionality for different models
type TokenCounter struct {
	// Model-specific token counting can be added here
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{}
}

// Count estimates the number of tokens in the given text
// Uses a simple heuristic: ~4 characters per token for English text
// This is accurate enough for most use cases and avoids external dependencies
func (tc *TokenCounter) Count(text string) int {
	if text == "" {
		return 0
	}

	// Simple estimation: characters / 4
	// This works reasonably well for English and code
	return len(text) / 4
}

// CountWords counts the number of words in text
// Useful for extractive summarization
func (tc *TokenCounter) CountWords(text string) int {
	// Split by whitespace and count
	words := strings.Fields(text)
	return len(words)
}

// CountSentences counts the number of sentences in text
// Used for extractive summarization
func (tc *TokenCounter) CountSentences(text string) int {
	// Simple sentence detection: split by . ! ?
	// This is a heuristic and may not be perfect for all cases
	sentenceEnders := regexp.MustCompile(`[.!?]+`)
	sentences := sentenceEnders.Split(text, -1)

	// Count non-empty sentences
	count := 0
	for _, s := range sentences {
		if strings.TrimSpace(s) != "" {
			count++
		}
	}
	return count
}

// CountCodeTokens estimates tokens for code
// Code typically has more tokens per character due to symbols
func (tc *TokenCounter) CountCodeTokens(code string) int {
	if code == "" {
		return 0
	}

	// For code, use ~3 characters per token (more dense than prose)
	return len(code) / 3
}

// EstimateMarkdownTokens estimates tokens for markdown text
// Accounts for markdown syntax
func (tc *TokenCounter) EstimateMarkdownTokens(markdown string) int {
	if markdown == "" {
		return 0
	}

	// Markdown has extra syntax, so slightly more tokens per character
	// Use ~3.5 characters per token
	return len(markdown) * 2 / 7 // Approximately len/3.5
}

// SplitSentences splits text into sentences
// Returns a slice of sentences
func (tc *TokenCounter) SplitSentences(text string) []string {
	// Simple sentence splitting by . ! ?
	// Keep the punctuation with the sentence
	sentencePattern := regexp.MustCompile(`[^.!?]+[.!?]+`)
	sentences := sentencePattern.FindAllString(text, -1)

	if len(sentences) == 0 && text != "" {
		// No sentence terminators found, return the whole text
		return []string{text}
	}

	// Trim whitespace from each sentence
	result := make([]string, 0, len(sentences))
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// EstimateTokensForModel estimates tokens for a specific model
// Different models have different tokenization schemes
func (tc *TokenCounter) EstimateTokensForModel(text string, modelName string) int {
	if text == "" {
		return 0
	}

	// For now, use simple heuristics
	// In the future, could integrate tiktoken-go for GPT models
	// or other model-specific tokenizers

	switch {
	case strings.Contains(modelName, "gpt"):
		// GPT models: ~4 chars per token
		return len(text) / 4
	case strings.Contains(modelName, "claude"):
		// Claude models: ~3.5 chars per token (slightly more efficient)
		return len(text) * 2 / 7
	case strings.Contains(modelName, "qwen"):
		// Qwen models: ~4 chars per token
		return len(text) / 4
	case strings.Contains(modelName, "deepseek"):
		// DeepSeek models: ~4 chars per token
		return len(text) / 4
	default:
		// Default: 4 chars per token
		return len(text) / 4
	}
}

// TokenBudget represents a token budget for context management
type TokenBudget struct {
	Total    int // Total tokens available
	Used     int // Tokens already used
	Reserved int // Tokens reserved for output
}

// NewTokenBudget creates a new token budget
func NewTokenBudget(contextWindow int, outputTokens int) *TokenBudget {
	return &TokenBudget{
		Total:    contextWindow,
		Used:     0,
		Reserved: outputTokens,
	}
}

// Available returns the number of tokens available for input
func (tb *TokenBudget) Available() int {
	return tb.Total - tb.Used - tb.Reserved
}

// CanFit checks if the given number of tokens can fit
func (tb *TokenBudget) CanFit(tokens int) bool {
	return tokens <= tb.Available()
}

// Use marks tokens as used
func (tb *TokenBudget) Use(tokens int) bool {
	if !tb.CanFit(tokens) {
		return false
	}
	tb.Used += tokens
	return true
}

// Reset resets the used tokens
func (tb *TokenBudget) Reset() {
	tb.Used = 0
}

// Remaining returns the remaining tokens
func (tb *TokenBudget) Remaining() int {
	return tb.Available()
}
