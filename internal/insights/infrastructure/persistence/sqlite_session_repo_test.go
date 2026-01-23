package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupInsightsTestDB creates an in-memory SQLite database with the schema applied.
func setupInsightsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	schemaPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read SQLite schema file")

	_, err = sqlDB.Exec(string(schema))
	require.NoError(t, err, "Failed to apply SQLite schema")

	return sqlDB
}

// createInsightsTestUser creates a user in the database for foreign key constraints.
func createInsightsTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
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

func TestSQLiteSessionRepository_Create(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	now := time.Now()
	session := &domain.TimeSession{
		ID:          uuid.New(),
		UserID:      userID,
		SessionType: domain.SessionTypeFocus,
		Title:       "Deep Work Session",
		Category:    "work",
		StartedAt:   now,
		Status:      domain.SessionStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, session.ID, found.ID)
	assert.Equal(t, session.Title, found.Title)
	assert.Equal(t, domain.SessionTypeFocus, found.SessionType)
}

func TestSQLiteSessionRepository_Create_WithReferenceID(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	now := time.Now()
	taskID := uuid.New()
	session := &domain.TimeSession{
		ID:          uuid.New(),
		UserID:      userID,
		SessionType: domain.SessionTypeTask,
		ReferenceID: &taskID,
		Title:       "Task Session",
		StartedAt:   now,
		Status:      domain.SessionStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.ReferenceID)
	assert.Equal(t, taskID, *found.ReferenceID)
}

func TestSQLiteSessionRepository_Update(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	now := time.Now()
	session := &domain.TimeSession{
		ID:          uuid.New(),
		UserID:      userID,
		SessionType: domain.SessionTypeFocus,
		Title:       "Deep Work",
		StartedAt:   now,
		Status:      domain.SessionStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)

	// Update to complete
	endedAt := now.Add(30 * time.Minute)
	duration := 30
	session.EndedAt = &endedAt
	session.DurationMinutes = &duration
	session.Status = domain.SessionStatusCompleted

	err = repo.Update(ctx, session)
	require.NoError(t, err)

	// Verify the update
	found, err := repo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, domain.SessionStatusCompleted, found.Status)
	require.NotNil(t, found.DurationMinutes)
	assert.Equal(t, 30, *found.DurationMinutes)
}

func TestSQLiteSessionRepository_GetActive(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	now := time.Now()

	// Create an active session
	activeSession := &domain.TimeSession{
		ID:          uuid.New(),
		UserID:      userID,
		SessionType: domain.SessionTypeFocus,
		Title:       "Active Session",
		StartedAt:   now,
		Status:      domain.SessionStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, repo.Create(ctx, activeSession))

	// Create a completed session
	endedAt := now.Add(-time.Hour)
	duration := 60
	completedSession := &domain.TimeSession{
		ID:              uuid.New(),
		UserID:          userID,
		SessionType:     domain.SessionTypeFocus,
		Title:           "Completed Session",
		StartedAt:       now.Add(-2 * time.Hour),
		EndedAt:         &endedAt,
		DurationMinutes: &duration,
		Status:          domain.SessionStatusCompleted,
		CreatedAt:       now.Add(-2 * time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	require.NoError(t, repo.Create(ctx, completedSession))

	// Get active should return only the active one
	found, err := repo.GetActive(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, activeSession.ID, found.ID)
}

func TestSQLiteSessionRepository_GetByDateRange(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create sessions on different days
	for i := 0; i < 5; i++ {
		session := &domain.TimeSession{
			ID:          uuid.New(),
			UserID:      userID,
			SessionType: domain.SessionTypeFocus,
			Title:       "Session",
			StartedAt:   today.Add(time.Duration(i) * 24 * time.Hour),
			Status:      domain.SessionStatusCompleted,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		require.NoError(t, repo.Create(ctx, session))
	}

	// Get sessions for 3-day range
	start := today
	end := today.Add(2*24*time.Hour + 23*time.Hour)
	sessions, err := repo.GetByDateRange(ctx, userID, start, end)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestSQLiteSessionRepository_GetByType(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	now := time.Now()

	// Create different types of sessions
	types := []domain.SessionType{
		domain.SessionTypeFocus,
		domain.SessionTypeFocus,
		domain.SessionTypeTask,
		domain.SessionTypeMeeting,
	}

	for i, st := range types {
		session := &domain.TimeSession{
			ID:          uuid.New(),
			UserID:      userID,
			SessionType: st,
			Title:       "Session",
			StartedAt:   now.Add(time.Duration(i) * time.Hour),
			Status:      domain.SessionStatusCompleted,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		require.NoError(t, repo.Create(ctx, session))
	}

	// Get only focus sessions
	focusSessions, err := repo.GetByType(ctx, userID, domain.SessionTypeFocus, 10)
	require.NoError(t, err)
	assert.Len(t, focusSessions, 2)

	// All should be focus type
	for _, s := range focusSessions {
		assert.Equal(t, domain.SessionTypeFocus, s.SessionType)
	}
}

func TestSQLiteSessionRepository_GetTotalFocusMinutes(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create completed focus sessions
	durations := []int{30, 45, 60}
	for i, dur := range durations {
		d := dur
		endedAt := today.Add(time.Duration(i+1)*time.Hour + time.Duration(dur)*time.Minute)
		session := &domain.TimeSession{
			ID:              uuid.New(),
			UserID:          userID,
			SessionType:     domain.SessionTypeFocus,
			Title:           "Focus Session",
			StartedAt:       today.Add(time.Duration(i+1) * time.Hour),
			EndedAt:         &endedAt,
			DurationMinutes: &d,
			Status:          domain.SessionStatusCompleted,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		require.NoError(t, repo.Create(ctx, session))
	}

	// Also create a non-focus session that shouldn't count
	d := 120
	endedAt := today.Add(6*time.Hour + 120*time.Minute)
	taskSession := &domain.TimeSession{
		ID:              uuid.New(),
		UserID:          userID,
		SessionType:     domain.SessionTypeTask,
		Title:           "Task Session",
		StartedAt:       today.Add(6 * time.Hour),
		EndedAt:         &endedAt,
		DurationMinutes: &d,
		Status:          domain.SessionStatusCompleted,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	require.NoError(t, repo.Create(ctx, taskSession))

	// Get total focus minutes
	total, err := repo.GetTotalFocusMinutes(ctx, userID, today, today.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 135, total) // 30 + 45 + 60
}

func TestSQLiteSessionRepository_Delete(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	session := &domain.TimeSession{
		ID:          uuid.New(),
		UserID:      userID,
		SessionType: domain.SessionTypeFocus,
		Title:       "To Be Deleted",
		StartedAt:   time.Now(),
		Status:      domain.SessionStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)

	err = repo.Delete(ctx, session.ID)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, session.ID)
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteSessionRepository_GetByID_NotFound(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteSessionRepository_SessionTypes(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSessionRepository(sqlDB)
	ctx := context.Background()

	now := time.Now()

	sessionTypes := []domain.SessionType{
		domain.SessionTypeTask,
		domain.SessionTypeHabit,
		domain.SessionTypeFocus,
		domain.SessionTypeMeeting,
		domain.SessionTypeOther,
	}

	for _, st := range sessionTypes {
		session := &domain.TimeSession{
			ID:          uuid.New(),
			UserID:      userID,
			SessionType: st,
			Title:       string(st) + " session",
			StartedAt:   now,
			Status:      domain.SessionStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		err := repo.Create(ctx, session)
		require.NoError(t, err, "Should create session with type %s", st)

		found, err := repo.GetByID(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, st, found.SessionType)
	}
}
