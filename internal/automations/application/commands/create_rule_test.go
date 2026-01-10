package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockRuleRepo is a mock implementation of domain.RuleRepository.
type mockRuleRepo struct {
	mock.Mock
}

func (m *mockRuleRepo) Create(ctx context.Context, rule *domain.AutomationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *mockRuleRepo) Update(ctx context.Context, rule *domain.AutomationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *mockRuleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockRuleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AutomationRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AutomationRule), args.Error(1)
}

func (m *mockRuleRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AutomationRule, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AutomationRule), args.Error(1)
}

func (m *mockRuleRepo) List(ctx context.Context, filter domain.RuleFilter) ([]*domain.AutomationRule, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*domain.AutomationRule), args.Get(1).(int64), args.Error(2)
}

func (m *mockRuleRepo) GetEnabledByTriggerType(ctx context.Context, userID uuid.UUID, triggerType domain.TriggerType) ([]*domain.AutomationRule, error) {
	args := m.Called(ctx, userID, triggerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AutomationRule), args.Error(1)
}

func (m *mockRuleRepo) GetEnabledByEventType(ctx context.Context, userID uuid.UUID, eventType string) ([]*domain.AutomationRule, error) {
	args := m.Called(ctx, userID, eventType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AutomationRule), args.Error(1)
}

func (m *mockRuleRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// mockPendingActionRepo is a mock implementation of domain.PendingActionRepository.
type mockPendingActionRepo struct {
	mock.Mock
}

func (m *mockPendingActionRepo) Create(ctx context.Context, action *domain.PendingAction) error {
	args := m.Called(ctx, action)
	return args.Error(0)
}

func (m *mockPendingActionRepo) Update(ctx context.Context, action *domain.PendingAction) error {
	args := m.Called(ctx, action)
	return args.Error(0)
}

func (m *mockPendingActionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.PendingAction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PendingAction), args.Error(1)
}

func (m *mockPendingActionRepo) GetDue(ctx context.Context, limit int) ([]*domain.PendingAction, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PendingAction), args.Error(1)
}

func (m *mockPendingActionRepo) GetByRuleID(ctx context.Context, ruleID uuid.UUID) ([]*domain.PendingAction, error) {
	args := m.Called(ctx, ruleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PendingAction), args.Error(1)
}

func (m *mockPendingActionRepo) GetByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*domain.PendingAction, error) {
	args := m.Called(ctx, executionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PendingAction), args.Error(1)
}

func (m *mockPendingActionRepo) List(ctx context.Context, filter domain.PendingActionFilter) ([]*domain.PendingAction, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*domain.PendingAction), args.Get(1).(int64), args.Error(2)
}

func (m *mockPendingActionRepo) CancelByRuleID(ctx context.Context, ruleID uuid.UUID) error {
	args := m.Called(ctx, ruleID)
	return args.Error(0)
}

func (m *mockPendingActionRepo) DeleteExecuted(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func TestCreateRuleCommand_Validate(t *testing.T) {
	validActions := []types.RuleAction{{Type: "notify"}}

	t.Run("valid command", func(t *testing.T) {
		cmd := CreateRuleCommand{
			UserID:      uuid.New(),
			Name:        "Test Rule",
			TriggerType: domain.TriggerTypeEvent,
			Actions:     validActions,
		}

		err := cmd.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing user_id", func(t *testing.T) {
		cmd := CreateRuleCommand{
			Name:        "Test Rule",
			TriggerType: domain.TriggerTypeEvent,
			Actions:     validActions,
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})

	t.Run("missing name", func(t *testing.T) {
		cmd := CreateRuleCommand{
			UserID:      uuid.New(),
			TriggerType: domain.TriggerTypeEvent,
			Actions:     validActions,
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("missing trigger_type", func(t *testing.T) {
		cmd := CreateRuleCommand{
			UserID:  uuid.New(),
			Name:    "Test Rule",
			Actions: validActions,
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trigger_type is required")
	})

	t.Run("missing actions", func(t *testing.T) {
		cmd := CreateRuleCommand{
			UserID:      uuid.New(),
			Name:        "Test Rule",
			TriggerType: domain.TriggerTypeEvent,
		}

		err := cmd.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one action is required")
	})
}

func TestCreateRuleHandler_Handle(t *testing.T) {
	userID := uuid.New()
	validActions := []types.RuleAction{{Type: "notify", Parameters: map[string]any{"message": "test"}}}

	t.Run("successfully creates rule", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewCreateRuleHandler(repo)

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		cmd := CreateRuleCommand{
			UserID:      userID,
			Name:        "Test Rule",
			TriggerType: domain.TriggerTypeEvent,
			Actions:     validActions,
		}

		rule, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, userID, rule.UserID)
		assert.Equal(t, "Test Rule", rule.Name)
		assert.Equal(t, domain.TriggerTypeEvent, rule.TriggerType)
		assert.True(t, rule.Enabled)

		repo.AssertExpectations(t)
	})

	t.Run("creates rule with all options", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewCreateRuleHandler(repo)

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		maxExec := 10
		cmd := CreateRuleCommand{
			UserID:            userID,
			Name:              "Full Rule",
			Description:       "A complete rule",
			TriggerType:       domain.TriggerTypeSchedule,
			TriggerConfig:     map[string]any{"schedule": "0 9 * * *"},
			Conditions:        []types.RuleCondition{{Field: "status", Operator: "eq", Value: "active"}},
			ConditionOperator: domain.ConditionOperatorOR,
			Actions:           validActions,
			CooldownSeconds:   300,
			MaxExecutionsPerHour: &maxExec,
			Priority:          5,
			Tags:              []string{"daily", "notification"},
		}

		rule, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, "A complete rule", rule.Description)
		assert.Equal(t, 5, rule.Priority)
		assert.Equal(t, 300, rule.CooldownSeconds)
		require.NotNil(t, rule.MaxExecutionsPerHour)
		assert.Equal(t, 10, *rule.MaxExecutionsPerHour)
		assert.Len(t, rule.Conditions, 1)
		assert.Equal(t, domain.ConditionOperatorOR, rule.ConditionOperator)
		assert.Contains(t, rule.Tags, "daily")
		assert.Contains(t, rule.Tags, "notification")

		repo.AssertExpectations(t)
	})

	t.Run("fails with invalid command", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewCreateRuleHandler(repo)

		cmd := CreateRuleCommand{
			UserID:  userID,
			Name:    "", // Invalid: empty name
			Actions: validActions,
		}

		rule, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, rule)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewCreateRuleHandler(repo)

		repoErr := errors.New("database error")
		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(repoErr)

		cmd := CreateRuleCommand{
			UserID:      userID,
			Name:        "Test Rule",
			TriggerType: domain.TriggerTypeEvent,
			Actions:     validActions,
		}

		rule, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, rule)

		repo.AssertExpectations(t)
	})
}

func TestNewCreateRuleHandler(t *testing.T) {
	repo := new(mockRuleRepo)
	handler := NewCreateRuleHandler(repo)

	require.NotNil(t, handler)
}
