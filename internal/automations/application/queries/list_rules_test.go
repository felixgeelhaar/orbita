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

func TestListRulesQuery_Validate(t *testing.T) {
	t.Run("valid query", func(t *testing.T) {
		q := ListRulesQuery{
			UserID: uuid.New(),
		}

		err := q.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing user_id", func(t *testing.T) {
		q := ListRulesQuery{}

		err := q.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})
}

func TestListRulesHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully lists rules", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		rule1 := createTestRule(userID)
		rule2 := createTestRule(userID)
		rules := []*domain.AutomationRule{rule1, rule2}

		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.RuleFilter) bool {
			return f.UserID == userID && f.Limit == 50
		})).Return(rules, int64(2), nil)

		q := ListRulesQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Rules, 2)
		assert.Equal(t, int64(2), result.Total)

		repo.AssertExpectations(t)
	})

	t.Run("uses custom limit", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.RuleFilter) bool {
			return f.Limit == 10
		})).Return([]*domain.AutomationRule{}, int64(0), nil)

		q := ListRulesQuery{
			UserID: userID,
			Limit:  10,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("filters by enabled status", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		enabled := true
		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.RuleFilter) bool {
			return f.Enabled != nil && *f.Enabled == true
		})).Return([]*domain.AutomationRule{}, int64(0), nil)

		q := ListRulesQuery{
			UserID:  userID,
			Enabled: &enabled,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("filters by trigger type", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		triggerType := domain.TriggerTypeSchedule
		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.RuleFilter) bool {
			return f.TriggerType != nil && *f.TriggerType == domain.TriggerTypeSchedule
		})).Return([]*domain.AutomationRule{}, int64(0), nil)

		q := ListRulesQuery{
			UserID:      userID,
			TriggerType: &triggerType,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("filters by tags", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.RuleFilter) bool {
			return len(f.Tags) == 2 && f.Tags[0] == "daily"
		})).Return([]*domain.AutomationRule{}, int64(0), nil)

		q := ListRulesQuery{
			UserID: userID,
			Tags:   []string{"daily", "notification"},
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("supports pagination with offset", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.RuleFilter) bool {
			return f.Offset == 20 && f.Limit == 10
		})).Return([]*domain.AutomationRule{}, int64(25), nil)

		q := ListRulesQuery{
			UserID: userID,
			Limit:  10,
			Offset: 20,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(25), result.Total)

		repo.AssertExpectations(t)
	})

	t.Run("returns empty list when no rules", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		repo.On("List", mock.Anything, mock.Anything).Return([]*domain.AutomationRule{}, int64(0), nil)

		q := ListRulesQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Rules)
		assert.Equal(t, int64(0), result.Total)

		repo.AssertExpectations(t)
	})

	t.Run("fails with invalid query", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		q := ListRulesQuery{
			UserID: uuid.Nil, // Invalid
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockRuleRepo)
		handler := NewListRulesHandler(repo)

		repo.On("List", mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("database error"))

		q := ListRulesQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), q)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestNewListRulesHandler(t *testing.T) {
	repo := new(mockRuleRepo)
	handler := NewListRulesHandler(repo)

	require.NotNil(t, handler)
}
