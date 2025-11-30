package models

import (
	"context"
	"fmt"
	"sync"
)

// StaticModel describes a model backed by a static provider.
type StaticModel struct {
	Name            string
	Route           string
	Description     string
	Available       bool
	Tier            AgentTier
	CostPer1KTokens float64
}

// StaticProvider is an in-memory implementation of ModelProvider used for defaults and tests.
type StaticProvider struct {
	info   ProviderInfo
	mu     sync.RWMutex
	models map[string]StaticModel
}

// NewStaticProvider constructs a StaticProvider from the supplied models.
func NewStaticProvider(info ProviderInfo, models []StaticModel) *StaticProvider {
	index := make(map[string]StaticModel, len(models))
	for _, model := range models {
		index[model.Name] = model
	}
	return &StaticProvider{
		info:   info,
		models: index,
	}
}

// Info returns provider metadata.
func (p *StaticProvider) Info() ProviderInfo {
	return p.info
}

// Models enumerates the known models.
func (p *StaticProvider) Models(_ context.Context) ([]ModelRef, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	refs := make([]ModelRef, 0, len(p.models))
	for _, model := range p.models {
		refs = append(refs, ModelRef{
			Name:        model.Name,
			Description: model.Description,
		})
	}

	return refs, nil
}

// SupportsModel returns true if the model has been registered.
func (p *StaticProvider) SupportsModel(_ context.Context, model string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.models[model]
	return ok, nil
}

// IsModelAvailable returns the cached availability flag.
func (p *StaticProvider) IsModelAvailable(_ context.Context, model string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data, ok := p.models[model]
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}
	return data.Available, nil
}

// ModelMetadata returns the tier and cost information for a model.
func (p *StaticProvider) ModelMetadata(_ context.Context, model string) (ModelMetadata, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data, ok := p.models[model]
	if !ok {
		return ModelMetadata{}, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}
	return ModelMetadata{
		Tier:            data.Tier,
		CostPer1KTokens: data.CostPer1KTokens,
		Description:     data.Description,
	}, nil
}

// ResolveModel returns the static routing information.
func (p *StaticProvider) ResolveModel(_ context.Context, model string) (*ResolvedModel, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data, ok := p.models[model]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}
	if !data.Available {
		return nil, fmt.Errorf("%w: %s via %s", ErrModelUnavailable, model, p.info.Name)
	}
	return &ResolvedModel{
		Name:            data.Name,
		Provider:        p.info,
		Route:           data.Route,
		Tier:            data.Tier,
		CostPer1KTokens: data.CostPer1KTokens,
		Metadata:        map[string]string{},
	}, nil
}

// CheckModelHealth verifies a specific model is available and provides actionable feedback.
func (p *StaticProvider) CheckModelHealth(_ context.Context, modelID string) (*HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	data, ok := p.models[modelID]
	if !ok {
		// Model not found in this provider
		availableModels := make([]string, 0, len(p.models))
		for name := range p.models {
			availableModels = append(availableModels, name)
		}

		return &HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Model '%s' not found in %s provider", modelID, p.info.Name),
			Suggestions: []string{
				fmt.Sprintf("Check model name spelling"),
				fmt.Sprintf("List available models from %s", p.info.Name),
			},
			Details: map[string]interface{}{
				"provider":         p.info.Name,
				"requested_model":  modelID,
				"available_models": availableModels,
			},
		}, nil
	}

	if !data.Available {
		return &HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Model '%s' is registered but currently unavailable", modelID),
			Suggestions: []string{
				"Check provider configuration",
				"Verify model is properly configured",
			},
			Details: map[string]interface{}{
				"provider":           p.info.Name,
				"model_name":         data.Name,
				"tier":               data.Tier,
				"cost_per_1k_tokens": data.CostPer1KTokens,
			},
		}, nil
	}

	return &HealthStatus{
		Healthy:     true,
		Message:     fmt.Sprintf("Model '%s' is available", data.Name),
		Suggestions: nil,
		Details: map[string]interface{}{
			"provider":           p.info.Name,
			"model_name":         data.Name,
			"tier":               data.Tier,
			"cost_per_1k_tokens": data.CostPer1KTokens,
			"route":              data.Route,
		},
	}, nil
}

// SetAvailability updates the availability flag for a model.
func (p *StaticProvider) SetAvailability(model string, available bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.models[model]
	if !ok {
		return
	}
	entry.Available = available
	p.models[model] = entry
}
