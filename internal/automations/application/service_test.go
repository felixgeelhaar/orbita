package application

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/application/commands"
	"github.com/felixgeelhaar/orbita/internal/automations/application/queries"
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

func (m *mockExecutionRepo) Create(ctx context.Context, exec *domain.RuleExecution) error {
	args := m.Called(ctx, exec)
	return args.Error(0)
}

func (m *mockExecutionRepo) Update(ctx context.Context, exec *domain.RuleExecution) error {
	args := m.Called(ctx, exec)
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

// Helper function to create a test rule
func createTestRule(userID uuid.UUID) *domain.AutomationRule {
	rule, _ := domain.NewAutomationRule(
		userID,
		"Test Rule",
		domain.TriggerTypeEvent,
		map[string]any{"event_type": "task.created"},
		[]types.RuleAction{{Type: "notify", Parameters: map[string]any{"message": "test"}}},
	)
	return rule
}

func TestNewService(t *testing.T) {
	ruleRepo := new(mockRuleRepo)
	execRepo := new(mockExecutionRepo)
	pendingRepo := new(mockPendingActionRepo)

	svc := NewService(ruleRepo, execRepo, pendingRepo)

	require.NotNil(t, svc)
}

func TestService_CreateRule(t *testing.T) {
	userID := uuid.New()
	validActions := []types.RuleAction{{Type: "notify", Parameters: map[string]any{"message": "test"}}}

	t.Run("successfully creates rule", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		ruleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		cmd := commands.CreateRuleCommand{
			UserID:      userID,
			Name:        "Test Rule",
			TriggerType: domain.TriggerTypeEvent,
			Actions:     validActions,
		}

		rule, err := svc.CreateRule(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, userID, rule.UserID)
		assert.Equal(t, "Test Rule", rule.Name)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails with invalid command", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		cmd := commands.CreateRuleCommand{
			UserID: userID,
			Name:   "", // Invalid
		}

		rule, err := svc.CreateRule(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, rule)
	})
}

func TestService_UpdateRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully updates rule", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		existingRule := createTestRule(userID)
		existingRule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(existingRule, nil)
		ruleRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		newName := "Updated Rule"
		cmd := commands.UpdateRuleCommand{
			UserID: userID,
			RuleID: ruleID,
			Name:   &newName,
		}

		rule, err := svc.UpdateRule(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, "Updated Rule", rule.Name)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		newName := "Updated Rule"
		cmd := commands.UpdateRuleCommand{
			UserID: userID,
			RuleID: ruleID,
			Name:   &newName,
		}

		rule, err := svc.UpdateRule(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, rule)

		ruleRepo.AssertExpectations(t)
	})
}

func TestService_DeleteRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully deletes rule", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		existingRule := createTestRule(userID)
		existingRule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(existingRule, nil)
		pendingRepo.On("CancelByRuleID", mock.Anything, ruleID).Return(nil)
		ruleRepo.On("Delete", mock.Anything, ruleID).Return(nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		cmd := commands.DeleteRuleCommand{
			UserID: userID,
			RuleID: ruleID,
		}

		err := svc.DeleteRule(context.Background(), cmd)

		require.NoError(t, err)

		ruleRepo.AssertExpectations(t)
		pendingRepo.AssertExpectations(t)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		cmd := commands.DeleteRuleCommand{
			UserID: userID,
			RuleID: ruleID,
		}

		err := svc.DeleteRule(context.Background(), cmd)

		assert.Error(t, err)

		ruleRepo.AssertExpectations(t)
	})
}

func TestService_EnableRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully enables rule", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		existingRule := createTestRule(userID)
		existingRule.ID = ruleID
		existingRule.Enabled = false

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(existingRule, nil)
		ruleRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		cmd := commands.EnableRuleCommand{
			UserID: userID,
			RuleID: ruleID,
		}

		rule, err := svc.EnableRule(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.True(t, rule.Enabled)

		ruleRepo.AssertExpectations(t)
	})
}

func TestService_DisableRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully disables rule and cancels pending actions", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		existingRule := createTestRule(userID)
		existingRule.ID = ruleID
		existingRule.Enabled = true

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(existingRule, nil)
		pendingRepo.On("CancelByRuleID", mock.Anything, ruleID).Return(nil)
		ruleRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.AutomationRule")).Return(nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		cmd := commands.DisableRuleCommand{
			UserID: userID,
			RuleID: ruleID,
		}

		rule, err := svc.DisableRule(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.False(t, rule.Enabled)

		ruleRepo.AssertExpectations(t)
		pendingRepo.AssertExpectations(t)
	})
}

func TestService_GetRule(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully gets rule", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		existingRule := createTestRule(userID)
		existingRule.ID = ruleID

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(existingRule, nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		q := queries.GetRuleQuery{
			UserID: userID,
			RuleID: ruleID,
		}

		rule, err := svc.GetRule(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, rule)
		assert.Equal(t, ruleID, rule.ID)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("fails when rule not found", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		ruleRepo.On("GetByID", mock.Anything, ruleID).Return(nil, domain.ErrRuleNotFound)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		q := queries.GetRuleQuery{
			UserID: userID,
			RuleID: ruleID,
		}

		rule, err := svc.GetRule(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, rule)

		ruleRepo.AssertExpectations(t)
	})
}

func TestService_ListRules(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully lists rules", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		rule1 := createTestRule(userID)
		rule2 := createTestRule(userID)
		rules := []*domain.AutomationRule{rule1, rule2}

		ruleRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.RuleFilter) bool {
			return f.UserID == userID
		})).Return(rules, int64(2), nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		q := queries.ListRulesQuery{
			UserID: userID,
		}

		result, err := svc.ListRules(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Rules, 2)
		assert.Equal(t, int64(2), result.Total)

		ruleRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no rules", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		ruleRepo.On("List", mock.Anything, mock.Anything).Return([]*domain.AutomationRule{}, int64(0), nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		q := queries.ListRulesQuery{
			UserID: userID,
		}

		result, err := svc.ListRules(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Rules)

		ruleRepo.AssertExpectations(t)
	})
}

func TestService_GetExecution(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()
	execID := uuid.New()

	t.Run("successfully gets execution", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		exec := domain.NewRuleExecution(ruleID, userID, "task.created", nil)
		exec.ID = execID

		execRepo.On("GetByID", mock.Anything, execID).Return(exec, nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		q := queries.GetExecutionQuery{
			UserID:      userID,
			ExecutionID: execID,
		}

		result, err := svc.GetExecution(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, execID, result.ID)

		execRepo.AssertExpectations(t)
	})
}

func TestService_ListExecutions(t *testing.T) {
	userID := uuid.New()
	ruleID := uuid.New()

	t.Run("successfully lists executions", func(t *testing.T) {
		ruleRepo := new(mockRuleRepo)
		execRepo := new(mockExecutionRepo)
		pendingRepo := new(mockPendingActionRepo)

		exec1 := domain.NewRuleExecution(ruleID, userID, "task.created", nil)
		exec2 := domain.NewRuleExecution(ruleID, userID, "task.completed", nil)
		executions := []*domain.RuleExecution{exec1, exec2}

		execRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.ExecutionFilter) bool {
			return f.UserID == userID
		})).Return(executions, int64(2), nil)

		svc := NewService(ruleRepo, execRepo, pendingRepo)

		q := queries.ListExecutionsQuery{
			UserID: userID,
		}

		result, err := svc.ListExecutions(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Executions, 2)
		assert.Equal(t, int64(2), result.Total)

		execRepo.AssertExpectations(t)
	})
}
