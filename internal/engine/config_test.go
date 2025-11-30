package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/models"
)

func TestResolveModelPolicyPrecedence(t *testing.T) {
	policy, err := ResolveModelPolicy("cost_optimized")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if policy != models.ResolvePolicyCostOptimized {
		t.Fatalf("expected cost_optimized, got %s", policy)
	}

	t.Setenv("SKILLRUNNER_MODEL_POLICY", "performance_first")
	policy, err = ResolveModelPolicy("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if policy != models.ResolvePolicyPerformanceFirst {
		t.Fatalf("expected performance_first, got %s", policy)
	}
}

func TestResolveModelPolicyConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	content, err := json.Marshal(runnerConfig{ModelPolicy: "local_first"})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("SKILLRUNNER_CONFIG", configPath)
	policy, err := ResolveModelPolicy("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if policy != models.ResolvePolicyLocalFirst {
		t.Fatalf("expected local_first, got %s", policy)
	}
}

func TestResolveModelPolicyInvalid(t *testing.T) {
	t.Setenv("SKILLRUNNER_MODEL_POLICY", "invalid")
	if _, err := ResolveModelPolicy(""); err == nil {
		t.Fatal("expected error for invalid policy")
	}
}

func TestResolveModelPolicyDefault(t *testing.T) {
	// Clear all environment variables
	t.Setenv("SKILLRUNNER_MODEL_POLICY", "")
	t.Setenv("SKILLRUNNER_CONFIG", "")

	policy, err := ResolveModelPolicy("")
	if err != nil {
		t.Fatalf("expected no error for default policy, got %v", err)
	}
	if policy != models.ResolvePolicyAuto {
		t.Fatalf("expected auto policy by default, got %s", policy)
	}
}

func TestResolveModelPolicyConfigFileInvalid(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Write invalid JSON
	if err := os.WriteFile(configPath, []byte("invalid json"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("SKILLRUNNER_CONFIG", configPath)
	policy, err := ResolveModelPolicy("")

	// Should return error for invalid JSON
	if err == nil {
		t.Fatal("expected error for invalid JSON config")
	}

	// Should still return default policy
	if policy != models.ResolvePolicyAuto {
		t.Fatalf("expected auto policy, got %s", policy)
	}
}

func TestResolveModelPolicyConfigFileMissing(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nonexistent.json")

	t.Setenv("SKILLRUNNER_CONFIG", configPath)
	policy, err := ResolveModelPolicy("")

	// Missing file should not error, just use default
	if err != nil {
		t.Fatalf("expected no error for missing config, got %v", err)
	}
	if policy != models.ResolvePolicyAuto {
		t.Fatalf("expected auto policy, got %s", policy)
	}
}

func TestLoadPolicyFromConfigHomeDir(t *testing.T) {
	// Test that it doesn't crash when HOME is not set
	// This just ensures the function handles the error gracefully
	t.Setenv("SKILLRUNNER_CONFIG", "")

	policy, err := loadPolicyFromConfig()
	if err != nil {
		t.Fatalf("loadPolicyFromConfig should not error on missing home, got %v", err)
	}
	if policy != "" {
		t.Fatalf("expected empty policy for missing config, got %s", policy)
	}
}
