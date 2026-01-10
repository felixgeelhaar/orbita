package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAvailableSlotsHandler_Handle(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)
	dayStart := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
	dayEnd := time.Date(2024, time.January, 15, 17, 0, 0, 0, time.UTC)

	t.Run("returns entire day when no schedule exists", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewFindAvailableSlotsHandler(repo)

		ctx := context.Background()

		repo.On("FindByUserAndDate", ctx, userID, date).Return(nil, nil)

		query := FindAvailableSlotsQuery{
			UserID:      userID,
			Date:        date,
			DayStart:    dayStart,
			DayEnd:      dayEnd,
			MinDuration: 30 * time.Minute,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, dayStart, result[0].Start)
		assert.Equal(t, dayEnd, result[0].End)
		assert.Equal(t, 480, result[0].DurationMin) // 8 hours in minutes

		repo.AssertExpectations(t)
	})

	t.Run("returns empty when day too short for min duration", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewFindAvailableSlotsHandler(repo)

		ctx := context.Background()
		shortDayEnd := dayStart.Add(15 * time.Minute)

		repo.On("FindByUserAndDate", ctx, userID, date).Return(nil, nil)

		query := FindAvailableSlotsQuery{
			UserID:      userID,
			Date:        date,
			DayStart:    dayStart,
			DayEnd:      shortDayEnd,
			MinDuration: 30 * time.Minute,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		assert.Empty(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("returns available slots around scheduled blocks", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewFindAvailableSlotsHandler(repo)

		ctx := context.Background()

		// Create schedule with a block from 10:00-11:00
		now := time.Now()
		scheduleID := uuid.New()
		blockStart := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)
		blockEnd := time.Date(2024, time.January, 15, 11, 0, 0, 0, time.UTC)
		block := domain.RehydrateTimeBlock(
			uuid.New(),
			userID,
			scheduleID,
			domain.BlockTypeTask,
			uuid.New(),
			"Test Block",
			blockStart,
			blockEnd,
			false,
			false,
			now,
			now,
		)
		schedule := domain.RehydrateSchedule(
			scheduleID,
			userID,
			date,
			[]*domain.TimeBlock{block},
			now.Add(-24*time.Hour),
			now,
		)

		repo.On("FindByUserAndDate", ctx, userID, date).Return(schedule, nil)

		query := FindAvailableSlotsQuery{
			UserID:      userID,
			Date:        date,
			DayStart:    dayStart,
			DayEnd:      dayEnd,
			MinDuration: 30 * time.Minute,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 2)

		// First slot: 9:00 - 10:00
		assert.Equal(t, dayStart, result[0].Start)
		assert.Equal(t, blockStart, result[0].End)
		assert.Equal(t, 60, result[0].DurationMin)

		// Second slot: 11:00 - 17:00
		assert.Equal(t, blockEnd, result[1].Start)
		assert.Equal(t, dayEnd, result[1].End)
		assert.Equal(t, 360, result[1].DurationMin)

		repo.AssertExpectations(t)
	})

	t.Run("filters slots by minimum duration", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewFindAvailableSlotsHandler(repo)

		ctx := context.Background()

		// Create schedule with blocks that leave only a 30-min gap
		now := time.Now()
		scheduleID := uuid.New()
		block1Start := time.Date(2024, time.January, 15, 9, 30, 0, 0, time.UTC)
		block1End := time.Date(2024, time.January, 15, 12, 0, 0, 0, time.UTC)
		block1 := domain.RehydrateTimeBlock(
			uuid.New(),
			userID,
			scheduleID,
			domain.BlockTypeTask,
			uuid.New(),
			"Block 1",
			block1Start,
			block1End,
			false,
			false,
			now,
			now,
		)

		schedule := domain.RehydrateSchedule(
			scheduleID,
			userID,
			date,
			[]*domain.TimeBlock{block1},
			now.Add(-24*time.Hour),
			now,
		)

		repo.On("FindByUserAndDate", ctx, userID, date).Return(schedule, nil)

		// Request slots of at least 2 hours
		query := FindAvailableSlotsQuery{
			UserID:      userID,
			Date:        date,
			DayStart:    dayStart,
			DayEnd:      dayEnd,
			MinDuration: 2 * time.Hour,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		// Should only return the 12:00-17:00 slot (5 hours), not the 9:00-9:30 slot (30 min)
		require.Len(t, result, 1)
		assert.Equal(t, block1End, result[0].Start)
		assert.Equal(t, dayEnd, result[0].End)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewFindAvailableSlotsHandler(repo)

		ctx := context.Background()

		repo.On("FindByUserAndDate", ctx, userID, date).Return(nil, errors.New("database error"))

		query := FindAvailableSlotsQuery{
			UserID:      userID,
			Date:        date,
			DayStart:    dayStart,
			DayEnd:      dayEnd,
			MinDuration: 30 * time.Minute,
		}

		result, err := handler.Handle(ctx, query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
	})

	t.Run("handles schedule with no blocks", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		handler := NewFindAvailableSlotsHandler(repo)

		ctx := context.Background()

		// Create empty schedule
		now := time.Now()
		scheduleID := uuid.New()
		schedule := domain.RehydrateSchedule(
			scheduleID,
			userID,
			date,
			[]*domain.TimeBlock{},
			now.Add(-24*time.Hour),
			now,
		)

		repo.On("FindByUserAndDate", ctx, userID, date).Return(schedule, nil)

		query := FindAvailableSlotsQuery{
			UserID:      userID,
			Date:        date,
			DayStart:    dayStart,
			DayEnd:      dayEnd,
			MinDuration: 30 * time.Minute,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, dayStart, result[0].Start)
		assert.Equal(t, dayEnd, result[0].End)

		repo.AssertExpectations(t)
	})
}

func TestNewFindAvailableSlotsHandler(t *testing.T) {
	repo := new(mockScheduleRepo)

	handler := NewFindAvailableSlotsHandler(repo)

	require.NotNil(t, handler)
}
