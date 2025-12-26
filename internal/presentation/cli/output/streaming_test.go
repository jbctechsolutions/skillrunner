package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamingOutput_Basic(t *testing.T) {
	var buf bytes.Buffer
	so := NewStreamingOutput(
		WithStreamingWriter(&buf),
		WithStreamingColor(false),
		WithShowTokenCounts(true),
		WithShowPhaseInfo(true),
	)

	so.StartWorkflow("Test Skill", "1.0.0", 2)
	output := buf.String()

	if !strings.Contains(output, "Test Skill v1.0.0") {
		t.Errorf("expected workflow header, got: %s", output)
	}
}

func TestStreamingOutput_PhaseLifecycle(t *testing.T) {
	var buf bytes.Buffer
	so := NewStreamingOutput(
		WithStreamingWriter(&buf),
		WithStreamingColor(false),
		WithShowTokenCounts(true),
		WithShowPhaseInfo(true),
	)

	so.StartWorkflow("Test", "1.0", 1)
	buf.Reset()

	so.StartPhase("phase1", "Analysis Phase", 1)
	if !strings.Contains(buf.String(), "[1/1] Analysis Phase") {
		t.Errorf("expected phase header, got: %s", buf.String())
	}

	buf.Reset()
	so.WriteChunk("Hello")
	so.WriteChunk(" World")
	if buf.String() != "Hello World" {
		t.Errorf("expected 'Hello World', got: %s", buf.String())
	}

	buf.Reset()
	so.CompletePhase(100, 50, "gpt-4")
	output := buf.String()
	if !strings.Contains(output, "Analysis Phase completed") {
		t.Errorf("expected completion message, got: %s", output)
	}
	if !strings.Contains(output, "tokens: 100 in, 50 out") {
		t.Errorf("expected token counts, got: %s", output)
	}
}

func TestStreamingOutput_FailedPhase(t *testing.T) {
	var buf bytes.Buffer
	so := NewStreamingOutput(
		WithStreamingWriter(&buf),
		WithStreamingColor(false),
	)

	so.StartWorkflow("Test", "1.0", 1)
	so.StartPhase("phase1", "Failing Phase", 1)
	buf.Reset()

	testErr := &testError{message: "connection failed"}
	so.FailPhase(testErr)

	output := buf.String()
	if !strings.Contains(output, "Failing Phase failed") {
		t.Errorf("expected failure message, got: %s", output)
	}
	if !strings.Contains(output, "connection failed") {
		t.Errorf("expected error message, got: %s", output)
	}
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func TestStreamingOutput_WorkflowCompletion(t *testing.T) {
	tests := []struct {
		name     string
		success  bool
		expected string
	}{
		{"success", true, "Workflow completed successfully"},
		{"failure", false, "Workflow failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			so := NewStreamingOutput(
				WithStreamingWriter(&buf),
				WithStreamingColor(false),
				WithShowTokenCounts(true),
			)

			so.StartWorkflow("Test", "1.0", 1)
			so.StartPhase("p1", "Phase 1", 1)
			so.CompletePhase(100, 50, "model")
			buf.Reset()

			so.CompleteWorkflow(tt.success)
			if !strings.Contains(buf.String(), tt.expected) {
				t.Errorf("expected %q in output, got: %s", tt.expected, buf.String())
			}
		})
	}
}

func TestStreamingOutput_TokenUpdates(t *testing.T) {
	so := NewStreamingOutput()

	so.UpdateTokens(100, 50)

	input, output := so.GetTotalTokens()
	if input != 100 {
		t.Errorf("expected input tokens 100, got %d", input)
	}
	if output != 50 {
		t.Errorf("expected output tokens 50, got %d", output)
	}
}

func TestStreamingOutput_NoPhaseInfo(t *testing.T) {
	var buf bytes.Buffer
	so := NewStreamingOutput(
		WithStreamingWriter(&buf),
		WithStreamingColor(false),
		WithShowPhaseInfo(false),
	)

	so.StartWorkflow("Test", "1.0", 1)
	buf.Reset()

	so.StartPhase("p1", "Phase 1", 1)
	if strings.Contains(buf.String(), "Phase 1") {
		t.Error("expected no phase info when disabled")
	}
}

func TestLiveTokenCounter_Basic(t *testing.T) {
	var buf bytes.Buffer
	ltc := NewLiveTokenCounter(&buf, false)

	ltc.Update(100, 50)
	ltc.Final()

	output := buf.String()
	if !strings.Contains(output, "150") {
		t.Errorf("expected total tokens 150, got: %s", output)
	}
}

func TestLiveTokenCounter_Clear(t *testing.T) {
	var buf bytes.Buffer
	ltc := NewLiveTokenCounter(&buf, false)

	ltc.Update(100, 50)
	buf.Reset()
	ltc.Clear()

	// Should contain spaces to clear the line
	if !strings.Contains(buf.String(), "\r") {
		t.Error("expected carriage return for clearing")
	}
}

func TestFormatStreamDuration(t *testing.T) {
	// Just validate the function exists and returns strings
	// Actual duration formatting is implementation-specific
	result := formatStreamDuration(0)
	if result == "" {
		t.Error("expected non-empty duration string")
	}
}
