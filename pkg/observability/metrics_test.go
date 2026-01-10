package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoopMetrics(t *testing.T) {
	m := NoopMetrics{}

	// Should not panic
	m.Counter("test", 1)
	m.Gauge("test", 1.0)
	m.Histogram("test", 1.0)
	m.Timing("test", time.Second)
}

func TestInMemoryMetrics(t *testing.T) {
	t.Run("Counter", func(t *testing.T) {
		m := NewInMemoryMetrics()

		m.Counter("requests", 1)
		m.Counter("requests", 1)
		m.Counter("requests", 1)

		assert.Equal(t, int64(3), m.GetCounter("requests"))
	})

	t.Run("Counter with tags", func(t *testing.T) {
		m := NewInMemoryMetrics()

		m.Counter("requests", 1, T("method", "GET"))
		m.Counter("requests", 1, T("method", "POST"))
		m.Counter("requests", 1, T("method", "GET"))

		assert.Equal(t, int64(2), m.GetCounter("requests", T("method", "GET")))
		assert.Equal(t, int64(1), m.GetCounter("requests", T("method", "POST")))
	})

	t.Run("Gauge", func(t *testing.T) {
		m := NewInMemoryMetrics()

		m.Gauge("temperature", 25.5)
		assert.Equal(t, 25.5, m.GetGauge("temperature"))

		m.Gauge("temperature", 30.0)
		assert.Equal(t, 30.0, m.GetGauge("temperature"))
	})

	t.Run("Gauge with tags", func(t *testing.T) {
		m := NewInMemoryMetrics()

		m.Gauge("connections", 10, T("pool", "primary"))
		m.Gauge("connections", 5, T("pool", "replica"))

		assert.Equal(t, 10.0, m.GetGauge("connections", T("pool", "primary")))
		assert.Equal(t, 5.0, m.GetGauge("connections", T("pool", "replica")))
	})

	t.Run("Histogram", func(t *testing.T) {
		m := NewInMemoryMetrics()

		m.Histogram("response_size", 100)
		m.Histogram("response_size", 200)
		m.Histogram("response_size", 150)

		values := m.GetHistogram("response_size")
		assert.Len(t, values, 3)
		assert.Contains(t, values, 100.0)
		assert.Contains(t, values, 200.0)
		assert.Contains(t, values, 150.0)
	})

	t.Run("Timing", func(t *testing.T) {
		m := NewInMemoryMetrics()

		m.Timing("query_duration", 100*time.Millisecond)
		m.Timing("query_duration", 200*time.Millisecond)

		timings := m.GetTimings("query_duration")
		assert.Len(t, timings, 2)
		assert.Contains(t, timings, 100*time.Millisecond)
		assert.Contains(t, timings, 200*time.Millisecond)
	})

	t.Run("Reset", func(t *testing.T) {
		m := NewInMemoryMetrics()

		m.Counter("test", 1)
		m.Gauge("test", 1.0)
		m.Histogram("test", 1.0)
		m.Timing("test", time.Second)

		m.Reset()

		assert.Equal(t, int64(0), m.GetCounter("test"))
		assert.Equal(t, 0.0, m.GetGauge("test"))
		assert.Empty(t, m.GetHistogram("test"))
		assert.Empty(t, m.GetTimings("test"))
	})
}

func TestTag(t *testing.T) {
	tag := T("key", "value")
	assert.Equal(t, "key", tag.Key)
	assert.Equal(t, "value", tag.Value)
}

func TestFormatKey(t *testing.T) {
	tests := []struct {
		name     string
		metric   string
		tags     []Tag
		expected string
	}{
		{
			name:     "no tags",
			metric:   "requests",
			tags:     nil,
			expected: "requests",
		},
		{
			name:     "single tag",
			metric:   "requests",
			tags:     []Tag{T("method", "GET")},
			expected: "requests:method=GET",
		},
		{
			name:     "multiple tags",
			metric:   "requests",
			tags:     []Tag{T("method", "GET"), T("status", "200")},
			expected: "requests:method=GET:status=200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatKey(tt.metric, tt.tags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetricConstants(t *testing.T) {
	// Verify metric names follow conventions
	assert.Equal(t, "orbita.operation.total", MetricOperationTotal)
	assert.Equal(t, "orbita.operation.duration", MetricOperationDuration)
	assert.Equal(t, "orbita.operation.errors", MetricOperationErrors)
	assert.Equal(t, "orbita.tasks.created", MetricTasksCreated)
	assert.Equal(t, "orbita.engine.executions", MetricEngineExecutions)
}
