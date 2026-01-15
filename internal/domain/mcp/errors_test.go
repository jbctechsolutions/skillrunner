package mcp

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrors_AreDistinct(t *testing.T) {
	allErrors := []error{
		ErrServerNotFound,
		ErrServerNotRunning,
		ErrServerAlreadyRunning,
		ErrServerStartFailed,
		ErrServerTimeout,
		ErrToolNotFound,
		ErrInvalidToolName,
		ErrToolExecutionFailed,
		ErrInitializeFailed,
		ErrInvalidResponse,
		ErrProtocolError,
		ErrConfigNotFound,
		ErrInvalidConfig,
	}

	// Ensure all errors are distinct
	for i, err1 := range allErrors {
		for j, err2 := range allErrors {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("errors should be distinct: %v and %v", err1, err2)
			}
		}
	}
}

func TestErrors_CanBeWrapped(t *testing.T) {
	tests := []struct {
		name   string
		base   error
		wrap   string
		target error
	}{
		{"server not found", ErrServerNotFound, "server 'linear' not found", ErrServerNotFound},
		{"server not running", ErrServerNotRunning, "cannot call tool", ErrServerNotRunning},
		{"tool not found", ErrToolNotFound, "tool 'create_issue' not found", ErrToolNotFound},
		{"invalid tool name", ErrInvalidToolName, "bad format", ErrInvalidToolName},
		{"config not found", ErrConfigNotFound, "~/.claude/mcp.json", ErrConfigNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("%s: %w", tt.wrap, tt.base)

			if !errors.Is(wrapped, tt.target) {
				t.Errorf("wrapped error should match target: got %v, want %v", wrapped, tt.target)
			}
		})
	}
}

func TestIs(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", ErrServerNotFound)

	if !Is(wrapped, ErrServerNotFound) {
		t.Error("Is should return true for wrapped error")
	}

	if Is(wrapped, ErrToolNotFound) {
		t.Error("Is should return false for different error")
	}

	if Is(nil, ErrServerNotFound) {
		t.Error("Is should return false for nil error")
	}
}

func TestErrors_HaveMessages(t *testing.T) {
	tests := []struct {
		err     error
		wantMsg string
	}{
		{ErrServerNotFound, "mcp server not found"},
		{ErrServerNotRunning, "mcp server not running"},
		{ErrServerAlreadyRunning, "mcp server already running"},
		{ErrServerStartFailed, "failed to start mcp server"},
		{ErrServerTimeout, "mcp server timeout"},
		{ErrToolNotFound, "tool not found"},
		{ErrInvalidToolName, "invalid tool name format"},
		{ErrToolExecutionFailed, "tool execution failed"},
		{ErrInitializeFailed, "mcp initialization failed"},
		{ErrInvalidResponse, "invalid mcp response"},
		{ErrProtocolError, "mcp protocol error"},
		{ErrConfigNotFound, "mcp config not found"},
		{ErrInvalidConfig, "invalid mcp config"},
	}

	for _, tt := range tests {
		t.Run(tt.wantMsg, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("error message = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}
