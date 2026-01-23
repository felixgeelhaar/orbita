package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPendingActionTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create users table
	_, err = db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT, name TEXT)`)
	require.NoError(t, err)

	// Create automation_rules table
	_, err = db.Exec(`CREATE TABLE automation_rules (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		trigger_type TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`)
	require.NoError(t, err)

	// Create automation_rule_executions table
	_, err = db.Exec(`CREATE TABLE automation_rule_executions (
		id TEXT PRIMARY KEY,
		rule_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at TEXT NOT NULL
	)`)
	require.NoError(t, err)

	// Create automation_pending_actions table
	_, err = db.Exec(`CREATE TABLE automation_pending_actions (
		id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL,
		rule_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		action_type TEXT NOT NULL,
		action_params TEXT NOT NULL DEFAULT '{}',
		scheduled_for TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		executed_at TEXT,
		result TEXT,
		error_message TEXT,
		retry_count INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		created_at TEXT NOT NULL
	)`)
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })
	return db
}

func createPendingActionTestFixtures(t *testing.T, db *sql.DB, userID, ruleID, executionID uuid.UUID) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec("INSERT INTO users (id, email, name) VALUES (?, ?, ?)",
		userID.String(), "pa@example.com", "PA User")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO automation_rules (id, user_id, name, trigger_type, created_at, updated_at)
		VALUES (?, ?, 'Test Rule', 'event', ?, ?)`, ruleID.String(), userID.String(), now, now)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO automation_rule_executions (id, rule_id, user_id, status, started_at)
		VALUES (?, ?, ?, 'success', ?)`, executionID.String(), ruleID.String(), userID.String(), now)
	require.NoError(t, err)
}

func createTestPendingAction(ruleID, userID, executionID uuid.UUID) *domain.PendingAction {
	return &domain.PendingAction{
		ID:           uuid.New(),
		ExecutionID:  executionID,
		RuleID:       ruleID,
		UserID:       userID,
		ActionType:   "send_email",
		ActionParams: map[string]any{"to": "test@example.com"},
		ScheduledFor: time.Now().UTC().Add(time.Hour).Truncate(time.Second),
		Status:       domain.PendingActionStatusPending,
		RetryCount:   0,
		MaxRetries:   3,
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
	}
}

func TestNewSQLitePendingActionRepository(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	assert.NotNil(t, repo)
}

func TestSQLitePendingActionRepository_Create(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	userID, ruleID, executionID := uuid.New(), uuid.New(), uuid.New()
	createPendingActionTestFixtures(t, db, userID, ruleID, executionID)

	action := createTestPendingAction(ruleID, userID, executionID)
	err := repo.Create(ctx, action)
	require.NoError(t, err)

	loaded, err := repo.GetByID(ctx, action.ID)
	require.NoError(t, err)
	assert.Equal(t, action.ActionType, loaded.ActionType)
	assert.Equal(t, action.Status, loaded.Status)
}

func TestSQLitePendingActionRepository_GetByID_NotFound(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	action, err := repo.GetByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, action) // Returns nil, nil for not found
}

func TestSQLitePendingActionRepository_Update(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	userID, ruleID, executionID := uuid.New(), uuid.New(), uuid.New()
	createPendingActionTestFixtures(t, db, userID, ruleID, executionID)

	action := createTestPendingAction(ruleID, userID, executionID)
	err := repo.Create(ctx, action)
	require.NoError(t, err)

	// Update action
	action.Status = domain.PendingActionStatusExecuted
	executedAt := time.Now().UTC().Truncate(time.Second)
	action.ExecutedAt = &executedAt
	action.Result = map[string]any{"sent": true}

	err = repo.Update(ctx, action)
	require.NoError(t, err)

	loaded, err := repo.GetByID(ctx, action.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.PendingActionStatusExecuted, loaded.Status)
	assert.NotNil(t, loaded.ExecutedAt)
}

func TestSQLitePendingActionRepository_GetDue(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	userID, ruleID, executionID := uuid.New(), uuid.New(), uuid.New()
	createPendingActionTestFixtures(t, db, userID, ruleID, executionID)

	// Create pending action due in the past
	pastAction := createTestPendingAction(ruleID, userID, executionID)
	pastAction.ScheduledFor = time.Now().Add(-time.Hour)
	err := repo.Create(ctx, pastAction)
	require.NoError(t, err)

	// Create pending action due in the future
	futureAction := createTestPendingAction(ruleID, userID, executionID)
	futureAction.ScheduledFor = time.Now().Add(time.Hour)
	err = repo.Create(ctx, futureAction)
	require.NoError(t, err)

	// Get due actions
	dueActions, err := repo.GetDue(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, dueActions, 1)
	assert.Equal(t, pastAction.ID, dueActions[0].ID)
}

func TestSQLitePendingActionRepository_GetByRuleID(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	userID, ruleID, executionID := uuid.New(), uuid.New(), uuid.New()
	createPendingActionTestFixtures(t, db, userID, ruleID, executionID)

	// Create multiple actions for the same rule
	for i := 0; i < 3; i++ {
		action := createTestPendingAction(ruleID, userID, executionID)
		err := repo.Create(ctx, action)
		require.NoError(t, err)
	}

	actions, err := repo.GetByRuleID(ctx, ruleID)
	require.NoError(t, err)
	assert.Len(t, actions, 3)
}

func TestSQLitePendingActionRepository_GetByExecutionID(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	userID, ruleID, executionID := uuid.New(), uuid.New(), uuid.New()
	createPendingActionTestFixtures(t, db, userID, ruleID, executionID)

	// Create action for this execution
	action := createTestPendingAction(ruleID, userID, executionID)
	err := repo.Create(ctx, action)
	require.NoError(t, err)

	actions, err := repo.GetByExecutionID(ctx, executionID)
	require.NoError(t, err)
	assert.Len(t, actions, 1)
	assert.Equal(t, action.ID, actions[0].ID)
}

func TestSQLitePendingActionRepository_CancelByRuleID(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	userID, ruleID, executionID := uuid.New(), uuid.New(), uuid.New()
	createPendingActionTestFixtures(t, db, userID, ruleID, executionID)

	// Create pending actions
	for i := 0; i < 2; i++ {
		action := createTestPendingAction(ruleID, userID, executionID)
		err := repo.Create(ctx, action)
		require.NoError(t, err)
	}

	// Cancel all actions for rule
	err := repo.CancelByRuleID(ctx, ruleID)
	require.NoError(t, err)

	// Verify all cancelled
	actions, err := repo.GetByRuleID(ctx, ruleID)
	require.NoError(t, err)
	for _, a := range actions {
		assert.Equal(t, domain.PendingActionStatusCancelled, a.Status)
	}
}

func TestSQLitePendingActionRepository_List(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	userID, ruleID, executionID := uuid.New(), uuid.New(), uuid.New()
	createPendingActionTestFixtures(t, db, userID, ruleID, executionID)

	// Create actions with different statuses
	pendingAction := createTestPendingAction(ruleID, userID, executionID)
	pendingAction.Status = domain.PendingActionStatusPending
	err := repo.Create(ctx, pendingAction)
	require.NoError(t, err)

	executedAction := createTestPendingAction(ruleID, userID, executionID)
	executedAction.Status = domain.PendingActionStatusExecuted
	err = repo.Create(ctx, executedAction)
	require.NoError(t, err)

	t.Run("filter by status", func(t *testing.T) {
		status := domain.PendingActionStatusPending
		actions, total, err := repo.List(ctx, domain.PendingActionFilter{
			UserID: userID,
			Status: &status,
			Limit:  10,
		})
		require.NoError(t, err)
		assert.Len(t, actions, 1)
		assert.Equal(t, int64(1), total)
	})

	t.Run("all actions", func(t *testing.T) {
		actions, total, err := repo.List(ctx, domain.PendingActionFilter{
			UserID: userID,
			Limit:  10,
		})
		require.NoError(t, err)
		assert.Len(t, actions, 2)
		assert.Equal(t, int64(2), total)
	})
}

func TestSQLitePendingActionRepository_DeleteExecuted(t *testing.T) {
	db := setupPendingActionTestDB(t)
	repo := NewSQLitePendingActionRepository(db)
	ctx := context.Background()

	userID, ruleID, executionID := uuid.New(), uuid.New(), uuid.New()
	createPendingActionTestFixtures(t, db, userID, ruleID, executionID)

	// Create old executed action
	oldAction := createTestPendingAction(ruleID, userID, executionID)
	oldAction.Status = domain.PendingActionStatusExecuted
	oldExecutedAt := time.Now().Add(-48 * time.Hour)
	oldAction.ExecutedAt = &oldExecutedAt
	err := repo.Create(ctx, oldAction)
	require.NoError(t, err)

	// Create recent executed action
	recentAction := createTestPendingAction(ruleID, userID, executionID)
	recentAction.Status = domain.PendingActionStatusExecuted
	recentExecutedAt := time.Now().Add(-time.Hour)
	recentAction.ExecutedAt = &recentExecutedAt
	err = repo.Create(ctx, recentAction)
	require.NoError(t, err)

	// Delete actions executed before 24 hours ago
	deleted, err := repo.DeleteExecuted(ctx, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify only recent remains
	actions, _, err := repo.List(ctx, domain.PendingActionFilter{UserID: userID, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, actions, 1)
}
