// Package opencode provides the OpenCode backend implementation.
package opencode

import (
	"context"
	"fmt"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
)

// ExperimentalError represents an error for experimental/unimplemented features.
type ExperimentalError struct {
	Backend   string
	Operation string
}

func (e *ExperimentalError) Error() string {
	return fmt.Sprintf(
		"opencode backend is experimental and not yet implemented: %s operation unavailable. "+
			"The OpenCode integration is planned for a future release. "+
			"Please use 'claude' or 'aider' backends instead.",
		e.Operation,
	)
}

// IsExperimentalError returns true if the error is an ExperimentalError.
func IsExperimentalError(err error) bool {
	_, ok := err.(*ExperimentalError)
	return ok
}

// newExperimentalError creates a new ExperimentalError for the given operation.
func newExperimentalError(operation string) *ExperimentalError {
	return &ExperimentalError{
		Backend:   "opencode",
		Operation: operation,
	}
}

// Backend implements the BackendPort interface for OpenCode.
// NOTE: This backend is EXPERIMENTAL and not yet implemented.
// All operations will return ExperimentalError with guidance to use alternative backends.
type Backend struct {
	machineID string
}

// NewBackend creates a new OpenCode backend.
// Note: The backend is created successfully but all operations are experimental.
func NewBackend(machineID string) (*Backend, error) {
	return &Backend{
		machineID: machineID,
	}, nil
}

// Info returns metadata about the OpenCode backend.
// This method works even though the backend is experimental.
func (b *Backend) Info() ports.BackendInfo {
	return ports.BackendInfo{
		Name:        "opencode",
		Version:     "0.1.0-experimental",
		Description: "OpenCode - Multi-session coding assistant (EXPERIMENTAL: Not yet implemented)",
		Executable:  "opencode",
		Features:    []string{"experimental", "planned"},
	}
}

// IsExperimental returns true indicating this backend is not yet ready for use.
func (b *Backend) IsExperimental() bool {
	return true
}

// IsAvailable returns false indicating this backend cannot be used yet.
func (b *Backend) IsAvailable() bool {
	return false
}

// Start creates and starts a new OpenCode session.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) Start(ctx context.Context, workspace string, config session.BackendConfig) (*session.Session, error) {
	return nil, newExperimentalError("Start")
}

// Attach connects to an existing OpenCode session.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) Attach(ctx context.Context, sessionID string) error {
	return newExperimentalError("Attach")
}

// Detach disconnects from a session without killing it.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) Detach(ctx context.Context) error {
	return newExperimentalError("Detach")
}

// Kill terminates an OpenCode session.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) Kill(ctx context.Context, sessionID string) error {
	return newExperimentalError("Kill")
}

// InjectContext injects contextual information into a running session.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) InjectContext(ctx context.Context, sessionID, content string) error {
	return newExperimentalError("InjectContext")
}

// InjectFile injects a file into a running session.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) InjectFile(ctx context.Context, sessionID, path string) error {
	return newExperimentalError("InjectFile")
}

// GetStatus retrieves the current status of a session.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) GetStatus(ctx context.Context, sessionID string) (*ports.SessionStatus, error) {
	return nil, newExperimentalError("GetStatus")
}

// GetTokenUsage retrieves token usage statistics.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) GetTokenUsage(ctx context.Context, sessionID string) (*session.TokenUsage, error) {
	return nil, newExperimentalError("GetTokenUsage")
}

// SetModel changes the model for a session.
// Returns ExperimentalError as this backend is not yet implemented.
func (b *Backend) SetModel(ctx context.Context, model string) error {
	return newExperimentalError("SetModel")
}

// GetSupportedModels returns a list of models supported by OpenCode.
// Returns an empty list as the backend is experimental.
func (b *Backend) GetSupportedModels(ctx context.Context) ([]string, error) {
	// Return empty list with no error to allow backend enumeration
	return []string{}, nil
}

// SupportsModelControl indicates whether the backend supports changing models.
// Returns false as the backend is not yet implemented.
func (b *Backend) SupportsModelControl() bool {
	return false
}
