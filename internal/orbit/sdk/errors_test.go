package sdk

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrbitError(t *testing.T) {
	t.Run("Error returns formatted message with orbit ID", func(t *testing.T) {
		err := &OrbitError{
			OrbitID: "acme.wellness",
			Op:      "Initialize",
			Err:     errors.New("config invalid"),
		}

		assert.Equal(t, "orbit acme.wellness: Initialize: config invalid", err.Error())
	})

	t.Run("Error returns formatted message without orbit ID", func(t *testing.T) {
		err := &OrbitError{
			Op:  "LoadManifest",
			Err: errors.New("file not found"),
		}

		assert.Equal(t, "LoadManifest: file not found", err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := &OrbitError{
			OrbitID: "test.orbit",
			Op:      "test",
			Err:     underlying,
		}

		assert.Equal(t, underlying, err.Unwrap())
	})
}

func TestNewOrbitError(t *testing.T) {
	t.Run("creates orbit error with all fields", func(t *testing.T) {
		underlying := errors.New("test error")

		err := NewOrbitError("acme.pomodoro", "RegisterTools", underlying)

		require.NotNil(t, err)
		assert.Equal(t, "acme.pomodoro", err.OrbitID)
		assert.Equal(t, "RegisterTools", err.Op)
		assert.Equal(t, underlying, err.Err)
	})
}

func TestCapabilityError(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		err := &CapabilityError{
			OrbitID:    "acme.wellness",
			Capability: CapReadTasks,
			Operation:  "ListTasks",
		}

		assert.Equal(t, "orbit acme.wellness: capability read:tasks required for ListTasks", err.Error())
	})
}

func TestNewCapabilityError(t *testing.T) {
	t.Run("creates capability error with all fields", func(t *testing.T) {
		err := NewCapabilityError("test.orbit", CapWriteStorage, "SetValue")

		require.NotNil(t, err)
		assert.Equal(t, "test.orbit", err.OrbitID)
		assert.Equal(t, CapWriteStorage, err.Capability)
		assert.Equal(t, "SetValue", err.Operation)
	})
}

func TestSentinelErrors(t *testing.T) {
	t.Run("metadata errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrMissingID)
		assert.NotNil(t, ErrMissingName)
		assert.NotNil(t, ErrMissingVersion)

		assert.Contains(t, ErrMissingID.Error(), "missing ID")
		assert.Contains(t, ErrMissingName.Error(), "missing name")
		assert.Contains(t, ErrMissingVersion.Error(), "missing version")
	})

	t.Run("capability errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrInvalidCapability)
		assert.NotNil(t, ErrCapabilityNotGranted)
		assert.NotNil(t, ErrCapabilityMismatch)
	})

	t.Run("lifecycle errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrOrbitNotInitialized)
		assert.NotNil(t, ErrOrbitAlreadyLoaded)
		assert.NotNil(t, ErrOrbitNotFound)
		assert.NotNil(t, ErrOrbitNotEntitled)
	})

	t.Run("registration errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrToolAlreadyRegistered)
		assert.NotNil(t, ErrCommandAlreadyRegistered)
		assert.NotNil(t, ErrInvalidToolName)
		assert.NotNil(t, ErrInvalidCommandName)
	})

	t.Run("storage errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrStorageKeyNotFound)
		assert.NotNil(t, ErrStorageKeyTooLong)
		assert.NotNil(t, ErrStorageValueTooBig)
	})

	t.Run("event errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrInvalidEventType)
		assert.NotNil(t, ErrEventHandlerFailed)
	})

	t.Run("API errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrResourceNotFound)
		assert.NotNil(t, ErrAccessDenied)
	})

	t.Run("manifest errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrManifestNotFound)
		assert.NotNil(t, ErrManifestInvalid)
		assert.NotNil(t, ErrManifestMissingID)
		assert.NotNil(t, ErrManifestMissingType)
	})

	t.Run("version errors are defined", func(t *testing.T) {
		assert.NotNil(t, ErrIncompatibleAPIVersion)
	})

	t.Run("sentinel errors are distinct", func(t *testing.T) {
		sentinels := []error{
			ErrMissingID,
			ErrMissingName,
			ErrMissingVersion,
			ErrOrbitNotFound,
			ErrOrbitNotEntitled,
			ErrStorageKeyNotFound,
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

func TestErrorWrapping(t *testing.T) {
	t.Run("OrbitError can wrap sentinel errors", func(t *testing.T) {
		err := NewOrbitError("test.orbit", "Load", ErrOrbitNotFound)

		assert.True(t, errors.Is(err, ErrOrbitNotFound))
	})

	t.Run("OrbitError can be extracted with errors.As", func(t *testing.T) {
		err := NewOrbitError("test.orbit", "Init", errors.New("failed"))
		wrapped := errors.Join(errors.New("context"), err)

		var orbitErr *OrbitError
		assert.True(t, errors.As(wrapped, &orbitErr))
		assert.Equal(t, "test.orbit", orbitErr.OrbitID)
	})
}
