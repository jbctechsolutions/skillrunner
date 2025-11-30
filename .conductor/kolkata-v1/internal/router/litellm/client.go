package litellm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/router/types"
)

const (
	// DefaultBaseURL is the default base URL for LiteLLM
	DefaultBaseURL = "http://localhost:18432"
	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
	// ChatCompletionsEndpoint is the chat completions API endpoint
	ChatCompletionsEndpoint = "/v1/chat/completions"
)

// Client is the LiteLLM HTTP client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new LiteLLM client
func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	// Ensure baseURL doesn't have trailing slash for consistent URL construction
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// Complete sends a completion request to LiteLLM and returns the response
func (c *Client) Complete(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("completion request cannot be nil")
	}

	// Construct URL
	url := c.baseURL + ChatCompletionsEndpoint

	// Marshal request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.handleHTTPError(resp.StatusCode, body)
	}

	// Parse response
	var completionResp types.CompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &completionResp, nil
}

// handleHTTPError creates an appropriate error from HTTP status code and body
func (c *Client) handleHTTPError(statusCode int, body []byte) error {
	// Try to parse error response
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
		return fmt.Errorf("HTTP %d: %s", statusCode, errorResp.Error.Message)
	}

	// Fallback to status code and body
	bodyStr := string(body)
	if bodyStr == "" {
		bodyStr = http.StatusText(statusCode)
	}

	return fmt.Errorf("HTTP %d: %s", statusCode, bodyStr)
}
