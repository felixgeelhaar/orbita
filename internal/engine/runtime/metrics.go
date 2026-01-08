package runtime

import (
	"sync"
	"time"
)

// MetricsCollector collects runtime metrics for engines.
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]*EngineMetrics
}

// EngineMetrics contains metrics for a single engine.
type EngineMetrics struct {
	// EngineID is the engine identifier.
	EngineID string `json:"engine_id"`

	// TotalCalls is the total number of calls to this engine.
	TotalCalls int64 `json:"total_calls"`

	// SuccessfulCalls is the number of successful calls.
	SuccessfulCalls int64 `json:"successful_calls"`

	// FailedCalls is the number of failed calls.
	FailedCalls int64 `json:"failed_calls"`

	// TotalDuration is the total time spent in engine operations.
	TotalDuration time.Duration `json:"total_duration"`

	// AverageDuration is the average operation duration.
	AverageDuration time.Duration `json:"average_duration"`

	// MinDuration is the minimum operation duration.
	MinDuration time.Duration `json:"min_duration"`

	// MaxDuration is the maximum operation duration.
	MaxDuration time.Duration `json:"max_duration"`

	// LastCallAt is the timestamp of the last call.
	LastCallAt time.Time `json:"last_call_at"`

	// LastError is the last error message, if any.
	LastError string `json:"last_error,omitempty"`

	// CircuitBreakerState is the current circuit breaker state.
	CircuitBreakerState string `json:"circuit_breaker_state"`

	// CircuitOpenCount is the number of times circuit breaker opened.
	CircuitOpenCount int64 `json:"circuit_open_count"`

	// OperationMetrics contains per-operation metrics.
	OperationMetrics map[string]*OperationMetrics `json:"operation_metrics"`
}

// OperationMetrics contains metrics for a specific operation.
type OperationMetrics struct {
	// Operation is the operation name.
	Operation string `json:"operation"`

	// TotalCalls is the total number of calls for this operation.
	TotalCalls int64 `json:"total_calls"`

	// SuccessfulCalls is the number of successful calls.
	SuccessfulCalls int64 `json:"successful_calls"`

	// FailedCalls is the number of failed calls.
	FailedCalls int64 `json:"failed_calls"`

	// TotalDuration is the total time spent in this operation.
	TotalDuration time.Duration `json:"total_duration"`

	// AverageDuration is the average operation duration.
	AverageDuration time.Duration `json:"average_duration"`

	// MinDuration is the minimum operation duration.
	MinDuration time.Duration `json:"min_duration"`

	// MaxDuration is the maximum operation duration.
	MaxDuration time.Duration `json:"max_duration"`

	// LastCallAt is the timestamp of the last call.
	LastCallAt time.Time `json:"last_call_at"`
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*EngineMetrics),
	}
}

// RecordOperation records metrics for an engine operation.
func (m *MetricsCollector) RecordOperation(engineID, operation string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.getOrCreateMetrics(engineID)
	now := time.Now()

	// Update engine-level metrics
	metrics.TotalCalls++
	metrics.TotalDuration += duration
	metrics.LastCallAt = now

	if err != nil {
		metrics.FailedCalls++
		metrics.LastError = err.Error()
	} else {
		metrics.SuccessfulCalls++
	}

	// Update duration stats
	if metrics.TotalCalls == 1 {
		metrics.MinDuration = duration
		metrics.MaxDuration = duration
	} else {
		if duration < metrics.MinDuration {
			metrics.MinDuration = duration
		}
		if duration > metrics.MaxDuration {
			metrics.MaxDuration = duration
		}
	}
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.TotalCalls)

	// Update operation-level metrics
	opMetrics := m.getOrCreateOperationMetrics(metrics, operation)
	opMetrics.TotalCalls++
	opMetrics.TotalDuration += duration
	opMetrics.LastCallAt = now

	if err != nil {
		opMetrics.FailedCalls++
	} else {
		opMetrics.SuccessfulCalls++
	}

	// Update operation duration stats
	if opMetrics.TotalCalls == 1 {
		opMetrics.MinDuration = duration
		opMetrics.MaxDuration = duration
	} else {
		if duration < opMetrics.MinDuration {
			opMetrics.MinDuration = duration
		}
		if duration > opMetrics.MaxDuration {
			opMetrics.MaxDuration = duration
		}
	}
	opMetrics.AverageDuration = opMetrics.TotalDuration / time.Duration(opMetrics.TotalCalls)
}

// RecordCircuitBreakerChange records a circuit breaker state change.
func (m *MetricsCollector) RecordCircuitBreakerChange(engineID, state string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.getOrCreateMetrics(engineID)
	metrics.CircuitBreakerState = state
}

// RecordCircuitOpen records when a circuit breaker opens.
func (m *MetricsCollector) RecordCircuitOpen(engineID, operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.getOrCreateMetrics(engineID)
	metrics.CircuitOpenCount++
}

// Get returns metrics for a specific engine.
func (m *MetricsCollector) Get(engineID string) *EngineMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metrics, exists := m.metrics[engineID]; exists {
		return m.copyMetrics(metrics)
	}
	return nil
}

// GetAll returns metrics for all engines.
func (m *MetricsCollector) GetAll() map[string]EngineMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]EngineMetrics, len(m.metrics))
	for id, metrics := range m.metrics {
		result[id] = *m.copyMetrics(metrics)
	}
	return result
}

// Reset resets all metrics.
func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = make(map[string]*EngineMetrics)
}

// ResetEngine resets metrics for a specific engine.
func (m *MetricsCollector) ResetEngine(engineID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.metrics, engineID)
}

// Counter increments a counter metric (implements sdk.MetricsRecorder).
func (m *MetricsCollector) Counter(name string, value int64, tags ...string) {
	// For now, counters are tracked as part of engine metrics
	// This method satisfies the MetricsRecorder interface for engine use
}

// Gauge sets a gauge metric value (implements sdk.MetricsRecorder).
func (m *MetricsCollector) Gauge(name string, value float64, tags ...string) {
	// Gauge metrics can be extended to track custom engine gauges
}

// Histogram records a value in a histogram (implements sdk.MetricsRecorder).
func (m *MetricsCollector) Histogram(name string, value float64, tags ...string) {
	// Histogram metrics can be extended for distribution tracking
}

// Timing records a duration (implements sdk.MetricsRecorder).
func (m *MetricsCollector) Timing(name string, duration time.Duration, tags ...string) {
	// Timing metrics are primarily tracked via RecordOperation
}

// getOrCreateMetrics gets or creates metrics for an engine.
func (m *MetricsCollector) getOrCreateMetrics(engineID string) *EngineMetrics {
	if metrics, exists := m.metrics[engineID]; exists {
		return metrics
	}

	metrics := &EngineMetrics{
		EngineID:         engineID,
		OperationMetrics: make(map[string]*OperationMetrics),
	}
	m.metrics[engineID] = metrics
	return metrics
}

// getOrCreateOperationMetrics gets or creates operation metrics.
func (m *MetricsCollector) getOrCreateOperationMetrics(metrics *EngineMetrics, operation string) *OperationMetrics {
	if opMetrics, exists := metrics.OperationMetrics[operation]; exists {
		return opMetrics
	}

	opMetrics := &OperationMetrics{
		Operation: operation,
	}
	metrics.OperationMetrics[operation] = opMetrics
	return opMetrics
}

// copyMetrics creates a copy of engine metrics to avoid race conditions.
func (m *MetricsCollector) copyMetrics(metrics *EngineMetrics) *EngineMetrics {
	copy := &EngineMetrics{
		EngineID:            metrics.EngineID,
		TotalCalls:          metrics.TotalCalls,
		SuccessfulCalls:     metrics.SuccessfulCalls,
		FailedCalls:         metrics.FailedCalls,
		TotalDuration:       metrics.TotalDuration,
		AverageDuration:     metrics.AverageDuration,
		MinDuration:         metrics.MinDuration,
		MaxDuration:         metrics.MaxDuration,
		LastCallAt:          metrics.LastCallAt,
		LastError:           metrics.LastError,
		CircuitBreakerState: metrics.CircuitBreakerState,
		CircuitOpenCount:    metrics.CircuitOpenCount,
		OperationMetrics:    make(map[string]*OperationMetrics, len(metrics.OperationMetrics)),
	}

	for op, opMetrics := range metrics.OperationMetrics {
		copy.OperationMetrics[op] = &OperationMetrics{
			Operation:       opMetrics.Operation,
			TotalCalls:      opMetrics.TotalCalls,
			SuccessfulCalls: opMetrics.SuccessfulCalls,
			FailedCalls:     opMetrics.FailedCalls,
			TotalDuration:   opMetrics.TotalDuration,
			AverageDuration: opMetrics.AverageDuration,
			MinDuration:     opMetrics.MinDuration,
			MaxDuration:     opMetrics.MaxDuration,
			LastCallAt:      opMetrics.LastCallAt,
		}
	}

	return copy
}

// Snapshot contains a point-in-time snapshot of all metrics.
type Snapshot struct {
	// Timestamp is when the snapshot was taken.
	Timestamp time.Time `json:"timestamp"`

	// Engines contains metrics for all engines.
	Engines map[string]EngineMetrics `json:"engines"`

	// Summary contains aggregated summary statistics.
	Summary SnapshotSummary `json:"summary"`
}

// SnapshotSummary contains aggregated summary statistics.
type SnapshotSummary struct {
	// TotalEngines is the number of engines with metrics.
	TotalEngines int `json:"total_engines"`

	// TotalCalls is the total number of calls across all engines.
	TotalCalls int64 `json:"total_calls"`

	// TotalSuccessful is the total successful calls.
	TotalSuccessful int64 `json:"total_successful"`

	// TotalFailed is the total failed calls.
	TotalFailed int64 `json:"total_failed"`

	// SuccessRate is the overall success rate (0-1).
	SuccessRate float64 `json:"success_rate"`

	// EnginesWithOpenCircuit lists engines with open circuit breakers.
	EnginesWithOpenCircuit []string `json:"engines_with_open_circuit"`
}

// TakeSnapshot creates a snapshot of current metrics.
func (m *MetricsCollector) TakeSnapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := Snapshot{
		Timestamp: time.Now(),
		Engines:   make(map[string]EngineMetrics, len(m.metrics)),
	}

	var totalCalls, totalSuccessful, totalFailed int64
	var enginesWithOpenCircuit []string

	for id, metrics := range m.metrics {
		snapshot.Engines[id] = *m.copyMetrics(metrics)

		totalCalls += metrics.TotalCalls
		totalSuccessful += metrics.SuccessfulCalls
		totalFailed += metrics.FailedCalls

		if metrics.CircuitBreakerState == "open" {
			enginesWithOpenCircuit = append(enginesWithOpenCircuit, id)
		}
	}

	snapshot.Summary = SnapshotSummary{
		TotalEngines:           len(m.metrics),
		TotalCalls:             totalCalls,
		TotalSuccessful:        totalSuccessful,
		TotalFailed:            totalFailed,
		EnginesWithOpenCircuit: enginesWithOpenCircuit,
	}

	if totalCalls > 0 {
		snapshot.Summary.SuccessRate = float64(totalSuccessful) / float64(totalCalls)
	}

	return snapshot
}
