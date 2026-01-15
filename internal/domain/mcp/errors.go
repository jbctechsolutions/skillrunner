// Package mcp provides domain types for Model Context Protocol server support.
package mcp

import "errors"

// Server lifecycle errors.
var (
	// ErrServerNotFound indicates the requested MCP server does not exist.
	ErrServerNotFound = errors.New("mcp server not found")

	// ErrServerNotRunning indicates the server is not in a running state.
	ErrServerNotRunning = errors.New("mcp server not running")

	// ErrServerAlreadyRunning indicates an attempt to start an already running server.
	ErrServerAlreadyRunning = errors.New("mcp server already running")

	// ErrServerStartFailed indicates the server failed to start.
	ErrServerStartFailed = errors.New("failed to start mcp server")

	// ErrServerTimeout indicates a timeout while waiting for the server.
	ErrServerTimeout = errors.New("mcp server timeout")
)

// Tool errors.
var (
	// ErrToolNotFound indicates the requested tool does not exist.
	ErrToolNotFound = errors.New("tool not found")

	// ErrInvalidToolName indicates the tool name format is invalid.
	ErrInvalidToolName = errors.New("invalid tool name format")

	// ErrToolExecutionFailed indicates the tool execution failed.
	ErrToolExecutionFailed = errors.New("tool execution failed")
)

// Protocol errors.
var (
	// ErrInitializeFailed indicates the MCP initialization handshake failed.
	ErrInitializeFailed = errors.New("mcp initialization failed")

	// ErrInvalidResponse indicates the server returned an invalid response.
	ErrInvalidResponse = errors.New("invalid mcp response")

	// ErrProtocolError indicates a general protocol error.
	ErrProtocolError = errors.New("mcp protocol error")
)

// Configuration errors.
var (
	// ErrConfigNotFound indicates the MCP configuration file was not found.
	ErrConfigNotFound = errors.New("mcp config not found")

	// ErrInvalidConfig indicates the MCP configuration is invalid.
	ErrInvalidConfig = errors.New("invalid mcp config")
)

// Is reports whether any error in err's chain matches target.
// This is a convenience wrapper around errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}
