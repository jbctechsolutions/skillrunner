package mcp

import (
	"errors"
	"testing"
)

func TestServerState_String(t *testing.T) {
	tests := []struct {
		state ServerState
		want  string
	}{
		{ServerStateStopped, "stopped"},
		{ServerStateStarting, "starting"},
		{ServerStateInitializing, "initializing"},
		{ServerStateReady, "ready"},
		{ServerStateStopping, "stopping"},
		{ServerStateError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServerState_IsRunning(t *testing.T) {
	tests := []struct {
		state ServerState
		want  bool
	}{
		{ServerStateStopped, false},
		{ServerStateStarting, false},
		{ServerStateInitializing, false},
		{ServerStateReady, true},
		{ServerStateStopping, false},
		{ServerStateError, false},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			if got := tt.state.IsRunning(); got != tt.want {
				t.Errorf("IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerState_IsTerminal(t *testing.T) {
	tests := []struct {
		state ServerState
		want  bool
	}{
		{ServerStateStopped, true},
		{ServerStateStarting, false},
		{ServerStateInitializing, false},
		{ServerStateReady, false},
		{ServerStateStopping, false},
		{ServerStateError, true},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.want {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    ServerConfig
		wantErr   bool
		errTarget error
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Name:    "linear",
				Command: "npx",
				Args:    []string{"-y", "@anthropic/linear-mcp-server"},
			},
			wantErr: false,
		},
		{
			name: "valid config with env",
			config: ServerConfig{
				Name:    "linear",
				Command: "npx",
				Args:    []string{"-y", "@anthropic/linear-mcp-server"},
				Env: map[string]string{
					"LINEAR_API_KEY": "test-key",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with workdir",
			config: ServerConfig{
				Name:    "linear",
				Command: "npx",
				Args:    []string{"-y", "@anthropic/linear-mcp-server"},
				WorkDir: "/home/user/project",
			},
			wantErr: false,
		},
		{
			name: "valid config minimal",
			config: ServerConfig{
				Name:    "test",
				Command: "echo",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: ServerConfig{
				Name:    "",
				Command: "npx",
			},
			wantErr:   true,
			errTarget: ErrInvalidConfig,
		},
		{
			name: "whitespace only name",
			config: ServerConfig{
				Name:    "   ",
				Command: "npx",
			},
			wantErr:   true,
			errTarget: ErrInvalidConfig,
		},
		{
			name: "empty command",
			config: ServerConfig{
				Name:    "linear",
				Command: "",
			},
			wantErr:   true,
			errTarget: ErrInvalidConfig,
		},
		{
			name: "whitespace only command",
			config: ServerConfig{
				Name:    "linear",
				Command: "   ",
			},
			wantErr:   true,
			errTarget: ErrInvalidConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

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
			}
		})
	}
}

func TestServerInfo(t *testing.T) {
	info := ServerInfo{
		Name:         "linear",
		State:        ServerStateReady,
		PID:          12345,
		ToolCount:    5,
		ErrorMessage: "",
	}

	if info.Name != "linear" {
		t.Errorf("Name = %q, want %q", info.Name, "linear")
	}
	if info.State != ServerStateReady {
		t.Errorf("State = %v, want %v", info.State, ServerStateReady)
	}
	if info.PID != 12345 {
		t.Errorf("PID = %d, want %d", info.PID, 12345)
	}
	if info.ToolCount != 5 {
		t.Errorf("ToolCount = %d, want %d", info.ToolCount, 5)
	}
	if info.ErrorMessage != "" {
		t.Errorf("ErrorMessage = %q, want empty", info.ErrorMessage)
	}
}

func TestServerCapabilities(t *testing.T) {
	caps := ServerCapabilities{
		Tools: &ToolsCapability{
			ListChanged: true,
		},
		Resources: &ResourcesCapability{
			Subscribe:   true,
			ListChanged: false,
		},
		Prompts: nil,
	}

	if caps.Tools == nil {
		t.Error("Tools should not be nil")
	}
	if !caps.Tools.ListChanged {
		t.Error("Tools.ListChanged should be true")
	}

	if caps.Resources == nil {
		t.Error("Resources should not be nil")
	}
	if !caps.Resources.Subscribe {
		t.Error("Resources.Subscribe should be true")
	}

	if caps.Prompts != nil {
		t.Error("Prompts should be nil")
	}
}

func TestProtocolInfo(t *testing.T) {
	info := ProtocolInfo{
		ProtocolVersion: "2024-11-05",
		ServerName:      "linear-mcp-server",
		ServerVersion:   "1.0.0",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{ListChanged: true},
		},
	}

	if info.ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q, want %q", info.ProtocolVersion, "2024-11-05")
	}
	if info.ServerName != "linear-mcp-server" {
		t.Errorf("ServerName = %q, want %q", info.ServerName, "linear-mcp-server")
	}
	if info.ServerVersion != "1.0.0" {
		t.Errorf("ServerVersion = %q, want %q", info.ServerVersion, "1.0.0")
	}
	if info.Capabilities.Tools == nil {
		t.Error("Capabilities.Tools should not be nil")
	}
}
