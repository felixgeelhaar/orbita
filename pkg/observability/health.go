package observability

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// HealthStatus represents the health state of a component.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheckResult is the result of a health check.
type HealthCheckResult struct {
	Status    HealthStatus   `json:"status"`
	Message   string         `json:"message,omitempty"`
	Duration  time.Duration  `json:"duration_ns"`
	Timestamp time.Time      `json:"timestamp"`
	Details   map[string]any `json:"details,omitempty"`
}

// HealthChecker is a function that performs a health check.
type HealthChecker func(ctx context.Context) HealthCheckResult

// HealthRegistry manages health checks for multiple components.
type HealthRegistry struct {
	mu       sync.RWMutex
	checkers map[string]HealthChecker
	results  map[string]HealthCheckResult
}

// NewHealthRegistry creates a new health registry.
func NewHealthRegistry() *HealthRegistry {
	return &HealthRegistry{
		checkers: make(map[string]HealthChecker),
		results:  make(map[string]HealthCheckResult),
	}
}

// Register adds a health checker for a component.
func (r *HealthRegistry) Register(name string, checker HealthChecker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers[name] = checker
}

// Unregister removes a health checker.
func (r *HealthRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.checkers, name)
	delete(r.results, name)
}

// Check runs all health checks and returns aggregated results.
func (r *HealthRegistry) Check(ctx context.Context) map[string]HealthCheckResult {
	r.mu.Lock()
	checkers := make(map[string]HealthChecker, len(r.checkers))
	for k, v := range r.checkers {
		checkers[k] = v
	}
	r.mu.Unlock()

	results := make(map[string]HealthCheckResult, len(checkers))
	var wg sync.WaitGroup

	resultCh := make(chan struct {
		name   string
		result HealthCheckResult
	}, len(checkers))

	for name, checker := range checkers {
		wg.Add(1)
		go func(name string, checker HealthChecker) {
			defer wg.Done()
			start := time.Now()
			result := checker(ctx)
			result.Duration = time.Since(start)
			result.Timestamp = time.Now()
			resultCh <- struct {
				name   string
				result HealthCheckResult
			}{name, result}
		}(name, checker)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for item := range resultCh {
		results[item.name] = item.result
	}

	// Update cached results
	r.mu.Lock()
	r.results = results
	r.mu.Unlock()

	return results
}

// CheckOne runs a single health check by name.
func (r *HealthRegistry) CheckOne(ctx context.Context, name string) (HealthCheckResult, bool) {
	r.mu.RLock()
	checker, ok := r.checkers[name]
	r.mu.RUnlock()

	if !ok {
		return HealthCheckResult{}, false
	}

	start := time.Now()
	result := checker(ctx)
	result.Duration = time.Since(start)
	result.Timestamp = time.Now()

	return result, true
}

// LastResults returns the cached results from the last Check call.
func (r *HealthRegistry) LastResults() map[string]HealthCheckResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]HealthCheckResult, len(r.results))
	for k, v := range r.results {
		results[k] = v
	}
	return results
}

// OverallStatus returns the overall health status based on all checks.
func (r *HealthRegistry) OverallStatus() HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.results) == 0 {
		return HealthStatusHealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range r.results {
		switch result.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}
	return HealthStatusHealthy
}

// OverallHealth returns a summary of the health status.
type OverallHealth struct {
	Status    HealthStatus                   `json:"status"`
	Timestamp time.Time                      `json:"timestamp"`
	Checks    map[string]HealthCheckResult `json:"checks"`
}

// GetOverallHealth runs all checks and returns overall health.
func (r *HealthRegistry) GetOverallHealth(ctx context.Context) OverallHealth {
	checks := r.Check(ctx)
	return OverallHealth{
		Status:    r.OverallStatus(),
		Timestamp: time.Now(),
		Checks:    checks,
	}
}

// ToJSON serializes the overall health to JSON.
func (h OverallHealth) ToJSON() ([]byte, error) {
	return json.Marshal(h)
}

// Common health checkers

// DatabaseHealthChecker creates a health checker for database connectivity.
func DatabaseHealthChecker(pingFunc func(ctx context.Context) error) HealthChecker {
	return func(ctx context.Context) HealthCheckResult {
		err := pingFunc(ctx)
		if err != nil {
			return HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: "database connection failed: " + err.Error(),
			}
		}
		return HealthCheckResult{
			Status:  HealthStatusHealthy,
			Message: "database connection healthy",
		}
	}
}

// RedisHealthChecker creates a health checker for Redis connectivity.
func RedisHealthChecker(pingFunc func(ctx context.Context) error) HealthChecker {
	return func(ctx context.Context) HealthCheckResult {
		err := pingFunc(ctx)
		if err != nil {
			return HealthCheckResult{
				Status:  HealthStatusDegraded,
				Message: "redis connection failed: " + err.Error(),
			}
		}
		return HealthCheckResult{
			Status:  HealthStatusHealthy,
			Message: "redis connection healthy",
		}
	}
}

// RabbitMQHealthChecker creates a health checker for RabbitMQ connectivity.
func RabbitMQHealthChecker(checkFunc func(ctx context.Context) error) HealthChecker {
	return func(ctx context.Context) HealthCheckResult {
		err := checkFunc(ctx)
		if err != nil {
			return HealthCheckResult{
				Status:  HealthStatusDegraded,
				Message: "rabbitmq connection failed: " + err.Error(),
			}
		}
		return HealthCheckResult{
			Status:  HealthStatusHealthy,
			Message: "rabbitmq connection healthy",
		}
	}
}
