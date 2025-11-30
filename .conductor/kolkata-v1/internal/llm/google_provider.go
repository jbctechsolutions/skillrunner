package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GoogleProvider implements the Provider interface for Google's Gemini API
type GoogleProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewGoogleProvider creates a new Google provider
func NewGoogleProvider(apiKeyEnv string) (*GoogleProvider, error) {
	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("%s environment variable not set", apiKeyEnv)
	}

	return &GoogleProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

// Name returns the provider name
func (p *GoogleProvider) Name() string {
	return "google"
}

// Chat sends a chat request to Google's Gemini API
func (p *GoogleProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Convert model name to Gemini format (remove "google/" prefix if present)
	modelName := strings.TrimPrefix(req.Model, "google/")
	// Model names in config are already in correct format (e.g., "gemini-1.5-flash")
	// No need to add prefix

	// Convert ChatMessage to Gemini format
	// Gemini uses "user" and "model" roles (not "assistant")
	contents := make([]map[string]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		role := string(msg.Role)
		// Gemini uses "model" instead of "assistant"
		if msg.Role == RoleAssistant {
			role = "model"
		}
		// Gemini doesn't have a system role, prepend to first user message
		if msg.Role == RoleSystem {
			if len(contents) == 0 {
				contents = append(contents, map[string]interface{}{
					"role": "user",
					"parts": []map[string]string{
						{"text": fmt.Sprintf("System: %s", msg.Content)},
					},
				})
			} else {
				// Prepend to first user message
				if firstParts, ok := contents[0]["parts"].([]map[string]string); ok && len(firstParts) > 0 {
					firstParts[0]["text"] = fmt.Sprintf("System: %s\n\n%s", msg.Content, firstParts[0]["text"])
				}
			}
			continue
		}

		contents = append(contents, map[string]interface{}{
			"role": role,
			"parts": []map[string]string{
				{"text": msg.Content},
			},
		})
	}

	// Build Gemini API request
	geminiReq := map[string]interface{}{
		"contents": contents,
	}

	// Add generation config
	generationConfig := map[string]interface{}{
		"maxOutputTokens": req.MaxTokens,
	}
	if req.Temperature > 0 {
		generationConfig["temperature"] = req.Temperature
	}
	geminiReq["generationConfig"] = generationConfig

	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	// Call Gemini API
	// Use v1beta endpoint for gemini-1.5 models
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("google API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ChatResponse{}, fmt.Errorf("google API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason  string `json:"finishReason"`
			Index         int    `json:"index"`
			SafetyRatings []struct {
				Category    string `json:"category"`
				Probability string `json:"probability"`
			} `json:"safetyRatings"`
		} `json:"candidates"`
		PromptFeedback struct {
			SafetyRatings []struct {
				Category    string `json:"category"`
				Probability string `json:"probability"`
			} `json:"safetyRatings"`
		} `json:"promptFeedback"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return ChatResponse{}, fmt.Errorf("decode response: %w", err)
	}

	// Extract content from first candidate
	var content string
	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		for _, part := range geminiResp.Candidates[0].Content.Parts {
			content += part.Text
		}
	}

	return ChatResponse{
		Content: content,
		Usage: Usage{
			InputTokens:  geminiResp.UsageMetadata.PromptTokenCount,
			OutputTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}
