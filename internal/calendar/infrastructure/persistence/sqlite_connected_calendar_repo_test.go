package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupCalendarTestDB creates an in-memory SQLite database with the schema applied.
func setupCalendarTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Open in-memory database
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Read and execute the initial schema
	schemaPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read SQLite schema file")

	_, err = sqlDB.Exec(string(schema))
	require.NoError(t, err, "Failed to apply SQLite schema")

	// Apply calendar sync state migration
	syncStatePath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000002_calendar_sync_state.up.sql")
	syncStateSchema, err := os.ReadFile(syncStatePath)
	require.NoError(t, err, "Failed to read calendar_sync_state migration")

	_, err = sqlDB.Exec(string(syncStateSchema))
	require.NoError(t, err, "Failed to apply calendar_sync_state migration")

	// Apply connected calendars migration
	connectedPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000003_connected_calendars.up.sql")
	connectedSchema, err := os.ReadFile(connectedPath)
	require.NoError(t, err, "Failed to read connected_calendars migration")

	_, err = sqlDB.Exec(string(connectedSchema))
	require.NoError(t, err, "Failed to apply connected_calendars migration")

	// Apply version column migration for optimistic concurrency
	versionPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000005_connected_calendars_version.up.sql")
	versionSchema, err := os.ReadFile(versionPath)
	require.NoError(t, err, "Failed to read connected_calendars_version migration")

	_, err = sqlDB.Exec(string(versionSchema))
	require.NoError(t, err, "Failed to apply connected_calendars_version migration")

	return sqlDB
}

// createCalendarTestUser creates a user in the database for foreign key constraints.
func createCalendarTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
	t.Helper()

	queries := db.New(sqlDB)
	_, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        userID.String(),
		Email:     "test-" + userID.String()[:8] + "@example.com",
		Name:      "Test User",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
	require.NoError(t, err)
}

func TestSQLiteConnectedCalendarRepository_Save_Create(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create a new connected calendar
	cal, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal Calendar")
	require.NoError(t, err)
	cal.SetPrimary(true, nil)
	cal.SetSyncPull(true)

	// Save it
	err = repo.Save(ctx, cal)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.FindByID(ctx, cal.ID())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, cal.ID(), found.ID())
	assert.Equal(t, userID, found.UserID())
	assert.Equal(t, domain.ProviderGoogle, found.Provider())
	assert.Equal(t, "primary", found.CalendarID())
	assert.Equal(t, "Personal Calendar", found.Name())
	assert.True(t, found.IsPrimary())
	assert.True(t, found.IsEnabled())
	assert.True(t, found.SyncPush())
	assert.True(t, found.SyncPull())
}

func TestSQLiteConnectedCalendarRepository_Save_Update(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create and save a calendar
	cal, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal Calendar")
	require.NoError(t, err)
	cal.SetPrimary(true, nil)

	err = repo.Save(ctx, cal)
	require.NoError(t, err)

	// Update the calendar
	cal.SetName("Updated Calendar")
	cal.SetSyncPull(true)
	cal.MarkSyncedSimple()

	err = repo.Save(ctx, cal)
	require.NoError(t, err)

	// Verify the update
	found, err := repo.FindByID(ctx, cal.ID())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "Updated Calendar", found.Name())
	assert.True(t, found.SyncPull())
	assert.False(t, found.LastSyncAt().IsZero())
}

func TestSQLiteConnectedCalendarRepository_FindByID_NotFound(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteConnectedCalendarRepository_FindByUserAndProvider(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create two Google calendars
	cal1, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "cal1", "Calendar 1")
	require.NoError(t, err)
	cal1.SetPrimary(true, nil)
	cal1.SetSyncPull(true)

	cal2, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "cal2", "Calendar 2")
	require.NoError(t, err)

	require.NoError(t, repo.Save(ctx, cal1))
	require.NoError(t, repo.Save(ctx, cal2))

	// Find by user and provider
	calendars, err := repo.FindByUserAndProvider(ctx, userID, domain.ProviderGoogle)
	require.NoError(t, err)
	assert.Len(t, calendars, 2)

	// Primary should be first (ORDER BY is_primary DESC)
	assert.True(t, calendars[0].IsPrimary())
}

func TestSQLiteConnectedCalendarRepository_FindByUserAndProvider_Empty(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	calendars, err := repo.FindByUserAndProvider(ctx, userID, domain.ProviderMicrosoft)
	require.NoError(t, err)
	assert.Empty(t, calendars)
}

func TestSQLiteConnectedCalendarRepository_FindByUserProviderAndCalendar(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	cal, err := domain.NewConnectedCalendar(userID, domain.ProviderMicrosoft, "work-cal", "Work Calendar")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, cal))

	// Find specific calendar
	found, err := repo.FindByUserProviderAndCalendar(ctx, userID, domain.ProviderMicrosoft, "work-cal")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "Work Calendar", found.Name())
}

func TestSQLiteConnectedCalendarRepository_FindByUserProviderAndCalendar_NotFound(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindByUserProviderAndCalendar(ctx, userID, domain.ProviderGoogle, "non-existent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteConnectedCalendarRepository_FindByUser(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create calendars from different providers
	googleCal, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "g-cal", "Google Cal")
	require.NoError(t, err)
	googleCal.SetPrimary(true, nil)
	googleCal.SetSyncPull(true)

	msCal, err := domain.NewConnectedCalendar(userID, domain.ProviderMicrosoft, "ms-cal", "MS Cal")
	require.NoError(t, err)
	caldavCal, err := domain.NewConnectedCalendar(userID, domain.ProviderCalDAV, "dav-cal", "CalDAV Cal")
	require.NoError(t, err)

	require.NoError(t, repo.Save(ctx, googleCal))
	require.NoError(t, repo.Save(ctx, msCal))
	require.NoError(t, repo.Save(ctx, caldavCal))

	// Find all for user
	calendars, err := repo.FindByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, calendars, 3)

	// Primary should be first
	assert.True(t, calendars[0].IsPrimary())
}

func TestSQLiteConnectedCalendarRepository_FindPrimaryForUser(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create calendars - one primary
	primaryCal, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Primary")
	require.NoError(t, err)
	primaryCal.SetPrimary(true, nil)
	primaryCal.SetSyncPull(true)

	secondaryCal, err := domain.NewConnectedCalendar(userID, domain.ProviderMicrosoft, "secondary", "Secondary")
	require.NoError(t, err)

	require.NoError(t, repo.Save(ctx, primaryCal))
	require.NoError(t, repo.Save(ctx, secondaryCal))

	// Find primary
	found, err := repo.FindPrimaryForUser(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.True(t, found.IsPrimary())
	assert.Equal(t, "Primary", found.Name())
}

func TestSQLiteConnectedCalendarRepository_FindPrimaryForUser_NoPrimary(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create only non-primary calendar
	cal, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "cal", "Cal")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, cal))

	// Find primary - should return nil
	found, err := repo.FindPrimaryForUser(ctx, userID)
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteConnectedCalendarRepository_FindEnabledPushCalendars(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create calendars with different push settings
	pushEnabled, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "push1", "Push Enabled")
	require.NoError(t, err)
	// Default is enabled and syncPush=true

	pushDisabled, err := domain.NewConnectedCalendar(userID, domain.ProviderMicrosoft, "push2", "Push Disabled")
	require.NoError(t, err)
	pushDisabled.SetSyncPush(false)

	disabled, err := domain.NewConnectedCalendar(userID, domain.ProviderCalDAV, "push3", "Disabled")
	require.NoError(t, err)
	disabled.SetEnabled(false)

	require.NoError(t, repo.Save(ctx, pushEnabled))
	require.NoError(t, repo.Save(ctx, pushDisabled))
	require.NoError(t, repo.Save(ctx, disabled))

	// Find enabled push calendars
	calendars, err := repo.FindEnabledPushCalendars(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, calendars, 1)
	assert.Equal(t, "Push Enabled", calendars[0].Name())
}

func TestSQLiteConnectedCalendarRepository_FindEnabledPullCalendars(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create calendars with different pull settings
	pullEnabled, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "pull1", "Pull Enabled")
	require.NoError(t, err)
	pullEnabled.SetPrimary(true, nil)
	pullEnabled.SetSyncPull(true)

	pullDisabled, err := domain.NewConnectedCalendar(userID, domain.ProviderMicrosoft, "pull2", "Pull Disabled")
	require.NoError(t, err)
	// Default syncPull=false

	require.NoError(t, repo.Save(ctx, pullEnabled))
	require.NoError(t, repo.Save(ctx, pullDisabled))

	// Find enabled pull calendars
	calendars, err := repo.FindEnabledPullCalendars(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, calendars, 1)
	assert.Equal(t, "Pull Enabled", calendars[0].Name())
}

func TestSQLiteConnectedCalendarRepository_Delete(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	cal, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "cal", "Cal")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, cal))

	// Delete it
	err = repo.Delete(ctx, cal.ID())
	require.NoError(t, err)

	// Verify it's gone
	found, err := repo.FindByID(ctx, cal.ID())
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteConnectedCalendarRepository_DeleteByUserAndProvider(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create multiple Google calendars
	cal1, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "g1", "G1")
	require.NoError(t, err)
	cal1.SetPrimary(true, nil)
	cal1.SetSyncPull(true)

	cal2, err := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "g2", "G2")
	require.NoError(t, err)
	// And one Microsoft calendar
	msCal, err := domain.NewConnectedCalendar(userID, domain.ProviderMicrosoft, "ms", "MS")
	require.NoError(t, err)

	require.NoError(t, repo.Save(ctx, cal1))
	require.NoError(t, repo.Save(ctx, cal2))
	require.NoError(t, repo.Save(ctx, msCal))

	// Delete all Google calendars
	err = repo.DeleteByUserAndProvider(ctx, userID, domain.ProviderGoogle)
	require.NoError(t, err)

	// Verify Google calendars are gone
	googleCals, err := repo.FindByUserAndProvider(ctx, userID, domain.ProviderGoogle)
	assert.NoError(t, err)
	assert.Empty(t, googleCals)

	// Microsoft calendar should remain
	msCals, err := repo.FindByUserAndProvider(ctx, userID, domain.ProviderMicrosoft)
	assert.NoError(t, err)
	assert.Len(t, msCals, 1)
}

func TestSQLiteConnectedCalendarRepository_AllProviders(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	providers := []domain.ProviderType{
		domain.ProviderGoogle,
		domain.ProviderMicrosoft,
		domain.ProviderApple,
		domain.ProviderCalDAV,
	}

	for _, provider := range providers {
		cal, err := domain.NewConnectedCalendar(userID, provider, provider.String()+"-cal", provider.String()+" Calendar")
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, cal))
	}

	// Verify all providers saved correctly
	calendars, err := repo.FindByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, calendars, 4)
}

func TestSQLiteConnectedCalendarRepository_WithConfig(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createCalendarTestUser(t, sqlDB, userID)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create CalDAV calendar with config
	cal, err := domain.NewConnectedCalendar(userID, domain.ProviderCalDAV, "caldav-cal", "CalDAV")
	require.NoError(t, err)
	cal.SetConfig("url", "https://caldav.example.com")
	cal.SetConfig("username", "user@example.com")

	require.NoError(t, repo.Save(ctx, cal))

	// Retrieve and verify config
	found, err := repo.FindByID(ctx, cal.ID())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "https://caldav.example.com", found.ConfigValue("url"))
	assert.Equal(t, "user@example.com", found.ConfigValue("username"))
}

func TestSQLiteConnectedCalendarRepository_MultipleUsers(t *testing.T) {
	sqlDB := setupCalendarTestDB(t)
	defer sqlDB.Close()

	user1 := uuid.New()
	user2 := uuid.New()
	createCalendarTestUser(t, sqlDB, user1)
	createCalendarTestUser(t, sqlDB, user2)

	repo := NewSQLiteConnectedCalendarRepository(sqlDB)
	ctx := context.Background()

	// Create calendars for both users
	cal1, err := domain.NewConnectedCalendar(user1, domain.ProviderGoogle, "cal", "User1 Cal")
	require.NoError(t, err)
	cal1.SetPrimary(true, nil)
	cal1.SetSyncPull(true)

	cal2, err := domain.NewConnectedCalendar(user2, domain.ProviderGoogle, "cal", "User2 Cal")
	require.NoError(t, err)
	cal2.SetPrimary(true, nil)
	cal2.SetSyncPull(true)

	require.NoError(t, repo.Save(ctx, cal1))
	require.NoError(t, repo.Save(ctx, cal2))

	// Each user should find only their calendar
	found1, err := repo.FindByUser(ctx, user1)
	require.NoError(t, err)
	require.Len(t, found1, 1)
	assert.Equal(t, "User1 Cal", found1[0].Name())

	found2, err := repo.FindByUser(ctx, user2)
	require.NoError(t, err)
	require.Len(t, found2, 1)
	assert.Equal(t, "User2 Cal", found2[0].Name())
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}

func TestIntToBool(t *testing.T) {
	assert.True(t, intToBool(1))
	assert.True(t, intToBool(42))
	assert.False(t, intToBool(0))
}
