package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath failed: %v", err)
	}

	if path == "" {
		t.Error("config path should not be empty")
	}

	// Check that directory was created
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("config directory should exist: %s", dir)
	}
}

func TestNewManager(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if manager.configPath != configPath {
		t.Errorf("expected config path %s, got %s", configPath, manager.configPath)
	}
}

func TestSetAndGetAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Set API key
	if err := manager.SetAPIKey("anthropic", "test-key-123"); err != nil {
		t.Fatalf("SetAPIKey failed: %v", err)
	}

	// Get API key
	key := manager.GetAPIKey("anthropic")
	if key != "test-key-123" {
		t.Errorf("expected key 'test-key-123', got '%s'", key)
	}

	// Verify it was saved
	manager2, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	key2 := manager2.GetAPIKey("anthropic")
	if key2 != "test-key-123" {
		t.Errorf("expected key 'test-key-123' after reload, got '%s'", key2)
	}
}

func TestGetEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Set API key in config
	manager.SetAPIKey("anthropic", "config-key")

	// Environment variable should take precedence
	os.Setenv("ANTHROPIC_API_KEY", "env-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	env := manager.GetEnvVars()
	if env["ANTHROPIC_API_KEY"] != "env-key" {
		t.Errorf("expected env var to take precedence, got '%s'", env["ANTHROPIC_API_KEY"])
	}

	// Without env var, should use config
	os.Unsetenv("ANTHROPIC_API_KEY")
	env = manager.GetEnvVars()
	if env["ANTHROPIC_API_KEY"] != "config-key" {
		t.Errorf("expected config key, got '%s'", env["ANTHROPIC_API_KEY"])
	}
}
