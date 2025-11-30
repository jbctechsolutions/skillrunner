package models

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AnthropicProvider integrates with the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client

	info ProviderInfo

	mu          sync.RWMutex
	modelsCache map[string]struct{}
	lastFetch   time.Time
	cacheTTL    time.Duration
}

type anthropicModelList struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type anthropicMessageRequest struct {
	Model     string                    `json:"model"`
	MaxTokens int                       `json:"max_tokens"`
	Messages  []anthropicMessageContent `json:"messages"`
	Stream    bool                      `json:"stream"`
}

type anthropicMessageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicMessageResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// NewAnthropicProvider constructs a provider for Anthropic's API.
func NewAnthropicProvider(apiKey string, baseURL string, client *http.Client) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &AnthropicProvider{
		apiKey:      apiKey,
		baseURL:     strings.TrimRight(baseURL, "/"),
		client:      client,
		info:        ProviderInfo{Name: "anthropic-live", Type: ProviderTypeCloud},
		modelsCache: make(map[string]struct{}),
		cacheTTL:    2 * time.Minute,
	}, nil
}

func (p *AnthropicProvider) Info() ProviderInfo {
	return p.info
}

func (p *AnthropicProvider) Models(ctx context.Context) ([]ModelRef, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return nil, err
	}
	results := make([]ModelRef, 0, len(models))
	for id := range models {
		results = append(results, ModelRef{
			Name:        id,
			Description: id,
		})
	}
	return results, nil
}

func (p *AnthropicProvider) SupportsModel(ctx context.Context, model string) (bool, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return false, err
	}
	_, ok := models[model]
	return ok, nil
}

func (p *AnthropicProvider) IsModelAvailable(ctx context.Context, model string) (bool, error) {
	return p.SupportsModel(ctx, model)
}

// CheckModelHealth verifies a specific model is available and provides actionable feedback.
func (p *AnthropicProvider) CheckModelHealth(ctx context.Context, modelID string) (*HealthStatus, error) {
	// Check if API key is set
	if p.apiKey == "" {
		return &HealthStatus{
			Healthy: false,
			Message: "Anthropic API key not configured",
			Suggestions: []string{
				"Set ANTHROPIC_API_KEY environment variable",
				"Get API key from: https://console.anthropic.com/",
				"Update configuration with valid API key",
			},
			Details: map[string]interface{}{
				"provider": "anthropic",
				"base_url": p.baseURL,
			},
		}, nil
	}

	// Fetch available models
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return &HealthStatus{
			Healthy: false,
			Message: "Unable to fetch models from Anthropic API",
			Suggestions: []string{
				"Check your API key is valid",
				"Verify internet connection",
				"Check Anthropic API status: https://status.anthropic.com/",
				"Ensure ANTHROPIC_API_KEY is set correctly",
			},
			Details: map[string]interface{}{
				"provider": "anthropic",
				"base_url": p.baseURL,
				"error":    err.Error(),
			},
		}, nil
	}

	// Check if model exists in known list
	if _, found := models[modelID]; !found {
		// Build list of available models
		knownModels := make([]string, 0, len(models))
		for id := range models {
			knownModels = append(knownModels, id)
		}

		return &HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Model '%s' not found in Anthropic catalog", modelID),
			Suggestions: []string{
				"Check model ID spelling",
				"List available models: skill models list --provider=anthropic",
				"Visit Anthropic docs: https://docs.anthropic.com/claude/docs/models-overview",
			},
			Details: map[string]interface{}{
				"provider":        "anthropic",
				"requested_model": modelID,
				"known_models":    knownModels,
			},
		}, nil
	}

	// Model is available - get metadata
	metadata, _ := p.ModelMetadata(ctx, modelID)

	return &HealthStatus{
		Healthy:     true,
		Message:     fmt.Sprintf("Model '%s' is available", modelID),
		Suggestions: nil,
		Details: map[string]interface{}{
			"model_id":           modelID,
			"provider":           "anthropic",
			"tier":               metadata.Tier,
			"cost_per_1k_tokens": metadata.CostPer1KTokens,
			"base_url":           p.baseURL,
		},
	}, nil
}

func (p *AnthropicProvider) ModelMetadata(ctx context.Context, model string) (ModelMetadata, error) {
	if ok, err := p.SupportsModel(ctx, model); err != nil {
		return ModelMetadata{}, err
	} else if !ok {
		return ModelMetadata{}, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	tier := AgentTierBalanced
	cost := 0.02
	if strings.Contains(model, "opus") {
		tier = AgentTierPowerful
		cost = 0.04
	} else if strings.Contains(model, "sonnet") {
		tier = AgentTierBalanced
		cost = 0.025
	} else if strings.Contains(model, "haiku") {
		tier = AgentTierFast
		cost = 0.01
	}

	return ModelMetadata{
		Tier:            tier,
		CostPer1KTokens: cost,
		Description:     model,
	}, nil
}

func (p *AnthropicProvider) ResolveModel(ctx context.Context, model string) (*ResolvedModel, error) {
	if ok, err := p.SupportsModel(ctx, model); err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	meta, _ := p.ModelMetadata(ctx, model)

	return &ResolvedModel{
		Name:            model,
		Provider:        p.Info(),
		Route:           fmt.Sprintf("%s/v1/messages", p.baseURL),
		Tier:            meta.Tier,
		CostPer1KTokens: meta.CostPer1KTokens,
		Metadata: map[string]string{
			"base_url": p.baseURL,
		},
	}, nil
}

func (p *AnthropicProvider) Generate(ctx context.Context, model, prompt string, stream bool, opts map[string]interface{}) (string, error) {
	if stream {
		return "", errors.New("anthropic streaming not implemented")
	}

	request := anthropicMessageRequest{
		Model:     model,
		MaxTokens: 1024,
		Messages: []anthropicMessageContent{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/messages", p.baseURL), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("anthropic request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed anthropicMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(parsed.Content) == 0 {
		return "", errors.New("anthropic response contained no content")
	}

	return parsed.Content[0].Text, nil
}

func (p *AnthropicProvider) fetchModels(ctx context.Context, force bool) (map[string]struct{}, error) {
	p.mu.RLock()
	if !force && time.Since(p.lastFetch) < p.cacheTTL && len(p.modelsCache) > 0 {
		defer p.mu.RUnlock()
		return cloneModelSet(p.modelsCache), nil
	}
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/models", p.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create model list request: %w", err)
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("fetch models: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed anthropicModelList
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}

	models := make(map[string]struct{}, len(parsed.Data))
	for _, mdl := range parsed.Data {
		models[mdl.ID] = struct{}{}
	}

	p.mu.Lock()
	p.modelsCache = models
	p.lastFetch = time.Now()
	p.mu.Unlock()

	return cloneModelSet(models), nil
}

func cloneModelSet(input map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(input))
	for key := range input {
		out[key] = struct{}{}
	}
	return out
}
