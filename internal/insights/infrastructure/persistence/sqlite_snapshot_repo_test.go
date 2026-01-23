package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestSnapshot(userID uuid.UUID, snapshotDate time.Time) *domain.ProductivitySnapshot {
	return &domain.ProductivitySnapshot{
		ID:                     uuid.New(),
		UserID:                 userID,
		SnapshotDate:           snapshotDate,
		TasksCreated:           10,
		TasksCompleted:         8,
		TasksOverdue:           1,
		TaskCompletionRate:     0.8,
		AvgTaskDurationMinutes: 25,
		BlocksScheduled:        6,
		BlocksCompleted:        5,
		BlocksMissed:           1,
		ScheduledMinutes:       180,
		CompletedMinutes:       150,
		BlockCompletionRate:    0.83,
		HabitsDue:              5,
		HabitsCompleted:        4,
		HabitCompletionRate:    0.8,
		LongestStreak:          7,
		FocusSessions:          3,
		TotalFocusMinutes:      90,
		AvgFocusSessionMinutes: 30,
		ProductivityScore:      75,
		PeakHours:              []domain.PeakHour{{Hour: 9, Completions: 5}, {Hour: 10, Completions: 4}, {Hour: 14, Completions: 3}},
		TimeByCategory:         map[string]int{"work": 120, "personal": 30},
		ComputedAt:             time.Now().UTC().Truncate(time.Second),
		CreatedAt:              time.Now().UTC().Truncate(time.Second),
		UpdatedAt:              time.Now().UTC().Truncate(time.Second),
	}
}

func TestNewSQLiteSnapshotRepository(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSnapshotRepository(sqlDB)
	assert.NotNil(t, repo)
}

func TestSQLiteSnapshotRepository_Save(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	snapshot := createTestSnapshot(userID, today)

	err := repo.Save(ctx, snapshot)
	require.NoError(t, err)

	// Verify it was saved
	found, err := repo.GetByDate(ctx, userID, today)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, snapshot.TasksCompleted, found.TasksCompleted)
	assert.Equal(t, snapshot.ProductivityScore, found.ProductivityScore)
	assert.Equal(t, snapshot.PeakHours, found.PeakHours)
}

func TestSQLiteSnapshotRepository_Save_Upsert(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	snapshot := createTestSnapshot(userID, today)

	// First save
	err := repo.Save(ctx, snapshot)
	require.NoError(t, err)

	// Update and save again (upsert)
	snapshot.TasksCompleted = 15
	snapshot.ProductivityScore = 85
	err = repo.Save(ctx, snapshot)
	require.NoError(t, err)

	// Verify the update
	found, err := repo.GetByDate(ctx, userID, today)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 15, found.TasksCompleted)
	assert.Equal(t, 85, found.ProductivityScore)
}

func TestSQLiteSnapshotRepository_GetByDate_NotFound(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.GetByDate(ctx, userID, time.Now())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteSnapshotRepository_GetDateRange(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create snapshots for 5 days
	for i := 0; i < 5; i++ {
		snapshot := createTestSnapshot(userID, today.AddDate(0, 0, i))
		snapshot.ProductivityScore = 70 + i*5
		err := repo.Save(ctx, snapshot)
		require.NoError(t, err)
	}

	// Get 3 day range
	snapshots, err := repo.GetDateRange(ctx, userID, today, today.AddDate(0, 0, 2))
	require.NoError(t, err)
	assert.Len(t, snapshots, 3)

	// Should be ordered by date ASC
	assert.Equal(t, 70, snapshots[0].ProductivityScore)
	assert.Equal(t, 75, snapshots[1].ProductivityScore)
	assert.Equal(t, 80, snapshots[2].ProductivityScore)
}

func TestSQLiteSnapshotRepository_GetLatest(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create snapshots for 3 days
	for i := 0; i < 3; i++ {
		snapshot := createTestSnapshot(userID, today.AddDate(0, 0, -i))
		snapshot.ProductivityScore = 90 - i*10
		err := repo.Save(ctx, snapshot)
		require.NoError(t, err)
	}

	// Get latest should return today's snapshot
	latest, err := repo.GetLatest(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, 90, latest.ProductivityScore)
}

func TestSQLiteSnapshotRepository_GetLatest_NoSnapshots(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	latest, err := repo.GetLatest(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, latest)
}

func TestSQLiteSnapshotRepository_GetRecent(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create 5 snapshots
	for i := 0; i < 5; i++ {
		snapshot := createTestSnapshot(userID, today.AddDate(0, 0, -i))
		err := repo.Save(ctx, snapshot)
		require.NoError(t, err)
	}

	// Get recent 3
	recent, err := repo.GetRecent(ctx, userID, 3)
	require.NoError(t, err)
	assert.Len(t, recent, 3)
}

func TestSQLiteSnapshotRepository_GetAverageScore(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create snapshots with known scores: 60, 70, 80 (avg = 70)
	scores := []int{60, 70, 80}
	for i, score := range scores {
		snapshot := createTestSnapshot(userID, today.AddDate(0, 0, i))
		snapshot.ProductivityScore = score
		err := repo.Save(ctx, snapshot)
		require.NoError(t, err)
	}

	avg, err := repo.GetAverageScore(ctx, userID, today, today.AddDate(0, 0, 2))
	require.NoError(t, err)
	assert.Equal(t, 70, avg)
}

func TestSQLiteSnapshotRepository_GetAverageScore_NoSnapshots(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSnapshotRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	avg, err := repo.GetAverageScore(ctx, userID, today, today.AddDate(0, 0, 7))
	require.NoError(t, err)
	assert.Equal(t, 0, avg)
}
