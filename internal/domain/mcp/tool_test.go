package mcp

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNewTool(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		description string
		inputSchema json.RawMessage
		serverName  string
		wantErr     bool
		errTarget   error
	}{
		{
			name:        "valid tool",
			toolName:    "create_issue",
			description: "Creates an issue",
			inputSchema: json.RawMessage(`{"type": "object"}`),
			serverName:  "linear",
			wantErr:     false,
		},
		{
			name:        "valid tool without description",
			toolName:    "list_issues",
			description: "",
			inputSchema: nil,
			serverName:  "linear",
			wantErr:     false,
		},
		{
			name:        "valid tool with whitespace trimmed",
			toolName:    "  create_issue  ",
			description: "Creates an issue",
			inputSchema: nil,
			serverName:  "  linear  ",
			wantErr:     false,
		},
		{
			name:        "empty name",
			toolName:    "",
			description: "desc",
			inputSchema: nil,
			serverName:  "linear",
			wantErr:     true,
			errTarget:   ErrInvalidToolName,
		},
		{
			name:        "whitespace only name",
			toolName:    "   ",
			description: "desc",
			inputSchema: nil,
			serverName:  "linear",
			wantErr:     true,
			errTarget:   ErrInvalidToolName,
		},
		{
			name:        "empty server name",
			toolName:    "create_issue",
			description: "desc",
			inputSchema: nil,
			serverName:  "",
			wantErr:     true,
			errTarget:   ErrInvalidToolName,
		},
		{
			name:        "whitespace only server name",
			toolName:    "create_issue",
			description: "desc",
			inputSchema: nil,
			serverName:  "   ",
			wantErr:     true,
			errTarget:   ErrInvalidToolName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := NewTool(tt.toolName, tt.description, tt.inputSchema, tt.serverName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errTarget != nil && !errors.Is(err, tt.errTarget) {
					t.Errorf("error = %v, want %v", err, tt.errTarget)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tool == nil {
				t.Error("expected tool, got nil")
				return
			}
		})
	}
}

func TestTool_Getters(t *testing.T) {
	schema := json.RawMessage(`{"type": "object", "properties": {"title": {"type": "string"}}}`)
	tool, err := NewTool("create_issue", "Creates an issue", schema, "linear")
	if err != nil {
		t.Fatalf("failed to create tool: %v", err)
	}

	if got := tool.Name(); got != "create_issue" {
		t.Errorf("Name() = %q, want %q", got, "create_issue")
	}

	if got := tool.Description(); got != "Creates an issue" {
		t.Errorf("Description() = %q, want %q", got, "Creates an issue")
	}

	if got := tool.ServerName(); got != "linear" {
		t.Errorf("ServerName() = %q, want %q", got, "linear")
	}

	if got := string(tool.InputSchema()); got != string(schema) {
		t.Errorf("InputSchema() = %q, want %q", got, string(schema))
	}
}

func TestTool_FullName(t *testing.T) {
	tests := []struct {
		toolName   string
		serverName string
		want       string
	}{
		{"create_issue", "linear", "mcp__linear__create_issue"},
		{"list_repos", "github", "mcp__github__list_repos"},
		{"search", "brave", "mcp__brave__search"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			tool, err := NewTool(tt.toolName, "", nil, tt.serverName)
			if err != nil {
				t.Fatalf("failed to create tool: %v", err)
			}

			if got := tool.FullName(); got != tt.want {
				t.Errorf("FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseToolName(t *testing.T) {
	tests := []struct {
		fullName   string
		wantServer string
		wantTool   string
		wantErr    bool
		errTarget  error
	}{
		{
			fullName:   "mcp__linear__create_issue",
			wantServer: "linear",
			wantTool:   "create_issue",
			wantErr:    false,
		},
		{
			fullName:   "mcp__github__list_repos",
			wantServer: "github",
			wantTool:   "list_repos",
			wantErr:    false,
		},
		{
			fullName:   "mcp__my_server__my_tool",
			wantServer: "my_server",
			wantTool:   "my_tool",
			wantErr:    false,
		},
		{
			fullName:   "mcp__server__tool__with__extra__parts",
			wantServer: "server",
			wantTool:   "tool__with__extra__parts",
			wantErr:    false,
		},
		{
			fullName:  "invalid",
			wantErr:   true,
			errTarget: ErrInvalidToolName,
		},
		{
			fullName:  "mcp_missing_underscores",
			wantErr:   true,
			errTarget: ErrInvalidToolName,
		},
		{
			fullName:  "mcp__",
			wantErr:   true,
			errTarget: ErrInvalidToolName,
		},
		{
			fullName:  "mcp____tool",
			wantErr:   true,
			errTarget: ErrInvalidToolName,
		},
		{
			fullName:  "mcp__server__",
			wantErr:   true,
			errTarget: ErrInvalidToolName,
		},
		{
			fullName:  "other__server__tool",
			wantErr:   true,
			errTarget: ErrInvalidToolName,
		},
		{
			fullName:  "",
			wantErr:   true,
			errTarget: ErrInvalidToolName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.fullName, func(t *testing.T) {
			server, tool, err := ParseToolName(tt.fullName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errTarget != nil && !errors.Is(err, tt.errTarget) {
					t.Errorf("error = %v, want %v", err, tt.errTarget)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if server != tt.wantServer {
				t.Errorf("server = %q, want %q", server, tt.wantServer)
			}
			if tool != tt.wantTool {
				t.Errorf("tool = %q, want %q", tool, tt.wantTool)
			}
		})
	}
}

func TestParseToolName_RoundTrip(t *testing.T) {
	// Create a tool and verify parsing its full name returns original values
	tool, err := NewTool("create_issue", "Creates an issue", nil, "linear")
	if err != nil {
		t.Fatalf("failed to create tool: %v", err)
	}

	fullName := tool.FullName()
	server, toolName, err := ParseToolName(fullName)
	if err != nil {
		t.Fatalf("failed to parse full name: %v", err)
	}

	if server != tool.ServerName() {
		t.Errorf("server = %q, want %q", server, tool.ServerName())
	}
	if toolName != tool.Name() {
		t.Errorf("toolName = %q, want %q", toolName, tool.Name())
	}
}

func TestToolDefinition_ToTool(t *testing.T) {
	schema := json.RawMessage(`{"type": "object"}`)
	def := &ToolDefinition{
		Name:        "create_issue",
		Description: "Creates an issue",
		InputSchema: schema,
	}

	tool, err := def.ToTool("linear")
	if err != nil {
		t.Fatalf("failed to convert: %v", err)
	}

	if tool.Name() != def.Name {
		t.Errorf("Name() = %q, want %q", tool.Name(), def.Name)
	}
	if tool.Description() != def.Description {
		t.Errorf("Description() = %q, want %q", tool.Description(), def.Description)
	}
	if tool.ServerName() != "linear" {
		t.Errorf("ServerName() = %q, want %q", tool.ServerName(), "linear")
	}
}

func TestToolDefinition_ToTool_Invalid(t *testing.T) {
	def := &ToolDefinition{
		Name:        "",
		Description: "Invalid tool",
	}

	_, err := def.ToTool("linear")
	if err == nil {
		t.Error("expected error for invalid tool definition")
	}
	if !errors.Is(err, ErrInvalidToolName) {
		t.Errorf("error = %v, want %v", err, ErrInvalidToolName)
	}
}
