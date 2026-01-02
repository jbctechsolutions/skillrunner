package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		check  func(t *testing.T, buf *bytes.Buffer)
	}{
		{
			name: "text format",
			config: Config{
				Level:  LevelInfo,
				Format: FormatText,
			},
			check: func(t *testing.T, buf *bytes.Buffer) {
				if !strings.Contains(buf.String(), "level=INFO") {
					t.Error("expected text format with level=INFO")
				}
			},
		},
		{
			name: "json format",
			config: Config{
				Level:  LevelInfo,
				Format: FormatJSON,
			},
			check: func(t *testing.T, buf *bytes.Buffer) {
				var m map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
					t.Errorf("expected valid JSON output: %v", err)
				}
				if m["level"] != "INFO" {
					t.Errorf("expected level INFO, got %v", m["level"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			tt.config.Output = buf

			logger := New(tt.config)
			logger.Info("test message")

			tt.check(t, buf)
		})
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     Level
		logMethod func(l *Logger)
		expected  bool
	}{
		{
			name:      "debug at debug level",
			level:     LevelDebug,
			logMethod: func(l *Logger) { l.Debug("test") },
			expected:  true,
		},
		{
			name:      "debug at info level",
			level:     LevelInfo,
			logMethod: func(l *Logger) { l.Debug("test") },
			expected:  false,
		},
		{
			name:      "info at info level",
			level:     LevelInfo,
			logMethod: func(l *Logger) { l.Info("test") },
			expected:  true,
		},
		{
			name:      "warn at error level",
			level:     LevelError,
			logMethod: func(l *Logger) { l.Warn("test") },
			expected:  false,
		},
		{
			name:      "error at error level",
			level:     LevelError,
			logMethod: func(l *Logger) { l.Error("test") },
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(Config{
				Level:  tt.level,
				Format: FormatText,
				Output: buf,
			})

			tt.logMethod(logger)

			hasOutput := buf.Len() > 0
			if hasOutput != tt.expected {
				t.Errorf("expected output=%v, got output=%v", tt.expected, hasOutput)
			}
		})
	}
}

func TestContextEnrichment(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  LevelDebug,
		Format: FormatJSON,
		Output: buf,
	})

	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "corr-123")
	ctx = WithWorkflowID(ctx, "wf-456")
	ctx = WithPhaseID(ctx, "phase-789")
	ctx = WithProvider(ctx, "anthropic")
	ctx = WithSkillID(ctx, "code-review")

	logger.InfoContext(ctx, "enriched log")

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	expected := map[string]string{
		"correlation_id": "corr-123",
		"workflow_id":    "wf-456",
		"phase_id":       "phase-789",
		"provider":       "anthropic",
		"skill_id":       "code-review",
	}

	for key, expectedVal := range expected {
		if m[key] != expectedVal {
			t.Errorf("expected %s=%s, got %v", key, expectedVal, m[key])
		}
	}
}

func TestWith(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: buf,
	})

	childLogger := logger.With("component", "executor")
	childLogger.Info("with attributes")

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if m["component"] != "executor" {
		t.Errorf("expected component=executor, got %v", m["component"])
	}
}

func TestWithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: buf,
	})

	childLogger := logger.WithGroup("metrics")
	childLogger.Info("grouped log", "count", 42)

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// The group should contain the "count" attribute
	metrics, ok := m["metrics"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metrics group, got %v", m["metrics"])
	}

	if metrics["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", metrics["count"])
	}
}

func TestCorrelationIDExtraction(t *testing.T) {
	ctx := context.Background()

	// No correlation ID
	if id := CorrelationID(ctx); id != "" {
		t.Errorf("expected empty correlation ID, got %s", id)
	}

	// With correlation ID
	ctx = WithCorrelationID(ctx, "test-id")
	if id := CorrelationID(ctx); id != "test-id" {
		t.Errorf("expected correlation ID 'test-id', got %s", id)
	}
}

func TestDomainLogHelpers(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  LevelDebug,
		Format: FormatJSON,
		Output: buf,
	})

	ctx := context.Background()

	t.Run("LogWorkflowStart", func(t *testing.T) {
		buf.Reset()
		LogWorkflowStart(ctx, logger, "code-review", "Code Review")

		var m map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if m["msg"] != "workflow execution started" {
			t.Errorf("unexpected message: %v", m["msg"])
		}
		if m["skill_id"] != "code-review" {
			t.Errorf("unexpected skill_id: %v", m["skill_id"])
		}
	})

	t.Run("LogWorkflowComplete", func(t *testing.T) {
		buf.Reset()
		LogWorkflowComplete(ctx, logger, "code-review", 5*time.Second, 1000)

		var m map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if m["duration_ms"] != float64(5000) {
			t.Errorf("unexpected duration_ms: %v", m["duration_ms"])
		}
		if m["total_tokens"] != float64(1000) {
			t.Errorf("unexpected total_tokens: %v", m["total_tokens"])
		}
	})

	t.Run("LogPhaseComplete", func(t *testing.T) {
		buf.Reset()
		LogPhaseComplete(ctx, logger, "analysis", 500, 200, 2*time.Second, true)

		var m map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if m["input_tokens"] != float64(500) {
			t.Errorf("unexpected input_tokens: %v", m["input_tokens"])
		}
		if m["cache_hit"] != true {
			t.Errorf("unexpected cache_hit: %v", m["cache_hit"])
		}
	})

	t.Run("LogCostIncurred", func(t *testing.T) {
		buf.Reset()
		LogCostIncurred(ctx, logger, "anthropic", "claude-3-5-sonnet", 0.0015, 500, 200)

		var m map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if m["provider"] != "anthropic" {
			t.Errorf("unexpected provider: %v", m["provider"])
		}
		if m["cost_usd"] != 0.0015 {
			t.Errorf("unexpected cost_usd: %v", m["cost_usd"])
		}
	})
}

func TestDefaultLogger(t *testing.T) {
	// Reset global for test
	global = nil
	globalOnce = sync.Once{}

	logger := Default()
	if logger == nil {
		t.Error("expected non-nil default logger")
	}

	// Calling Default() again should return the same instance
	logger2 := Default()
	if logger != logger2 {
		t.Error("expected same logger instance from Default()")
	}
}
