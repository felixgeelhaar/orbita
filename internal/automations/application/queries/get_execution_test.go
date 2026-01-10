package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetExecutionQuery_Validate(t *testing.T) {
	t.Run("valid query", func(t *testing.T) {
		q := GetExecutionQuery{
			ExecutionID: uuid.New(),
			UserID:      uuid.New(),
		}

		err := q.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing execution_id", func(t *testing.T) {
		q := GetExecutionQuery{
			UserID: uuid.New(),
		}

		err := q.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution_id is required")
	})

	t.Run("missing user_id", func(t *testing.T) {
		q := GetExecutionQuery{
			ExecutionID: uuid.New(),
		}

		err := q.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})
}

func TestGetExecutionHandler_Handle(t *testing.T) {
	userID := uuid.New()
	executionID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully returns execution", func(t *testing.T) {
		repo := new(mockExecutionRepo)
		handler := NewGetExecutionHandler(repo)

		execution := domain.NewRuleExecution(ruleID, userID, "task.completed", nil)
		execution.ID = executionID

		repo.On("GetByID", mock.Anything, executionID).Return(execution, nil)

		q := GetExecutionQuery{
			ExecutionID: executionID,
			UserID:      userID,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, executionID, result.ID)
		assert.Equal(t, userID, result.UserID)

		repo.AssertExpectations(t)
	})

	t.Run("fails with invalid query", func(t *testing.T) {
		repo := new(mockExecutionRepo)
		handler := NewGetExecutionHandler(repo)

		q := GetExecutionQuery{
			ExecutionID: uuid.Nil, // Invalid
			UserID:      userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("fails when execution not found", func(t *testing.T) {
		repo := new(mockExecutionRepo)
		handler := NewGetExecutionHandler(repo)

		repo.On("GetByID", mock.Anything, executionID).Return(nil, domain.ErrExecutionNotFound)

		q := GetExecutionQuery{
			ExecutionID: executionID,
			UserID:      userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when user not authorized", func(t *testing.T) {
		repo := new(mockExecutionRepo)
		handler := NewGetExecutionHandler(repo)

		execution := domain.NewRuleExecution(ruleID, uuid.New(), "task.completed", nil) // Different user
		execution.ID = executionID

		repo.On("GetByID", mock.Anything, executionID).Return(execution, nil)

		q := GetExecutionQuery{
			ExecutionID: executionID,
			UserID:      userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.ErrorIs(t, err, domain.ErrExecutionNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockExecutionRepo)
		handler := NewGetExecutionHandler(repo)

		repo.On("GetByID", mock.Anything, executionID).Return(nil, errors.New("database error"))

		q := GetExecutionQuery{
			ExecutionID: executionID,
			UserID:      userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestNewGetExecutionHandler(t *testing.T) {
	repo := new(mockExecutionRepo)
	handler := NewGetExecutionHandler(repo)

	require.NotNil(t, handler)
}
