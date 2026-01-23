// Package persistence provides database implementations for automation repositories.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// SQLiteRuleRepository implements domain.RuleRepository using SQLite.
type SQLiteRuleRepository struct {
	db *sql.DB
}

// NewSQLiteRuleRepository creates a new SQLite rule repository.
func NewSQLiteRuleRepository(db *sql.DB) *SQLiteRuleRepository {
	return &SQLiteRuleRepository{db: db}
}

// Create creates a new automation rule.
func (r *SQLiteRuleRepository) Create(ctx context.Context, rule *domain.AutomationRule) error {
	triggerConfig, err := json.Marshal(rule.TriggerConfig)
	if err != nil {
		return err
	}
	conditions, err := json.Marshal(rule.Conditions)
	if err != nil {
		return err
	}
	actions, err := json.Marshal(rule.Actions)
	if err != nil {
		return err
	}
	tags, err := json.Marshal(rule.Tags)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO automation_rules (
			id, user_id, name, description, enabled, priority,
			trigger_type, trigger_config, conditions, condition_operator,
			actions, cooldown_seconds, max_executions_per_hour, tags,
			created_at, updated_at, last_triggered_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var maxExecPerHour sql.NullInt32
	if rule.MaxExecutionsPerHour != nil {
		maxExecPerHour = sql.NullInt32{Int32: int32(*rule.MaxExecutionsPerHour), Valid: true}
	}

	var lastTriggeredAt sql.NullString
	if rule.LastTriggeredAt != nil {
		lastTriggeredAt = sql.NullString{String: rule.LastTriggeredAt.Format(time.RFC3339), Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		rule.ID.String(),
		rule.UserID.String(),
		rule.Name,
		rule.Description,
		boolToInt(rule.Enabled),
		rule.Priority,
		string(rule.TriggerType),
		string(triggerConfig),
		string(conditions),
		string(rule.ConditionOperator),
		string(actions),
		rule.CooldownSeconds,
		maxExecPerHour,
		string(tags),
		rule.CreatedAt.Format(time.RFC3339),
		rule.UpdatedAt.Format(time.RFC3339),
		lastTriggeredAt,
	)
	return err
}

// Update updates an existing automation rule.
func (r *SQLiteRuleRepository) Update(ctx context.Context, rule *domain.AutomationRule) error {
	triggerConfig, err := json.Marshal(rule.TriggerConfig)
	if err != nil {
		return err
	}
	conditions, err := json.Marshal(rule.Conditions)
	if err != nil {
		return err
	}
	actions, err := json.Marshal(rule.Actions)
	if err != nil {
		return err
	}
	tags, err := json.Marshal(rule.Tags)
	if err != nil {
		return err
	}

	query := `
		UPDATE automation_rules SET
			name = ?, description = ?, enabled = ?, priority = ?,
			trigger_type = ?, trigger_config = ?, conditions = ?, condition_operator = ?,
			actions = ?, cooldown_seconds = ?, max_executions_per_hour = ?, tags = ?,
			updated_at = ?, last_triggered_at = ?
		WHERE id = ?
	`

	var maxExecPerHour sql.NullInt32
	if rule.MaxExecutionsPerHour != nil {
		maxExecPerHour = sql.NullInt32{Int32: int32(*rule.MaxExecutionsPerHour), Valid: true}
	}

	var lastTriggeredAt sql.NullString
	if rule.LastTriggeredAt != nil {
		lastTriggeredAt = sql.NullString{String: rule.LastTriggeredAt.Format(time.RFC3339), Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		rule.Name,
		rule.Description,
		boolToInt(rule.Enabled),
		rule.Priority,
		string(rule.TriggerType),
		string(triggerConfig),
		string(conditions),
		string(rule.ConditionOperator),
		string(actions),
		rule.CooldownSeconds,
		maxExecPerHour,
		string(tags),
		rule.UpdatedAt.Format(time.RFC3339),
		lastTriggeredAt,
		rule.ID.String(),
	)
	return err
}

// Delete deletes an automation rule by ID.
func (r *SQLiteRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM automation_rules WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id.String())
	return err
}

// GetByID retrieves a rule by ID.
func (r *SQLiteRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AutomationRule, error) {
	query := `
		SELECT id, user_id, name, description, enabled, priority,
			trigger_type, trigger_config, conditions, condition_operator,
			actions, cooldown_seconds, max_executions_per_hour, tags,
			created_at, updated_at, last_triggered_at
		FROM automation_rules
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanRule(row)
}

// GetByUserID retrieves all rules for a user.
func (r *SQLiteRuleRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AutomationRule, error) {
	query := `
		SELECT id, user_id, name, description, enabled, priority,
			trigger_type, trigger_config, conditions, condition_operator,
			actions, cooldown_seconds, max_executions_per_hour, tags,
			created_at, updated_at, last_triggered_at
		FROM automation_rules
		WHERE user_id = ?
		ORDER BY priority DESC, created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRules(rows)
}

// List retrieves rules matching the filter.
func (r *SQLiteRuleRepository) List(ctx context.Context, filter domain.RuleFilter) ([]*domain.AutomationRule, int64, error) {
	query := `
		SELECT id, user_id, name, description, enabled, priority,
			trigger_type, trigger_config, conditions, condition_operator,
			actions, cooldown_seconds, max_executions_per_hour, tags,
			created_at, updated_at, last_triggered_at
		FROM automation_rules
		WHERE user_id = ?
	`
	countQuery := `SELECT COUNT(*) FROM automation_rules WHERE user_id = ?`
	args := []any{filter.UserID.String()}
	countArgs := []any{filter.UserID.String()}

	if filter.Enabled != nil {
		query += " AND enabled = ?"
		countQuery += " AND enabled = ?"
		args = append(args, boolToInt(*filter.Enabled))
		countArgs = append(countArgs, boolToInt(*filter.Enabled))
	}
	if filter.TriggerType != nil {
		query += " AND trigger_type = ?"
		countQuery += " AND trigger_type = ?"
		args = append(args, string(*filter.TriggerType))
		countArgs = append(countArgs, string(*filter.TriggerType))
	}
	// Note: Tag filtering in SQLite JSON requires json_each which adds complexity
	// For simplicity, we skip tag filtering for now

	query += " ORDER BY priority DESC, created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	// Get total count
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rules, err := r.scanRules(rows)
	if err != nil {
		return nil, 0, err
	}
	return rules, total, nil
}

// GetEnabledByTriggerType retrieves enabled rules by trigger type.
func (r *SQLiteRuleRepository) GetEnabledByTriggerType(ctx context.Context, userID uuid.UUID, triggerType domain.TriggerType) ([]*domain.AutomationRule, error) {
	query := `
		SELECT id, user_id, name, description, enabled, priority,
			trigger_type, trigger_config, conditions, condition_operator,
			actions, cooldown_seconds, max_executions_per_hour, tags,
			created_at, updated_at, last_triggered_at
		FROM automation_rules
		WHERE user_id = ? AND enabled = 1 AND trigger_type = ?
		ORDER BY priority DESC, created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), string(triggerType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRules(rows)
}

// GetEnabledByEventType retrieves enabled rules that trigger on a specific event type.
func (r *SQLiteRuleRepository) GetEnabledByEventType(ctx context.Context, userID uuid.UUID, eventType string) ([]*domain.AutomationRule, error) {
	// Get all event-triggered rules, then filter by event type
	allRules, err := r.GetEnabledByTriggerType(ctx, userID, domain.TriggerTypeEvent)
	if err != nil {
		return nil, err
	}

	// Filter by event type in trigger config
	var result []*domain.AutomationRule
	for _, rule := range allRules {
		if eventTypes, ok := rule.TriggerConfig["event_types"].([]any); ok {
			for _, et := range eventTypes {
				if s, ok := et.(string); ok && s == eventType {
					result = append(result, rule)
					break
				}
			}
		}
	}
	return result, nil
}

// CountByUserID counts rules for a user.
func (r *SQLiteRuleRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM automation_rules WHERE user_id = ?`
	var count int64
	err := r.db.QueryRowContext(ctx, query, userID.String()).Scan(&count)
	return count, err
}

// Helper methods

func (r *SQLiteRuleRepository) scanRule(row *sql.Row) (*domain.AutomationRule, error) {
	var rule domain.AutomationRule
	var idStr, userIDStr string
	var description sql.NullString
	var enabled int
	var triggerConfigStr, conditionsStr, actionsStr, tagsStr string
	var conditionOperator string
	var maxExecPerHour sql.NullInt32
	var createdAtStr, updatedAtStr string
	var lastTriggeredAtStr sql.NullString

	err := row.Scan(
		&idStr,
		&userIDStr,
		&rule.Name,
		&description,
		&enabled,
		&rule.Priority,
		&rule.TriggerType,
		&triggerConfigStr,
		&conditionsStr,
		&conditionOperator,
		&actionsStr,
		&rule.CooldownSeconds,
		&maxExecPerHour,
		&tagsStr,
		&createdAtStr,
		&updatedAtStr,
		&lastTriggeredAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRuleNotFound
		}
		return nil, err
	}

	rule.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	rule.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	rule.Description = description.String
	rule.Enabled = enabled == 1
	rule.ConditionOperator = domain.ConditionOperator(conditionOperator)

	if err := json.Unmarshal([]byte(triggerConfigStr), &rule.TriggerConfig); err != nil {
		return nil, err
	}

	var conditions []types.RuleCondition
	if err := json.Unmarshal([]byte(conditionsStr), &conditions); err != nil {
		return nil, err
	}
	rule.Conditions = conditions

	var actions []types.RuleAction
	if err := json.Unmarshal([]byte(actionsStr), &actions); err != nil {
		return nil, err
	}
	rule.Actions = actions

	var tags []string
	if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
		return nil, err
	}
	rule.Tags = tags

	if maxExecPerHour.Valid {
		max := int(maxExecPerHour.Int32)
		rule.MaxExecutionsPerHour = &max
	}

	rule.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}
	rule.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, err
	}

	if lastTriggeredAtStr.Valid {
		lastTriggered, err := time.Parse(time.RFC3339, lastTriggeredAtStr.String)
		if err == nil {
			rule.LastTriggeredAt = &lastTriggered
		}
	}

	return &rule, nil
}

func (r *SQLiteRuleRepository) scanRules(rows *sql.Rows) ([]*domain.AutomationRule, error) {
	var rules []*domain.AutomationRule
	for rows.Next() {
		var rule domain.AutomationRule
		var idStr, userIDStr string
		var description sql.NullString
		var enabled int
		var triggerConfigStr, conditionsStr, actionsStr, tagsStr string
		var conditionOperator string
		var maxExecPerHour sql.NullInt32
		var createdAtStr, updatedAtStr string
		var lastTriggeredAtStr sql.NullString

		err := rows.Scan(
			&idStr,
			&userIDStr,
			&rule.Name,
			&description,
			&enabled,
			&rule.Priority,
			&rule.TriggerType,
			&triggerConfigStr,
			&conditionsStr,
			&conditionOperator,
			&actionsStr,
			&rule.CooldownSeconds,
			&maxExecPerHour,
			&tagsStr,
			&createdAtStr,
			&updatedAtStr,
			&lastTriggeredAtStr,
		)
		if err != nil {
			return nil, err
		}

		rule.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		rule.UserID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, err
		}

		rule.Description = description.String
		rule.Enabled = enabled == 1
		rule.ConditionOperator = domain.ConditionOperator(conditionOperator)

		if err := json.Unmarshal([]byte(triggerConfigStr), &rule.TriggerConfig); err != nil {
			return nil, err
		}

		var conditions []types.RuleCondition
		if err := json.Unmarshal([]byte(conditionsStr), &conditions); err != nil {
			return nil, err
		}
		rule.Conditions = conditions

		var actions []types.RuleAction
		if err := json.Unmarshal([]byte(actionsStr), &actions); err != nil {
			return nil, err
		}
		rule.Actions = actions

		var tags []string
		if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
			return nil, err
		}
		rule.Tags = tags

		if maxExecPerHour.Valid {
			max := int(maxExecPerHour.Int32)
			rule.MaxExecutionsPerHour = &max
		}

		rule.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, err
		}
		rule.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
		if err != nil {
			return nil, err
		}

		if lastTriggeredAtStr.Valid {
			lastTriggered, err := time.Parse(time.RFC3339, lastTriggeredAtStr.String)
			if err == nil {
				rule.LastTriggeredAt = &lastTriggered
			}
		}

		rules = append(rules, &rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
