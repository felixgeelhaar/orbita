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

func TestArchiveMeetingHandler_Handle(t *testing.T) {
	userID := uuid.New()
	meetingID := uuid.New()

	t.Run("successfully archives a meeting", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewArchiveMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Weekly sync")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := ArchiveMeetingCommand{
			MeetingID: meetingID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.True(t, meeting.IsArchived())

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("archives already archived meeting idempotently", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewArchiveMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		now := time.Now()
		archivedMeeting := domain.RehydrateMeeting(
			meetingID,
			userID,
			"Already archived",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			10*time.Hour,
			nil,
			true, // Already archived
			now,
			now,
		)

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(archivedMeeting, nil)
		repo.On("Save", txCtx, archivedMeeting).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := ArchiveMeetingCommand{
			MeetingID: meetingID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.True(t, archivedMeeting.IsArchived())

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrArchiveMeetingNotFound when meeting does not exist", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewArchiveMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(nil, nil)

		cmd := ArchiveMeetingCommand{
			MeetingID: meetingID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrArchiveMeetingNotFound)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrArchiveMeetingNotOwner when user does not own meeting", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewArchiveMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		differentUserID := uuid.New()
		meeting := createTestMeeting(differentUserID, "Someone else's meeting")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)

		cmd := ArchiveMeetingCommand{
			MeetingID: meetingID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrArchiveMeetingNotOwner)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository find fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewArchiveMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(nil, errors.New("database error"))

		cmd := ArchiveMeetingCommand{
			MeetingID: meetingID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewArchiveMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Meeting")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(errors.New("save error"))

		cmd := ArchiveMeetingCommand{
			MeetingID: meetingID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when outbox save fails", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewArchiveMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Meeting")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(errors.New("outbox error"))

		cmd := ArchiveMeetingCommand{
			MeetingID: meetingID,
			UserID:    userID,
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outbox error")

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})
}

func TestNewArchiveMeetingHandler(t *testing.T) {
	repo := new(mockMeetingRepo)
	outboxRepo := new(mockOutboxRepo)
	uow := new(mockUnitOfWork)

	handler := NewArchiveMeetingHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
