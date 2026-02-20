// Package config provides configuration structs and utilities for the skillrunner application.
package config

import (
	"errors"
	"fmt"

	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// RoutingConfiguration is the top-level routing configuration.
// It defines providers, their models, and profile mappings for routing decisions.
type RoutingConfiguration struct {
	// Providers maps provider names to their configurations.
	Providers map[string]*ProviderConfiguration `yaml:"providers"`

	// DefaultProvider is the provider to use when no specific routing is defined.
	DefaultProvider string `yaml:"default_provider"`

	// Profiles maps routing profiles (cheap/balanced/premium) to model selections.
	Profiles map[string]*ProfileConfiguration `yaml:"profiles"`

	// FallbackChain defines the order of fallback providers when the primary is unavailable.
	FallbackChain []string `yaml:"fallback_chain"`
}

// ProviderConfiguration defines configuration for a single LLM provider.
type ProviderConfiguration struct {
	// Enabled determines if this provider is active.
	Enabled bool `yaml:"enabled"`

	// Priority determines the order of preference (lower = higher priority).
	Priority int `yaml:"priority"`

	// Models maps model IDs to their configurations.
	Models map[string]*ModelConfiguration `yaml:"models"`

	// RateLimits defines rate limiting for this provider.
	RateLimits *RateLimitConfiguration `yaml:"rate_limits"`

	// BaseURL overrides the default base URL for the provider.
	BaseURL string `yaml:"base_url,omitempty"`

	// Timeout is the request timeout in seconds.
	Timeout int `yaml:"timeout"`
}

// ModelConfiguration defines configuration for a single model.
type ModelConfiguration struct {
	// Tier specifies the cost/capability tier (cheap, balanced, premium).
	Tier string `yaml:"tier"`

	// CostPerInputToken is the cost per input token in USD.
	CostPerInputToken float64 `yaml:"cost_per_input_token"`

	// CostPerOutputToken is the cost per output token in USD.
	CostPerOutputToken float64 `yaml:"cost_per_output_token"`

	// MaxTokens is the maximum tokens this model can generate per request.
	MaxTokens int `yaml:"max_tokens"`

	// ContextWindow is the maximum context size in tokens.
	ContextWindow int `yaml:"context_window"`

	// Enabled determines if this model is available for routing.
	Enabled bool `yaml:"enabled"`

	// Capabilities lists the model's capabilities (e.g., vision, function_calling).
	Capabilities []string `yaml:"capabilities,omitempty"`

	// Aliases are alternative names for this model.
	Aliases []string `yaml:"aliases,omitempty"`
}

// RateLimitConfiguration defines rate limiting for a provider.
type RateLimitConfiguration struct {
	// RequestsPerMinute is the maximum requests allowed per minute.
	RequestsPerMinute int `yaml:"requests_per_minute"`

	// TokensPerMinute is the maximum tokens allowed per minute.
	TokensPerMinute int `yaml:"tokens_per_minute"`

	// ConcurrentRequests is the maximum concurrent requests allowed.
	ConcurrentRequests int `yaml:"concurrent_requests"`

	// BurstLimit is the maximum burst size for rate limiting.
	BurstLimit int `yaml:"burst_limit"`
}

// ProfileConfiguration maps a routing profile to specific model selections.
type ProfileConfiguration struct {
	// GenerationModel is the model to use for generation phases.
	GenerationModel string `yaml:"generation_model"`

	// ReviewModel is the model to use for review phases.
	ReviewModel string `yaml:"review_model"`

	// FallbackModel is the model to use when primary models are unavailable.
	FallbackModel string `yaml:"fallback_model"`

	// MaxContextTokens is the maximum context tokens for this profile.
	MaxContextTokens int `yaml:"max_context_tokens"`

	// PreferLocal indicates whether to prefer local models when available.
	PreferLocal bool `yaml:"prefer_local"`
}

// NewRoutingConfiguration creates a new RoutingConfiguration with sensible defaults.
func NewRoutingConfiguration() *RoutingConfiguration {
	return &RoutingConfiguration{
		Providers:       make(map[string]*ProviderConfiguration),
		DefaultProvider: provider.ProviderOllama,
		Profiles:        defaultProfiles(),
		FallbackChain:   []string{provider.ProviderOllama, provider.ProviderGroq, provider.ProviderOpenAI, provider.ProviderAnthropic},
	}
}

// defaultProfiles returns the default profile configurations.
func defaultProfiles() map[string]*ProfileConfiguration {
	return map[string]*ProfileConfiguration{
		skill.ProfileCheap: {
			GenerationModel:  "llama3.2:3b",
			ReviewModel:      "llama3.2:3b",
			FallbackModel:    "llama3.2:1b",
			MaxContextTokens: 4096,
			PreferLocal:      true,
		},
		skill.ProfileBalanced: {
			GenerationModel:  "llama3.2:8b",
			ReviewModel:      "llama3.2:8b",
			FallbackModel:    "llama3.2:3b",
			MaxContextTokens: 8192,
			PreferLocal:      true,
		},
		skill.ProfilePremium: {
			GenerationModel:  "claude-3-5-sonnet-20241022",
			ReviewModel:      "gpt-4o",
			FallbackModel:    "llama3.2:70b",
			MaxContextTokens: 128000,
			PreferLocal:      false,
		},
	}
}

// NewRoutingConfigurationFromConfig creates a RoutingConfiguration from a user's Config.
// It merges user-defined profiles over the defaults, ensuring user settings take precedence.
func NewRoutingConfigurationFromConfig(cfg *Config) *RoutingConfiguration {
	rc := NewRoutingConfiguration()

	if cfg == nil {
		return rc
	}

	// Merge user-defined profiles over defaults
	if cfg.Routing.Profiles != nil {
		for name, profile := range cfg.Routing.Profiles {
			if existing, ok := rc.Profiles[name]; ok {
				existing.Merge(profile)
			} else {
				rc.Profiles[name] = profile
			}
		}
	}

	return rc
}

// Validate checks if the RoutingConfiguration is valid.
func (r *RoutingConfiguration) Validate() error {
	if r == nil {
		return errors.New("routing configuration is nil")
	}

	var errs []error

	// Validate default provider
	if r.DefaultProvider == "" {
		errs = append(errs, errors.New("default_provider is required"))
	}

	// Validate providers
	for name, cfg := range r.Providers {
		if err := cfg.Validate(name); err != nil {
			errs = append(errs, fmt.Errorf("provider %q: %w", name, err))
		}
	}

	// Validate profiles
	validProfiles := map[string]bool{
		skill.ProfileCheap:    true,
		skill.ProfileBalanced: true,
		skill.ProfilePremium:  true,
	}

	for name, cfg := range r.Profiles {
		if !validProfiles[name] {
			errs = append(errs, fmt.Errorf("invalid profile name %q: must be one of cheap, balanced, premium", name))
			continue
		}
		if err := cfg.Validate(name); err != nil {
			errs = append(errs, fmt.Errorf("profile %q: %w", name, err))
		}
	}

	// Validate fallback chain references valid providers
	for _, providerName := range r.FallbackChain {
		if providerName == "" {
			errs = append(errs, errors.New("fallback_chain contains empty provider name"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// GetProvider returns the provider configuration for the given name.
// Returns nil if the provider is not configured.
func (r *RoutingConfiguration) GetProvider(name string) *ProviderConfiguration {
	if r == nil || r.Providers == nil {
		return nil
	}
	return r.Providers[name]
}

// GetProfile returns the profile configuration for the given profile name.
// Returns nil if the profile is not configured.
func (r *RoutingConfiguration) GetProfile(name string) *ProfileConfiguration {
	if r == nil || r.Profiles == nil {
		return nil
	}
	return r.Profiles[name]
}

// GetEnabledProviders returns a list of enabled provider names in priority order.
func (r *RoutingConfiguration) GetEnabledProviders() []string {
	if r == nil || r.Providers == nil {
		return nil
	}

	type providerPriority struct {
		name     string
		priority int
	}

	var providers []providerPriority
	for name, cfg := range r.Providers {
		if cfg.Enabled {
			providers = append(providers, providerPriority{name: name, priority: cfg.Priority})
		}
	}

	// Sort by priority (lower = higher priority)
	for i := 0; i < len(providers)-1; i++ {
		for j := i + 1; j < len(providers); j++ {
			if providers[j].priority < providers[i].priority {
				providers[i], providers[j] = providers[j], providers[i]
			}
		}
	}

	result := make([]string, len(providers))
	for i, p := range providers {
		result[i] = p.name
	}

	return result
}

// Validate checks if the ProviderConfiguration is valid.
func (p *ProviderConfiguration) Validate(providerName string) error {
	if p == nil {
		return errors.New("provider configuration is nil")
	}

	var errs []error

	if p.Priority < 0 {
		errs = append(errs, errors.New("priority must be non-negative"))
	}

	if p.Timeout < 0 {
		errs = append(errs, errors.New("timeout must be non-negative"))
	}

	// Validate models
	for modelID, cfg := range p.Models {
		if err := cfg.Validate(modelID); err != nil {
			errs = append(errs, fmt.Errorf("model %q: %w", modelID, err))
		}
	}

	// Validate rate limits
	if p.RateLimits != nil {
		if err := p.RateLimits.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("rate_limits: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// GetModel returns the model configuration for the given model ID.
// Returns nil if the model is not configured.
func (p *ProviderConfiguration) GetModel(modelID string) *ModelConfiguration {
	if p == nil || p.Models == nil {
		return nil
	}
	return p.Models[modelID]
}

// GetEnabledModels returns a list of enabled model IDs.
func (p *ProviderConfiguration) GetEnabledModels() []string {
	if p == nil || p.Models == nil {
		return nil
	}

	var result []string
	for id, cfg := range p.Models {
		if cfg.Enabled {
			result = append(result, id)
		}
	}

	return result
}

// Validate checks if the ModelConfiguration is valid.
func (m *ModelConfiguration) Validate(modelID string) error {
	if m == nil {
		return errors.New("model configuration is nil")
	}

	var errs []error

	// Validate tier
	if m.Tier != "" {
		tier := provider.AgentTier(m.Tier)
		if !tier.IsValid() {
			errs = append(errs, fmt.Errorf("invalid tier %q: must be one of cheap, balanced, premium", m.Tier))
		}
	}

	// Validate costs
	if m.CostPerInputToken < 0 {
		errs = append(errs, errors.New("cost_per_input_token must be non-negative"))
	}

	if m.CostPerOutputToken < 0 {
		errs = append(errs, errors.New("cost_per_output_token must be non-negative"))
	}

	// Validate token limits
	if m.MaxTokens < 0 {
		errs = append(errs, errors.New("max_tokens must be non-negative"))
	}

	if m.ContextWindow < 0 {
		errs = append(errs, errors.New("context_window must be non-negative"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// GetTier returns the AgentTier for this model.
// Returns TierBalanced as default if tier is not set.
func (m *ModelConfiguration) GetTier() provider.AgentTier {
	if m == nil || m.Tier == "" {
		return provider.TierBalanced
	}
	return provider.AgentTier(m.Tier)
}

// CostPer1K returns the costs per 1000 tokens for input and output.
func (m *ModelConfiguration) CostPer1K() (inputCost, outputCost float64) {
	if m == nil {
		return 0, 0
	}
	return m.CostPerInputToken * 1000, m.CostPerOutputToken * 1000
}

// HasCapability returns true if the model has the specified capability.
func (m *ModelConfiguration) HasCapability(cap string) bool {
	if m == nil || m.Capabilities == nil {
		return false
	}
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// Validate checks if the RateLimitConfiguration is valid.
func (r *RateLimitConfiguration) Validate() error {
	if r == nil {
		return nil
	}

	var errs []error

	if r.RequestsPerMinute < 0 {
		errs = append(errs, errors.New("requests_per_minute must be non-negative"))
	}

	if r.TokensPerMinute < 0 {
		errs = append(errs, errors.New("tokens_per_minute must be non-negative"))
	}

	if r.ConcurrentRequests < 0 {
		errs = append(errs, errors.New("concurrent_requests must be non-negative"))
	}

	if r.BurstLimit < 0 {
		errs = append(errs, errors.New("burst_limit must be non-negative"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate checks if the ProfileConfiguration is valid.
func (p *ProfileConfiguration) Validate(profileName string) error {
	if p == nil {
		return errors.New("profile configuration is nil")
	}

	var errs []error

	if p.MaxContextTokens < 0 {
		errs = append(errs, errors.New("max_context_tokens must be non-negative"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ToSkillRoutingConfig converts a ProfileConfiguration to a skill.RoutingConfig.
func (p *ProfileConfiguration) ToSkillRoutingConfig(profile string) *skill.RoutingConfig {
	if p == nil {
		return skill.NewRoutingConfig()
	}

	return skill.NewRoutingConfig().
		WithDefaultProfile(profile).
		WithGenerationModel(p.GenerationModel).
		WithReviewModel(p.ReviewModel).
		WithFallbackModel(p.FallbackModel).
		WithMaxContextTokens(p.MaxContextTokens)
}

// SetDefaults applies default values to a RoutingConfiguration.
func (r *RoutingConfiguration) SetDefaults() {
	if r.Providers == nil {
		r.Providers = make(map[string]*ProviderConfiguration)
	}

	if r.DefaultProvider == "" {
		r.DefaultProvider = provider.ProviderOllama
	}

	if r.Profiles == nil {
		r.Profiles = defaultProfiles()
	}

	if len(r.FallbackChain) == 0 {
		r.FallbackChain = []string{provider.ProviderOllama, provider.ProviderGroq, provider.ProviderOpenAI, provider.ProviderAnthropic}
	}

	// Apply defaults to each provider
	for _, cfg := range r.Providers {
		cfg.SetDefaults()
	}
}

// SetDefaults applies default values to a ProviderConfiguration.
func (p *ProviderConfiguration) SetDefaults() {
	if p == nil {
		return
	}

	if p.Models == nil {
		p.Models = make(map[string]*ModelConfiguration)
	}

	if p.Timeout == 0 {
		p.Timeout = 30 // 30 seconds default
	}

	// Apply defaults to each model
	for _, cfg := range p.Models {
		cfg.SetDefaults()
	}
}

// SetDefaults applies default values to a ModelConfiguration.
func (m *ModelConfiguration) SetDefaults() {
	if m == nil {
		return
	}

	if m.Tier == "" {
		m.Tier = string(provider.TierBalanced)
	}

	if m.ContextWindow == 0 {
		m.ContextWindow = 4096
	}

	if m.MaxTokens == 0 {
		m.MaxTokens = 2048
	}
}

// Merge merges another RoutingConfiguration into this one.
// Values from other take precedence over values in r.
func (r *RoutingConfiguration) Merge(other *RoutingConfiguration) {
	if other == nil {
		return
	}

	if other.DefaultProvider != "" {
		r.DefaultProvider = other.DefaultProvider
	}

	if len(other.FallbackChain) > 0 {
		r.FallbackChain = other.FallbackChain
	}

	// Merge providers
	if r.Providers == nil {
		r.Providers = make(map[string]*ProviderConfiguration)
	}
	for name, cfg := range other.Providers {
		if existing, ok := r.Providers[name]; ok {
			existing.Merge(cfg)
		} else {
			r.Providers[name] = cfg
		}
	}

	// Merge profiles
	if r.Profiles == nil {
		r.Profiles = make(map[string]*ProfileConfiguration)
	}
	for name, cfg := range other.Profiles {
		if existing, ok := r.Profiles[name]; ok {
			existing.Merge(cfg)
		} else {
			r.Profiles[name] = cfg
		}
	}
}

// Merge merges another ProviderConfiguration into this one.
func (p *ProviderConfiguration) Merge(other *ProviderConfiguration) {
	if other == nil {
		return
	}

	p.Enabled = other.Enabled
	p.Priority = other.Priority

	if other.BaseURL != "" {
		p.BaseURL = other.BaseURL
	}

	if other.Timeout > 0 {
		p.Timeout = other.Timeout
	}

	if other.RateLimits != nil {
		p.RateLimits = other.RateLimits
	}

	// Merge models
	if p.Models == nil {
		p.Models = make(map[string]*ModelConfiguration)
	}
	for id, cfg := range other.Models {
		p.Models[id] = cfg
	}
}

// Merge merges another ProfileConfiguration into this one.
func (p *ProfileConfiguration) Merge(other *ProfileConfiguration) {
	if other == nil {
		return
	}

	if other.GenerationModel != "" {
		p.GenerationModel = other.GenerationModel
	}

	if other.ReviewModel != "" {
		p.ReviewModel = other.ReviewModel
	}

	if other.FallbackModel != "" {
		p.FallbackModel = other.FallbackModel
	}

	if other.MaxContextTokens > 0 {
		p.MaxContextTokens = other.MaxContextTokens
	}

	p.PreferLocal = other.PreferLocal
}
