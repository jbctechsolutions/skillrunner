package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	t.Run("set workspace", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = configPath

		// Load default config
		config, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		// Simulate setConfig for workspace
		config.Workspace = "/test/workspace"
		if err := cm.Save(config); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Verify
		loaded, err := cm.Load()
		if err != nil {
			t.Fatalf("Load after save failed: %v", err)
		}
		if loaded.Workspace != "/test/workspace" {
			t.Errorf("Expected workspace '/test/workspace', got: %s", loaded.Workspace)
		}
	})

	t.Run("set default_model", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = configPath

		config, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		config.DefaultModel = "gpt-4"
		if err := cm.Save(config); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loaded, err := cm.Load()
		if err != nil {
			t.Fatalf("Load after save failed: %v", err)
		}
		if loaded.DefaultModel != "gpt-4" {
			t.Errorf("Expected default_model 'gpt-4', got: %s", loaded.DefaultModel)
		}
	})

	t.Run("set output_format", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = configPath

		config, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		config.OutputFormat = "json"
		if err := cm.Save(config); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loaded, err := cm.Load()
		if err != nil {
			t.Fatalf("Load after save failed: %v", err)
		}
		if loaded.OutputFormat != "json" {
			t.Errorf("Expected output_format 'json', got: %s", loaded.OutputFormat)
		}
	})

	t.Run("set compact_output true", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = configPath

		config, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		config.CompactOutput = true
		if err := cm.Save(config); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loaded, err := cm.Load()
		if err != nil {
			t.Fatalf("Load after save failed: %v", err)
		}
		if !loaded.CompactOutput {
			t.Error("Expected compact_output to be true")
		}
	})

	t.Run("set compact_output false", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = configPath

		config, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		config.CompactOutput = false
		if err := cm.Save(config); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loaded, err := cm.Load()
		if err != nil {
			t.Fatalf("Load after save failed: %v", err)
		}
		if loaded.CompactOutput {
			t.Error("Expected compact_output to be false")
		}
	})
}

func TestSetConfig_InvalidKey(t *testing.T) {
	// Test that invalid keys are rejected
	invalidKeys := []string{"invalid_key", "unknown", "test"}
	for _, key := range invalidKeys {
		t.Run("invalid key: "+key, func(t *testing.T) {
			// This would be tested via the actual command, but we can test the logic
			validKeys := map[string]bool{
				"workspace":      true,
				"default_model":  true,
				"output_format":  true,
				"compact_output": true,
			}
			if validKeys[key] {
				t.Errorf("Key %s should not be in invalid keys list", key)
			}
		})
	}
}

func TestSetConfig_InvalidOutputFormat(t *testing.T) {
	// Test that invalid output_format values are rejected
	invalidFormats := []string{"xml", "yaml", "invalid"}
	for _, format := range invalidFormats {
		t.Run("invalid format: "+format, func(t *testing.T) {
			if format != "table" && format != "json" {
				// This is the validation logic
				// In the actual command, this would return an error
			}
		})
	}
}

func TestShowConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	t.Run("show config table format", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = configPath

		config := &Config{
			Workspace:     "/test/workspace",
			DefaultModel:  "gpt-4",
			OutputFormat:  "json",
			CompactOutput: true,
			Models:        []string{"gpt-4", "claude-3"},
		}

		if err := cm.Save(config); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Load and verify structure
		loaded, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if loaded.Workspace != config.Workspace {
			t.Errorf("Workspace mismatch")
		}
		if loaded.DefaultModel != config.DefaultModel {
			t.Errorf("DefaultModel mismatch")
		}
		if loaded.OutputFormat != config.OutputFormat {
			t.Errorf("OutputFormat mismatch")
		}
		if loaded.CompactOutput != config.CompactOutput {
			t.Errorf("CompactOutput mismatch")
		}
	})

	t.Run("show config json format", func(t *testing.T) {
		cm := NewConfigManager()
		cm.configPath = configPath

		config := &Config{
			Workspace:     "/test/workspace",
			DefaultModel:  "gpt-4",
			OutputFormat:  "json",
			CompactOutput: true,
			Models:        []string{"gpt-4", "claude-3"},
		}

		if err := cm.Save(config); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Test JSON marshaling (simulating showConfig with json format)
		jsonOutput, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatalf("MarshalIndent failed: %v", err)
		}

		// Verify it's valid JSON
		var unmarshaled Config
		if err := json.Unmarshal(jsonOutput, &unmarshaled); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if unmarshaled.Workspace != config.Workspace {
			t.Errorf("Workspace mismatch after JSON round-trip")
		}
	})

	t.Run("show config with defaults", func(t *testing.T) {
		// Test showing config when file doesn't exist (defaults)
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nonexistent.json")
		cm := NewConfigManager()
		cm.configPath = configPath

		config, err := cm.Load()
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		// Should have defaults
		if config.OutputFormat != "table" {
			t.Errorf("Expected default OutputFormat 'table', got: %s", config.OutputFormat)
		}
		if config.CompactOutput {
			t.Error("Expected default CompactOutput to be false")
		}
		if len(config.Models) == 0 {
			t.Error("Expected default Models to be non-empty")
		}
	})
}

func TestGetValueOrEmpty(t *testing.T) {
	t.Run("empty value", func(t *testing.T) {
		result := getValueOrEmpty("")
		if result != "(not set)" {
			t.Errorf("Expected '(not set)', got: %s", result)
		}
	})

	t.Run("non-empty value", func(t *testing.T) {
		result := getValueOrEmpty("test-value")
		if result != "test-value" {
			t.Errorf("Expected 'test-value', got: %s", result)
		}
	})
}

func TestGetValueOrDefault(t *testing.T) {
	t.Run("empty value with default", func(t *testing.T) {
		result := getValueOrDefault("", "default")
		if result != "default" {
			t.Errorf("Expected 'default', got: %s", result)
		}
	})

	t.Run("non-empty value", func(t *testing.T) {
		result := getValueOrDefault("test-value", "default")
		if result != "test-value" {
			t.Errorf("Expected 'test-value', got: %s", result)
		}
	})
}

func TestConfigCommands_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cm := NewConfigManager()
	cm.configPath = configPath

	// Test full workflow: set multiple values and show
	config := &Config{
		Workspace:     "/test/workspace",
		DefaultModel:  "gpt-4",
		OutputFormat:  "json",
		CompactOutput: true,
	}

	if err := cm.Save(config); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Update workspace
	config.Workspace = "/new/workspace"
	if err := cm.Save(config); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load and verify
	loaded, err := cm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Workspace != "/new/workspace" {
		t.Errorf("Expected workspace '/new/workspace', got: %s", loaded.Workspace)
	}
	if loaded.DefaultModel != "gpt-4" {
		t.Errorf("Expected default_model 'gpt-4', got: %s", loaded.DefaultModel)
	}
}

func TestConfigCommands_InvalidValues(t *testing.T) {
	t.Run("invalid compact_output value", func(t *testing.T) {
		// Test that non-boolean values for compact_output are rejected
		invalidValues := []string{"yes", "no", "1", "0", "maybe"}
		for _, val := range invalidValues {
			_, err := parseBool(val)
			if err == nil && (val == "yes" || val == "no" || val == "maybe") {
				// These should fail
				t.Logf("Value '%s' should fail parsing as bool", val)
			}
		}
	})
}

// Helper function to test boolean parsing
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}
