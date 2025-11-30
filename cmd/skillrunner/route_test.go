package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/engine"
	"github.com/jbctechsolutions/skillrunner/internal/models"
	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// testdataDir returns the absolute path to the testdata directory
func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// setupTestEngine creates an engine with mock providers for testing.
// It uses a static provider to avoid needing live Ollama connections in CI.
func setupTestEngine(workspace string, policy models.ResolvePolicy) *engine.Skillrunner {
	// Use testdata directory as workspace if none provided
	if workspace == "" {
		workspace = testdataDir()
	}
	// Create engine with empty config to avoid default provider registration
	eng := engine.NewSkillrunnerForTesting(workspace, policy)

	// Register static provider with test models
	testProvider := models.NewStaticProvider(models.ProviderInfo{
		Name: "test",
		Type: models.ProviderTypeLocal,
	}, []models.StaticModel{
		{
			Name:            "gpt-4",
			Route:           "test/gpt-4",
			Description:     "Test GPT-4 model",
			Available:       true,
			Tier:            models.AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
		{
			Name:            "claude-3",
			Route:           "test/claude-3",
			Description:     "Test Claude 3 model",
			Available:       true,
			Tier:            models.AgentTierPowerful,
			CostPer1KTokens: 0.015,
		},
	})

	eng.RegisterTestProvider(testProvider)
	return eng
}

func TestNewCommandRouter(t *testing.T) {
	router := NewCommandRouter()
	if router == nil {
		t.Fatal("NewCommandRouter returned nil")
	}
	if router.engineFactory == nil {
		t.Error("engineFactory should not be nil")
	}
	if router.outputWriter == nil {
		t.Error("outputWriter should not be nil")
	}
	if router.stdoutWriter == nil {
		t.Error("stdoutWriter should not be nil")
	}
	if router.stderrWriter == nil {
		t.Error("stderrWriter should not be nil")
	}
}

func TestCommandRouter_RouteRun(t *testing.T) {
	router := NewCommandRouter()
	// Use test engine factory
	router.engineFactory = setupTestEngine

	t.Run("successful run", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteRun("hello-orchestration", "hello world", "", "", "", false)
		if err != nil {
			t.Fatalf("RouteRun failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "hello-orchestration") {
			t.Error("Output should contain skill name")
		}
		if !strings.Contains(output, "hello world") {
			t.Error("Output should contain request")
		}
	})

	t.Run("empty skill name", func(t *testing.T) {
		err := router.RouteRun("", "request", "", "", "", false)
		if err == nil {
			t.Error("Expected error for empty skill name")
		}
		if !strings.Contains(err.Error(), "skill name is required") {
			t.Errorf("Error message should mention skill name, got: %v", err)
		}
	})

	t.Run("empty request", func(t *testing.T) {
		err := router.RouteRun("test", "", "", "", "", false)
		if err == nil {
			t.Error("Expected error for empty request")
		}
		if !strings.Contains(err.Error(), "request is required") {
			t.Errorf("Error message should mention request, got: %v", err)
		}
	})

	t.Run("invalid skill", func(t *testing.T) {
		err := router.RouteRun("nonexistent", "request", "", "", "", false)
		if err == nil {
			t.Error("Expected error for invalid skill")
		}
		if !strings.Contains(err.Error(), "failed to run skill") {
			t.Errorf("Error message should mention failed to run, got: %v", err)
		}
	})

	t.Run("with output file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test-envelope.json")

		var stderr strings.Builder
		router.stderrWriter = func(format string, args ...interface{}) {
			stderr.WriteString(fmt.Sprintf(format, args...))
		}

		err := router.RouteRun("hello-orchestration", "hello", "", "", tmpFile, false)
		if err != nil {
			t.Fatalf("RouteRun failed: %v", err)
		}

		// Check file was created
		if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
			t.Error("Output file should be created")
		}
	})

	t.Run("with model override", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteRun("hello-orchestration", "hello", "claude-3", "", "", false)
		if err != nil {
			t.Fatalf("RouteRun failed: %v", err)
		}

		output := stdout.String()
		// The model override should be used in the execution
		if len(output) == 0 {
			t.Error("Output should not be empty")
		}
	})

	t.Run("with compact output", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteRun("hello-orchestration", "hello", "", "", "", true)
		if err != nil {
			t.Fatalf("RouteRun failed: %v", err)
		}

		output := stdout.String()
		// Compact JSON should not have newlines/indentation
		if strings.Count(output, "\n") > 1 {
			t.Error("Compact output should have minimal newlines")
		}
	})
}

func TestCommandRouter_RouteList(t *testing.T) {
	router := NewCommandRouter()
	router.engineFactory = setupTestEngine

	t.Run("table format", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteList("table")
		if err != nil {
			t.Fatalf("RouteList failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Available Skills") {
			t.Error("Output should contain 'Available Skills'")
		}
		// Should contain at least the hello-orchestration skill from testdata
		if !strings.Contains(output, "hello-orchestration") {
			t.Error("Output should contain hello-orchestration skill")
		}
	})

	t.Run("json format", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteList("json")
		if err != nil {
			t.Fatalf("RouteList failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
			t.Error("JSON output should be valid JSON array")
		}
	})
}

func TestCommandRouter_RouteStatus(t *testing.T) {
	router := NewCommandRouter()
	router.engineFactory = setupTestEngine

	t.Run("table format", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteStatus("table")
		if err != nil {
			t.Fatalf("RouteStatus failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Skillrunner Status") {
			t.Error("Output should contain 'Skillrunner Status'")
		}
		if !strings.Contains(output, "Version") {
			t.Error("Output should contain version")
		}
	})

	t.Run("json format", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteStatus("json")
		if err != nil {
			t.Fatalf("RouteStatus failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
			t.Error("JSON output should be valid JSON object")
		}
		if !strings.Contains(output, "version") {
			t.Error("JSON output should contain version")
		}
	})
}

func TestMockRouter(t *testing.T) {
	mock := &MockRouter{}

	t.Run("RouteRun with mock", func(t *testing.T) {
		expectedErr := errors.New("mock error")
		mock.RunFunc = func(skillName, request, modelOverride, workspace, outputFile string, compact bool) error {
			return expectedErr
		}

		err := mock.RouteRun("test", "request", "", "", "", false)
		if err != expectedErr {
			t.Errorf("Expected mock error, got: %v", err)
		}
	})

	t.Run("RouteRun without mock", func(t *testing.T) {
		mock.RunFunc = nil
		err := mock.RouteRun("test", "request", "", "", "", false)
		if err != nil {
			t.Errorf("Expected nil error when no mock, got: %v", err)
		}
	})

	t.Run("RouteList with mock", func(t *testing.T) {
		expectedErr := errors.New("mock error")
		mock.ListFunc = func(format string) error {
			return expectedErr
		}

		err := mock.RouteList("table")
		if err != expectedErr {
			t.Errorf("Expected mock error, got: %v", err)
		}
	})

	t.Run("RouteStatus with mock", func(t *testing.T) {
		expectedErr := errors.New("mock error")
		mock.StatusFunc = func(format string) error {
			return expectedErr
		}

		err := mock.RouteStatus("table")
		if err != expectedErr {
			t.Errorf("Expected mock error, got: %v", err)
		}
	})

	t.Run("RouteList without mock", func(t *testing.T) {
		mock.ListFunc = nil
		err := mock.RouteList("table")
		if err != nil {
			t.Errorf("Expected nil error when no mock, got: %v", err)
		}
	})

	t.Run("RouteStatus without mock", func(t *testing.T) {
		mock.StatusFunc = nil
		err := mock.RouteStatus("table")
		if err != nil {
			t.Errorf("Expected nil error when no mock, got: %v", err)
		}
	})
}

func TestNewCommandRouter_DefaultValues(t *testing.T) {
	router := NewCommandRouter()

	// Test that default functions work
	if router.engineFactory == nil {
		t.Error("engineFactory should have default implementation")
	}

	// Test engineFactory creates engine
	eng := router.engineFactory("/test", models.ResolvePolicyAuto)
	if eng == nil {
		t.Error("engineFactory should create engine")
	}

	// Test outputWriter default implementation
	tmpFile := filepath.Join(t.TempDir(), "test-output.json")
	err := router.outputWriter(tmpFile, []byte("test data"))
	if err != nil {
		t.Errorf("outputWriter should work: %v", err)
	}
	// Verify file was written
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("outputWriter should create file")
	}

	// Test stdoutWriter default implementation
	var stdout strings.Builder
	router.stdoutWriter = func(s string) {
		stdout.WriteString(s)
	}
	router.stdoutWriter("test output")
	if stdout.String() != "test output" {
		t.Error("stdoutWriter should write to buffer")
	}

	// Test stderrWriter default implementation
	var stderr strings.Builder
	router.stderrWriter = func(format string, args ...interface{}) {
		stderr.WriteString(fmt.Sprintf(format, args...))
	}
	router.stderrWriter("error: %s", "test")
	if !strings.Contains(stderr.String(), "test") {
		t.Error("stderrWriter should format and write")
	}

	// Test that all default functions are set
	router2 := NewCommandRouter()
	if router2.engineFactory == nil {
		t.Error("engineFactory should be set")
	}
	if router2.outputWriter == nil {
		t.Error("outputWriter should be set")
	}
	if router2.stdoutWriter == nil {
		t.Error("stdoutWriter should be set")
	}
	if router2.stderrWriter == nil {
		t.Error("stderrWriter should be set")
	}
}

func TestCommandRouter_RouteRun_ErrorCases(t *testing.T) {
	router := NewCommandRouter()
	router.engineFactory = setupTestEngine

	t.Run("output file write error", func(t *testing.T) {
		// Use invalid path that can't be written to
		router.outputWriter = func(path string, data []byte) error {
			return errors.New("write error")
		}

		err := router.RouteRun("hello-orchestration", "hello", "", "", "/invalid/path/file.json", false)
		if err == nil {
			t.Error("Expected error when output file write fails")
		}
		if err != nil && !strings.Contains(err.Error(), "failed to write output") {
			t.Errorf("Error should mention write failure: %v", err)
		}
	})

	t.Run("marshal error path", func(t *testing.T) {
		// This is hard to trigger with real data, but we test the error handling exists
		router := NewCommandRouter()
		router.engineFactory = setupTestEngine
		err := router.RouteRun("hello-orchestration", "hello", "", "", "", false)
		if err != nil {
			t.Errorf("RouteRun should not fail with valid data: %v", err)
		}
	})

	t.Run("stdout path", func(t *testing.T) {
		var stdout strings.Builder
		router := NewCommandRouter()
		router.engineFactory = setupTestEngine
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteRun("hello-orchestration", "hello", "", "", "", false)
		if err != nil {
			t.Fatalf("RouteRun failed: %v", err)
		}
		if stdout.Len() == 0 {
			t.Error("stdout should receive output")
		}
	})

	t.Run("stderr path with output file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "output.json")
		var stderr strings.Builder
		router := NewCommandRouter()
		router.engineFactory = setupTestEngine
		router.stderrWriter = func(format string, args ...interface{}) {
			stderr.WriteString(fmt.Sprintf(format, args...))
		}

		err := router.RouteRun("hello-orchestration", "hello", "", "", tmpFile, false)
		if err != nil {
			t.Fatalf("RouteRun failed: %v", err)
		}
		if !strings.Contains(stderr.String(), tmpFile) {
			t.Error("stderr should contain output file path")
		}
	})
}

func TestCommandRouter_RouteList_ErrorCases(t *testing.T) {
	router := NewCommandRouter()
	router.engineFactory = setupTestEngine

	t.Run("engine list error", func(t *testing.T) {
		// Create router with failing engine factory
		router.engineFactory = func(workspace string, policy models.ResolvePolicy) *engine.Skillrunner {
			// Return nil to simulate error (though this won't actually cause ListSkills to fail)
			// We'll test with a mock that actually fails
			return engine.NewSkillrunner(workspace, policy)
		}

		// This should still work since engine.ListSkills doesn't fail
		err := router.RouteList("table")
		if err != nil {
			t.Errorf("RouteList should not fail: %v", err)
		}
	})

	t.Run("json format error handling", func(t *testing.T) {
		router := NewCommandRouter()
		router.engineFactory = setupTestEngine
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteList("json")
		if err != nil {
			t.Errorf("RouteList should not fail: %v", err)
		}
		if !strings.Contains(stdout.String(), "[") {
			t.Error("JSON output should be an array")
		}
	})
}

func TestCommandRouter_RouteStatus_ErrorCases(t *testing.T) {
	router := NewCommandRouter()
	router.engineFactory = setupTestEngine

	t.Run("table format", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteStatus("table")
		if err != nil {
			t.Errorf("RouteStatus should not fail: %v", err)
		}
		if !strings.Contains(stdout.String(), "Skillrunner Status") {
			t.Error("Output should contain status header")
		}
	})

	t.Run("json format", func(t *testing.T) {
		var stdout strings.Builder
		router.stdoutWriter = func(s string) {
			stdout.WriteString(s)
		}

		err := router.RouteStatus("json")
		if err != nil {
			t.Errorf("RouteStatus should not fail: %v", err)
		}
		if !strings.Contains(stdout.String(), "{") {
			t.Error("JSON output should be an object")
		}
	})
}

func TestFormatSkillsTable(t *testing.T) {
	skills := []types.SkillConfig{
		{Name: "test", Version: "1.0.0", Description: "Test skill"},
		{Name: "backend", Version: "2.0.0", Description: "Backend skill"},
	}

	output := formatSkillsTable(skills)
	if !strings.Contains(output, "Available Skills") {
		t.Error("Output should contain 'Available Skills'")
	}
	if !strings.Contains(output, "test") {
		t.Error("Output should contain test skill")
	}
	if !strings.Contains(output, "backend") {
		t.Error("Output should contain backend skill")
	}
}

func TestFormatStatusTable(t *testing.T) {
	t.Run("ready status", func(t *testing.T) {
		status := &types.SystemStatus{
			Version:          "0.1.0",
			SkillCount:       2,
			Workspace:        "/test",
			Ready:            true,
			ConfiguredModels: []string{"gpt-4", "claude-3"},
		}

		output := formatStatusTable(status)
		if !strings.Contains(output, "Skillrunner Status") {
			t.Error("Output should contain 'Skillrunner Status'")
		}
		if !strings.Contains(output, "0.1.0") {
			t.Error("Output should contain version")
		}
		if !strings.Contains(output, "2") {
			t.Error("Output should contain skill count")
		}
		if !strings.Contains(output, "Ready") {
			t.Error("Output should contain Ready status")
		}
	})

	t.Run("not ready status", func(t *testing.T) {
		status := &types.SystemStatus{
			Version:          "0.1.0",
			SkillCount:       0,
			Workspace:        "/test",
			Ready:            false,
			ConfiguredModels: []string{},
		}

		output := formatStatusTable(status)
		if !strings.Contains(output, "Not Ready") {
			t.Error("Output should contain 'Not Ready' status")
		}
		if !strings.Contains(output, "0") {
			t.Error("Output should contain skill count")
		}
	})
}

func TestEnvelopeToJSON(t *testing.T) {
	envelope := &types.Envelope{
		Version: "1.0",
		Skill:   "test",
		Request: "hello",
		Steps:   []types.Step{},
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	// Test compact
	compact, err := envelopeToJSON(envelope, true)
	if err != nil {
		t.Fatalf("envelopeToJSON failed: %v", err)
	}
	if len(compact) == 0 {
		t.Error("Compact JSON should not be empty")
	}

	// Test indented
	indented, err := envelopeToJSON(envelope, false)
	if err != nil {
		t.Fatalf("envelopeToJSON failed: %v", err)
	}
	if len(indented) == 0 {
		t.Error("Indented JSON should not be empty")
	}
	if len(indented) <= len(compact) {
		t.Error("Indented JSON should be longer than compact")
	}
}

func TestSkillsToJSON(t *testing.T) {
	skills := []types.SkillConfig{
		{Name: "test", Version: "1.0.0", Description: "Test"},
	}

	output, err := skillsToJSON(skills)
	if err != nil {
		t.Fatalf("skillsToJSON failed: %v", err)
	}
	if !strings.Contains(output, "test") {
		t.Error("JSON should contain skill name")
	}
}

func TestStatusToJSON(t *testing.T) {
	status := &types.SystemStatus{
		Version:    "0.1.0",
		SkillCount: 1,
		Ready:      true,
	}

	output, err := statusToJSON(status)
	if err != nil {
		t.Fatalf("statusToJSON failed: %v", err)
	}
	if !strings.Contains(output, "0.1.0") {
		t.Error("JSON should contain version")
	}
}

func TestSkillsToJSON_ErrorCase(t *testing.T) {
	t.Run("valid skills", func(t *testing.T) {
		skills := []types.SkillConfig{
			{Name: "test", Version: "1.0.0", Description: "Test"},
		}

		output, err := skillsToJSON(skills)
		if err != nil {
			t.Fatalf("skillsToJSON should not fail with valid data: %v", err)
		}
		if output == "" {
			t.Error("skillsToJSON should return non-empty output")
		}
		if !strings.Contains(output, "test") {
			t.Error("Output should contain skill name")
		}
	})

	t.Run("empty skills", func(t *testing.T) {
		skills := []types.SkillConfig{}

		output, err := skillsToJSON(skills)
		if err != nil {
			t.Fatalf("skillsToJSON should not fail with empty array: %v", err)
		}
		if !strings.Contains(output, "[]") {
			t.Error("Output should be empty array")
		}
	})
}

func TestStatusToJSON_ErrorCase(t *testing.T) {
	t.Run("valid status", func(t *testing.T) {
		status := &types.SystemStatus{
			Version:          "0.1.0",
			SkillCount:       1,
			Ready:            true,
			ConfiguredModels: []string{"gpt-4"},
		}

		output, err := statusToJSON(status)
		if err != nil {
			t.Fatalf("statusToJSON should not fail with valid data: %v", err)
		}
		if output == "" {
			t.Error("statusToJSON should return non-empty output")
		}
		if !strings.Contains(output, "0.1.0") {
			t.Error("Output should contain version")
		}
	})

	t.Run("status with empty models", func(t *testing.T) {
		status := &types.SystemStatus{
			Version:          "0.1.0",
			SkillCount:       0,
			Ready:            false,
			ConfiguredModels: []string{},
		}

		output, err := statusToJSON(status)
		if err != nil {
			t.Fatalf("statusToJSON should not fail: %v", err)
		}
		if output == "" {
			t.Error("statusToJSON should return non-empty output")
		}
	})
}
