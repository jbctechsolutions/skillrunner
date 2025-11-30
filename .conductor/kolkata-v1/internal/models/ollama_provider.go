package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"errors"
)

// OllamaProvider integrates local Ollama models into the orchestrator.
type OllamaProvider struct {
	baseURL string
	client  *http.Client

	info ProviderInfo

	mu          sync.RWMutex
	modelsCache map[string]ollamaModel
	lastFetch   time.Time
	cacheTTL    time.Duration
}

type ollamaModel struct {
	Name        string
	Description string
	Digest      string
	ModifiedAt  time.Time
	Size        int64
	Details     map[string]interface{}
}

type ollamaTagsResponse struct {
	Models []struct {
		Name       string                 `json:"name"`
		Model      string                 `json:"model"`
		Digest     string                 `json:"digest"`
		ModifiedAt string                 `json:"modified_at"`
		Size       int64                  `json:"size"`
		Details    map[string]interface{} `json:"details"`
	} `json:"models"`
}

type ollamaGenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Model      string                 `json:"model"`
	CreatedAt  string                 `json:"created_at"`
	Response   string                 `json:"response"`
	Done       bool                   `json:"done"`
	DoneReason string                 `json:"done_reason"`
	Metrics    map[string]interface{} `json:"metrics"`
}

// NewOllamaProvider creates a provider backed by an Ollama HTTP API.
func NewOllamaProvider(baseURL string, client *http.Client) (*OllamaProvider, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is required for Ollama provider")
	}
	if client == nil {
		client = &http.Client{
			Timeout: 15 * time.Minute, // Large models (32B+) + complex prompts can take time
		}
	}

	baseURL = strings.TrimRight(baseURL, "/")

	return &OllamaProvider{
		baseURL:     baseURL,
		client:      client,
		info:        ProviderInfo{Name: "ollama", Type: ProviderTypeLocal},
		modelsCache: make(map[string]ollamaModel),
		cacheTTL:    30 * time.Second,
	}, nil
}

// Info returns provider metadata.
func (p *OllamaProvider) Info() ProviderInfo {
	return p.info
}

// Models lists all models available from the Ollama server.
func (p *OllamaProvider) Models(ctx context.Context) ([]ModelRef, error) {
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
			Name:        model.Name,
			Description: desc,
		})
	}
	return refs, nil
}

// normalizeModelName strips the "ollama/" prefix if present and lowercases
func normalizeModelName(model string) string {
	model = strings.ToLower(model)
	if strings.HasPrefix(model, "ollama/") {
		return model[7:] // Remove "ollama/" prefix
	}
	return model
}

// SupportsModel reports whether the model exists on the Ollama server.
func (p *OllamaProvider) SupportsModel(ctx context.Context, model string) (bool, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return false, err
	}
	_, ok := models[normalizeModelName(model)]
	return ok, nil
}

// IsModelAvailable returns true when the model is present and the server is reachable.
func (p *OllamaProvider) IsModelAvailable(ctx context.Context, model string) (bool, error) {
	return p.SupportsModel(ctx, model)
}

// CheckModelHealth verifies a specific model is available and provides actionable feedback.
func (p *OllamaProvider) CheckModelHealth(ctx context.Context, modelID string) (*HealthStatus, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return &HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Unable to connect to Ollama at %s", p.baseURL),
			Suggestions: []string{
				"Verify Ollama is running: ollama list",
				"Check Ollama URL configuration",
				fmt.Sprintf("Test connection: curl %s/api/tags", p.baseURL),
			},
			Details: map[string]interface{}{
				"ollama_url": p.baseURL,
				"error":      err.Error(),
			},
		}, nil
	}

	normalizedModelID := normalizeModelName(modelID)
	modelData, found := models[normalizedModelID]

	if !found {
		// Model not found - provide helpful suggestions
		availableModels := make([]string, 0, len(models))
		for _, m := range models {
			sizeGB := float64(m.Size) / (1024 * 1024 * 1024)
			availableModels = append(availableModels, fmt.Sprintf("%s (%.1fGB)", m.Name, sizeGB))
		}

		return &HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Model '%s' not found in Ollama", modelID),
			Suggestions: []string{
				fmt.Sprintf("Pull the model: ollama pull %s", modelID),
				"Check model name spelling",
				"List available models: ollama list",
			},
			Details: map[string]interface{}{
				"ollama_url":       p.baseURL,
				"available_models": availableModels,
				"requested_model":  modelID,
				"total_models":     len(models),
			},
		}, nil
	}

	// Model found and healthy
	sizeGB := float64(modelData.Size) / (1024 * 1024 * 1024)

	// Extract context window from details if available
	contextWindow := "unknown"
	if params, ok := modelData.Details["parameter_size"].(string); ok {
		contextWindow = params
	}

	return &HealthStatus{
		Healthy:     true,
		Message:     fmt.Sprintf("Model '%s' is available", modelData.Name),
		Suggestions: nil,
		Details: map[string]interface{}{
			"model_name":     modelData.Name,
			"size_bytes":     modelData.Size,
			"size_gb":        fmt.Sprintf("%.1f", sizeGB),
			"digest":         modelData.Digest,
			"modified_at":    modelData.ModifiedAt.Format("2006-01-02 15:04:05"),
			"context_window": contextWindow,
			"ollama_url":     p.baseURL,
		},
	}, nil
}

// ModelMetadata returns tier and cost information for a model.
func (p *OllamaProvider) ModelMetadata(ctx context.Context, model string) (ModelMetadata, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return ModelMetadata{}, err
	}
	data, ok := models[normalizeModelName(model)]
	if !ok {
		return ModelMetadata{}, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	return ModelMetadata{
		Tier:            AgentTierFast,
		CostPer1KTokens: 0,
		Description:     data.Description,
	}, nil
}

// ResolveModel ensures the model exists and returns route metadata.
func (p *OllamaProvider) ResolveModel(ctx context.Context, model string) (*ResolvedModel, error) {
	models, err := p.fetchModels(ctx, false)
	if err != nil {
		return nil, err
	}
	data, ok := models[normalizeModelName(model)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrModelNotFound, model)
	}

	return &ResolvedModel{
		Name:            data.Name,
		Provider:        p.info,
		Route:           fmt.Sprintf("%s/api/generate", p.baseURL),
		Tier:            AgentTierFast,
		CostPer1KTokens: 0,
		Metadata: map[string]string{
			"base_url":     p.baseURL,
			"model_digest": data.Digest,
		},
	}, nil
}

// Generate issues a completion request to the Ollama server. When stream is true,
// the returned channel yields partial responses terminated by a closed channel.
func (p *OllamaProvider) Generate(ctx context.Context, model, prompt string, stream bool, opts map[string]interface{}) (<-chan string, error) {
	reqBody, err := json.Marshal(ollamaGenerateRequest{
		Model:   model,
		Prompt:  prompt,
		Stream:  stream,
		Options: opts,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal generate request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/generate", p.baseURL), bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	output := make(chan string)

	go func() {
		defer resp.Body.Close()
		defer close(output)

		if !stream {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				select {
				case <-ctx.Done():
				case output <- fmt.Sprintf("error: %v", err):
				}
				return
			}
			var parsed ollamaGenerateResponse
			if err := json.Unmarshal(body, &parsed); err != nil {
				select {
				case <-ctx.Done():
				case output <- fmt.Sprintf("error: %v", err):
				}
				return
			}
			select {
			case <-ctx.Done():
				return
			case output <- parsed.Response:
			}
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				var event ollamaGenerateResponse
				if err := json.Unmarshal(line, &event); err == nil {
					select {
					case <-ctx.Done():
						return
					case output <- event.Response:
					}
					if event.Done {
						return
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

func (p *OllamaProvider) fetchModels(ctx context.Context, force bool) (map[string]ollamaModel, error) {
	p.mu.RLock()
	if !force && time.Since(p.lastFetch) < p.cacheTTL && len(p.modelsCache) > 0 {
		defer p.mu.RUnlock()
		return cloneOllamaModelMap(p.modelsCache), nil
	}
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/tags", p.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create tags request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("fetch tags: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode tags response: %w", err)
	}

	models := make(map[string]ollamaModel, len(parsed.Models))
	for _, mdl := range parsed.Models {
		modified, _ := time.Parse(time.RFC3339, mdl.ModifiedAt)
		description := mdl.Model
		if detailsDesc, ok := mdl.Details["description"].(string); ok && detailsDesc != "" {
			description = detailsDesc
		}
		if description == "" {
			description = mdl.Name
		}
		models[strings.ToLower(mdl.Name)] = ollamaModel{
			Name:        mdl.Name,
			Description: description,
			Digest:      mdl.Digest,
			ModifiedAt:  modified,
			Size:        mdl.Size,
			Details:     mdl.Details,
		}
	}

	p.mu.Lock()
	p.modelsCache = models
	p.lastFetch = time.Now()
	p.mu.Unlock()

	return cloneOllamaModelMap(models), nil
}

func cloneOllamaModelMap(input map[string]ollamaModel) map[string]ollamaModel {
	out := make(map[string]ollamaModel, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
