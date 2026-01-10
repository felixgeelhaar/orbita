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

func createTestMeeting(userID uuid.UUID, name string) *domain.Meeting {
	now := time.Now()
	return domain.RehydrateMeeting(
		uuid.New(),
		userID,
		name,
		domain.CadenceWeekly,
		7,
		30*time.Minute,
		10*time.Hour,
		nil,
		false,
		now,
		now,
	)
}

func TestUpdateMeetingHandler_Handle(t *testing.T) {
	userID := uuid.New()
	meetingID := uuid.New()

	t.Run("successfully updates meeting name", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Old name")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(nil)

		cmd := UpdateMeetingCommand{
			UserID:    userID,
			MeetingID: meetingID,
			Name:      "New name",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, "New name", meeting.Name())

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("successfully updates cadence", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Weekly sync")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(nil)
		outboxRepo.On("SaveBatch", txCtx, mock.AnythingOfType("[]*outbox.Message")).Return(nil)

		cmd := UpdateMeetingCommand{
			UserID:    userID,
			MeetingID: meetingID,
			Cadence:   "biweekly",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, domain.CadenceBiweekly, meeting.Cadence())

		repo.AssertExpectations(t)
		outboxRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("successfully updates duration", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Meeting")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(nil)

		cmd := UpdateMeetingCommand{
			UserID:       userID,
			MeetingID:    meetingID,
			DurationMins: 60,
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, 60*time.Minute, meeting.Duration())

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("successfully updates preferred time", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Meeting")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Commit", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)
		repo.On("Save", txCtx, meeting).Return(nil)

		cmd := UpdateMeetingCommand{
			UserID:        userID,
			MeetingID:     meetingID,
			PreferredTime: "14:30",
		}

		err := handler.Handle(ctx, cmd)

		require.NoError(t, err)
		assert.Equal(t, 14*time.Hour+30*time.Minute, meeting.PreferredTime())

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrMeetingNotFound when meeting does not exist", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(nil, nil)

		cmd := UpdateMeetingCommand{
			UserID:    userID,
			MeetingID: meetingID,
			Name:      "New name",
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrMeetingNotFound)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns ErrMeetingNotOwner when user does not own meeting", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		differentUserID := uuid.New()
		meeting := createTestMeeting(differentUserID, "Someone else's meeting")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)

		cmd := UpdateMeetingCommand{
			UserID:    userID,
			MeetingID: meetingID,
			Name:      "New name",
		}

		err := handler.Handle(ctx, cmd)

		assert.ErrorIs(t, err, ErrMeetingNotOwner)

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails with invalid preferred time format", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		meeting := createTestMeeting(userID, "Meeting")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(meeting, nil)

		cmd := UpdateMeetingCommand{
			UserID:        userID,
			MeetingID:     meetingID,
			PreferredTime: "invalid",
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid preferred time format")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		outboxRepo := new(mockOutboxRepo)
		uow := new(mockUnitOfWork)
		handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

		ctx := context.Background()
		txCtx := context.WithValue(ctx, "tx", "transaction")

		uow.On("Begin", ctx).Return(txCtx, nil)
		uow.On("Rollback", txCtx).Return(nil)
		repo.On("FindByID", txCtx, meetingID).Return(nil, errors.New("database error"))

		cmd := UpdateMeetingCommand{
			UserID:    userID,
			MeetingID: meetingID,
			Name:      "New name",
		}

		err := handler.Handle(ctx, cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})
}

func TestNewUpdateMeetingHandler(t *testing.T) {
	repo := new(mockMeetingRepo)
	outboxRepo := new(mockOutboxRepo)
	uow := new(mockUnitOfWork)

	handler := NewUpdateMeetingHandler(repo, outboxRepo, uow)

	require.NotNil(t, handler)
}
