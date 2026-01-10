package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetTrendsHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("returns trends with improving productivity", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		// Current period: higher scores (improving)
		currentSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -1), 85),
			createTestSnapshot(userID, today.AddDate(0, 0, -2), 80),
			createTestSnapshot(userID, today.AddDate(0, 0, -3), 75),
		}

		// Previous period: lower scores
		previousSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -8), 65),
			createTestSnapshot(userID, today.AddDate(0, 0, -9), 60),
			createTestSnapshot(userID, today.AddDate(0, 0, -10), 55),
		}

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(currentSnapshots, nil).Once()
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(previousSnapshots, nil).Once()

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Snapshots, 3)
		assert.Equal(t, "up", result.ProductivityTrend.Direction)
		assert.Greater(t, result.ProductivityTrend.Change, float64(0))
		assert.Greater(t, result.CurrentPeriodAvg, result.PreviousPeriodAvg)

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("returns trends with declining productivity", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		// Current period: lower scores (declining)
		currentSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -1), 50),
			createTestSnapshot(userID, today.AddDate(0, 0, -2), 55),
			createTestSnapshot(userID, today.AddDate(0, 0, -3), 45),
		}

		// Previous period: higher scores
		previousSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -8), 80),
			createTestSnapshot(userID, today.AddDate(0, 0, -9), 85),
			createTestSnapshot(userID, today.AddDate(0, 0, -10), 75),
		}

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(currentSnapshots, nil).Once()
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(previousSnapshots, nil).Once()

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "down", result.ProductivityTrend.Direction)
		assert.Less(t, result.ProductivityTrend.Change, float64(0))
		assert.Less(t, result.CurrentPeriodAvg, result.PreviousPeriodAvg)

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("returns stable trend when change is minimal", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		// Both periods have similar scores
		currentSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -1), 75),
			createTestSnapshot(userID, today.AddDate(0, 0, -2), 74),
		}

		previousSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -8), 73),
			createTestSnapshot(userID, today.AddDate(0, 0, -9), 74),
		}

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(currentSnapshots, nil).Once()
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(previousSnapshots, nil).Once()

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "stable", result.ProductivityTrend.Direction)

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("uses default days when not specified", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil)

		query := GetTrendsQuery{
			UserID: userID,
			Days:   0, // Should default to 14
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("finds best and worst days", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		bestDay := today.AddDate(0, 0, -2)
		worstDay := today.AddDate(0, 0, -4)

		currentSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -1), 75),
			createTestSnapshot(userID, bestDay, 95),
			createTestSnapshot(userID, today.AddDate(0, 0, -3), 70),
			createTestSnapshot(userID, worstDay, 40),
		}
		currentSnapshots[1].SnapshotDate = bestDay
		currentSnapshots[3].SnapshotDate = worstDay

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(currentSnapshots, nil).Once()
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil).Once()

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.BestDay)
		require.NotNil(t, result.WorstDay)
		assert.Equal(t, 95, result.BestDay.ProductivityScore)
		assert.Equal(t, 40, result.WorstDay.ProductivityScore)

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("returns empty result when no snapshots exist", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil)

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Snapshots)
		assert.Nil(t, result.BestDay)
		assert.Nil(t, result.WorstDay)
		assert.Equal(t, "stable", result.ProductivityTrend.Direction)

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("fails when current period fetch returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, errors.New("database error")).Once()

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("fails when previous period fetch returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil).Once()
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, errors.New("database error")).Once()

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("calculates task completion trend", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		currentSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -1), 80),
		}
		currentSnapshots[0].TaskCompletionRate = 0.9

		previousSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -8), 70),
		}
		previousSnapshots[0].TaskCompletionRate = 0.6

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(currentSnapshots, nil).Once()
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(previousSnapshots, nil).Once()

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "up", result.TaskCompletionTrend.Direction)
		assert.Greater(t, result.TaskCompletionTrend.CurrentAvg, result.TaskCompletionTrend.PreviousAvg)

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("calculates focus time trend", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		handler := NewGetTrendsHandler(snapshotRepo)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		currentSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -1), 80),
		}
		currentSnapshots[0].TotalFocusMinutes = 180

		previousSnapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, today.AddDate(0, 0, -8), 70),
		}
		previousSnapshots[0].TotalFocusMinutes = 120

		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(currentSnapshots, nil).Once()
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(previousSnapshots, nil).Once()

		query := GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "up", result.FocusTimeTrend.Direction)
		assert.Greater(t, result.FocusTimeTrend.CurrentAvg, result.FocusTimeTrend.PreviousAvg)

		snapshotRepo.AssertExpectations(t)
	})
}

func TestCalculateTrend(t *testing.T) {
	t.Run("returns up for significant increase", func(t *testing.T) {
		current := []float64{80, 85, 90}
		previous := []float64{60, 65, 70}

		result := calculateTrend(current, previous)

		assert.Equal(t, "up", result.Direction)
		assert.Greater(t, result.Change, float64(0))
	})

	t.Run("returns down for significant decrease", func(t *testing.T) {
		current := []float64{50, 55, 60}
		previous := []float64{80, 85, 90}

		result := calculateTrend(current, previous)

		assert.Equal(t, "down", result.Direction)
		assert.Less(t, result.Change, float64(0))
	})

	t.Run("returns stable for minimal change", func(t *testing.T) {
		current := []float64{75, 76, 77}
		previous := []float64{74, 75, 76}

		result := calculateTrend(current, previous)

		assert.Equal(t, "stable", result.Direction)
	})

	t.Run("handles empty slices", func(t *testing.T) {
		result := calculateTrend([]float64{}, []float64{})

		assert.Equal(t, "stable", result.Direction)
		assert.Equal(t, float64(0), result.CurrentAvg)
		assert.Equal(t, float64(0), result.PreviousAvg)
	})

	t.Run("handles zero previous average", func(t *testing.T) {
		current := []float64{80, 85}
		previous := []float64{}

		result := calculateTrend(current, previous)

		assert.Equal(t, "stable", result.Direction)
		assert.Equal(t, float64(0), result.Change)
	})
}

func TestAverage(t *testing.T) {
	t.Run("calculates average correctly", func(t *testing.T) {
		values := []float64{10, 20, 30}
		result := average(values)
		assert.Equal(t, float64(20), result)
	})

	t.Run("returns zero for empty slice", func(t *testing.T) {
		result := average([]float64{})
		assert.Equal(t, float64(0), result)
	})

	t.Run("handles single value", func(t *testing.T) {
		result := average([]float64{42})
		assert.Equal(t, float64(42), result)
	})
}

func TestFindBestWorstDays(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	t.Run("finds best and worst days correctly", func(t *testing.T) {
		snapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, now, 70),
			createTestSnapshot(userID, now.AddDate(0, 0, -1), 95),
			createTestSnapshot(userID, now.AddDate(0, 0, -2), 40),
		}
		snapshots[0].ProductivityScore = 70
		snapshots[1].ProductivityScore = 95
		snapshots[2].ProductivityScore = 40

		best, worst := findBestWorstDays(snapshots)

		require.NotNil(t, best)
		require.NotNil(t, worst)
		assert.Equal(t, 95, best.ProductivityScore)
		assert.Equal(t, 40, worst.ProductivityScore)
	})

	t.Run("returns nil for empty snapshots", func(t *testing.T) {
		best, worst := findBestWorstDays([]*domain.ProductivitySnapshot{})

		assert.Nil(t, best)
		assert.Nil(t, worst)
	})

	t.Run("handles single snapshot", func(t *testing.T) {
		snapshots := []*domain.ProductivitySnapshot{
			createTestSnapshot(userID, now, 75),
		}

		best, worst := findBestWorstDays(snapshots)

		require.NotNil(t, best)
		require.NotNil(t, worst)
		assert.Equal(t, 75, best.ProductivityScore)
		assert.Equal(t, 75, worst.ProductivityScore)
	})
}

func TestNewGetTrendsHandler(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)

	handler := NewGetTrendsHandler(snapshotRepo)

	require.NotNil(t, handler)
}
