// Package context provides domain entities for workspace and context management.
package context

import (
	"testing"
)

func TestNewWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wname     string
		repoPath  string
		wantErr   bool
		errString string
	}{
		{
			name:     "valid workspace",
			id:       "test-workspace",
			wname:    "Test Workspace",
			repoPath: "/path/to/repo",
			wantErr:  false,
		},
		{
			name:     "missing id",
			id:       "",
			wname:    "Test",
			repoPath: "/path",
			wantErr:  true,
		},
		{
			name:     "missing name",
			id:       "test",
			wname:    "",
			repoPath: "/path",
			wantErr:  true,
		},
		{
			name:     "missing repo path",
			id:       "test",
			wname:    "Test",
			repoPath: "",
			wantErr:  true,
		},
		{
			name:     "whitespace trimmed",
			id:       "  test  ",
			wname:    "  Test  ",
			repoPath: "  /path  ",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, err := NewWorkspace(tt.id, tt.wname, tt.repoPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewWorkspace() expected error but got none")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("NewWorkspace() unexpected error = %v", err)
				return
			}

			if workspace == nil {
				t.Error("NewWorkspace() returned nil workspace")
				return
			}

			// Verify workspace properties
			if workspace.ID() == "" {
				t.Error("workspace ID is empty")
			}
			if workspace.Name() == "" {
				t.Error("workspace name is empty")
			}
			if workspace.RepoPath() == "" {
				t.Error("workspace repo path is empty")
			}
			if workspace.Status() != WorkspaceStatusActive {
				t.Errorf("workspace status = %v, want %v", workspace.Status(), WorkspaceStatusActive)
			}
		})
	}
}

func TestWorkspace_SetFocus(t *testing.T) {
	workspace, err := NewWorkspace("test", "Test", "/path")
	if err != nil {
		t.Fatalf("NewWorkspace() error = %v", err)
	}

	// Set focus
	workspace.SetFocus("ISSUE-123")

	if workspace.Focus() != "ISSUE-123" {
		t.Errorf("Focus() = %v, want %v", workspace.Focus(), "ISSUE-123")
	}

	// Clear focus
	workspace.SetFocus("")

	if workspace.Focus() != "" {
		t.Errorf("Focus() = %v, want empty", workspace.Focus())
	}
}

func TestWorkspace_StatusTransitions(t *testing.T) {
	workspace, err := NewWorkspace("test", "Test", "/path")
	if err != nil {
		t.Fatalf("NewWorkspace() error = %v", err)
	}

	// Initial status should be active
	if workspace.Status() != WorkspaceStatusActive {
		t.Errorf("initial status = %v, want %v", workspace.Status(), WorkspaceStatusActive)
	}

	// Set to idle
	workspace.SetIdle()
	if workspace.Status() != WorkspaceStatusIdle {
		t.Errorf("status after SetIdle() = %v, want %v", workspace.Status(), WorkspaceStatusIdle)
	}

	// Activate
	workspace.Activate()
	if workspace.Status() != WorkspaceStatusActive {
		t.Errorf("status after Activate() = %v, want %v", workspace.Status(), WorkspaceStatusActive)
	}

	// Archive
	workspace.Archive()
	if workspace.Status() != WorkspaceStatusArchived {
		t.Errorf("status after Archive() = %v, want %v", workspace.Status(), WorkspaceStatusArchived)
	}
}

func TestWorkspace_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *Workspace
		wantErr bool
	}{
		{
			name: "valid workspace",
			setup: func() *Workspace {
				w, _ := NewWorkspace("test", "Test", "/path")
				return w
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := tt.setup()
			err := workspace.Validate()

			if tt.wantErr && err == nil {
				t.Error("Validate() expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}
