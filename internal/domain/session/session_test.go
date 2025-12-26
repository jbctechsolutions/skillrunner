package session

import (
	"testing"
	"time"
)

func TestSessionIsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"Active", StatusActive, true},
		{"Idle", StatusIdle, true},
		{"Detached", StatusDetached, true},
		{"Completed", StatusCompleted, false},
		{"Failed", StatusFailed, false},
		{"Killed", StatusKilled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{Status: tt.status}
			if got := session.IsActive(); got != tt.expected {
				t.Errorf("IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSessionDuration(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	tests := []struct {
		name    string
		session *Session
		minDur  time.Duration
		maxDur  time.Duration
	}{
		{
			name: "Running session",
			session: &Session{
				StartedAt: oneHourAgo,
				EndedAt:   nil,
			},
			minDur: 59 * time.Minute,
			maxDur: 61 * time.Minute,
		},
		{
			name: "Completed session",
			session: &Session{
				StartedAt: oneHourAgo,
				EndedAt:   &now,
			},
			minDur: 59 * time.Minute,
			maxDur: 61 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.session.Duration()
			if duration < tt.minDur || duration > tt.maxDur {
				t.Errorf("Duration() = %v, want between %v and %v", duration, tt.minDur, tt.maxDur)
			}
		})
	}
}

func TestStartOptions(t *testing.T) {
	opts := StartOptions{
		WorkspaceID: "test-workspace",
		Backend:     "aider",
		Model:       "gpt-4",
		Background:  true,
		Task:        "Add tests",
	}

	if opts.WorkspaceID != "test-workspace" {
		t.Errorf("Expected WorkspaceID 'test-workspace', got '%s'", opts.WorkspaceID)
	}

	if opts.Backend != "aider" {
		t.Errorf("Expected Backend 'aider', got '%s'", opts.Backend)
	}

	if !opts.Background {
		t.Error("Expected Background to be true")
	}
}

func TestBackendConfig(t *testing.T) {
	config := BackendConfig{
		AiderEditFormat: "diff",
		AiderAutoCommit: true,
		AiderMapTokens:  1024,
		ClaudeRulesFile: "CLAUDE.md",
		OpenCodeMode:    "chat",
		Timeout:         30 * time.Minute,
		MaxTokens:       4096,
	}

	if config.AiderEditFormat != "diff" {
		t.Errorf("Expected AiderEditFormat 'diff', got '%s'", config.AiderEditFormat)
	}

	if !config.AiderAutoCommit {
		t.Error("Expected AiderAutoCommit to be true")
	}

	if config.AiderMapTokens != 1024 {
		t.Errorf("Expected AiderMapTokens 1024, got %d", config.AiderMapTokens)
	}

	if config.Timeout != 30*time.Minute {
		t.Errorf("Expected Timeout 30m, got %v", config.Timeout)
	}
}

func TestTokenUsage(t *testing.T) {
	usage := TokenUsage{
		InputTokens:   1000,
		OutputTokens:  500,
		TotalTokens:   1500,
		EstimatedCost: 0.05,
	}

	if usage.InputTokens != 1000 {
		t.Errorf("Expected InputTokens 1000, got %d", usage.InputTokens)
	}

	if usage.TotalTokens != 1500 {
		t.Errorf("Expected TotalTokens 1500, got %d", usage.TotalTokens)
	}

	if usage.EstimatedCost != 0.05 {
		t.Errorf("Expected EstimatedCost 0.05, got %f", usage.EstimatedCost)
	}
}

func TestContext(t *testing.T) {
	ctx := Context{
		WorkingDirectory: "/path/to/workspace",
		GitBranch:        "main",
		Files:            []string{"main.go", "utils.go"},
		Documentation: []DocReference{
			{
				Type:  "file",
				Path:  "README.md",
				Title: "Project README",
			},
		},
	}

	if ctx.WorkingDirectory != "/path/to/workspace" {
		t.Errorf("Expected WorkingDirectory '/path/to/workspace', got '%s'", ctx.WorkingDirectory)
	}

	if len(ctx.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(ctx.Files))
	}

	if len(ctx.Documentation) != 1 {
		t.Errorf("Expected 1 documentation reference, got %d", len(ctx.Documentation))
	}
}

func TestFilter(t *testing.T) {
	filter := Filter{
		WorkspaceID: "test-workspace",
		Backend:     "aider",
		Status:      []Status{StatusActive, StatusIdle},
		Limit:       10,
	}

	if filter.WorkspaceID != "test-workspace" {
		t.Errorf("Expected WorkspaceID 'test-workspace', got '%s'", filter.WorkspaceID)
	}

	if len(filter.Status) != 2 {
		t.Errorf("Expected 2 status filters, got %d", len(filter.Status))
	}

	if filter.Limit != 10 {
		t.Errorf("Expected Limit 10, got %d", filter.Limit)
	}
}
