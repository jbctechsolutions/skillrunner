package mcp

import (
	"context"
	"sync"
	"time"

	domainMCP "github.com/jbctechsolutions/skillrunner/internal/domain/mcp"
)

// serverInstance holds the runtime state of an MCP server.
type serverInstance struct {
	config       domainMCP.ServerConfig
	client       *Client
	state        domainMCP.ServerState
	startedAt    time.Time
	lastActivity time.Time
	tools        []*domainMCP.Tool
	err          error
}

// ServerManager manages the lifecycle of MCP servers.
type ServerManager struct {
	mu      sync.RWMutex
	servers map[string]*serverInstance
	configs map[string]domainMCP.ServerConfig
}

// NewServerManager creates a new ServerManager.
func NewServerManager() *ServerManager {
	return &ServerManager{
		servers: make(map[string]*serverInstance),
		configs: make(map[string]domainMCP.ServerConfig),
	}
}

// RegisterConfig registers a server configuration for later use.
func (m *ServerManager) RegisterConfig(config domainMCP.ServerConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.configs[config.Name] = config
	return nil
}

// GetConfig returns a registered server configuration.
func (m *ServerManager) GetConfig(serverName string) (domainMCP.ServerConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[serverName]
	return config, exists
}

// Start starts an MCP server and performs initialization.
func (m *ServerManager) Start(ctx context.Context, config domainMCP.ServerConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	m.mu.Lock()

	// Check if already running
	if inst, exists := m.servers[config.Name]; exists {
		if inst.state == domainMCP.ServerStateReady {
			m.mu.Unlock()
			return domainMCP.ErrServerAlreadyRunning
		}
	}

	// Create instance in starting state
	inst := &serverInstance{
		config:    config,
		state:     domainMCP.ServerStateStarting,
		startedAt: time.Now(),
	}
	m.servers[config.Name] = inst
	m.mu.Unlock()

	// Start the server
	client, err := NewClient(ctx, config)
	if err != nil {
		m.updateState(config.Name, domainMCP.ServerStateError, err)
		return err
	}

	m.mu.Lock()
	inst.client = client
	inst.state = domainMCP.ServerStateInitializing
	m.mu.Unlock()

	// Initialize the server
	if err := client.Initialize(ctx); err != nil {
		_ = client.Close(ctx)
		m.updateState(config.Name, domainMCP.ServerStateError, err)
		return err
	}

	// Discover tools
	tools, err := client.DiscoverTools(ctx)
	if err != nil {
		_ = client.Close(ctx)
		m.updateState(config.Name, domainMCP.ServerStateError, err)
		return err
	}

	m.mu.Lock()
	inst.tools = tools
	inst.state = domainMCP.ServerStateReady
	inst.lastActivity = time.Now()
	m.mu.Unlock()

	return nil
}

// Stop gracefully stops an MCP server.
func (m *ServerManager) Stop(ctx context.Context, serverName string) error {
	m.mu.Lock()
	inst, exists := m.servers[serverName]
	if !exists {
		m.mu.Unlock()
		return domainMCP.ErrServerNotFound
	}

	if inst.state == domainMCP.ServerStateStopped {
		m.mu.Unlock()
		return nil
	}

	inst.state = domainMCP.ServerStateStopping
	client := inst.client
	m.mu.Unlock()

	var closeErr error
	if client != nil {
		closeErr = client.Close(ctx)
	}

	m.mu.Lock()
	inst.state = domainMCP.ServerStateStopped
	inst.client = nil
	delete(m.servers, serverName)
	m.mu.Unlock()

	return closeErr
}

// StopAll stops all running servers.
func (m *ServerManager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	m.mu.RUnlock()

	var lastErr error
	for _, name := range names {
		if err := m.Stop(ctx, name); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetInfo returns information about a server.
func (m *ServerManager) GetInfo(serverName string) (*domainMCP.ServerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, exists := m.servers[serverName]
	if !exists {
		return nil, domainMCP.ErrServerNotFound
	}

	info := &domainMCP.ServerInfo{
		Name:         serverName,
		State:        inst.state,
		StartedAt:    inst.startedAt,
		ToolCount:    len(inst.tools),
		LastActivity: inst.lastActivity,
	}

	if inst.client != nil {
		info.PID = inst.client.PID()
	}

	if inst.err != nil {
		info.ErrorMessage = inst.err.Error()
	}

	return info, nil
}

// ListServers returns information about all servers.
func (m *ServerManager) ListServers() []domainMCP.ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]domainMCP.ServerInfo, 0, len(m.servers))
	for name, inst := range m.servers {
		info := domainMCP.ServerInfo{
			Name:         name,
			State:        inst.state,
			StartedAt:    inst.startedAt,
			ToolCount:    len(inst.tools),
			LastActivity: inst.lastActivity,
		}
		if inst.err != nil {
			info.ErrorMessage = inst.err.Error()
		}
		result = append(result, info)
	}

	return result
}

// IsRunning returns true if the server is in ready state.
func (m *ServerManager) IsRunning(serverName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, exists := m.servers[serverName]
	return exists && inst.state == domainMCP.ServerStateReady
}

// ListTools returns all tools from a server.
func (m *ServerManager) ListTools(ctx context.Context, serverName string) ([]*domainMCP.Tool, error) {
	m.mu.RLock()
	inst, exists := m.servers[serverName]
	if !exists {
		m.mu.RUnlock()
		return nil, domainMCP.ErrServerNotFound
	}

	if inst.state != domainMCP.ServerStateReady {
		m.mu.RUnlock()
		return nil, domainMCP.ErrServerNotRunning
	}

	tools := make([]*domainMCP.Tool, len(inst.tools))
	copy(tools, inst.tools)
	m.mu.RUnlock()

	return tools, nil
}

// GetTool returns a specific tool from a server.
func (m *ServerManager) GetTool(ctx context.Context, serverName, toolName string) (*domainMCP.Tool, error) {
	tools, err := m.ListTools(ctx, serverName)
	if err != nil {
		return nil, err
	}

	for _, tool := range tools {
		if tool.Name() == toolName {
			return tool, nil
		}
	}

	return nil, domainMCP.ErrToolNotFound
}

// CallTool executes a tool on a server.
func (m *ServerManager) CallTool(ctx context.Context, serverName, toolName string, arguments map[string]any) (*domainMCP.ToolCallResult, error) {
	m.mu.RLock()
	inst, exists := m.servers[serverName]
	if !exists {
		m.mu.RUnlock()
		return nil, domainMCP.ErrServerNotFound
	}

	if inst.state != domainMCP.ServerStateReady || inst.client == nil {
		m.mu.RUnlock()
		return nil, domainMCP.ErrServerNotRunning
	}

	client := inst.client
	m.mu.RUnlock()

	result, err := client.CallTool(ctx, toolName, arguments)

	// Update last activity
	m.mu.Lock()
	if inst, exists := m.servers[serverName]; exists {
		inst.lastActivity = time.Now()
	}
	m.mu.Unlock()

	return result, err
}

// updateState updates the state of a server.
func (m *ServerManager) updateState(serverName string, state domainMCP.ServerState, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inst, exists := m.servers[serverName]; exists {
		inst.state = state
		inst.err = err
	}
}
