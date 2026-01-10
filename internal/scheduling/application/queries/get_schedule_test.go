package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockScheduleRepo is a mock implementation of domain.ScheduleRepository.
type mockScheduleRepo struct {
	mock.Mock
}

func (m *mockScheduleRepo) Save(ctx context.Context, schedule *domain.Schedule) error {
	args := m.Called(ctx, schedule)
	return args.Error(0)
}

func (m *mockScheduleRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Schedule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Schedule), args.Error(1)
}

func (m *mockScheduleRepo) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.Schedule, error) {
	args := m.Called(ctx, userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Schedule), args.Error(1)
}

func (m *mockScheduleRepo) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*domain.Schedule, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Schedule), args.Error(1)
}

func (m *mockScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func createTestScheduleWithBlocks(userID uuid.UUID, date time.Time) *domain.Schedule {
	now := time.Now()
	scheduleID := uuid.New()

	startTime1 := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, time.UTC)
	endTime1 := time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, time.UTC)
	block1 := domain.RehydrateTimeBlock(
		uuid.New(),
		userID,
		scheduleID,
		domain.BlockTypeTask,
		uuid.New(),
		"Task Block",
		startTime1,
		endTime1,
		true, // completed
		false,
		now,
		now,
	)

	startTime2 := time.Date(date.Year(), date.Month(), date.Day(), 14, 0, 0, 0, time.UTC)
	endTime2 := time.Date(date.Year(), date.Month(), date.Day(), 15, 0, 0, 0, time.UTC)
	block2 := domain.RehydrateTimeBlock(
		uuid.New(),
		userID,
		scheduleID,
		domain.BlockTypeHabit,
		uuid.New(),
		"Habit Block",
		startTime2,
		endTime2,
		false,
		true, // missed
		now,
		now,
	)

	startTime3 := time.Date(date.Year(), date.Month(), date.Day(), 16, 0, 0, 0, time.UTC)
	endTime3 := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, time.UTC)
	block3 := domain.RehydrateTimeBlock(
		uuid.New(),
		userID,
		scheduleID,
		domain.BlockTypeMeeting,
		uuid.New(),
		"Meeting Block",
		startTime3,
		endTime3,
		false, // pending
		false,
		now,
		now,
	)

	return domain.RehydrateSchedule(
		scheduleID,
		userID,
		date,
		[]*domain.TimeBlock{block1, block2, block3},
		now.Add(-24*time.Hour),
		now,
	)
}

func TestGetScheduleHandler_Handle(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

	t.Run("successfully retrieves schedule with blocks", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewGetScheduleHandler(repo)

		ctx := context.Background()
		schedule := createTestScheduleWithBlocks(userID, date)

		repo.On("FindByUserAndDate", ctx, userID, date).Return(schedule, nil)

		query := GetScheduleQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, schedule.ID(), result.ID)
		assert.Equal(t, date, result.Date)
		assert.Len(t, result.Blocks, 3)
		assert.Equal(t, 180, result.TotalScheduledMins) // 3 blocks * 60 mins
		assert.Equal(t, 1, result.CompletedCount)
		assert.Equal(t, 1, result.MissedCount)
		assert.Equal(t, 1, result.PendingCount)

		repo.AssertExpectations(t)
	})

	t.Run("returns empty schedule when none exists", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewGetScheduleHandler(repo)

		ctx := context.Background()

		repo.On("FindByUserAndDate", ctx, userID, date).Return(nil, nil)

		query := GetScheduleQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, uuid.Nil, result.ID)
		assert.Equal(t, date, result.Date)
		assert.Empty(t, result.Blocks)
		assert.Equal(t, 0, result.TotalScheduledMins)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewGetScheduleHandler(repo)

		ctx := context.Background()

		repo.On("FindByUserAndDate", ctx, userID, date).Return(nil, errors.New("database error"))

		query := GetScheduleQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
	})

	t.Run("correctly maps block types", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewGetScheduleHandler(repo)

		ctx := context.Background()
		schedule := createTestScheduleWithBlocks(userID, date)

		repo.On("FindByUserAndDate", ctx, userID, date).Return(schedule, nil)

		query := GetScheduleQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result.Blocks, 3)

		// Verify block types are mapped correctly
		blockTypes := make(map[string]bool)
		for _, block := range result.Blocks {
			blockTypes[block.BlockType] = true
		}
		assert.True(t, blockTypes["task"])
		assert.True(t, blockTypes["habit"])
		assert.True(t, blockTypes["meeting"])

		repo.AssertExpectations(t)
	})

	t.Run("correctly calculates block durations", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewGetScheduleHandler(repo)

		ctx := context.Background()
		schedule := createTestScheduleWithBlocks(userID, date)

		repo.On("FindByUserAndDate", ctx, userID, date).Return(schedule, nil)

		query := GetScheduleQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		for _, block := range result.Blocks {
			assert.Equal(t, 60, block.DurationMin) // All blocks are 1 hour
		}

		repo.AssertExpectations(t)
	})
}

func TestNewGetScheduleHandler(t *testing.T) {
	repo := new(mockScheduleRepo)

	handler := NewGetScheduleHandler(repo)

	require.NotNil(t, handler)
}
