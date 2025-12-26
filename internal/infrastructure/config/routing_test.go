// Package config provides configuration structs and utilities for the skillrunner application.
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// --- RoutingConfiguration Tests ---

func TestNewRoutingConfiguration(t *testing.T) {
	cfg := NewRoutingConfiguration()

	if cfg == nil {
		t.Fatal("NewRoutingConfiguration returned nil")
	}

	if cfg.Providers == nil {
		t.Error("Providers map should be initialized")
	}

	if cfg.DefaultProvider != provider.ProviderOllama {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, provider.ProviderOllama)
	}

	if cfg.Profiles == nil {
		t.Error("Profiles map should be initialized")
	}

	// Check default profiles exist
	expectedProfiles := []string{skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium}
	for _, profile := range expectedProfiles {
		if _, ok := cfg.Profiles[profile]; !ok {
			t.Errorf("Expected profile %q not found in default config", profile)
		}
	}

	// Check fallback chain
	if len(cfg.FallbackChain) != 4 {
		t.Errorf("FallbackChain length = %d, want 4", len(cfg.FallbackChain))
	}
}

func TestRoutingConfiguration_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RoutingConfiguration
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &RoutingConfiguration{
				DefaultProvider: "ollama",
				Providers:       make(map[string]*ProviderConfiguration),
				Profiles:        make(map[string]*ProfileConfiguration),
			},
			wantErr: false,
		},
		{
			name: "empty default provider",
			config: &RoutingConfiguration{
				DefaultProvider: "",
				Providers:       make(map[string]*ProviderConfiguration),
				Profiles:        make(map[string]*ProfileConfiguration),
			},
			wantErr: true,
		},
		{
			name: "invalid profile name",
			config: &RoutingConfiguration{
				DefaultProvider: "ollama",
				Providers:       make(map[string]*ProviderConfiguration),
				Profiles: map[string]*ProfileConfiguration{
					"invalid_profile": {MaxContextTokens: 4096},
				},
			},
			wantErr: true,
		},
		{
			name: "valid profiles",
			config: &RoutingConfiguration{
				DefaultProvider: "ollama",
				Providers:       make(map[string]*ProviderConfiguration),
				Profiles: map[string]*ProfileConfiguration{
					skill.ProfileCheap:    {MaxContextTokens: 4096},
					skill.ProfileBalanced: {MaxContextTokens: 8192},
					skill.ProfilePremium:  {MaxContextTokens: 128000},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid provider config",
			config: &RoutingConfiguration{
				DefaultProvider: "ollama",
				Providers: map[string]*ProviderConfiguration{
					"test": {Priority: -1},
				},
				Profiles: make(map[string]*ProfileConfiguration),
			},
			wantErr: true,
		},
		{
			name: "empty fallback chain entry",
			config: &RoutingConfiguration{
				DefaultProvider: "ollama",
				Providers:       make(map[string]*ProviderConfiguration),
				Profiles:        make(map[string]*ProfileConfiguration),
				FallbackChain:   []string{"ollama", ""},
			},
			wantErr: true,
		},
		{
			name: "nil profile config",
			config: &RoutingConfiguration{
				DefaultProvider: "ollama",
				Providers:       make(map[string]*ProviderConfiguration),
				Profiles: map[string]*ProfileConfiguration{
					skill.ProfileCheap: nil,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoutingConfiguration_GetProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   *RoutingConfiguration
		provider string
		wantNil  bool
	}{
		{
			name:     "nil config",
			config:   nil,
			provider: "test",
			wantNil:  true,
		},
		{
			name:     "nil providers map",
			config:   &RoutingConfiguration{Providers: nil},
			provider: "test",
			wantNil:  true,
		},
		{
			name: "provider exists",
			config: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"ollama": {Enabled: true},
				},
			},
			provider: "ollama",
			wantNil:  false,
		},
		{
			name: "provider not found",
			config: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"ollama": {Enabled: true},
				},
			},
			provider: "openai",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetProvider(tt.provider)
			if (result == nil) != tt.wantNil {
				t.Errorf("GetProvider() = %v, wantNil %v", result, tt.wantNil)
			}
		})
	}
}

func TestRoutingConfiguration_GetProfile(t *testing.T) {
	tests := []struct {
		name    string
		config  *RoutingConfiguration
		profile string
		wantNil bool
	}{
		{
			name:    "nil config",
			config:  nil,
			profile: skill.ProfileBalanced,
			wantNil: true,
		},
		{
			name:    "nil profiles map",
			config:  &RoutingConfiguration{Profiles: nil},
			profile: skill.ProfileBalanced,
			wantNil: true,
		},
		{
			name: "profile exists",
			config: &RoutingConfiguration{
				Profiles: map[string]*ProfileConfiguration{
					skill.ProfileBalanced: {MaxContextTokens: 8192},
				},
			},
			profile: skill.ProfileBalanced,
			wantNil: false,
		},
		{
			name: "profile not found",
			config: &RoutingConfiguration{
				Profiles: map[string]*ProfileConfiguration{
					skill.ProfileBalanced: {MaxContextTokens: 8192},
				},
			},
			profile: skill.ProfilePremium,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetProfile(tt.profile)
			if (result == nil) != tt.wantNil {
				t.Errorf("GetProfile() = %v, wantNil %v", result, tt.wantNil)
			}
		})
	}
}

func TestRoutingConfiguration_GetEnabledProviders(t *testing.T) {
	tests := []struct {
		name   string
		config *RoutingConfiguration
		want   []string
	}{
		{
			name:   "nil config",
			config: nil,
			want:   nil,
		},
		{
			name:   "nil providers",
			config: &RoutingConfiguration{Providers: nil},
			want:   nil,
		},
		{
			name: "no enabled providers",
			config: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"ollama": {Enabled: false, Priority: 1},
					"openai": {Enabled: false, Priority: 2},
				},
			},
			want: []string{},
		},
		{
			name: "single enabled provider",
			config: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"ollama": {Enabled: true, Priority: 1},
					"openai": {Enabled: false, Priority: 2},
				},
			},
			want: []string{"ollama"},
		},
		{
			name: "multiple enabled providers sorted by priority",
			config: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"ollama":    {Enabled: true, Priority: 2},
					"openai":    {Enabled: true, Priority: 1},
					"anthropic": {Enabled: true, Priority: 3},
				},
			},
			want: []string{"openai", "ollama", "anthropic"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEnabledProviders()

			if tt.want == nil {
				if result != nil {
					t.Errorf("GetEnabledProviders() = %v, want nil", result)
				}
				return
			}

			if len(result) != len(tt.want) {
				t.Errorf("GetEnabledProviders() len = %d, want %d", len(result), len(tt.want))
				return
			}

			for i, p := range result {
				if p != tt.want[i] {
					t.Errorf("GetEnabledProviders()[%d] = %q, want %q", i, p, tt.want[i])
				}
			}
		})
	}
}

func TestRoutingConfiguration_SetDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config *RoutingConfiguration
		check  func(*RoutingConfiguration) error
	}{
		{
			name:   "empty config gets defaults",
			config: &RoutingConfiguration{},
			check: func(cfg *RoutingConfiguration) error {
				if cfg.Providers == nil {
					return errorf("Providers should not be nil")
				}
				if cfg.DefaultProvider != provider.ProviderOllama {
					return errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, provider.ProviderOllama)
				}
				if cfg.Profiles == nil {
					return errorf("Profiles should not be nil")
				}
				if len(cfg.FallbackChain) == 0 {
					return errorf("FallbackChain should not be empty")
				}
				return nil
			},
		},
		{
			name: "existing values preserved",
			config: &RoutingConfiguration{
				DefaultProvider: "openai",
				FallbackChain:   []string{"openai"},
			},
			check: func(cfg *RoutingConfiguration) error {
				if cfg.DefaultProvider != "openai" {
					return errorf("DefaultProvider should be preserved")
				}
				if len(cfg.FallbackChain) != 1 || cfg.FallbackChain[0] != "openai" {
					return errorf("FallbackChain should be preserved")
				}
				return nil
			},
		},
		{
			name: "provider defaults applied",
			config: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"test": {Enabled: true},
				},
			},
			check: func(cfg *RoutingConfiguration) error {
				p := cfg.Providers["test"]
				if p.Timeout != 30 {
					return errorf("Provider timeout = %d, want 30", p.Timeout)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.SetDefaults()
			if err := tt.check(tt.config); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRoutingConfiguration_Merge(t *testing.T) {
	tests := []struct {
		name  string
		base  *RoutingConfiguration
		other *RoutingConfiguration
		check func(*RoutingConfiguration) error
	}{
		{
			name:  "merge nil does nothing",
			base:  &RoutingConfiguration{DefaultProvider: "ollama"},
			other: nil,
			check: func(cfg *RoutingConfiguration) error {
				if cfg.DefaultProvider != "ollama" {
					return errorf("DefaultProvider changed")
				}
				return nil
			},
		},
		{
			name: "override default provider",
			base: &RoutingConfiguration{DefaultProvider: "ollama"},
			other: &RoutingConfiguration{
				DefaultProvider: "openai",
			},
			check: func(cfg *RoutingConfiguration) error {
				if cfg.DefaultProvider != "openai" {
					return errorf("DefaultProvider = %q, want openai", cfg.DefaultProvider)
				}
				return nil
			},
		},
		{
			name: "merge providers",
			base: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"ollama": {Enabled: true, Priority: 1},
				},
			},
			other: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"openai": {Enabled: true, Priority: 2},
				},
			},
			check: func(cfg *RoutingConfiguration) error {
				if len(cfg.Providers) != 2 {
					return errorf("Providers count = %d, want 2", len(cfg.Providers))
				}
				if _, ok := cfg.Providers["ollama"]; !ok {
					return errorf("ollama provider missing")
				}
				if _, ok := cfg.Providers["openai"]; !ok {
					return errorf("openai provider missing")
				}
				return nil
			},
		},
		{
			name: "override existing provider",
			base: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"ollama": {Enabled: false, Priority: 1},
				},
			},
			other: &RoutingConfiguration{
				Providers: map[string]*ProviderConfiguration{
					"ollama": {Enabled: true, Priority: 2},
				},
			},
			check: func(cfg *RoutingConfiguration) error {
				p := cfg.Providers["ollama"]
				if !p.Enabled {
					return errorf("Provider should be enabled after merge")
				}
				if p.Priority != 2 {
					return errorf("Provider priority = %d, want 2", p.Priority)
				}
				return nil
			},
		},
		{
			name: "merge profiles",
			base: &RoutingConfiguration{
				Profiles: map[string]*ProfileConfiguration{
					skill.ProfileCheap: {MaxContextTokens: 4096},
				},
			},
			other: &RoutingConfiguration{
				Profiles: map[string]*ProfileConfiguration{
					skill.ProfilePremium: {MaxContextTokens: 128000},
				},
			},
			check: func(cfg *RoutingConfiguration) error {
				if len(cfg.Profiles) != 2 {
					return errorf("Profiles count = %d, want 2", len(cfg.Profiles))
				}
				return nil
			},
		},
		{
			name: "override fallback chain",
			base: &RoutingConfiguration{
				FallbackChain: []string{"ollama", "openai"},
			},
			other: &RoutingConfiguration{
				FallbackChain: []string{"openai"},
			},
			check: func(cfg *RoutingConfiguration) error {
				if len(cfg.FallbackChain) != 1 {
					return errorf("FallbackChain len = %d, want 1", len(cfg.FallbackChain))
				}
				if cfg.FallbackChain[0] != "openai" {
					return errorf("FallbackChain[0] = %q, want openai", cfg.FallbackChain[0])
				}
				return nil
			},
		},
		{
			name:  "merge into nil providers map",
			base:  &RoutingConfiguration{Providers: nil},
			other: &RoutingConfiguration{Providers: map[string]*ProviderConfiguration{"test": {Enabled: true}}},
			check: func(cfg *RoutingConfiguration) error {
				if cfg.Providers == nil || cfg.Providers["test"] == nil {
					return errorf("Providers should be set after merge")
				}
				return nil
			},
		},
		{
			name:  "merge into nil profiles map",
			base:  &RoutingConfiguration{Profiles: nil},
			other: &RoutingConfiguration{Profiles: map[string]*ProfileConfiguration{skill.ProfileCheap: {MaxContextTokens: 4096}}},
			check: func(cfg *RoutingConfiguration) error {
				if cfg.Profiles == nil || cfg.Profiles[skill.ProfileCheap] == nil {
					return errorf("Profiles should be set after merge")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(tt.other)
			if err := tt.check(tt.base); err != nil {
				t.Error(err)
			}
		})
	}
}

// --- ProviderConfiguration Tests ---

func TestProviderConfiguration_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfiguration
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid empty config",
			config:  &ProviderConfiguration{},
			wantErr: false,
		},
		{
			name:    "negative priority",
			config:  &ProviderConfiguration{Priority: -1},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			config:  &ProviderConfiguration{Timeout: -1},
			wantErr: true,
		},
		{
			name: "invalid model config",
			config: &ProviderConfiguration{
				Models: map[string]*ModelConfiguration{
					"test": {CostPerInputToken: -1},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid rate limits",
			config: &ProviderConfiguration{
				RateLimits: &RateLimitConfiguration{RequestsPerMinute: -1},
			},
			wantErr: true,
		},
		{
			name: "valid full config",
			config: &ProviderConfiguration{
				Enabled:  true,
				Priority: 1,
				Timeout:  30,
				Models: map[string]*ModelConfiguration{
					"model1": {Tier: "balanced", Enabled: true},
				},
				RateLimits: &RateLimitConfiguration{RequestsPerMinute: 100},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProviderConfiguration_GetModel(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfiguration
		modelID string
		wantNil bool
	}{
		{
			name:    "nil config",
			config:  nil,
			modelID: "test",
			wantNil: true,
		},
		{
			name:    "nil models map",
			config:  &ProviderConfiguration{Models: nil},
			modelID: "test",
			wantNil: true,
		},
		{
			name: "model exists",
			config: &ProviderConfiguration{
				Models: map[string]*ModelConfiguration{
					"gpt-4": {Enabled: true},
				},
			},
			modelID: "gpt-4",
			wantNil: false,
		},
		{
			name: "model not found",
			config: &ProviderConfiguration{
				Models: map[string]*ModelConfiguration{
					"gpt-4": {Enabled: true},
				},
			},
			modelID: "gpt-3.5",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetModel(tt.modelID)
			if (result == nil) != tt.wantNil {
				t.Errorf("GetModel() = %v, wantNil %v", result, tt.wantNil)
			}
		})
	}
}

func TestProviderConfiguration_GetEnabledModels(t *testing.T) {
	tests := []struct {
		name   string
		config *ProviderConfiguration
		want   int
	}{
		{
			name:   "nil config",
			config: nil,
			want:   0,
		},
		{
			name:   "nil models map",
			config: &ProviderConfiguration{Models: nil},
			want:   0,
		},
		{
			name: "no enabled models",
			config: &ProviderConfiguration{
				Models: map[string]*ModelConfiguration{
					"model1": {Enabled: false},
					"model2": {Enabled: false},
				},
			},
			want: 0,
		},
		{
			name: "some enabled models",
			config: &ProviderConfiguration{
				Models: map[string]*ModelConfiguration{
					"model1": {Enabled: true},
					"model2": {Enabled: false},
					"model3": {Enabled: true},
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEnabledModels()
			if len(result) != tt.want {
				t.Errorf("GetEnabledModels() len = %d, want %d", len(result), tt.want)
			}
		})
	}
}

func TestProviderConfiguration_SetDefaults(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		var p *ProviderConfiguration
		p.SetDefaults() // Should not panic
	})

	t.Run("empty config gets defaults", func(t *testing.T) {
		p := &ProviderConfiguration{}
		p.SetDefaults()

		if p.Models == nil {
			t.Error("Models should be initialized")
		}
		if p.Timeout != 30 {
			t.Errorf("Timeout = %d, want 30", p.Timeout)
		}
	})

	t.Run("existing timeout preserved", func(t *testing.T) {
		p := &ProviderConfiguration{Timeout: 60}
		p.SetDefaults()

		if p.Timeout != 60 {
			t.Errorf("Timeout = %d, want 60", p.Timeout)
		}
	})

	t.Run("model defaults applied", func(t *testing.T) {
		p := &ProviderConfiguration{
			Models: map[string]*ModelConfiguration{
				"test": {},
			},
		}
		p.SetDefaults()

		m := p.Models["test"]
		if m.Tier != string(provider.TierBalanced) {
			t.Errorf("Model tier = %q, want %q", m.Tier, provider.TierBalanced)
		}
	})
}

func TestProviderConfiguration_Merge(t *testing.T) {
	t.Run("merge nil does nothing", func(t *testing.T) {
		p := &ProviderConfiguration{Enabled: true}
		p.Merge(nil)
		if !p.Enabled {
			t.Error("Enabled should still be true")
		}
	})

	t.Run("merge updates fields", func(t *testing.T) {
		p := &ProviderConfiguration{
			Enabled:  false,
			Priority: 1,
			Timeout:  30,
		}
		other := &ProviderConfiguration{
			Enabled:  true,
			Priority: 2,
			Timeout:  60,
			BaseURL:  "http://example.com",
		}
		p.Merge(other)

		if !p.Enabled {
			t.Error("Enabled should be true after merge")
		}
		if p.Priority != 2 {
			t.Errorf("Priority = %d, want 2", p.Priority)
		}
		if p.Timeout != 60 {
			t.Errorf("Timeout = %d, want 60", p.Timeout)
		}
		if p.BaseURL != "http://example.com" {
			t.Errorf("BaseURL = %q, want http://example.com", p.BaseURL)
		}
	})

	t.Run("merge models", func(t *testing.T) {
		p := &ProviderConfiguration{
			Models: map[string]*ModelConfiguration{
				"model1": {Enabled: true},
			},
		}
		other := &ProviderConfiguration{
			Models: map[string]*ModelConfiguration{
				"model2": {Enabled: true},
			},
		}
		p.Merge(other)

		if len(p.Models) != 2 {
			t.Errorf("Models count = %d, want 2", len(p.Models))
		}
	})

	t.Run("merge rate limits", func(t *testing.T) {
		p := &ProviderConfiguration{}
		other := &ProviderConfiguration{
			RateLimits: &RateLimitConfiguration{RequestsPerMinute: 100},
		}
		p.Merge(other)

		if p.RateLimits == nil || p.RateLimits.RequestsPerMinute != 100 {
			t.Error("RateLimits should be set after merge")
		}
	})

	t.Run("merge into nil models map", func(t *testing.T) {
		p := &ProviderConfiguration{Models: nil}
		other := &ProviderConfiguration{
			Models: map[string]*ModelConfiguration{"test": {Enabled: true}},
		}
		p.Merge(other)

		if p.Models == nil || p.Models["test"] == nil {
			t.Error("Models should be set after merge")
		}
	})
}

// --- ModelConfiguration Tests ---

func TestModelConfiguration_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ModelConfiguration
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid empty config",
			config:  &ModelConfiguration{},
			wantErr: false,
		},
		{
			name:    "valid tier cheap",
			config:  &ModelConfiguration{Tier: "cheap"},
			wantErr: false,
		},
		{
			name:    "valid tier balanced",
			config:  &ModelConfiguration{Tier: "balanced"},
			wantErr: false,
		},
		{
			name:    "valid tier premium",
			config:  &ModelConfiguration{Tier: "premium"},
			wantErr: false,
		},
		{
			name:    "invalid tier",
			config:  &ModelConfiguration{Tier: "invalid"},
			wantErr: true,
		},
		{
			name:    "negative input cost",
			config:  &ModelConfiguration{CostPerInputToken: -0.001},
			wantErr: true,
		},
		{
			name:    "negative output cost",
			config:  &ModelConfiguration{CostPerOutputToken: -0.001},
			wantErr: true,
		},
		{
			name:    "negative max tokens",
			config:  &ModelConfiguration{MaxTokens: -1},
			wantErr: true,
		},
		{
			name:    "negative context window",
			config:  &ModelConfiguration{ContextWindow: -1},
			wantErr: true,
		},
		{
			name: "valid full config",
			config: &ModelConfiguration{
				Tier:               "balanced",
				CostPerInputToken:  0.00001,
				CostPerOutputToken: 0.00002,
				MaxTokens:          4096,
				ContextWindow:      128000,
				Enabled:            true,
				Capabilities:       []string{"vision", "function_calling"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelConfiguration_GetTier(t *testing.T) {
	tests := []struct {
		name   string
		config *ModelConfiguration
		want   provider.AgentTier
	}{
		{
			name:   "nil config returns balanced",
			config: nil,
			want:   provider.TierBalanced,
		},
		{
			name:   "empty tier returns balanced",
			config: &ModelConfiguration{Tier: ""},
			want:   provider.TierBalanced,
		},
		{
			name:   "cheap tier",
			config: &ModelConfiguration{Tier: "cheap"},
			want:   provider.TierCheap,
		},
		{
			name:   "premium tier",
			config: &ModelConfiguration{Tier: "premium"},
			want:   provider.TierPremium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetTier()
			if got != tt.want {
				t.Errorf("GetTier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelConfiguration_CostPer1K(t *testing.T) {
	tests := []struct {
		name       string
		config     *ModelConfiguration
		wantInput  float64
		wantOutput float64
	}{
		{
			name:       "nil config",
			config:     nil,
			wantInput:  0,
			wantOutput: 0,
		},
		{
			name:       "zero costs",
			config:     &ModelConfiguration{},
			wantInput:  0,
			wantOutput: 0,
		},
		{
			name: "with costs",
			config: &ModelConfiguration{
				CostPerInputToken:  0.00001,
				CostPerOutputToken: 0.00002,
			},
			wantInput:  0.01,
			wantOutput: 0.02,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, output := tt.config.CostPer1K()
			if input != tt.wantInput {
				t.Errorf("CostPer1K() input = %v, want %v", input, tt.wantInput)
			}
			if output != tt.wantOutput {
				t.Errorf("CostPer1K() output = %v, want %v", output, tt.wantOutput)
			}
		})
	}
}

func TestModelConfiguration_HasCapability(t *testing.T) {
	tests := []struct {
		name   string
		config *ModelConfiguration
		cap    string
		want   bool
	}{
		{
			name:   "nil config",
			config: nil,
			cap:    "vision",
			want:   false,
		},
		{
			name:   "nil capabilities",
			config: &ModelConfiguration{Capabilities: nil},
			cap:    "vision",
			want:   false,
		},
		{
			name:   "capability exists",
			config: &ModelConfiguration{Capabilities: []string{"vision", "streaming"}},
			cap:    "vision",
			want:   true,
		},
		{
			name:   "capability not exists",
			config: &ModelConfiguration{Capabilities: []string{"vision", "streaming"}},
			cap:    "function_calling",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.HasCapability(tt.cap)
			if got != tt.want {
				t.Errorf("HasCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelConfiguration_SetDefaults(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		var m *ModelConfiguration
		m.SetDefaults() // Should not panic
	})

	t.Run("empty config gets defaults", func(t *testing.T) {
		m := &ModelConfiguration{}
		m.SetDefaults()

		if m.Tier != string(provider.TierBalanced) {
			t.Errorf("Tier = %q, want %q", m.Tier, provider.TierBalanced)
		}
		if m.ContextWindow != 4096 {
			t.Errorf("ContextWindow = %d, want 4096", m.ContextWindow)
		}
		if m.MaxTokens != 2048 {
			t.Errorf("MaxTokens = %d, want 2048", m.MaxTokens)
		}
	})

	t.Run("existing values preserved", func(t *testing.T) {
		m := &ModelConfiguration{
			Tier:          "premium",
			ContextWindow: 128000,
			MaxTokens:     8192,
		}
		m.SetDefaults()

		if m.Tier != "premium" {
			t.Errorf("Tier = %q, want premium", m.Tier)
		}
		if m.ContextWindow != 128000 {
			t.Errorf("ContextWindow = %d, want 128000", m.ContextWindow)
		}
		if m.MaxTokens != 8192 {
			t.Errorf("MaxTokens = %d, want 8192", m.MaxTokens)
		}
	})
}

// --- RateLimitConfiguration Tests ---

func TestRateLimitConfiguration_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RateLimitConfiguration
		wantErr bool
	}{
		{
			name:    "nil config is valid",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "valid empty config",
			config:  &RateLimitConfiguration{},
			wantErr: false,
		},
		{
			name:    "negative requests per minute",
			config:  &RateLimitConfiguration{RequestsPerMinute: -1},
			wantErr: true,
		},
		{
			name:    "negative tokens per minute",
			config:  &RateLimitConfiguration{TokensPerMinute: -1},
			wantErr: true,
		},
		{
			name:    "negative concurrent requests",
			config:  &RateLimitConfiguration{ConcurrentRequests: -1},
			wantErr: true,
		},
		{
			name:    "negative burst limit",
			config:  &RateLimitConfiguration{BurstLimit: -1},
			wantErr: true,
		},
		{
			name: "valid full config",
			config: &RateLimitConfiguration{
				RequestsPerMinute:  100,
				TokensPerMinute:    100000,
				ConcurrentRequests: 10,
				BurstLimit:         20,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- ProfileConfiguration Tests ---

func TestProfileConfiguration_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProfileConfiguration
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid empty config",
			config:  &ProfileConfiguration{},
			wantErr: false,
		},
		{
			name:    "negative max context tokens",
			config:  &ProfileConfiguration{MaxContextTokens: -1},
			wantErr: true,
		},
		{
			name: "valid full config",
			config: &ProfileConfiguration{
				GenerationModel:  "gpt-4",
				ReviewModel:      "gpt-4",
				FallbackModel:    "gpt-3.5-turbo",
				MaxContextTokens: 8192,
				PreferLocal:      false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProfileConfiguration_ToSkillRoutingConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProfileConfiguration
		profile string
	}{
		{
			name:    "nil config returns default",
			config:  nil,
			profile: skill.ProfileBalanced,
		},
		{
			name: "converts to skill routing config",
			config: &ProfileConfiguration{
				GenerationModel:  "gpt-4",
				ReviewModel:      "gpt-4o",
				FallbackModel:    "gpt-3.5-turbo",
				MaxContextTokens: 8192,
			},
			profile: skill.ProfilePremium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ToSkillRoutingConfig(tt.profile)

			if result == nil {
				t.Fatal("ToSkillRoutingConfig() returned nil")
			}

			if tt.config == nil {
				// Default should have balanced profile
				if result.DefaultProfile != skill.ProfileBalanced {
					t.Errorf("DefaultProfile = %q, want %q", result.DefaultProfile, skill.ProfileBalanced)
				}
			} else {
				if result.DefaultProfile != tt.profile {
					t.Errorf("DefaultProfile = %q, want %q", result.DefaultProfile, tt.profile)
				}
				if result.GenerationModel != tt.config.GenerationModel {
					t.Errorf("GenerationModel = %q, want %q", result.GenerationModel, tt.config.GenerationModel)
				}
				if result.ReviewModel != tt.config.ReviewModel {
					t.Errorf("ReviewModel = %q, want %q", result.ReviewModel, tt.config.ReviewModel)
				}
				if result.FallbackModel != tt.config.FallbackModel {
					t.Errorf("FallbackModel = %q, want %q", result.FallbackModel, tt.config.FallbackModel)
				}
				if result.MaxContextTokens != tt.config.MaxContextTokens {
					t.Errorf("MaxContextTokens = %d, want %d", result.MaxContextTokens, tt.config.MaxContextTokens)
				}
			}
		})
	}
}

func TestProfileConfiguration_Merge(t *testing.T) {
	t.Run("merge nil does nothing", func(t *testing.T) {
		p := &ProfileConfiguration{GenerationModel: "gpt-4"}
		p.Merge(nil)
		if p.GenerationModel != "gpt-4" {
			t.Error("GenerationModel should still be gpt-4")
		}
	})

	t.Run("merge updates fields", func(t *testing.T) {
		p := &ProfileConfiguration{
			GenerationModel:  "gpt-3.5",
			ReviewModel:      "gpt-3.5",
			MaxContextTokens: 4096,
			PreferLocal:      true,
		}
		other := &ProfileConfiguration{
			GenerationModel:  "gpt-4",
			ReviewModel:      "gpt-4o",
			FallbackModel:    "gpt-3.5-turbo",
			MaxContextTokens: 8192,
			PreferLocal:      false,
		}
		p.Merge(other)

		if p.GenerationModel != "gpt-4" {
			t.Errorf("GenerationModel = %q, want gpt-4", p.GenerationModel)
		}
		if p.ReviewModel != "gpt-4o" {
			t.Errorf("ReviewModel = %q, want gpt-4o", p.ReviewModel)
		}
		if p.FallbackModel != "gpt-3.5-turbo" {
			t.Errorf("FallbackModel = %q, want gpt-3.5-turbo", p.FallbackModel)
		}
		if p.MaxContextTokens != 8192 {
			t.Errorf("MaxContextTokens = %d, want 8192", p.MaxContextTokens)
		}
		if p.PreferLocal {
			t.Error("PreferLocal should be false after merge")
		}
	})

	t.Run("empty strings don't override", func(t *testing.T) {
		p := &ProfileConfiguration{
			GenerationModel: "gpt-4",
			ReviewModel:     "gpt-4o",
		}
		other := &ProfileConfiguration{
			GenerationModel: "",
			ReviewModel:     "",
		}
		p.Merge(other)

		if p.GenerationModel != "gpt-4" {
			t.Errorf("GenerationModel = %q, want gpt-4", p.GenerationModel)
		}
		if p.ReviewModel != "gpt-4o" {
			t.Errorf("ReviewModel = %q, want gpt-4o", p.ReviewModel)
		}
	})
}

// --- Loader Tests ---

func TestLoadRoutingConfig(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		_, err := LoadRoutingConfig("")
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := LoadRoutingConfig("/nonexistent/path/config.yaml")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "routing.yaml")

		content := `
default_provider: ollama
fallback_chain:
  - ollama
  - openai
providers:
  ollama:
    enabled: true
    priority: 1
    timeout: 30
profiles:
  cheap:
    generation_model: "llama3.2:3b"
    max_context_tokens: 4096
`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		cfg, err := LoadRoutingConfig(configPath)
		if err != nil {
			t.Fatalf("LoadRoutingConfig() error = %v", err)
		}

		if cfg.DefaultProvider != "ollama" {
			t.Errorf("DefaultProvider = %q, want ollama", cfg.DefaultProvider)
		}

		if len(cfg.FallbackChain) != 2 {
			t.Errorf("FallbackChain len = %d, want 2", len(cfg.FallbackChain))
		}

		p := cfg.GetProvider("ollama")
		if p == nil {
			t.Error("Expected ollama provider to exist")
		} else if !p.Enabled {
			t.Error("ollama provider should be enabled")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yaml")

		content := `
invalid: [yaml: content
`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := LoadRoutingConfig(configPath)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})
}

func TestLoadRoutingConfigFromBytes(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		_, err := LoadRoutingConfigFromBytes([]byte{})
		if err == nil {
			t.Error("Expected error for empty data")
		}
	})

	t.Run("valid yaml", func(t *testing.T) {
		data := []byte(`
default_provider: openai
providers:
  openai:
    enabled: true
    priority: 1
`)
		cfg, err := LoadRoutingConfigFromBytes(data)
		if err != nil {
			t.Fatalf("LoadRoutingConfigFromBytes() error = %v", err)
		}

		if cfg.DefaultProvider != "openai" {
			t.Errorf("DefaultProvider = %q, want openai", cfg.DefaultProvider)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		data := []byte(`invalid: [yaml`)
		_, err := LoadRoutingConfigFromBytes(data)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		// Test a validation error that SetDefaults() won't fix
		// Use an invalid profile name (not cheap/balanced/premium)
		data := []byte(`
default_provider: ollama
profiles:
  invalid_profile:
    max_context_tokens: 4096
`)
		_, err := LoadRoutingConfigFromBytes(data)
		if err == nil {
			t.Error("Expected error for validation failure")
		}
	})
}

func TestSaveRoutingConfig(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		cfg := NewRoutingConfiguration()
		err := SaveRoutingConfig("", cfg)
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "routing.yaml")

		err := SaveRoutingConfig(configPath, nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})

	t.Run("valid save", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "routing.yaml")

		cfg := NewRoutingConfiguration()
		cfg.DefaultProvider = "openai"
		cfg.Providers["openai"] = &ProviderConfiguration{
			Enabled:  true,
			Priority: 1,
		}

		err := SaveRoutingConfig(configPath, cfg)
		if err != nil {
			t.Fatalf("SaveRoutingConfig() error = %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Read back and verify
		loaded, err := LoadRoutingConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to load saved config: %v", err)
		}

		if loaded.DefaultProvider != "openai" {
			t.Errorf("Loaded DefaultProvider = %q, want openai", loaded.DefaultProvider)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nested", "dir", "routing.yaml")

		cfg := NewRoutingConfiguration()
		err := SaveRoutingConfig(configPath, cfg)
		if err != nil {
			t.Fatalf("SaveRoutingConfig() error = %v", err)
		}

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created in nested directory")
		}
	})
}

func TestMergeRoutingConfigs(t *testing.T) {
	t.Run("empty configs", func(t *testing.T) {
		result := MergeRoutingConfigs()
		if result != nil {
			t.Error("Expected nil for empty configs")
		}
	})

	t.Run("all nil configs", func(t *testing.T) {
		result := MergeRoutingConfigs(nil, nil, nil)
		if result != nil {
			t.Error("Expected nil for all nil configs")
		}
	})

	t.Run("single config", func(t *testing.T) {
		cfg := &RoutingConfiguration{DefaultProvider: "ollama"}
		result := MergeRoutingConfigs(cfg)

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.DefaultProvider != "ollama" {
			t.Errorf("DefaultProvider = %q, want ollama", result.DefaultProvider)
		}
	})

	t.Run("multiple configs", func(t *testing.T) {
		cfg1 := &RoutingConfiguration{
			DefaultProvider: "ollama",
			Providers: map[string]*ProviderConfiguration{
				"ollama": {Enabled: true, Priority: 1},
			},
		}
		cfg2 := &RoutingConfiguration{
			DefaultProvider: "openai",
			Providers: map[string]*ProviderConfiguration{
				"openai": {Enabled: true, Priority: 2},
			},
		}

		result := MergeRoutingConfigs(cfg1, cfg2)

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.DefaultProvider != "openai" {
			t.Errorf("DefaultProvider = %q, want openai", result.DefaultProvider)
		}
		if len(result.Providers) != 2 {
			t.Errorf("Providers count = %d, want 2", len(result.Providers))
		}
	})

	t.Run("skip nil configs", func(t *testing.T) {
		cfg := &RoutingConfiguration{DefaultProvider: "ollama"}
		result := MergeRoutingConfigs(nil, cfg, nil)

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.DefaultProvider != "ollama" {
			t.Errorf("DefaultProvider = %q, want ollama", result.DefaultProvider)
		}
	})
}

func TestLoadRoutingConfigWithDefaults(t *testing.T) {
	t.Run("empty path returns defaults", func(t *testing.T) {
		cfg, err := LoadRoutingConfigWithDefaults("")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg == nil {
			t.Fatal("Expected non-nil config")
		}
		if cfg.DefaultProvider != provider.ProviderOllama {
			t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, provider.ProviderOllama)
		}
	})

	t.Run("non-existent file returns defaults", func(t *testing.T) {
		cfg, err := LoadRoutingConfigWithDefaults("/nonexistent/config.yaml")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg == nil {
			t.Fatal("Expected non-nil config")
		}
	})

	t.Run("existing file is loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "routing.yaml")

		content := `default_provider: openai`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		cfg, err := LoadRoutingConfigWithDefaults(configPath)
		if err != nil {
			t.Fatalf("LoadRoutingConfigWithDefaults() error = %v", err)
		}

		if cfg.DefaultProvider != "openai" {
			t.Errorf("DefaultProvider = %q, want openai", cfg.DefaultProvider)
		}
	})
}

func TestLoadAndMergeRoutingConfigs(t *testing.T) {
	t.Run("no paths returns defaults", func(t *testing.T) {
		cfg, err := LoadAndMergeRoutingConfigs()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg == nil {
			t.Fatal("Expected non-nil config")
		}
	})

	t.Run("missing files are skipped", func(t *testing.T) {
		cfg, err := LoadAndMergeRoutingConfigs("/nonexistent1.yaml", "/nonexistent2.yaml")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg == nil {
			t.Fatal("Expected non-nil config")
		}
	})

	t.Run("empty paths are skipped", func(t *testing.T) {
		cfg, err := LoadAndMergeRoutingConfigs("", "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg == nil {
			t.Fatal("Expected non-nil config")
		}
	})

	t.Run("multiple files merged", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create first config
		config1Path := filepath.Join(tmpDir, "config1.yaml")
		content1 := `
default_provider: ollama
providers:
  ollama:
    enabled: true
    priority: 1
`
		if err := os.WriteFile(config1Path, []byte(content1), 0o644); err != nil {
			t.Fatalf("Failed to write config1: %v", err)
		}

		// Create second config
		config2Path := filepath.Join(tmpDir, "config2.yaml")
		content2 := `
default_provider: openai
providers:
  openai:
    enabled: true
    priority: 2
`
		if err := os.WriteFile(config2Path, []byte(content2), 0o644); err != nil {
			t.Fatalf("Failed to write config2: %v", err)
		}

		cfg, err := LoadAndMergeRoutingConfigs(config1Path, config2Path)
		if err != nil {
			t.Fatalf("LoadAndMergeRoutingConfigs() error = %v", err)
		}

		// Second config takes precedence
		if cfg.DefaultProvider != "openai" {
			t.Errorf("DefaultProvider = %q, want openai", cfg.DefaultProvider)
		}

		// Both providers should exist
		if len(cfg.Providers) != 2 {
			t.Errorf("Providers count = %d, want 2", len(cfg.Providers))
		}
	})

	t.Run("invalid file returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")

		content := `invalid: [yaml`
		if err := os.WriteFile(invalidPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write invalid config: %v", err)
		}

		_, err := LoadAndMergeRoutingConfigs(invalidPath)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})
}

// --- Deep Copy Tests ---

func TestDeepCopyRoutingConfig(t *testing.T) {
	t.Run("nil source", func(t *testing.T) {
		result := deepCopyRoutingConfig(nil)
		if result != nil {
			t.Error("Expected nil for nil source")
		}
	})

	t.Run("full deep copy", func(t *testing.T) {
		src := &RoutingConfiguration{
			DefaultProvider: "ollama",
			FallbackChain:   []string{"ollama", "openai"},
			Providers: map[string]*ProviderConfiguration{
				"ollama": {
					Enabled:  true,
					Priority: 1,
					BaseURL:  "http://localhost:11434",
					Timeout:  30,
					RateLimits: &RateLimitConfiguration{
						RequestsPerMinute:  100,
						TokensPerMinute:    100000,
						ConcurrentRequests: 10,
						BurstLimit:         20,
					},
					Models: map[string]*ModelConfiguration{
						"llama3.2:3b": {
							Tier:               "cheap",
							CostPerInputToken:  0,
							CostPerOutputToken: 0,
							MaxTokens:          4096,
							ContextWindow:      8192,
							Enabled:            true,
							Capabilities:       []string{"streaming"},
							Aliases:            []string{"llama3"},
						},
					},
				},
			},
			Profiles: map[string]*ProfileConfiguration{
				skill.ProfileCheap: {
					GenerationModel:  "llama3.2:3b",
					ReviewModel:      "llama3.2:3b",
					FallbackModel:    "llama3.2:1b",
					MaxContextTokens: 4096,
					PreferLocal:      true,
				},
			},
		}

		dst := deepCopyRoutingConfig(src)

		// Verify values are copied
		if dst.DefaultProvider != src.DefaultProvider {
			t.Errorf("DefaultProvider not copied correctly")
		}

		// Verify deep copy (modifying dst should not affect src)
		dst.DefaultProvider = "modified"
		if src.DefaultProvider == "modified" {
			t.Error("Modifying copy affected original")
		}

		dst.FallbackChain[0] = "modified"
		if src.FallbackChain[0] == "modified" {
			t.Error("Modifying fallback chain copy affected original")
		}

		// Verify providers are deep copied
		dstProvider := dst.Providers["ollama"]
		srcProvider := src.Providers["ollama"]

		dstProvider.Enabled = false
		if !srcProvider.Enabled {
			t.Error("Modifying provider copy affected original")
		}

		dstProvider.RateLimits.RequestsPerMinute = 999
		if srcProvider.RateLimits.RequestsPerMinute == 999 {
			t.Error("Modifying rate limits copy affected original")
		}

		// Verify models are deep copied
		dstModel := dstProvider.Models["llama3.2:3b"]
		srcModel := srcProvider.Models["llama3.2:3b"]

		dstModel.Enabled = false
		if !srcModel.Enabled {
			t.Error("Modifying model copy affected original")
		}

		dstModel.Capabilities[0] = "modified"
		if srcModel.Capabilities[0] == "modified" {
			t.Error("Modifying capabilities copy affected original")
		}

		dstModel.Aliases[0] = "modified"
		if srcModel.Aliases[0] == "modified" {
			t.Error("Modifying aliases copy affected original")
		}

		// Verify profiles are deep copied
		dstProfile := dst.Profiles[skill.ProfileCheap]
		srcProfile := src.Profiles[skill.ProfileCheap]

		dstProfile.GenerationModel = "modified"
		if srcProfile.GenerationModel == "modified" {
			t.Error("Modifying profile copy affected original")
		}
	})

	t.Run("nil nested fields", func(t *testing.T) {
		src := &RoutingConfiguration{
			DefaultProvider: "ollama",
			FallbackChain:   nil,
			Providers:       nil,
			Profiles:        nil,
		}

		dst := deepCopyRoutingConfig(src)

		if dst.FallbackChain != nil {
			t.Error("FallbackChain should be nil")
		}
		if dst.Providers != nil {
			t.Error("Providers should be nil")
		}
		if dst.Profiles != nil {
			t.Error("Profiles should be nil")
		}
	})
}

func TestDeepCopyProviderConfig(t *testing.T) {
	t.Run("nil source", func(t *testing.T) {
		result := deepCopyProviderConfig(nil)
		if result != nil {
			t.Error("Expected nil for nil source")
		}
	})

	t.Run("nil rate limits", func(t *testing.T) {
		src := &ProviderConfiguration{
			Enabled:    true,
			RateLimits: nil,
		}

		dst := deepCopyProviderConfig(src)
		if dst.RateLimits != nil {
			t.Error("RateLimits should be nil")
		}
	})

	t.Run("nil models", func(t *testing.T) {
		src := &ProviderConfiguration{
			Enabled: true,
			Models:  nil,
		}

		dst := deepCopyProviderConfig(src)
		if dst.Models != nil {
			t.Error("Models should be nil")
		}
	})
}

func TestDeepCopyModelConfig(t *testing.T) {
	t.Run("nil source", func(t *testing.T) {
		result := deepCopyModelConfig(nil)
		if result != nil {
			t.Error("Expected nil for nil source")
		}
	})

	t.Run("nil capabilities", func(t *testing.T) {
		src := &ModelConfiguration{
			Enabled:      true,
			Capabilities: nil,
		}

		dst := deepCopyModelConfig(src)
		if dst.Capabilities != nil {
			t.Error("Capabilities should be nil")
		}
	})

	t.Run("nil aliases", func(t *testing.T) {
		src := &ModelConfiguration{
			Enabled: true,
			Aliases: nil,
		}

		dst := deepCopyModelConfig(src)
		if dst.Aliases != nil {
			t.Error("Aliases should be nil")
		}
	})
}

func TestDeepCopyProfileConfig(t *testing.T) {
	t.Run("nil source", func(t *testing.T) {
		result := deepCopyProfileConfig(nil)
		if result != nil {
			t.Error("Expected nil for nil source")
		}
	})

	t.Run("copies all fields", func(t *testing.T) {
		src := &ProfileConfiguration{
			GenerationModel:  "gpt-4",
			ReviewModel:      "gpt-4o",
			FallbackModel:    "gpt-3.5-turbo",
			MaxContextTokens: 8192,
			PreferLocal:      true,
		}

		dst := deepCopyProfileConfig(src)

		if dst.GenerationModel != src.GenerationModel {
			t.Error("GenerationModel not copied")
		}
		if dst.ReviewModel != src.ReviewModel {
			t.Error("ReviewModel not copied")
		}
		if dst.FallbackModel != src.FallbackModel {
			t.Error("FallbackModel not copied")
		}
		if dst.MaxContextTokens != src.MaxContextTokens {
			t.Error("MaxContextTokens not copied")
		}
		if dst.PreferLocal != src.PreferLocal {
			t.Error("PreferLocal not copied")
		}

		// Verify deep copy
		dst.GenerationModel = "modified"
		if src.GenerationModel == "modified" {
			t.Error("Modifying copy affected original")
		}
	})
}

// Helper for creating error messages in checks
type testError string

func (e testError) Error() string {
	return string(e)
}

func errorf(format string, args ...interface{}) error {
	return testError(sprintf(format, args...))
}

func sprintf(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}

	result := format
	for _, arg := range args {
		// Simple replacement - find %q, %d, %v and replace with value
		for _, placeholder := range []string{"%q", "%d", "%v", "%s"} {
			idx := indexOf(result, placeholder)
			if idx >= 0 {
				var replacement string
				switch placeholder {
				case "%q":
					replacement = "\"" + stringValue(arg) + "\""
				case "%d":
					replacement = stringValue(arg)
				case "%v", "%s":
					replacement = stringValue(arg)
				}
				result = result[:idx] + replacement + result[idx+2:]
				break
			}
		}
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func stringValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return intToString(val)
	case int64:
		return int64ToString(val)
	case float64:
		return float64ToString(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return "<value>"
	}
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

func int64ToString(n int64) string {
	return intToString(int(n))
}

func float64ToString(f float64) string {
	// Simple conversion - for test purposes
	intPart := int(f)
	fracPart := int((f - float64(intPart)) * 100)
	if fracPart < 0 {
		fracPart = -fracPart
	}
	if fracPart == 0 {
		return intToString(intPart)
	}
	return intToString(intPart) + "." + intToString(fracPart)
}
