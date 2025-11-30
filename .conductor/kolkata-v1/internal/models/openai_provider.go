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

// OpenAIProvider integrates the OpenAI Chat Completion API.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client

	info ProviderInfo

	mu          sync.RWMutex
	modelsCache map[string]openaiModel
	lastFetch   time.Time
	cacheTTL    time.Duration
}

type openaiModel struct {
	ID string `json:"id"`
}

type openaiModelList struct {
	Data []openaiModel `json:"data"`
}

type openaiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openaiChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openaiChatChoice struct {
	Message      openaiChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openaiChatResponse struct {
	Choices []openaiChatChoice `json:"choices"`
}

// NewOpenAIProvider constructs a provider using the supplied API key and base URL.
func NewOpenAIProvider(apiKey string, baseURL string, client *http.Client) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai api key is required")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &OpenAIProvider{
		apiKey:      apiKey,
		baseURL:     strings.TrimRight(baseURL, "/"),
		client:      client,
		info:        ProviderInfo{Name: "openai-live", Type: ProviderTypeCloud},
		modelsCache: make(map[string]openaiModel),
		cacheTTL:    2 * time.Minute,
	}, nil
}

func (p *OpenAIProvider) Info() ProviderInfo {
	return p.info
}

func (p *OpenAIProvider) Models(ctx context.Context) ([]ModelRef, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return nil, err
	}

	refs := make([]ModelRef, 0, len(models))
	for _, mdl := range models {
		refs = append(refs, ModelRef{
			Name:        mdl.ID,
			Description: mdl.ID,
		})
	}
	return refs, nil
}

func (p *OpenAIProvider) SupportsModel(ctx context.Context, model string) (bool, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return false, err
	}
	_, ok := models[model]
	return ok, nil
}

func (p *OpenAIProvider) IsModelAvailable(ctx context.Context, model string) (bool, error) {
	return p.SupportsModel(ctx, model)
}

// CheckModelHealth verifies a specific model is available and provides actionable feedback.
func (p *OpenAIProvider) CheckModelHealth(ctx context.Context, modelID string) (*HealthStatus, error) {
	// Check if API key is set
	if p.apiKey == "" {
		return &HealthStatus{
			Healthy: false,
			Message: "OpenAI API key not configured",
			Suggestions: []string{
				"Set OPENAI_API_KEY environment variable",
				"Get API key from: https://platform.openai.com/api-keys",
				"Update configuration with valid API key",
			},
			Details: map[string]interface{}{
				"provider": "openai",
				"base_url": p.baseURL,
			},
		}, nil
	}

	// Fetch available models
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return &HealthStatus{
			Healthy: false,
			Message: "Unable to fetch models from OpenAI API",
			Suggestions: []string{
				"Check your API key is valid",
				"Verify internet connection",
				"Check OpenAI API status: https://status.openai.com/",
				"Ensure OPENAI_API_KEY is set correctly",
			},
			Details: map[string]interface{}{
				"provider": "openai",
				"base_url": p.baseURL,
				"error":    err.Error(),
			},
		}, nil
	}

	// Check if model exists in known list
	modelData, found := models[modelID]
	if !found {
		// Build list of available models
		knownModels := make([]string, 0, len(models))
		for id := range models {
			knownModels = append(knownModels, id)
		}

		return &HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Model '%s' not found in OpenAI catalog", modelID),
			Suggestions: []string{
				"Check model ID spelling",
				"List available models: skill models list --provider=openai",
				"Visit OpenAI docs: https://platform.openai.com/docs/models",
			},
			Details: map[string]interface{}{
				"provider":        "openai",
				"requested_model": modelID,
				"known_models":    knownModels,
			},
		}, nil
	}

	// Model is available - get metadata
	metadata, _ := p.ModelMetadata(ctx, modelID)

	return &HealthStatus{
		Healthy:     true,
		Message:     fmt.Sprintf("Model '%s' is available", modelData.ID),
		Suggestions: nil,
		Details: map[string]interface{}{
			"model_id":           modelID,
			"provider":           "openai",
			"tier":               metadata.Tier,
			"cost_per_1k_tokens": metadata.CostPer1KTokens,
			"base_url":           p.baseURL,
		},
	}, nil
}

func (p *OpenAIProvider) ModelMetadata(ctx context.Context, model string) (ModelMetadata, error) {
	if ok, err := p.SupportsModel(ctx, model); err != nil {
		return ModelMetadata{}, err
	} else if !ok {
		return ModelMetadata{}, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	tier := AgentTierBalanced
	cost := 0.01
	switch {
	case strings.HasPrefix(model, "gpt-4"):
		tier = AgentTierPowerful
		cost = 0.03
	case strings.HasPrefix(model, "gpt-3.5"):
		tier = AgentTierBalanced
		cost = 0.015
	case strings.Contains(model, "turbo"):
		tier = AgentTierBalanced
		cost = 0.012
	default:
		tier = AgentTierBalanced
	}

	return ModelMetadata{
		Tier:            tier,
		CostPer1KTokens: cost,
		Description:     model,
	}, nil
}

func (p *OpenAIProvider) ResolveModel(ctx context.Context, model string) (*ResolvedModel, error) {
	if ok, err := p.SupportsModel(ctx, model); err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	meta, _ := p.ModelMetadata(ctx, model)

	return &ResolvedModel{
		Name:            model,
		Provider:        p.Info(),
		Route:           fmt.Sprintf("%s/v1/chat/completions", p.baseURL),
		Tier:            meta.Tier,
		CostPer1KTokens: meta.CostPer1KTokens,
		Metadata: map[string]string{
			"base_url": p.baseURL,
		},
	}, nil
}

func (p *OpenAIProvider) Generate(ctx context.Context, model, prompt string, stream bool, opts map[string]interface{}) (string, error) {
	if stream {
		return "", errors.New("openai streaming not implemented")
	}

	request := openaiChatRequest{
		Model: model,
		Messages: []openaiChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	if opts != nil {
		if val, ok := opts["temperature"].(float64); ok {
			request.Temperature = val
		}
		if val, ok := opts["max_tokens"].(int); ok {
			request.MaxTokens = val
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/chat/completions", p.baseURL), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("chat completion failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return "", errors.New("openai response contained no choices")
	}

	return parsed.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) fetchModels(ctx context.Context, force bool) (map[string]openaiModel, error) {
	p.mu.RLock()
	if !force && time.Since(p.lastFetch) < p.cacheTTL && len(p.modelsCache) > 0 {
		defer p.mu.RUnlock()
		return cloneOpenAIModelMap(p.modelsCache), nil
	}
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/models", p.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create model list request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("fetch models: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed openaiModelList
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}

	models := make(map[string]openaiModel, len(parsed.Data))
	for _, mdl := range parsed.Data {
		models[mdl.ID] = mdl
	}

	p.mu.Lock()
	p.modelsCache = models
	p.lastFetch = time.Now()
	p.mu.Unlock()

	return cloneOpenAIModelMap(models), nil
}

func cloneOpenAIModelMap(input map[string]openaiModel) map[string]openaiModel {
	output := make(map[string]openaiModel, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
