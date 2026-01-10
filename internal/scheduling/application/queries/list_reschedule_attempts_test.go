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

// mockRescheduleAttemptRepo is a mock implementation of domain.RescheduleAttemptRepository.
type mockRescheduleAttemptRepo struct {
	mock.Mock
}

func (m *mockRescheduleAttemptRepo) Create(ctx context.Context, attempt domain.RescheduleAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *mockRescheduleAttemptRepo) ListByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.RescheduleAttempt, error) {
	args := m.Called(ctx, userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.RescheduleAttempt), args.Error(1)
}

func TestListRescheduleAttemptsHandler_Handle(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

	t.Run("successfully lists reschedule attempts", func(t *testing.T) {
		repo := new(mockRescheduleAttemptRepo)
		handler := NewListRescheduleAttemptsHandler(repo)

		ctx := context.Background()

		newStart := time.Date(2024, time.January, 15, 14, 0, 0, 0, time.UTC)
		newEnd := time.Date(2024, time.January, 15, 15, 0, 0, 0, time.UTC)

		attempts := []domain.RescheduleAttempt{
			{
				ID:          uuid.New(),
				UserID:      userID,
				ScheduleID:  uuid.New(),
				BlockID:     uuid.New(),
				AttemptType: domain.RescheduleAttemptAutoMissed,
				AttemptedAt: time.Now(),
				OldStart:    time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC),
				OldEnd:      time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC),
				NewStart:    &newStart,
				NewEnd:      &newEnd,
				Success:     true,
			},
			{
				ID:            uuid.New(),
				UserID:        userID,
				ScheduleID:    uuid.New(),
				BlockID:       uuid.New(),
				AttemptType:   domain.RescheduleAttemptAutoConflict,
				AttemptedAt:   time.Now(),
				OldStart:      time.Date(2024, time.January, 15, 11, 0, 0, 0, time.UTC),
				OldEnd:        time.Date(2024, time.January, 15, 12, 0, 0, 0, time.UTC),
				NewStart:      nil,
				NewEnd:        nil,
				Success:       false,
				FailureReason: "no available slots",
			},
		}

		repo.On("ListByUserAndDate", ctx, userID, date).Return(attempts, nil)

		query := ListRescheduleAttemptsQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 2)

		// Check first attempt (successful)
		assert.Equal(t, attempts[0].ID, result[0].ID)
		assert.Equal(t, attempts[0].BlockID, result[0].BlockID)
		assert.Equal(t, "auto-missed", result[0].AttemptType)
		assert.True(t, result[0].Success)
		assert.NotNil(t, result[0].NewStart)
		assert.NotNil(t, result[0].NewEnd)
		assert.Empty(t, result[0].FailureReason)

		// Check second attempt (failed)
		assert.Equal(t, attempts[1].ID, result[1].ID)
		assert.Equal(t, "auto-conflict", result[1].AttemptType)
		assert.False(t, result[1].Success)
		assert.Nil(t, result[1].NewStart)
		assert.Nil(t, result[1].NewEnd)
		assert.Equal(t, "no available slots", result[1].FailureReason)

		repo.AssertExpectations(t)
	})

	t.Run("returns empty list when no attempts exist", func(t *testing.T) {
		repo := new(mockRescheduleAttemptRepo)
		handler := NewListRescheduleAttemptsHandler(repo)

		ctx := context.Background()

		repo.On("ListByUserAndDate", ctx, userID, date).Return([]domain.RescheduleAttempt{}, nil)

		query := ListRescheduleAttemptsQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		assert.Empty(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		repo := new(mockRescheduleAttemptRepo)
		handler := NewListRescheduleAttemptsHandler(repo)

		ctx := context.Background()

		repo.On("ListByUserAndDate", ctx, userID, date).Return(nil, errors.New("database error"))

		query := ListRescheduleAttemptsQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
	})

	t.Run("correctly maps manual attempt type", func(t *testing.T) {
		repo := new(mockRescheduleAttemptRepo)
		handler := NewListRescheduleAttemptsHandler(repo)

		ctx := context.Background()

		newStart := time.Date(2024, time.January, 15, 16, 0, 0, 0, time.UTC)
		newEnd := time.Date(2024, time.January, 15, 17, 0, 0, 0, time.UTC)

		attempts := []domain.RescheduleAttempt{
			{
				ID:          uuid.New(),
				UserID:      userID,
				ScheduleID:  uuid.New(),
				BlockID:     uuid.New(),
				AttemptType: domain.RescheduleAttemptManual,
				AttemptedAt: time.Now(),
				OldStart:    time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC),
				OldEnd:      time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC),
				NewStart:    &newStart,
				NewEnd:      &newEnd,
				Success:     true,
			},
		}

		repo.On("ListByUserAndDate", ctx, userID, date).Return(attempts, nil)

		query := ListRescheduleAttemptsQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "manual", result[0].AttemptType)

		repo.AssertExpectations(t)
	})

	t.Run("correctly maps all attempt fields", func(t *testing.T) {
		repo := new(mockRescheduleAttemptRepo)
		handler := NewListRescheduleAttemptsHandler(repo)

		ctx := context.Background()

		attemptID := uuid.New()
		blockID := uuid.New()
		attemptedAt := time.Now()
		oldStart := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		oldEnd := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)
		newStart := time.Date(2024, time.January, 15, 14, 0, 0, 0, time.UTC)
		newEnd := time.Date(2024, time.January, 15, 15, 0, 0, 0, time.UTC)

		attempts := []domain.RescheduleAttempt{
			{
				ID:          attemptID,
				UserID:      userID,
				ScheduleID:  uuid.New(),
				BlockID:     blockID,
				AttemptType: domain.RescheduleAttemptAutoMissed,
				AttemptedAt: attemptedAt,
				OldStart:    oldStart,
				OldEnd:      oldEnd,
				NewStart:    &newStart,
				NewEnd:      &newEnd,
				Success:     true,
			},
		}

		repo.On("ListByUserAndDate", ctx, userID, date).Return(attempts, nil)

		query := ListRescheduleAttemptsQuery{
			UserID: userID,
			Date:   date,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 1)

		dto := result[0]
		assert.Equal(t, attemptID, dto.ID)
		assert.Equal(t, blockID, dto.BlockID)
		assert.Equal(t, "auto-missed", dto.AttemptType)
		assert.Equal(t, attemptedAt, dto.AttemptedAt)
		assert.Equal(t, oldStart, dto.OldStart)
		assert.Equal(t, oldEnd, dto.OldEnd)
		assert.Equal(t, &newStart, dto.NewStart)
		assert.Equal(t, &newEnd, dto.NewEnd)
		assert.True(t, dto.Success)
		assert.Empty(t, dto.FailureReason)

		repo.AssertExpectations(t)
	})
}

func TestNewListRescheduleAttemptsHandler(t *testing.T) {
	repo := new(mockRescheduleAttemptRepo)

	handler := NewListRescheduleAttemptsHandler(repo)

	require.NotNil(t, handler)
}
