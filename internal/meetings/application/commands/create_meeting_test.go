package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockMeetingRepo is a mock implementation of domain.Repository.
type mockMeetingRepo struct {
	mock.Mock
}

func (m *mockMeetingRepo) Save(ctx context.Context, meeting *domain.Meeting) error {
	args := m.Called(ctx, meeting)
	return args.Error(0)
}

func (m *mockMeetingRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Meeting), args.Error(1)
}

func (m *mockMeetingRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Meeting), args.Error(1)
}

func (m *mockMeetingRepo) FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Meeting, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Meeting), args.Error(1)
}

// mockOutboxRepo is a mock implementation of outbox.Repository.
type mockOutboxRepo struct {
	mock.Mock
}

func (m *mockOutboxRepo) Save(ctx context.Context, msg *outbox.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockOutboxRepo) SaveBatch(ctx context.Context, msgs []*outbox.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *mockOutboxRepo) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockOutboxRepo) MarkPublished(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error {
	args := m.Called(ctx, id, err, nextRetryAt)
	return args.Error(0)
}

func (m *mockOutboxRepo) MarkDead(ctx context.Context, id int64, reason string) error {
	args := m.Called(ctx, id, reason)
	return args.Error(0)
}

func (m *mockOutboxRepo) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	args := m.Called(ctx, maxRetries, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outbox.Message), args.Error(1)
}

func (m *mockOutboxRepo) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	args := m.Called(ctx, olderThanDays)
	return args.Get(0).(int64), args.Error(1)
}

// mockUnitOfWork is a mock implementation of UnitOfWork.
type mockUnitOfWork struct {
	mock.Mock
}

func (m *mockUnitOfWork) Begin(ctx context.Context) (context.Context, error) {
	args := m.Called(ctx)
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *mockUnitOfWork) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockUnitOfWork) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestCreateMeetingHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully creates a meeting", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Meeting")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CreateMeetingCommand{
			UserID:        userID,
			Name:          "Weekly 1:1 with John",
			Cadence:       "weekly",
			DurationMins:  30,
			PreferredTime: "10:00",
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.MeetingID)

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("creates meeting with default cadence when invalid", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Meeting")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CreateMeetingCommand{
			UserID:        userID,
			Name:          "Team sync",
			Cadence:       "invalid",
			DurationMins:  60,
			PreferredTime: "",
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("creates meeting with default preferred time", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Meeting")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := CreateMeetingCommand{
			UserID:        userID,
			Name:          "Morning standup",
			Cadence:       "weekly",
			DurationMins:  15,
			PreferredTime: "",
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails with invalid preferred time format", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)

		cmd := CreateMeetingCommand{
			UserID:        userID,
			Name:          "Meeting",
			Cadence:       "weekly",
			DurationMins:  30,
			PreferredTime: "invalid-time",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid preferred time format")

		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Meeting")).Return(errors.New("database error"))

		cmd := CreateMeetingCommand{
			UserID:        userID,
			Name:          "Weekly sync",
			Cadence:       "weekly",
			DurationMins:  30,
			PreferredTime: "14:00",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when outbox save fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("Save", txCtx, mock.AnythingOfType("*domain.Meeting")).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(errors.New("outbox error"))

		cmd := CreateMeetingCommand{
			UserID:        userID,
			Name:          "Weekly sync",
			Cadence:       "weekly",
			DurationMins:  30,
			PreferredTime: "14:00",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "outbox error")

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when begin transaction fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := CreateMeetingCommand{
			UserID:        userID,
			Name:          "Meeting",
			Cadence:       "weekly",
			DurationMins:  30,
			PreferredTime: "10:00",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})

	t.Run("fails with empty meeting name", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)

		cmd := CreateMeetingCommand{
			UserID:        userID,
			Name:          "",
			Cadence:       "weekly",
			DurationMins:  30,
			PreferredTime: "10:00",
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrMeetingEmptyName)

		uow.AssertExpectations(t)
	})
}

func TestNewCreateMeetingHandler(t *testing.T) {
	repo := new(mockMeetingRepo)
	outboxRepo := new(mockOutboxRepo)
	uow := new(mockUnitOfWork)

	handler := NewCreateMeetingHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
