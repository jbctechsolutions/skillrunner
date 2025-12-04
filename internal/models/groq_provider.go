package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GroqProvider integrates with the Groq API for ultra-fast LLM inference.
// Groq uses an OpenAI-compatible API format.
type GroqProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client

	info ProviderInfo

	mu          sync.RWMutex
	modelsCache map[string]groqModelInfo
	lastFetch   time.Time
	cacheTTL    time.Duration
}

// groqModelInfo contains information about a Groq model
type groqModelInfo struct {
	ID            string
	OwnedBy       string
	ContextWindow int
	Active        bool
}

type groqModelListResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID            string `json:"id"`
		Object        string `json:"object"`
		Created       int64  `json:"created"`
		OwnedBy       string `json:"owned_by"`
		Active        bool   `json:"active"`
		ContextWindow int    `json:"context_window"`
	} `json:"data"`
}

type groqChatRequest struct {
	Model       string              `json:"model"`
	Messages    []groqChatMessage   `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream"`
}

type groqChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewGroqProvider constructs a provider for Groq's ultra-fast inference API.
func NewGroqProvider(apiKey string, baseURL string, client *http.Client) (*GroqProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("groq api key is required")
	}
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai"
	}
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Minute,
		}
	}

	return &GroqProvider{
		apiKey:      apiKey,
		baseURL:     strings.TrimRight(baseURL, "/"),
		client:      client,
		info:        ProviderInfo{Name: "groq", Type: ProviderTypeCloud},
		modelsCache: make(map[string]groqModelInfo),
		cacheTTL:    5 * time.Minute,
	}, nil
}

func (p *GroqProvider) Info() ProviderInfo {
	return p.info
}

func (p *GroqProvider) Models(ctx context.Context) ([]ModelRef, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return nil, err
	}
	results := make([]ModelRef, 0, len(models))
	for _, model := range models {
		results = append(results, ModelRef{
			Name:        model.ID,
			Description: fmt.Sprintf("%s (context: %d)", model.ID, model.ContextWindow),
		})
	}
	return results, nil
}

func (p *GroqProvider) SupportsModel(ctx context.Context, model string) (bool, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return false, err
	}
	_, ok := models[model]
	return ok, nil
}

func (p *GroqProvider) IsModelAvailable(ctx context.Context, model string) (bool, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return false, err
	}
	info, ok := models[model]
	if !ok {
		return false, nil
	}
	return info.Active, nil
}

// CheckModelHealth verifies a specific model is available and provides actionable feedback.
func (p *GroqProvider) CheckModelHealth(ctx context.Context, modelID string) (*HealthStatus, error) {
	if p.apiKey == "" {
		return &HealthStatus{
			Healthy: false,
			Message: "Groq API key not configured",
			Suggestions: []string{
				"Set GROQ_API_KEY environment variable",
				"Get API key from: https://console.groq.com/",
				"Update configuration with valid API key",
			},
			Details: map[string]any{
				"provider": "groq",
				"base_url": p.baseURL,
			},
		}, nil
	}

	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return &HealthStatus{
			Healthy: false,
			Message: "Unable to fetch models from Groq API",
			Suggestions: []string{
				"Check your API key is valid",
				"Verify internet connection",
				"Check Groq API status",
				"Ensure GROQ_API_KEY is set correctly",
			},
			Details: map[string]any{
				"provider": "groq",
				"base_url": p.baseURL,
				"error":    err.Error(),
			},
		}, nil
	}

	info, found := models[modelID]
	if !found {
		knownModels := make([]string, 0, len(models))
		for id := range models {
			knownModels = append(knownModels, id)
		}

		return &HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Model '%s' not found in Groq catalog", modelID),
			Suggestions: []string{
				"Check model ID spelling",
				"List available models: skill models list --provider=groq",
				"Visit Groq docs: https://console.groq.com/docs/models",
			},
			Details: map[string]any{
				"provider":        "groq",
				"requested_model": modelID,
				"known_models":    knownModels,
			},
		}, nil
	}

	if !info.Active {
		return &HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Model '%s' is not currently active", modelID),
			Suggestions: []string{
				"Try a different model",
				"Check Groq API status",
			},
			Details: map[string]any{
				"provider": "groq",
				"model_id": modelID,
				"active":   false,
			},
		}, nil
	}

	metadata, _ := p.ModelMetadata(ctx, modelID)

	return &HealthStatus{
		Healthy:     true,
		Message:     fmt.Sprintf("Model '%s' is available", modelID),
		Suggestions: nil,
		Details: map[string]any{
			"model_id":           modelID,
			"provider":           "groq",
			"tier":               metadata.Tier,
			"cost_per_1k_tokens": metadata.CostPer1KTokens,
			"context_window":     info.ContextWindow,
			"base_url":           p.baseURL,
		},
	}, nil
}

func (p *GroqProvider) ModelMetadata(ctx context.Context, model string) (ModelMetadata, error) {
	if ok, err := p.SupportsModel(ctx, model); err != nil {
		return ModelMetadata{}, err
	} else if !ok {
		return ModelMetadata{}, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	// Set tier and cost based on model
	// Groq pricing (last verified: 2024-01). Pricing may have changed.
	// - llama-3.1-70b-versatile: $0.59/$0.79 per 1M tokens (~$0.0007 per 1K)
	// - llama-3.1-8b-instant: $0.05/$0.08 per 1M tokens (~$0.00007 per 1K)
	// - mixtral-8x7b-32768: $0.24/$0.24 per 1M tokens (~$0.00024 per 1K)
	// For current pricing, see: https://console.groq.com/docs/pricing
	tier := AgentTierFast
	cost := 0.0003 // Default cost per 1K tokens (average)

	switch {
	case strings.Contains(model, "70b"):
		tier = AgentTierPowerful
		cost = 0.0007
	case strings.Contains(model, "8b"):
		tier = AgentTierFast
		cost = 0.00007
	case strings.Contains(model, "mixtral"):
		tier = AgentTierBalanced
		cost = 0.00024
	}

	return ModelMetadata{
		Tier:            tier,
		CostPer1KTokens: cost,
		Description:     fmt.Sprintf("Groq ultra-fast inference: %s", model),
	}, nil
}

func (p *GroqProvider) ResolveModel(ctx context.Context, model string) (*ResolvedModel, error) {
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

// Generate issues a completion request to Groq. When stream is true,
// the returned channel yields partial responses terminated by a closed channel.
func (p *GroqProvider) Generate(ctx context.Context, model, prompt string, stream bool, opts map[string]any) (<-chan string, error) {
	request := groqChatRequest{
		Model: model,
		Messages: []groqChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: stream,
	}

	if maxTokens, ok := opts["max_tokens"].(int); ok {
		request.MaxTokens = maxTokens
	}
	if temp, ok := opts["temperature"].(float64); ok {
		request.Temperature = temp
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/chat/completions", p.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	output := make(chan string)

	go func() {
		defer resp.Body.Close()
		defer close(output)

		if resp.StatusCode != http.StatusOK {
			payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			select {
			case <-ctx.Done():
			case output <- fmt.Sprintf("error: groq request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload))):
			}
			return
		}

		if !stream {
			var parsed groqChatResponse
			if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
				select {
				case <-ctx.Done():
				case output <- fmt.Sprintf("error: %v", err):
				}
				return
			}

			if len(parsed.Choices) > 0 {
				select {
				case <-ctx.Done():
					return
				case output <- parsed.Choices[0].Message.Content:
				}
			}
			return
		}

		// Handle streaming response (SSE format)
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				lineStr := strings.TrimSpace(string(line))
				if data, ok := strings.CutPrefix(lineStr, "data: "); ok {
					if data == "[DONE]" {
						return
					}

					var event groqChatResponse
					if err := json.Unmarshal([]byte(data), &event); err == nil {
						if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
							select {
							case <-ctx.Done():
								return
							case output <- event.Choices[0].Delta.Content:
							}
						}
					}
				}
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					select {
					case <-ctx.Done():
					case output <- fmt.Sprintf("error: %v", err):
					}
				}
				return
			}
		}
	}()

	return output, nil
}

func (p *GroqProvider) fetchModels(ctx context.Context, force bool) (map[string]groqModelInfo, error) {
	p.mu.RLock()
	if !force && time.Since(p.lastFetch) < p.cacheTTL && len(p.modelsCache) > 0 {
		defer p.mu.RUnlock()
		return cloneGroqModelMap(p.modelsCache), nil
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

	var parsed groqModelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}

	models := make(map[string]groqModelInfo, len(parsed.Data))
	for _, mdl := range parsed.Data {
		models[mdl.ID] = groqModelInfo{
			ID:            mdl.ID,
			OwnedBy:       mdl.OwnedBy,
			ContextWindow: mdl.ContextWindow,
			Active:        mdl.Active,
		}
	}

	p.mu.Lock()
	p.modelsCache = models
	p.lastFetch = time.Now()
	p.mu.Unlock()

	return cloneGroqModelMap(models), nil
}

func cloneGroqModelMap(input map[string]groqModelInfo) map[string]groqModelInfo {
	out := make(map[string]groqModelInfo, len(input))
	maps.Copy(out, input)
	return out
}
