package mcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

func getTestServerConfig(t *testing.T) domainMCP.ServerConfig {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}

	mockServerPath := filepath.Join(filepath.Dir(filename), "testdata", "mock_server.go")

	return domainMCP.ServerConfig{
		Name:    "test",
		Command: "go",
		Args:    []string{"run", mockServerPath},
	}
}

func TestServerManager_RegisterConfig(t *testing.T) {
	m := NewServerManager()

	config := domainMCP.ServerConfig{
		Name:    "test",
		Command: "echo",
	}

	if err := m.RegisterConfig(config); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should be able to get the config
	got, exists := m.GetConfig("test")
	if !exists {
		t.Error("expected config to exist")
	}
	if got.Name != config.Name {
		t.Errorf("Name = %q, want %q", got.Name, config.Name)
	}
}

func TestServerManager_RegisterConfig_Invalid(t *testing.T) {
	m := NewServerManager()

	config := domainMCP.ServerConfig{
		Name:    "",
		Command: "",
	}

	err := m.RegisterConfig(config)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestServerManager_IsRunning_NotStarted(t *testing.T) {
	m := NewServerManager()

	if m.IsRunning("nonexistent") {
		t.Error("IsRunning should return false for non-existent server")
	}
}

func TestServerManager_GetInfo_NotFound(t *testing.T) {
	m := NewServerManager()

	_, err := m.GetInfo("nonexistent")
	if !errors.Is(err, domainMCP.ErrServerNotFound) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrServerNotFound)
	}
}

func TestServerManager_Stop_NotFound(t *testing.T) {
	m := NewServerManager()

	err := m.Stop(context.Background(), "nonexistent")
	if !errors.Is(err, domainMCP.ErrServerNotFound) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrServerNotFound)
	}
}

func TestServerManager_ListServers_Empty(t *testing.T) {
	m := NewServerManager()

	servers := m.ListServers()
	if len(servers) != 0 {
		t.Errorf("expected empty list, got %d servers", len(servers))
	}
}

func TestServerManager_ListTools_NotFound(t *testing.T) {
	m := NewServerManager()

	_, err := m.ListTools(context.Background(), "nonexistent")
	if !errors.Is(err, domainMCP.ErrServerNotFound) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrServerNotFound)
	}
}

func TestServerManager_GetTool_NotFound(t *testing.T) {
	m := NewServerManager()

	_, err := m.GetTool(context.Background(), "nonexistent", "tool")
	if !errors.Is(err, domainMCP.ErrServerNotFound) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrServerNotFound)
	}
}

func TestServerManager_CallTool_NotFound(t *testing.T) {
	m := NewServerManager()

	_, err := m.CallTool(context.Background(), "nonexistent", "tool", nil)
	if !errors.Is(err, domainMCP.ErrServerNotFound) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrServerNotFound)
	}
}

func TestServerManager_StopAll_Empty(t *testing.T) {
	m := NewServerManager()

	err := m.StopAll(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestServerManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := getTestServerConfig(t)
	mockServerPath := config.Args[1]
	if _, err := os.Stat(mockServerPath); os.IsNotExist(err) {
		t.Skipf("mock server not found at %s", mockServerPath)
	}

	m := NewServerManager()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test Start
	t.Run("Start", func(t *testing.T) {
		if err := m.Start(ctx, config); err != nil {
			t.Fatalf("failed to start server: %v", err)
		}
	})

	// Test IsRunning
	t.Run("IsRunning", func(t *testing.T) {
		if !m.IsRunning("test") {
			t.Error("server should be running")
		}
	})

	// Test Start again (should error)
	t.Run("Start_AlreadyRunning", func(t *testing.T) {
		err := m.Start(ctx, config)
		if !errors.Is(err, domainMCP.ErrServerAlreadyRunning) {
			t.Errorf("error = %v, want %v", err, domainMCP.ErrServerAlreadyRunning)
		}
	})

	// Test GetInfo
	t.Run("GetInfo", func(t *testing.T) {
		info, err := m.GetInfo("test")
		if err != nil {
			t.Fatalf("failed to get info: %v", err)
		}

		if info.Name != "test" {
			t.Errorf("Name = %q, want %q", info.Name, "test")
		}
		if info.State != domainMCP.ServerStateReady {
			t.Errorf("State = %v, want %v", info.State, domainMCP.ServerStateReady)
		}
		if info.PID == 0 {
			t.Error("expected non-zero PID")
		}
		if info.ToolCount != 2 {
			t.Errorf("ToolCount = %d, want %d", info.ToolCount, 2)
		}
	})

	// Test ListServers
	t.Run("ListServers", func(t *testing.T) {
		servers := m.ListServers()
		if len(servers) != 1 {
			t.Errorf("expected 1 server, got %d", len(servers))
		}
	})

	// Test ListTools
	t.Run("ListTools", func(t *testing.T) {
		tools, err := m.ListTools(ctx, "test")
		if err != nil {
			t.Fatalf("failed to list tools: %v", err)
		}

		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})

	// Test GetTool
	t.Run("GetTool", func(t *testing.T) {
		tool, err := m.GetTool(ctx, "test", "test_tool")
		if err != nil {
			t.Fatalf("failed to get tool: %v", err)
		}

		if tool.Name() != "test_tool" {
			t.Errorf("Name = %q, want %q", tool.Name(), "test_tool")
		}
	})

	// Test GetTool not found
	t.Run("GetTool_NotFound", func(t *testing.T) {
		_, err := m.GetTool(ctx, "test", "nonexistent")
		if !errors.Is(err, domainMCP.ErrToolNotFound) {
			t.Errorf("error = %v, want %v", err, domainMCP.ErrToolNotFound)
		}
	})

	// Test CallTool
	t.Run("CallTool", func(t *testing.T) {
		result, err := m.CallTool(ctx, "test", "echo", map[string]any{
			"text": "Hello from ServerManager",
		})
		if err != nil {
			t.Fatalf("failed to call tool: %v", err)
		}

		text := result.TextContent()
		if text != "Hello from ServerManager" {
			t.Errorf("result = %q, want %q", text, "Hello from ServerManager")
		}
	})

	// Test Stop
	t.Run("Stop", func(t *testing.T) {
		if err := m.Stop(ctx, "test"); err != nil {
			t.Fatalf("failed to stop server: %v", err)
		}

		if m.IsRunning("test") {
			t.Error("server should not be running after stop")
		}
	})
}
