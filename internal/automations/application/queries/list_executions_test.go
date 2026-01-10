package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListExecutionsQuery_Validate(t *testing.T) {
	t.Run("valid query", func(t *testing.T) {
		q := ListExecutionsQuery{
			UserID: uuid.New(),
		}

		err := q.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing user_id", func(t *testing.T) {
		q := ListExecutionsQuery{}

		err := q.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})
}

func TestListExecutionsHandler_Handle(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully lists executions", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		exec1 := domain.NewRuleExecution(ruleID, userID, "task.completed", nil)
		exec2 := domain.NewRuleExecution(ruleID, userID, "task.created", nil)
		executions := []*domain.RuleExecution{exec1, exec2}

		execRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.ExecutionFilter) bool {
			return f.UserID == userID && f.Limit == 50
		})).Return(executions, int64(2), nil)

		q := ListExecutionsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Executions, 2)
		assert.Equal(t, int64(2), result.Total)

		execRepo.AssertExpectations(t)
	})

	t.Run("filters by rule ID with authorization", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		execRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.ExecutionFilter) bool {
			return f.RuleID != nil && *f.RuleID == ruleID
		})).Return([]*domain.RuleExecution{}, int64(0), nil)

		q := ListExecutionsQuery{
			UserID: userID,
			RuleID: &ruleID,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)

		ruleRepo.AssertExpectations(t)
		execRepo.AssertExpectations(t)
	})

	t.Run("fails when filtering by rule of another user", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		rule := createTestRule(uuid.New()) // Different user
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)

		q := ListExecutionsQuery{
			UserID: userID,
			RuleID: &ruleID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.ErrorIs(t, err, domain.ErrRuleNotFound)
		assert.Nil(t, result)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		q := ListExecutionsQuery{
			UserID: userID,
			RuleID: &ruleID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("filters by status", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		status := domain.ExecutionStatusSuccess
		execRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.ExecutionFilter) bool {
			return f.Status != nil && *f.Status == domain.ExecutionStatusSuccess
		})).Return([]*domain.RuleExecution{}, int64(0), nil)

		q := ListExecutionsQuery{
			UserID: userID,
			Status: &status,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)

		execRepo.AssertExpectations(t)
	})

	t.Run("filters by time range", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		startAfter := time.Now().Add(-24 * time.Hour)
		startBefore := time.Now()

		execRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.ExecutionFilter) bool {
			return f.StartAfter != nil && f.StartBefore != nil
		})).Return([]*domain.RuleExecution{}, int64(0), nil)

		q := ListExecutionsQuery{
			UserID:      userID,
			StartAfter:  &startAfter,
			StartBefore: &startBefore,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)

		execRepo.AssertExpectations(t)
	})

	t.Run("uses custom limit", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		execRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.ExecutionFilter) bool {
			return f.Limit == 10
		})).Return([]*domain.RuleExecution{}, int64(0), nil)

		q := ListExecutionsQuery{
			UserID: userID,
			Limit:  10,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)

		execRepo.AssertExpectations(t)
	})

	t.Run("supports pagination with offset", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		execRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.ExecutionFilter) bool {
			return f.Offset == 20 && f.Limit == 10
		})).Return([]*domain.RuleExecution{}, int64(25), nil)

		q := ListExecutionsQuery{
			UserID: userID,
			Limit:  10,
			Offset: 20,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(25), result.Total)

		execRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no executions", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		execRepo.On("List", mock.Anything, mock.Anything).Return([]*domain.RuleExecution{}, int64(0), nil)

		q := ListExecutionsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Executions)
		assert.Equal(t, int64(0), result.Total)

		execRepo.AssertExpectations(t)
	})

	t.Run("fails with invalid query", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		q := ListExecutionsQuery{
			UserID: uuid.Nil, // Invalid
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		execRepo := new(mockExecutionRepo)
		ruleRepo := new(mockRuleRepo)
		handler := NewListExecutionsHandler(execRepo, ruleRepo)

		execRepo.On("List", mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("database error"))

		q := ListExecutionsQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)

		execRepo.AssertExpectations(t)
	})
}

func TestNewListExecutionsHandler(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ruleRepo := new(mockRuleRepo)
	handler := NewListExecutionsHandler(execRepo, ruleRepo)

	require.NotNil(t, handler)
}
