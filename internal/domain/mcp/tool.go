package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Tool represents an MCP tool discovered from a server.
// It is a value object - immutable after creation.
type Tool struct {
	name        string
	description string
	inputSchema json.RawMessage
	serverName  string
}

// NewTool creates a new Tool with validation.
func NewTool(name, description string, inputSchema json.RawMessage, serverName string) (*Tool, error) {
	name = strings.TrimSpace(name)
	serverName = strings.TrimSpace(serverName)

	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidToolName)
	}
	if serverName == "" {
		return nil, fmt.Errorf("%w: server name is required", ErrInvalidToolName)
	}

	return &Tool{
		name:        name,
		description: description,
		inputSchema: inputSchema,
		serverName:  serverName,
	}, nil
}

// Name returns the tool's name.
func (t *Tool) Name() string { return t.name }

// Description returns the tool's description.
func (t *Tool) Description() string { return t.description }

// InputSchema returns the JSON Schema for input validation.
func (t *Tool) InputSchema() json.RawMessage { return t.inputSchema }

// ServerName returns the owning server's name.
func (t *Tool) ServerName() string { return t.serverName }

// FullName returns the fully qualified name: mcp__{server}__{tool}
func (t *Tool) FullName() string {
	return fmt.Sprintf("mcp__%s__%s", t.serverName, t.name)
}

// ParseToolName extracts server and tool name from "mcp__server__tool" format.
// Returns an error if the format is invalid.
func ParseToolName(fullName string) (serverName, toolName string, err error) {
	if !strings.HasPrefix(fullName, "mcp__") {
		return "", "", fmt.Errorf("%w: must start with 'mcp__'", ErrInvalidToolName)
	}

	parts := strings.SplitN(fullName, "__", 3)
	if len(parts) != 3 {
		return "", "", fmt.Errorf("%w: expected format 'mcp__server__tool'", ErrInvalidToolName)
	}

	serverName = parts[1]
	toolName = parts[2]

	if serverName == "" {
		return "", "", fmt.Errorf("%w: server name cannot be empty", ErrInvalidToolName)
	}
	if toolName == "" {
		return "", "", fmt.Errorf("%w: tool name cannot be empty", ErrInvalidToolName)
	}

	return serverName, toolName, nil
}

// ToolDefinition is the JSON representation of a tool from MCP servers.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToTool converts a ToolDefinition to a Tool domain object.
func (d *ToolDefinition) ToTool(serverName string) (*Tool, error) {
	return NewTool(d.Name, d.Description, d.InputSchema, serverName)
}
