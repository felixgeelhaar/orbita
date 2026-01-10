package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAdjustMeetingCadenceHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("returns result with no updates when no active meetings", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewAdjustMeetingCadenceHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindActiveByUserID", txCtx, userID).Return([]*domain.Meeting{}, nil)

		cmd := AdjustMeetingCadenceCommand{
			UserID: userID,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.Evaluated)
		assert.Equal(t, 0, result.Updated)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("evaluates meetings without adjustment when conditions not met", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewAdjustMeetingCadenceHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		meeting := domain.RehydrateMeeting(
			uuid.New(),
			userID,
			"Weekly sync",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			10*time.Hour,
			nil, // No last held - should not adjust
			false,
			now.Add(-30*24*time.Hour),
			now,
		)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindActiveByUserID", txCtx, userID).Return([]*domain.Meeting{meeting}, nil)

		cmd := AdjustMeetingCadenceCommand{
			UserID: userID,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.Evaluated)
		assert.Equal(t, 0, result.Updated)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("increases cadence when meeting held very infrequently", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewAdjustMeetingCadenceHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		lastHeld := now.Add(-20 * 24 * time.Hour) // 20 days ago - exceeds 2x weekly threshold
		meeting := domain.RehydrateMeeting(
			uuid.New(),
			userID,
			"Weekly sync",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			10*time.Hour,
			&lastHeld,
			false,
			now.Add(-60*24*time.Hour),
			now,
		)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindActiveByUserID", txCtx, userID).Return([]*domain.Meeting{meeting}, nil)
		repo.On("Save", txCtx, meeting).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := AdjustMeetingCadenceCommand{
			UserID: userID,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.Evaluated)
		assert.Equal(t, 1, result.Updated)
		assert.Equal(t, 14, meeting.CadenceDays())

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("decreases cadence when meeting held very frequently", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewAdjustMeetingCadenceHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		lastHeld := now.Add(-3 * 24 * time.Hour) // 3 days ago - less than 0.6x biweekly threshold
		meeting := domain.RehydrateMeeting(
			uuid.New(),
			userID,
			"Biweekly sync",
			domain.CadenceBiweekly,
			14,
			30*time.Minute,
			10*time.Hour,
			&lastHeld,
			false,
			now.Add(-60*24*time.Hour),
			now,
		)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindActiveByUserID", txCtx, userID).Return([]*domain.Meeting{meeting}, nil)
		repo.On("Save", txCtx, meeting).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := AdjustMeetingCadenceCommand{
			UserID: userID,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.Evaluated)
		assert.Equal(t, 1, result.Updated)
		assert.Equal(t, 7, meeting.CadenceDays())

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("processes multiple meetings", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewAdjustMeetingCadenceHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meetings := []*domain.Meeting{
			createTestMeeting(userID, "Meeting 1"),
			createTestMeeting(userID, "Meeting 2"),
			createTestMeeting(userID, "Meeting 3"),
		}

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindActiveByUserID", txCtx, userID).Return(meetings, nil)

		cmd := AdjustMeetingCadenceCommand{
			UserID: userID,
		}

		result, err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, result.Evaluated)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository find fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewAdjustMeetingCadenceHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindActiveByUserID", txCtx, userID).Return(nil, errors.New("database error"))

		cmd := AdjustMeetingCadenceCommand{
			UserID: userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when begin transaction fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewAdjustMeetingCadenceHandler(repo, outboxRepo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := AdjustMeetingCadenceCommand{
			UserID: userID,
		}

		result, err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})
}

func TestNewAdjustMeetingCadenceHandler(t *testing.T) {
	repo := new(mockMeetingRepo)
	outboxRepo := new(mockOutboxRepo)
	uow := new(mockUnitOfWork)

	handler := NewAdjustMeetingCadenceHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}

func TestCadenceFromDays(t *testing.T) {
	tests := []struct {
		days     int
		expected domain.Cadence
	}{
		{7, domain.CadenceWeekly},
		{14, domain.CadenceBiweekly},
		{30, domain.CadenceMonthly},
		{21, domain.CadenceCustom},
		{10, domain.CadenceCustom},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			result := cadenceFromDays(tc.days)
			assert.Equal(t, tc.expected, result)
		})
	}
}
