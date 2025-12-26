package provider

import (
	"testing"
)

func TestAgentTier_String(t *testing.T) {
	tests := []struct {
		tier     AgentTier
		expected string
	}{
		{TierCheap, "cheap"},
		{TierBalanced, "balanced"},
		{TierPremium, "premium"},
		{AgentTier("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			if got := tt.tier.String(); got != tt.expected {
				t.Errorf("AgentTier.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgentTier_IsValid(t *testing.T) {
	tests := []struct {
		tier     AgentTier
		expected bool
	}{
		{TierCheap, true},
		{TierBalanced, true},
		{TierPremium, true},
		{AgentTier("unknown"), false},
		{AgentTier(""), false},
		{AgentTier("CHEAP"), false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			if got := tt.tier.IsValid(); got != tt.expected {
				t.Errorf("AgentTier.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgentTier_Order(t *testing.T) {
	tests := []struct {
		tier     AgentTier
		expected int
	}{
		{TierCheap, 0},
		{TierBalanced, 1},
		{TierPremium, 2},
		{AgentTier("unknown"), -1},
		{AgentTier(""), -1},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			if got := tt.tier.Order(); got != tt.expected {
				t.Errorf("AgentTier.Order() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseAgentTier(t *testing.T) {
	tests := []struct {
		input       string
		expected    AgentTier
		expectError bool
	}{
		{"cheap", TierCheap, false},
		{"balanced", TierBalanced, false},
		{"premium", TierPremium, false},
		{"CHEAP", TierCheap, false},       // case-insensitive
		{"Balanced", TierBalanced, false}, // case-insensitive
		{"PREMIUM", TierPremium, false},   // case-insensitive
		{"  cheap  ", TierCheap, false},   // whitespace trimmed
		{"unknown", "", true},
		{"", "", true},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseAgentTier(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseAgentTier(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParseAgentTier(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("ParseAgentTier(%q) = %v, want %v", tt.input, got, tt.expected)
				}
			}
		})
	}
}

func TestCompareTiers(t *testing.T) {
	tests := []struct {
		name     string
		a        AgentTier
		b        AgentTier
		expected int
	}{
		{"cheap < balanced", TierCheap, TierBalanced, -1},
		{"cheap < premium", TierCheap, TierPremium, -1},
		{"balanced < premium", TierBalanced, TierPremium, -1},
		{"balanced > cheap", TierBalanced, TierCheap, 1},
		{"premium > cheap", TierPremium, TierCheap, 1},
		{"premium > balanced", TierPremium, TierBalanced, 1},
		{"cheap == cheap", TierCheap, TierCheap, 0},
		{"balanced == balanced", TierBalanced, TierBalanced, 0},
		{"premium == premium", TierPremium, TierPremium, 0},
		{"invalid < cheap", AgentTier("invalid"), TierCheap, -1},
		{"cheap > invalid", TierCheap, AgentTier("invalid"), 1},
		{"invalid == invalid", AgentTier("invalid"), AgentTier("other"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareTiers(tt.a, tt.b); got != tt.expected {
				t.Errorf("CompareTiers(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestTierOrderConsistency(t *testing.T) {
	// Verify that tiers maintain expected ordering
	tiers := []AgentTier{TierCheap, TierBalanced, TierPremium}

	for i := 0; i < len(tiers)-1; i++ {
		if tiers[i].Order() >= tiers[i+1].Order() {
			t.Errorf("Expected %v.Order() < %v.Order(), got %d >= %d",
				tiers[i], tiers[i+1], tiers[i].Order(), tiers[i+1].Order())
		}
	}
}
