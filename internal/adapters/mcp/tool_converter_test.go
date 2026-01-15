package mcp

import (
	"encoding/json"
	"testing"

	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

func TestToProviderTools(t *testing.T) {
	tests := []struct {
		name         string
		tools        []*domainMCP.Tool
		deferLoading bool
		wantCount    int
	}{
		{
			name:      "nil tools",
			tools:     nil,
			wantCount: 0,
		},
		{
			name:      "empty tools",
			tools:     []*domainMCP.Tool{},
			wantCount: 0,
		},
		{
			name: "single tool",
			tools: func() []*domainMCP.Tool {
				tool, _ := domainMCP.NewTool("create_issue", "Creates an issue", json.RawMessage(`{}`), "linear")
				return []*domainMCP.Tool{tool}
			}(),
			deferLoading: false,
			wantCount:    1,
		},
		{
			name: "multiple tools",
			tools: func() []*domainMCP.Tool {
				tool1, _ := domainMCP.NewTool("create_issue", "Creates an issue", nil, "linear")
				tool2, _ := domainMCP.NewTool("list_repos", "Lists repos", nil, "github")
				return []*domainMCP.Tool{tool1, tool2}
			}(),
			deferLoading: true,
			wantCount:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToProviderTools(tt.tools, tt.deferLoading)

			if len(result) != tt.wantCount {
				t.Errorf("len(result) = %d, want %d", len(result), tt.wantCount)
			}

			// Check defer loading is set correctly
			for _, tool := range result {
				if tool.DeferLoading != tt.deferLoading {
					t.Errorf("DeferLoading = %v, want %v", tool.DeferLoading, tt.deferLoading)
				}
			}
		})
	}
}

func TestToProviderTools_FullName(t *testing.T) {
	tool, err := domainMCP.NewTool("create_issue", "Creates an issue", json.RawMessage(`{"type":"object"}`), "linear")
	if err != nil {
		t.Fatalf("failed to create tool: %v", err)
	}

	result := ToProviderTools([]*domainMCP.Tool{tool}, false)

	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	// Name should be in mcp__server__tool format
	if result[0].Name != "mcp__linear__create_issue" {
		t.Errorf("Name = %q, want %q", result[0].Name, "mcp__linear__create_issue")
	}

	if result[0].Description != "Creates an issue" {
		t.Errorf("Description = %q, want %q", result[0].Description, "Creates an issue")
	}

	if string(result[0].InputSchema) != `{"type":"object"}` {
		t.Errorf("InputSchema = %q, want %q", string(result[0].InputSchema), `{"type":"object"}`)
	}
}

func TestToProviderTool(t *testing.T) {
	tool, err := domainMCP.NewTool("echo", "Echoes input", nil, "test")
	if err != nil {
		t.Fatalf("failed to create tool: %v", err)
	}

	// Test without defer loading
	result := ToProviderTool(tool, false)

	if result.Name != "mcp__test__echo" {
		t.Errorf("Name = %q, want %q", result.Name, "mcp__test__echo")
	}
	if result.DeferLoading {
		t.Error("DeferLoading should be false")
	}

	// Test with defer loading
	result = ToProviderTool(tool, true)

	if !result.DeferLoading {
		t.Error("DeferLoading should be true")
	}
}
