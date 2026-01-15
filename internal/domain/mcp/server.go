package mcp

import (
	"fmt"
	"strings"
	"time"
)

// ServerState represents the lifecycle state of an MCP server.
type ServerState string

const (
	// ServerStateStopped indicates the server is not running.
	ServerStateStopped ServerState = "stopped"

	// ServerStateStarting indicates the server is being started.
	ServerStateStarting ServerState = "starting"

	// ServerStateInitializing indicates the server is performing initialization handshake.
	ServerStateInitializing ServerState = "initializing"

	// ServerStateReady indicates the server is ready to accept requests.
	ServerStateReady ServerState = "ready"

	// ServerStateStopping indicates the server is being stopped.
	ServerStateStopping ServerState = "stopping"

	// ServerStateError indicates the server encountered an error.
	ServerStateError ServerState = "error"
)

// String returns the string representation of the state.
func (s ServerState) String() string {
	return string(s)
}

// IsRunning returns true if the server is in a running state.
func (s ServerState) IsRunning() bool {
	return s == ServerStateReady
}

// IsTerminal returns true if the server is in a terminal state.
func (s ServerState) IsTerminal() bool {
	return s == ServerStateStopped || s == ServerStateError
}

// ServerConfig holds the configuration for an MCP server.
type ServerConfig struct {
	Name    string            // Server identifier (e.g., "linear")
	Command string            // Command to execute (e.g., "npx")
	Args    []string          // Command arguments
	Env     map[string]string // Environment variables
	WorkDir string            // Working directory (optional)
}

// Validate checks if the ServerConfig is valid.
func (c *ServerConfig) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(c.Command) == "" {
		return fmt.Errorf("%w: command is required", ErrInvalidConfig)
	}
	return nil
}

// ServerInfo contains runtime information about a server.
type ServerInfo struct {
	Name         string
	State        ServerState
	PID          int
	StartedAt    time.Time
	ToolCount    int
	LastActivity time.Time
	ErrorMessage string
}

// ProtocolInfo contains MCP protocol negotiation information.
type ProtocolInfo struct {
	ProtocolVersion string
	ServerName      string
	ServerVersion   string
	Capabilities    ServerCapabilities
}

// ServerCapabilities describes what the MCP server supports.
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability describes tool-related capabilities.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Server can notify when tool list changes
}

// ResourcesCapability describes resource-related capabilities.
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability describes prompt-related capabilities.
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}
