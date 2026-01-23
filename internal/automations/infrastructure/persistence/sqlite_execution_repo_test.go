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

func setupExecutionTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create users table
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Create automation_rules table
	_, err = db.Exec(`
		CREATE TABLE automation_rules (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			priority INTEGER NOT NULL DEFAULT 0,
			trigger_type TEXT NOT NULL,
			trigger_config TEXT NOT NULL DEFAULT '{}',
			conditions TEXT NOT NULL DEFAULT '[]',
			condition_operator TEXT NOT NULL DEFAULT 'AND',
			actions TEXT NOT NULL DEFAULT '[]',
			cooldown_seconds INTEGER NOT NULL DEFAULT 0,
			max_executions_per_hour INTEGER,
			tags TEXT DEFAULT '[]',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_triggered_at TEXT
		)
	`)
	require.NoError(t, err)

	// Create automation_rule_executions table
	_, err = db.Exec(`
		CREATE TABLE automation_rule_executions (
			id TEXT PRIMARY KEY,
			rule_id TEXT NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			trigger_event_type TEXT,
			trigger_event_payload TEXT,
			status TEXT NOT NULL CHECK (status IN ('success', 'failed', 'skipped', 'pending', 'partial')),
			actions_executed TEXT NOT NULL DEFAULT '[]',
			error_message TEXT,
			error_details TEXT,
			started_at TEXT NOT NULL,
			completed_at TEXT,
			duration_ms INTEGER,
			skip_reason TEXT
		)
	`)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func createExecutionTestUser(t *testing.T, db *sql.DB, userID uuid.UUID) {
	_, err := db.Exec(
		"INSERT INTO users (id, email, name) VALUES (?, ?, ?)",
		userID.String(), "exec@example.com", "Execution User",
	)
	require.NoError(t, err)
}

func createExecutionTestRule(t *testing.T, db *sql.DB, ruleID, userID uuid.UUID) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO automation_rules (id, user_id, name, trigger_type, created_at, updated_at)
		VALUES (?, ?, 'Test Rule', 'event', ?, ?)
	`, ruleID.String(), userID.String(), now, now)
	require.NoError(t, err)
}

func createTestExecution(ruleID, userID uuid.UUID) *domain.RuleExecution {
	durationMs := 150
	completedAt := time.Now().UTC().Truncate(time.Second)
	return &domain.RuleExecution{
		ID:                  uuid.New(),
		RuleID:              ruleID,
		UserID:              userID,
		TriggerEventType:    "task.created",
		TriggerEventPayload: map[string]any{"task_id": "123"},
		Status:              domain.ExecutionStatusSuccess,
		ActionsExecuted:     []domain.ActionResult{{Action: "notify", Status: "success"}},
		StartedAt:           time.Now().UTC().Add(-time.Second).Truncate(time.Second),
		CompletedAt:         &completedAt,
		DurationMs:          &durationMs,
	}
}

func TestNewSQLiteExecutionRepository(t *testing.T) {
	db := setupExecutionTestDB(t)
	repo := NewSQLiteExecutionRepository(db)
	assert.NotNil(t, repo)
}

func TestSQLiteExecutionRepository_Create(t *testing.T) {
	db := setupExecutionTestDB(t)
	repo := NewSQLiteExecutionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	ruleID := uuid.New()
	createExecutionTestUser(t, db, userID)
	createExecutionTestRule(t, db, ruleID, userID)

	execution := createTestExecution(ruleID, userID)

	err := repo.Create(ctx, execution)
	require.NoError(t, err)

	// Verify execution was created
	loaded, err := repo.GetByID(ctx, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, execution.RuleID, loaded.RuleID)
	assert.Equal(t, execution.Status, loaded.Status)
	assert.Equal(t, execution.TriggerEventType, loaded.TriggerEventType)
}

func TestSQLiteExecutionRepository_GetByID_NotFound(t *testing.T) {
	db := setupExecutionTestDB(t)
	repo := NewSQLiteExecutionRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrExecutionNotFound)
}

func TestSQLiteExecutionRepository_Update(t *testing.T) {
	db := setupExecutionTestDB(t)
	repo := NewSQLiteExecutionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	ruleID := uuid.New()
	createExecutionTestUser(t, db, userID)
	createExecutionTestRule(t, db, ruleID, userID)

	execution := createTestExecution(ruleID, userID)
	err := repo.Create(ctx, execution)
	require.NoError(t, err)

	// Update execution
	execution.Status = domain.ExecutionStatusFailed
	execution.ErrorMessage = "Something went wrong"
	newDuration := 250
	execution.DurationMs = &newDuration

	err = repo.Update(ctx, execution)
	require.NoError(t, err)

	// Verify update
	loaded, err := repo.GetByID(ctx, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusFailed, loaded.Status)
	assert.Equal(t, "Something went wrong", loaded.ErrorMessage)
	assert.NotNil(t, loaded.DurationMs)
	assert.Equal(t, 250, *loaded.DurationMs)
}

func TestSQLiteExecutionRepository_GetByRuleID(t *testing.T) {
	db := setupExecutionTestDB(t)
	repo := NewSQLiteExecutionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	ruleID := uuid.New()
	createExecutionTestUser(t, db, userID)
	createExecutionTestRule(t, db, ruleID, userID)

	// Create multiple executions
	for i := 0; i < 5; i++ {
		execution := createTestExecution(ruleID, userID)
		execution.StartedAt = time.Now().Add(time.Duration(i) * time.Minute)
		err := repo.Create(ctx, execution)
		require.NoError(t, err)
	}

	// Get executions with limit
	executions, err := repo.GetByRuleID(ctx, ruleID, 3)
	require.NoError(t, err)
	assert.Len(t, executions, 3)
}

func TestSQLiteExecutionRepository_List(t *testing.T) {
	db := setupExecutionTestDB(t)
	repo := NewSQLiteExecutionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	ruleID := uuid.New()
	createExecutionTestUser(t, db, userID)
	createExecutionTestRule(t, db, ruleID, userID)

	// Create some executions
	for i := 0; i < 3; i++ {
		execution := createTestExecution(ruleID, userID)
		err := repo.Create(ctx, execution)
		require.NoError(t, err)
	}

	// List with filter
	executions, total, err := repo.List(ctx, domain.ExecutionFilter{
		UserID: userID,
		RuleID: &ruleID,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Len(t, executions, 3)
	assert.Equal(t, int64(3), total)
}

func TestSQLiteExecutionRepository_CountByRuleIDSince(t *testing.T) {
	db := setupExecutionTestDB(t)
	repo := NewSQLiteExecutionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	ruleID := uuid.New()
	createExecutionTestUser(t, db, userID)
	createExecutionTestRule(t, db, ruleID, userID)

	// Create executions at different times
	execution1 := createTestExecution(ruleID, userID)
	execution1.StartedAt = time.Now().Add(-30 * time.Minute) // 30 min ago
	err := repo.Create(ctx, execution1)
	require.NoError(t, err)

	execution2 := createTestExecution(ruleID, userID)
	execution2.StartedAt = time.Now().Add(-5 * time.Minute) // 5 min ago
	err = repo.Create(ctx, execution2)
	require.NoError(t, err)

	// Count executions since 10 minutes ago
	count, err := repo.CountByRuleIDSince(ctx, ruleID, time.Now().Add(-10*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Count executions since 1 hour ago
	count, err = repo.CountByRuleIDSince(ctx, ruleID, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestSQLiteExecutionRepository_WithSkipReason(t *testing.T) {
	db := setupExecutionTestDB(t)
	repo := NewSQLiteExecutionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	ruleID := uuid.New()
	createExecutionTestUser(t, db, userID)
	createExecutionTestRule(t, db, ruleID, userID)

	execution := createTestExecution(ruleID, userID)
	execution.Status = domain.ExecutionStatusSkipped
	execution.SkipReason = "Cooldown period active"

	err := repo.Create(ctx, execution)
	require.NoError(t, err)

	loaded, err := repo.GetByID(ctx, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusSkipped, loaded.Status)
	assert.Equal(t, "Cooldown period active", loaded.SkipReason)
}
