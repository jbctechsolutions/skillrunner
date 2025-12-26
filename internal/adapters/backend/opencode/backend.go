// Package opencode provides the OpenCode backend implementation.
package opencode

import (
	"context"
	"fmt"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// Backend implements the BackendPort interface for OpenCode.
// This is a stub implementation for future integration.
type Backend struct {
	machineID string
}

// NewBackend creates a new OpenCode backend.
func NewBackend(machineID string) (*Backend, error) {
	return &Backend{
		machineID: machineID,
	}, nil
}

// Info returns metadata about the OpenCode backend.
func (b *Backend) Info() ports.BackendInfo {
	return ports.BackendInfo{
		Name:        "opencode",
		Version:     "0.1.0",
		Description: "OpenCode - Multi-session coding assistant (stub)",
		Executable:  "opencode",
		Features:    []string{"multi-session", "workspace-aware"},
	}
}

// Start creates and starts a new OpenCode session.
func (b *Backend) Start(ctx context.Context, workspace string, config session.BackendConfig) (*session.Session, error) {
	return nil, fmt.Errorf("opencode backend not yet implemented")
}

// Attach connects to an existing OpenCode session.
func (b *Backend) Attach(ctx context.Context, sessionID string) error {
	return fmt.Errorf("opencode backend not yet implemented")
}

// Detach disconnects from a session without killing it.
func (b *Backend) Detach(ctx context.Context) error {
	return fmt.Errorf("opencode backend not yet implemented")
}

// Kill terminates an OpenCode session.
func (b *Backend) Kill(ctx context.Context, sessionID string) error {
	return fmt.Errorf("opencode backend not yet implemented")
}

// InjectContext injects contextual information into a running session.
func (b *Backend) InjectContext(ctx context.Context, sessionID, content string) error {
	return fmt.Errorf("opencode backend not yet implemented")
}

// InjectFile injects a file into a running session.
func (b *Backend) InjectFile(ctx context.Context, sessionID, path string) error {
	return fmt.Errorf("opencode backend not yet implemented")
}

// GetStatus retrieves the current status of a session.
func (b *Backend) GetStatus(ctx context.Context, sessionID string) (*ports.SessionStatus, error) {
	return nil, fmt.Errorf("opencode backend not yet implemented")
}

// GetTokenUsage retrieves token usage statistics.
func (b *Backend) GetTokenUsage(ctx context.Context, sessionID string) (*session.TokenUsage, error) {
	return nil, fmt.Errorf("opencode backend not yet implemented")
}

// SetModel changes the model for a session.
func (b *Backend) SetModel(ctx context.Context, model string) error {
	return fmt.Errorf("opencode backend not yet implemented")
}

// GetSupportedModels returns a list of models supported by OpenCode.
func (b *Backend) GetSupportedModels(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// SupportsModelControl indicates whether the backend supports changing models.
func (b *Backend) SupportsModelControl() bool {
	return true
}
