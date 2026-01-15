package anthropic

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

// Client handles HTTP communication with the Anthropic API.
type Client struct {
	httpClient *http.Client
	config     Config
}

// NewClient creates a new Anthropic API client.
func NewClient(config Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}
}

// SendMessage sends a message request to the Anthropic API.
func (c *Client) SendMessage(ctx context.Context, req *MessagesRequest) (*MessagesResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.NewError(errors.CodeProvider, "failed to marshal request", err)
	}

	// Check if we need the Tool Search Tool beta header
	betaHeader := ""
	if hasDeferredLoadingTools(req.Tools) {
		betaHeader = BetaToolSearch
	}

	resp, err := c.doRequestWithRetryAndBeta(ctx, http.MethodPost, "/messages", body, betaHeader)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var result MessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.NewError(errors.CodeProvider, "failed to decode response", err)
	}

	return &result, nil
}

// hasDeferredLoadingTools checks if any tool has deferred loading enabled.
func hasDeferredLoadingTools(tools []Tool) bool {
	for _, tool := range tools {
		if tool.DeferLoading {
			return true
		}
	}
	return false
}

// StreamMessage sends a streaming message request to the Anthropic API.
func (c *Client) StreamMessage(ctx context.Context, req *MessagesRequest, callback func(event *StreamEvent) error) error {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return errors.NewError(errors.CodeProvider, "failed to marshal request", err)
	}

	// For streaming, we don't retry as it's a long-running operation
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/messages", body)
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

// parseSSEStream parses the Server-Sent Events stream from Anthropic.
func (c *Client) parseSSEStream(reader io.Reader, callback func(event *StreamEvent) error) error {
	scanner := bufio.NewScanner(reader)
	var eventType string
	var dataBuffer strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line indicates end of event
			if eventType != "" && dataBuffer.Len() > 0 {
				var event StreamEvent
				if err := json.Unmarshal([]byte(dataBuffer.String()), &event); err != nil {
					return errors.NewError(errors.CodeProvider, "failed to parse SSE event", err)
				}

				if err := callback(&event); err != nil {
					return err
				}
			}
			eventType = ""
			dataBuffer.Reset()
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			dataBuffer.WriteString(data)
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.NewError(errors.CodeProvider, "error reading SSE stream", err)
	}

	return nil
}

// doRequestWithRetry performs an HTTP request with exponential backoff retry.
func (c *Client) doRequestWithRetry(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	return c.doRequestWithRetryAndBeta(ctx, method, path, body, "")
}

// doRequestWithRetryAndBeta performs an HTTP request with exponential backoff retry and optional beta header.
func (c *Client) doRequestWithRetryAndBeta(ctx context.Context, method, path string, body []byte, betaHeader string) (*http.Response, error) {
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

		req, err := c.newRequestWithBeta(ctx, method, path, body, betaHeader)
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
	return c.newRequestWithBeta(ctx, method, path, body, "")
}

// newRequestWithBeta creates a new HTTP request with required headers and optional beta header.
func (c *Client) newRequestWithBeta(ctx context.Context, method, path string, body []byte, betaHeader string) (*http.Request, error) {
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
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", c.config.Version)

	// Add beta header if provided (for Tool Search Tool feature)
	if betaHeader != "" {
		req.Header.Set("anthropic-beta", betaHeader)
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

	return errors.NewError(errCode,
		fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message), nil)
}

// HealthCheck performs a lightweight check to verify API connectivity.
func (c *Client) HealthCheck(ctx context.Context) error {
	// Send a minimal request to verify connectivity
	req := &MessagesRequest{
		Model:     ModelClaude35Haiku, // Use the cheapest model for health checks
		MaxTokens: 1,
		Messages: []Message{
			{
				Role: RoleUser,
				Content: MessageContent{
					{Type: "text", Text: "Hi"},
				},
			},
		},
	}

	_, err := c.SendMessage(ctx, req)
	return err
}
