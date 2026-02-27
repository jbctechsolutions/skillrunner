package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

func TestConfigLoader_LoadFromPath(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantCount  int
		wantErr    bool
		errTarget  error
		checkNames []string
	}{
		{
			name: "valid config with single server",
			content: `{
				"mcpServers": {
					"linear": {
						"command": "npx",
						"args": ["-y", "@anthropic/linear-mcp-server"]
					}
				}
			}`,
			wantCount:  1,
			wantErr:    false,
			checkNames: []string{"linear"},
		},
		{
			name: "valid config with multiple servers",
			content: `{
				"mcpServers": {
					"linear": {
						"command": "npx",
						"args": ["-y", "@anthropic/linear-mcp-server"]
					},
					"github": {
						"command": "npx",
						"args": ["-y", "@anthropic/github-mcp-server"],
						"env": {
							"GITHUB_TOKEN": "test-token"
						}
					}
				}
			}`,
			wantCount:  2,
			wantErr:    false,
			checkNames: []string{"linear", "github"},
		},
		{
			name: "empty mcpServers",
			content: `{
				"mcpServers": {}
			}`,
			wantCount:  0,
			wantErr:    false,
			checkNames: []string{},
		},
		{
			name:      "invalid JSON",
			content:   `{ invalid json }`,
			wantErr:   true,
			errTarget: domainMCP.ErrInvalidConfig,
		},
		{
			name: "server with missing command (skipped)",
			content: `{
				"mcpServers": {
					"valid": {
						"command": "npx",
						"args": []
					},
					"invalid": {
						"args": ["test"]
					}
				}
			}`,
			wantCount:  1,
			wantErr:    false,
			checkNames: []string{"valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "mcp.json")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			loader := NewConfigLoader()
			configs, err := loader.LoadFromPath(context.Background(), configPath)

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

			if len(configs) != tt.wantCount {
				t.Errorf("got %d configs, want %d", len(configs), tt.wantCount)
			}

			for _, name := range tt.checkNames {
				if _, exists := configs[name]; !exists {
					t.Errorf("expected config for %q", name)
				}
			}
		})
	}
}

func TestConfigLoader_LoadFromPath_FileNotFound(t *testing.T) {
	loader := NewConfigLoader()
	_, err := loader.LoadFromPath(context.Background(), "/nonexistent/path/mcp.json")

	if err == nil {
		t.Error("expected error for non-existent file")
		return
	}

	if !errors.Is(err, domainMCP.ErrConfigNotFound) {
		t.Errorf("error = %v, want %v", err, domainMCP.ErrConfigNotFound)
	}
}

func TestConfigLoader_ServerConfigFields(t *testing.T) {
	content := `{
		"mcpServers": {
			"test": {
				"command": "npx",
				"args": ["-y", "@test/server"],
				"env": {
					"API_KEY": "test-key",
					"DEBUG": "true"
				}
			}
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	loader := NewConfigLoader()
	configs, err := loader.LoadFromPath(context.Background(), configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, exists := configs["test"]
	if !exists {
		t.Fatal("expected config for 'test'")
	}

	if config.Name != "test" {
		t.Errorf("Name = %q, want %q", config.Name, "test")
	}
	if config.Command != "npx" {
		t.Errorf("Command = %q, want %q", config.Command, "npx")
	}
	if len(config.Args) != 2 {
		t.Errorf("Args length = %d, want %d", len(config.Args), 2)
	}
	if config.Args[0] != "-y" || config.Args[1] != "@test/server" {
		t.Errorf("Args = %v, want %v", config.Args, []string{"-y", "@test/server"})
	}
	if len(config.Env) != 2 {
		t.Errorf("Env length = %d, want %d", len(config.Env), 2)
	}
	if config.Env["API_KEY"] != "test-key" {
		t.Errorf("Env[API_KEY] = %q, want %q", config.Env["API_KEY"], "test-key")
	}
}

func TestConfigLoader_LoadFlatFormat(t *testing.T) {
	content := `{
		"filesystem": {
			"command": "npx",
			"args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/user"]
		},
		"git": {
			"command": "npx",
			"args": ["-y", "@modelcontextprotocol/server-git"]
		}
	}`

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "mcp_servers.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	loader := NewConfigLoader()
	configs, err := loader.loadFlatFormat(path)
	if err != nil {
		t.Fatalf("loadFlatFormat: %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("got %d configs, want 2", len(configs))
	}
	if _, ok := configs["filesystem"]; !ok {
		t.Error("expected 'filesystem' entry")
	}
	if _, ok := configs["git"]; !ok {
		t.Error("expected 'git' entry")
	}
	if configs["filesystem"].Command != "npx" {
		t.Errorf("filesystem command = %q, want npx", configs["filesystem"].Command)
	}
}

func TestConfigLoader_Load_MergesBothFormats(t *testing.T) {
	homeDir := t.TempDir()

	// Write ~/.claude/mcp.json (Claude format)
	claudeDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	claudeContent := `{"mcpServers": {"from-claude": {"command": "cmd-a", "args": []}}}`
	if err := os.WriteFile(filepath.Join(claudeDir, "mcp.json"), []byte(claudeContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write ~/.skillrunner/mcp_servers.json (flat format — should take precedence on conflicts)
	srDir := filepath.Join(homeDir, ".skillrunner")
	if err := os.MkdirAll(srDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srContent := `{"from-sr": {"command": "cmd-b", "args": []}, "from-claude": {"command": "cmd-override", "args": []}}`
	if err := os.WriteFile(filepath.Join(srDir, "mcp_servers.json"), []byte(srContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Temporarily override HOME
	t.Setenv("HOME", homeDir)

	loader := NewConfigLoader()
	configs, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, ok := configs["from-sr"]; !ok {
		t.Error("expected 'from-sr' from skillrunner config")
	}
	// skillrunner entry should override Claude entry for same name
	if cfg, ok := configs["from-claude"]; !ok {
		t.Error("expected 'from-claude' entry")
	} else if cfg.Command != "cmd-override" {
		t.Errorf("from-claude command = %q, want cmd-override (skillrunner should win)", cfg.Command)
	}
}

func TestClaudeConfigJSON(t *testing.T) {
	// Test that claudeConfig matches the expected JSON structure
	jsonStr := `{
		"mcpServers": {
			"linear": {
				"command": "npx",
				"args": ["-y", "@anthropic/linear-mcp-server"]
			}
		}
	}`

	var config claudeConfig
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(config.MCPServers) != 1 {
		t.Errorf("MCPServers length = %d, want %d", len(config.MCPServers), 1)
	}

	entry, exists := config.MCPServers["linear"]
	if !exists {
		t.Fatal("expected entry for 'linear'")
	}

	if entry.Command != "npx" {
		t.Errorf("Command = %q, want %q", entry.Command, "npx")
	}
}
