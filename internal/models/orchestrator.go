package models

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// AgentTier mirrors Orchestra's tier taxonomy to express capability/cost.
type AgentTier int

const (
	AgentTierFast AgentTier = iota + 1
	AgentTierBalanced
	AgentTierPowerful
)

// ResolvePolicy controls how providers are ranked when selecting a model.
type ResolvePolicy string

const (
	ResolvePolicyAuto             ResolvePolicy = "auto"
	ResolvePolicyLocalFirst       ResolvePolicy = "local_first"
	ResolvePolicyPerformanceFirst ResolvePolicy = "performance_first"
	ResolvePolicyCostOptimized    ResolvePolicy = "cost_optimized"
)

// ParseResolvePolicy converts string input to a ResolvePolicy.
func ParseResolvePolicy(value string) (ResolvePolicy, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(ResolvePolicyAuto):
		return ResolvePolicyAuto, nil
	case string(ResolvePolicyLocalFirst):
		return ResolvePolicyLocalFirst, nil
	case string(ResolvePolicyPerformanceFirst):
		return ResolvePolicyPerformanceFirst, nil
	case string(ResolvePolicyCostOptimized):
		return ResolvePolicyCostOptimized, nil
	default:
		return ResolvePolicy(""), fmt.Errorf("unknown model policy: %s", value)
	}
}

// ModelMetadata describes qualitative and cost characteristics for a model.
type ModelMetadata struct {
	Tier            AgentTier
	CostPer1KTokens float64
	Description     string
}

// ProviderType distinguishes between local and cloud-backed model providers.
type ProviderType string

const (
	ProviderTypeLocal ProviderType = "local"
	ProviderTypeCloud ProviderType = "cloud"
)

// ProviderInfo captures metadata about a model provider.
type ProviderInfo struct {
	Name string
	Type ProviderType
}

// ModelRef describes a model exposed by a provider.
type ModelRef struct {
	Name        string
	Description string
}

// ResolvedModel encapsulates routing details for executing against a model.
type ResolvedModel struct {
	Name            string
	Provider        ProviderInfo
	Route           string
	Tier            AgentTier
	CostPer1KTokens float64
	Metadata        map[string]string
}

// HealthStatus provides detailed health information with actionable suggestions.
type HealthStatus struct {
	Healthy     bool
	Message     string
	Suggestions []string               // User-actionable suggestions
	Details     map[string]interface{} // Additional context for debugging
}

// ModelProvider exposes models that the orchestrator can route to.
type ModelProvider interface {
	Info() ProviderInfo
	Models(ctx context.Context) ([]ModelRef, error)
	SupportsModel(ctx context.Context, model string) (bool, error)
	IsModelAvailable(ctx context.Context, model string) (bool, error)
	ModelMetadata(ctx context.Context, model string) (ModelMetadata, error)
	ResolveModel(ctx context.Context, model string) (*ResolvedModel, error)
	// CheckModelHealth verifies a specific model is available and provides actionable feedback
	CheckModelHealth(ctx context.Context, modelID string) (*HealthStatus, error)
}

// ResolveRequest captures preferences for selecting a model.
type ResolveRequest struct {
	Model             string
	PreferredProvider ProviderType
	FallbackModels    []string
	Policy            ResolvePolicy
}

// RegisteredModel represents an available model/provider pairing.
type RegisteredModel struct {
	Name            string
	Provider        ProviderInfo
	Available       bool
	Tier            AgentTier
	CostPer1KTokens float64
}

// ProviderMetrics captures aggregate execution data for a provider.
type ProviderMetrics struct {
	TotalCalls    int
	SuccessCalls  int
	FailureCalls  int
	TotalLatency  time.Duration
	TotalCostHint float64
}

// SuccessRate returns the ratio of successful calls to total calls.
func (pm *ProviderMetrics) SuccessRate() float64 {
	if pm == nil || pm.TotalCalls == 0 {
		return 0.0
	}
	return float64(pm.SuccessCalls) / float64(pm.TotalCalls)
}

// AverageLatency returns the mean latency across calls.
func (pm *ProviderMetrics) AverageLatency() time.Duration {
	if pm == nil || pm.SuccessCalls == 0 {
		return 0
	}
	return time.Duration(int64(pm.TotalLatency) / int64(pm.SuccessCalls))
}

// Orchestrator provides model selection, routing, and availability checks.
type Orchestrator struct {
	mu        sync.RWMutex
	providers map[string]ModelProvider
	metrics   map[string]*ProviderMetrics
}

var (
	// ErrModelNotFound indicates that no registered provider serves the requested model.
	ErrModelNotFound = errors.New("model not found")
	// ErrModelUnavailable indicates that providers serve the model but none are currently available.
	ErrModelUnavailable = errors.New("model unavailable")
)

const (
	successRateWeight      = 0.7
	costWeight             = 0.3
	preferredProviderBonus = 0.05
	fastTierBonus          = 0.03
	balancedTierBonus      = 0.02
	powerfulTierBonus      = 0.01
	maxResolveAttempts     = 3
	initialResolveBackoff  = 20 * time.Millisecond
	backoffMultiplier      = 2.0
	defaultSuccessBaseline = 0.6
	defaultCostScore       = 0.8
)

type providerOption struct {
	provider ModelProvider
	metadata ModelMetadata
	score    float64
}

// NewOrchestrator creates an empty orchestrator instance.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		providers: make(map[string]ModelProvider),
		metrics:   make(map[string]*ProviderMetrics),
	}
}

// RegisterProvider adds a provider. The most recent registration wins on name clashes.
func (o *Orchestrator) RegisterProvider(provider ModelProvider) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.providers[provider.Info().Name] = provider
	if _, ok := o.metrics[provider.Info().Name]; !ok {
		o.metrics[provider.Info().Name] = &ProviderMetrics{}
	}
}

// Metrics returns a copy of the provider metrics for inspection.
func (o *Orchestrator) Metrics(provider string) ProviderMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if metrics, ok := o.metrics[provider]; ok && metrics != nil {
		return *metrics
	}
	return ProviderMetrics{}
}

func (o *Orchestrator) recordSuccess(provider string, latency time.Duration, costHint float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	metrics, ok := o.metrics[provider]
	if !ok || metrics == nil {
		metrics = &ProviderMetrics{}
		o.metrics[provider] = metrics
	}
	metrics.TotalCalls++
	metrics.SuccessCalls++
	metrics.TotalLatency += latency
	metrics.TotalCostHint += costHint
}

func (o *Orchestrator) recordFailure(provider string, latency time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	metrics, ok := o.metrics[provider]
	if !ok || metrics == nil {
		metrics = &ProviderMetrics{}
		o.metrics[provider] = metrics
	}
	metrics.TotalCalls++
	metrics.FailureCalls++
	metrics.TotalLatency += latency
}

// ResolveModel returns the best available route for the supplied request.
func (o *Orchestrator) ResolveModel(ctx context.Context, req ResolveRequest) (*ResolvedModel, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model name is required")
	}

	policy := req.Policy
	if policy == "" {
		policy = ResolvePolicyAuto
	}

	preferred := req.PreferredProvider
	if preferred == "" && policy == ResolvePolicyLocalFirst {
		preferred = ProviderTypeLocal
	}

	candidates := uniqueStrings(append([]string{req.Model}, req.FallbackModels...))
	var lastErr error

	for _, candidate := range candidates {
		resolved, err := o.resolveSingle(ctx, candidate, preferred, policy)
		if err == nil {
			return resolved, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no candidates provided")
	}

	return nil, lastErr
}

// IsAvailable returns true if any provider exposes the supplied model and is currently reachable.
func (o *Orchestrator) IsAvailable(ctx context.Context, model string) (bool, error) {
	_, err := o.resolveSingle(ctx, model, "", ResolvePolicyAuto)
	if err != nil {
		if errors.Is(err, ErrModelNotFound) || errors.Is(err, ErrModelUnavailable) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListModels enumerates all registered provider/model pairs with availability metadata.
func (o *Orchestrator) ListModels(ctx context.Context) ([]RegisteredModel, error) {
	o.mu.RLock()
	providers := make([]ModelProvider, 0, len(o.providers))
	for _, provider := range o.providers {
		providers = append(providers, provider)
	}
	o.mu.RUnlock()

	results := make([]RegisteredModel, 0)
	for _, provider := range providers {
		refs, err := provider.Models(ctx)
		if err != nil {
			return nil, fmt.Errorf("list models for provider %q: %w", provider.Info().Name, err)
		}
		for _, ref := range refs {
			metadata, err := provider.ModelMetadata(ctx, ref.Name)
			if err != nil {
				return nil, fmt.Errorf("get metadata for %q via %q: %w", ref.Name, provider.Info().Name, err)
			}
			available, err := provider.IsModelAvailable(ctx, ref.Name)
			if err != nil {
				return nil, fmt.Errorf("check availability for %q via %q: %w", ref.Name, provider.Info().Name, err)
			}
			results = append(results, RegisteredModel{
				Name:            ref.Name,
				Provider:        provider.Info(),
				Available:       available,
				Tier:            metadata.Tier,
				CostPer1KTokens: metadata.CostPer1KTokens,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Name == results[j].Name {
			return results[i].Provider.Name < results[j].Provider.Name
		}
		return results[i].Name < results[j].Name
	})

	return results, nil
}

func (o *Orchestrator) resolveSingle(ctx context.Context, model string, preferred ProviderType, policy ResolvePolicy) (*ResolvedModel, error) {
	if policy == "" {
		policy = ResolvePolicyAuto
	}
	o.mu.RLock()
	providers := make([]ModelProvider, 0, len(o.providers))
	for _, provider := range o.providers {
		providers = append(providers, provider)
	}
	o.mu.RUnlock()

	if len(providers) == 0 {
		return nil, fmt.Errorf("no model providers registered")
	}

	options, err := o.orderProviders(ctx, providers, model, preferred, policy)
	if err != nil {
		return nil, err
	}

	var (
		notFoundErr      error
		unavailableErr   error
		lastAvailability error
	)

	for _, option := range options {
		isAvailable, err := option.provider.IsModelAvailable(ctx, model)
		if err != nil {
			lastAvailability = err
			continue
		}
		if !isAvailable {
			lastAvailability = fmt.Errorf("model %q unavailable via provider %q", model, option.provider.Info().Name)
			continue
		}

		resolved, err := o.resolveWithRetry(ctx, option.provider, model, option.metadata)
		if err == nil {
			return resolved, nil
		}

		if errors.Is(err, ErrModelNotFound) {
			notFoundErr = err
			continue
		}
		if errors.Is(err, ErrModelUnavailable) {
			unavailableErr = err
			continue
		}
		unavailableErr = err
	}

	if notFoundErr != nil {
		return nil, notFoundErr
	}

	if unavailableErr != nil || lastAvailability != nil {
		if unavailableErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrModelUnavailable, unavailableErr)
		}
		return nil, fmt.Errorf("%w: %v", ErrModelUnavailable, lastAvailability)
	}

	return nil, fmt.Errorf("%w: %s", ErrModelNotFound, model)
}

func (o *Orchestrator) orderProviders(ctx context.Context, providers []ModelProvider, model string, preferred ProviderType, policy ResolvePolicy) ([]providerOption, error) {
	candidates := make([]providerOption, 0, len(providers))
	errorsByProvider := make([]error, 0)

	for _, provider := range providers {
		supports, err := provider.SupportsModel(ctx, model)
		if err != nil {
			errorsByProvider = append(errorsByProvider, fmt.Errorf("checking model support via %q: %w", provider.Info().Name, err))
			continue
		}
		if !supports {
			continue
		}
		metadata, err := provider.ModelMetadata(ctx, model)
		if err != nil {
			errorsByProvider = append(errorsByProvider, fmt.Errorf("metadata for %q via %q: %w", model, provider.Info().Name, err))
			continue
		}
		candidates = append(candidates, providerOption{
			provider: provider,
			metadata: metadata,
		})
	}

	if len(candidates) == 0 {
		if len(errorsByProvider) > 0 {
			return nil, errorsByProvider[0]
		}
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	minCost, maxCost := math.MaxFloat64, 0.0
	for _, candidate := range candidates {
		cost := candidate.metadata.CostPer1KTokens
		if cost <= 0 {
			continue
		}
		if cost < minCost {
			minCost = cost
		}
		if cost > maxCost {
			maxCost = cost
		}
	}
	if minCost == math.MaxFloat64 {
		minCost = 0
		maxCost = 0
	}

	successWeight := successRateWeight
	costWeight := costWeight
	tierScale := 1.0
	localPreferenceBonus := 0.0

	switch policy {
	case ResolvePolicyLocalFirst:
		successWeight = 0.75
		costWeight = 0.15
		tierScale = 0.9
		localPreferenceBonus = 0.2
	case ResolvePolicyPerformanceFirst:
		successWeight = 0.85
		costWeight = 0.15
		tierScale = 1.2
	case ResolvePolicyCostOptimized:
		successWeight = 0.45
		costWeight = 0.55
		tierScale = 0.8
	}

	for idx := range candidates {
		info := candidates[idx].provider.Info()
		metrics := o.Metrics(info.Name)
		successRate := metrics.SuccessRate()
		if metrics.TotalCalls == 0 {
			successRate = defaultSuccessBaseline
		}

		costScore := computeCostScore(candidates[idx].metadata.CostPer1KTokens, minCost, maxCost)
		tierScore := tierScale * tierBonus(candidates[idx].metadata.Tier)

		score := successWeight*successRate + costWeight*costScore + tierScore
		if preferred != "" && info.Type == preferred {
			score += preferredProviderBonus
		}
		if policy == ResolvePolicyLocalFirst && info.Type == ProviderTypeLocal {
			score += localPreferenceBonus
		}
		if policy == ResolvePolicyCostOptimized && info.Type == ProviderTypeCloud {
			score -= 0.02
		}
		candidates[idx].score = score
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			if candidates[i].provider.Info().Type == candidates[j].provider.Info().Type {
				return candidates[i].provider.Info().Name < candidates[j].provider.Info().Name
			}
			return candidates[i].provider.Info().Type == ProviderTypeLocal
		}
		return candidates[i].score > candidates[j].score
	})

	return candidates, nil
}

func (o *Orchestrator) resolveWithRetry(ctx context.Context, provider ModelProvider, model string, metadata ModelMetadata) (*ResolvedModel, error) {
	delay := initialResolveBackoff
	for attempt := 1; attempt <= maxResolveAttempts; attempt++ {
		start := time.Now()
		resolved, err := provider.ResolveModel(ctx, model)
		latency := time.Since(start)
		if err == nil {
			if resolved != nil {
				if resolved.Tier == 0 {
					resolved.Tier = metadata.Tier
				}
				if resolved.CostPer1KTokens == 0 {
					resolved.CostPer1KTokens = metadata.CostPer1KTokens
				}
			}
			o.recordSuccess(provider.Info().Name, latency, metadata.CostPer1KTokens)
			return resolved, nil
		}

		o.recordFailure(provider.Info().Name, latency)
		if attempt == maxResolveAttempts {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
		delay = time.Duration(float64(delay) * backoffMultiplier)
	}
	return nil, fmt.Errorf("exhausted resolve attempts for %s", provider.Info().Name)
}

func computeCostScore(cost, minCost, maxCost float64) float64 {
	if cost <= 0 || minCost == maxCost {
		return defaultCostScore
	}
	if cost < minCost {
		cost = minCost
	}
	if cost > maxCost {
		cost = maxCost
	}
	return 1 - (cost-minCost)/(maxCost-minCost)
}

func tierBonus(tier AgentTier) float64 {
	switch tier {
	case AgentTierFast:
		return fastTierBonus
	case AgentTierBalanced:
		return balancedTierBonus
	case AgentTierPowerful:
		return powerfulTierBonus
	default:
		return 0
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}
