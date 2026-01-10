package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("creates text logger", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := LogConfig{
			Level:  LogLevelInfo,
			Format: LogFormatText,
			Output: &buf,
		}

		logger := NewLogger(cfg)
		require.NotNil(t, logger)

		logger.Info("test message", "key", "value")
		output := buf.String()

		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "key=value")
	})

	t.Run("creates JSON logger", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := LogConfig{
			Level:  LogLevelInfo,
			Format: LogFormatJSON,
			Output: &buf,
		}

		logger := NewLogger(cfg)
		require.NotNil(t, logger)

		logger.Info("test message", "key", "value")
		output := buf.String()

		// Should be valid JSON
		var logEntry map[string]any
		err := json.Unmarshal([]byte(output), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test message", logEntry["msg"])
		assert.Equal(t, "value", logEntry["key"])
	})

	t.Run("respects log level", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := LogConfig{
			Level:  LogLevelWarn,
			Format: LogFormatText,
			Output: &buf,
		}

		logger := NewLogger(cfg)
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		output := buf.String()
		assert.NotContains(t, output, "debug message")
		assert.NotContains(t, output, "info message")
		assert.Contains(t, output, "warn message")
		assert.Contains(t, output, "error message")
	})

	t.Run("adds service attributes", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := LogConfig{
			Level:          LogLevelInfo,
			Format:         LogFormatJSON,
			Output:         &buf,
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
		}

		logger := NewLogger(cfg)
		logger.Info("test")

		var logEntry map[string]any
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test-service", logEntry["service"])
		assert.Equal(t, "1.0.0", logEntry["version"])
	})

	t.Run("adds correlation ID from context", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := LogConfig{
			Level:  LogLevelInfo,
			Format: LogFormatJSON,
			Output: &buf,
		}

		logger := NewLogger(cfg)
		ctx := WithCorrelationID(context.Background(), "test-correlation-id")

		// Use InfoContext to pass context
		logger.InfoContext(ctx, "test message")

		output := buf.String()

		// The correlation ID should be logged
		// Note: The custom handler extracts it from context during Handle()
		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "test-correlation-id")
	})
}

func TestDefaultLogConfig(t *testing.T) {
	cfg := DefaultLogConfig()

	assert.Equal(t, LogLevelInfo, cfg.Level)
	assert.Equal(t, LogFormatText, cfg.Format)
	assert.Equal(t, "orbita", cfg.ServiceName)
}

func TestProductionLogConfig(t *testing.T) {
	cfg := ProductionLogConfig()

	assert.Equal(t, LogLevelInfo, cfg.Level)
	assert.Equal(t, LogFormatJSON, cfg.Format)
	assert.True(t, cfg.AddSource)
	assert.Equal(t, "orbita", cfg.ServiceName)
}

func TestLogOperation(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	opLogger := LogOperation(logger, "test-operation", "extra", "attr")
	opLogger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "operation=test-operation")
	assert.Contains(t, output, "extra=attr")
}

func TestParseSlogLevel(t *testing.T) {
	tests := []struct {
		input    LogLevel
		expected slog.Level
	}{
		{LogLevelDebug, slog.LevelDebug},
		{LogLevelInfo, slog.LevelInfo},
		{LogLevelWarn, slog.LevelWarn},
		{LogLevelError, slog.LevelError},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := parseSlogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAttributeHandler(t *testing.T) {
	t.Run("WithAttrs returns new handler", func(t *testing.T) {
		var buf bytes.Buffer
		base := slog.NewJSONHandler(&buf, nil)
		handler := &attributeHandler{
			handler: base,
			attrs:   []slog.Attr{slog.String("default", "value")},
		}

		newHandler := handler.WithAttrs([]slog.Attr{slog.String("extra", "attr")})
		assert.NotEqual(t, handler, newHandler)
	})

	t.Run("WithGroup returns new handler", func(t *testing.T) {
		var buf bytes.Buffer
		base := slog.NewJSONHandler(&buf, nil)
		handler := &attributeHandler{
			handler: base,
			attrs:   []slog.Attr{},
		}

		newHandler := handler.WithGroup("group")
		assert.NotEqual(t, handler, newHandler)
	})

	t.Run("Enabled delegates to base handler", func(t *testing.T) {
		var buf bytes.Buffer
		base := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
		handler := &attributeHandler{
			handler: base,
			attrs:   []slog.Attr{},
		}

		assert.False(t, handler.Enabled(context.Background(), slog.LevelInfo))
		assert.True(t, handler.Enabled(context.Background(), slog.LevelWarn))
		assert.True(t, handler.Enabled(context.Background(), slog.LevelError))
	})
}

func TestLogDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Log a duration with a start time
	start := time.Now().Add(-100 * time.Millisecond)
	LogDuration(logger, "test-operation", start)

	output := buf.String()
	assert.Contains(t, output, "operation completed")
	assert.Contains(t, output, "test-operation")
	assert.Contains(t, output, "duration_ms")
}

func TestContextIntegration(t *testing.T) {
	t.Run("logs include correlation ID when set", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := LogConfig{
			Level:  LogLevelInfo,
			Format: LogFormatJSON,
			Output: &buf,
		}

		logger := NewLogger(cfg)

		ctx := context.Background()
		ctx = WithCorrelationID(ctx, "corr-123")
		ctx = WithRequestID(ctx, "req-456")

		logger.InfoContext(ctx, "test with context")

		output := buf.String()
		// The correlation and request IDs should be in the output
		assert.Contains(t, output, "corr-123")
		assert.Contains(t, output, "req-456")
	})
}
