package sdk

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionContext(t *testing.T) {
	t.Run("creates context with all fields", func(t *testing.T) {
		ctx := context.Background()
		userID := uuid.New()
		engineID := "acme.scheduler"

		before := time.Now()
		execCtx := NewExecutionContext(ctx, userID, engineID)
		after := time.Now()

		require.NotNil(t, execCtx)
		assert.Equal(t, userID, execCtx.UserID)
		assert.Equal(t, engineID, execCtx.EngineID)
		assert.NotEmpty(t, execCtx.RequestID)
		assert.NotNil(t, execCtx.Logger)
		assert.NotNil(t, execCtx.Metrics)
		assert.True(t, execCtx.StartTime.After(before) || execCtx.StartTime.Equal(before))
		assert.True(t, execCtx.StartTime.Before(after) || execCtx.StartTime.Equal(after))
	})

	t.Run("generates unique request IDs", func(t *testing.T) {
		ctx := context.Background()
		userID := uuid.New()

		ctx1 := NewExecutionContext(ctx, userID, "engine1")
		ctx2 := NewExecutionContext(ctx, userID, "engine2")

		assert.NotEqual(t, ctx1.RequestID, ctx2.RequestID)
	})

	t.Run("uses default logger", func(t *testing.T) {
		ctx := context.Background()
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		assert.Equal(t, slog.Default(), execCtx.Logger)
	})

	t.Run("uses noop metrics recorder by default", func(t *testing.T) {
		ctx := context.Background()
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		_, ok := execCtx.Metrics.(*noopMetricsRecorder)
		assert.True(t, ok, "Expected noopMetricsRecorder")
	})
}

func TestExecutionContext_Context(t *testing.T) {
	t.Run("returns underlying context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "key", "value")
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		assert.Equal(t, ctx, execCtx.Context())
	})
}

func TestExecutionContext_ContextInterface(t *testing.T) {
	t.Run("Deadline returns from underlying context", func(t *testing.T) {
		deadline := time.Now().Add(time.Hour)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		execCtx := NewExecutionContext(ctx, uuid.New(), "test")
		gotDeadline, ok := execCtx.Deadline()

		assert.True(t, ok)
		assert.Equal(t, deadline, gotDeadline)
	})

	t.Run("Deadline returns false for context without deadline", func(t *testing.T) {
		ctx := context.Background()
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		_, ok := execCtx.Deadline()
		assert.False(t, ok)
	})

	t.Run("Done returns channel from underlying context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		done := execCtx.Done()
		assert.NotNil(t, done)

		// Channel should not be closed yet
		select {
		case <-done:
			t.Fatal("Done channel should not be closed yet")
		default:
			// Expected
		}

		cancel()

		// Channel should be closed now
		select {
		case <-done:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Done channel should be closed after cancel")
		}
	})

	t.Run("Err returns nil before cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		assert.NoError(t, execCtx.Err())
	})

	t.Run("Err returns error after cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		cancel()
		assert.ErrorIs(t, execCtx.Err(), context.Canceled)
	})

	t.Run("Value returns value from underlying context", func(t *testing.T) {
		type contextKey string
		key := contextKey("testKey")
		ctx := context.WithValue(context.Background(), key, "testValue")
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		assert.Equal(t, "testValue", execCtx.Value(key))
	})

	t.Run("Value returns nil for missing key", func(t *testing.T) {
		ctx := context.Background()
		execCtx := NewExecutionContext(ctx, uuid.New(), "test")

		assert.Nil(t, execCtx.Value("missing"))
	})
}

func TestExecutionContext_WithLogger(t *testing.T) {
	t.Run("sets custom logger with context fields", func(t *testing.T) {
		userID := uuid.New()
		engineID := "acme.priority"
		execCtx := NewExecutionContext(context.Background(), userID, engineID)

		customLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		result := execCtx.WithLogger(customLogger)

		assert.Same(t, execCtx, result) // Returns same instance for chaining
		assert.NotNil(t, execCtx.Logger)
		// The logger should be enhanced with context fields
		// We can't easily assert the fields directly, but we verify it doesn't panic
	})

	t.Run("returns same execution context for chaining", func(t *testing.T) {
		execCtx := NewExecutionContext(context.Background(), uuid.New(), "test")
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		result := execCtx.WithLogger(logger)

		assert.Same(t, execCtx, result)
	})
}

func TestExecutionContext_WithMetrics(t *testing.T) {
	t.Run("sets custom metrics recorder", func(t *testing.T) {
		execCtx := NewExecutionContext(context.Background(), uuid.New(), "test")

		customMetrics := &mockMetricsRecorder{}
		result := execCtx.WithMetrics(customMetrics)

		assert.Same(t, execCtx, result)
		assert.Same(t, customMetrics, execCtx.Metrics)
	})

	t.Run("replaces default noop recorder", func(t *testing.T) {
		execCtx := NewExecutionContext(context.Background(), uuid.New(), "test")

		_, isNoop := execCtx.Metrics.(*noopMetricsRecorder)
		assert.True(t, isNoop)

		customMetrics := &mockMetricsRecorder{}
		execCtx.WithMetrics(customMetrics)

		_, isNoop = execCtx.Metrics.(*noopMetricsRecorder)
		assert.False(t, isNoop)
	})
}

func TestExecutionContext_Elapsed(t *testing.T) {
	t.Run("returns duration since start", func(t *testing.T) {
		execCtx := NewExecutionContext(context.Background(), uuid.New(), "test")

		time.Sleep(10 * time.Millisecond)
		elapsed := execCtx.Elapsed()

		assert.GreaterOrEqual(t, elapsed, 10*time.Millisecond)
		assert.Less(t, elapsed, 100*time.Millisecond) // Should not take too long
	})

	t.Run("increases over time", func(t *testing.T) {
		execCtx := NewExecutionContext(context.Background(), uuid.New(), "test")

		elapsed1 := execCtx.Elapsed()
		time.Sleep(5 * time.Millisecond)
		elapsed2 := execCtx.Elapsed()

		assert.Greater(t, elapsed2, elapsed1)
	})
}

func TestNewNoopMetricsRecorder(t *testing.T) {
	t.Run("returns non-nil recorder", func(t *testing.T) {
		recorder := NewNoopMetricsRecorder()

		assert.NotNil(t, recorder)
	})

	t.Run("returns noopMetricsRecorder type", func(t *testing.T) {
		recorder := NewNoopMetricsRecorder()

		_, ok := recorder.(*noopMetricsRecorder)
		assert.True(t, ok)
	})
}

func TestNoopMetricsRecorder(t *testing.T) {
	recorder := NewNoopMetricsRecorder()

	t.Run("Counter does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			recorder.Counter("test.counter", 1)
			recorder.Counter("test.counter", 100, "tag1", "tag2")
		})
	})

	t.Run("Gauge does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			recorder.Gauge("test.gauge", 3.14)
			recorder.Gauge("test.gauge", 99.9, "tag1", "tag2")
		})
	})

	t.Run("Histogram does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			recorder.Histogram("test.histogram", 42.0)
			recorder.Histogram("test.histogram", 0.5, "tag1", "tag2")
		})
	})

	t.Run("Timing does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			recorder.Timing("test.timing", time.Second)
			recorder.Timing("test.timing", 100*time.Millisecond, "tag1", "tag2")
		})
	})
}

// mockMetricsRecorder is a simple mock for testing custom metrics injection.
type mockMetricsRecorder struct {
	counters   []string
	gauges     []string
	histograms []string
	timings    []string
}

func (m *mockMetricsRecorder) Counter(name string, _ int64, _ ...string) {
	m.counters = append(m.counters, name)
}

func (m *mockMetricsRecorder) Gauge(name string, _ float64, _ ...string) {
	m.gauges = append(m.gauges, name)
}

func (m *mockMetricsRecorder) Histogram(name string, _ float64, _ ...string) {
	m.histograms = append(m.histograms, name)
}

func (m *mockMetricsRecorder) Timing(name string, _ time.Duration, _ ...string) {
	m.timings = append(m.timings, name)
}
