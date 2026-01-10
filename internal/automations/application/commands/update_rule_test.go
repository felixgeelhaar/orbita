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

func TestUpdateRuleCommand_Validate(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		cmd := UpdateRuleCommand{
			RuleID: uuid.New(),
			UserID: uuid.New(),
		}

		err := cmd.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing rule_id", func(t *testing.T) {
		cmd := UpdateRuleCommand{
			UserID: uuid.New(),
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule_id is required")
	})

	t.Run("missing user_id", func(t *testing.T) {
		cmd := UpdateRuleCommand{
			RuleID: uuid.New(),
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})
}

func TestUpdateRuleHandler_Handle(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully updates rule name", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		newName := "Updated Name"
		cmd := UpdateRuleCommand{
			RuleID: ruleID,
			UserID: userID,
			Name:   &newName,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Updated Name", result.Name)

		repo.AssertExpectations(t)
	})

	t.Run("successfully updates rule description", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		newDesc := "New description"
		cmd := UpdateRuleCommand{
			RuleID:      ruleID,
			UserID:      userID,
			Description: &newDesc,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Equal(t, "New description", result.Description)

		repo.AssertExpectations(t)
	})

	t.Run("successfully enables rule", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID
		rule.Disable()

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		enabled := true
		cmd := UpdateRuleCommand{
			RuleID:  ruleID,
			UserID:  userID,
			Enabled: &enabled,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.True(t, result.Enabled)

		repo.AssertExpectations(t)
	})

	t.Run("successfully disables rule", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		enabled := false
		cmd := UpdateRuleCommand{
			RuleID:  ruleID,
			UserID:  userID,
			Enabled: &enabled,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.False(t, result.Enabled)

		repo.AssertExpectations(t)
	})

	t.Run("successfully updates priority", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		priority := 10
		cmd := UpdateRuleCommand{
			RuleID:   ruleID,
			UserID:   userID,
			Priority: &priority,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Equal(t, 10, result.Priority)

		repo.AssertExpectations(t)
	})

	t.Run("successfully updates conditions", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		operator := domain.ConditionOperatorOR
		cmd := UpdateRuleCommand{
			RuleID:            ruleID,
			UserID:            userID,
			Conditions:        []types.RuleCondition{{Field: "status", Operator: "eq", Value: "done"}},
			ConditionOperator: &operator,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Len(t, result.Conditions, 1)
		assert.Equal(t, domain.ConditionOperatorOR, result.ConditionOperator)

		repo.AssertExpectations(t)
	})

	t.Run("successfully updates actions", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		newActions := []types.RuleAction{
			{Type: "notify", Parameters: map[string]any{"message": "new"}},
			{Type: "email", Parameters: map[string]any{"to": "test@example.com"}},
		}
		cmd := UpdateRuleCommand{
			RuleID:  ruleID,
			UserID:  userID,
			Actions: newActions,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Len(t, result.Actions, 2)

		repo.AssertExpectations(t)
	})

	t.Run("successfully updates cooldown", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		cooldown := 600
		cmd := UpdateRuleCommand{
			RuleID:          ruleID,
			UserID:          userID,
			CooldownSeconds: &cooldown,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Equal(t, 600, result.CooldownSeconds)

		repo.AssertExpectations(t)
	})

	t.Run("successfully updates tags", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID
		rule.Tags = []string{"old-tag"}

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		cmd := UpdateRuleCommand{
			RuleID: ruleID,
			UserID: userID,
			Tags:   []string{"new-tag", "another-tag"},
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		assert.Contains(t, result.Tags, "new-tag")
		assert.Contains(t, result.Tags, "another-tag")
		assert.NotContains(t, result.Tags, "old-tag")

		repo.AssertExpectations(t)
	})

	t.Run("fails with invalid command", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		cmd := UpdateRuleCommand{
			RuleID: uuid.Nil, // Invalid
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		repo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		cmd := UpdateRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when user not authorized", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(uuid.New()) // Different user
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)

		cmd := UpdateRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, domain.ErrRuleNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when update error", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewUpdateRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(errors.New("database error"))

		cmd := UpdateRuleCommand{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestNewUpdateRuleHandler(t *testing.T) {
	repo := new(mockRuleRepo)
	handler := NewUpdateRuleHandler(repo)

	require.NotNil(t, handler)
}
