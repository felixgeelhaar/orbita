package api

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStorageAPI_Get(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	// Set a value first
	err := storage.Set(ctx, "mykey", []byte("myvalue"), 0)
	require.NoError(t, err)

	// Get the value
	val, err := storage.Get(ctx, "mykey")
	require.NoError(t, err)
	assert.Equal(t, []byte("myvalue"), val)
}

func TestInMemoryStorageAPI_GetNotFound(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	_, err := storage.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, sdk.ErrStorageKeyNotFound)
}

func TestInMemoryStorageAPI_Set(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	err := storage.Set(ctx, "key1", []byte("value1"), 0)
	require.NoError(t, err)

	// Verify it was stored
	val, err := storage.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, []byte("value1"), val)
}

func TestInMemoryStorageAPI_Delete(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	// Set and then delete
	err := storage.Set(ctx, "todelete", []byte("value"), 0)
	require.NoError(t, err)

	err = storage.Delete(ctx, "todelete")
	require.NoError(t, err)

	// Verify deletion
	_, err = storage.Get(ctx, "todelete")
	assert.ErrorIs(t, err, sdk.ErrStorageKeyNotFound)
}

func TestInMemoryStorageAPI_List(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	// Set multiple values with different prefixes
	require.NoError(t, storage.Set(ctx, "settings:theme", []byte("dark"), 0))
	require.NoError(t, storage.Set(ctx, "settings:language", []byte("en"), 0))
	require.NoError(t, storage.Set(ctx, "data:items", []byte("[]"), 0))

	// List settings prefix
	keys, err := storage.List(ctx, "settings:")
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "settings:theme")
	assert.Contains(t, keys, "settings:language")
}

func TestInMemoryStorageAPI_Exists(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	// Check non-existent key
	exists, err := storage.Exists(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)

	// Set a key and check
	require.NoError(t, storage.Set(ctx, "exists", []byte("value"), 0))
	exists, err = storage.Exists(ctx, "exists")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestInMemoryStorageAPI_KeyTooLong(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	// Create a key longer than StorageKeyMaxLength
	longKey := string(make([]byte, StorageKeyMaxLength+1))

	err := storage.Set(ctx, longKey, []byte("value"), 0)
	assert.ErrorIs(t, err, sdk.ErrStorageKeyTooLong)
}

func TestInMemoryStorageAPI_ValueTooBig(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	// Create a value larger than StorageValueMaxSize
	bigValue := make([]byte, StorageValueMaxSize+1)

	err := storage.Set(ctx, "key", bigValue, 0)
	assert.ErrorIs(t, err, sdk.ErrStorageValueTooBig)
}

func TestInMemoryStorageAPI_Namespacing(t *testing.T) {
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})

	// Two different users/orbits
	storage1 := NewInMemoryStorageAPI("orbit-a", "user-1", caps)
	storage2 := NewInMemoryStorageAPI("orbit-a", "user-2", caps)
	storage3 := NewInMemoryStorageAPI("orbit-b", "user-1", caps)

	ctx := context.Background()

	// Set same key in all storages
	require.NoError(t, storage1.Set(ctx, "key", []byte("value1"), 0))
	require.NoError(t, storage2.Set(ctx, "key", []byte("value2"), 0))
	require.NoError(t, storage3.Set(ctx, "key", []byte("value3"), 0))

	// Each should get their own value (isolated)
	// Since in-memory uses shared map internally with namespacing,
	// we verify the values are different
	val1, _ := storage1.Get(ctx, "key")
	val2, _ := storage2.Get(ctx, "key")
	val3, _ := storage3.Get(ctx, "key")

	assert.Equal(t, []byte("value1"), val1)
	assert.Equal(t, []byte("value2"), val2)
	assert.Equal(t, []byte("value3"), val3)
}

func TestInMemoryStorageAPI_CapabilityEnforcement(t *testing.T) {
	ctx := context.Background()

	t.Run("read without CapReadStorage", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapWriteStorage})
		storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)

		_, err := storage.Get(ctx, "key")
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("write without CapWriteStorage", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage})
		storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)

		err := storage.Set(ctx, "key", []byte("value"), 0)
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("delete without CapWriteStorage", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage})
		storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)

		err := storage.Delete(ctx, "key")
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("list without CapReadStorage", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapWriteStorage})
		storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)

		_, err := storage.List(ctx, "")
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})

	t.Run("exists without CapReadStorage", func(t *testing.T) {
		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapWriteStorage})
		storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)

		_, err := storage.Exists(ctx, "key")
		assert.ErrorIs(t, err, sdk.ErrCapabilityNotGranted)
	})
}

func TestStorageAPIFactory(t *testing.T) {
	t.Run("without Redis falls back to in-memory", func(t *testing.T) {
		factories := &APIFactories{
			RedisClient: nil,
		}

		caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
		factory := factories.StorageAPIFactory()
		storage := factory("test-orbit", uuid.New(), caps)

		// Should be InMemoryStorageAPI
		_, ok := storage.(*InMemoryStorageAPI)
		assert.True(t, ok, "should return in-memory implementation when Redis is nil")
	})
}

func TestInMemoryStorageAPI_TTLIgnored(t *testing.T) {
	// In-memory implementation doesn't honor TTL (simplification for testing)
	// This test documents the behavior
	caps := sdk.NewCapabilitySet([]sdk.Capability{sdk.CapReadStorage, sdk.CapWriteStorage})
	storage := NewInMemoryStorageAPI("test-orbit", "user-123", caps)
	ctx := context.Background()

	// Set with TTL
	err := storage.Set(ctx, "key", []byte("value"), 1*time.Nanosecond)
	require.NoError(t, err)

	// In-memory doesn't expire, so value should still exist
	// (This is intentional for test simplicity)
	val, err := storage.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, []byte("value"), val)
}
