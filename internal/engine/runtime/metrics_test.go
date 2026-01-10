package runtime

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsCollector(t *testing.T) {
	t.Run("creates new collector with empty metrics", func(t *testing.T) {
		collector := NewMetricsCollector()

		require.NotNil(t, collector)
		assert.NotNil(t, collector.metrics)
		assert.Empty(t, collector.metrics)
	})
}

func TestMetricsCollector_RecordOperation(t *testing.T) {
	t.Run("records successful operation", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordOperation("engine1", "CalculatePriority", 100*time.Millisecond, nil)

		metrics := collector.Get("engine1")
		require.NotNil(t, metrics)
		assert.Equal(t, "engine1", metrics.EngineID)
		assert.Equal(t, int64(1), metrics.TotalCalls)
		assert.Equal(t, int64(1), metrics.SuccessfulCalls)
		assert.Equal(t, int64(0), metrics.FailedCalls)
		assert.Equal(t, 100*time.Millisecond, metrics.TotalDuration)
		assert.Equal(t, 100*time.Millisecond, metrics.AverageDuration)
		assert.Equal(t, 100*time.Millisecond, metrics.MinDuration)
		assert.Equal(t, 100*time.Millisecond, metrics.MaxDuration)
		assert.Empty(t, metrics.LastError)
	})

	t.Run("records failed operation", func(t *testing.T) {
		collector := NewMetricsCollector()
		err := errors.New("connection timeout")

		collector.RecordOperation("engine1", "ScheduleTasks", 50*time.Millisecond, err)

		metrics := collector.Get("engine1")
		require.NotNil(t, metrics)
		assert.Equal(t, int64(1), metrics.TotalCalls)
		assert.Equal(t, int64(0), metrics.SuccessfulCalls)
		assert.Equal(t, int64(1), metrics.FailedCalls)
		assert.Equal(t, "connection timeout", metrics.LastError)
	})

	t.Run("updates min/max duration correctly", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op", 50*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op", 200*time.Millisecond, nil)

		metrics := collector.Get("engine1")
		assert.Equal(t, 50*time.Millisecond, metrics.MinDuration)
		assert.Equal(t, 200*time.Millisecond, metrics.MaxDuration)
	})

	t.Run("calculates average duration", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op", 200*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op", 300*time.Millisecond, nil)

		metrics := collector.Get("engine1")
		assert.Equal(t, 200*time.Millisecond, metrics.AverageDuration)
	})

	t.Run("updates last call timestamp", func(t *testing.T) {
		collector := NewMetricsCollector()
		before := time.Now()

		collector.RecordOperation("engine1", "op", time.Millisecond, nil)

		metrics := collector.Get("engine1")
		assert.True(t, metrics.LastCallAt.After(before) || metrics.LastCallAt.Equal(before))
	})

	t.Run("tracks operation-level metrics", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordOperation("engine1", "CalculatePriority", 100*time.Millisecond, nil)
		collector.RecordOperation("engine1", "BatchCalculate", 200*time.Millisecond, nil)
		collector.RecordOperation("engine1", "CalculatePriority", 50*time.Millisecond, errors.New("err"))

		metrics := collector.Get("engine1")
		require.Len(t, metrics.OperationMetrics, 2)

		calcMetrics := metrics.OperationMetrics["CalculatePriority"]
		assert.Equal(t, int64(2), calcMetrics.TotalCalls)
		assert.Equal(t, int64(1), calcMetrics.SuccessfulCalls)
		assert.Equal(t, int64(1), calcMetrics.FailedCalls)
		assert.Equal(t, 50*time.Millisecond, calcMetrics.MinDuration)
		assert.Equal(t, 100*time.Millisecond, calcMetrics.MaxDuration)

		batchMetrics := metrics.OperationMetrics["BatchCalculate"]
		assert.Equal(t, int64(1), batchMetrics.TotalCalls)
		assert.Equal(t, int64(1), batchMetrics.SuccessfulCalls)
	})

	t.Run("tracks multiple engines independently", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine2", "op", 200*time.Millisecond, nil)

		metrics1 := collector.Get("engine1")
		metrics2 := collector.Get("engine2")

		assert.Equal(t, int64(1), metrics1.TotalCalls)
		assert.Equal(t, int64(1), metrics2.TotalCalls)
		assert.Equal(t, 100*time.Millisecond, metrics1.TotalDuration)
		assert.Equal(t, 200*time.Millisecond, metrics2.TotalDuration)
	})
}

func TestMetricsCollector_RecordCircuitBreakerChange(t *testing.T) {
	t.Run("records circuit breaker state", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordCircuitBreakerChange("engine1", "closed")
		metrics := collector.Get("engine1")
		assert.Equal(t, "closed", metrics.CircuitBreakerState)

		collector.RecordCircuitBreakerChange("engine1", "open")
		metrics = collector.Get("engine1")
		assert.Equal(t, "open", metrics.CircuitBreakerState)

		collector.RecordCircuitBreakerChange("engine1", "half-open")
		metrics = collector.Get("engine1")
		assert.Equal(t, "half-open", metrics.CircuitBreakerState)
	})

	t.Run("creates metrics if engine not exists", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordCircuitBreakerChange("new-engine", "open")

		metrics := collector.Get("new-engine")
		require.NotNil(t, metrics)
		assert.Equal(t, "open", metrics.CircuitBreakerState)
	})
}

func TestMetricsCollector_RecordCircuitOpen(t *testing.T) {
	t.Run("increments circuit open count", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordCircuitOpen("engine1", "op")
		collector.RecordCircuitOpen("engine1", "op")
		collector.RecordCircuitOpen("engine1", "op")

		metrics := collector.Get("engine1")
		assert.Equal(t, int64(3), metrics.CircuitOpenCount)
	})

	t.Run("creates metrics if engine not exists", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordCircuitOpen("new-engine", "op")

		metrics := collector.Get("new-engine")
		require.NotNil(t, metrics)
		assert.Equal(t, int64(1), metrics.CircuitOpenCount)
	})
}

func TestMetricsCollector_Get(t *testing.T) {
	t.Run("returns nil for non-existent engine", func(t *testing.T) {
		collector := NewMetricsCollector()

		metrics := collector.Get("non-existent")

		assert.Nil(t, metrics)
	})

	t.Run("returns copy of metrics", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)

		metrics1 := collector.Get("engine1")
		metrics2 := collector.Get("engine1")

		// Should be different pointers (copies)
		assert.NotSame(t, metrics1, metrics2)
		assert.Equal(t, metrics1.TotalCalls, metrics2.TotalCalls)
	})

	t.Run("returned copy includes operation metrics", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op1", 100*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op2", 200*time.Millisecond, nil)

		metrics := collector.Get("engine1")

		require.Len(t, metrics.OperationMetrics, 2)
		assert.NotNil(t, metrics.OperationMetrics["op1"])
		assert.NotNil(t, metrics.OperationMetrics["op2"])
	})
}

func TestMetricsCollector_GetAll(t *testing.T) {
	t.Run("returns empty map when no metrics", func(t *testing.T) {
		collector := NewMetricsCollector()

		all := collector.GetAll()

		assert.Empty(t, all)
	})

	t.Run("returns all engine metrics", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine2", "op", 200*time.Millisecond, nil)
		collector.RecordOperation("engine3", "op", 300*time.Millisecond, nil)

		all := collector.GetAll()

		assert.Len(t, all, 3)
		assert.Contains(t, all, "engine1")
		assert.Contains(t, all, "engine2")
		assert.Contains(t, all, "engine3")
	})

	t.Run("returns copies of metrics", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)

		all := collector.GetAll()
		all["engine1"] = EngineMetrics{TotalCalls: 999}

		// Original should be unchanged
		metrics := collector.Get("engine1")
		assert.Equal(t, int64(1), metrics.TotalCalls)
	})
}

func TestMetricsCollector_Reset(t *testing.T) {
	t.Run("clears all metrics", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine2", "op", 200*time.Millisecond, nil)

		collector.Reset()

		assert.Nil(t, collector.Get("engine1"))
		assert.Nil(t, collector.Get("engine2"))
		assert.Empty(t, collector.GetAll())
	})
}

func TestMetricsCollector_ResetEngine(t *testing.T) {
	t.Run("clears metrics for specific engine", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine2", "op", 200*time.Millisecond, nil)

		collector.ResetEngine("engine1")

		assert.Nil(t, collector.Get("engine1"))
		assert.NotNil(t, collector.Get("engine2"))
	})

	t.Run("no-op for non-existent engine", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)

		collector.ResetEngine("non-existent")

		// Should not panic, and existing metrics should be unchanged
		assert.NotNil(t, collector.Get("engine1"))
	})
}

func TestMetricsCollector_TakeSnapshot(t *testing.T) {
	t.Run("creates snapshot of empty collector", func(t *testing.T) {
		collector := NewMetricsCollector()

		snapshot := collector.TakeSnapshot()

		assert.NotZero(t, snapshot.Timestamp)
		assert.Empty(t, snapshot.Engines)
		assert.Equal(t, 0, snapshot.Summary.TotalEngines)
		assert.Equal(t, int64(0), snapshot.Summary.TotalCalls)
		assert.Equal(t, float64(0), snapshot.Summary.SuccessRate)
	})

	t.Run("includes all engine metrics", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine2", "op", 200*time.Millisecond, nil)

		snapshot := collector.TakeSnapshot()

		assert.Len(t, snapshot.Engines, 2)
		assert.Contains(t, snapshot.Engines, "engine1")
		assert.Contains(t, snapshot.Engines, "engine2")
	})

	t.Run("calculates summary correctly", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, errors.New("err"))
		collector.RecordOperation("engine2", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine2", "op", 100*time.Millisecond, nil)

		snapshot := collector.TakeSnapshot()

		assert.Equal(t, 2, snapshot.Summary.TotalEngines)
		assert.Equal(t, int64(4), snapshot.Summary.TotalCalls)
		assert.Equal(t, int64(3), snapshot.Summary.TotalSuccessful)
		assert.Equal(t, int64(1), snapshot.Summary.TotalFailed)
		assert.Equal(t, 0.75, snapshot.Summary.SuccessRate) // 3/4 = 0.75
	})

	t.Run("tracks engines with open circuit", func(t *testing.T) {
		collector := NewMetricsCollector()
		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordCircuitBreakerChange("engine1", "open")
		collector.RecordOperation("engine2", "op", 100*time.Millisecond, nil)
		collector.RecordCircuitBreakerChange("engine2", "closed")
		collector.RecordOperation("engine3", "op", 100*time.Millisecond, nil)
		collector.RecordCircuitBreakerChange("engine3", "open")

		snapshot := collector.TakeSnapshot()

		assert.Len(t, snapshot.Summary.EnginesWithOpenCircuit, 2)
		assert.Contains(t, snapshot.Summary.EnginesWithOpenCircuit, "engine1")
		assert.Contains(t, snapshot.Summary.EnginesWithOpenCircuit, "engine3")
	})

	t.Run("snapshot timestamp is recent", func(t *testing.T) {
		collector := NewMetricsCollector()
		before := time.Now()

		snapshot := collector.TakeSnapshot()

		after := time.Now()
		assert.True(t, snapshot.Timestamp.After(before) || snapshot.Timestamp.Equal(before))
		assert.True(t, snapshot.Timestamp.Before(after) || snapshot.Timestamp.Equal(after))
	})
}

func TestMetricsCollector_InterfaceMethods(t *testing.T) {
	collector := NewMetricsCollector()

	t.Run("Counter does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			collector.Counter("test.counter", 1)
			collector.Counter("test.counter", 100, "tag1", "value1")
		})
	})

	t.Run("Gauge does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			collector.Gauge("test.gauge", 3.14)
			collector.Gauge("test.gauge", 99.9, "tag1", "value1")
		})
	})

	t.Run("Histogram does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			collector.Histogram("test.histogram", 42.0)
			collector.Histogram("test.histogram", 0.5, "tag1", "value1")
		})
	})

	t.Run("Timing does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			collector.Timing("test.timing", time.Second)
			collector.Timing("test.timing", 100*time.Millisecond, "tag1", "value1")
		})
	})
}

func TestMetricsCollector_Concurrency(t *testing.T) {
	t.Run("handles concurrent operations safely", func(t *testing.T) {
		collector := NewMetricsCollector()
		done := make(chan bool)

		// Start multiple goroutines recording metrics
		for i := 0; i < 10; i++ {
			go func(id int) {
				for j := 0; j < 100; j++ {
					collector.RecordOperation("engine1", "op", time.Millisecond, nil)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		metrics := collector.Get("engine1")
		assert.Equal(t, int64(1000), metrics.TotalCalls)
	})

	t.Run("handles concurrent reads and writes", func(t *testing.T) {
		collector := NewMetricsCollector()
		done := make(chan bool)

		// Writer goroutines
		for i := 0; i < 5; i++ {
			go func() {
				for j := 0; j < 50; j++ {
					collector.RecordOperation("engine1", "op", time.Millisecond, nil)
				}
				done <- true
			}()
		}

		// Reader goroutines
		for i := 0; i < 5; i++ {
			go func() {
				for j := 0; j < 50; j++ {
					collector.Get("engine1")
					collector.GetAll()
					collector.TakeSnapshot()
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should complete without race conditions
		metrics := collector.Get("engine1")
		assert.Equal(t, int64(250), metrics.TotalCalls)
	})
}

func TestOperationMetrics(t *testing.T) {
	t.Run("operation metrics track min/max correctly", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op", 50*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op", 200*time.Millisecond, nil)

		metrics := collector.Get("engine1")
		opMetrics := metrics.OperationMetrics["op"]

		assert.Equal(t, 50*time.Millisecond, opMetrics.MinDuration)
		assert.Equal(t, 200*time.Millisecond, opMetrics.MaxDuration)
	})

	t.Run("operation metrics calculate average", func(t *testing.T) {
		collector := NewMetricsCollector()

		collector.RecordOperation("engine1", "op", 100*time.Millisecond, nil)
		collector.RecordOperation("engine1", "op", 200*time.Millisecond, nil)

		metrics := collector.Get("engine1")
		opMetrics := metrics.OperationMetrics["op"]

		assert.Equal(t, 150*time.Millisecond, opMetrics.AverageDuration)
	})
}
