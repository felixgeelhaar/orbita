package sdk

import (
	"errors"
	"fmt"
)

// Sentinel errors for common engine error conditions.
var (
	// ErrEngineNotFound is returned when an engine cannot be found in the registry.
	ErrEngineNotFound = errors.New("engine not found")

	// ErrEngineAlreadyExists is returned when trying to register a duplicate engine.
	ErrEngineAlreadyExists = errors.New("engine already exists")

	// ErrEngineNotInitialized is returned when operating on an uninitialized engine.
	ErrEngineNotInitialized = errors.New("engine not initialized")

	// ErrInvalidConfig is returned when engine configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrUnsupportedOperation is returned when an engine doesn't support a requested operation.
	ErrUnsupportedOperation = errors.New("unsupported operation")

	// ErrEngineShutdown is returned when operating on a shutdown engine.
	ErrEngineShutdown = errors.New("engine has been shut down")

	// ErrVersionIncompatible is returned when SDK and engine versions are incompatible.
	ErrVersionIncompatible = errors.New("incompatible version")

	// ErrTimeout is returned when an engine operation times out.
	ErrTimeout = errors.New("operation timed out")

	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker open")
)

// EngineError wraps an error with engine context.
type EngineError struct {
	// EngineID is the ID of the engine that produced the error.
	EngineID string

	// Operation is the operation that failed.
	Operation string

	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *EngineError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("engine %s: %s: %v", e.EngineID, e.Operation, e.Err)
	}
	return fmt.Sprintf("engine %s: %v", e.EngineID, e.Err)
}

// Unwrap returns the underlying error.
func (e *EngineError) Unwrap() error {
	return e.Err
}

// NewEngineError creates a new engine error.
func NewEngineError(engineID, operation string, err error) *EngineError {
	return &EngineError{
		EngineID:  engineID,
		Operation: operation,
		Err:       err,
	}
}

// ConfigValidationError represents a configuration validation failure.
type ConfigValidationError struct {
	// Field is the configuration field that failed validation.
	Field string

	// Message describes the validation failure.
	Message string

	// Value is the invalid value (if safe to include).
	Value any
}

// Error implements the error interface.
func (e *ConfigValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("config validation failed for %q: %s (got: %v)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("config validation failed for %q: %s", e.Field, e.Message)
}

// NewConfigValidationError creates a new configuration validation error.
func NewConfigValidationError(field, message string, value any) *ConfigValidationError {
	return &ConfigValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}

// LoadError represents an error during plugin loading.
type LoadError struct {
	// Path is the path to the plugin that failed to load.
	Path string

	// Reason describes why loading failed.
	Reason string

	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *LoadError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("failed to load plugin %q: %s: %v", e.Path, e.Reason, e.Err)
	}
	return fmt.Sprintf("failed to load plugin %q: %s", e.Path, e.Reason)
}

// Unwrap returns the underlying error.
func (e *LoadError) Unwrap() error {
	return e.Err
}

// NewLoadError creates a new load error.
func NewLoadError(path, reason string, err error) *LoadError {
	return &LoadError{
		Path:   path,
		Reason: reason,
		Err:    err,
	}
}

// ExecutionError represents an error during engine execution.
type ExecutionError struct {
	// EngineID is the ID of the engine that produced the error.
	EngineID string

	// RequestID is the request that failed.
	RequestID string

	// Operation is the operation that failed.
	Operation string

	// Err is the underlying error.
	Err error

	// Retryable indicates if the operation can be retried.
	Retryable bool
}

// Error implements the error interface.
func (e *ExecutionError) Error() string {
	return fmt.Sprintf("execution error in %s (request %s, operation %s): %v",
		e.EngineID, e.RequestID, e.Operation, e.Err)
}

// Unwrap returns the underlying error.
func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// NewExecutionError creates a new execution error.
func NewExecutionError(engineID, requestID, operation string, err error, retryable bool) *ExecutionError {
	return &ExecutionError{
		EngineID:  engineID,
		RequestID: requestID,
		Operation: operation,
		Err:       err,
		Retryable: retryable,
	}
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	var execErr *ExecutionError
	if errors.As(err, &execErr) {
		return execErr.Retryable
	}
	return false
}

// IsEngineNotFound checks if the error is ErrEngineNotFound.
func IsEngineNotFound(err error) bool {
	return errors.Is(err, ErrEngineNotFound)
}

// IsConfigInvalid checks if the error is a configuration validation error.
func IsConfigInvalid(err error) bool {
	var configErr *ConfigValidationError
	return errors.As(err, &configErr) || errors.Is(err, ErrInvalidConfig)
}

// IsCircuitOpen checks if the error is due to an open circuit breaker.
func IsCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}
