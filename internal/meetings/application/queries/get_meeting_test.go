package queries

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

func TestGetMeetingHandler_Handle(t *testing.T) {
	userID := uuid.New()
	meetingID := uuid.New()

	t.Run("successfully retrieves a meeting", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewGetMeetingHandler(repo)

		ctx := context.Background()

		now := time.Now()
		lastHeld := now.Add(-48 * time.Hour)
		meeting := domain.RehydrateMeeting(
			meetingID,
			userID,
			"Weekly Standup",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			9*time.Hour,
			&lastHeld,
			false,
			now.Add(-30*24*time.Hour),
			now,
		)

		repo.On("FindByID", ctx, meetingID).Return(meeting, nil)

		query := GetMeetingQuery{
			MeetingID: meetingID,
			UserID:    userID,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, meetingID, result.ID)
		assert.Equal(t, "Weekly Standup", result.Name)
		assert.Equal(t, "weekly", result.Cadence)
		assert.Equal(t, 7, result.CadenceDays)
		assert.Equal(t, 30, result.DurationMins)
		assert.Equal(t, "09:00", result.PreferredTime)
		assert.NotNil(t, result.LastHeldAt)
		assert.False(t, result.Archived)
		assert.NotNil(t, result.NextOccurrence)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrMeetingNotFound when meeting does not exist", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewGetMeetingHandler(repo)

		ctx := context.Background()

		repo.On("FindByID", ctx, meetingID).Return(nil, nil)

		query := GetMeetingQuery{
			MeetingID: meetingID,
			UserID:    userID,
		}

		result, err := handler.Handle(ctx, query)

		assert.ErrorIs(t, err, ErrMeetingNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrMeetingNotFound when user does not own meeting", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewGetMeetingHandler(repo)

		ctx := context.Background()

		differentUserID := uuid.New()
		now := time.Now()
		meeting := domain.RehydrateMeeting(
			meetingID,
			differentUserID, // Different user owns this meeting
			"Someone's Meeting",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			9*time.Hour,
			nil,
			false,
			now.Add(-30*24*time.Hour),
			now,
		)

		repo.On("FindByID", ctx, meetingID).Return(meeting, nil)

		query := GetMeetingQuery{
			MeetingID: meetingID,
			UserID:    userID, // This user is trying to access
		}

		result, err := handler.Handle(ctx, query)

		assert.ErrorIs(t, err, ErrMeetingNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository returns error", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewGetMeetingHandler(repo)

		ctx := context.Background()

		repo.On("FindByID", ctx, meetingID).Return(nil, errors.New("database error"))

		query := GetMeetingQuery{
			MeetingID: meetingID,
			UserID:    userID,
		}

		result, err := handler.Handle(ctx, query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
	})

	t.Run("retrieves archived meeting without next occurrence", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewGetMeetingHandler(repo)

		ctx := context.Background()

		now := time.Now()
		lastHeld := now.Add(-14 * 24 * time.Hour)
		meeting := domain.RehydrateMeeting(
			meetingID,
			userID,
			"Archived Meeting",
			domain.CadenceWeekly,
			7,
			60*time.Minute,
			10*time.Hour,
			&lastHeld,
			true, // Archived
			now.Add(-60*24*time.Hour),
			now,
		)

		repo.On("FindByID", ctx, meetingID).Return(meeting, nil)

		query := GetMeetingQuery{
			MeetingID: meetingID,
			UserID:    userID,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Archived)
		assert.Nil(t, result.NextOccurrence) // Archived meetings don't have next occurrence

		repo.AssertExpectations(t)
	})

	t.Run("retrieves meeting with different cadence types", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewGetMeetingHandler(repo)

		ctx := context.Background()

		now := time.Now()
		meeting := domain.RehydrateMeeting(
			meetingID,
			userID,
			"Biweekly Sync",
			domain.CadenceBiweekly,
			14,
			45*time.Minute,
			14*time.Hour+30*time.Minute,
			nil,
			false,
			now.Add(-30*24*time.Hour),
			now,
		)

		repo.On("FindByID", ctx, meetingID).Return(meeting, nil)

		query := GetMeetingQuery{
			MeetingID: meetingID,
			UserID:    userID,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "biweekly", result.Cadence)
		assert.Equal(t, 14, result.CadenceDays)
		assert.Equal(t, 45, result.DurationMins)
		assert.Equal(t, "14:30", result.PreferredTime)

		repo.AssertExpectations(t)
	})

	t.Run("retrieves meeting without last held date", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewGetMeetingHandler(repo)

		ctx := context.Background()

		now := time.Now()
		meeting := domain.RehydrateMeeting(
			meetingID,
			userID,
			"New Meeting",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			11*time.Hour,
			nil, // Never held
			false,
			now.Add(-24*time.Hour),
			now,
		)

		repo.On("FindByID", ctx, meetingID).Return(meeting, nil)

		query := GetMeetingQuery{
			MeetingID: meetingID,
			UserID:    userID,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.LastHeldAt)
		assert.NotNil(t, result.NextOccurrence)

		repo.AssertExpectations(t)
	})
}

func TestNewGetMeetingHandler(t *testing.T) {
	repo := new(mockMeetingRepo)

	handler := NewGetMeetingHandler(repo)

	require.NotNil(t, handler)
}
