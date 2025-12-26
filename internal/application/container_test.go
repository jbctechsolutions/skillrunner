// Package application provides application-level services and dependency injection.
package application

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
)

func TestNewContainer(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "skillrunner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set HOME to temp dir so the database is created there
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create the .skillrunner directory
	skillrunnerDir := filepath.Join(tmpDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatalf("Failed to create .skillrunner directory: %v", err)
	}

	cfg := config.NewDefaultConfig()

	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer failed: %v", err)
	}
	defer container.Close()

	// Verify container is properly initialized
	if container.Config() == nil {
		t.Error("Config should not be nil")
	}
	if container.DB() == nil {
		t.Error("DB should not be nil")
	}
	if container.SessionRepository() == nil {
		t.Error("SessionRepository should not be nil")
	}
	if container.WorkspaceRepository() == nil {
		t.Error("WorkspaceRepository should not be nil")
	}
	if container.CheckpointRepository() == nil {
		t.Error("CheckpointRepository should not be nil")
	}
	if container.ContextItemRepository() == nil {
		t.Error("ContextItemRepository should not be nil")
	}
	if container.RulesRepository() == nil {
		t.Error("RulesRepository should not be nil")
	}
	if container.SessionManager() == nil {
		t.Error("SessionManager should not be nil")
	}
	if container.WorkflowExecutor() == nil {
		t.Error("WorkflowExecutor should not be nil")
	}
	if container.SkillLoader() == nil {
		t.Error("SkillLoader should not be nil")
	}
	if container.ProviderRegistry() == nil {
		t.Error("ProviderRegistry should not be nil")
	}
	if container.BackendRegistry() == nil {
		t.Error("BackendRegistry should not be nil")
	}
	if container.MachineID() == "" {
		t.Error("MachineID should not be empty")
	}
}

func TestNewContainer_WithNilConfig(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "skillrunner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set HOME to temp dir so the database is created there
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create the .skillrunner directory
	skillrunnerDir := filepath.Join(tmpDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatalf("Failed to create .skillrunner directory: %v", err)
	}

	// NewContainer should create a default config when nil is passed
	container, err := NewContainer(nil)
	if err != nil {
		t.Fatalf("NewContainer with nil config failed: %v", err)
	}
	defer container.Close()

	if container.Config() == nil {
		t.Error("Config should not be nil even when nil is passed")
	}
}

func TestContainer_Close(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "skillrunner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set HOME to temp dir so the database is created there
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create the .skillrunner directory
	skillrunnerDir := filepath.Join(tmpDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatalf("Failed to create .skillrunner directory: %v", err)
	}

	cfg := config.NewDefaultConfig()

	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer failed: %v", err)
	}

	// Close the container
	if err := container.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Closing again should not error
	if err := container.Close(); err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

func TestGetMachineID(t *testing.T) {
	machineID := getMachineID()
	if machineID == "" {
		t.Error("getMachineID should return a non-empty string")
	}
}

func TestSessionStorageAdapter(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "skillrunner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set HOME to temp dir so the database is created there
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create the .skillrunner directory
	skillrunnerDir := filepath.Join(tmpDir, ".skillrunner")
	if err := os.MkdirAll(skillrunnerDir, 0755); err != nil {
		t.Fatalf("Failed to create .skillrunner directory: %v", err)
	}

	cfg := config.NewDefaultConfig()

	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer failed: %v", err)
	}
	defer container.Close()

	// The session storage adapter is created internally and used by SessionManager
	// We verify it was created correctly by checking the SessionManager works
	if container.SessionManager() == nil {
		t.Error("SessionManager should be properly initialized with session storage adapter")
	}
}
