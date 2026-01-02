package skill

import (
	"testing"
)

func TestNewRoutingConfig(t *testing.T) {
	config := NewRoutingConfig()

	if config == nil {
		t.Fatal("expected non-nil config")
	}

	if config.DefaultProfile != ProfileBalanced {
		t.Errorf("expected default profile %q, got %q", ProfileBalanced, config.DefaultProfile)
	}

	if config.MaxContextTokens != 4096 {
		t.Errorf("expected max context tokens 4096, got %d", config.MaxContextTokens)
	}

	if config.GenerationModel != "" {
		t.Errorf("expected empty generation model, got %q", config.GenerationModel)
	}

	if config.ReviewModel != "" {
		t.Errorf("expected empty review model, got %q", config.ReviewModel)
	}

	if config.FallbackModel != "" {
		t.Errorf("expected empty fallback model, got %q", config.FallbackModel)
	}
}

func TestRoutingConfig_WithDefaultProfile(t *testing.T) {
	config := NewRoutingConfig().WithDefaultProfile(ProfilePremium)

	if config.DefaultProfile != ProfilePremium {
		t.Errorf("expected profile %q, got %q", ProfilePremium, config.DefaultProfile)
	}
}

func TestRoutingConfig_WithGenerationModel(t *testing.T) {
	model := "gpt-4"
	config := NewRoutingConfig().WithGenerationModel(model)

	if config.GenerationModel != model {
		t.Errorf("expected generation model %q, got %q", model, config.GenerationModel)
	}
}

func TestRoutingConfig_WithReviewModel(t *testing.T) {
	model := "claude-3-opus"
	config := NewRoutingConfig().WithReviewModel(model)

	if config.ReviewModel != model {
		t.Errorf("expected review model %q, got %q", model, config.ReviewModel)
	}
}

func TestRoutingConfig_WithFallbackModel(t *testing.T) {
	model := "gpt-3.5-turbo"
	config := NewRoutingConfig().WithFallbackModel(model)

	if config.FallbackModel != model {
		t.Errorf("expected fallback model %q, got %q", model, config.FallbackModel)
	}
}

func TestRoutingConfig_WithMaxContextTokens(t *testing.T) {
	maxTokens := 8192
	config := NewRoutingConfig().WithMaxContextTokens(maxTokens)

	if config.MaxContextTokens != maxTokens {
		t.Errorf("expected max context tokens %d, got %d", maxTokens, config.MaxContextTokens)
	}
}

func TestRoutingConfig_BuilderChaining(t *testing.T) {
	config := NewRoutingConfig().
		WithDefaultProfile(ProfilePremium).
		WithGenerationModel("gpt-4").
		WithReviewModel("claude-3-opus").
		WithFallbackModel("gpt-3.5-turbo").
		WithMaxContextTokens(16384)

	if config.DefaultProfile != ProfilePremium {
		t.Errorf("expected profile %q, got %q", ProfilePremium, config.DefaultProfile)
	}
	if config.GenerationModel != "gpt-4" {
		t.Errorf("expected generation model %q, got %q", "gpt-4", config.GenerationModel)
	}
	if config.ReviewModel != "claude-3-opus" {
		t.Errorf("expected review model %q, got %q", "claude-3-opus", config.ReviewModel)
	}
	if config.FallbackModel != "gpt-3.5-turbo" {
		t.Errorf("expected fallback model %q, got %q", "gpt-3.5-turbo", config.FallbackModel)
	}
	if config.MaxContextTokens != 16384 {
		t.Errorf("expected max context tokens %d, got %d", 16384, config.MaxContextTokens)
	}
}

func TestRoutingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RoutingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			config:  NewRoutingConfig(),
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "routing config is nil",
		},
		{
			name: "empty default profile",
			config: &RoutingConfig{
				DefaultProfile:   "",
				MaxContextTokens: 4096,
			},
			wantErr: true,
			errMsg:  "default profile is required",
		},
		{
			name: "invalid default profile",
			config: &RoutingConfig{
				DefaultProfile:   "invalid",
				MaxContextTokens: 4096,
			},
			wantErr: true,
			errMsg:  "invalid default profile",
		},
		{
			name: "zero max context tokens",
			config: &RoutingConfig{
				DefaultProfile:   ProfileBalanced,
				MaxContextTokens: 0,
			},
			wantErr: true,
			errMsg:  "max context tokens must be positive",
		},
		{
			name: "negative max context tokens",
			config: &RoutingConfig{
				DefaultProfile:   ProfileBalanced,
				MaxContextTokens: -100,
			},
			wantErr: true,
			errMsg:  "max context tokens must be positive",
		},
		{
			name: "valid cheap profile",
			config: &RoutingConfig{
				DefaultProfile:   ProfileCheap,
				MaxContextTokens: 2048,
			},
			wantErr: false,
		},
		{
			name: "valid premium profile",
			config: &RoutingConfig{
				DefaultProfile:   ProfilePremium,
				MaxContextTokens: 32768,
			},
			wantErr: false,
		},
		{
			name: "valid with all fields set",
			config: NewRoutingConfig().
				WithDefaultProfile(ProfilePremium).
				WithGenerationModel("gpt-4").
				WithReviewModel("claude-3-opus").
				WithFallbackModel("gpt-3.5-turbo").
				WithMaxContextTokens(8192),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestRoutingConfig_ProfileConstants(t *testing.T) {
	// Verify constants have expected values
	if ProfileCheap != "cheap" {
		t.Errorf("expected ProfileCheap to be %q, got %q", "cheap", ProfileCheap)
	}
	if ProfileBalanced != "balanced" {
		t.Errorf("expected ProfileBalanced to be %q, got %q", "balanced", ProfileBalanced)
	}
	if ProfilePremium != "premium" {
		t.Errorf("expected ProfilePremium to be %q, got %q", "premium", ProfilePremium)
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
