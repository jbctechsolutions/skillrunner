package metrics

import (
	"os"
	"testing"
	"time"
)

func TestNewStorage(t *testing.T) {
	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	if storage == nil {
		t.Fatal("Storage should not be nil")
	}
}

func TestStorage_RecordExecution(t *testing.T) {
	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	record := ExecutionRecord{
		Timestamp:    time.Now(),
		Skill:        "test-skill",
		Model:        "ollama/test-model",
		InputTokens:  100,
		OutputTokens: 50,
		Cost:         0.0,
		DurationMs:   1000,
		Success:      true,
	}

	err = storage.RecordExecution(record)
	if err != nil {
		t.Fatalf("RecordExecution failed: %v", err)
	}
}

func TestStorage_GetMetrics(t *testing.T) {
	// Use temp directory to avoid loading existing records
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// Record some executions
	_ = storage.RecordExecution(ExecutionRecord{
		Timestamp:    time.Now(),
		Skill:        "test-skill",
		Model:        "ollama/test-model",
		InputTokens:  100,
		OutputTokens: 50,
		Cost:         0.0,
		DurationMs:   1000,
		Success:      true,
	})

	_ = storage.RecordExecution(ExecutionRecord{
		Timestamp:    time.Now(),
		Skill:        "test-skill",
		Model:        "anthropic/claude-sonnet",
		InputTokens:  200,
		OutputTokens: 100,
		Cost:         0.001,
		DurationMs:   2000,
		Success:      true,
	})

	// Get metrics
	since := time.Now().Add(-1 * time.Hour)
	metrics := storage.GetMetrics(since)

	if metrics.LocalCalls != 1 {
		t.Errorf("LocalCalls = %d, want 1", metrics.LocalCalls)
	}
	if metrics.CloudCalls != 1 {
		t.Errorf("CloudCalls = %d, want 1", metrics.CloudCalls)
	}
	if metrics.TotalTokens != 450 {
		t.Errorf("TotalTokens = %d, want 450", metrics.TotalTokens)
	}
}

func TestStorage_GetRecords(t *testing.T) {
	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// Record execution
	_ = storage.RecordExecution(ExecutionRecord{
		Timestamp:    time.Now(),
		Skill:        "test-skill",
		Model:        "ollama/test-model",
		InputTokens:  100,
		OutputTokens: 50,
		Success:      true,
	})

	// Get records
	since := time.Now().Add(-1 * time.Hour)
	records := storage.GetRecords(since, "", "")

	if len(records) == 0 {
		t.Error("Should return at least one record")
	}

	// Filter by skill
	records = storage.GetRecords(since, "test-skill", "")
	if len(records) == 0 {
		t.Error("Should return records for test-skill")
	}

	// Filter by model
	records = storage.GetRecords(since, "", "ollama/test-model")
	if len(records) == 0 {
		t.Error("Should return records for ollama/test-model")
	}
}

func TestIsLocalModel(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"ollama/test-model", true},
		{"anthropic/claude-sonnet", false},
		{"openai/gpt-4", false},
		{"test-model", false},
	}

	for _, tt := range tests {
		result := isLocalModel(tt.model)
		if result != tt.expected {
			t.Errorf("isLocalModel(%s) = %v, want %v", tt.model, result, tt.expected)
		}
	}
}

func TestStorage_LoadSave(t *testing.T) {
	// Use temp directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)

	storage1, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// Record execution
	_ = storage1.RecordExecution(ExecutionRecord{
		Timestamp:    time.Now(),
		Skill:        "test-skill",
		Model:        "ollama/test-model",
		InputTokens:  100,
		OutputTokens: 50,
		Success:      true,
	})

	// Create new storage (should load from disk)
	storage2, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// Should have the record
	since := time.Now().Add(-1 * time.Hour)
	records := storage2.GetRecords(since, "", "")
	if len(records) == 0 {
		t.Error("Should load records from disk")
	}
}
