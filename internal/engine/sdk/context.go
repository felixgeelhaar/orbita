package sdk

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ExecutionContext provides runtime context for engine operations.
// It wraps the standard context.Context and adds engine-specific utilities.
type ExecutionContext struct {
	// ctx is the underlying context.
	ctx context.Context

	// UserID is the user on whose behalf this operation is running.
	UserID uuid.UUID

	// EngineID identifies which engine is executing.
	EngineID string

	// RequestID is a unique identifier for this execution request.
	RequestID string

	// Logger is a structured logger for the engine to use.
	Logger *slog.Logger

	// Metrics allows engines to record custom metrics.
	Metrics MetricsRecorder

	// StartTime is when this execution started.
	StartTime time.Time
}

// NewExecutionContext creates a new execution context.
func NewExecutionContext(ctx context.Context, userID uuid.UUID, engineID string) *ExecutionContext {
	return &ExecutionContext{
		ctx:       ctx,
		UserID:    userID,
		EngineID:  engineID,
		RequestID: uuid.New().String(),
		Logger:    slog.Default(),
		Metrics:   &noopMetricsRecorder{},
		StartTime: time.Now(),
	}
}

// Context returns the underlying context.Context.
func (ec *ExecutionContext) Context() context.Context {
	return ec.ctx
}

// Deadline returns the time when work done on behalf of this context should be canceled.
func (ec *ExecutionContext) Deadline() (deadline time.Time, ok bool) {
	return ec.ctx.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this context should be canceled.
func (ec *ExecutionContext) Done() <-chan struct{} {
	return ec.ctx.Done()
}

// Err returns a non-nil error value after Done is closed.
func (ec *ExecutionContext) Err() error {
	return ec.ctx.Err()
}

// Value returns the value associated with this context for key.
func (ec *ExecutionContext) Value(key any) any {
	return ec.ctx.Value(key)
}

// WithLogger sets a custom logger for this context.
func (ec *ExecutionContext) WithLogger(logger *slog.Logger) *ExecutionContext {
	ec.Logger = logger.With(
		"engine_id", ec.EngineID,
		"user_id", ec.UserID.String(),
		"request_id", ec.RequestID,
	)
	return ec
}

// WithMetrics sets a custom metrics recorder for this context.
func (ec *ExecutionContext) WithMetrics(metrics MetricsRecorder) *ExecutionContext {
	ec.Metrics = metrics
	return ec
}

// Elapsed returns the duration since this execution started.
func (ec *ExecutionContext) Elapsed() time.Duration {
	return time.Since(ec.StartTime)
}

// MetricsRecorder allows engines to record custom metrics.
type MetricsRecorder interface {
	// Counter increments a counter metric.
	Counter(name string, value int64, tags ...string)

	// Gauge sets a gauge metric value.
	Gauge(name string, value float64, tags ...string)

	// Histogram records a value in a histogram.
	Histogram(name string, value float64, tags ...string)

	// Timing records a duration.
	Timing(name string, duration time.Duration, tags ...string)
}

// noopMetricsRecorder is a no-op implementation of MetricsRecorder.
type noopMetricsRecorder struct{}

func (n *noopMetricsRecorder) Counter(_ string, _ int64, _ ...string)        {}
func (n *noopMetricsRecorder) Gauge(_ string, _ float64, _ ...string)        {}
func (n *noopMetricsRecorder) Histogram(_ string, _ float64, _ ...string)    {}
func (n *noopMetricsRecorder) Timing(_ string, _ time.Duration, _ ...string) {}

// NewNoopMetricsRecorder returns a no-op metrics recorder.
func NewNoopMetricsRecorder() MetricsRecorder {
	return &noopMetricsRecorder{}
}
