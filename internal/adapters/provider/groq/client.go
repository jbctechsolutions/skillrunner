package groq

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// Client handles HTTP communication with the Groq API.
type Client struct {
	httpClient *http.Client
	config     Config
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.config.Timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.config.MaxRetries = maxRetries
	}
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.config.BaseURL = baseURL
	}
}

// NewClient creates a new Groq API client with the provided API key and options.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	config := DefaultConfig(apiKey)

	client := &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Chat sends a chat completion request to the Groq API.
func (c *Client) Chat(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.NewError(errors.CodeProvider, "failed to marshal request", err)
	}

	resp, err := c.doRequestWithRetry(ctx, http.MethodPost, EndpointChatCompletions, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.NewError(errors.CodeProvider, "failed to decode response", err)
	}

	return &result, nil
}

// ChatStream sends a streaming chat completion request to the Groq API.
func (c *Client) ChatStream(ctx context.Context, req *ChatCompletionRequest, callback func(chunk *ChatCompletionChunk) error) error {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return errors.NewError(errors.CodeProvider, "failed to marshal request", err)
	}

	// For streaming, we don't retry as it's a long-running operation
	httpReq, err := c.newRequest(ctx, http.MethodPost, EndpointChatCompletions, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return errors.NewError(errors.CodeProvider, "request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	return c.parseSSEStream(resp.Body, callback)
}

// parseSSEStream parses the Server-Sent Events stream from Groq.
// Groq uses OpenAI-compatible SSE format with 'data: ' prefix and [DONE] sentinel.
func (c *Client) parseSSEStream(reader io.Reader, callback func(chunk *ChatCompletionChunk) error) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse data lines
		data, found := strings.CutPrefix(line, "data: ")
		if !found {
			continue
		}

		// Check for [DONE] sentinel indicating end of stream
		if data == "[DONE]" {
			return nil
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return errors.NewError(errors.CodeProvider, "failed to parse SSE chunk", err)
		}

		if err := callback(&chunk); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.NewError(errors.CodeProvider, "error reading SSE stream", err)
	}

	return nil
}

// ListModels retrieves the list of available models from the Groq API.
func (c *Client) ListModels(ctx context.Context) (*ModelsResponse, error) {
	resp, err := c.doRequestWithRetry(ctx, http.MethodGet, EndpointModels, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var result ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.NewError(errors.CodeProvider, "failed to decode models response", err)
	}

	return &result, nil
}

// doRequestWithRetry performs an HTTP request with exponential backoff retry.
func (c *Client) doRequestWithRetry(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var lastErr error
	baseDelay := 500 * time.Millisecond

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 500ms, 1s, 2s, 4s...
			delay := baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := c.newRequest(ctx, method, path, body)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = errors.NewError(errors.CodeProvider, "request failed", err)
			continue
		}

		// Retry on rate limit (429) or server errors (5xx)
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, errors.NewError(errors.CodeProvider,
		fmt.Sprintf("request failed after %d retries", c.config.MaxRetries+1), lastErr)
}

// newRequest creates a new HTTP request with required headers.
func (c *Client) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	url := c.config.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, errors.NewError(errors.CodeProvider, "failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	return req, nil
}

// handleErrorResponse extracts error information from an error response.
func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.NewError(errors.CodeProvider,
			fmt.Sprintf("HTTP %d: failed to read error response", resp.StatusCode), err)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// If we can't parse the error, return the raw body
		return errors.NewError(errors.CodeProvider,
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil)
	}

	errCode := errors.CodeProvider
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		errCode = errors.CodeConfiguration
	case http.StatusNotFound:
		errCode = errors.CodeNotFound
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		errCode = errors.CodeValidation
	}

	return errors.NewError(errCode,
		fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message), nil)
}

// HealthCheck performs a lightweight check to verify API connectivity.
func (c *Client) HealthCheck(ctx context.Context) error {
	// Use the models endpoint for health check since it's lightweight
	_, err := c.ListModels(ctx)
	return err
}
