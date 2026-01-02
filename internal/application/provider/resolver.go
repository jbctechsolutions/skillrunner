// Package provider provides model routing and provider selection for LLM requests.
package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainProvider "github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
)

// Resolver errors
var (
	ErrResolverConfigNil   = errors.New("resolver configuration is nil")
	ErrResolverRegistryNil = errors.New("resolver registry is nil")
	ErrResolverRouterNil   = errors.New("resolver router is nil")
	ErrModelNotResolved    = errors.New("failed to resolve model")
)

// Resolution represents the result of resolving a model for a request.
// It includes the selected model, provider, and cost information.
type Resolution struct {
	ModelID       string
	ProviderName  string
	IsFallback    bool
	ModelConfig   *config.ModelConfiguration
	EstimatedCost *domainProvider.CostBreakdown
}

// Resolver provides a unified service for resolving models based on routing rules,
// provider availability, and cost considerations. It combines registry lookup with
// routing configuration to provide intelligent model selection.
type Resolver struct {
	mu           sync.RWMutex
	router       *Router
	registry     *adapterProvider.Registry
	config       *config.RoutingConfiguration
	costTracking *domainProvider.CostSummary
}

// NewResolver creates a new Resolver with the given dependencies.
// The router is used for model selection, and the registry for provider lookup.
func NewResolver(router *Router, registry *adapterProvider.Registry, cfg *config.RoutingConfiguration) (*Resolver, error) {
	if router == nil {
		return nil, ErrResolverRouterNil
	}
	if registry == nil {
		return nil, ErrResolverRegistryNil
	}
	if cfg == nil {
		return nil, ErrResolverConfigNil
	}

	return &Resolver{
		router:       router,
		registry:     registry,
		config:       cfg,
		costTracking: domainProvider.NewCostSummary(),
	}, nil
}

// Resolve selects a model based on the given routing profile and returns
// a complete resolution including model configuration and cost estimate.
func (r *Resolver) Resolve(ctx context.Context, profile string) (*Resolution, error) {
	selection, err := r.router.SelectModel(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrModelNotResolved, err)
	}

	return r.buildResolution(selection)
}

// ResolveForPhase selects a model based on the phase's routing requirements.
// It considers the phase type (generation vs review) and routing profile.
func (r *Resolver) ResolveForPhase(ctx context.Context, phase *skill.Phase) (*Resolution, error) {
	if phase == nil {
		return nil, errors.New("phase is nil")
	}

	selection, err := r.router.SelectModelForPhase(ctx, phase)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrModelNotResolved, err)
	}

	return r.buildResolution(selection)
}

// ResolveWithCapabilities selects a model that has all the required capabilities.
// Falls back to regular resolution if no model with all capabilities is found.
func (r *Resolver) ResolveWithCapabilities(ctx context.Context, profile string, capabilities []string) (*Resolution, error) {
	selection, err := r.router.SelectModelWithCapabilities(ctx, profile, capabilities)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrModelNotResolved, err)
	}

	resolution, err := r.buildResolution(selection)
	if err != nil {
		return nil, err
	}

	// Verify capabilities are actually present
	if resolution.ModelConfig != nil && len(capabilities) > 0 {
		if !hasAllCapabilities(resolution.ModelConfig, capabilities) {
			// Model was a fallback, note that capabilities may not be fully met
			resolution.IsFallback = true
		}
	}

	return resolution, nil
}

// buildResolution converts a ModelSelection to a complete Resolution.
func (r *Resolver) buildResolution(selection *ModelSelection) (*Resolution, error) {
	if selection == nil {
		return nil, ErrModelNotResolved
	}

	r.mu.RLock()
	modelConfig := r.router.GetModelConfig(selection.ProviderName, selection.ModelID)
	r.mu.RUnlock()

	resolution := &Resolution{
		ModelID:      selection.ModelID,
		ProviderName: selection.ProviderName,
		IsFallback:   selection.IsFallback,
		ModelConfig:  modelConfig,
	}

	return resolution, nil
}

// TrackCost records the cost of a model invocation and adds it to the running total.
func (r *Resolver) TrackCost(modelID, providerName string, inputTokens, outputTokens int) *domainProvider.CostBreakdown {
	r.mu.RLock()
	modelConfig := r.router.GetModelConfig(providerName, modelID)
	r.mu.RUnlock()

	// Create a domain model for cost calculation
	var model *domainProvider.Model
	if modelConfig != nil {
		model = domainProvider.NewModel(modelID, modelID, providerName).
			WithCosts(modelConfig.CostPerInputToken*1000, modelConfig.CostPerOutputToken*1000)
	} else {
		// Create a default model with zero costs
		model = domainProvider.NewModel(modelID, modelID, providerName).
			WithCosts(0, 0)
	}

	breakdown := domainProvider.CalculateCost(model, inputTokens, outputTokens)

	r.mu.Lock()
	r.costTracking.Add(breakdown)
	r.mu.Unlock()

	return breakdown
}

// GetCostSummary returns a copy of the current cost tracking summary.
func (r *Resolver) GetCostSummary() *domainProvider.CostSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.costTracking.Clone()
}

// ResetCostTracking clears the cost tracking summary.
func (r *Resolver) ResetCostTracking() {
	r.mu.Lock()
	r.costTracking = domainProvider.NewCostSummary()
	r.mu.Unlock()
}

// GetProvider returns the provider instance for the given name.
func (r *Resolver) GetProvider(name string) ProviderPort {
	return r.registry.Get(name)
}

// IsModelAvailable checks if a model is available through any provider.
func (r *Resolver) IsModelAvailable(ctx context.Context, modelID string) bool {
	return r.router.IsModelAvailable(ctx, modelID)
}

// GetEnabledProviders returns the list of enabled provider names.
func (r *Resolver) GetEnabledProviders() []string {
	return r.router.GetEnabledProviders()
}

// GetDefaultProvider returns the default provider name.
func (r *Resolver) GetDefaultProvider() string {
	return r.router.GetDefaultProvider()
}

// EstimateCost calculates the estimated cost for a given model and token counts
// without tracking it.
func (r *Resolver) EstimateCost(modelID, providerName string, inputTokens, outputTokens int) *domainProvider.CostBreakdown {
	r.mu.RLock()
	modelConfig := r.router.GetModelConfig(providerName, modelID)
	r.mu.RUnlock()

	var model *domainProvider.Model
	if modelConfig != nil {
		model = domainProvider.NewModel(modelID, modelID, providerName).
			WithCosts(modelConfig.CostPerInputToken*1000, modelConfig.CostPerOutputToken*1000)
	} else {
		model = domainProvider.NewModel(modelID, modelID, providerName).
			WithCosts(0, 0)
	}

	return domainProvider.CalculateCost(model, inputTokens, outputTokens)
}

// GetProfileConfig returns the profile configuration for the given profile name.
func (r *Resolver) GetProfileConfig(profile string) *config.ProfileConfiguration {
	return r.router.GetProfileConfig(profile)
}

// ProviderPort re-exports the interface from ports for convenience.
type ProviderPort = ports.ProviderPort
