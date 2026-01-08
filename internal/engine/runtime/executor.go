// Package runtime provides execution management for engines with circuit breakers and metrics.
package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/registry"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/sony/gobreaker/v2"
)

// Executor manages engine execution with circuit breakers and metrics.
type Executor struct {
	registry *registry.Registry
	breakers map[string]*gobreaker.CircuitBreaker[any]
	metrics  *MetricsCollector
	logger   *slog.Logger
	config   ExecutorConfig
}

// ExecutorConfig configures the executor behavior.
type ExecutorConfig struct {
	// CircuitBreakerEnabled enables circuit breakers.
	CircuitBreakerEnabled bool

	// MaxRequests is the maximum number of requests allowed in half-open state.
	MaxRequests uint32

	// Interval is the cyclic period of the closed state.
	Interval time.Duration

	// Timeout is the period of the open state.
	Timeout time.Duration

	// ReadyToTrip determines when to trip the circuit breaker.
	// Default: 5 consecutive failures.
	FailureThreshold uint32

	// DefaultTimeout is the default timeout for engine operations.
	DefaultTimeout time.Duration
}

// DefaultExecutorConfig returns a sensible default configuration.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		CircuitBreakerEnabled: true,
		MaxRequests:           3,
		Interval:              10 * time.Second,
		Timeout:               30 * time.Second,
		FailureThreshold:      5,
		DefaultTimeout:        10 * time.Second,
	}
}

// NewExecutor creates a new engine executor.
func NewExecutor(reg *registry.Registry, metrics *MetricsCollector, logger *slog.Logger, config ExecutorConfig) *Executor {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NewMetricsCollector()
	}
	return &Executor{
		registry: reg,
		breakers: make(map[string]*gobreaker.CircuitBreaker[any]),
		metrics:  metrics,
		logger:   logger,
		config:   config,
	}
}

// getBreaker returns the circuit breaker for an engine, creating it if needed.
func (e *Executor) getBreaker(engineID string) *gobreaker.CircuitBreaker[any] {
	if !e.config.CircuitBreakerEnabled {
		return nil
	}

	if breaker, exists := e.breakers[engineID]; exists {
		return breaker
	}

	settings := gobreaker.Settings{
		Name:        engineID,
		MaxRequests: e.config.MaxRequests,
		Interval:    e.config.Interval,
		Timeout:     e.config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= e.config.FailureThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			e.logger.Info("circuit breaker state changed",
				"engine_id", name,
				"from", from.String(),
				"to", to.String(),
			)
			e.metrics.RecordCircuitBreakerChange(name, to.String())
		},
	}

	breaker := gobreaker.NewCircuitBreaker[any](settings)
	e.breakers[engineID] = breaker
	return breaker
}

// execute runs an operation with circuit breaker protection.
func (e *Executor) execute(ctx context.Context, engineID string, operation string, fn func() (any, error)) (any, error) {
	start := time.Now()

	breaker := e.getBreaker(engineID)
	var result any
	var err error

	if breaker != nil {
		result, err = breaker.Execute(func() (any, error) {
			return fn()
		})
		if err == gobreaker.ErrOpenState {
			e.metrics.RecordCircuitOpen(engineID, operation)
			return nil, sdk.ErrCircuitOpen
		}
	} else {
		result, err = fn()
	}

	duration := time.Since(start)
	e.metrics.RecordOperation(engineID, operation, duration, err)

	return result, err
}

// createContext creates an ExecutionContext for an operation.
func (e *Executor) createContext(ctx context.Context, userID uuid.UUID, engineID string) *sdk.ExecutionContext {
	execCtx := sdk.NewExecutionContext(ctx, userID, engineID)
	execCtx.WithLogger(e.logger)
	execCtx.WithMetrics(e.metrics)
	return execCtx
}

// ExecuteScheduler executes a scheduler engine operation.
func (e *Executor) ExecuteScheduler(ctx context.Context, engineID string, userID uuid.UUID, input types.ScheduleTasksInput) (*types.ScheduleTasksOutput, error) {
	engine, err := e.registry.Get(ctx, engineID)
	if err != nil {
		return nil, err
	}

	scheduler, ok := engine.(types.SchedulerEngine)
	if !ok {
		return nil, fmt.Errorf("engine %s is not a scheduler engine", engineID)
	}

	execCtx := e.createContext(ctx, userID, engineID)

	result, err := e.execute(ctx, engineID, "schedule_tasks", func() (any, error) {
		return scheduler.ScheduleTasks(execCtx, input)
	})
	if err != nil {
		return nil, err
	}

	return result.(*types.ScheduleTasksOutput), nil
}

// ExecuteFindSlot executes a find slot operation.
func (e *Executor) ExecuteFindSlot(ctx context.Context, engineID string, userID uuid.UUID, input types.FindSlotInput) (*types.TimeSlot, error) {
	engine, err := e.registry.Get(ctx, engineID)
	if err != nil {
		return nil, err
	}

	scheduler, ok := engine.(types.SchedulerEngine)
	if !ok {
		return nil, fmt.Errorf("engine %s is not a scheduler engine", engineID)
	}

	execCtx := e.createContext(ctx, userID, engineID)

	result, err := e.execute(ctx, engineID, "find_optimal_slot", func() (any, error) {
		return scheduler.FindOptimalSlot(execCtx, input)
	})
	if err != nil {
		return nil, err
	}

	return result.(*types.TimeSlot), nil
}

// ExecutePriority executes a priority calculation.
func (e *Executor) ExecutePriority(ctx context.Context, engineID string, userID uuid.UUID, input types.PriorityInput) (*types.PriorityOutput, error) {
	engine, err := e.registry.Get(ctx, engineID)
	if err != nil {
		return nil, err
	}

	priority, ok := engine.(types.PriorityEngine)
	if !ok {
		return nil, fmt.Errorf("engine %s is not a priority engine", engineID)
	}

	execCtx := e.createContext(ctx, userID, engineID)

	result, err := e.execute(ctx, engineID, "calculate_priority", func() (any, error) {
		return priority.CalculatePriority(execCtx, input)
	})
	if err != nil {
		return nil, err
	}

	return result.(*types.PriorityOutput), nil
}

// ExecuteBatchPriority executes batch priority calculation.
func (e *Executor) ExecuteBatchPriority(ctx context.Context, engineID string, userID uuid.UUID, inputs []types.PriorityInput) ([]types.PriorityOutput, error) {
	engine, err := e.registry.Get(ctx, engineID)
	if err != nil {
		return nil, err
	}

	priority, ok := engine.(types.PriorityEngine)
	if !ok {
		return nil, fmt.Errorf("engine %s is not a priority engine", engineID)
	}

	execCtx := e.createContext(ctx, userID, engineID)

	result, err := e.execute(ctx, engineID, "batch_calculate", func() (any, error) {
		return priority.BatchCalculate(execCtx, inputs)
	})
	if err != nil {
		return nil, err
	}

	return result.([]types.PriorityOutput), nil
}

// ExecuteClassify executes a classification.
func (e *Executor) ExecuteClassify(ctx context.Context, engineID string, userID uuid.UUID, input types.ClassifyInput) (*types.ClassifyOutput, error) {
	engine, err := e.registry.Get(ctx, engineID)
	if err != nil {
		return nil, err
	}

	classifier, ok := engine.(types.ClassifierEngine)
	if !ok {
		return nil, fmt.Errorf("engine %s is not a classifier engine", engineID)
	}

	execCtx := e.createContext(ctx, userID, engineID)

	result, err := e.execute(ctx, engineID, "classify", func() (any, error) {
		return classifier.Classify(execCtx, input)
	})
	if err != nil {
		return nil, err
	}

	return result.(*types.ClassifyOutput), nil
}

// ExecuteAutomation executes automation rules.
func (e *Executor) ExecuteAutomation(ctx context.Context, engineID string, userID uuid.UUID, input types.AutomationInput) (*types.AutomationOutput, error) {
	engine, err := e.registry.Get(ctx, engineID)
	if err != nil {
		return nil, err
	}

	automation, ok := engine.(types.AutomationEngine)
	if !ok {
		return nil, fmt.Errorf("engine %s is not an automation engine", engineID)
	}

	execCtx := e.createContext(ctx, userID, engineID)

	result, err := e.execute(ctx, engineID, "evaluate", func() (any, error) {
		return automation.Evaluate(execCtx, input)
	})
	if err != nil {
		return nil, err
	}

	return result.(*types.AutomationOutput), nil
}

// HealthCheck checks the health of an engine.
func (e *Executor) HealthCheck(ctx context.Context, engineID string) (sdk.HealthStatus, error) {
	engine, err := e.registry.Get(ctx, engineID)
	if err != nil {
		return sdk.HealthStatus{Healthy: false, Message: err.Error()}, err
	}

	return engine.HealthCheck(ctx), nil
}

// GetMetrics returns the current metrics.
func (e *Executor) GetMetrics() map[string]EngineMetrics {
	return e.metrics.GetAll()
}

// GetCircuitBreakerState returns the circuit breaker state for an engine.
func (e *Executor) GetCircuitBreakerState(engineID string) string {
	breaker := e.breakers[engineID]
	if breaker == nil {
		return "none"
	}
	return breaker.State().String()
}

// ResetCircuitBreaker resets the circuit breaker for an engine.
func (e *Executor) ResetCircuitBreaker(engineID string) {
	delete(e.breakers, engineID)
	e.logger.Info("circuit breaker reset", "engine_id", engineID)
}
