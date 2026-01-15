package ports

import (
	"context"

	"github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

// MCPServerPort defines the interface for MCP server management.
type MCPServerPort interface {
	// Server lifecycle
	Start(ctx context.Context, config mcp.ServerConfig) error
	Stop(ctx context.Context, serverName string) error
	StopAll(ctx context.Context) error

	// Server status
	GetInfo(serverName string) (*mcp.ServerInfo, error)
	ListServers() []mcp.ServerInfo
	IsRunning(serverName string) bool

	// Tool discovery
	ListTools(ctx context.Context, serverName string) ([]*mcp.Tool, error)
	GetTool(ctx context.Context, serverName, toolName string) (*mcp.Tool, error)

	// Tool execution
	CallTool(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.ToolCallResult, error)
}

// MCPConfigPort defines the interface for loading MCP configuration.
type MCPConfigPort interface {
	// Load reads MCP server configurations from the standard location.
	// Returns an empty map if no config file exists.
	Load(ctx context.Context) (map[string]mcp.ServerConfig, error)

	// LoadFromPath reads MCP configuration from a specific path.
	LoadFromPath(ctx context.Context, path string) (map[string]mcp.ServerConfig, error)
}

// MCPToolRegistryPort provides access to MCP tools for skill execution.
type MCPToolRegistryPort interface {
	// GetAllTools returns all tools from all running servers.
	GetAllTools(ctx context.Context) ([]*mcp.Tool, error)

	// CallToolByFullName executes a tool using its full name (mcp__server__tool).
	CallToolByFullName(ctx context.Context, fullName string, arguments map[string]any) (*mcp.ToolCallResult, error)

	// EnsureServerRunning starts a server if not already running.
	EnsureServerRunning(ctx context.Context, serverName string) error
}
