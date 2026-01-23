package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create users table (required for foreign key)
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
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
			trigger_type TEXT NOT NULL CHECK (trigger_type IN ('event', 'schedule', 'state_change', 'pattern')),
			trigger_config TEXT NOT NULL DEFAULT '{}',
			conditions TEXT NOT NULL DEFAULT '[]',
			condition_operator TEXT NOT NULL DEFAULT 'AND' CHECK (condition_operator IN ('AND', 'OR')),
			actions TEXT NOT NULL DEFAULT '[]',
			cooldown_seconds INTEGER NOT NULL DEFAULT 0,
			max_executions_per_hour INTEGER,
			tags TEXT DEFAULT '[]',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			last_triggered_at TEXT
		)
	`)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func createTestUser(t *testing.T, db *sql.DB, userID uuid.UUID) {
	_, err := db.Exec(
		"INSERT INTO users (id, email, name) VALUES (?, ?, ?)",
		userID.String(), "test@example.com", "Test User",
	)
	require.NoError(t, err)
}

func createTestRule(userID uuid.UUID) *domain.AutomationRule {
	maxExec := 10
	return &domain.AutomationRule{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        "Test Rule",
		Description: "A test automation rule",
		Enabled:     true,
		Priority:    5,
		TriggerType: domain.TriggerTypeEvent,
		TriggerConfig: map[string]any{
			"event_types": []any{"task.created", "task.completed"},
		},
		Conditions: []types.RuleCondition{
			{
				Field:    "priority",
				Operator: types.OperatorEquals,
				Value:    "high",
			},
		},
		ConditionOperator:    domain.ConditionOperatorAND,
		Actions:              []types.RuleAction{{Type: "notify", Parameters: map[string]any{"channel": "email"}}},
		CooldownSeconds:      60,
		MaxExecutionsPerHour: &maxExec,
		Tags:                 []string{"test", "automation"},
		CreatedAt:            time.Now().UTC().Truncate(time.Second),
		UpdatedAt:            time.Now().UTC().Truncate(time.Second),
	}
}

func TestNewSQLiteRuleRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	assert.NotNil(t, repo)
}

func TestSQLiteRuleRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	rule := createTestRule(userID)

	err := repo.Create(ctx, rule)
	require.NoError(t, err)

	// Verify rule was created
	loaded, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, rule.Name, loaded.Name)
	assert.Equal(t, rule.Description, loaded.Description)
	assert.Equal(t, rule.Enabled, loaded.Enabled)
	assert.Equal(t, rule.Priority, loaded.Priority)
}

func TestSQLiteRuleRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrRuleNotFound)
}

func TestSQLiteRuleRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	rule := createTestRule(userID)
	err := repo.Create(ctx, rule)
	require.NoError(t, err)

	// Update rule
	rule.Name = "Updated Rule"
	rule.Enabled = false
	rule.Priority = 10
	rule.UpdatedAt = time.Now().UTC().Truncate(time.Second)

	err = repo.Update(ctx, rule)
	require.NoError(t, err)

	// Verify update
	loaded, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Rule", loaded.Name)
	assert.False(t, loaded.Enabled)
	assert.Equal(t, 10, loaded.Priority)
}

func TestSQLiteRuleRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	rule := createTestRule(userID)
	err := repo.Create(ctx, rule)
	require.NoError(t, err)

	// Delete rule
	err = repo.Delete(ctx, rule.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, rule.ID)
	assert.ErrorIs(t, err, domain.ErrRuleNotFound)
}

func TestSQLiteRuleRepository_GetByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	// Create multiple rules
	rule1 := createTestRule(userID)
	rule1.Name = "Rule 1"
	rule1.Priority = 1
	err := repo.Create(ctx, rule1)
	require.NoError(t, err)

	rule2 := createTestRule(userID)
	rule2.Name = "Rule 2"
	rule2.Priority = 2
	err = repo.Create(ctx, rule2)
	require.NoError(t, err)

	// Get rules by user ID
	rules, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, rules, 2)

	// Should be ordered by priority DESC
	assert.Equal(t, "Rule 2", rules[0].Name)
	assert.Equal(t, "Rule 1", rules[1].Name)
}

func TestSQLiteRuleRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	// Create rules with different settings
	rule1 := createTestRule(userID)
	rule1.Enabled = true
	rule1.TriggerType = domain.TriggerTypeEvent
	err := repo.Create(ctx, rule1)
	require.NoError(t, err)

	rule2 := createTestRule(userID)
	rule2.Enabled = false
	rule2.TriggerType = domain.TriggerTypeSchedule
	err = repo.Create(ctx, rule2)
	require.NoError(t, err)

	t.Run("filter by enabled", func(t *testing.T) {
		enabled := true
		rules, total, err := repo.List(ctx, domain.RuleFilter{
			UserID:  userID,
			Enabled: &enabled,
		})
		require.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, int64(1), total)
	})

	t.Run("filter by trigger type", func(t *testing.T) {
		triggerType := domain.TriggerTypeSchedule
		rules, total, err := repo.List(ctx, domain.RuleFilter{
			UserID:      userID,
			TriggerType: &triggerType,
		})
		require.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, int64(1), total)
	})

	t.Run("with limit and offset", func(t *testing.T) {
		rules, total, err := repo.List(ctx, domain.RuleFilter{
			UserID: userID,
			Limit:  1,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, int64(2), total)
	})
}

func TestSQLiteRuleRepository_GetEnabledByTriggerType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	// Create enabled event rule
	rule1 := createTestRule(userID)
	rule1.Enabled = true
	rule1.TriggerType = domain.TriggerTypeEvent
	err := repo.Create(ctx, rule1)
	require.NoError(t, err)

	// Create disabled event rule
	rule2 := createTestRule(userID)
	rule2.Enabled = false
	rule2.TriggerType = domain.TriggerTypeEvent
	err = repo.Create(ctx, rule2)
	require.NoError(t, err)

	// Create enabled schedule rule
	rule3 := createTestRule(userID)
	rule3.Enabled = true
	rule3.TriggerType = domain.TriggerTypeSchedule
	err = repo.Create(ctx, rule3)
	require.NoError(t, err)

	// Get enabled event rules
	rules, err := repo.GetEnabledByTriggerType(ctx, userID, domain.TriggerTypeEvent)
	require.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, rule1.ID, rules[0].ID)
}

func TestSQLiteRuleRepository_GetEnabledByEventType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	// Create rule for task.created events
	rule1 := createTestRule(userID)
	rule1.TriggerType = domain.TriggerTypeEvent
	rule1.TriggerConfig = map[string]any{
		"event_types": []any{"task.created"},
	}
	err := repo.Create(ctx, rule1)
	require.NoError(t, err)

	// Create rule for task.completed events
	rule2 := createTestRule(userID)
	rule2.TriggerType = domain.TriggerTypeEvent
	rule2.TriggerConfig = map[string]any{
		"event_types": []any{"task.completed"},
	}
	err = repo.Create(ctx, rule2)
	require.NoError(t, err)

	// Get rules for task.created
	rules, err := repo.GetEnabledByEventType(ctx, userID, "task.created")
	require.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, rule1.ID, rules[0].ID)
}

func TestSQLiteRuleRepository_CountByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	// Initially no rules
	count, err := repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Create some rules
	for i := 0; i < 3; i++ {
		rule := createTestRule(userID)
		err := repo.Create(ctx, rule)
		require.NoError(t, err)
	}

	// Count should be 3
	count, err = repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestSQLiteRuleRepository_WithLastTriggeredAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteRuleRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	createTestUser(t, db, userID)

	rule := createTestRule(userID)
	lastTriggered := time.Now().UTC().Truncate(time.Second)
	rule.LastTriggeredAt = &lastTriggered

	err := repo.Create(ctx, rule)
	require.NoError(t, err)

	loaded, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.LastTriggeredAt)
	assert.Equal(t, lastTriggered.Unix(), loaded.LastTriggeredAt.Unix())
}

func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}
