// Package logging provides structured logging infrastructure for the skillrunner application.
// It wraps Go's standard log/slog package with context-aware logging, correlation IDs,
// and domain-specific log attributes.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// contextKey is used for storing logger-related values in context.
type contextKey string

const (
	// CorrelationIDKey is the context key for correlation IDs.
	CorrelationIDKey contextKey = "correlation_id"
	// WorkflowIDKey is the context key for workflow execution IDs.
	WorkflowIDKey contextKey = "workflow_id"
	// PhaseIDKey is the context key for phase IDs.
	PhaseIDKey contextKey = "phase_id"
	// ProviderKey is the context key for provider names.
	ProviderKey contextKey = "provider"
	// SkillIDKey is the context key for skill IDs.
	SkillIDKey contextKey = "skill_id"
)

// Level represents log levels.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Format represents log output formats.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Config holds logging configuration.
type Config struct {
	Level      Level
	Format     Format
	Output     io.Writer
	AddSource  bool
	TimeFormat string
}

// DefaultConfig returns sensible default logging configuration.
func DefaultConfig() Config {
	return Config{
		Level:      LevelInfo,
		Format:     FormatText,
		Output:     os.Stderr,
		AddSource:  false,
		TimeFormat: time.RFC3339,
	}
}

// Logger wraps slog.Logger with additional functionality for skillrunner.
type Logger struct {
	slogger *slog.Logger
	level   slog.Level
	mu      sync.RWMutex
}

// global is the package-level default logger.
var (
	global     *Logger
	globalOnce sync.Once
)

// Init initializes the global logger with the provided configuration.
func Init(cfg Config) *Logger {
	globalOnce.Do(func() {
		global = New(cfg)
	})
	return global
}

// Default returns the global logger, initializing it with defaults if necessary.
func Default() *Logger {
	if global == nil {
		Init(DefaultConfig())
	}
	return global
}

// New creates a new Logger with the provided configuration.
func New(cfg Config) *Logger {
	level := parseLevel(cfg.Level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format
			if a.Key == slog.TimeKey && cfg.TimeFormat != "" {
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.String(slog.TimeKey, t.Format(cfg.TimeFormat))
				}
			}
			return a
		},
	}

	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}

	switch cfg.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(output, opts)
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	return &Logger{
		slogger: slog.New(handler),
		level:   level,
	}
}

// parseLevel converts a Level to slog.Level.
func parseLevel(l Level) slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SetLevel dynamically changes the log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = parseLevel(level)
}

// With returns a new Logger with the given attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		slogger: l.slogger.With(args...),
		level:   l.level,
	}
}

// WithGroup returns a new Logger with the given group name.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		slogger: l.slogger.WithGroup(name),
		level:   l.level,
	}
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, args ...any) {
	l.slogger.Debug(msg, args...)
}

// Info logs at info level.
func (l *Logger) Info(msg string, args ...any) {
	l.slogger.Info(msg, args...)
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, args ...any) {
	l.slogger.Warn(msg, args...)
}

// Error logs at error level.
func (l *Logger) Error(msg string, args ...any) {
	l.slogger.Error(msg, args...)
}

// DebugContext logs at debug level with context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.slogger.DebugContext(ctx, msg, l.enrichArgs(ctx, args)...)
}

// InfoContext logs at info level with context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.slogger.InfoContext(ctx, msg, l.enrichArgs(ctx, args)...)
}

// WarnContext logs at warn level with context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.slogger.WarnContext(ctx, msg, l.enrichArgs(ctx, args)...)
}

// ErrorContext logs at error level with context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.slogger.ErrorContext(ctx, msg, l.enrichArgs(ctx, args)...)
}

// enrichArgs extracts context values and adds them as log attributes.
func (l *Logger) enrichArgs(ctx context.Context, args []any) []any {
	enriched := make([]any, 0, len(args)+10)

	// Extract standard context values
	if v := ctx.Value(CorrelationIDKey); v != nil {
		enriched = append(enriched, "correlation_id", v)
	}
	if v := ctx.Value(WorkflowIDKey); v != nil {
		enriched = append(enriched, "workflow_id", v)
	}
	if v := ctx.Value(PhaseIDKey); v != nil {
		enriched = append(enriched, "phase_id", v)
	}
	if v := ctx.Value(ProviderKey); v != nil {
		enriched = append(enriched, "provider", v)
	}
	if v := ctx.Value(SkillIDKey); v != nil {
		enriched = append(enriched, "skill_id", v)
	}

	enriched = append(enriched, args...)
	return enriched
}

// Underlying returns the underlying slog.Logger.
func (l *Logger) Underlying() *slog.Logger {
	return l.slogger
}

// --- Context helpers ---

// WithCorrelationID adds a correlation ID to the context.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, id)
}

// WithWorkflowID adds a workflow ID to the context.
func WithWorkflowID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, WorkflowIDKey, id)
}

// WithPhaseID adds a phase ID to the context.
func WithPhaseID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, PhaseIDKey, id)
}

// WithProvider adds a provider name to the context.
func WithProvider(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, ProviderKey, name)
}

// WithSkillID adds a skill ID to the context.
func WithSkillID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, SkillIDKey, id)
}

// CorrelationID extracts the correlation ID from context.
func CorrelationID(ctx context.Context) string {
	if v := ctx.Value(CorrelationIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// CorrelationIDFromContext is an alias for CorrelationID for semantic clarity.
func CorrelationIDFromContext(ctx context.Context) string {
	return CorrelationID(ctx)
}

// --- Domain-specific logging helpers ---

// LogWorkflowStart logs the start of a workflow execution.
func LogWorkflowStart(ctx context.Context, logger *Logger, skillID, skillName string) {
	logger.InfoContext(ctx, "workflow execution started",
		"skill_id", skillID,
		"skill_name", skillName,
	)
}

// LogWorkflowComplete logs the completion of a workflow execution.
func LogWorkflowComplete(ctx context.Context, logger *Logger, skillID string, duration time.Duration, totalTokens int) {
	logger.InfoContext(ctx, "workflow execution completed",
		"skill_id", skillID,
		"duration_ms", duration.Milliseconds(),
		"total_tokens", totalTokens,
	)
}

// LogWorkflowFailed logs a failed workflow execution.
func LogWorkflowFailed(ctx context.Context, logger *Logger, skillID string, err error, duration time.Duration) {
	logger.ErrorContext(ctx, "workflow execution failed",
		"skill_id", skillID,
		"error", err.Error(),
		"duration_ms", duration.Milliseconds(),
	)
}

// LogPhaseStart logs the start of a phase execution.
func LogPhaseStart(ctx context.Context, logger *Logger, phaseID, phaseName, model string) {
	logger.DebugContext(ctx, "phase execution started",
		"phase_id", phaseID,
		"phase_name", phaseName,
		"model", model,
	)
}

// LogPhaseComplete logs the completion of a phase execution.
func LogPhaseComplete(ctx context.Context, logger *Logger, phaseID string, inputTokens, outputTokens int, duration time.Duration, cacheHit bool) {
	logger.InfoContext(ctx, "phase execution completed",
		"phase_id", phaseID,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"duration_ms", duration.Milliseconds(),
		"cache_hit", cacheHit,
	)
}

// LogPhaseFailed logs a failed phase execution.
func LogPhaseFailed(ctx context.Context, logger *Logger, phaseID string, err error, duration time.Duration) {
	logger.ErrorContext(ctx, "phase execution failed",
		"phase_id", phaseID,
		"error", err.Error(),
		"duration_ms", duration.Milliseconds(),
	)
}

// LogProviderRequest logs an outgoing provider request.
func LogProviderRequest(ctx context.Context, logger *Logger, provider, model string, inputTokens int) {
	logger.DebugContext(ctx, "provider request",
		"provider", provider,
		"model", model,
		"input_tokens", inputTokens,
	)
}

// LogProviderResponse logs a provider response.
func LogProviderResponse(ctx context.Context, logger *Logger, provider, model string, outputTokens int, latency time.Duration) {
	logger.DebugContext(ctx, "provider response",
		"provider", provider,
		"model", model,
		"output_tokens", outputTokens,
		"latency_ms", latency.Milliseconds(),
	)
}

// LogCacheHit logs a cache hit.
func LogCacheHit(ctx context.Context, logger *Logger, key string, savedTokens int) {
	logger.DebugContext(ctx, "cache hit",
		"cache_key", key,
		"saved_tokens", savedTokens,
	)
}

// LogCacheMiss logs a cache miss.
func LogCacheMiss(ctx context.Context, logger *Logger, key string) {
	logger.DebugContext(ctx, "cache miss",
		"cache_key", key,
	)
}

// LogCostIncurred logs when cost is incurred.
func LogCostIncurred(ctx context.Context, logger *Logger, provider, model string, cost float64, inputTokens, outputTokens int) {
	logger.InfoContext(ctx, "cost incurred",
		"provider", provider,
		"model", model,
		"cost_usd", cost,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
	)
}

// LogPhaseError is an alias for LogPhaseFailed for semantic consistency.
func LogPhaseError(ctx context.Context, logger *Logger, phaseID string, err error) {
	logger.ErrorContext(ctx, "phase execution error",
		"phase_id", phaseID,
		"error", err.Error(),
	)
}

// LogWorkflowError is an alias for LogWorkflowFailed for semantic consistency.
func LogWorkflowError(ctx context.Context, logger *Logger, skillID string, err error, duration time.Duration) {
	LogWorkflowFailed(ctx, logger, skillID, err, duration)
}
