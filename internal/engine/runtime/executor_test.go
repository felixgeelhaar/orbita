package runtime

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/registry"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEngine is a simple mock engine for testing.
type mockEngine struct {
	metadata   sdk.EngineMetadata
	engineType sdk.EngineType
	healthy    bool
}

func (m *mockEngine) Metadata() sdk.EngineMetadata {
	return m.metadata
}

func (m *mockEngine) Type() sdk.EngineType {
	return m.engineType
}

func (m *mockEngine) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{
		Schema:     "https://json-schema.org/draft/2020-12/schema",
		Properties: make(map[string]sdk.PropertySchema),
	}
}

func (m *mockEngine) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	return nil
}

func (m *mockEngine) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{
		Healthy: m.healthy,
		Message: "mock engine",
	}
}

func (m *mockEngine) Shutdown(ctx context.Context) error {
	return nil
}

func newMockEngine(id, name string, engineType sdk.EngineType) *mockEngine {
	return &mockEngine{
		metadata: sdk.EngineMetadata{
			ID:      id,
			Name:    name,
			Version: "1.0.0",
		},
		engineType: engineType,
		healthy:    true,
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNewExecutor(t *testing.T) {
	reg := registry.NewRegistry(testLogger())
	metrics := NewMetricsCollector()
	config := DefaultExecutorConfig()

	exec := NewExecutor(reg, metrics, testLogger(), config)
	assert.NotNil(t, exec)
}

func TestNewExecutorWithNilDependencies(t *testing.T) {
	reg := registry.NewRegistry(testLogger())
	config := DefaultExecutorConfig()

	// Should work with nil metrics and logger
	exec := NewExecutor(reg, nil, nil, config)
	assert.NotNil(t, exec)
}

func TestDefaultExecutorConfig(t *testing.T) {
	config := DefaultExecutorConfig()

	assert.True(t, config.CircuitBreakerEnabled)
	assert.Equal(t, uint32(3), config.MaxRequests)
	assert.Equal(t, 10*time.Second, config.Interval)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, uint32(5), config.FailureThreshold)
	assert.Equal(t, 10*time.Second, config.DefaultTimeout)
}

func TestExecutorHealthCheck(t *testing.T) {
	reg := registry.NewRegistry(testLogger())
	engine := newMockEngine("test.engine", "Test Engine", sdk.EngineTypePriority)
	require.NoError(t, reg.RegisterBuiltin(engine))

	metrics := NewMetricsCollector()
	config := DefaultExecutorConfig()
	exec := NewExecutor(reg, metrics, testLogger(), config)

	ctx := context.Background()
	health, err := exec.HealthCheck(ctx, "test.engine")
	require.NoError(t, err)
	assert.True(t, health.Healthy)
	assert.Equal(t, "mock engine", health.Message)
}

func TestExecutorHealthCheckNotFound(t *testing.T) {
	reg := registry.NewRegistry(testLogger())
	metrics := NewMetricsCollector()
	config := DefaultExecutorConfig()
	exec := NewExecutor(reg, metrics, testLogger(), config)

	ctx := context.Background()
	health, err := exec.HealthCheck(ctx, "nonexistent.engine")
	assert.Error(t, err)
	assert.False(t, health.Healthy)
}

func TestExecutorGetMetrics(t *testing.T) {
	reg := registry.NewRegistry(testLogger())
	metrics := NewMetricsCollector()
	config := DefaultExecutorConfig()
	exec := NewExecutor(reg, metrics, testLogger(), config)

	// Record some operations directly
	metrics.RecordOperation("test.engine", "test_op", 100*time.Millisecond, nil)

	allMetrics := exec.GetMetrics()
	assert.NotNil(t, allMetrics)
	assert.Contains(t, allMetrics, "test.engine")
}

func TestExecutorCircuitBreakerState(t *testing.T) {
	reg := registry.NewRegistry(testLogger())
	metrics := NewMetricsCollector()
	config := DefaultExecutorConfig()
	exec := NewExecutor(reg, metrics, testLogger(), config)

	// Initially no circuit breaker
	state := exec.GetCircuitBreakerState("test.engine")
	assert.Equal(t, "none", state)
}

func TestExecutorResetCircuitBreaker(t *testing.T) {
	reg := registry.NewRegistry(testLogger())
	metrics := NewMetricsCollector()
	config := DefaultExecutorConfig()
	exec := NewExecutor(reg, metrics, testLogger(), config)

	// Reset should not panic even if no breaker exists
	exec.ResetCircuitBreaker("test.engine")
}

func TestMetricsCollector(t *testing.T) {
	metrics := NewMetricsCollector()
	assert.NotNil(t, metrics)

	// Record operations (should not panic)
	metrics.RecordOperation("test.engine", "calculate_priority", 100*time.Millisecond, nil)
	metrics.RecordOperation("test.engine", "calculate_priority", 50*time.Millisecond, assert.AnError)

	// Get stats
	stats := metrics.Get("test.engine")
	assert.NotNil(t, stats)
	assert.Equal(t, int64(2), stats.TotalCalls)
	assert.Equal(t, int64(1), stats.SuccessfulCalls)
	assert.Equal(t, int64(1), stats.FailedCalls)
}

func TestMetricsCollectorGetNonexistent(t *testing.T) {
	metrics := NewMetricsCollector()

	stats := metrics.Get("nonexistent.engine")
	assert.Nil(t, stats)
}

func TestMetricsCollectorGetAll(t *testing.T) {
	metrics := NewMetricsCollector()

	metrics.RecordOperation("engine1", "op1", 100*time.Millisecond, nil)
	metrics.RecordOperation("engine2", "op2", 200*time.Millisecond, nil)
	metrics.RecordOperation("engine1", "op1", 150*time.Millisecond, assert.AnError)

	allStats := metrics.GetAll()
	assert.Len(t, allStats, 2)

	assert.Contains(t, allStats, "engine1")
	assert.Contains(t, allStats, "engine2")

	// Verify engine1 metrics
	e1 := allStats["engine1"]
	assert.Equal(t, int64(2), e1.TotalCalls)
	assert.Equal(t, int64(1), e1.SuccessfulCalls)
	assert.Equal(t, int64(1), e1.FailedCalls)

	// Verify engine2 metrics
	e2 := allStats["engine2"]
	assert.Equal(t, int64(1), e2.TotalCalls)
	assert.Equal(t, int64(1), e2.SuccessfulCalls)
	assert.Equal(t, int64(0), e2.FailedCalls)
}

func TestMetricsCollectorDurationTracking(t *testing.T) {
	metrics := NewMetricsCollector()

	metrics.RecordOperation("test.engine", "op", 100*time.Millisecond, nil)
	metrics.RecordOperation("test.engine", "op", 200*time.Millisecond, nil)
	metrics.RecordOperation("test.engine", "op", 50*time.Millisecond, nil)

	stats := metrics.Get("test.engine")
	assert.NotNil(t, stats)
	assert.Equal(t, 50*time.Millisecond, stats.MinDuration)
	assert.Equal(t, 200*time.Millisecond, stats.MaxDuration)
	assert.Equal(t, 350*time.Millisecond, stats.TotalDuration)
}

func TestMetricsCollectorOperationMetrics(t *testing.T) {
	metrics := NewMetricsCollector()

	metrics.RecordOperation("test.engine", "op1", 100*time.Millisecond, nil)
	metrics.RecordOperation("test.engine", "op2", 200*time.Millisecond, nil)
	metrics.RecordOperation("test.engine", "op1", 150*time.Millisecond, nil)

	stats := metrics.Get("test.engine")
	assert.NotNil(t, stats)
	assert.Len(t, stats.OperationMetrics, 2)

	op1 := stats.OperationMetrics["op1"]
	assert.NotNil(t, op1)
	assert.Equal(t, int64(2), op1.TotalCalls)

	op2 := stats.OperationMetrics["op2"]
	assert.NotNil(t, op2)
	assert.Equal(t, int64(1), op2.TotalCalls)
}

func TestMetricsCollectorCircuitBreaker(t *testing.T) {
	metrics := NewMetricsCollector()

	metrics.RecordCircuitBreakerChange("test.engine", "open")
	metrics.RecordCircuitOpen("test.engine", "op1")

	stats := metrics.Get("test.engine")
	assert.NotNil(t, stats)
	assert.Equal(t, "open", stats.CircuitBreakerState)
	assert.Equal(t, int64(1), stats.CircuitOpenCount)
}

func TestMetricsCollectorReset(t *testing.T) {
	metrics := NewMetricsCollector()

	metrics.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
	metrics.RecordOperation("engine2", "op", 200*time.Millisecond, nil)

	// Reset all
	metrics.Reset()

	allStats := metrics.GetAll()
	assert.Len(t, allStats, 0)
}

func TestMetricsCollectorResetEngine(t *testing.T) {
	metrics := NewMetricsCollector()

	metrics.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
	metrics.RecordOperation("engine2", "op", 200*time.Millisecond, nil)

	// Reset single engine
	metrics.ResetEngine("engine1")

	allStats := metrics.GetAll()
	assert.Len(t, allStats, 1)
	assert.NotContains(t, allStats, "engine1")
	assert.Contains(t, allStats, "engine2")
}

func TestMetricsCollectorTakeSnapshot(t *testing.T) {
	metrics := NewMetricsCollector()

	metrics.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
	metrics.RecordOperation("engine2", "op", 200*time.Millisecond, nil)
	metrics.RecordOperation("engine1", "op", 150*time.Millisecond, assert.AnError)

	snapshot := metrics.TakeSnapshot()

	assert.NotZero(t, snapshot.Timestamp)
	assert.Len(t, snapshot.Engines, 2)
	assert.Equal(t, 2, snapshot.Summary.TotalEngines)
	assert.Equal(t, int64(3), snapshot.Summary.TotalCalls)
	assert.Equal(t, int64(2), snapshot.Summary.TotalSuccessful)
	assert.Equal(t, int64(1), snapshot.Summary.TotalFailed)
}

func TestMetricsRecorderInterface(t *testing.T) {
	metrics := NewMetricsCollector()

	// Test MetricsRecorder interface methods don't panic
	metrics.Counter("test.counter", 1)
	metrics.Gauge("test.gauge", 1.5)
	metrics.Histogram("test.histogram", 100)
	metrics.Timing("test.timing", 100*time.Millisecond)
}
