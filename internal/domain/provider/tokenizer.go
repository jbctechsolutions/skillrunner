// Package provider contains domain types for AI provider and model management.
package provider

// TokenEstimator provides token count estimation for text content.
// This interface allows for different tokenization strategies to be used
// (e.g., tiktoken, simple heuristics) without coupling the domain to a specific implementation.
type TokenEstimator interface {
	// CountTokens returns the estimated token count for the given text.
	CountTokens(text string) int
}
