// Package e2e provides end-to-end integration tests for skillrunner.
package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/skills"
	"github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands"
)

// executeCommand executes a cobra command with the given args and captures output.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// TestE2E_CLICommands tests all CLI commands execute without error.
func TestE2E_CLICommands(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		// Version command
		{"version", []string{"version"}, false},
		{"version short", []string{"version", "--short"}, false},
		{"version json", []string{"version", "-o", "json"}, false},

		// List command
		{"list", []string{"list"}, false},
		{"list alias ls", []string{"ls"}, false},
		{"list json", []string{"list", "-f", "json"}, false},
		{"list table", []string{"list", "--format", "table"}, false},

		// Status command
		{"status", []string{"status"}, false},
		{"status detailed", []string{"status", "--detailed"}, false},
		{"status json", []string{"status", "-o", "json"}, false},

		// Run command
		// Note: These expect errors because the skills don't exist in test environment
		{"run valid syntax", []string{"run", "test-skill", "test request"}, true},               // skill not found
		{"run with profile", []string{"run", "skill", "request", "--profile", "premium"}, true}, // skill not found
		{"run with stream", []string{"run", "skill", "request", "--stream"}, true},              // skill not found
		{"run missing args", []string{"run"}, true},
		{"run invalid profile", []string{"run", "skill", "request", "--profile", "invalid"}, true},

		// Ask command (expects errors without providers configured)
		{"ask valid no providers", []string{"ask", "test-skill", "what is this?"}, true},
		{"ask with model no providers", []string{"ask", "test-skill", "question", "--model", "gpt-4"}, true},
		{"ask missing args", []string{"ask"}, true},

		// Metrics command
		{"metrics", []string{"metrics"}, false},
		{"metrics json", []string{"metrics", "-o", "json"}, false},
		{"metrics with since", []string{"metrics", "--since", "7d"}, false},

		// Init command (doesn't actually create files in test)
		{"init help", []string{"init", "--help"}, false},

		// Import command
		{"import help", []string{"import", "--help"}, false},

		// Help
		{"help", []string{"--help"}, false},
		{"help run", []string{"run", "--help"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := commands.NewRootCmd()
			_, err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("command %v: error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

// TestE2E_SkillLoading tests skill loading from the skills directory.
func TestE2E_SkillLoading(t *testing.T) {
	// Find the skills directory relative to the project root
	projectRoot := findProjectRoot(t)
	skillsDir := filepath.Join(projectRoot, "skills")

	// Check skills directory exists
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		t.Skipf("skills directory not found at %s", skillsDir)
	}

	// Load all skills from directory
	loader := skills.NewLoader()
	loadedSkills, err := loader.LoadSkillsDir(skillsDir)
	if err != nil {
		t.Fatalf("failed to load skills: %v", err)
	}

	// Verify we have the expected demo skills (by ID)
	expectedSkillIDs := []string{"code-review", "test-gen", "doc-gen"}
	for _, expectedID := range expectedSkillIDs {
		if _, ok := loadedSkills[expectedID]; !ok {
			t.Errorf("expected skill ID %q not found in loaded skills", expectedID)
		}
	}

	// Verify each skill has valid structure
	for _, s := range loadedSkills {
		if s.Name() == "" {
			t.Error("skill has empty name")
		}
		if s.Description() == "" {
			t.Errorf("skill %q has empty description", s.Name())
		}
		if len(s.Phases()) == 0 {
			t.Errorf("skill %q has no phases", s.Name())
		}

		// Verify each phase has required fields
		for i, phase := range s.Phases() {
			if phase.Name == "" {
				t.Errorf("skill %q phase %d has empty name", s.Name(), i)
			}
		}
	}
}

// TestE2E_IndividualSkillLoading tests loading each demo skill individually.
func TestE2E_IndividualSkillLoading(t *testing.T) {
	projectRoot := findProjectRoot(t)
	skillsDir := filepath.Join(projectRoot, "skills")

	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		t.Skipf("skills directory not found at %s", skillsDir)
	}

	tests := []struct {
		testName   string
		file       string
		wantID     string
		wantName   string
		wantPhases int
	}{
		{"code-review", "code-review.yaml", "code-review", "Code Review", 3},
		{"test-gen", "test-gen.yaml", "test-gen", "Test Generation", 3},
		{"doc-gen", "doc-gen.yaml", "doc-gen", "Documentation Generator", 2},
	}

	loader := skills.NewLoader()
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			skillPath := filepath.Join(skillsDir, tt.file)
			if _, err := os.Stat(skillPath); os.IsNotExist(err) {
				t.Skipf("skill file not found: %s", skillPath)
			}

			skill, err := loader.LoadSkill(skillPath)
			if err != nil {
				t.Fatalf("failed to load skill %s: %v", tt.file, err)
			}

			if skill.ID() != tt.wantID {
				t.Errorf("expected skill ID %q, got %q", tt.wantID, skill.ID())
			}

			if skill.Name() != tt.wantName {
				t.Errorf("expected skill name %q, got %q", tt.wantName, skill.Name())
			}

			if len(skill.Phases()) != tt.wantPhases {
				t.Errorf("expected %d phases, got %d", tt.wantPhases, len(skill.Phases()))
			}
		})
	}
}

// TestE2E_SubcommandStructure verifies all expected subcommands exist.
func TestE2E_SubcommandStructure(t *testing.T) {
	rootCmd := commands.NewRootCmd()

	expectedCmds := []string{
		"version",
		"list",
		"run",
		"status",
		"ask",
		"import",
		"init",
		"metrics",
	}

	// Get all subcommand names
	subcmds := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		subcmds[cmd.Name()] = true
	}

	for _, expected := range expectedCmds {
		if !subcmds[expected] {
			t.Errorf("expected subcommand %q not found", expected)
		}
	}
}

// TestE2E_GlobalFlags verifies global flags are available.
func TestE2E_GlobalFlags(t *testing.T) {
	rootCmd := commands.NewRootCmd()

	expectedFlags := []string{"config", "output", "verbose"}

	for _, flag := range expectedFlags {
		if rootCmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("expected global flag %q not found", flag)
		}
	}
}

// TestE2E_CommandAliases tests command aliases work.
func TestE2E_CommandAliases(t *testing.T) {
	tests := []struct {
		alias   string
		wantErr bool
	}{
		{"ls", false}, // alias for list
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			cmd := commands.NewRootCmd()
			_, err := executeCommand(cmd, tt.alias)
			if (err != nil) != tt.wantErr {
				t.Errorf("alias %q: error = %v, wantErr %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}

// TestE2E_RunCommandProfiles tests all valid routing profiles.
// Note: These tests verify that profile validation works correctly.
// The commands will fail with "skill not found" because test skills don't exist,
// but invalid profiles should fail with a different error message.
func TestE2E_RunCommandProfiles(t *testing.T) {
	validProfiles := []string{"cheap", "balanced", "premium"}

	for _, profile := range validProfiles {
		t.Run(profile, func(t *testing.T) {
			cmd := commands.NewRootCmd()
			_, err := executeCommand(cmd, "run", "test-skill", "request", "--profile", profile)
			// The command will fail because the skill doesn't exist, but the error
			// should NOT be about an invalid profile - it should be about the missing skill
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "invalid profile") {
					t.Errorf("profile %q should be valid, got profile validation error: %v", profile, err)
				}
				// "skill not found" errors are expected since test skills don't exist
			}
		})
	}
}

// TestE2E_IntegrationFlow tests a realistic user workflow.
// This tests the core commands work correctly, even without actual skills loaded.
func TestE2E_IntegrationFlow(t *testing.T) {
	// Simulate a user checking system and listing skills
	// Note: Running skills requires actual skill definitions which aren't available in tests

	// Step 1: Check status
	cmd := commands.NewRootCmd()
	_, err := executeCommand(cmd, "status")
	if err != nil {
		t.Fatalf("status check failed: %v", err)
	}

	// Step 2: List available skills (will show empty list in test environment)
	cmd = commands.NewRootCmd()
	_, err = executeCommand(cmd, "list")
	if err != nil {
		t.Fatalf("list skills failed: %v", err)
	}

	// Step 3: Attempt to run a skill - expect failure due to missing skill
	// This validates the error handling path works correctly
	// Use --force to ensure clean execution regardless of any leftover checkpoints
	cmd = commands.NewRootCmd()
	_, err = executeCommand(cmd, "run", "code-review", "Review this code for issues", "--force")
	if err == nil {
		t.Log("run command succeeded (skill might be available in test environment)")
	} else if !strings.Contains(err.Error(), "skill not found") && !strings.Contains(err.Error(), "not found") {
		// Only fail if the error is unexpected (not about missing skill)
		t.Fatalf("run skill failed with unexpected error: %v", err)
	}

	// Step 4: Check metrics
	cmd = commands.NewRootCmd()
	_, err = executeCommand(cmd, "metrics")
	if err != nil {
		t.Fatalf("metrics check failed: %v", err)
	}
}

// findProjectRoot finds the project root directory for tests.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current working directory and go up until we find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			// Try relative path from test file location
			return filepath.Join("..", "..")
		}
		dir = parent
	}
}

// TestE2E_ErrorMessages tests that error messages are helpful.
func TestE2E_ErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantErr       bool
		errorContains string
	}{
		{
			name:          "run without skill",
			args:          []string{"run"},
			wantErr:       true,
			errorContains: "accepts",
		},
		{
			name:          "run invalid profile",
			args:          []string{"run", "skill", "req", "-p", "superfast"},
			wantErr:       true,
			errorContains: "invalid profile",
		},
		{
			name:          "ask without args",
			args:          []string{"ask"},
			wantErr:       true,
			errorContains: "accepts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := commands.NewRootCmd()
			_, err := executeCommand(cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if tt.errorContains != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errorContains)
				}
			}
		})
	}
}
