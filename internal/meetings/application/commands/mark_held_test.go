package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkMeetingHeldHandler_Handle(t *testing.T) {
	userID := uuid.New()
	meetingID := uuid.New()

	t.Run("successfully marks meeting as held", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		uow := new(mockUnitOfWork)
		handler := NewMarkMeetingHeldHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Weekly sync")
		heldAt := time.Now()

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(nil)

		cmd := MarkMeetingHeldCommand{
			UserID:    userID,
			MeetingID: meetingID,
			HeldAt:    heldAt,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, meeting.LastHeldAt())
		assert.Equal(t, heldAt, *meeting.LastHeldAt())

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("updates existing last held timestamp", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		uow := new(mockUnitOfWork)
		handler := NewMarkMeetingHeldHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		previousHeld := now.Add(-7 * 24 * time.Hour)
		meeting := domain.RehydrateMeeting(
			meetingID,
			userID,
			"Weekly sync",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			10*time.Hour,
			&previousHeld,
			false,
			now.Add(-30*24*time.Hour),
			now,
		)

		newHeldAt := now

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(nil)

		cmd := MarkMeetingHeldCommand{
			UserID:    userID,
			MeetingID: meetingID,
			HeldAt:    newHeldAt,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		require.NotNil(t, meeting.LastHeldAt())
		assert.Equal(t, newHeldAt, *meeting.LastHeldAt())

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrMarkMeetingNotFound when meeting does not exist", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		uow := new(mockUnitOfWork)
		handler := NewMarkMeetingHeldHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(nil, nil)

		cmd := MarkMeetingHeldCommand{
			UserID:    userID,
			MeetingID: meetingID,
			HeldAt:    time.Now(),
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrMarkMeetingNotFound)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrMarkMeetingNotOwner when user does not own meeting", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		uow := new(mockUnitOfWork)
		handler := NewMarkMeetingHeldHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		differentUserID := uuid.New()
		meeting := createTestMeeting(differentUserID, "Someone else's meeting")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)

		cmd := MarkMeetingHeldCommand{
			UserID:    userID,
			MeetingID: meetingID,
			HeldAt:    time.Now(),
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrMarkMeetingNotOwner)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when marking archived meeting as held", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		uow := new(mockUnitOfWork)
		handler := NewMarkMeetingHeldHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		archivedMeeting := domain.RehydrateMeeting(
			meetingID,
			userID,
			"Archived meeting",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			10*time.Hour,
			nil,
			true, // Archived
			now.Add(-30*24*time.Hour),
			now,
		)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(archivedMeeting, nil)

		cmd := MarkMeetingHeldCommand{
			UserID:    userID,
			MeetingID: meetingID,
			HeldAt:    time.Now(),
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, domain.ErrMeetingArchived)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository find fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		uow := new(mockUnitOfWork)
		handler := NewMarkMeetingHeldHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(nil, errors.New("database error"))

		cmd := MarkMeetingHeldCommand{
			UserID:    userID,
			MeetingID: meetingID,
			HeldAt:    time.Now(),
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		uow := new(mockUnitOfWork)
		handler := NewMarkMeetingHeldHandler(repo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Weekly sync")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(errors.New("save error"))

		cmd := MarkMeetingHeldCommand{
			UserID:    userID,
			MeetingID: meetingID,
			HeldAt:    time.Now(),
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when begin transaction fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		uow := new(mockUnitOfWork)
		handler := NewMarkMeetingHeldHandler(repo, uow)

		ctx := context.Background()

		uow.On("Begin", ctx).Return(ctx, errors.New("transaction error"))

		cmd := MarkMeetingHeldCommand{
			UserID:    userID,
			MeetingID: meetingID,
			HeldAt:    time.Now(),
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction error")

		uow.AssertExpectations(t)
	})
}

func TestNewMarkMeetingHeldHandler(t *testing.T) {
	repo := new(mockMeetingRepo)
	uow := new(mockUnitOfWork)

	handler := NewMarkMeetingHeldHandler(repo, uow)

	require.NotNil(t, handler)
}
