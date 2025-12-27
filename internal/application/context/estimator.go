// Package context provides application-level context management services.
package context

import (
	"os"
	"strings"
	"unicode/utf8"
)

// Estimator provides token count estimation for text.
// Uses a simple heuristic of ~4 characters per token, which is a reasonable
// approximation for English text with typical LLM tokenizers.
type Estimator struct {
	charsPerToken float64
}

// NewEstimator creates a new token estimator with default settings.
func NewEstimator() *Estimator {
	return &Estimator{
		charsPerToken: 4.0, // Default: ~4 characters per token
	}
}

// NewEstimatorWithRatio creates a new token estimator with a custom chars-per-token ratio.
func NewEstimatorWithRatio(charsPerToken float64) *Estimator {
	if charsPerToken <= 0 {
		charsPerToken = 4.0
	}
	return &Estimator{
		charsPerToken: charsPerToken,
	}
}

// Estimate returns the estimated token count for the given text.
// The estimation is based on character count divided by the chars-per-token ratio.
func (e *Estimator) Estimate(text string) int {
	if text == "" {
		return 0
	}

	// Count characters (excluding excessive whitespace)
	// Use utf8.RuneCountInString for proper Unicode character counting
	text = strings.TrimSpace(text)
	charCount := utf8.RuneCountInString(text)

	// Apply the ratio
	tokens := float64(charCount) / e.charsPerToken

	// Round up to nearest integer
	return int(tokens + 0.5)
}

// EstimateFromFile reads a file and estimates its token count.
// Returns an error if the file cannot be read.
func (e *Estimator) EstimateFromFile(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	return e.Estimate(string(content)), nil
}

// EstimateMultiple estimates total token count for multiple text strings.
func (e *Estimator) EstimateMultiple(texts ...string) int {
	total := 0
	for _, text := range texts {
		total += e.Estimate(text)
	}
	return total
}

// WithinBudget checks if the text fits within a token budget.
func (e *Estimator) WithinBudget(text string, budget int) bool {
	return e.Estimate(text) <= budget
}

// TruncateToFit truncates text to fit within a token budget.
// Returns the truncated text and the estimated token count.
// Uses a simple character-based truncation; may not produce perfect results
// but is fast and good enough for most cases.
func (e *Estimator) TruncateToFit(text string, budget int) (string, int) {
	estimate := e.Estimate(text)
	if estimate <= budget {
		return text, estimate
	}

	// Calculate target character count
	targetChars := int(float64(budget) * e.charsPerToken)

	// Truncate to target length
	if targetChars <= 0 {
		return "", 0
	}

	runes := []rune(text)
	if len(runes) <= targetChars {
		return text, estimate
	}

	truncated := string(runes[:targetChars])
	return truncated, e.Estimate(truncated)
}

// EstimateLines estimates token count for a specific number of lines from text.
// Useful for processing large files in chunks.
func (e *Estimator) EstimateLines(text string, numLines int) int {
	if numLines <= 0 || text == "" {
		return 0
	}

	lines := strings.Split(text, "\n")
	if numLines > len(lines) {
		numLines = len(lines)
	}

	subset := strings.Join(lines[:numLines], "\n")
	return e.Estimate(subset)
}
