package queries

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

func createTestMeeting(userID uuid.UUID, name string, archived bool) *domain.Meeting {
	now := time.Now()
	lastHeld := now.Add(-48 * time.Hour)

	return domain.RehydrateMeeting(
		uuid.New(),
		userID,
		name,
		domain.CadenceWeekly,
		7,
		30*time.Minute,
		9*time.Hour, // 9:00 AM preferred
		&lastHeld,
		archived,
		now.Add(-30*24*time.Hour),
		now,
	)
}

func TestListMeetingsHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully lists active meetings", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewListMeetingsHandler(repo)

		ctx := context.Background()

		meeting1 := createTestMeeting(userID, "Weekly Standup", false)
		meeting2 := createTestMeeting(userID, "Sprint Planning", false)
		meetings := []*domain.Meeting{meeting1, meeting2}

		repo.On("FindActiveByUserID", ctx, userID).Return(meetings, nil)

		query := ListMeetingsQuery{
			UserID:          userID,
			IncludeArchived: false,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "Weekly Standup", result[0].Name)
		assert.Equal(t, "Sprint Planning", result[1].Name)
		assert.Equal(t, "weekly", result[0].Cadence)
		assert.Equal(t, 7, result[0].CadenceDays)
		assert.Equal(t, 30, result[0].DurationMins)
		assert.Equal(t, "09:00", result[0].PreferredTime)
		assert.False(t, result[0].Archived)
		assert.NotNil(t, result[0].NextOccurrence)

		repo.AssertExpectations(t)
	})

	t.Run("successfully lists all meetings including archived", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewListMeetingsHandler(repo)

		ctx := context.Background()

		activeMeeting := createTestMeeting(userID, "Weekly Standup", false)
		archivedMeeting := createTestMeeting(userID, "Old Meeting", true)
		meetings := []*domain.Meeting{activeMeeting, archivedMeeting}

		repo.On("FindByUserID", ctx, userID).Return(meetings, nil)

		query := ListMeetingsQuery{
			UserID:          userID,
			IncludeArchived: true,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 2)

		// Find the archived meeting in results
		var foundArchived bool
		for _, m := range result {
			if m.Archived {
				foundArchived = true
				assert.Nil(t, m.NextOccurrence) // Archived meetings don't have next occurrence
			}
		}
		assert.True(t, foundArchived)

		repo.AssertExpectations(t)
	})

	t.Run("returns empty list when no meetings exist", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewListMeetingsHandler(repo)

		ctx := context.Background()

		repo.On("FindActiveByUserID", ctx, userID).Return([]*domain.Meeting{}, nil)

		query := ListMeetingsQuery{
			UserID:          userID,
			IncludeArchived: false,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		assert.Empty(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository returns error for active meetings", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewListMeetingsHandler(repo)

		ctx := context.Background()

		repo.On("FindActiveByUserID", ctx, userID).Return(nil, errors.New("database error"))

		query := ListMeetingsQuery{
			UserID:          userID,
			IncludeArchived: false,
		}

		result, err := handler.Handle(ctx, query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository returns error for all meetings", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewListMeetingsHandler(repo)

		ctx := context.Background()

		repo.On("FindByUserID", ctx, userID).Return(nil, errors.New("database error"))

		query := ListMeetingsQuery{
			UserID:          userID,
			IncludeArchived: true,
		}

		result, err := handler.Handle(ctx, query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		repo.AssertExpectations(t)
	})

	t.Run("correctly formats preferred time", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewListMeetingsHandler(repo)

		ctx := context.Background()

		// Create meeting with 14:30 preferred time
		now := time.Now()
		meeting := domain.RehydrateMeeting(
			uuid.New(),
			userID,
			"Afternoon Meeting",
			domain.CadenceWeekly,
			7,
			45*time.Minute,
			14*time.Hour+30*time.Minute, // 14:30
			nil,
			false,
			now.Add(-30*24*time.Hour),
			now,
		)

		repo.On("FindActiveByUserID", ctx, userID).Return([]*domain.Meeting{meeting}, nil)

		query := ListMeetingsQuery{
			UserID:          userID,
			IncludeArchived: false,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "14:30", result[0].PreferredTime)
		assert.Equal(t, 45, result[0].DurationMins)

		repo.AssertExpectations(t)
	})

	t.Run("handles meeting without last held date", func(t *testing.T) {
		repo := new(mockMeetingRepo)
		handler := NewListMeetingsHandler(repo)

		ctx := context.Background()

		now := time.Now()
		meeting := domain.RehydrateMeeting(
			uuid.New(),
			userID,
			"New Meeting",
			domain.CadenceWeekly,
			7,
			30*time.Minute,
			10*time.Hour,
			nil, // Never held
			false,
			now.Add(-24*time.Hour),
			now,
		)

		repo.On("FindActiveByUserID", ctx, userID).Return([]*domain.Meeting{meeting}, nil)

		query := ListMeetingsQuery{
			UserID:          userID,
			IncludeArchived: false,
		}

		result, err := handler.Handle(ctx, query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Nil(t, result[0].LastHeldAt)
		assert.NotNil(t, result[0].NextOccurrence)

		repo.AssertExpectations(t)
	})
}

func TestNewListMeetingsHandler(t *testing.T) {
	repo := new(mockMeetingRepo)

	handler := NewListMeetingsHandler(repo)

	require.NotNil(t, handler)
}
