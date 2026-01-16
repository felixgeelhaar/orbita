package domain_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnectedCalendar(t *testing.T) {
	userID := uuid.New()
	provider := domain.ProviderGoogle
	calendarID := "primary"
	name := "My Calendar"

	cal := domain.NewConnectedCalendar(userID, provider, calendarID, name)

	require.NotNil(t, cal)
	assert.NotEqual(t, uuid.Nil, cal.ID())
	assert.Equal(t, userID, cal.UserID())
	assert.Equal(t, provider, cal.Provider())
	assert.Equal(t, calendarID, cal.CalendarID())
	assert.Equal(t, name, cal.Name())
	assert.False(t, cal.IsPrimary())
	assert.True(t, cal.IsEnabled())
	assert.True(t, cal.SyncPush())
	assert.False(t, cal.SyncPull())
	assert.NotNil(t, cal.Config())
	assert.True(t, cal.LastSyncAt().IsZero())
	assert.False(t, cal.HasSynced())
}

func TestConnectedCalendar_SetName(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "Old Name")

	cal.SetName("New Name")

	assert.Equal(t, "New Name", cal.Name())
}

func TestConnectedCalendar_SetPrimary(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "Calendar")

	assert.False(t, cal.IsPrimary())

	cal.SetPrimary(true)
	assert.True(t, cal.IsPrimary())

	cal.SetPrimary(false)
	assert.False(t, cal.IsPrimary())
}

func TestConnectedCalendar_SetEnabled(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "Calendar")

	assert.True(t, cal.IsEnabled())

	cal.SetEnabled(false)
	assert.False(t, cal.IsEnabled())

	cal.SetEnabled(true)
	assert.True(t, cal.IsEnabled())
}

func TestConnectedCalendar_SetSyncPush(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "Calendar")

	assert.True(t, cal.SyncPush())

	cal.SetSyncPush(false)
	assert.False(t, cal.SyncPush())

	cal.SetSyncPush(true)
	assert.True(t, cal.SyncPush())
}

func TestConnectedCalendar_SetSyncPull(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "Calendar")

	assert.False(t, cal.SyncPull())

	cal.SetSyncPull(true)
	assert.True(t, cal.SyncPull())

	cal.SetSyncPull(false)
	assert.False(t, cal.SyncPull())
}

func TestConnectedCalendar_SetConfig(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "cal1", "Calendar")

	cal.SetConfig("url", "https://example.com/caldav")
	cal.SetConfig("username", "user@example.com")

	assert.Equal(t, "https://example.com/caldav", cal.ConfigValue("url"))
	assert.Equal(t, "user@example.com", cal.ConfigValue("username"))
	assert.Equal(t, "", cal.ConfigValue("nonexistent"))
}

func TestConnectedCalendar_ConfigValue_NilConfig(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "Calendar")

	// ConfigValue should return empty string for missing keys
	assert.Equal(t, "", cal.ConfigValue("missing"))
}

func TestConnectedCalendar_ConfigJSON(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "cal1", "Calendar")

	// Empty config should return "{}"
	assert.Equal(t, "{}", cal.ConfigJSON())

	// Add config
	cal.SetConfig("url", "https://example.com")

	json := cal.ConfigJSON()
	assert.Contains(t, json, "url")
	assert.Contains(t, json, "https://example.com")
}

func TestConnectedCalendar_MarkSynced(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "Calendar")

	assert.False(t, cal.HasSynced())
	assert.True(t, cal.LastSyncAt().IsZero())

	cal.MarkSynced()

	assert.True(t, cal.HasSynced())
	assert.False(t, cal.LastSyncAt().IsZero())
	assert.WithinDuration(t, time.Now().UTC(), cal.LastSyncAt(), time.Second)
}

func TestConnectedCalendar_SetCalDAVConfig(t *testing.T) {
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "calendar", "Calendar")

	cal.SetCalDAVConfig("https://caldav.example.com", "user@example.com")

	assert.Equal(t, "https://caldav.example.com", cal.CalDAVURL())
	assert.Equal(t, "user@example.com", cal.CalDAVUsername())
}

func TestRehydrateConnectedCalendar(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	provider := domain.ProviderMicrosoft
	calendarID := "work@example.com"
	name := "Work Calendar"
	isPrimary := true
	isEnabled := true
	syncPush := true
	syncPull := true
	configJSON := `{"tenant_id":"abc123"}`
	lastSyncAt := time.Now().UTC().Add(-time.Hour)
	createdAt := time.Now().UTC().Add(-24 * time.Hour)
	updatedAt := time.Now().UTC().Add(-time.Hour)

	cal := domain.RehydrateConnectedCalendar(
		id, userID, provider, calendarID, name,
		isPrimary, isEnabled, syncPush, syncPull,
		configJSON, lastSyncAt, createdAt, updatedAt,
	)

	require.NotNil(t, cal)
	assert.Equal(t, id, cal.ID())
	assert.Equal(t, userID, cal.UserID())
	assert.Equal(t, provider, cal.Provider())
	assert.Equal(t, calendarID, cal.CalendarID())
	assert.Equal(t, name, cal.Name())
	assert.True(t, cal.IsPrimary())
	assert.True(t, cal.IsEnabled())
	assert.True(t, cal.SyncPush())
	assert.True(t, cal.SyncPull())
	assert.Equal(t, "abc123", cal.ConfigValue("tenant_id"))
	assert.Equal(t, lastSyncAt, cal.LastSyncAt())
}

func TestRehydrateConnectedCalendar_EmptyConfig(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	cal := domain.RehydrateConnectedCalendar(
		id, userID, domain.ProviderGoogle, "primary", "Calendar",
		false, true, true, false,
		"", now, now, now,
	)

	require.NotNil(t, cal)
	assert.NotNil(t, cal.Config())
	assert.Equal(t, "", cal.ConfigValue("any_key"))
}

func TestRehydrateConnectedCalendar_EmptyBracesConfig(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	cal := domain.RehydrateConnectedCalendar(
		id, userID, domain.ProviderGoogle, "primary", "Calendar",
		false, true, true, false,
		"{}", now, now, now,
	)

	require.NotNil(t, cal)
	assert.NotNil(t, cal.Config())
}
