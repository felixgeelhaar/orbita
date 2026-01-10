package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEndSessionHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully ends active session", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewEndSessionHandler(repo)

		activeSession := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Deep work")
		repo.On("GetActive", mock.Anything, userID).Return(activeSession, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(nil)

		cmd := EndSessionCommand{
			UserID: userID,
			Status: domain.SessionStatusCompleted,
		}

		session, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, domain.SessionStatusCompleted, session.Status)
		assert.NotNil(t, session.EndedAt)

		repo.AssertExpectations(t)
	})

	t.Run("ends session with notes", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewEndSessionHandler(repo)

		activeSession := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Deep work")
		repo.On("GetActive", mock.Anything, userID).Return(activeSession, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(nil)

		cmd := EndSessionCommand{
			UserID: userID,
			Status: domain.SessionStatusCompleted,
			Notes:  "Completed the feature implementation",
		}

		session, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Equal(t, "Completed the feature implementation", session.Notes)

		repo.AssertExpectations(t)
	})

	t.Run("ends session as interrupted", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewEndSessionHandler(repo)

		activeSession := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Deep work")
		repo.On("GetActive", mock.Anything, userID).Return(activeSession, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(nil)

		cmd := EndSessionCommand{
			UserID: userID,
			Status: domain.SessionStatusInterrupted,
		}

		session, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Equal(t, domain.SessionStatusInterrupted, session.Status)

		repo.AssertExpectations(t)
	})

	t.Run("fails when no active session (ErrNotFound)", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewEndSessionHandler(repo)

		repo.On("GetActive", mock.Anything, userID).Return(nil, ErrNotFound)

		cmd := EndSessionCommand{
			UserID: userID,
			Status: domain.SessionStatusCompleted,
		}

		session, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrNoActiveSession)
		assert.Nil(t, session)

		repo.AssertExpectations(t)
	})

	t.Run("fails when no active session (nil return)", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewEndSessionHandler(repo)

		repo.On("GetActive", mock.Anything, userID).Return(nil, nil)

		cmd := EndSessionCommand{
			UserID: userID,
			Status: domain.SessionStatusCompleted,
		}

		session, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrNoActiveSession)
		assert.Nil(t, session)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error on GetActive", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewEndSessionHandler(repo)

		repoErr := errors.New("database error")
		repo.On("GetActive", mock.Anything, userID).Return(nil, repoErr)

		cmd := EndSessionCommand{
			UserID: userID,
			Status: domain.SessionStatusCompleted,
		}

		session, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, session)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error on Update", func(t *testing.T) {
		repo := new(mockSessionRepo)
		handler := NewEndSessionHandler(repo)

		activeSession := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Deep work")
		repo.On("GetActive", mock.Anything, userID).Return(activeSession, nil)
		repoErr := errors.New("database error")
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(repoErr)

		cmd := EndSessionCommand{
			UserID: userID,
			Status: domain.SessionStatusCompleted,
		}

		session, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, session)

		repo.AssertExpectations(t)
	})
}

func TestNewEndSessionHandler(t *testing.T) {
	repo := new(mockSessionRepo)
	handler := NewEndSessionHandler(repo)

	require.NotNil(t, handler)
}
