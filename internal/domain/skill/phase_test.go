package skill

import (
	"errors"
	"testing"
)

func TestNewPhase(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		phaseName      string
		promptTemplate string
		wantErr        error
	}{
		{
			name:           "valid phase",
			id:             "phase-1",
			phaseName:      "Analysis Phase",
			promptTemplate: "Analyze the following: {{.Input}}",
			wantErr:        nil,
		},
		{
			name:           "empty id",
			id:             "",
			phaseName:      "Analysis Phase",
			promptTemplate: "Analyze the following: {{.Input}}",
			wantErr:        ErrPhaseIDRequired,
		},
		{
			name:           "whitespace only id",
			id:             "   ",
			phaseName:      "Analysis Phase",
			promptTemplate: "Analyze the following: {{.Input}}",
			wantErr:        ErrPhaseIDRequired,
		},
		{
			name:           "empty name",
			id:             "phase-1",
			phaseName:      "",
			promptTemplate: "Analyze the following: {{.Input}}",
			wantErr:        ErrPhaseNameRequired,
		},
		{
			name:           "whitespace only name",
			id:             "phase-1",
			phaseName:      "   ",
			promptTemplate: "Analyze the following: {{.Input}}",
			wantErr:        ErrPhaseNameRequired,
		},
		{
			name:           "empty prompt template",
			id:             "phase-1",
			phaseName:      "Analysis Phase",
			promptTemplate: "",
			wantErr:        ErrPhasePromptTemplateRequired,
		},
		{
			name:           "whitespace only prompt template",
			id:             "phase-1",
			phaseName:      "Analysis Phase",
			promptTemplate: "   ",
			wantErr:        ErrPhasePromptTemplateRequired,
		},
		{
			name:           "trims whitespace from inputs",
			id:             "  phase-1  ",
			phaseName:      "  Analysis Phase  ",
			promptTemplate: "  Analyze this  ",
			wantErr:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, err := NewPhase(tt.id, tt.phaseName, tt.promptTemplate)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("NewPhase() expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewPhase() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("NewPhase() unexpected error: %v", err)
				return
			}

			if phase == nil {
				t.Error("NewPhase() returned nil phase with no error")
				return
			}

			// Verify default values
			if phase.RoutingProfile != DefaultRoutingProfile {
				t.Errorf("NewPhase() RoutingProfile = %v, want %v", phase.RoutingProfile, DefaultRoutingProfile)
			}
			if phase.MaxTokens != DefaultMaxTokens {
				t.Errorf("NewPhase() MaxTokens = %v, want %v", phase.MaxTokens, DefaultMaxTokens)
			}
			if phase.Temperature != DefaultTemperature {
				t.Errorf("NewPhase() Temperature = %v, want %v", phase.Temperature, DefaultTemperature)
			}
			if phase.DependsOn != nil {
				t.Errorf("NewPhase() DependsOn = %v, want nil", phase.DependsOn)
			}
		})
	}
}

func TestPhase_WithRoutingProfile(t *testing.T) {
	phase, err := NewPhase("phase-1", "Test Phase", "Template")
	if err != nil {
		t.Fatalf("Failed to create phase: %v", err)
	}

	result := phase.WithRoutingProfile(RoutingProfilePremium)

	// Should return the same pointer for chaining
	if result != phase {
		t.Error("WithRoutingProfile() should return the same pointer")
	}

	if phase.RoutingProfile != RoutingProfilePremium {
		t.Errorf("WithRoutingProfile() = %v, want %v", phase.RoutingProfile, RoutingProfilePremium)
	}
}

func TestPhase_WithDependencies(t *testing.T) {
	phase, err := NewPhase("phase-1", "Test Phase", "Template")
	if err != nil {
		t.Fatalf("Failed to create phase: %v", err)
	}

	deps := []string{"phase-0", "phase-init"}
	result := phase.WithDependencies(deps)

	// Should return the same pointer for chaining
	if result != phase {
		t.Error("WithDependencies() should return the same pointer")
	}

	if len(phase.DependsOn) != len(deps) {
		t.Errorf("WithDependencies() len = %v, want %v", len(phase.DependsOn), len(deps))
	}

	// Verify it's a copy, not the same slice
	deps[0] = "modified"
	if phase.DependsOn[0] == "modified" {
		t.Error("WithDependencies() should copy the slice, not reference it")
	}

	// Test with nil
	phase.WithDependencies(nil)
	if phase.DependsOn != nil {
		t.Error("WithDependencies(nil) should set DependsOn to nil")
	}
}

func TestPhase_WithMaxTokens(t *testing.T) {
	phase, err := NewPhase("phase-1", "Test Phase", "Template")
	if err != nil {
		t.Fatalf("Failed to create phase: %v", err)
	}

	result := phase.WithMaxTokens(8192)

	if result != phase {
		t.Error("WithMaxTokens() should return the same pointer")
	}

	if phase.MaxTokens != 8192 {
		t.Errorf("WithMaxTokens() = %v, want %v", phase.MaxTokens, 8192)
	}
}

func TestPhase_WithTemperature(t *testing.T) {
	phase, err := NewPhase("phase-1", "Test Phase", "Template")
	if err != nil {
		t.Fatalf("Failed to create phase: %v", err)
	}

	result := phase.WithTemperature(0.5)

	if result != phase {
		t.Error("WithTemperature() should return the same pointer")
	}

	if phase.Temperature != 0.5 {
		t.Errorf("WithTemperature() = %v, want %v", phase.Temperature, 0.5)
	}
}

func TestPhase_BuilderChaining(t *testing.T) {
	phase, err := NewPhase("phase-1", "Test Phase", "Template")
	if err != nil {
		t.Fatalf("Failed to create phase: %v", err)
	}

	phase.
		WithRoutingProfile(RoutingProfileCheap).
		WithDependencies([]string{"phase-0"}).
		WithMaxTokens(2048).
		WithTemperature(1.0)

	if phase.RoutingProfile != RoutingProfileCheap {
		t.Errorf("Chained RoutingProfile = %v, want %v", phase.RoutingProfile, RoutingProfileCheap)
	}
	if len(phase.DependsOn) != 1 || phase.DependsOn[0] != "phase-0" {
		t.Errorf("Chained DependsOn = %v, want [phase-0]", phase.DependsOn)
	}
	if phase.MaxTokens != 2048 {
		t.Errorf("Chained MaxTokens = %v, want 2048", phase.MaxTokens)
	}
	if phase.Temperature != 1.0 {
		t.Errorf("Chained Temperature = %v, want 1.0", phase.Temperature)
	}
}

func TestPhase_Validate(t *testing.T) {
	tests := []struct {
		name    string
		phase   *Phase
		wantErr error
	}{
		{
			name: "valid phase with defaults",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfileBalanced,
				MaxTokens:      4096,
				Temperature:    0.7,
			},
			wantErr: nil,
		},
		{
			name: "valid phase with premium profile",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfilePremium,
				MaxTokens:      8192,
				Temperature:    0.0,
			},
			wantErr: nil,
		},
		{
			name: "valid phase with cheap profile",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfileCheap,
				MaxTokens:      1024,
				Temperature:    2.0,
			},
			wantErr: nil,
		},
		{
			name: "empty id",
			phase: &Phase{
				ID:             "",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfileBalanced,
				MaxTokens:      4096,
				Temperature:    0.7,
			},
			wantErr: ErrPhaseIDRequired,
		},
		{
			name: "empty name",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfileBalanced,
				MaxTokens:      4096,
				Temperature:    0.7,
			},
			wantErr: ErrPhaseNameRequired,
		},
		{
			name: "empty prompt template",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "",
				RoutingProfile: RoutingProfileBalanced,
				MaxTokens:      4096,
				Temperature:    0.7,
			},
			wantErr: ErrPhasePromptTemplateRequired,
		},
		{
			name: "invalid routing profile",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: "invalid",
				MaxTokens:      4096,
				Temperature:    0.7,
			},
			wantErr: ErrInvalidRoutingProfile,
		},
		{
			name: "zero max tokens",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfileBalanced,
				MaxTokens:      0,
				Temperature:    0.7,
			},
			wantErr: ErrInvalidMaxTokens,
		},
		{
			name: "negative max tokens",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfileBalanced,
				MaxTokens:      -1,
				Temperature:    0.7,
			},
			wantErr: ErrInvalidMaxTokens,
		},
		{
			name: "negative temperature",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfileBalanced,
				MaxTokens:      4096,
				Temperature:    -0.1,
			},
			wantErr: ErrInvalidTemperature,
		},
		{
			name: "temperature too high",
			phase: &Phase{
				ID:             "phase-1",
				Name:           "Test Phase",
				PromptTemplate: "Template",
				RoutingProfile: RoutingProfileBalanced,
				MaxTokens:      4096,
				Temperature:    2.1,
			},
			wantErr: ErrInvalidTemperature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.phase.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestPhase_HasDependencies(t *testing.T) {
	tests := []struct {
		name     string
		deps     []string
		expected bool
	}{
		{
			name:     "no dependencies (nil)",
			deps:     nil,
			expected: false,
		},
		{
			name:     "no dependencies (empty)",
			deps:     []string{},
			expected: false,
		},
		{
			name:     "has one dependency",
			deps:     []string{"phase-0"},
			expected: true,
		},
		{
			name:     "has multiple dependencies",
			deps:     []string{"phase-0", "phase-1", "phase-2"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, err := NewPhase("test", "Test", "Template")
			if err != nil {
				t.Fatalf("Failed to create phase: %v", err)
			}
			phase.WithDependencies(tt.deps)

			if got := phase.HasDependencies(); got != tt.expected {
				t.Errorf("HasDependencies() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPhase_DependsOnPhase(t *testing.T) {
	phase, err := NewPhase("phase-3", "Test Phase", "Template")
	if err != nil {
		t.Fatalf("Failed to create phase: %v", err)
	}
	phase.WithDependencies([]string{"phase-1", "phase-2"})

	tests := []struct {
		phaseID  string
		expected bool
	}{
		{"phase-1", true},
		{"phase-2", true},
		{"phase-0", false},
		{"phase-3", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.phaseID, func(t *testing.T) {
			if got := phase.DependsOnPhase(tt.phaseID); got != tt.expected {
				t.Errorf("DependsOnPhase(%q) = %v, want %v", tt.phaseID, got, tt.expected)
			}
		})
	}
}

func TestPhase_CreatedViaNewPhasePassesValidation(t *testing.T) {
	phase, err := NewPhase("phase-1", "Test Phase", "Analyze: {{.Input}}")
	if err != nil {
		t.Fatalf("Failed to create phase: %v", err)
	}

	if err := phase.Validate(); err != nil {
		t.Errorf("Phase created via NewPhase should pass validation, got error: %v", err)
	}
}

func TestRoutingProfileConstants(t *testing.T) {
	// Ensure constants are what we expect
	if RoutingProfileCheap != "cheap" {
		t.Errorf("RoutingProfileCheap = %q, want %q", RoutingProfileCheap, "cheap")
	}
	if RoutingProfileBalanced != "balanced" {
		t.Errorf("RoutingProfileBalanced = %q, want %q", RoutingProfileBalanced, "balanced")
	}
	if RoutingProfilePremium != "premium" {
		t.Errorf("RoutingProfilePremium = %q, want %q", RoutingProfilePremium, "premium")
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultRoutingProfile != RoutingProfileBalanced {
		t.Errorf("DefaultRoutingProfile = %q, want %q", DefaultRoutingProfile, RoutingProfileBalanced)
	}
	if DefaultMaxTokens != 4096 {
		t.Errorf("DefaultMaxTokens = %d, want %d", DefaultMaxTokens, 4096)
	}
	if DefaultTemperature != 0.7 {
		t.Errorf("DefaultTemperature = %f, want %f", DefaultTemperature, 0.7)
	}
}
