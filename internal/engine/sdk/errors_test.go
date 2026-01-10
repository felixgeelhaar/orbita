package sdk

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineError(t *testing.T) {
	t.Run("Error returns formatted message with operation", func(t *testing.T) {
		err := &EngineError{
			EngineID:  "acme.scheduler",
			Operation: "ScheduleTasks",
			Err:       errors.New("connection failed"),
		}

		assert.Equal(t, "engine acme.scheduler: ScheduleTasks: connection failed", err.Error())
	})

	t.Run("Error returns formatted message without operation", func(t *testing.T) {
		err := &EngineError{
			EngineID: "acme.scheduler",
			Err:      errors.New("initialization failed"),
		}

		assert.Equal(t, "engine acme.scheduler: initialization failed", err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := &EngineError{
			EngineID: "test.engine",
			Err:      underlying,
		}

		assert.Equal(t, underlying, err.Unwrap())
	})
}

func TestNewEngineError(t *testing.T) {
	t.Run("creates engine error with all fields", func(t *testing.T) {
		underlying := errors.New("test error")

		err := NewEngineError("acme.priority", "CalculatePriority", underlying)

		require.NotNil(t, err)
		assert.Equal(t, "acme.priority", err.EngineID)
		assert.Equal(t, "CalculatePriority", err.Operation)
		assert.Equal(t, underlying, err.Err)
	})
}

func TestConfigValidationError(t *testing.T) {
	t.Run("Error returns message with value", func(t *testing.T) {
		err := &ConfigValidationError{
			Field:   "max_retries",
			Message: "must be positive",
			Value:   -1,
		}

		assert.Equal(t, `config validation failed for "max_retries": must be positive (got: -1)`, err.Error())
	})

	t.Run("Error returns message without value", func(t *testing.T) {
		err := &ConfigValidationError{
			Field:   "api_key",
			Message: "is required",
		}

		assert.Equal(t, `config validation failed for "api_key": is required`, err.Error())
	})
}

func TestNewConfigValidationError(t *testing.T) {
	t.Run("creates config validation error", func(t *testing.T) {
		err := NewConfigValidationError("timeout", "must be greater than 0", 0)

		require.NotNil(t, err)
		assert.Equal(t, "timeout", err.Field)
		assert.Equal(t, "must be greater than 0", err.Message)
		assert.Equal(t, 0, err.Value)
	})
}

func TestLoadError(t *testing.T) {
	t.Run("Error returns message with underlying error", func(t *testing.T) {
		err := &LoadError{
			Path:   "/plugins/acme-scheduler",
			Reason: "binary not found",
			Err:    errors.New("no such file"),
		}

		assert.Equal(t, `failed to load plugin "/plugins/acme-scheduler": binary not found: no such file`, err.Error())
	})

	t.Run("Error returns message without underlying error", func(t *testing.T) {
		err := &LoadError{
			Path:   "/plugins/acme-scheduler",
			Reason: "invalid manifest",
		}

		assert.Equal(t, `failed to load plugin "/plugins/acme-scheduler": invalid manifest`, err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := errors.New("permission denied")
		err := &LoadError{
			Path:   "/plugins/test",
			Reason: "cannot read",
			Err:    underlying,
		}

		assert.Equal(t, underlying, err.Unwrap())
	})
}

func TestNewLoadError(t *testing.T) {
	t.Run("creates load error with all fields", func(t *testing.T) {
		underlying := errors.New("file not found")

		err := NewLoadError("/path/to/plugin", "cannot open", underlying)

		require.NotNil(t, err)
		assert.Equal(t, "/path/to/plugin", err.Path)
		assert.Equal(t, "cannot open", err.Reason)
		assert.Equal(t, underlying, err.Err)
	})
}

func TestExecutionError(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		err := &ExecutionError{
			EngineID:  "acme.scheduler",
			RequestID: "req-123",
			Operation: "ScheduleTasks",
			Err:       errors.New("timeout"),
			Retryable: true,
		}

		assert.Equal(t, "execution error in acme.scheduler (request req-123, operation ScheduleTasks): timeout", err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := errors.New("connection reset")
		err := &ExecutionError{
			EngineID: "test.engine",
			Err:      underlying,
		}

		assert.Equal(t, underlying, err.Unwrap())
	})
}

func TestNewExecutionError(t *testing.T) {
	t.Run("creates execution error with all fields", func(t *testing.T) {
		underlying := errors.New("deadline exceeded")

		err := NewExecutionError("acme.priority", "req-456", "BatchCalculate", underlying, true)

		require.NotNil(t, err)
		assert.Equal(t, "acme.priority", err.EngineID)
		assert.Equal(t, "req-456", err.RequestID)
		assert.Equal(t, "BatchCalculate", err.Operation)
		assert.Equal(t, underlying, err.Err)
		assert.True(t, err.Retryable)
	})

	t.Run("creates non-retryable execution error", func(t *testing.T) {
		err := NewExecutionError("test.engine", "req-789", "Init", errors.New("invalid config"), false)

		assert.False(t, err.Retryable)
	})
}

func TestIsRetryable(t *testing.T) {
	t.Run("returns true for retryable execution error", func(t *testing.T) {
		err := NewExecutionError("test", "req", "op", errors.New("timeout"), true)

		assert.True(t, IsRetryable(err))
	})

	t.Run("returns false for non-retryable execution error", func(t *testing.T) {
		err := NewExecutionError("test", "req", "op", errors.New("invalid input"), false)

		assert.False(t, IsRetryable(err))
	})

	t.Run("returns false for non-execution error", func(t *testing.T) {
		err := errors.New("some error")

		assert.False(t, IsRetryable(err))
	})

	t.Run("returns true for wrapped retryable error", func(t *testing.T) {
		execErr := NewExecutionError("test", "req", "op", errors.New("transient"), true)
		wrapped := errors.Join(errors.New("context"), execErr)

		assert.True(t, IsRetryable(wrapped))
	})
}

func TestIsEngineNotFound(t *testing.T) {
	t.Run("returns true for ErrEngineNotFound", func(t *testing.T) {
		assert.True(t, IsEngineNotFound(ErrEngineNotFound))
	})

	t.Run("returns true for wrapped ErrEngineNotFound", func(t *testing.T) {
		wrapped := errors.Join(errors.New("lookup failed"), ErrEngineNotFound)

		assert.True(t, IsEngineNotFound(wrapped))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		assert.False(t, IsEngineNotFound(ErrEngineShutdown))
		assert.False(t, IsEngineNotFound(errors.New("random error")))
	})
}

func TestIsConfigInvalid(t *testing.T) {
	t.Run("returns true for ConfigValidationError", func(t *testing.T) {
		err := NewConfigValidationError("field", "invalid", nil)

		assert.True(t, IsConfigInvalid(err))
	})

	t.Run("returns true for ErrInvalidConfig", func(t *testing.T) {
		assert.True(t, IsConfigInvalid(ErrInvalidConfig))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		assert.False(t, IsConfigInvalid(errors.New("random error")))
	})
}

func TestIsCircuitOpen(t *testing.T) {
	t.Run("returns true for ErrCircuitOpen", func(t *testing.T) {
		assert.True(t, IsCircuitOpen(ErrCircuitOpen))
	})

	t.Run("returns true for wrapped ErrCircuitOpen", func(t *testing.T) {
		wrapped := errors.Join(errors.New("call failed"), ErrCircuitOpen)

		assert.True(t, IsCircuitOpen(wrapped))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		assert.False(t, IsCircuitOpen(errors.New("random error")))
	})
}

func TestSentinelErrors(t *testing.T) {
	t.Run("sentinel errors are distinct", func(t *testing.T) {
		sentinels := []error{
			ErrEngineNotFound,
			ErrEngineAlreadyExists,
			ErrEngineNotInitialized,
			ErrInvalidConfig,
			ErrUnsupportedOperation,
			ErrEngineShutdown,
			ErrVersionIncompatible,
			ErrTimeout,
			ErrCircuitOpen,
			ErrNoSlotAvailable,
		}

		for i, err1 := range sentinels {
			for j, err2 := range sentinels {
				if i == j {
					assert.True(t, errors.Is(err1, err2))
				} else {
					assert.False(t, errors.Is(err1, err2))
				}
			}
		}
	})
}
