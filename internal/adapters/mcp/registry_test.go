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

func TestRegistry_LoadConfigs_FileNotFound(t *testing.T) {
	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	// Loading configs when file doesn't exist should not error
	// (MCP is optional)
	err := registry.LoadConfigs(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegistry_LoadConfigs_Valid(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "mcp.json")
	content := `{
		"mcpServers": {
			"test": {
				"command": "echo",
				"args": ["hello"]
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	// Load from specific path
	configs, err := loader.LoadFromPath(context.Background(), configPath)
	if err != nil {
		t.Fatalf("failed to load configs: %v", err)
	}

	// Manually set configs (since Load() uses home directory)
	registry.mu.Lock()
	registry.configs = configs
	registry.mu.Unlock()

	// Check configured servers
	servers := registry.ListConfiguredServers()
	if len(servers) != 1 {
		t.Errorf("expected 1 configured server, got %d", len(servers))
	}

	config, exists := registry.GetServerConfig("test")
	if !exists {
		t.Error("expected config for 'test'")
	}
	if config.Command != "echo" {
		t.Errorf("Command = %q, want %q", config.Command, "echo")
	}
}

func TestRegistry_GetAllTools_Empty(t *testing.T) {
	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	tools, err := registry.GetAllTools(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected empty tools list, got %d", len(tools))
	}
}

func TestRegistry_CallToolByFullName_InvalidFormat(t *testing.T) {
	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	_, err := registry.CallToolByFullName(context.Background(), "invalid", nil)
	if !errors.Is(err, domainMCP.ErrInvalidToolName) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrInvalidToolName)
	}
}

func TestRegistry_CallToolByFullName_ServerNotFound(t *testing.T) {
	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	_, err := registry.CallToolByFullName(context.Background(), "mcp__nonexistent__tool", nil)
	if !errors.Is(err, domainMCP.ErrServerNotFound) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrServerNotFound)
	}
}

func TestRegistry_EnsureServerRunning_NotConfigured(t *testing.T) {
	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	err := registry.EnsureServerRunning(context.Background(), "nonexistent")
	if !errors.Is(err, domainMCP.ErrServerNotFound) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrServerNotFound)
	}
}

func TestRegistry_Manager(t *testing.T) {
	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	if registry.Manager() != manager {
		t.Error("Manager() should return the underlying manager")
	}
}

func TestRegistry_Close_Empty(t *testing.T) {
	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	err := registry.Close(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegistry_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}

	mockServerPath := filepath.Join(filepath.Dir(filename), "testdata", "mock_server.go")
	if _, err := os.Stat(mockServerPath); os.IsNotExist(err) {
		t.Skipf("mock server not found at %s", mockServerPath)
	}

	manager := NewServerManager()
	loader := NewConfigLoader()
	registry := NewRegistry(manager, loader)

	// Manually configure a server
	config := domainMCP.ServerConfig{
		Name:    "test",
		Command: "go",
		Args:    []string{"run", mockServerPath},
	}

	registry.mu.Lock()
	registry.configs["test"] = config
	registry.mu.Unlock()
	manager.RegisterConfig(config)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test EnsureServerRunning
	t.Run("EnsureServerRunning", func(t *testing.T) {
		if err := registry.EnsureServerRunning(ctx, "test"); err != nil {
			t.Fatalf("failed to ensure server running: %v", err)
		}

		if !manager.IsRunning("test") {
			t.Error("server should be running")
		}
	})

	// Test EnsureServerRunning when already running
	t.Run("EnsureServerRunning_AlreadyRunning", func(t *testing.T) {
		if err := registry.EnsureServerRunning(ctx, "test"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Test GetAllTools
	t.Run("GetAllTools", func(t *testing.T) {
		tools, err := registry.GetAllTools(ctx)
		if err != nil {
			t.Fatalf("failed to get all tools: %v", err)
		}

		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})

	// Test CallToolByFullName
	t.Run("CallToolByFullName", func(t *testing.T) {
		result, err := registry.CallToolByFullName(ctx, "mcp__test__echo", map[string]any{
			"text": "Hello from Registry",
		})
		if err != nil {
			t.Fatalf("failed to call tool: %v", err)
		}

		text := result.TextContent()
		if text != "Hello from Registry" {
			t.Errorf("result = %q, want %q", text, "Hello from Registry")
		}
	})

	// Test Close
	t.Run("Close", func(t *testing.T) {
		if err := registry.Close(ctx); err != nil {
			t.Fatalf("failed to close registry: %v", err)
		}

		if manager.IsRunning("test") {
			t.Error("server should not be running after close")
		}
	})
}
