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

func createTestGoal(userID uuid.UUID, periodStart time.Time) *domain.ProductivityGoal {
	return &domain.ProductivityGoal{
		ID:           uuid.New(),
		UserID:       userID,
		GoalType:     domain.GoalTypeWeeklyTasks,
		TargetValue:  50,
		CurrentValue: 25,
		PeriodType:   domain.PeriodTypeWeekly,
		PeriodStart:  periodStart,
		PeriodEnd:    periodStart.AddDate(0, 0, 7),
		Achieved:     false,
		AchievedAt:   nil,
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
		UpdatedAt:    time.Now().UTC().Truncate(time.Second),
	}
}

func TestNewSQLiteGoalRepository(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteGoalRepository(sqlDB)
	assert.NotNil(t, repo)
}

func TestSQLiteGoalRepository_Create(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	goal := createTestGoal(userID, today)

	err := repo.Create(ctx, goal)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.GetByID(ctx, goal.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, goal.GoalType, found.GoalType)
	assert.Equal(t, goal.TargetValue, found.TargetValue)
	assert.Equal(t, goal.CurrentValue, found.CurrentValue)
	assert.False(t, found.Achieved)
}

func TestSQLiteGoalRepository_GetByID_NotFound(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteGoalRepository_Update(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	goal := createTestGoal(userID, today)

	err := repo.Create(ctx, goal)
	require.NoError(t, err)

	// Update the goal - achieve it
	goal.CurrentValue = 50
	goal.Achieved = true
	achievedAt := time.Now().UTC().Truncate(time.Second)
	goal.AchievedAt = &achievedAt

	err = repo.Update(ctx, goal)
	require.NoError(t, err)

	// Verify the update
	found, err := repo.GetByID(ctx, goal.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 50, found.CurrentValue)
	assert.True(t, found.Achieved)
	require.NotNil(t, found.AchievedAt)
}

func TestSQLiteGoalRepository_GetActive(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create an active goal (not achieved, not expired)
	activeGoal := createTestGoal(userID, today)
	err := repo.Create(ctx, activeGoal)
	require.NoError(t, err)

	// Create an achieved goal
	achievedGoal := createTestGoal(userID, today.AddDate(0, 0, -7))
	achievedGoal.Achieved = true
	achievedAt := time.Now().UTC()
	achievedGoal.AchievedAt = &achievedAt
	err = repo.Create(ctx, achievedGoal)
	require.NoError(t, err)

	// Create an expired goal (period ended in the past)
	expiredGoal := createTestGoal(userID, today.AddDate(0, 0, -30))
	expiredGoal.PeriodEnd = today.AddDate(0, 0, -23)
	err = repo.Create(ctx, expiredGoal)
	require.NoError(t, err)

	// Get active should return only the active one
	active, err := repo.GetActive(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, activeGoal.ID, active[0].ID)
}

func TestSQLiteGoalRepository_GetByPeriod(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create goals in different periods
	for i := 0; i < 4; i++ {
		goal := createTestGoal(userID, today.AddDate(0, 0, i*7))
		goal.PeriodEnd = today.AddDate(0, 0, i*7+7)
		err := repo.Create(ctx, goal)
		require.NoError(t, err)
	}

	// Get goals within first 3 weeks
	goals, err := repo.GetByPeriod(ctx, userID, today, today.AddDate(0, 0, 21))
	require.NoError(t, err)
	assert.Len(t, goals, 3)
}

func TestSQLiteGoalRepository_GetAchieved(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	// Create some achieved goals
	for i := 0; i < 3; i++ {
		goal := createTestGoal(userID, today.AddDate(0, 0, -i*7))
		goal.PeriodEnd = today.AddDate(0, 0, -i*7+7)
		goal.Achieved = true
		achievedAt := time.Now().Add(-time.Duration(i) * 24 * time.Hour)
		goal.AchievedAt = &achievedAt
		err := repo.Create(ctx, goal)
		require.NoError(t, err)
	}

	// Create a non-achieved goal
	activeGoal := createTestGoal(userID, today)
	err := repo.Create(ctx, activeGoal)
	require.NoError(t, err)

	// Get achieved with limit 2
	achieved, err := repo.GetAchieved(ctx, userID, 2)
	require.NoError(t, err)
	assert.Len(t, achieved, 2)

	// All should be achieved
	for _, g := range achieved {
		assert.True(t, g.Achieved)
	}
}

func TestSQLiteGoalRepository_Delete(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)
	goal := createTestGoal(userID, today)

	err := repo.Create(ctx, goal)
	require.NoError(t, err)

	// Delete
	err = repo.Delete(ctx, goal.ID)
	require.NoError(t, err)

	// Verify deleted
	found, err := repo.GetByID(ctx, goal.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestSQLiteGoalRepository_GoalTypes(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	goalTypes := []domain.GoalType{
		domain.GoalTypeDailyTasks,
		domain.GoalTypeDailyFocusMinutes,
		domain.GoalTypeWeeklyTasks,
		domain.GoalTypeHabitStreak,
	}

	for _, gt := range goalTypes {
		goal := createTestGoal(userID, today)
		goal.GoalType = gt
		err := repo.Create(ctx, goal)
		require.NoError(t, err, "Should create goal with type %s", gt)

		found, err := repo.GetByID(ctx, goal.ID)
		require.NoError(t, err)
		assert.Equal(t, gt, found.GoalType)
	}
}

func TestSQLiteGoalRepository_PeriodTypes(t *testing.T) {
	sqlDB := setupInsightsTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createInsightsTestUser(t, sqlDB, userID)

	repo := NewSQLiteGoalRepository(sqlDB)
	ctx := context.Background()

	today := time.Now().Truncate(24 * time.Hour)

	periodTypes := []domain.PeriodType{
		domain.PeriodTypeDaily,
		domain.PeriodTypeWeekly,
		domain.PeriodTypeMonthly,
	}

	for _, pt := range periodTypes {
		goal := createTestGoal(userID, today)
		goal.PeriodType = pt
		err := repo.Create(ctx, goal)
		require.NoError(t, err, "Should create goal with period type %s", pt)

		found, err := repo.GetByID(ctx, goal.ID)
		require.NoError(t, err)
		assert.Equal(t, pt, found.PeriodType)
	}
}
