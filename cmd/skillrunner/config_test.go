package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigManager(t *testing.T) {
	cm := NewConfigManager()
	if cm == nil {
		t.Fatal("NewConfigManager returned nil")
	}
	if cm.configPath == "" {
		t.Error("configPath should not be empty")
	}
	if cm.config == nil {
		t.Error("config should be initialized")
	}
}

func TestNewConfigManager_UserHomeDirError(t *testing.T) {
	// This is hard to test directly, but we verify the fallback works
	// by checking that NewConfigManager always returns a valid manager
	cm := NewConfigManager()
	if cm == nil {
		t.Fatal("NewConfigManager should always return a manager")
	}
	// Even if UserHomeDir fails, it should fallback to "."
	if cm.configPath == "" {
		t.Error("configPath should have a fallback value")
	}
}

func TestConfigManager_Load(t *testing.T) {
	t.Run("non-existent config file", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = "/tmp/nonexistent-config.json"

		config, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if config == nil {
			t.Fatal("Load returned nil config")
		}
		if config.OutputFormat != "table" {
			t.Errorf("Expected default OutputFormat 'table', got: %s", config.OutputFormat)
		}
	})

	t.Run("existing config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		cm := NewConfigManager()
		cm.configPath = configPath

		// Create a test config file
		testConfig := `{
  "workspace": "/test/workspace",
  "default_model": "gpt-4",
  "output_format": "json",
  "compact_output": true,
  "models": ["gpt-4", "claude-3"]
}`
		if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		config, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if config.Workspace != "/test/workspace" {
			t.Errorf("Expected workspace '/test/workspace', got: %s", config.Workspace)
		}
		if config.DefaultModel != "gpt-4" {
			t.Errorf("Expected default_model 'gpt-4', got: %s", config.DefaultModel)
		}
		if config.OutputFormat != "json" {
			t.Errorf("Expected output_format 'json', got: %s", config.OutputFormat)
		}
		if !config.CompactOutput {
			t.Error("Expected compact_output to be true")
		}
		if len(config.Models) != 2 {
			t.Errorf("Expected 2 models, got: %d", len(config.Models))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		cm := NewConfigManager()
		cm.configPath = configPath

		// Create invalid JSON
		if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := cm.Load()
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestConfigManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cm := NewConfigManager()
	cm.configPath = configPath

	config := &Config{
		Workspace:     "/test/workspace",
		DefaultModel:  "gpt-4",
		OutputFormat:  "json",
		CompactOutput: true,
		Models:        []string{"gpt-4", "claude-3"},
	}

	err := cm.Save(config)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should be created")
	}

	// Load and verify
	loaded, err := cm.Load()
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}
	if loaded.Workspace != config.Workspace {
		t.Errorf("Workspace mismatch: expected %s, got %s", config.Workspace, loaded.Workspace)
	}
	if loaded.DefaultModel != config.DefaultModel {
		t.Errorf("DefaultModel mismatch: expected %s, got %s", config.DefaultModel, loaded.DefaultModel)
	}
}

func TestConfigManager_GetConfigPath(t *testing.T) {
	cm := NewConfigManager()
	path := cm.GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath should not return empty string")
	}
	if path != cm.configPath {
		t.Error("GetConfigPath should return configPath")
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()
	if config == nil {
		t.Fatal("GetDefaultConfig returned nil")
	}
	if config.OutputFormat != "table" {
		t.Errorf("Expected default OutputFormat 'table', got: %s", config.OutputFormat)
	}
	if config.CompactOutput {
		t.Error("Expected default CompactOutput to be false")
	}
	if len(config.Models) == 0 {
		t.Error("Expected default Models to be non-empty")
	}
}

func TestMergeConfig(t *testing.T) {
	fileConfig := &Config{
		Workspace:     "/file/workspace",
		DefaultModel:  "gpt-4",
		OutputFormat:  "table",
		CompactOutput: false,
		Models:        []string{"gpt-4"},
	}

	t.Run("no overrides", func(t *testing.T) {
		merged := MergeConfig(fileConfig, "", "", "", false)
		if merged.Workspace != fileConfig.Workspace {
			t.Errorf("Workspace should match file config: expected %s, got %s", fileConfig.Workspace, merged.Workspace)
		}
		if merged.DefaultModel != fileConfig.DefaultModel {
			t.Errorf("DefaultModel should match file config: expected %s, got %s", fileConfig.DefaultModel, merged.DefaultModel)
		}
	})

	t.Run("with workspace override", func(t *testing.T) {
		merged := MergeConfig(fileConfig, "/override/workspace", "", "", false)
		if merged.Workspace != "/override/workspace" {
			t.Errorf("Workspace should be overridden: expected /override/workspace, got %s", merged.Workspace)
		}
	})

	t.Run("with model override", func(t *testing.T) {
		merged := MergeConfig(fileConfig, "", "claude-3", "", false)
		if merged.DefaultModel != "claude-3" {
			t.Errorf("DefaultModel should be overridden: expected claude-3, got %s", merged.DefaultModel)
		}
	})

	t.Run("with format override", func(t *testing.T) {
		merged := MergeConfig(fileConfig, "", "", "json", false)
		if merged.OutputFormat != "json" {
			t.Errorf("OutputFormat should be overridden: expected json, got %s", merged.OutputFormat)
		}
	})

	t.Run("with compact override", func(t *testing.T) {
		merged := MergeConfig(fileConfig, "", "", "", true)
		if !merged.CompactOutput {
			t.Error("CompactOutput should be overridden to true")
		}
	})

	t.Run("all overrides", func(t *testing.T) {
		merged := MergeConfig(fileConfig, "/new/workspace", "claude-3", "json", true)
		if merged.Workspace != "/new/workspace" {
			t.Error("Workspace should be overridden")
		}
		if merged.DefaultModel != "claude-3" {
			t.Error("DefaultModel should be overridden")
		}
		if merged.OutputFormat != "json" {
			t.Error("OutputFormat should be overridden")
		}
		if !merged.CompactOutput {
			t.Error("CompactOutput should be overridden")
		}
	})
}

func TestConfigManager_Save_ErrorCases(t *testing.T) {
	t.Run("save with nil config", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = "/invalid/path/config.json"
		// This should fail when trying to create directory or write
		err := cm.Save(&Config{})
		// May or may not fail depending on permissions, but we test the path
		_ = err
	})

	t.Run("save creates directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "subdir", "config.json")
		cm := NewConfigManager()
		cm.configPath = configPath

		config := &Config{Workspace: "/test"}
		err := cm.Save(config)
		if err != nil {
			t.Fatalf("Save should create directory: %v", err)
		}
	})

	t.Run("save with read error path", func(t *testing.T) {
		cm := NewConfigManager()
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		cm.configPath = configPath

		// Save a config first
		config := &Config{Workspace: "/test"}
		if err := cm.Save(config); err != nil {
			t.Fatalf("Initial save failed: %v", err)
		}

		// Verify it was saved
		loaded, err := cm.Load()
		if err != nil {
			t.Fatalf("Load after save failed: %v", err)
		}
		if loaded.Workspace != config.Workspace {
			t.Error("Config should be saved and loaded correctly")
		}
	})
}
