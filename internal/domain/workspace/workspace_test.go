package workspace

import (
	"testing"
)

func TestWorkspaceIsGitWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workspace *Workspace
		expected  bool
	}{
		{
			name: "Worktree type",
			workspace: &Workspace{
				Type: TypeWorktree,
			},
			expected: true,
		},
		{
			name: "Directory with git branch",
			workspace: &Workspace{
				Type:       TypeDirectory,
				GitBranch:  "main",
				ParentRepo: "/path/to/repo",
			},
			expected: true,
		},
		{
			name: "Regular directory",
			workspace: &Workspace{
				Type: TypeDirectory,
			},
			expected: false,
		},
		{
			name: "Container",
			workspace: &Workspace{
				Type: TypeContainer,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.workspace.IsGitWorkspace(); got != tt.expected {
				t.Errorf("IsGitWorkspace() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCreateOptions(t *testing.T) {
	opts := CreateOptions{
		Name:        "my-workspace",
		Path:        "/path/to/workspace",
		Type:        TypeDirectory,
		GitWorktree: false,
		Description: "Test workspace",
	}

	if opts.Name != "my-workspace" {
		t.Errorf("Expected Name 'my-workspace', got '%s'", opts.Name)
	}

	if opts.Type != TypeDirectory {
		t.Errorf("Expected Type 'directory', got '%s'", opts.Type)
	}

	if opts.GitWorktree {
		t.Error("Expected GitWorktree to be false")
	}
}

func TestFilter(t *testing.T) {
	filter := Filter{
		Type:      TypeWorktree,
		Status:    []Status{StatusActive, StatusClean},
		MachineID: "machine-123",
	}

	if filter.Type != TypeWorktree {
		t.Errorf("Expected Type 'worktree', got '%s'", filter.Type)
	}

	if len(filter.Status) != 2 {
		t.Errorf("Expected 2 status filters, got %d", len(filter.Status))
	}

	if filter.MachineID != "machine-123" {
		t.Errorf("Expected MachineID 'machine-123', got '%s'", filter.MachineID)
	}
}

func TestSpawnOptions(t *testing.T) {
	opts := SpawnOptions{
		WorkspaceID: "ws-123",
		Terminal:    "iterm2",
		Command:     "vim .",
		Background:  true,
	}

	if opts.WorkspaceID != "ws-123" {
		t.Errorf("Expected WorkspaceID 'ws-123', got '%s'", opts.WorkspaceID)
	}

	if opts.Terminal != "iterm2" {
		t.Errorf("Expected Terminal 'iterm2', got '%s'", opts.Terminal)
	}

	if !opts.Background {
		t.Error("Expected Background to be true")
	}
}

func TestWorkspaceTypes(t *testing.T) {
	types := []Type{
		TypeDirectory,
		TypeWorktree,
		TypeContainer,
	}

	expected := []string{"directory", "worktree", "container"}

	for i, typ := range types {
		if string(typ) != expected[i] {
			t.Errorf("Expected type '%s', got '%s'", expected[i], string(typ))
		}
	}
}

func TestWorkspaceStatus(t *testing.T) {
	statuses := []Status{
		StatusActive,
		StatusInactive,
		StatusDirty,
		StatusClean,
	}

	expected := []string{"active", "inactive", "dirty", "clean"}

	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("Expected status '%s', got '%s'", expected[i], string(status))
		}
	}
}
