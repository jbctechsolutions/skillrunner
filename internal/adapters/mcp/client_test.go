package mcp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

func getMockServerPath(t *testing.T) string {
	t.Helper()

	// Get the path to the mock server
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}

	return filepath.Join(filepath.Dir(filename), "testdata", "mock_server.go")
}

func TestClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mockServerPath := getMockServerPath(t)
	if _, err := os.Stat(mockServerPath); os.IsNotExist(err) {
		t.Skipf("mock server not found at %s", mockServerPath)
	}

	config := domainMCP.ServerConfig{
		Name:    "test",
		Command: "go",
		Args:    []string{"run", mockServerPath},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Test Initialize
	t.Run("Initialize", func(t *testing.T) {
		if err := client.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		info := client.GetProtocolInfo()
		if info == nil {
			t.Fatal("protocol info is nil")
		}
		if info.ProtocolVersion != "2024-11-05" {
			t.Errorf("ProtocolVersion = %q, want %q", info.ProtocolVersion, "2024-11-05")
		}
		if info.ServerName != "mock-mcp-server" {
			t.Errorf("ServerName = %q, want %q", info.ServerName, "mock-mcp-server")
		}
	})

	// Test DiscoverTools
	t.Run("DiscoverTools", func(t *testing.T) {
		tools, err := client.DiscoverTools(ctx)
		if err != nil {
			t.Fatalf("failed to discover tools: %v", err)
		}

		if len(tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(tools))
		}

		// Check first tool
		var testTool *domainMCP.Tool
		for _, tool := range tools {
			if tool.Name() == "test_tool" {
				testTool = tool
				break
			}
		}

		if testTool == nil {
			t.Error("expected to find test_tool")
		} else {
			if testTool.Description() != "A test tool" {
				t.Errorf("Description = %q, want %q", testTool.Description(), "A test tool")
			}
			if testTool.ServerName() != "test" {
				t.Errorf("ServerName = %q, want %q", testTool.ServerName(), "test")
			}
		}
	})

	// Test CallTool
	t.Run("CallTool", func(t *testing.T) {
		result, err := client.CallTool(ctx, "test_tool", map[string]any{
			"message": "hello",
		})
		if err != nil {
			t.Fatalf("failed to call tool: %v", err)
		}

		if len(result.Content) == 0 {
			t.Error("expected content in result")
		}

		text := result.TextContent()
		if text != "Test tool executed successfully" {
			t.Errorf("result = %q, want %q", text, "Test tool executed successfully")
		}
	})

	// Test CallTool with echo
	t.Run("CallTool_Echo", func(t *testing.T) {
		result, err := client.CallTool(ctx, "echo", map[string]any{
			"text": "Hello, World!",
		})
		if err != nil {
			t.Fatalf("failed to call tool: %v", err)
		}

		text := result.TextContent()
		if text != "Hello, World!" {
			t.Errorf("result = %q, want %q", text, "Hello, World!")
		}
	})

	// Test GetTools
	t.Run("GetTools", func(t *testing.T) {
		tools := client.GetTools()
		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})

	// Test PID
	t.Run("PID", func(t *testing.T) {
		pid := client.PID()
		if pid == 0 {
			t.Error("expected non-zero PID")
		}
	})
}

func TestClient_InvalidConfig(t *testing.T) {
	config := domainMCP.ServerConfig{
		Name:    "",
		Command: "",
	}

	ctx := context.Background()
	_, err := NewClient(ctx, config)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestClient_CommandNotFound(t *testing.T) {
	config := domainMCP.ServerConfig{
		Name:    "test",
		Command: "nonexistent-command-12345",
	}

	ctx := context.Background()
	_, err := NewClient(ctx, config)
	if err == nil {
		t.Error("expected error for non-existent command")
	}
}

func TestMapToEnvSlice(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]string
		want int
	}{
		{
			name: "empty map",
			m:    map[string]string{},
			want: 0,
		},
		{
			name: "single entry",
			m:    map[string]string{"KEY": "value"},
			want: 1,
		},
		{
			name: "multiple entries",
			m:    map[string]string{"KEY1": "value1", "KEY2": "value2"},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapToEnvSlice(tt.m)
			if len(result) != tt.want {
				t.Errorf("len(result) = %d, want %d", len(result), tt.want)
			}
		})
	}
}
