package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// Client handles HTTP communication with the OpenAI API.
type Client struct {
	httpClient *http.Client
	config     Config
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client for the Client.
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

// WithBaseURL sets the base URL for API requests.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.config.BaseURL = strings.TrimSuffix(baseURL, "/")
	}
}

// WithOrganization sets the organization header for API requests.
func WithOrganization(org string) ClientOption {
	return func(c *Client) {
		c.config.Organization = org
	}
}

// NewClient creates a new OpenAI API client with functional options.
func NewClient(config Config, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Chat sends a chat completion request to the OpenAI API.
func (c *Client) Chat(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, *RateLimitInfo, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, nil, errors.NewError(errors.CodeProvider, "failed to marshal request", err)
	}

	resp, err := c.doRequestWithRetry(ctx, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	rateLimitInfo := c.parseRateLimitHeaders(resp.Header)

	if resp.StatusCode != http.StatusOK {
		return nil, rateLimitInfo, c.handleErrorResponse(resp)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, rateLimitInfo, errors.NewError(errors.CodeProvider, "failed to decode response", err)
	}

	return &result, rateLimitInfo, nil
}

// ChatStream sends a streaming chat completion request to the OpenAI API.
func (c *Client) ChatStream(ctx context.Context, req *ChatCompletionRequest, callback func(chunk *StreamChunk) error) (*RateLimitInfo, error) {
	req.Stream = true
	if req.StreamOptions == nil {
		req.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.NewError(errors.CodeProvider, "failed to marshal request", err)
	}

	// For streaming, we don't retry as it's a long-running operation
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.NewError(errors.CodeProvider, "request failed", err)
	}
	defer resp.Body.Close()

	rateLimitInfo := c.parseRateLimitHeaders(resp.Header)

	if resp.StatusCode != http.StatusOK {
		return rateLimitInfo, c.handleErrorResponse(resp)
	}

	return rateLimitInfo, c.parseSSEStream(resp.Body, callback)
}

// parseSSEStream parses the Server-Sent Events stream from OpenAI.
func (c *Client) parseSSEStream(reader io.Reader, callback func(chunk *StreamChunk) error) error {
	scanner := bufio.NewScanner(reader)
	// Set a larger buffer for potentially large SSE messages
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for data prefix
		data, found := strings.CutPrefix(line, "data: ")
		if !found {
			continue
		}

		// Handle [DONE] sentinel
		if data == "[DONE]" {
			return nil
		}

		var chunk StreamChunk
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

// ListModels retrieves the list of available models from the OpenAI API.
func (c *Client) ListModels(ctx context.Context) (*ModelsResponse, error) {
	resp, err := c.doRequestWithRetry(ctx, http.MethodGet, "/models", nil)
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
	delay := c.config.RetryBaseDelay
	if delay == 0 {
		delay = 500 * time.Millisecond
	}

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			// Exponential backoff with cap
			delay *= 2
			if c.config.RetryMaxDelay > 0 && delay > c.config.RetryMaxDelay {
				delay = c.config.RetryMaxDelay
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

		// Check for retryable status codes (429 Too Many Requests, 5xx Server Errors)
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			// Check for Retry-After header
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
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

	if c.config.Organization != "" {
		req.Header.Set("OpenAI-Organization", c.config.Organization)
	}

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

	errType := errResp.Error.Type
	if errType == "" {
		errType = "error"
	}

	return errors.NewError(errCode,
		fmt.Sprintf("%s: %s", errType, errResp.Error.Message), nil)
}

// parseRateLimitHeaders extracts rate limit information from response headers.
func (c *Client) parseRateLimitHeaders(headers http.Header) *RateLimitInfo {
	info := &RateLimitInfo{}

	if v := headers.Get("x-ratelimit-limit-requests"); v != "" {
		info.LimitRequests, _ = strconv.Atoi(v)
	}
	if v := headers.Get("x-ratelimit-limit-tokens"); v != "" {
		info.LimitTokens, _ = strconv.Atoi(v)
	}
	if v := headers.Get("x-ratelimit-remaining-requests"); v != "" {
		info.RemainingRequests, _ = strconv.Atoi(v)
	}
	if v := headers.Get("x-ratelimit-remaining-tokens"); v != "" {
		info.RemainingTokens, _ = strconv.Atoi(v)
	}
	if v := headers.Get("x-ratelimit-reset-requests"); v != "" {
		info.ResetRequests = parseDuration(v)
	}
	if v := headers.Get("x-ratelimit-reset-tokens"); v != "" {
		info.ResetTokens = parseDuration(v)
	}

	return info
}

// parseDuration parses OpenAI's duration format (e.g., "1s", "100ms", "6m0s")
// and returns the time when the rate limit resets.
func parseDuration(s string) time.Time {
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}
	}
	return time.Now().Add(d)
}

// HealthCheck performs a lightweight check to verify API connectivity.
func (c *Client) HealthCheck(ctx context.Context) error {
	// Use ListModels as a lightweight health check since it doesn't consume tokens
	_, err := c.ListModels(ctx)
	return err
}
