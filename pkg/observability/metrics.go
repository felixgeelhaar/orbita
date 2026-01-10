package observability

import (
	"sync"
	"time"
)

// Metrics provides an interface for recording application metrics.
type Metrics interface {
	// Counter increments a counter metric.
	Counter(name string, value int64, tags ...Tag)

	// Gauge sets a gauge metric to the given value.
	Gauge(name string, value float64, tags ...Tag)

	// Histogram records a value in a histogram.
	Histogram(name string, value float64, tags ...Tag)

	// Timing records a duration.
	Timing(name string, duration time.Duration, tags ...Tag)
}

// Tag represents a key-value pair for metric labeling.
type Tag struct {
	Key   string
	Value string
}

// T creates a new Tag.
func T(key, value string) Tag {
	return Tag{Key: key, Value: value}
}

// NoopMetrics is a no-op implementation of Metrics.
type NoopMetrics struct{}

func (NoopMetrics) Counter(name string, value int64, tags ...Tag)        {}
func (NoopMetrics) Gauge(name string, value float64, tags ...Tag)        {}
func (NoopMetrics) Histogram(name string, value float64, tags ...Tag)    {}
func (NoopMetrics) Timing(name string, duration time.Duration, tags ...Tag) {}

// InMemoryMetrics is an in-memory implementation for testing and development.
type InMemoryMetrics struct {
	mu         sync.RWMutex
	counters   map[string]int64
	gauges     map[string]float64
	histograms map[string][]float64
	timings    map[string][]time.Duration
}

// NewInMemoryMetrics creates a new in-memory metrics collector.
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
		timings:    make(map[string][]time.Duration),
	}
}

func (m *InMemoryMetrics) Counter(name string, value int64, tags ...Tag) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := formatKey(name, tags)
	m.counters[key] += value
}

func (m *InMemoryMetrics) Gauge(name string, value float64, tags ...Tag) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := formatKey(name, tags)
	m.gauges[key] = value
}

func (m *InMemoryMetrics) Histogram(name string, value float64, tags ...Tag) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := formatKey(name, tags)
	m.histograms[key] = append(m.histograms[key], value)
}

func (m *InMemoryMetrics) Timing(name string, duration time.Duration, tags ...Tag) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := formatKey(name, tags)
	m.timings[key] = append(m.timings[key], duration)
}

// GetCounter returns the current value of a counter.
func (m *InMemoryMetrics) GetCounter(name string, tags ...Tag) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[formatKey(name, tags)]
}

// GetGauge returns the current value of a gauge.
func (m *InMemoryMetrics) GetGauge(name string, tags ...Tag) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[formatKey(name, tags)]
}

// GetHistogram returns all recorded values for a histogram.
func (m *InMemoryMetrics) GetHistogram(name string, tags ...Tag) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.histograms[formatKey(name, tags)]
}

// GetTimings returns all recorded timings.
func (m *InMemoryMetrics) GetTimings(name string, tags ...Tag) []time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.timings[formatKey(name, tags)]
}

// Reset clears all recorded metrics.
func (m *InMemoryMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters = make(map[string]int64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string][]float64)
	m.timings = make(map[string][]time.Duration)
}

func formatKey(name string, tags []Tag) string {
	if len(tags) == 0 {
		return name
	}
	key := name
	for _, t := range tags {
		key += ":" + t.Key + "=" + t.Value
	}
	return key
}

// Standard metric names used throughout Orbita.
const (
	// Operation metrics
	MetricOperationTotal    = "orbita.operation.total"
	MetricOperationDuration = "orbita.operation.duration"
	MetricOperationErrors   = "orbita.operation.errors"

	// Task metrics
	MetricTasksCreated   = "orbita.tasks.created"
	MetricTasksCompleted = "orbita.tasks.completed"
	MetricTasksArchived  = "orbita.tasks.archived"

	// Habit metrics
	MetricHabitsCreated   = "orbita.habits.created"
	MetricHabitsCompleted = "orbita.habits.completed"

	// Meeting metrics
	MetricMeetingsCreated   = "orbita.meetings.created"
	MetricMeetingsScheduled = "orbita.meetings.scheduled"

	// Schedule metrics
	MetricBlocksScheduled = "orbita.schedule.blocks"
	MetricReschedules     = "orbita.schedule.reschedules"

	// Engine metrics
	MetricEngineExecutions = "orbita.engine.executions"
	MetricEngineDuration   = "orbita.engine.duration"
	MetricEngineErrors     = "orbita.engine.errors"

	// Orbit metrics
	MetricOrbitInitializations = "orbita.orbit.initializations"
	MetricOrbitToolCalls       = "orbita.orbit.tool_calls"
	MetricOrbitEvents          = "orbita.orbit.events"

	// Database metrics
	MetricDBQueries      = "orbita.db.queries"
	MetricDBQueryDuration = "orbita.db.query_duration"

	// Event bus metrics
	MetricEventsPublished = "orbita.events.published"
	MetricEventsConsumed  = "orbita.events.consumed"
)
