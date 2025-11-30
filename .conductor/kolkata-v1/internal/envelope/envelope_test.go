package envelope

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("test-skill", "test request")

	if builder == nil {
		t.Fatal("NewBuilder returned nil")
	}

	if builder.envelope.Skill != "test-skill" {
		t.Errorf("Skill = %s; want test-skill", builder.envelope.Skill)
	}

	if builder.envelope.Request != "test request" {
		t.Errorf("Request = %s; want test request", builder.envelope.Request)
	}

	if builder.envelope.Version != "1.0" {
		t.Errorf("Version = %s; want 1.0", builder.envelope.Version)
	}

	if builder.envelope.Metadata["created_at"] == nil {
		t.Error("created_at metadata not set")
	}
}

func TestAddStep(t *testing.T) {
	builder := NewBuilder("test", "test")

	step := types.Step{
		Intent:  types.IntentPlan,
		Model:   "gpt-4",
		Prompt:  "test prompt",
		Context: nil, // Test nil handling
		FileOps: nil, // Test nil handling
	}

	builder.AddStep(step)

	if len(builder.envelope.Steps) != 1 {
		t.Errorf("Steps count = %d; want 1", len(builder.envelope.Steps))
	}

	// Check that nil slices were converted to empty slices
	addedStep := builder.envelope.Steps[0]
	if addedStep.Context == nil {
		t.Error("Context should not be nil")
	}
	if addedStep.FileOps == nil {
		t.Error("FileOps should not be nil")
	}
}

func TestAddMetadata(t *testing.T) {
	builder := NewBuilder("test", "test")

	builder.AddMetadata("key1", "value1")
	builder.AddMetadata("key2", 42)

	if builder.envelope.Metadata["key1"] != "value1" {
		t.Error("key1 metadata not set correctly")
	}

	if builder.envelope.Metadata["key2"] != 42 {
		t.Error("key2 metadata not set correctly")
	}
}

func TestChaining(t *testing.T) {
	builder := NewBuilder("test", "test")

	// Test method chaining
	result := builder.
		AddMetadata("key", "value").
		AddStep(types.Step{
			Intent:  types.IntentPlan,
			Model:   "gpt-4",
			Context: []types.ContextItem{},
			FileOps: []types.FileOperation{},
		}).
		AddMetadata("another", "value")

	if result != builder {
		t.Error("Method chaining broken")
	}

	if len(builder.envelope.Steps) != 1 {
		t.Error("Step not added during chaining")
	}

	if builder.envelope.Metadata["key"] != "value" {
		t.Error("First metadata not added during chaining")
	}

	if builder.envelope.Metadata["another"] != "value" {
		t.Error("Second metadata not added during chaining")
	}
}

func TestBuild(t *testing.T) {
	builder := NewBuilder("test", "test")

	step := types.Step{
		Intent:   types.IntentPlan,
		Model:    "gpt-4",
		Prompt:   "test",
		Context:  []types.ContextItem{},
		FileOps:  []types.FileOperation{},
		Metadata: map[string]interface{}{},
	}

	builder.AddStep(step)
	envelope := builder.Build()

	if envelope == nil {
		t.Fatal("Build returned nil")
	}

	if envelope.Skill != "test" {
		t.Errorf("Built envelope skill = %s; want test", envelope.Skill)
	}

	if len(envelope.Steps) != 1 {
		t.Errorf("Built envelope steps = %d; want 1", len(envelope.Steps))
	}
}

func TestToJSON(t *testing.T) {
	builder := NewBuilder("test", "test request")

	step := types.Step{
		Intent:   types.IntentPlan,
		Model:    "gpt-4",
		Prompt:   "test prompt",
		Context:  []types.ContextItem{},
		FileOps:  []types.FileOperation{},
		Metadata: map[string]interface{}{},
	}

	builder.AddStep(step)

	// Test pretty JSON
	prettyJSON, err := builder.ToJSON(false)
	if err != nil {
		t.Fatalf("ToJSON(false) failed: %v", err)
	}

	if !strings.Contains(prettyJSON, "\n") {
		t.Error("Pretty JSON should contain newlines")
	}

	// Test compact JSON
	compactJSON, err := builder.ToJSON(true)
	if err != nil {
		t.Fatalf("ToJSON(true) failed: %v", err)
	}

	if strings.Contains(compactJSON, "\n  ") {
		t.Error("Compact JSON should not contain indentation")
	}

	// Verify it's valid JSON
	var decoded types.Envelope
	err = json.Unmarshal([]byte(prettyJSON), &decoded)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	if decoded.Skill != "test" {
		t.Error("JSON decoding produced incorrect skill name")
	}
}

func TestToDict(t *testing.T) {
	builder := NewBuilder("test", "test request")

	step := types.Step{
		Intent:   types.IntentPlan,
		Model:    "gpt-4",
		Context:  []types.ContextItem{},
		FileOps:  []types.FileOperation{},
		Metadata: map[string]interface{}{},
	}

	builder.AddStep(step)
	dict := builder.ToDict()

	if dict["skill"] != "test" {
		t.Error("Dict should contain skill key")
	}

	if dict["request"] != "test request" {
		t.Error("Dict should contain request key")
	}

	steps, ok := dict["steps"].([]interface{})
	if !ok || len(steps) != 1 {
		t.Error("Dict should contain steps array with 1 element")
	}
}

func TestMultipleSteps(t *testing.T) {
	builder := NewBuilder("backend-architect", "add auth")

	// Add multiple steps
	builder.AddStep(types.Step{
		Intent:   types.IntentPlan,
		Model:    "gpt-4",
		Prompt:   "plan",
		Context:  []types.ContextItem{},
		FileOps:  []types.FileOperation{},
		Metadata: map[string]interface{}{"phase": "planning"},
	})

	builder.AddStep(types.Step{
		Intent:   types.IntentEdit,
		Model:    "gpt-4",
		Prompt:   "edit",
		Context:  []types.ContextItem{},
		FileOps:  []types.FileOperation{},
		Metadata: map[string]interface{}{"phase": "implementation"},
	})

	builder.AddStep(types.Step{
		Intent:   types.IntentRun,
		Model:    "gpt-4",
		Prompt:   "run",
		Context:  []types.ContextItem{},
		FileOps:  []types.FileOperation{},
		Metadata: map[string]interface{}{"phase": "verification"},
	})

	envelope := builder.Build()

	if len(envelope.Steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(envelope.Steps))
	}

	// Verify step order and intents
	if envelope.Steps[0].Intent != types.IntentPlan {
		t.Error("First step should be plan")
	}
	if envelope.Steps[1].Intent != types.IntentEdit {
		t.Error("Second step should be edit")
	}
	if envelope.Steps[2].Intent != types.IntentRun {
		t.Error("Third step should be run")
	}
}
