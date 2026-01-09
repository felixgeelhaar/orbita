package registry

import (
	"context"
	"log/slog"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrbit is a test orbit implementation
type mockOrbit struct {
	id           string
	name         string
	version      string
	capabilities []sdk.Capability
	initialized  bool
	shutdown     bool
}

func newMockOrbit(id, name, version string, caps ...sdk.Capability) *mockOrbit {
	return &mockOrbit{
		id:           id,
		name:         name,
		version:      version,
		capabilities: caps,
	}
}

func (o *mockOrbit) Metadata() sdk.Metadata {
	return sdk.Metadata{
		ID:      o.id,
		Name:    o.name,
		Version: o.version,
	}
}

func (o *mockOrbit) RequiredCapabilities() []sdk.Capability {
	return o.capabilities
}

func (o *mockOrbit) Initialize(ctx sdk.Context) error {
	o.initialized = true
	return nil
}

func (o *mockOrbit) Shutdown(ctx context.Context) error {
	o.shutdown = true
	return nil
}

func (o *mockOrbit) RegisterTools(registry sdk.ToolRegistry) error {
	return nil
}

func (o *mockOrbit) RegisterCommands(registry sdk.CommandRegistry) error {
	return nil
}

func (o *mockOrbit) SubscribeEvents(bus sdk.EventBus) error {
	return nil
}

// mockEntitlementChecker for testing
type mockEntitlementChecker struct {
	entitlements map[string]bool
}

func (m *mockEntitlementChecker) HasEntitlement(ctx context.Context, userID uuid.UUID, entitlement string) (bool, error) {
	return m.entitlements[entitlement], nil
}

func TestRegistry_RegisterBuiltin(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit := newMockOrbit("test.orbit", "Test Orbit", "1.0.0", sdk.CapReadTasks)

	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Verify it was registered
	entries := registry.List()
	assert.Len(t, entries, 1)
	assert.True(t, entries[0].Builtin)
	assert.Equal(t, StatusReady, entries[0].Status)
}

func TestRegistry_RegisterBuiltin_DuplicateID(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit1 := newMockOrbit("test.orbit", "Test Orbit 1", "1.0.0")
	orbit2 := newMockOrbit("test.orbit", "Test Orbit 2", "2.0.0")

	err := registry.RegisterBuiltin(orbit1)
	require.NoError(t, err)

	err = registry.RegisterBuiltin(orbit2)
	assert.ErrorIs(t, err, sdk.ErrOrbitAlreadyLoaded)
}

func TestRegistry_RegisterBuiltin_MissingID(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit := newMockOrbit("", "Test Orbit", "1.0.0")

	err := registry.RegisterBuiltin(orbit)
	assert.ErrorIs(t, err, sdk.ErrMissingID)
}

func TestRegistry_RegisterFactory(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	factory := func() (sdk.Orbit, error) {
		return newMockOrbit("lazy.orbit", "Lazy Orbit", "1.0.0"), nil
	}

	manifest := &Manifest{
		ID:      "lazy.orbit",
		Name:    "Lazy Orbit",
		Version: "1.0.0",
	}

	err := registry.RegisterFactory("lazy.orbit", factory, manifest)
	require.NoError(t, err)

	// Verify it was registered but not loaded
	entries := registry.List()
	assert.Len(t, entries, 1)
	assert.False(t, entries[0].Builtin)
	assert.Equal(t, StatusUnloaded, entries[0].Status)
	assert.Nil(t, entries[0].Orbit)
}

func TestRegistry_Get_BuiltinOrbit(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit := newMockOrbit("test.orbit", "Test Orbit", "1.0.0")
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	ctx := context.Background()
	userID := uuid.New()

	result, err := registry.Get(ctx, "test.orbit", userID)
	require.NoError(t, err)
	assert.Equal(t, orbit, result)
}

func TestRegistry_Get_LazyLoadedOrbit(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	var factoryCalled bool
	factory := func() (sdk.Orbit, error) {
		factoryCalled = true
		return newMockOrbit("lazy.orbit", "Lazy Orbit", "1.0.0"), nil
	}

	manifest := &Manifest{
		ID:      "lazy.orbit",
		Name:    "Lazy Orbit",
		Version: "1.0.0",
	}

	err := registry.RegisterFactory("lazy.orbit", factory, manifest)
	require.NoError(t, err)

	ctx := context.Background()
	userID := uuid.New()

	// First call should trigger lazy loading
	result, err := registry.Get(ctx, "lazy.orbit", userID)
	require.NoError(t, err)
	assert.True(t, factoryCalled)
	assert.NotNil(t, result)

	// Verify status is now ready
	status, _ := registry.Status("lazy.orbit")
	assert.Equal(t, StatusReady, status)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	ctx := context.Background()
	userID := uuid.New()

	_, err := registry.Get(ctx, "nonexistent.orbit", userID)
	assert.ErrorIs(t, err, sdk.ErrOrbitNotFound)
}

func TestRegistry_Get_WithEntitlement(t *testing.T) {
	logger := slog.Default()
	entitlementChecker := &mockEntitlementChecker{
		entitlements: map[string]bool{
			"premium-orbit": true,
		},
	}
	registry := NewRegistry(logger, entitlementChecker)

	orbit := newMockOrbit("premium.orbit", "Premium Orbit", "1.0.0")
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Set entitlement requirement via manifest
	registry.orbits["premium.orbit"].Manifest.Entitlement = "premium-orbit"

	ctx := context.Background()
	userID := uuid.New()

	// User has entitlement - should succeed
	result, err := registry.Get(ctx, "premium.orbit", userID)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRegistry_Get_WithoutEntitlement(t *testing.T) {
	logger := slog.Default()
	entitlementChecker := &mockEntitlementChecker{
		entitlements: map[string]bool{}, // No entitlements
	}
	registry := NewRegistry(logger, entitlementChecker)

	orbit := newMockOrbit("premium.orbit", "Premium Orbit", "1.0.0")
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Set entitlement requirement via manifest
	registry.orbits["premium.orbit"].Manifest.Entitlement = "premium-orbit"

	ctx := context.Background()
	userID := uuid.New()

	// User lacks entitlement - should fail
	_, err = registry.Get(ctx, "premium.orbit", userID)
	assert.ErrorIs(t, err, sdk.ErrOrbitNotEntitled)
}

func TestRegistry_GetMetadata(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit := newMockOrbit("test.orbit", "Test Orbit", "1.0.0")
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	metadata, err := registry.GetMetadata("test.orbit")
	require.NoError(t, err)
	assert.Equal(t, "test.orbit", metadata.ID)
	assert.Equal(t, "Test Orbit", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
}

func TestRegistry_GetManifest(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit := newMockOrbit("test.orbit", "Test Orbit", "1.0.0", sdk.CapReadTasks)
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	manifest, err := registry.GetManifest("test.orbit")
	require.NoError(t, err)
	assert.Equal(t, "test.orbit", manifest.ID)
	assert.Contains(t, manifest.Capabilities, string(sdk.CapReadTasks))
}

func TestRegistry_Status(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit := newMockOrbit("test.orbit", "Test Orbit", "1.0.0")
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	status, err := registry.Status("test.orbit")
	require.NoError(t, err)
	assert.Equal(t, StatusReady, status)
}

func TestRegistry_Unregister(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit := newMockOrbit("test.orbit", "Test Orbit", "1.0.0")
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	ctx := context.Background()

	err = registry.Unregister(ctx, "test.orbit")
	require.NoError(t, err)

	// Verify it's gone
	_, err = registry.Status("test.orbit")
	assert.ErrorIs(t, err, sdk.ErrOrbitNotFound)

	// Verify shutdown was called
	assert.True(t, orbit.shutdown)
}

func TestRegistry_Shutdown(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit1 := newMockOrbit("orbit1", "Orbit 1", "1.0.0")
	orbit2 := newMockOrbit("orbit2", "Orbit 2", "1.0.0")

	err := registry.RegisterBuiltin(orbit1)
	require.NoError(t, err)
	err = registry.RegisterBuiltin(orbit2)
	require.NoError(t, err)

	ctx := context.Background()
	err = registry.Shutdown(ctx)
	require.NoError(t, err)

	// Verify both orbits were shut down
	assert.True(t, orbit1.shutdown)
	assert.True(t, orbit2.shutdown)
}

func TestRegistry_ValidateCapabilities(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	// Create orbit with capabilities
	orbit := newMockOrbit("test.orbit", "Test Orbit", "1.0.0", sdk.CapReadTasks, sdk.CapWriteStorage)
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Validation should pass since manifest includes same capabilities
	err = registry.ValidateCapabilities("test.orbit")
	require.NoError(t, err)
}

func TestRegistry_ListAvailable(t *testing.T) {
	logger := slog.Default()
	entitlementChecker := &mockEntitlementChecker{
		entitlements: map[string]bool{
			"premium": true,
		},
	}
	registry := NewRegistry(logger, entitlementChecker)

	// Register free orbit
	freeOrbit := newMockOrbit("free.orbit", "Free Orbit", "1.0.0")
	err := registry.RegisterBuiltin(freeOrbit)
	require.NoError(t, err)

	// Register premium orbit
	premiumOrbit := newMockOrbit("premium.orbit", "Premium Orbit", "1.0.0")
	err = registry.RegisterBuiltin(premiumOrbit)
	require.NoError(t, err)
	registry.orbits["premium.orbit"].Manifest.Entitlement = "premium"

	// Register unaccessible orbit
	exclusiveOrbit := newMockOrbit("exclusive.orbit", "Exclusive Orbit", "1.0.0")
	err = registry.RegisterBuiltin(exclusiveOrbit)
	require.NoError(t, err)
	registry.orbits["exclusive.orbit"].Manifest.Entitlement = "exclusive"

	ctx := context.Background()
	userID := uuid.New()

	available := registry.ListAvailable(ctx, userID)
	assert.Len(t, available, 2) // free + premium, not exclusive
}

func TestRegistry_Has(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	orbit := newMockOrbit("test.orbit", "Test Orbit", "1.0.0")
	err := registry.RegisterBuiltin(orbit)
	require.NoError(t, err)

	// Has should return true for registered orbit
	assert.True(t, registry.Has("test.orbit"))

	// Has should return false for unregistered orbit
	assert.False(t, registry.Has("nonexistent.orbit"))
}

func TestRegistry_RegisterManifest(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	manifest := &Manifest{
		ID:           "discovered.orbit",
		Name:         "Discovered Orbit",
		Version:      "1.0.0",
		Type:         "orbit",
		Capabilities: []string{"read:storage"},
	}

	err := registry.RegisterManifest(manifest, "/path/to/orbit")
	require.NoError(t, err)

	// Verify it was registered
	entries := registry.List()
	assert.Len(t, entries, 1)
	assert.Equal(t, StatusUnloaded, entries[0].Status)
	assert.Nil(t, entries[0].Orbit)
	assert.Equal(t, manifest, entries[0].Manifest)
}

func TestRegistry_RegisterManifest_DuplicateID(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	manifest := &Manifest{
		ID:      "discovered.orbit",
		Name:    "Discovered Orbit",
		Version: "1.0.0",
		Type:    "orbit",
	}

	err := registry.RegisterManifest(manifest, "/path/to/orbit")
	require.NoError(t, err)

	// Second registration should fail
	err = registry.RegisterManifest(manifest, "/another/path")
	assert.ErrorIs(t, err, sdk.ErrOrbitAlreadyLoaded)
}

func TestRegistry_RegisterManifest_NilManifest(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	err := registry.RegisterManifest(nil, "/path/to/orbit")
	assert.ErrorIs(t, err, sdk.ErrManifestInvalid)
}

func TestRegistry_RegisterManifest_MissingID(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	manifest := &Manifest{
		Name:    "Discovered Orbit",
		Version: "1.0.0",
		Type:    "orbit",
	}

	err := registry.RegisterManifest(manifest, "/path/to/orbit")
	assert.ErrorIs(t, err, sdk.ErrMissingID)
}

func TestRegistry_Has_AfterRegisterManifest(t *testing.T) {
	logger := slog.Default()
	registry := NewRegistry(logger, nil)

	manifest := &Manifest{
		ID:      "discovered.orbit",
		Name:    "Discovered Orbit",
		Version: "1.0.0",
		Type:    "orbit",
	}

	// Before registration
	assert.False(t, registry.Has("discovered.orbit"))

	err := registry.RegisterManifest(manifest, "/path/to/orbit")
	require.NoError(t, err)

	// After registration
	assert.True(t, registry.Has("discovered.orbit"))
}
