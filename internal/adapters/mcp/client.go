// Package mcp provides an adapter for MCP (Model Context Protocol) servers.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

// Client handles JSON-RPC communication with an MCP server over stdio.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	mu        sync.Mutex
	requestID atomic.Int64
	pending   map[int64]chan *domainMCP.Response

	protocolInfo *domainMCP.ProtocolInfo
	tools        []*domainMCP.Tool
	serverName   string

	readErr   error
	closeOnce sync.Once
	done      chan struct{}
}

// NewClient creates a new MCP client for the given server configuration.
func NewClient(ctx context.Context, config domainMCP.ServerConfig) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, config.Command, config.Args...)

	// Set environment variables
	if len(config.Env) > 0 {
		cmd.Env = append(os.Environ(), mapToEnvSlice(config.Env)...)
	}

	if config.WorkDir != "" {
		cmd.Dir = config.WorkDir
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create stdin pipe: %v", domainMCP.ErrServerStartFailed, err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("%w: failed to create stdout pipe: %v", domainMCP.ErrServerStartFailed, err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("%w: failed to create stderr pipe: %v", domainMCP.ErrServerStartFailed, err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("%w: %v", domainMCP.ErrServerStartFailed, err)
	}

	c := &Client{
		cmd:        cmd,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		pending:    make(map[int64]chan *domainMCP.Response),
		serverName: config.Name,
		done:       make(chan struct{}),
	}

	// Start reading responses in background
	go c.readLoop()

	return c, nil
}

// Initialize performs the MCP handshake with the server.
func (c *Client) Initialize(ctx context.Context) error {
	params := domainMCP.InitializeParams{
		ProtocolVersion: "2024-11-05", // Latest MCP protocol version
		Capabilities:    domainMCP.ClientCapabilities{},
		ClientInfo: domainMCP.ClientInfo{
			Name:    "skillrunner",
			Version: "1.0.0",
		},
	}

	resp, err := c.call(ctx, domainMCP.MethodInitialize, params)
	if err != nil {
		return fmt.Errorf("%w: %v", domainMCP.ErrInitializeFailed, err)
	}

	var result domainMCP.InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("%w: failed to parse initialize result: %v", domainMCP.ErrInitializeFailed, err)
	}

	serverName := ""
	serverVersion := ""
	if result.ServerInfo != nil {
		serverName = result.ServerInfo.Name
		serverVersion = result.ServerInfo.Version
	}

	c.protocolInfo = &domainMCP.ProtocolInfo{
		ProtocolVersion: result.ProtocolVersion,
		ServerName:      serverName,
		ServerVersion:   serverVersion,
		Capabilities:    result.Capabilities,
	}

	return nil
}

// DiscoverTools fetches the list of available tools from the server.
func (c *Client) DiscoverTools(ctx context.Context) ([]*domainMCP.Tool, error) {
	resp, err := c.call(ctx, domainMCP.MethodToolsList, nil)
	if err != nil {
		return nil, err
	}

	var result domainMCP.ToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("%w: failed to parse tools list: %v", domainMCP.ErrInvalidResponse, err)
	}

	tools := make([]*domainMCP.Tool, 0, len(result.Tools))
	for _, def := range result.Tools {
		tool, err := domainMCP.NewTool(def.Name, def.Description, def.InputSchema, c.serverName)
		if err != nil {
			continue // Skip invalid tools
		}
		tools = append(tools, tool)
	}

	c.mu.Lock()
	c.tools = tools
	c.mu.Unlock()

	return tools, nil
}

// CallTool executes a tool and returns the result.
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]any) (*domainMCP.ToolCallResult, error) {
	argsJSON, err := json.Marshal(arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	params := domainMCP.ToolCallParams{
		Name:      toolName,
		Arguments: argsJSON,
	}

	resp, err := c.call(ctx, domainMCP.MethodToolsCall, params)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domainMCP.ErrToolExecutionFailed, err)
	}

	var result domainMCP.ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("%w: failed to parse tool result: %v", domainMCP.ErrInvalidResponse, err)
	}

	return &result, nil
}

// GetTools returns the cached list of tools.
func (c *Client) GetTools() []*domainMCP.Tool {
	c.mu.Lock()
	defer c.mu.Unlock()

	tools := make([]*domainMCP.Tool, len(c.tools))
	copy(tools, c.tools)
	return tools
}

// GetProtocolInfo returns the protocol info from initialization.
func (c *Client) GetProtocolInfo() *domainMCP.ProtocolInfo {
	return c.protocolInfo
}

// PID returns the process ID of the server.
func (c *Client) PID() int {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return 0
}

// Close gracefully shuts down the client and terminates the server process.
func (c *Client) Close(ctx context.Context) error {
	var closeErr error

	c.closeOnce.Do(func() {
		// Try graceful shutdown first
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		_, _ = c.call(shutdownCtx, domainMCP.MethodShutdown, nil)

		// Close streams
		c.stdin.Close()
		c.stdout.Close()
		c.stderr.Close()

		// Signal read loop to stop
		close(c.done)

		// Wait for process with timeout
		done := make(chan error, 1)
		go func() {
			done <- c.cmd.Wait()
		}()

		select {
		case closeErr = <-done:
			// Process exited
		case <-time.After(10 * time.Second):
			// Force kill if still running
			if c.cmd.Process != nil {
				c.cmd.Process.Kill()
			}
			closeErr = <-done
		}
	})

	return closeErr
}

// call sends a JSON-RPC request and waits for the response.
func (c *Client) call(ctx context.Context, method string, params any) (*domainMCP.Response, error) {
	id := c.requestID.Add(1)

	req, err := domainMCP.NewRequest(id, method, params)
	if err != nil {
		return nil, err
	}

	respChan := make(chan *domainMCP.Response, 1)

	c.mu.Lock()
	c.pending[id] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, domainMCP.ErrServerNotRunning
	}
}

// readLoop reads responses from stdout and dispatches them.
func (c *Client) readLoop() {
	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		select {
		case <-c.done:
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp domainMCP.Response
		if err := json.Unmarshal(line, &resp); err != nil {
			continue // Skip malformed responses
		}

		c.mu.Lock()
		if ch, ok := c.pending[resp.ID]; ok {
			ch <- &resp
		}
		c.mu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		c.mu.Lock()
		c.readErr = err
		c.mu.Unlock()
	}
}

// mapToEnvSlice converts a map to KEY=VALUE format.
func mapToEnvSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}
