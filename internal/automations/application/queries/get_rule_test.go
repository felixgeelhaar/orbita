package queries

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

// mockExecutionRepo is a mock implementation of domain.ExecutionRepository.
type mockExecutionRepo struct {
	mock.Mock
}

func (m *mockExecutionRepo) Create(ctx context.Context, execution *domain.RuleExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *mockExecutionRepo) Update(ctx context.Context, execution *domain.RuleExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *mockExecutionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RuleExecution, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RuleExecution), args.Error(1)
}

func (m *mockExecutionRepo) GetByRuleID(ctx context.Context, ruleID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	args := m.Called(ctx, ruleID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.RuleExecution), args.Error(1)
}

func (m *mockExecutionRepo) List(ctx context.Context, filter domain.ExecutionFilter) ([]*domain.RuleExecution, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*domain.RuleExecution), args.Get(1).(int64), args.Error(2)
}

func (m *mockExecutionRepo) CountByRuleIDSince(ctx context.Context, ruleID uuid.UUID, since time.Time) (int64, error) {
	args := m.Called(ctx, ruleID, since)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockExecutionRepo) GetLatestByRuleID(ctx context.Context, ruleID uuid.UUID) (*domain.RuleExecution, error) {
	args := m.Called(ctx, ruleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RuleExecution), args.Error(1)
}

func (m *mockExecutionRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func createTestRule(userID uuid.UUID) *domain.AutomationRule {
	actions := []types.RuleAction{{Type: "notify"}}
	rule, _ := domain.NewAutomationRule(userID, "Test Rule", domain.TriggerTypeEvent, nil, actions)
	return rule
}

func TestGetRuleQuery_Validate(t *testing.T) {
	t.Run("valid query", func(t *testing.T) {
		q := GetRuleQuery{
			RuleID: uuid.New(),
			UserID: uuid.New(),
		}

		err := q.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing rule_id", func(t *testing.T) {
		q := GetRuleQuery{
			UserID: uuid.New(),
		}

		err := q.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule_id is required")
	})

	t.Run("missing user_id", func(t *testing.T) {
		q := GetRuleQuery{
			RuleID: uuid.New(),
		}

		err := q.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})
}

func TestGetRuleHandler_Handle(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully returns rule", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewGetRuleHandler(repo)

		rule := createTestRule(userID)
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)

		q := GetRuleQuery{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, ruleID, result.ID)
		assert.Equal(t, userID, result.UserID)

		repo.AssertExpectations(t)
	})

	t.Run("fails with invalid query", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewGetRuleHandler(repo)

		q := GetRuleQuery{
			RuleID: uuid.Nil, // Invalid
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewGetRuleHandler(repo)

		repo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		q := GetRuleQuery{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when user not authorized", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewGetRuleHandler(repo)

		rule := createTestRule(uuid.New()) // Different user
		rule.ID = ruleID

		repo.On("GetByID", mock.Anything, ruleID).Return(rule, nil)

		q := GetRuleQuery{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.ErrorIs(t, err, domain.ErrRuleNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewGetRuleHandler(repo)

		repo.On("GetByID", mock.Anything, ruleID).Return(nil, errors.New("database error"))

		q := GetRuleQuery{
			RuleID: ruleID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestNewGetRuleHandler(t *testing.T) {
	repo := new(mockRuleRepo)
	handler := NewGetRuleHandler(repo)

	require.NotNil(t, handler)
}
