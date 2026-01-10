package commands

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

func TestDeleteRuleCommand_Validate(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		cmd := DeleteRuleCommand{
			RuleID: uuid.New(),
			UserID: uuid.New(),
		}

		err := cmd.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing rule_id", func(t *testing.T) {
		cmd := DeleteRuleCommand{
			UserID: uuid.New(),
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule_id is required")
	})

	t.Run("missing user_id", func(t *testing.T) {
		cmd := DeleteRuleCommand{
			RuleID: uuid.New(),
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})
}

func TestDeleteRuleHandler_Handle(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully deletes rule", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewDeleteRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		pendingRepo.On("CancelByRuleID", mock.Anything, ruleID).Return(nil)
		ruleRepo.On("Delete", mock.Anything, ruleID).Return(nil)

		cmd := DeleteRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)

		ruleRepo.AssertExpectations(t)
		pendingRepo.AssertExpectations(t)
	})

	t.Run("fails with invalid command", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewDeleteRuleHandler(ruleRepo, pendingRepo)

		cmd := DeleteRuleCommand{
			RuleID: uuid.Nil, // Invalid
			UserID: userID,
		}

		err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewDeleteRuleHandler(ruleRepo, pendingRepo)

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		cmd := DeleteRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when user not authorized", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewDeleteRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(uuid.New()) // Different user
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)

		cmd := DeleteRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, domain.ErrRuleNotFound)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when cancel pending actions error", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewDeleteRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		pendingRepo.On("CancelByRuleID", mock.Anything, ruleID).Return(errors.New("database error"))

		cmd := DeleteRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)

		ruleRepo.AssertExpectations(t)
		pendingRepo.AssertExpectations(t)
	})

	t.Run("fails when delete error", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewDeleteRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		pendingRepo.On("CancelByRuleID", mock.Anything, ruleID).Return(nil)
		ruleRepo.On("Delete", mock.Anything, ruleID).Return(errors.New("database error"))

		cmd := DeleteRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)

		ruleRepo.AssertExpectations(t)
		pendingRepo.AssertExpectations(t)
	})
}

func TestNewDeleteRuleHandler(t *testing.T) {
	ruleRepo := new(mockRuleRepo)
	pendingRepo := new(mockPendingActionRepo)
	handler := NewDeleteRuleHandler(ruleRepo, pendingRepo)

	require.NotNil(t, handler)
}
