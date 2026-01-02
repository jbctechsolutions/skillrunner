// Package workflow provides the workflow executor for skill execution.
package workflow

import (
	"context"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// CachingPhaseExecutor wraps a phase executor with caching capabilities.
type CachingPhaseExecutor struct {
	delegate   *phaseExecutor
	cache      ports.ResponseCachePort
	enabled    bool
	defaultTTL time.Duration
	// Fingerprinter is optional; if nil, uses default fingerprinting
	Fingerprinter func(ports.CompletionRequest) string
}

// CachingConfig holds configuration for the caching executor.
type CachingConfig struct {
	Enabled    bool
	DefaultTTL time.Duration
}

// NewCachingPhaseExecutor creates a new caching phase executor.
func NewCachingPhaseExecutor(provider ports.ProviderPort, cache ports.ResponseCachePort, cfg CachingConfig) *CachingPhaseExecutor {
	return &CachingPhaseExecutor{
		delegate:   newPhaseExecutor(provider),
		cache:      cache,
		enabled:    cfg.Enabled,
		defaultTTL: cfg.DefaultTTL,
	}
}

// Execute runs a single phase with caching support.
func (e *CachingPhaseExecutor) Execute(ctx context.Context, phase *skill.Phase, dependencyOutputs map[string]string) *PhaseResult {
	if !e.enabled || e.cache == nil {
		return e.delegate.Execute(ctx, phase, dependencyOutputs)
	}

	result := &PhaseResult{
		PhaseID:   phase.ID,
		PhaseName: phase.Name,
		Status:    PhaseStatusRunning,
		StartTime: time.Now(),
	}

	// Build the prompt and request to generate cache key
	prompt, err := e.delegate.buildPrompt(phase.PromptTemplate, dependencyOutputs)
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Build the completion request
	req := ports.CompletionRequest{
		ModelID:     e.delegate.selectModel(phase.RoutingProfile),
		Messages:    e.delegate.buildMessages(prompt, dependencyOutputs),
		MaxTokens:   phase.MaxTokens,
		Temperature: phase.Temperature,
	}

	// Generate cache key
	cacheKey := e.fingerprint(req)

	// Try to get from cache
	if cachedResp, found := e.cache.GetResponse(ctx, cacheKey); found {
		result.Status = PhaseStatusCompleted
		result.Output = cachedResp.Content
		result.InputTokens = cachedResp.InputTokens
		result.OutputTokens = cachedResp.OutputTokens
		result.ModelUsed = cachedResp.ModelUsed
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.CacheHit = true
		return result
	}

	// Cache miss - call provider
	resp, err := e.delegate.provider.Complete(ctx, req)
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Store in cache
	ttl := e.defaultTTL
	if ttl == 0 {
		ttl = 24 * time.Hour // Default 24h TTL
	}
	_ = e.cache.SetResponse(ctx, cacheKey, resp, ttl)

	// Populate the result
	result.Status = PhaseStatusCompleted
	result.Output = resp.Content
	result.InputTokens = resp.InputTokens
	result.OutputTokens = resp.OutputTokens
	result.ModelUsed = resp.ModelUsed
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.CacheHit = false

	return result
}

// fingerprint generates a cache key for the request.
func (e *CachingPhaseExecutor) fingerprint(req ports.CompletionRequest) string {
	if e.Fingerprinter != nil {
		return e.Fingerprinter(req)
	}
	// Use default fingerprinting from cache package
	return defaultFingerprint(req)
}

// defaultFingerprint creates a simple hash-based fingerprint.
// This is a fallback; the cache package has a more robust implementation.
func defaultFingerprint(req ports.CompletionRequest) string {
	// Build a simple key from request components
	key := req.ModelID + ":"
	for _, msg := range req.Messages {
		key += msg.Role + ":" + msg.Content + "|"
	}
	return key
}

// CachingStreamingPhaseExecutor wraps a streaming phase executor with caching.
type CachingStreamingPhaseExecutor struct {
	delegate   *streamingPhaseExecutor
	cache      ports.ResponseCachePort
	enabled    bool
	defaultTTL time.Duration
	// Fingerprinter is optional; if nil, uses default fingerprinting
	Fingerprinter func(ports.CompletionRequest) string
}

// NewCachingStreamingPhaseExecutor creates a new caching streaming phase executor.
func NewCachingStreamingPhaseExecutor(provider ports.ProviderPort, cache ports.ResponseCachePort, cfg CachingConfig) *CachingStreamingPhaseExecutor {
	return &CachingStreamingPhaseExecutor{
		delegate:   newStreamingPhaseExecutor(provider),
		cache:      cache,
		enabled:    cfg.Enabled,
		defaultTTL: cfg.DefaultTTL,
	}
}

// ExecuteWithStreaming runs a single phase with streaming and caching support.
func (e *CachingStreamingPhaseExecutor) ExecuteWithStreaming(
	ctx context.Context,
	phase *skill.Phase,
	dependencyOutputs map[string]string,
	callback PhaseStreamCallback,
) *PhaseResult {
	if !e.enabled || e.cache == nil {
		return e.delegate.ExecuteWithStreaming(ctx, phase, dependencyOutputs, callback)
	}

	result := &PhaseResult{
		PhaseID:   phase.ID,
		PhaseName: phase.Name,
		Status:    PhaseStatusRunning,
		StartTime: time.Now(),
	}

	// Build the prompt and request to generate cache key
	prompt, err := e.delegate.buildPrompt(phase.PromptTemplate, dependencyOutputs)
	if err != nil {
		result.Status = PhaseStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Build the completion request
	req := ports.CompletionRequest{
		ModelID:     e.delegate.selectModel(phase.RoutingProfile),
		Messages:    e.delegate.buildMessages(prompt, dependencyOutputs),
		MaxTokens:   phase.MaxTokens,
		Temperature: phase.Temperature,
	}

	// Generate cache key
	cacheKey := e.fingerprint(req)

	// Try to get from cache
	if cachedResp, found := e.cache.GetResponse(ctx, cacheKey); found {
		// For cache hit, simulate streaming by sending the full content
		if callback != nil {
			_ = callback(cachedResp.Content, cachedResp.InputTokens, cachedResp.OutputTokens)
		}

		result.Status = PhaseStatusCompleted
		result.Output = cachedResp.Content
		result.InputTokens = cachedResp.InputTokens
		result.OutputTokens = cachedResp.OutputTokens
		result.ModelUsed = cachedResp.ModelUsed
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.CacheHit = true
		return result
	}

	// Cache miss - call provider with streaming
	phaseResult := e.delegate.ExecuteWithStreaming(ctx, phase, dependencyOutputs, callback)

	// If successful, store in cache
	if phaseResult.Status == PhaseStatusCompleted {
		resp := &ports.CompletionResponse{
			Content:      phaseResult.Output,
			InputTokens:  phaseResult.InputTokens,
			OutputTokens: phaseResult.OutputTokens,
			ModelUsed:    phaseResult.ModelUsed,
			Duration:     phaseResult.Duration,
		}

		ttl := e.defaultTTL
		if ttl == 0 {
			ttl = 24 * time.Hour
		}
		_ = e.cache.SetResponse(ctx, cacheKey, resp, ttl)
	}

	phaseResult.CacheHit = false
	return phaseResult
}

// Execute runs a single phase without streaming (for compatibility).
func (e *CachingStreamingPhaseExecutor) Execute(ctx context.Context, phase *skill.Phase, dependencyOutputs map[string]string) *PhaseResult {
	return e.ExecuteWithStreaming(ctx, phase, dependencyOutputs, nil)
}

// fingerprint generates a cache key for the request.
func (e *CachingStreamingPhaseExecutor) fingerprint(req ports.CompletionRequest) string {
	if e.Fingerprinter != nil {
		return e.Fingerprinter(req)
	}
	return defaultFingerprint(req)
}
