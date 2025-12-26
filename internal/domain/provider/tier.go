// Package provider contains domain types for AI provider management.
package provider

import (
	"fmt"
	"strings"
)

// AgentTier represents the cost/capability tier of an AI agent.
type AgentTier string

const (
	// TierCheap represents local/free models.
	TierCheap AgentTier = "cheap"
	// TierBalanced represents mid-tier cloud models.
	TierBalanced AgentTier = "balanced"
	// TierPremium represents high-end cloud models.
	TierPremium AgentTier = "premium"
)

// String returns the string representation of the tier.
func (t AgentTier) String() string {
	return string(t)
}

// IsValid returns true if the tier is a recognized value.
func (t AgentTier) IsValid() bool {
	switch t {
	case TierCheap, TierBalanced, TierPremium:
		return true
	default:
		return false
	}
}

// Order returns the tier priority where lower values indicate cheaper tiers.
// Returns -1 for invalid tiers.
func (t AgentTier) Order() int {
	switch t {
	case TierCheap:
		return 0
	case TierBalanced:
		return 1
	case TierPremium:
		return 2
	default:
		return -1
	}
}

// ParseAgentTier parses a string into an AgentTier.
// The parsing is case-insensitive.
func ParseAgentTier(s string) (AgentTier, error) {
	tier := AgentTier(strings.ToLower(strings.TrimSpace(s)))
	if !tier.IsValid() {
		return "", fmt.Errorf("invalid agent tier: %q", s)
	}
	return tier, nil
}

// CompareTiers compares two tiers by their order.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
// Invalid tiers are considered less than valid tiers.
func CompareTiers(a, b AgentTier) int {
	orderA := a.Order()
	orderB := b.Order()

	if orderA < orderB {
		return -1
	}
	if orderA > orderB {
		return 1
	}
	return 0
}
