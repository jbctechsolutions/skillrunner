package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

// Compile-time interface assertion.
var _ ports.MCPToolRegistryPort = (*Registry)(nil)

// Registry provides unified access to MCP tools across all servers.
type Registry struct {
	manager *ServerManager
	loader  *ConfigLoader
	configs map[string]domainMCP.ServerConfig
	mu      sync.RWMutex
}

// NewRegistry creates a new MCP Registry.
func NewRegistry(manager *ServerManager, loader *ConfigLoader) *Registry {
	return &Registry{
		manager: manager,
		loader:  loader,
		configs: make(map[string]domainMCP.ServerConfig),
	}
}

// LoadConfigs loads MCP configurations and stores them for later use.
func (r *Registry) LoadConfigs(ctx context.Context) error {
	configs, err := r.loader.Load(ctx)
	if err != nil {
		// Config not found is not an error - MCP is optional
		if domainMCP.Is(err, domainMCP.ErrConfigNotFound) {
			return nil
		}
		return err
	}

	r.mu.Lock()
	r.configs = configs
	r.mu.Unlock()

	// Register configs with manager
	for _, config := range configs {
		_ = r.manager.RegisterConfig(config)
	}

	return nil
}

// GetAllTools returns all tools from all running servers.
func (r *Registry) GetAllTools(ctx context.Context) ([]*domainMCP.Tool, error) {
	servers := r.manager.ListServers()

	var allTools []*domainMCP.Tool
	for _, server := range servers {
		if server.State != domainMCP.ServerStateReady {
			continue
		}

		tools, err := r.manager.ListTools(ctx, server.Name)
		if err != nil {
			continue
		}

		allTools = append(allTools, tools...)
	}

	return allTools, nil
}

// CallToolByFullName executes a tool using its full name (mcp__server__tool).
func (r *Registry) CallToolByFullName(ctx context.Context, fullName string, arguments map[string]any) (*domainMCP.ToolCallResult, error) {
	serverName, toolName, err := domainMCP.ParseToolName(fullName)
	if err != nil {
		return nil, err
	}

	// Ensure server is running
	if err := r.EnsureServerRunning(ctx, serverName); err != nil {
		return nil, err
	}

	return r.manager.CallTool(ctx, serverName, toolName, arguments)
}

// EnsureServerRunning starts a server if not already running.
func (r *Registry) EnsureServerRunning(ctx context.Context, serverName string) error {
	if r.manager.IsRunning(serverName) {
		return nil
	}

	r.mu.RLock()
	config, exists := r.configs[serverName]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%w: %s", domainMCP.ErrServerNotFound, serverName)
	}

	return r.manager.Start(ctx, config)
}

// ListConfiguredServers returns the names of all configured servers.
func (r *Registry) ListConfiguredServers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.configs))
	for name := range r.configs {
		names = append(names, name)
	}
	return names
}

// GetServerConfig returns the configuration for a server.
func (r *Registry) GetServerConfig(serverName string) (domainMCP.ServerConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.configs[serverName]
	return config, exists
}

// Manager returns the underlying ServerManager.
func (r *Registry) Manager() *ServerManager {
	return r.manager
}

// Close stops all servers and cleans up resources.
func (r *Registry) Close(ctx context.Context) error {
	return r.manager.StopAll(ctx)
}
