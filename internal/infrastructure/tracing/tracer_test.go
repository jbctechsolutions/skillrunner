package tracing

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled {
		t.Error("expected tracing to be disabled by default")
	}

	if cfg.ExporterType != ExporterNone {
		t.Errorf("expected exporter type 'none', got %s", cfg.ExporterType)
	}

	if cfg.ServiceName != "skillrunner" {
		t.Errorf("expected service name 'skillrunner', got %s", cfg.ServiceName)
	}

	if cfg.SampleRate != 1.0 {
		t.Errorf("expected sample rate 1.0, got %f", cfg.SampleRate)
	}
}

func TestNew_Disabled(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Enabled:      false,
		ExporterType: ExporterNone,
	}

	tracer, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}

	// Starting a span should work even when disabled
	ctx, span := tracer.Start(ctx, "test-span")
	if span == nil {
		t.Error("expected non-nil span")
	}
	span.End()

	_ = ctx // Use ctx to avoid unused variable warning
}

func TestNew_StdoutExporter(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	cfg := Config{
		Enabled:      true,
		ExporterType: ExporterStdout,
		ServiceName:  "test-service",
		Environment:  "test",
		SampleRate:   1.0,
		Output:       buf,
	}

	tracer, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tracer.Shutdown(ctx)

	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}

	if tracer.provider == nil {
		t.Error("expected non-nil provider for enabled tracer")
	}
}

func TestWorkflowSpan(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	cfg := Config{
		Enabled:      true,
		ExporterType: ExporterStdout,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Output:       buf,
	}

	tracer, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tracer.Shutdown(ctx)

	ctx, ws := tracer.StartWorkflowSpan(ctx, "code-review", "Code Review")

	ws.SetPhaseCount(3)
	ws.SetTotalTokens(1000, 500)
	ws.SetCost(0.015)
	ws.SetCacheStats(2, 1)
	ws.End()

	// Flush the provider
	tracer.Shutdown(ctx)

	// Check that some output was generated
	if buf.Len() == 0 {
		t.Error("expected trace output to be written")
	}
}

func TestWorkflowSpan_Error(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	cfg := Config{
		Enabled:      true,
		ExporterType: ExporterStdout,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Output:       buf,
	}

	tracer, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tracer.Shutdown(ctx)

	ctx, ws := tracer.StartWorkflowSpan(ctx, "code-review", "Code Review")
	ws.EndWithError(errors.New("test error"))

	tracer.Shutdown(ctx)

	if buf.Len() == 0 {
		t.Error("expected trace output to be written")
	}
}

func TestPhaseSpan(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	cfg := Config{
		Enabled:      true,
		ExporterType: ExporterStdout,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Output:       buf,
	}

	tracer, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tracer.Shutdown(ctx)

	ctx, ps := tracer.StartPhaseSpan(ctx, "analysis", "Pattern Analysis")

	ps.SetProvider("anthropic", "claude-3-5-sonnet")
	ps.SetTokens(500, 200)
	ps.SetCost(0.005)
	ps.SetCacheHit(false)
	ps.End()

	tracer.Shutdown(ctx)

	if buf.Len() == 0 {
		t.Error("expected trace output to be written")
	}
}

func TestProviderSpan(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	cfg := Config{
		Enabled:      true,
		ExporterType: ExporterStdout,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Output:       buf,
	}

	tracer, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tracer.Shutdown(ctx)

	ctx, ps := tracer.StartProviderSpan(ctx, "anthropic", "claude-3-5-sonnet")

	ps.SetRequestTokens(500)
	ps.SetResponse(200, "stop")
	ps.End()

	tracer.Shutdown(ctx)

	if buf.Len() == 0 {
		t.Error("expected trace output to be written")
	}
}

func TestDefault(t *testing.T) {
	// Reset global for test
	global = nil

	tracer := Default()
	if tracer == nil {
		t.Error("expected non-nil default tracer")
	}

	// Should return a no-op tracer
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test")
	span.End()
	_ = ctx
}

func TestSpanHelpers(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	cfg := Config{
		Enabled:      true,
		ExporterType: ExporterStdout,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Output:       buf,
	}

	tracer, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tracer.Shutdown(ctx)

	ctx, span := tracer.Start(ctx, "test-span")

	// Test AddEvent
	AddEvent(ctx, "test-event")

	// Test RecordError
	RecordError(ctx, errors.New("test error"))

	// Test SetAttribute with different types
	SetAttribute(ctx, "string-attr", "value")
	SetAttribute(ctx, "int-attr", 42)
	SetAttribute(ctx, "int64-attr", int64(100))
	SetAttribute(ctx, "float-attr", 3.14)
	SetAttribute(ctx, "bool-attr", true)

	span.End()
	tracer.Shutdown(ctx)

	if buf.Len() == 0 {
		t.Error("expected trace output to be written")
	}
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	cfg := Config{
		Enabled:      true,
		ExporterType: ExporterStdout,
		ServiceName:  "test-service",
		SampleRate:   1.0,
		Output:       buf,
	}

	tracer, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tracer.Shutdown(ctx)

	ctx, _ = tracer.Start(ctx, "test-span")
	span := SpanFromContext(ctx)

	if span == nil {
		t.Error("expected non-nil span from context")
	}
}

func TestSamplers(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate float64
	}{
		{"always sample", 1.0},
		{"never sample", 0.0},
		{"ratio sample", 0.5},
		{"above max", 1.5},
		{"below min", -0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			buf := &bytes.Buffer{}

			cfg := Config{
				Enabled:      true,
				ExporterType: ExporterStdout,
				ServiceName:  "test-service",
				SampleRate:   tt.sampleRate,
				Output:       buf,
			}

			tracer, err := New(ctx, cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tracer.Shutdown(ctx)
		})
	}
}
