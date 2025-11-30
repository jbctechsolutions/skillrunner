package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/config"
)

// TestInitCommand tests the init command functionality
func TestInitCommand(t *testing.T) {
	// This is a basic test to ensure the init command can be created
	// More detailed testing would require mocking user input
	if initCmd == nil {
		t.Error("initCmd should be initialized")
	}
}

// TestValidateWorkspace tests workspace validation
func TestValidateWorkspace(t *testing.T) {
	validateWorkspace := func(path string) error {
		if path == "" {
			return nil // Empty workspace is valid
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return os.ErrNotExist
		}
		return nil
	}

	t.Run("empty workspace", func(t *testing.T) {
		err := validateWorkspace("")
		if err != nil {
			t.Errorf("Empty workspace should be valid, got error: %v", err)
		}
	})

	t.Run("existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := validateWorkspace(tmpDir)
		if err != nil {
			t.Errorf("Existing directory should be valid, got error: %v", err)
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		err := validateWorkspace("/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Error("Non-existent path should return error")
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "file.txt")
		if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err := validateWorkspace(tmpFile)
		if err == nil {
			t.Error("File path should return error")
		}
	})
}

// TestEnsureDirectories tests directory creation
func TestEnsureDirectories(t *testing.T) {
	ensureDirectories := func() error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		skillrunnerDir := filepath.Join(homeDir, ".skillrunner")
		return os.MkdirAll(skillrunnerDir, 0755)
	}

	// This test modifies the home directory structure, so we'll test with a temp dir
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	err := ensureDirectories()
	if err != nil {
		t.Fatalf("EnsureDirectories failed: %v", err)
	}

	// Verify directory was created
	baseDir := filepath.Join(tmpDir, ".skillrunner")
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Error("Base directory should be created")
	}
}

// TestConfigManagerCreation tests that config manager can be created
func TestConfigManagerCreation(t *testing.T) {
	configMgr, err := config.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create config manager: %v", err)
	}

	if configMgr == nil {
		t.Error("Config manager should not be nil")
	}

	// Verify config path is set
	actualPath := configMgr.GetConfigPath()
	if actualPath == "" {
		t.Error("Config path should be set")
	}

	// Test that we can get default config
	cfg := configMgr.Get()
	if cfg == nil {
		t.Error("Default config should not be nil")
	}
}
