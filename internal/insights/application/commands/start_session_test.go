package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockSessionRepo is a mock implementation of domain.SessionRepository.
type mockSessionRepo struct {
	mock.Mock
}

func (m *mockSessionRepo) Create(ctx context.Context, session *domain.TimeSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockSessionRepo) Update(ctx context.Context, session *domain.TimeSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.TimeSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TimeSession), args.Error(1)
}

func (m *mockSessionRepo) GetActive(ctx context.Context, userID uuid.UUID) (*domain.TimeSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TimeSession), args.Error(1)
}

func (m *mockSessionRepo) GetByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.TimeSession, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TimeSession), args.Error(1)
}

func (m *mockSessionRepo) GetByType(ctx context.Context, userID uuid.UUID, sessionType domain.SessionType, limit int) ([]*domain.TimeSession, error) {
	args := m.Called(ctx, userID, sessionType, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TimeSession), args.Error(1)
}

func (m *mockSessionRepo) GetTotalFocusMinutes(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	args := m.Called(ctx, userID, start, end)
	return args.Int(0), args.Error(1)
}

func (m *mockSessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestStartSessionHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully starts new session", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewStartSessionHandler(repo)

		repo.On("GetActive", mock.Anything, userID).Return(nil, ErrNotFound)
		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(nil)

		cmd := StartSessionCommand{
			UserID:      userID,
			SessionType: domain.SessionTypeFocus,
			Title:       "Deep work",
			Category:    "development",
		}

		session, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, domain.SessionTypeFocus, session.SessionType)
		assert.Equal(t, "Deep work", session.Title)
		assert.Equal(t, "development", session.Category)
		assert.Equal(t, domain.SessionStatusActive, session.Status)

		repo.AssertExpectations(t)
	})

	t.Run("starts session with reference ID", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewStartSessionHandler(repo)
		refID := uuid.New()

		repo.On("GetActive", mock.Anything, userID).Return(nil, ErrNotFound)
		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(nil)

		cmd := StartSessionCommand{
			UserID:      userID,
			SessionType: domain.SessionTypeTask,
			Title:       "Work on task",
			ReferenceID: &refID,
		}

		session, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, session.ReferenceID)
		assert.Equal(t, refID, *session.ReferenceID)

		repo.AssertExpectations(t)
	})

	t.Run("fails when session already active", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewStartSessionHandler(repo)

		existingSession := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Existing")
		repo.On("GetActive", mock.Anything, userID).Return(existingSession, nil)

		cmd := StartSessionCommand{
			UserID:      userID,
			SessionType: domain.SessionTypeFocus,
			Title:       "New session",
		}

		session, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrSessionAlreadyActive)
		assert.Nil(t, session)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error on GetActive", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewStartSessionHandler(repo)

		repoErr := errors.New("database error")
		repo.On("GetActive", mock.Anything, userID).Return(nil, repoErr)

		cmd := StartSessionCommand{
			UserID:      userID,
			SessionType: domain.SessionTypeFocus,
			Title:       "Session",
		}

		session, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, session)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error on Create", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewStartSessionHandler(repo)

		repo.On("GetActive", mock.Anything, userID).Return(nil, ErrNotFound)
		repoErr := errors.New("database error")
		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(repoErr)

		cmd := StartSessionCommand{
			UserID:      userID,
			SessionType: domain.SessionTypeFocus,
			Title:       "Session",
		}

		session, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, session)

		repo.AssertExpectations(t)
	})
}

func TestNewStartSessionHandler(t *testing.T) {
	repo := new(mockSessionRepo)
	handler := NewStartSessionHandler(repo)

	require.NotNil(t, handler)
}
