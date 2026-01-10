package observability

import (
	"context"
	"log/slog"
	"time"
)

// Timer tracks the duration of operations and records metrics.
type Timer struct {
	operation string
	start     time.Time
	logger    *slog.Logger
	metrics   Metrics
	tags      []Tag
}

// StartTimer creates a new timer for the given operation.
func StartTimer(operation string) *Timer {
	return &Timer{
		operation: operation,
		start:     time.Now(),
	}
}

// WithLogger adds a logger to the timer for automatic logging on stop.
func (t *Timer) WithLogger(logger *slog.Logger) *Timer {
	t.logger = logger
	return t
}

// WithMetrics adds a metrics collector to the timer.
func (t *Timer) WithMetrics(metrics Metrics) *Timer {
	t.metrics = metrics
	return t
}

// WithTags adds tags to the timer for metrics labeling.
func (t *Timer) WithTags(tags ...Tag) *Timer {
	t.tags = append(t.tags, tags...)
	return t
}

// Stop records the operation duration.
func (t *Timer) Stop() time.Duration {
	duration := time.Since(t.start)

	if t.logger != nil {
		t.logger.Info("operation completed",
			"operation", t.operation,
			"duration_ms", duration.Milliseconds(),
		)
	}

	if t.metrics != nil {
		tags := append(t.tags, T("operation", t.operation))
		t.metrics.Timing(MetricOperationDuration, duration, tags...)
		t.metrics.Counter(MetricOperationTotal, 1, tags...)
	}

	return duration
}

// StopWithError records the operation duration with error status.
func (t *Timer) StopWithError(err error) time.Duration {
	duration := time.Since(t.start)

	if t.logger != nil {
		if err != nil {
			t.logger.Error("operation failed",
				"operation", t.operation,
				"duration_ms", duration.Milliseconds(),
				"error", err.Error(),
			)
		} else {
			t.logger.Info("operation completed",
				"operation", t.operation,
				"duration_ms", duration.Milliseconds(),
			)
		}
	}

	if t.metrics != nil {
		tags := append(t.tags, T("operation", t.operation))
		t.metrics.Timing(MetricOperationDuration, duration, tags...)
		t.metrics.Counter(MetricOperationTotal, 1, tags...)

		if err != nil {
			t.metrics.Counter(MetricOperationErrors, 1, tags...)
		}
	}

	return duration
}

// Elapsed returns the elapsed time without stopping the timer.
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// TimeOperation is a helper that times a function and records metrics.
func TimeOperation(ctx context.Context, logger *slog.Logger, metrics Metrics, operation string, fn func() error) error {
	timer := StartTimer(operation).
		WithLogger(logger).
		WithMetrics(metrics)

	err := fn()
	timer.StopWithError(err)
	return err
}

// TimeOperationResult is a helper that times a function that returns a value.
func TimeOperationResult[T any](ctx context.Context, logger *slog.Logger, metrics Metrics, operation string, fn func() (T, error)) (T, error) {
	timer := StartTimer(operation).
		WithLogger(logger).
		WithMetrics(metrics)

	result, err := fn()
	timer.StopWithError(err)
	return result, err
}

// Span represents a traced span of execution.
type Span struct {
	operation string
	start     time.Time
	parent    *Span
	attrs     map[string]any
}

// StartSpan creates a new span, optionally as a child of a parent span.
func StartSpan(ctx context.Context, operation string) (*Span, context.Context) {
	span := &Span{
		operation: operation,
		start:     time.Now(),
		attrs:     make(map[string]any),
	}

	// Check for parent span in context
	if parent, ok := ctx.Value(spanCtxKey).(*Span); ok {
		span.parent = parent
	}

	// Add span to context
	ctx = context.WithValue(ctx, spanCtxKey, span)
	return span, ctx
}

// SetAttribute adds an attribute to the span.
func (s *Span) SetAttribute(key string, value any) {
	s.attrs[key] = value
}

// End completes the span and returns its duration.
func (s *Span) End() time.Duration {
	return time.Since(s.start)
}

// Operation returns the span's operation name.
func (s *Span) Operation() string {
	return s.operation
}

// Attributes returns the span's attributes.
func (s *Span) Attributes() map[string]any {
	return s.attrs
}

type spanContextKey struct{}

var spanCtxKey = spanContextKey{}

// SpanFromContext extracts the current span from context.
func SpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanCtxKey).(*Span); ok {
		return span
	}
	return nil
}
