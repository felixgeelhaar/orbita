// Package observability provides structured logging, metrics collection,
// and request tracing utilities for Orbita.
package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// LogFormat specifies the output format for logs.
type LogFormat string

const (
	// LogFormatText outputs human-readable text logs.
	LogFormatText LogFormat = "text"
	// LogFormatJSON outputs JSON-structured logs for production.
	LogFormatJSON LogFormat = "json"
)

// LogLevel represents logging verbosity.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogConfig configures the logger.
type LogConfig struct {
	// Level sets the minimum log level.
	Level LogLevel
	// Format specifies the output format (text or json).
	Format LogFormat
	// Output is the writer for logs. Defaults to os.Stderr.
	Output io.Writer
	// AddSource adds source code location to logs.
	AddSource bool
	// ServiceName is included in all log entries.
	ServiceName string
	// ServiceVersion is included in all log entries.
	ServiceVersion string
}

// DefaultLogConfig returns sensible defaults for development.
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:          LogLevelInfo,
		Format:         LogFormatText,
		Output:         os.Stderr,
		AddSource:      false,
		ServiceName:    "orbita",
		ServiceVersion: "dev",
	}
}

// ProductionLogConfig returns recommended settings for production.
func ProductionLogConfig() LogConfig {
	return LogConfig{
		Level:          LogLevelInfo,
		Format:         LogFormatJSON,
		Output:         os.Stdout,
		AddSource:      true,
		ServiceName:    "orbita",
		ServiceVersion: "unknown",
	}
}

// NewLogger creates a new structured logger with the given configuration.
func NewLogger(cfg LogConfig) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}

	level := parseSlogLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	switch cfg.Format {
	case LogFormatJSON:
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	// Always wrap with attributeHandler for context-aware logging
	// This ensures correlation IDs are added from context
	attrs := []slog.Attr{}
	if cfg.ServiceName != "" {
		attrs = append(attrs, slog.String("service", cfg.ServiceName))
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, slog.String("version", cfg.ServiceVersion))
	}
	handler = &attributeHandler{
		handler: handler,
		attrs:   attrs,
	}

	return slog.New(handler)
}

// LoggerFromEnv creates a logger based on environment variables.
// ORBITA_LOG_LEVEL: debug, info, warn, error
// ORBITA_LOG_FORMAT: text, json
// ORBITA_ENV: production enables JSON format by default
func LoggerFromEnv() *slog.Logger {
	cfg := DefaultLogConfig()

	if env := os.Getenv("ORBITA_ENV"); env == "production" {
		cfg = ProductionLogConfig()
	}

	if level := os.Getenv("ORBITA_LOG_LEVEL"); level != "" {
		cfg.Level = LogLevel(level)
	}

	if format := os.Getenv("ORBITA_LOG_FORMAT"); format != "" {
		cfg.Format = LogFormat(format)
	}

	if version := os.Getenv("ORBITA_VERSION"); version != "" {
		cfg.ServiceVersion = version
	}

	return NewLogger(cfg)
}

func parseSlogLevel(level LogLevel) slog.Level {
	switch level {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// attributeHandler wraps a handler to add default attributes.
type attributeHandler struct {
	handler slog.Handler
	attrs   []slog.Attr
}

func (h *attributeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *attributeHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add default attributes
	for _, attr := range h.attrs {
		r.AddAttrs(attr)
	}

	// Add correlation ID from context if present
	if corrID := CorrelationIDFromContext(ctx); corrID != "" {
		r.AddAttrs(slog.String(CorrelationIDKey, corrID))
	}

	// Add request ID from context if present
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		r.AddAttrs(slog.String(RequestIDKey, reqID))
	}

	return h.handler.Handle(ctx, r)
}

func (h *attributeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &attributeHandler{
		handler: h.handler.WithAttrs(attrs),
		attrs:   h.attrs,
	}
}

func (h *attributeHandler) WithGroup(name string) slog.Handler {
	return &attributeHandler{
		handler: h.handler.WithGroup(name),
		attrs:   h.attrs,
	}
}

// LogOperation creates a logger with operation-specific attributes.
func LogOperation(logger *slog.Logger, operation string, attrs ...any) *slog.Logger {
	args := append([]any{"operation", operation}, attrs...)
	return logger.With(args...)
}

// LogDuration logs the duration of an operation.
func LogDuration(logger *slog.Logger, operation string, start time.Time) {
	duration := time.Since(start)
	logger.Info("operation completed",
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
	)
}
