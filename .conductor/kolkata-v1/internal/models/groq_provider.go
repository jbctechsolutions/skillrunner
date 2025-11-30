package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// GroqProvider integrates the Groq API for ultra-fast inference.
type GroqProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client

	info ProviderInfo

	mu             sync.RWMutex
	validatedCache bool
	validationErr  error
	cacheTTL       time.Duration
}

type groqModelInfo struct {
	Description   string
	ContextWindow int
}

// Known Groq models with metadata.
var groqModels = map[string]groqModelInfo{
	"llama-3.3-70b-versatile": {
		Description:   "Llama 3.3 70B",
		ContextWindow: 128000,
	},
	"llama-3.1-70b-versatile": {
		Description:   "Llama 3.1 70B",
		ContextWindow: 128000,
	},
	"llama-3.1-8b-instant": {
		Description:   "Llama 3.1 8B (Fast)",
		ContextWindow: 128000,
	},
	"mixtral-8x7b-32768": {
		Description:   "Mixtral 8x7B",
		ContextWindow: 32768,
	},
	"gemma2-9b-it": {
		Description:   "Gemma 2 9B",
		ContextWindow: 8192,
	},
}

type groqChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqChatRequest struct {
	Model       string            `json:"model"`
	Messages    []groqChatMessage `json:"messages"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
}

type groqChatChoice struct {
	Message      groqChatMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type groqChatResponse struct {
	Choices []groqChatChoice `json:"choices"`
}

// NewGroqProvider creates a provider using the Groq API.
func NewGroqProvider(apiKey string) (*GroqProvider, error) {
	if apiKey == "" {
		// Try to get from environment
		apiKey = os.Getenv("GROQ_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("groq api key is required (set GROQ_API_KEY environment variable)")
	}

	return &GroqProvider{
		apiKey:  apiKey,
		baseURL: "https://api.groq.com/openai/v1",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		info:     ProviderInfo{Name: "groq", Type: ProviderTypeCloud},
		cacheTTL: 5 * time.Minute,
	}, nil
}

// Info returns provider metadata.
func (p *GroqProvider) Info() ProviderInfo {
	return p.info
}

// Models lists all known Groq models.
func (p *GroqProvider) Models(ctx context.Context) ([]ModelRef, error) {
	refs := make([]ModelRef, 0, len(groqModels))
	for modelID, info := range groqModels {
		refs = append(refs, ModelRef{
			Name:        modelID,
			Description: info.Description,
		})
	}
	return refs, nil
}

// SupportsModel checks if the model is in the known Groq models list.
func (p *GroqProvider) SupportsModel(ctx context.Context, model string) (bool, error) {
	_, ok := groqModels[model]
	return ok, nil
}

// IsModelAvailable verifies the API key is valid and the model is supported.
func (p *GroqProvider) IsModelAvailable(ctx context.Context, model string) (bool, error) {
	// First check if model is supported
	if ok, _ := p.SupportsModel(ctx, model); !ok {
		return false, nil
	}

	// Validate API key on first use
	if err := p.validateAPIKey(ctx); err != nil {
		return false, err
	}

	return true, nil
}

// ModelMetadata returns tier and cost information for a Groq model.
// All Groq models are AgentTierFast (optimized for speed) with low/zero cost.
func (p *GroqProvider) ModelMetadata(ctx context.Context, model string) (ModelMetadata, error) {
	info, ok := groqModels[model]
	if !ok {
		return ModelMetadata{}, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	return ModelMetadata{
		Tier:            AgentTierFast, // Groq specializes in ultra-fast inference
		CostPer1KTokens: 0,             // Groq typically offers free/low-cost inference
		Description:     info.Description,
	}, nil
}

// ResolveModel returns routing information for executing against a Groq model.
func (p *GroqProvider) ResolveModel(ctx context.Context, model string) (*ResolvedModel, error) {
	info, ok := groqModels[model]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	// Validate API key before resolving
	if err := p.validateAPIKey(ctx); err != nil {
		return nil, fmt.Errorf("groq api key validation failed: %w", err)
	}

	return &ResolvedModel{
		Name:            model,
		Provider:        p.info,
		Route:           fmt.Sprintf("%s/chat/completions", p.baseURL),
		Tier:            AgentTierFast,
		CostPer1KTokens: 0,
		Metadata: map[string]string{
			"base_url":       p.baseURL,
			"context_window": fmt.Sprintf("%d", info.ContextWindow),
			"description":    info.Description,
		},
	}, nil
}

// Generate sends a completion request to the Groq API.
func (p *GroqProvider) Generate(ctx context.Context, model, prompt string, opts map[string]interface{}) (string, error) {
	request := groqChatRequest{
		Model: model,
		Messages: []groqChatMessage{
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/chat/completions", p.baseURL), bytes.NewReader(body))
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

	var parsed groqChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("groq response contained no choices")
	}

	return parsed.Choices[0].Message.Content, nil
}

// validateAPIKey performs a lightweight request to verify the API key works.
// Results are cached to avoid repeated validation calls.
func (p *GroqProvider) validateAPIKey(ctx context.Context) error {
	p.mu.RLock()
	if p.validatedCache {
		err := p.validationErr
		p.mu.RUnlock()
		return err
	}
	p.mu.RUnlock()

	// Make a minimal request to validate the key
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/models", p.baseURL), nil)
	if err != nil {
		p.mu.Lock()
		p.validatedCache = true
		p.validationErr = fmt.Errorf("create validation request: %w", err)
		p.mu.Unlock()
		return p.validationErr
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		p.mu.Lock()
		p.validatedCache = true
		p.validationErr = fmt.Errorf("validation request failed: %w", err)
		p.mu.Unlock()
		return p.validationErr
	}
	defer resp.Body.Close()

	// Accept both 200 OK and 401 Unauthorized as valid responses
	// (401 means the endpoint exists but key is invalid)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		p.mu.Lock()
		p.validatedCache = true
		p.validationErr = fmt.Errorf("validation failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
		p.mu.Unlock()
		return p.validationErr
	}

	if resp.StatusCode == http.StatusUnauthorized {
		p.mu.Lock()
		p.validatedCache = true
		p.validationErr = fmt.Errorf("invalid groq api key")
		p.mu.Unlock()
		return p.validationErr
	}

	// Success - cache the positive result
	p.mu.Lock()
	p.validatedCache = true
	p.validationErr = nil
	p.mu.Unlock()

	return nil
}
