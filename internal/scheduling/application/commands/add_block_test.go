package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
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

// mockSchedulingOutboxRepo is a mock implementation of outbox.Repository.
type mockSchedulingOutboxRepo struct {
	mock.Mock
}

func (m *mockSchedulingOutboxRepo) Save(ctx context.Context, msg *outbox.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockSchedulingOutboxRepo) SaveBatch(ctx context.Context, msgs []*outbox.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *mockSchedulingOutboxRepo) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockSchedulingOutboxRepo) MarkPublished(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockSchedulingOutboxRepo) MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error {
	args := m.Called(ctx, id, err, nextRetryAt)
	return args.Error(0)
}

func (m *mockSchedulingOutboxRepo) MarkDead(ctx context.Context, id int64, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *mockSchedulingOutboxRepo) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, maxRetries, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockSchedulingOutboxRepo) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	args := m.Called(ctx, olderThanDays)
	return args.Get(0).(int64), args.Error(1)
}

// mockSchedulingUnitOfWork is a mock implementation of UnitOfWork.
type mockSchedulingUnitOfWork struct {
	mock.Mock
}

func (m *mockSchedulingUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *mockSchedulingUnitOfWork) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockSchedulingUnitOfWork) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func createTestSchedule(userID uuid.UUID, date time.Time) *domain.Schedule {
	now := time.Now()
	return domain.RehydrateSchedule(
		uuid.New(),
		userID,
		date,
		nil,
		now.Add(-24*time.Hour),
		now,
	)
}

func TestAddBlockHandler_Handle(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC)

	t.Run("successfully adds a block to existing schedule", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewAddBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule := createTestSchedule(userID, date)

		startTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(schedule, nil)
		repo.On("Save", txCtx, schedule).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := AddBlockCommand{
			UserID:      userID,
			Date:        date,
			BlockType:   "task",
			ReferenceID: uuid.New(),
			Title:       "Work on project",
			StartTime:   startTime,
			EndTime:     endTime,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.ScheduleID)
		assert.NotEqual(t, uuid.Nil, result.BlockID)

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("creates new schedule when none exists", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewAddBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		startTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(nil, nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Schedule")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := AddBlockCommand{
			UserID:      userID,
			Date:        date,
			BlockType:   "task",
			ReferenceID: uuid.New(),
			Title:       "New task",
			StartTime:   startTime,
			EndTime:     endTime,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.BlockID)

		repo.AssertExpectations(t)
	})

	t.Run("fails with invalid time range", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewAddBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule := createTestSchedule(userID, date)

		// End time before start time
		startTime := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(schedule, nil)

		cmd := AddBlockCommand{
			UserID:      userID,
			Date:        date,
			BlockType:   "task",
			ReferenceID: uuid.New(),
			Title:       "Invalid block",
			StartTime:   startTime,
			EndTime:     endTime,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrInvalidTimeRange)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails with too short duration", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewAddBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule := createTestSchedule(userID, date)

		// Only 2 minutes duration
		startTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, time.January, 15, 9, 2, 0, 0, time.UTC)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(schedule, nil)

		cmd := AddBlockCommand{
			UserID:      userID,
			Date:        date,
			BlockType:   "task",
			ReferenceID: uuid.New(),
			Title:       "Short block",
			StartTime:   startTime,
			EndTime:     endTime,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrTimeBlockTooShort)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository find fails", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewAddBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		startTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(nil, errors.New("database error"))

		cmd := AddBlockCommand{
			UserID:      userID,
			Date:        date,
			BlockType:   "task",
			ReferenceID: uuid.New(),
			Title:       "Test",
			StartTime:   startTime,
			EndTime:     endTime,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewAddBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		schedule := createTestSchedule(userID, date)

		startTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByUserAndDate", txCtx, userID, date).Return(schedule, nil)
		repo.On("Save", txCtx, schedule).Return(errors.New("save error"))

		cmd := AddBlockCommand{
			UserID:      userID,
			Date:        date,
			BlockType:   "task",
			ReferenceID: uuid.New(),
			Title:       "Test",
			StartTime:   startTime,
			EndTime:     endTime,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "save error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when begin transaction fails", func(t *testing.T) {
		repo := new(mockScheduleRepo)
		outboxRepo := new(mockSchedulingOutboxRepo)
		uow := new(mockSchedulingUnitOfWork)
		handler := NewAddBlockHandler(repo, outboxRepo, uow)

		ctx := context.Background()

		startTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := AddBlockCommand{
			UserID:      userID,
			Date:        date,
			BlockType:   "task",
			ReferenceID: uuid.New(),
			Title:       "Test",
			StartTime:   startTime,
			EndTime:     endTime,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})
}

func TestNewAddBlockHandler(t *testing.T) {
	repo := new(mockScheduleRepo)
	outboxRepo := new(mockSchedulingOutboxRepo)
	uow := new(mockSchedulingUnitOfWork)

	handler := NewAddBlockHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
