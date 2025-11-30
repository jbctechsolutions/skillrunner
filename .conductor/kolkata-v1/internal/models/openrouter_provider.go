package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OpenRouterProvider integrates with the OpenRouter API to access 100+ models.
type OpenRouterProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client

	info ProviderInfo

	mu          sync.RWMutex
	modelsCache map[string]openRouterModel
	lastFetch   time.Time
	cacheTTL    time.Duration
}

type openRouterModel struct {
	ID                string
	Name              string
	Description       string
	ContextLength     int
	PricingPrompt     float64
	PricingCompletion float64
}

type openRouterModelsResponse struct {
	Data []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		ContextLength int    `json:"context_length"`
		Pricing       struct {
			Prompt     string `json:"prompt"`
			Completion string `json:"completion"`
		} `json:"pricing"`
	} `json:"data"`
}

// NewOpenRouterProvider constructs a provider for OpenRouter's API.
func NewOpenRouterProvider(apiKey, baseURL string) (*OpenRouterProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openrouter api key is required")
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	return &OpenRouterProvider{
		apiKey:      apiKey,
		baseURL:     strings.TrimRight(baseURL, "/"),
		client:      &http.Client{Timeout: 30 * time.Second},
		info:        ProviderInfo{Name: "openrouter", Type: ProviderTypeCloud},
		modelsCache: make(map[string]openRouterModel),
		cacheTTL:    5 * time.Minute,
	}, nil
}

// Info returns provider metadata.
func (p *OpenRouterProvider) Info() ProviderInfo {
	return p.info
}

// Models lists all models available from OpenRouter.
func (p *OpenRouterProvider) Models(ctx context.Context) ([]ModelRef, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return nil, err
	}

	refs := make([]ModelRef, 0, len(models))
	for _, model := range models {
		desc := model.Description
		if desc == "" {
			desc = model.Name
		}
		refs = append(refs, ModelRef{
			Name:        model.ID,
			Description: desc,
		})
	}
	return refs, nil
}

// SupportsModel reports whether the model exists in OpenRouter's catalog.
func (p *OpenRouterProvider) SupportsModel(ctx context.Context, model string) (bool, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return false, err
	}
	_, ok := models[model]
	return ok, nil
}

// IsModelAvailable returns true when the model is present in the OpenRouter catalog.
// For cloud providers, this is equivalent to SupportsModel.
func (p *OpenRouterProvider) IsModelAvailable(ctx context.Context, model string) (bool, error) {
	return p.SupportsModel(ctx, model)
}

// ModelMetadata returns tier and pricing information for a model.
func (p *OpenRouterProvider) ModelMetadata(ctx context.Context, model string) (ModelMetadata, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return ModelMetadata{}, err
	}
	data, ok := models[model]
	if !ok {
		return ModelMetadata{}, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	// Determine tier based on model name patterns
	tier := AgentTierBalanced
	modelLower := strings.ToLower(model)

	// Powerful tier: GPT-4, Claude Opus/Sonnet, Gemini Pro
	if strings.Contains(modelLower, "gpt-4") ||
		strings.Contains(modelLower, "claude") && (strings.Contains(modelLower, "opus") || strings.Contains(modelLower, "sonnet")) ||
		strings.Contains(modelLower, "gemini") && strings.Contains(modelLower, "pro") {
		tier = AgentTierPowerful
	}
	// Fast tier: smaller/cheaper models
	if strings.Contains(modelLower, "gpt-3.5") ||
		strings.Contains(modelLower, "haiku") ||
		strings.Contains(modelLower, "llama") && strings.Contains(modelLower, "8b") ||
		strings.Contains(modelLower, "mistral") && strings.Contains(modelLower, "7b") ||
		strings.Contains(modelLower, "gemini") && strings.Contains(modelLower, "flash") {
		tier = AgentTierFast
	}

	// Calculate average cost per 1K tokens (average of prompt and completion)
	// OpenRouter pricing is per million tokens, so we divide by 1000
	avgCost := (data.PricingPrompt + data.PricingCompletion) / 2000.0

	return ModelMetadata{
		Tier:            tier,
		CostPer1KTokens: avgCost,
		Description:     data.Description,
	}, nil
}

// ResolveModel ensures the model exists and returns route metadata.
func (p *OpenRouterProvider) ResolveModel(ctx context.Context, model string) (*ResolvedModel, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return nil, err
	}
	data, ok := models[model]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	meta, _ := p.ModelMetadata(ctx, model)

	return &ResolvedModel{
		Name:            data.ID,
		Provider:        p.info,
		Route:           fmt.Sprintf("%s/chat/completions", p.baseURL),
		Tier:            meta.Tier,
		CostPer1KTokens: meta.CostPer1KTokens,
		Metadata: map[string]string{
			"base_url":       p.baseURL,
			"context_length": strconv.Itoa(data.ContextLength),
			"display_name":   data.Name,
		},
	}, nil
}

func (p *OpenRouterProvider) fetchModels(ctx context.Context, force bool) (map[string]openRouterModel, error) {
	// Check cache first
	p.mu.RLock()
	if !force && time.Since(p.lastFetch) < p.cacheTTL && len(p.modelsCache) > 0 {
		defer p.mu.RUnlock()
		return cloneOpenRouterModelMap(p.modelsCache), nil
	}
	p.mu.RUnlock()

	// Fetch from API
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/models", p.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("fetch models: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed openRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}

	// Convert to internal model map
	models := make(map[string]openRouterModel, len(parsed.Data))
	for _, mdl := range parsed.Data {
		// Parse pricing strings to float64
		// OpenRouter returns pricing as strings like "0.000001" (per token)
		// We need to convert to per million tokens
		promptPrice := parsePrice(mdl.Pricing.Prompt)
		completionPrice := parsePrice(mdl.Pricing.Completion)

		models[mdl.ID] = openRouterModel{
			ID:                mdl.ID,
			Name:              mdl.Name,
			Description:       mdl.Description,
			ContextLength:     mdl.ContextLength,
			PricingPrompt:     promptPrice * 1_000_000, // Convert to per million tokens
			PricingCompletion: completionPrice * 1_000_000,
		}
	}

	// Update cache
	p.mu.Lock()
	p.modelsCache = models
	p.lastFetch = time.Now()
	p.mu.Unlock()

	return cloneOpenRouterModelMap(models), nil
}

func parsePrice(priceStr string) float64 {
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0
	}
	return price
}

func cloneOpenRouterModelMap(input map[string]openRouterModel) map[string]openRouterModel {
	out := make(map[string]openRouterModel, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
