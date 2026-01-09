package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/redis/go-redis/v9"
)

const (
	// StorageKeyMaxLength is the maximum length of a storage key.
	StorageKeyMaxLength = 256

	// StorageValueMaxSize is the maximum size of a storage value in bytes.
	StorageValueMaxSize = 1024 * 1024 // 1MB
)

// StorageAPIImpl implements sdk.StorageAPI with Redis-backed scoped storage.
// Keys are automatically namespaced: orbit:{orbit_id}:user:{user_id}:{key}
type StorageAPIImpl struct {
	client       *redis.Client
	orbitID      string
	userID       string
	capabilities sdk.CapabilitySet
}

// NewStorageAPI creates a new StorageAPI implementation.
func NewStorageAPI(
	client *redis.Client,
	orbitID string,
	userID string,
	caps sdk.CapabilitySet,
) *StorageAPIImpl {
	return &StorageAPIImpl{
		client:       client,
		orbitID:      orbitID,
		userID:       userID,
		capabilities: caps,
	}
}

// namespaceKey creates a fully-qualified key with orbit and user namespace.
func (a *StorageAPIImpl) namespaceKey(key string) string {
	return fmt.Sprintf("orbit:%s:user:%s:%s", a.orbitID, a.userID, key)
}

// stripNamespace removes the namespace prefix from a key.
func (a *StorageAPIImpl) stripNamespace(fullKey string) string {
	prefix := fmt.Sprintf("orbit:%s:user:%s:", a.orbitID, a.userID)
	return strings.TrimPrefix(fullKey, prefix)
}

func (a *StorageAPIImpl) checkReadCapability() error {
	if !a.capabilities.Has(sdk.CapReadStorage) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

func (a *StorageAPIImpl) checkWriteCapability() error {
	if !a.capabilities.Has(sdk.CapWriteStorage) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

// Get retrieves a value by key.
func (a *StorageAPIImpl) Get(ctx context.Context, key string) ([]byte, error) {
	if err := a.checkReadCapability(); err != nil {
		return nil, err
	}

	if len(key) > StorageKeyMaxLength {
		return nil, sdk.ErrStorageKeyTooLong
	}

	fullKey := a.namespaceKey(key)
	val, err := a.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return nil, sdk.ErrStorageKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	return val, nil
}

// Set stores a value with an optional TTL.
// Pass 0 for ttl to store without expiration.
func (a *StorageAPIImpl) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := a.checkWriteCapability(); err != nil {
		return err
	}

	if len(key) > StorageKeyMaxLength {
		return sdk.ErrStorageKeyTooLong
	}

	if len(value) > StorageValueMaxSize {
		return sdk.ErrStorageValueTooBig
	}

	fullKey := a.namespaceKey(key)
	return a.client.Set(ctx, fullKey, value, ttl).Err()
}

// Delete removes a value by key.
func (a *StorageAPIImpl) Delete(ctx context.Context, key string) error {
	if err := a.checkWriteCapability(); err != nil {
		return err
	}

	if len(key) > StorageKeyMaxLength {
		return sdk.ErrStorageKeyTooLong
	}

	fullKey := a.namespaceKey(key)
	return a.client.Del(ctx, fullKey).Err()
}

// List returns all keys matching a prefix.
func (a *StorageAPIImpl) List(ctx context.Context, prefix string) ([]string, error) {
	if err := a.checkReadCapability(); err != nil {
		return nil, err
	}

	fullPrefix := a.namespaceKey(prefix)
	pattern := fullPrefix + "*"

	var keys []string
	iter := a.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, a.stripNamespace(iter.Val()))
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

// Exists checks if a key exists.
func (a *StorageAPIImpl) Exists(ctx context.Context, key string) (bool, error) {
	if err := a.checkReadCapability(); err != nil {
		return false, err
	}

	if len(key) > StorageKeyMaxLength {
		return false, sdk.ErrStorageKeyTooLong
	}

	fullKey := a.namespaceKey(key)
	n, err := a.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}

	return n > 0, nil
}

// InMemoryStorageAPI is a simple in-memory implementation for testing.
type InMemoryStorageAPI struct {
	data         map[string][]byte
	orbitID      string
	userID       string
	capabilities sdk.CapabilitySet
}

// NewInMemoryStorageAPI creates a new in-memory storage API for testing.
func NewInMemoryStorageAPI(orbitID, userID string, caps sdk.CapabilitySet) *InMemoryStorageAPI {
	return &InMemoryStorageAPI{
		data:         make(map[string][]byte),
		orbitID:      orbitID,
		userID:       userID,
		capabilities: caps,
	}
}

func (a *InMemoryStorageAPI) namespaceKey(key string) string {
	return fmt.Sprintf("orbit:%s:user:%s:%s", a.orbitID, a.userID, key)
}

func (a *InMemoryStorageAPI) stripNamespace(fullKey string) string {
	prefix := fmt.Sprintf("orbit:%s:user:%s:", a.orbitID, a.userID)
	return strings.TrimPrefix(fullKey, prefix)
}

func (a *InMemoryStorageAPI) checkReadCapability() error {
	if !a.capabilities.Has(sdk.CapReadStorage) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

func (a *InMemoryStorageAPI) checkWriteCapability() error {
	if !a.capabilities.Has(sdk.CapWriteStorage) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

func (a *InMemoryStorageAPI) Get(ctx context.Context, key string) ([]byte, error) {
	if err := a.checkReadCapability(); err != nil {
		return nil, err
	}
	fullKey := a.namespaceKey(key)
	val, ok := a.data[fullKey]
	if !ok {
		return nil, sdk.ErrStorageKeyNotFound
	}
	return val, nil
}

func (a *InMemoryStorageAPI) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := a.checkWriteCapability(); err != nil {
		return err
	}
	if len(key) > StorageKeyMaxLength {
		return sdk.ErrStorageKeyTooLong
	}
	if len(value) > StorageValueMaxSize {
		return sdk.ErrStorageValueTooBig
	}
	fullKey := a.namespaceKey(key)
	a.data[fullKey] = value
	return nil
}

func (a *InMemoryStorageAPI) Delete(ctx context.Context, key string) error {
	if err := a.checkWriteCapability(); err != nil {
		return err
	}
	fullKey := a.namespaceKey(key)
	delete(a.data, fullKey)
	return nil
}

func (a *InMemoryStorageAPI) List(ctx context.Context, prefix string) ([]string, error) {
	if err := a.checkReadCapability(); err != nil {
		return nil, err
	}
	fullPrefix := a.namespaceKey(prefix)
	var keys []string
	for k := range a.data {
		if strings.HasPrefix(k, fullPrefix) {
			keys = append(keys, a.stripNamespace(k))
		}
	}
	return keys, nil
}

func (a *InMemoryStorageAPI) Exists(ctx context.Context, key string) (bool, error) {
	if err := a.checkReadCapability(); err != nil {
		return false, err
	}
	fullKey := a.namespaceKey(key)
	_, ok := a.data[fullKey]
	return ok, nil
}
