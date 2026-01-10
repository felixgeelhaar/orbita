package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createTestRule(userID uuid.UUID) *domain.AutomationRule {
	actions := []types.RuleAction{{Type: "notify"}}
	rule, _ := domain.NewAutomationRule(userID, "Test Rule", domain.TriggerTypeEvent, nil, actions)
	return rule
}

func TestEnableRuleCommand_Validate(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		cmd := EnableRuleCommand{
			RuleID: uuid.New(),
			UserID: uuid.New(),
		}

		err := cmd.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing rule_id", func(t *testing.T) {
		cmd := EnableRuleCommand{
			UserID: uuid.New(),
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule_id is required")
	})

	t.Run("missing user_id", func(t *testing.T) {
		cmd := EnableRuleCommand{
			RuleID: uuid.New(),
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})
}

func TestDisableRuleCommand_Validate(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		cmd := DisableRuleCommand{
			RuleID: uuid.New(),
			UserID: uuid.New(),
		}

		err := cmd.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing rule_id", func(t *testing.T) {
		cmd := DisableRuleCommand{
			UserID: uuid.New(),
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule_id is required")
	})

	t.Run("missing user_id", func(t *testing.T) {
		cmd := DisableRuleCommand{
			RuleID: uuid.New(),
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})
}

func TestToggleRuleHandler_Enable(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully enables rule", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(userID)
		rule.ID = ruleID
		rule.Disable() // Start disabled

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		ruleRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		cmd := EnableRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Enable(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Enabled)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails with invalid command", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		cmd := EnableRuleCommand{
			RuleID: uuid.Nil, // Invalid
			UserID: userID,
		}

		result, err := handler.Enable(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		cmd := EnableRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Enable(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when user not authorized", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(uuid.New()) // Different user
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)

		cmd := EnableRuleCommand{
			RuleID: ruleID,
			UserID: userID, // Different from rule owner
		}

		result, err := handler.Enable(context.Background(), cmd)

		assert.ErrorIs(t, err, domain.ErrRuleNotFound)
		assert.Nil(t, result)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when update error", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		ruleRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(errors.New("database error"))

		cmd := EnableRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Enable(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)

		ruleRepo.AssertExpectations(t)
	})
}

func TestToggleRuleHandler_Disable(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully disables rule and cancels pending actions", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		pendingRepo.On("CancelByRuleID", mock.Anything, ruleID).Return(nil)
		ruleRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		cmd := DisableRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Disable(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Enabled)

		ruleRepo.AssertExpectations(t)
		pendingRepo.AssertExpectations(t)
	})

	t.Run("fails with invalid command", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		cmd := DisableRuleCommand{
			RuleID: uuid.Nil, // Invalid
			UserID: userID,
		}

		result, err := handler.Disable(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		cmd := DisableRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Disable(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when user not authorized", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(uuid.New()) // Different user
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)

		cmd := DisableRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Disable(context.Background(), cmd)

		assert.ErrorIs(t, err, domain.ErrRuleNotFound)
		assert.Nil(t, result)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when cancel pending actions error", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		pendingRepo := new(mockPendingActionRepo)
		handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		pendingRepo.On("CancelByRuleID", mock.Anything, ruleID).Return(errors.New("database error"))

		cmd := DisableRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Disable(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)

		ruleRepo.AssertExpectations(t)
		pendingRepo.AssertExpectations(t)
	})
}

func TestNewToggleRuleHandler(t *testing.T) {
	ruleRepo := new(mockRuleRepo)
	pendingRepo := new(mockPendingActionRepo)
	handler := NewToggleRuleHandler(ruleRepo, pendingRepo)

	require.NotNil(t, handler)
}
