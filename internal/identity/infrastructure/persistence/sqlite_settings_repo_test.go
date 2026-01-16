package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupSettingsTestDB creates an in-memory SQLite database with the schema applied.
func setupSettingsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Open in-memory database
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Read and execute the schema
	schemaPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read SQLite schema file")

	_, err = sqlDB.Exec(string(schema))
	require.NoError(t, err, "Failed to apply SQLite schema")

	return sqlDB
}

// createSettingsTestUser creates a user in the database for foreign key constraints.
func createSettingsTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
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

func TestSQLiteSettingsRepository_GetCalendarID_NotSet(t *testing.T) {
	sqlDB := setupSettingsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createSettingsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSettingsRepository(sqlDB)
	ctx := context.Background()

	// Get calendar ID when not set
	calendarID, err := repo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, calendarID)
}

func TestSQLiteSettingsRepository_SetAndGetCalendarID(t *testing.T) {
	sqlDB := setupSettingsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createSettingsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSettingsRepository(sqlDB)
	ctx := context.Background()

	// Set calendar ID
	expectedCalendarID := "test-calendar-123"
	err := repo.SetCalendarID(ctx, userID, expectedCalendarID)
	require.NoError(t, err)

	// Get calendar ID
	calendarID, err := repo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, expectedCalendarID, calendarID)
}

func TestSQLiteSettingsRepository_UpdateCalendarID(t *testing.T) {
	sqlDB := setupSettingsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createSettingsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSettingsRepository(sqlDB)
	ctx := context.Background()

	// Set initial calendar ID
	err := repo.SetCalendarID(ctx, userID, "initial-calendar")
	require.NoError(t, err)

	// Update calendar ID
	err = repo.SetCalendarID(ctx, userID, "updated-calendar")
	require.NoError(t, err)

	// Verify update
	calendarID, err := repo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "updated-calendar", calendarID)
}

func TestSQLiteSettingsRepository_GetDeleteMissing_NotSet(t *testing.T) {
	sqlDB := setupSettingsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createSettingsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSettingsRepository(sqlDB)
	ctx := context.Background()

	// Get delete missing when not set (default is false)
	deleteMissing, err := repo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.False(t, deleteMissing)
}

func TestSQLiteSettingsRepository_SetAndGetDeleteMissing_True(t *testing.T) {
	sqlDB := setupSettingsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createSettingsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSettingsRepository(sqlDB)
	ctx := context.Background()

	// Set delete missing to true
	err := repo.SetDeleteMissing(ctx, userID, true)
	require.NoError(t, err)

	// Get delete missing
	deleteMissing, err := repo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.True(t, deleteMissing)
}

func TestSQLiteSettingsRepository_SetAndGetDeleteMissing_False(t *testing.T) {
	sqlDB := setupSettingsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createSettingsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSettingsRepository(sqlDB)
	ctx := context.Background()

	// Set delete missing to true first
	err := repo.SetDeleteMissing(ctx, userID, true)
	require.NoError(t, err)

	// Set delete missing to false
	err = repo.SetDeleteMissing(ctx, userID, false)
	require.NoError(t, err)

	// Verify it's false
	deleteMissing, err := repo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.False(t, deleteMissing)
}

func TestSQLiteSettingsRepository_MultipleUsers(t *testing.T) {
	sqlDB := setupSettingsTestDB(t)
	defer sqlDB.Close()

	user1 := uuid.New()
	user2 := uuid.New()
	createSettingsTestUser(t, sqlDB, user1)
	createSettingsTestUser(t, sqlDB, user2)

	repo := NewSQLiteSettingsRepository(sqlDB)
	ctx := context.Background()

	// Set different settings for each user
	err := repo.SetCalendarID(ctx, user1, "user1-calendar")
	require.NoError(t, err)
	err = repo.SetCalendarID(ctx, user2, "user2-calendar")
	require.NoError(t, err)

	err = repo.SetDeleteMissing(ctx, user1, true)
	require.NoError(t, err)
	err = repo.SetDeleteMissing(ctx, user2, false)
	require.NoError(t, err)

	// Verify user1 settings
	cal1, err := repo.GetCalendarID(ctx, user1)
	require.NoError(t, err)
	assert.Equal(t, "user1-calendar", cal1)

	del1, err := repo.GetDeleteMissing(ctx, user1)
	require.NoError(t, err)
	assert.True(t, del1)

	// Verify user2 settings
	cal2, err := repo.GetCalendarID(ctx, user2)
	require.NoError(t, err)
	assert.Equal(t, "user2-calendar", cal2)

	del2, err := repo.GetDeleteMissing(ctx, user2)
	require.NoError(t, err)
	assert.False(t, del2)
}

func TestSQLiteSettingsRepository_FullCycle(t *testing.T) {
	sqlDB := setupSettingsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createSettingsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSettingsRepository(sqlDB)
	ctx := context.Background()

	// Initially empty/false
	cal, err := repo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, cal)

	del, err := repo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.False(t, del)

	// Set values
	err = repo.SetCalendarID(ctx, userID, "my-calendar")
	require.NoError(t, err)
	err = repo.SetDeleteMissing(ctx, userID, true)
	require.NoError(t, err)

	// Verify values
	cal, err = repo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "my-calendar", cal)

	del, err = repo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.True(t, del)

	// Update values
	err = repo.SetCalendarID(ctx, userID, "new-calendar")
	require.NoError(t, err)
	err = repo.SetDeleteMissing(ctx, userID, false)
	require.NoError(t, err)

	// Verify updates
	cal, err = repo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "new-calendar", cal)

	del, err = repo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.False(t, del)
}
