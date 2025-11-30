package types

import (
	"encoding/json"
	"testing"
)

func TestIntentConstants(t *testing.T) {
	tests := []struct {
		intent   Intent
		expected string
	}{
		{IntentPlan, "plan"},
		{IntentEdit, "edit"},
		{IntentRun, "run"},
	}

	for _, tt := range tests {
		if string(tt.intent) != tt.expected {
			t.Errorf("Intent %v = %s; want %s", tt.intent, string(tt.intent), tt.expected)
		}
	}
}

func TestFileOperationJSON(t *testing.T) {
	op := FileOperation{
		Op:      "write",
		Path:    "/test/file.go",
		Content: "package test",
		Pattern: "*.go",
	}

	data, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("Failed to marshal FileOperation: %v", err)
	}

	var decoded FileOperation
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal FileOperation: %v", err)
	}

	if decoded.Op != op.Op || decoded.Path != op.Path {
		t.Errorf("Decoded FileOperation doesn't match original")
	}
}

func TestContextItemJSON(t *testing.T) {
	ctx := ContextItem{
		Type:   "folder",
		Source: "/src",
		Filter: "*.go",
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("Failed to marshal ContextItem: %v", err)
	}

	var decoded ContextItem
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ContextItem: %v", err)
	}

	if decoded.Type != ctx.Type || decoded.Source != ctx.Source {
		t.Errorf("Decoded ContextItem doesn't match original")
	}
}

func TestStepJSON(t *testing.T) {
	step := Step{
		Intent:  IntentPlan,
		Model:   "gpt-4",
		Prompt:  "Test prompt",
		Context: []ContextItem{{Type: "folder", Source: "/src"}},
		FileOps: []FileOperation{{Op: "read", Path: "/test"}},
		Metadata: map[string]interface{}{
			"phase": "testing",
		},
	}

	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("Failed to marshal Step: %v", err)
	}

	var decoded Step
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Step: %v", err)
	}

	if decoded.Intent != step.Intent || decoded.Model != step.Model {
		t.Errorf("Decoded Step doesn't match original")
	}
}

func TestEnvelopeJSON(t *testing.T) {
	envelope := Envelope{
		Version: "1.0",
		Skill:   "test",
		Request: "test request",
		Steps: []Step{
			{
				Intent:   IntentPlan,
				Model:    "gpt-4",
				Context:  []ContextItem{},
				FileOps:  []FileOperation{},
				Metadata: map[string]interface{}{},
			},
		},
		Metadata: map[string]interface{}{
			"test": true,
		},
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Failed to marshal Envelope: %v", err)
	}

	var decoded Envelope
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Envelope: %v", err)
	}

	if decoded.Version != envelope.Version || decoded.Skill != envelope.Skill {
		t.Errorf("Decoded Envelope doesn't match original")
	}

	if len(decoded.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(decoded.Steps))
	}
}

func TestSkillConfigJSON(t *testing.T) {
	config := SkillConfig{
		Name:         "test-skill",
		Version:      "1.0.0",
		Description:  "Test skill",
		DefaultModel: "gpt-4",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal SkillConfig: %v", err)
	}

	var decoded SkillConfig
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal SkillConfig: %v", err)
	}

	if decoded.Name != config.Name || decoded.Version != config.Version {
		t.Errorf("Decoded SkillConfig doesn't match original")
	}
}

func TestSystemStatusJSON(t *testing.T) {
	status := SystemStatus{
		Version:          "0.1.0",
		SkillCount:       5,
		Workspace:        "/test",
		Ready:            true,
		ConfiguredModels: []string{"gpt-4", "claude-3"},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal SystemStatus: %v", err)
	}

	var decoded SystemStatus
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal SystemStatus: %v", err)
	}

	if decoded.Version != status.Version || decoded.SkillCount != status.SkillCount {
		t.Errorf("Decoded SystemStatus doesn't match original")
	}
}
