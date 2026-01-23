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

func createTestSummary(userID uuid.UUID, weekStart time.Time) *domain.WeeklySummary {
	mostProductiveDay := weekStart.AddDate(0, 0, 2)
	leastProductiveDay := weekStart.AddDate(0, 0, 5)
	return &domain.WeeklySummary{
		ID:                       uuid.New(),
		UserID:                   userID,
		WeekStart:                weekStart,
		WeekEnd:                  weekStart.AddDate(0, 0, 7),
		TotalTasksCompleted:      25,
		TotalHabitsCompleted:     15,
		TotalBlocksCompleted:     20,
		TotalFocusMinutes:        450,
		AvgDailyProductivityScore: 72,
		AvgDailyFocusMinutes:     64,
		ProductivityTrend:        1.05,
		FocusTrend:               0.95,
		MostProductiveDay:        &mostProductiveDay,
		LeastProductiveDay:       &leastProductiveDay,
		HabitsWithStreak:         3,
		LongestStreak:            14,
		ComputedAt:               time.Now().UTC().Truncate(time.Second),
		CreatedAt:                time.Now().UTC().Truncate(time.Second),
	}
}

func TestNewSQLiteSummaryRepository(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteSummaryRepository(sqlDB)
	assert.NotNil(t, repo)
}

func TestSQLiteSummaryRepository_Save(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSummaryRepository(sqlDB)
	ctx := context.Background()

	weekStart := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -int(time.Now().Weekday()))
	summary := createTestSummary(userID, weekStart)

	err := repo.Save(ctx, summary)
	require.NoError(t, err)

	// Verify it was saved
	found, err := repo.GetByWeek(ctx, userID, weekStart)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, summary.TotalTasksCompleted, found.TotalTasksCompleted)
	assert.Equal(t, summary.AvgDailyProductivityScore, found.AvgDailyProductivityScore)
}

func TestSQLiteSummaryRepository_Save_Upsert(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSummaryRepository(sqlDB)
	ctx := context.Background()

	weekStart := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -int(time.Now().Weekday()))
	summary := createTestSummary(userID, weekStart)

	// First save
	err := repo.Save(ctx, summary)
	require.NoError(t, err)

	// Update and save again (upsert)
	summary.TotalTasksCompleted = 50
	summary.AvgDailyProductivityScore = 85.0
	err = repo.Save(ctx, summary)
	require.NoError(t, err)

	// Verify the update
	found, err := repo.GetByWeek(ctx, userID, weekStart)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 50, found.TotalTasksCompleted)
	assert.InDelta(t, 85.0, found.AvgDailyProductivityScore, 0.01)
}

func TestSQLiteSummaryRepository_GetByWeek_NotFound(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSummaryRepository(sqlDB)
	ctx := context.Background()

	weekStart := time.Now().Truncate(24 * time.Hour)
	found, err := repo.GetByWeek(ctx, userID, weekStart)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteSummaryRepository_GetRecent(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSummaryRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))

	// Create 5 weeks of summaries
	for i := 0; i < 5; i++ {
		summary := createTestSummary(userID, weekStart.AddDate(0, 0, -i*7))
		summary.AvgDailyProductivityScore = float64(80 - i*5)
		err := repo.Save(ctx, summary)
		require.NoError(t, err)
	}

	// Get recent 3
	recent, err := repo.GetRecent(ctx, userID, 3)
	require.NoError(t, err)
	assert.Len(t, recent, 3)

	// Should be ordered by week_start DESC
	assert.InDelta(t, 80.0, recent[0].AvgDailyProductivityScore, 0.01)
	assert.InDelta(t, 75.0, recent[1].AvgDailyProductivityScore, 0.01)
	assert.InDelta(t, 70.0, recent[2].AvgDailyProductivityScore, 0.01)
}

func TestSQLiteSummaryRepository_GetLatest(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSummaryRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))

	// Create summaries for 3 weeks
	for i := 0; i < 3; i++ {
		summary := createTestSummary(userID, weekStart.AddDate(0, 0, -i*7))
		summary.TotalTasksCompleted = 30 - i*5
		err := repo.Save(ctx, summary)
		require.NoError(t, err)
	}

	// Get latest should return most recent week's summary
	latest, err := repo.GetLatest(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, 30, latest.TotalTasksCompleted)
}

func TestSQLiteSummaryRepository_GetLatest_NoSummaries(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSummaryRepository(sqlDB)
	ctx := context.Background()

	latest, err := repo.GetLatest(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, latest)
}

func TestSQLiteSummaryRepository_Save_NilProductiveDays(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSummaryRepository(sqlDB)
	ctx := context.Background()

	weekStart := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -int(time.Now().Weekday()))
	summary := createTestSummary(userID, weekStart)
	summary.MostProductiveDay = nil
	summary.LeastProductiveDay = nil

	err := repo.Save(ctx, summary)
	require.NoError(t, err)

	found, err := repo.GetByWeek(ctx, userID, weekStart)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Nil(t, found.MostProductiveDay)
	assert.Nil(t, found.LeastProductiveDay)
}

func TestSQLiteSummaryRepository_Trends(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteSummaryRepository(sqlDB)
	ctx := context.Background()

	weekStart := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -int(time.Now().Weekday()))
	summary := createTestSummary(userID, weekStart)
	summary.ProductivityTrend = 1.15
	summary.FocusTrend = 0.85

	err := repo.Save(ctx, summary)
	require.NoError(t, err)

	found, err := repo.GetByWeek(ctx, userID, weekStart)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.InDelta(t, 1.15, found.ProductivityTrend, 0.001)
	assert.InDelta(t, 0.85, found.FocusTrend, 0.001)
}
