package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the Ollama API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// ClientOption is a functional option for configuring the Client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the client
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new Ollama API client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ListModels returns all available models
func (c *Client) ListModels(ctx context.Context) (*TagsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+EndpointTags, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var tagsResp TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &tagsResp, nil
}

// Chat performs a chat completion request
func (c *Client) Chat(ctx context.Context, chatReq *ChatRequest) (*ChatResponse, error) {
	chatReq.Stream = false

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+EndpointChat, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &chatResp, nil
}

// StreamCallback is called for each chunk in a streaming response
type StreamCallback func(response *ChatResponse) error

// ChatStream performs a streaming chat completion request
func (c *Client) ChatStream(ctx context.Context, chatReq *ChatRequest, callback StreamCallback) (*ChatResponse, error) {
	chatReq.Stream = true

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+EndpointChat, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var finalResponse *ChatResponse
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk ChatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			return nil, fmt.Errorf("decoding stream chunk: %w", err)
		}

		if callback != nil {
			if err := callback(&chunk); err != nil {
				return nil, fmt.Errorf("callback error: %w", err)
			}
		}

		if chunk.Done {
			finalResponse = &chunk
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stream: %w", err)
	}

	return finalResponse, nil
}

// Generate performs a text generation request
func (c *Client) Generate(ctx context.Context, genReq *GenerateRequest) (*GenerateResponse, error) {
	genReq.Stream = false

	body, err := json.Marshal(genReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+EndpointGenerate, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &genResp, nil
}

// GenerateStreamCallback is called for each chunk in a streaming generate response
type GenerateStreamCallback func(response *GenerateResponse) error

// GenerateStream performs a streaming text generation request
func (c *Client) GenerateStream(ctx context.Context, genReq *GenerateRequest, callback GenerateStreamCallback) (*GenerateResponse, error) {
	genReq.Stream = true

	body, err := json.Marshal(genReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+EndpointGenerate, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var finalResponse *GenerateResponse
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk GenerateResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			return nil, fmt.Errorf("decoding stream chunk: %w", err)
		}

		if callback != nil {
			if err := callback(&chunk); err != nil {
				return nil, fmt.Errorf("callback error: %w", err)
			}
		}

		if chunk.Done {
			finalResponse = &chunk
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stream: %w", err)
	}

	return finalResponse, nil
}

// Ping checks if the Ollama server is available
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+EndpointTags, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// parseError extracts error information from a failed response
func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("status %d: failed to read error body", resp.StatusCode)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("ollama error: %s", errResp.Error)
}
