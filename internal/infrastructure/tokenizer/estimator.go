// Package tokenizer provides token counting infrastructure using tiktoken.
// It implements the domain TokenEstimator interface for accurate token estimation.
package tokenizer

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"

	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
)

// Estimator provides token counting using tiktoken-go.
// It uses the cl100k_base encoding which is compatible with GPT-4 and
// provides a reasonable approximation for Claude models.
type Estimator struct {
	encoding *tiktoken.Tiktoken
	mu       sync.RWMutex
}

// Ensure Estimator implements provider.TokenEstimator.
var _ provider.TokenEstimator = (*Estimator)(nil)

// NewEstimator creates a new token estimator using cl100k_base encoding.
// This encoding is used by GPT-4 and provides a reasonable approximation
// for most modern LLMs including Claude.
func NewEstimator() (*Estimator, error) {
	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, err
	}

	return &Estimator{
		encoding: encoding,
	}, nil
}

// CountTokens returns the token count for the given text.
// This method is thread-safe.
func (e *Estimator) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	tokens := e.encoding.Encode(text, nil, nil)
	return len(tokens)
}

// EstimateOutputTokens provides a heuristic estimate for expected output tokens.
// It uses a fraction of the maximum allowed tokens, defaulting to 50%.
func EstimateOutputTokens(maxTokens int, fraction float64) int {
	if maxTokens <= 0 {
		return 500 // Default estimate
	}
	if fraction <= 0 || fraction > 1 {
		fraction = 0.5
	}
	return int(float64(maxTokens) * fraction)
}

// SimpleEstimator provides a simple heuristic-based token estimator
// that doesn't require external dependencies. Uses ~4 characters per token.
type SimpleEstimator struct{}

// Ensure SimpleEstimator implements provider.TokenEstimator.
var _ provider.TokenEstimator = (*SimpleEstimator)(nil)

// NewSimpleEstimator creates a new simple token estimator.
// This is useful for testing or when tiktoken is not available.
func NewSimpleEstimator() *SimpleEstimator {
	return &SimpleEstimator{}
}

// CountTokens returns an estimated token count using ~4 characters per token heuristic.
func (e *SimpleEstimator) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	// Approximate: ~4 characters per token for English text
	return (len(text) + 3) / 4
}
