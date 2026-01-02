package session

import (
	"strings"
	"testing"
)

func TestGenerateSessionName(t *testing.T) {
	name := GenerateSessionName()

	if name == "" {
		t.Error("GenerateSessionName returned empty string")
	}

	// Should contain a hyphen
	if !strings.Contains(name, "-") {
		t.Errorf("expected name with hyphen, got %q", name)
	}

	// Should have two parts
	parts := strings.Split(name, "-")
	if len(parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(parts))
	}

	// Both parts should be non-empty
	if parts[0] == "" || parts[1] == "" {
		t.Errorf("expected non-empty parts, got %v", parts)
	}
}

func TestGenerateSessionNameUniqueness(t *testing.T) {
	// Generate multiple names
	names := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		name := GenerateSessionName()
		names[name] = true
	}

	// We should get some variety (not all the same)
	if len(names) < 10 {
		t.Errorf("expected at least 10 unique names in %d iterations, got %d", iterations, len(names))
	}
}

func TestGenerateUniqueSessionName(t *testing.T) {
	existing := map[string]bool{
		"brave-turing":   true,
		"swift-lovelace": true,
	}

	exists := func(name string) bool {
		return existing[name]
	}

	name := GenerateUniqueSessionName(exists)
	if name == "" {
		t.Error("GenerateUniqueSessionName returned empty string")
	}

	if existing[name] {
		t.Errorf("generated name %q already exists", name)
	}
}

func TestGenerateUniqueSessionNameFallback(t *testing.T) {
	// Create a function that always returns true (all names exist)
	alwaysExists := func(name string) bool {
		return true
	}

	name := GenerateUniqueSessionName(alwaysExists)

	// Should fall back to timestamp suffix
	if name == "" {
		t.Error("GenerateUniqueSessionName returned empty string")
	}

	// Should still contain a hyphen
	if !strings.Contains(name, "-") {
		t.Errorf("expected name with hyphen, got %q", name)
	}

	// Should have at least 3 parts (adjective-pioneer-timestamp)
	parts := strings.Split(name, "-")
	if len(parts) < 3 {
		t.Errorf("expected at least 3 parts with timestamp, got %d: %v", len(parts), parts)
	}
}

func TestIsValidSessionName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"brave-turing", true},
		{"swift-lovelace", true},
		{"bold-hopper", true},
		{"", false},
		{"nohyphen", false},
		{"multiple-hyphens-ok", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSessionName(tt.name); got != tt.valid {
				t.Errorf("IsValidSessionName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestSessionNameFormat(t *testing.T) {
	// Generate many names and verify format
	for i := 0; i < 50; i++ {
		name := GenerateSessionName()
		parts := strings.Split(name, "-")

		if len(parts) != 2 {
			t.Errorf("expected 2 parts, got %d in %q", len(parts), name)
		}

		// Check that first part is an adjective
		adj := parts[0]
		found := false
		for _, a := range adjectives {
			if a == adj {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("adjective %q not in adjectives list", adj)
		}

		// Check that second part is a pioneer
		pioneer := parts[1]
		found = false
		for _, p := range pioneers {
			if p == pioneer {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pioneer %q not in pioneers list", pioneer)
		}
	}
}
