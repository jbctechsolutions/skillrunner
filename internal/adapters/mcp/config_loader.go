package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

// Compile-time interface assertion.
var _ ports.MCPConfigPort = (*ConfigLoader)(nil)

// claudeConfig represents the structure of .claude/mcp.json
type claudeConfig struct {
	MCPServers map[string]serverEntry `json:"mcpServers"`
}

type serverEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// ConfigLoader loads MCP configurations from .claude/mcp.json
type ConfigLoader struct{}

// NewConfigLoader creates a new ConfigLoader.
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

// Load reads MCP server configurations from the user's home directory.
func (l *ConfigLoader) Load(ctx context.Context) (map[string]domainMCP.ServerConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".claude", "mcp.json")
	return l.LoadFromPath(ctx, configPath)
}

// LoadFromPath reads MCP configuration from a specific path.
func (l *ConfigLoader) LoadFromPath(ctx context.Context, path string) (map[string]domainMCP.ServerConfig, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- Path is from trusted configuration sources
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", domainMCP.ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config claudeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", domainMCP.ErrInvalidConfig, err)
	}

	result := make(map[string]domainMCP.ServerConfig, len(config.MCPServers))
	for name, entry := range config.MCPServers {
		serverConfig := domainMCP.ServerConfig{
			Name:    name,
			Command: entry.Command,
			Args:    entry.Args,
			Env:     entry.Env,
		}

		if err := serverConfig.Validate(); err != nil {
			continue // Skip invalid entries
		}

		result[name] = serverConfig
	}

	return result, nil
}
