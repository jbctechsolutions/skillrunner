// Package tracing provides OpenTelemetry-based distributed tracing infrastructure.
// It supports multiple exporters (stdout, OTLP, Jaeger) and provides domain-specific
// span helpers for workflow and phase execution tracing.
package tracing

import (
	"context"
	"fmt"
	"io"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	// TracerName is the name used for the skillrunner tracer.
	TracerName = "github.com/jbctechsolutions/skillrunner"

	// Version is the semantic version of the tracer.
	Version = "2.0.0"
)

// ExporterType defines the type of trace exporter.
type ExporterType string

const (
	ExporterNone   ExporterType = "none"
	ExporterStdout ExporterType = "stdout"
	ExporterOTLP   ExporterType = "otlp"
)

// Config holds tracing configuration.
type Config struct {
	Enabled      bool         // Whether tracing is enabled
	ExporterType ExporterType // Type of exporter to use
	OTLPEndpoint string       // OTLP collector endpoint (for OTLP exporter)
	ServiceName  string       // Service name for traces
	Environment  string       // Deployment environment (development, production)
	SampleRate   float64      // Sampling rate (0.0 to 1.0)
	Output       io.Writer    // Output for stdout exporter (defaults to os.Stdout)
}

// DefaultConfig returns sensible default tracing configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:      false,
		ExporterType: ExporterNone,
		ServiceName:  "skillrunner",
		Environment:  "development",
		SampleRate:   1.0,
	}
}

// Tracer wraps an OpenTelemetry tracer with domain-specific functionality.
type Tracer struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	config   Config
}

// global is the package-level default tracer.
var (
	global     *Tracer
	globalOnce sync.Once
)

// Init initializes the global tracer with the provided configuration.
func Init(ctx context.Context, cfg Config) (*Tracer, error) {
	var err error
	globalOnce.Do(func() {
		global, err = New(ctx, cfg)
	})
	return global, err
}

// Default returns the global tracer, or a no-op tracer if not initialized.
func Default() *Tracer {
	if global == nil {
		return &Tracer{
			tracer: otel.Tracer(TracerName),
			config: DefaultConfig(),
		}
	}
	return global
}

// New creates a new Tracer with the provided configuration.
func New(ctx context.Context, cfg Config) (*Tracer, error) {
	if !cfg.Enabled || cfg.ExporterType == ExporterNone {
		return &Tracer{
			tracer: noop.NewTracerProvider().Tracer(TracerName),
			config: cfg,
		}, nil
	}

	// Create exporter
	exporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create resource without merging with Default() to avoid schema URL conflicts.
	// The default resource's schema URL may conflict with our semconv version.
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(Version),
			attribute.String("deployment.environment", cfg.Environment),
		),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	return &Tracer{
		tracer:   provider.Tracer(TracerName, trace.WithInstrumentationVersion(Version)),
		provider: provider,
		config:   cfg,
	}, nil
}

// createExporter creates the appropriate exporter based on configuration.
func createExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	switch cfg.ExporterType {
	case ExporterStdout:
		opts := []stdouttrace.Option{
			stdouttrace.WithPrettyPrint(),
		}
		if cfg.Output != nil {
			opts = append(opts, stdouttrace.WithWriter(cfg.Output))
		}
		return stdouttrace.New(opts...)

	case ExporterOTLP:
		opts := []otlptracehttp.Option{
			otlptracehttp.WithInsecure(),
		}
		if cfg.OTLPEndpoint != "" {
			opts = append(opts, otlptracehttp.WithEndpoint(cfg.OTLPEndpoint))
		}
		return otlptracehttp.New(ctx, opts...)

	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", cfg.ExporterType)
	}
}

// Shutdown gracefully shuts down the tracer provider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// Start starts a new span with the given name.
func (t *Tracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// --- Domain-specific span helpers ---

// WorkflowSpan represents a workflow execution span.
type WorkflowSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartWorkflowSpan starts a span for workflow execution.
func (t *Tracer) StartWorkflowSpan(ctx context.Context, skillID, skillName string) (context.Context, *WorkflowSpan) {
	ctx, span := t.tracer.Start(ctx, "workflow.execute",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("skill.id", skillID),
			attribute.String("skill.name", skillName),
		),
	)

	return ctx, &WorkflowSpan{span: span, ctx: ctx}
}

// SetPhaseCount sets the number of phases in the workflow.
func (ws *WorkflowSpan) SetPhaseCount(count int) {
	ws.span.SetAttributes(attribute.Int("workflow.phase_count", count))
}

// SetTotalTokens sets the total tokens used.
func (ws *WorkflowSpan) SetTotalTokens(input, output int) {
	ws.span.SetAttributes(
		attribute.Int("workflow.tokens.input", input),
		attribute.Int("workflow.tokens.output", output),
		attribute.Int("workflow.tokens.total", input+output),
	)
}

// SetCost sets the total cost.
func (ws *WorkflowSpan) SetCost(cost float64) {
	ws.span.SetAttributes(attribute.Float64("workflow.cost_usd", cost))
}

// SetCacheStats sets cache hit/miss statistics.
func (ws *WorkflowSpan) SetCacheStats(hits, misses int) {
	total := hits + misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	ws.span.SetAttributes(
		attribute.Int("workflow.cache.hits", hits),
		attribute.Int("workflow.cache.misses", misses),
		attribute.Float64("workflow.cache.hit_rate", hitRate),
	)
}

// End ends the workflow span with success status.
func (ws *WorkflowSpan) End() {
	ws.span.SetStatus(codes.Ok, "workflow completed successfully")
	ws.span.End()
}

// EndWithError ends the workflow span with error status.
func (ws *WorkflowSpan) EndWithError(err error) {
	ws.span.RecordError(err)
	ws.span.SetStatus(codes.Error, err.Error())
	ws.span.End()
}

// PhaseSpan represents a phase execution span.
type PhaseSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartPhaseSpan starts a span for phase execution.
func (t *Tracer) StartPhaseSpan(ctx context.Context, phaseID, phaseName string) (context.Context, *PhaseSpan) {
	ctx, span := t.tracer.Start(ctx, "phase.execute",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("phase.id", phaseID),
			attribute.String("phase.name", phaseName),
		),
	)

	return ctx, &PhaseSpan{span: span, ctx: ctx}
}

// SetProvider sets the provider used for this phase.
func (ps *PhaseSpan) SetProvider(provider, model string) {
	ps.span.SetAttributes(
		attribute.String("phase.provider", provider),
		attribute.String("phase.model", model),
	)
}

// SetTokens sets the token counts for this phase.
func (ps *PhaseSpan) SetTokens(input, output int) {
	ps.span.SetAttributes(
		attribute.Int("phase.tokens.input", input),
		attribute.Int("phase.tokens.output", output),
	)
}

// SetCost sets the cost for this phase.
func (ps *PhaseSpan) SetCost(cost float64) {
	ps.span.SetAttributes(attribute.Float64("phase.cost_usd", cost))
}

// SetCacheHit marks whether this was a cache hit.
func (ps *PhaseSpan) SetCacheHit(hit bool) {
	ps.span.SetAttributes(attribute.Bool("phase.cache_hit", hit))
}

// End ends the phase span with success status.
func (ps *PhaseSpan) End() {
	ps.span.SetStatus(codes.Ok, "phase completed successfully")
	ps.span.End()
}

// EndWithError ends the phase span with error status.
func (ps *PhaseSpan) EndWithError(err error) {
	ps.span.RecordError(err)
	ps.span.SetStatus(codes.Error, err.Error())
	ps.span.End()
}

// ProviderSpan represents a provider request span.
type ProviderSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartProviderSpan starts a span for provider request.
func (t *Tracer) StartProviderSpan(ctx context.Context, provider, model string) (context.Context, *ProviderSpan) {
	ctx, span := t.tracer.Start(ctx, "provider.request",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("provider.name", provider),
			attribute.String("provider.model", model),
		),
	)

	return ctx, &ProviderSpan{span: span, ctx: ctx}
}

// SetRequestTokens sets the input token estimate.
func (ps *ProviderSpan) SetRequestTokens(tokens int) {
	ps.span.SetAttributes(attribute.Int("provider.request.tokens", tokens))
}

// SetResponse sets response information.
func (ps *ProviderSpan) SetResponse(outputTokens int, finishReason string) {
	ps.span.SetAttributes(
		attribute.Int("provider.response.tokens", outputTokens),
		attribute.String("provider.response.finish_reason", finishReason),
	)
}

// End ends the provider span with success status.
func (ps *ProviderSpan) End() {
	ps.span.SetStatus(codes.Ok, "provider request completed")
	ps.span.End()
}

// EndWithError ends the provider span with error status.
func (ps *ProviderSpan) EndWithError(err error) {
	ps.span.RecordError(err)
	ps.span.SetStatus(codes.Error, err.Error())
	ps.span.End()
}

// AddEvent adds an event to the current span.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// SetAttribute sets an attribute on the current span.
func SetAttribute(ctx context.Context, key string, value any) {
	span := trace.SpanFromContext(ctx)
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	}
}
