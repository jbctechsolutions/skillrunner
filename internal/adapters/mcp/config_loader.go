package mcp

import (
	"context"
	"encoding/json"
	"errors"
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
// It checks, in order:
//  1. ~/.skillrunner/mcp_servers.json  (flat format: {"name": {command, args, env}})
//  2. ~/.claude/mcp.json               (Claude format: {"mcpServers": {name: {command, args, env}}})
//
// Results from both files are merged; ~/.skillrunner entries take precedence on name conflicts.
func (l *ConfigLoader) Load(ctx context.Context) (map[string]domainMCP.ServerConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	merged := make(map[string]domainMCP.ServerConfig)

	// 1. ~/.claude/mcp.json (Claude format)
	claudePath := filepath.Join(homeDir, ".claude", "mcp.json")
	claudeConfigs, claudeErr := l.LoadFromPath(ctx, claudePath)
	if claudeErr == nil {
		for k, v := range claudeConfigs {
			merged[k] = v
		}
	} else if !errors.Is(claudeErr, domainMCP.ErrConfigNotFound) {
		return nil, fmt.Errorf("invalid Claude MCP config at %s: %w", claudePath, claudeErr)
	}

	// 2. ~/.skillrunner/mcp_servers.json (flat format — takes precedence)
	srPath := filepath.Join(homeDir, ".skillrunner", "mcp_servers.json")
	srConfigs, srErr := l.loadFlatFormat(srPath)
	if srErr == nil {
		for k, v := range srConfigs {
			merged[k] = v
		}
	} else if !os.IsNotExist(srErr) {
		return nil, fmt.Errorf("invalid MCP config at %s: %w", srPath, srErr)
	}

	return merged, nil
}

// loadFlatFormat reads a flat JSON file mapping server names to their configs directly.
// Format: {"server-name": {"command": "...", "args": [...], "env": {...}}}
func (l *ConfigLoader) loadFlatFormat(path string) (map[string]domainMCP.ServerConfig, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- Path is from trusted configuration sources
	if err != nil {
		return nil, err
	}

	var flat map[string]serverEntry
	if err := json.Unmarshal(data, &flat); err != nil {
		return nil, fmt.Errorf("%w: %v", domainMCP.ErrInvalidConfig, err)
	}

	result := make(map[string]domainMCP.ServerConfig, len(flat))
	for name, entry := range flat {
		cfg := domainMCP.ServerConfig{
			Name:    name,
			Command: entry.Command,
			Args:    entry.Args,
			Env:     entry.Env,
		}
		if err := cfg.Validate(); err != nil {
			continue
		}
		result[name] = cfg
	}
	return result, nil
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
