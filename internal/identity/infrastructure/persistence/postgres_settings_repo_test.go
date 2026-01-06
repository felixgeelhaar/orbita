package persistence_test

import (
	"context"
	"os"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/identity/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSettingsTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Failed to ping test database: %v", err)
	}

	_, _ = pool.Exec(ctx, "DELETE FROM user_settings")

	return pool
}

func TestSettingsRepository_CalendarID(t *testing.T) {
	pool := setupSettingsTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewSettingsRepository(pool)
	userID := uuid.New()

	calendarID, err := repo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "", calendarID)

	require.NoError(t, repo.SetCalendarID(ctx, userID, "work"))

	calendarID, err = repo.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "work", calendarID)
}

func TestSettingsRepository_DeleteMissing(t *testing.T) {
	pool := setupSettingsTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	repo := persistence.NewSettingsRepository(pool)
	userID := uuid.New()

	value, err := repo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.False(t, value)

	require.NoError(t, repo.SetDeleteMissing(ctx, userID, true))

	value, err = repo.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.True(t, value)
}
