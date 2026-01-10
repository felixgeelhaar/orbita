// Package persistence provides PostgreSQL implementations for automation repositories.
package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/felixgeelhaar/orbita/db/generated"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/convert"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RuleRepository implements domain.RuleRepository using PostgreSQL.
type RuleRepository struct {
	queries *db.Queries
}

// NewRuleRepository creates a new PostgreSQL rule repository.
func NewRuleRepository(queries *db.Queries) *RuleRepository {
	return &RuleRepository{queries: queries}
}

// Create creates a new automation rule.
func (r *RuleRepository) Create(ctx context.Context, rule *domain.AutomationRule) error {
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

	params := db.CreateAutomationRuleParams{
		ID:                toPgUUID(rule.ID),
		UserID:            toPgUUID(rule.UserID),
		Name:              rule.Name,
		Description:       toPgText(rule.Description),
		Enabled:           rule.Enabled,
		Priority:          convert.IntToInt32Safe(rule.Priority),
		TriggerType:       string(rule.TriggerType),
		TriggerConfig:     triggerConfig,
		Conditions:        conditions,
		ConditionOperator: string(rule.ConditionOperator),
		Actions:           actions,
		CooldownSeconds:   convert.IntToInt32Safe(rule.CooldownSeconds),
		Tags:              rule.Tags,
		CreatedAt:         toPgTimestamp(rule.CreatedAt),
		UpdatedAt:         toPgTimestamp(rule.UpdatedAt),
	}

	if rule.MaxExecutionsPerHour != nil {
		params.MaxExecutionsPerHour = pgtype.Int4{Int32: convert.IntToInt32Safe(*rule.MaxExecutionsPerHour), Valid: true}
	}
	if rule.LastTriggeredAt != nil {
		params.LastTriggeredAt = toPgTimestamp(*rule.LastTriggeredAt)
	}

	return r.queries.CreateAutomationRule(ctx, params)
}

// Update updates an existing automation rule.
func (r *RuleRepository) Update(ctx context.Context, rule *domain.AutomationRule) error {
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

	params := db.UpdateAutomationRuleParams{
		ID:                toPgUUID(rule.ID),
		Name:              rule.Name,
		Description:       toPgText(rule.Description),
		Enabled:           rule.Enabled,
		Priority:          convert.IntToInt32Safe(rule.Priority),
		TriggerType:       string(rule.TriggerType),
		TriggerConfig:     triggerConfig,
		Conditions:        conditions,
		ConditionOperator: string(rule.ConditionOperator),
		Actions:           actions,
		CooldownSeconds:   convert.IntToInt32Safe(rule.CooldownSeconds),
		Tags:              rule.Tags,
		UpdatedAt:         toPgTimestamp(rule.UpdatedAt),
	}

	if rule.MaxExecutionsPerHour != nil {
		params.MaxExecutionsPerHour = pgtype.Int4{Int32: convert.IntToInt32Safe(*rule.MaxExecutionsPerHour), Valid: true}
	}
	if rule.LastTriggeredAt != nil {
		params.LastTriggeredAt = toPgTimestamp(*rule.LastTriggeredAt)
	}

	return r.queries.UpdateAutomationRule(ctx, params)
}

// Delete deletes an automation rule by ID.
func (r *RuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteAutomationRule(ctx, toPgUUID(id))
}

// GetByID retrieves a rule by ID.
func (r *RuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AutomationRule, error) {
	row, err := r.queries.GetAutomationRuleByID(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrRuleNotFound
		}
		return nil, err
	}
	return r.toDomainRule(row)
}

// GetByUserID retrieves all rules for a user.
func (r *RuleRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AutomationRule, error) {
	rows, err := r.queries.GetAutomationRulesByUserID(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}
	return r.toDomainRules(rows)
}

// List retrieves rules matching the filter.
func (r *RuleRepository) List(ctx context.Context, filter domain.RuleFilter) ([]*domain.AutomationRule, int64, error) {
	params := db.ListAutomationRulesParams{
		UserID: toPgUUID(filter.UserID),
		Limit:  convert.IntToInt32Safe(filter.Limit),
		Offset: convert.IntToInt32Safe(filter.Offset),
	}

	if filter.Enabled != nil {
		params.Column2 = *filter.Enabled
	}
	if filter.TriggerType != nil {
		params.Column3 = string(*filter.TriggerType)
	}
	if len(filter.Tags) > 0 {
		params.Column4 = filter.Tags
	}

	rows, err := r.queries.ListAutomationRules(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	countParams := db.CountAutomationRulesParams{
		UserID:  params.UserID,
		Column2: params.Column2,
		Column3: params.Column3,
		Column4: params.Column4,
	}
	total, err := r.queries.CountAutomationRules(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}

	rules, err := r.toDomainRules(rows)
	if err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// GetEnabledByTriggerType retrieves enabled rules by trigger type.
func (r *RuleRepository) GetEnabledByTriggerType(ctx context.Context, userID uuid.UUID, triggerType domain.TriggerType) ([]*domain.AutomationRule, error) {
	params := db.GetEnabledAutomationRulesByTriggerTypeParams{
		UserID:      toPgUUID(userID),
		TriggerType: string(triggerType),
	}
	rows, err := r.queries.GetEnabledAutomationRulesByTriggerType(ctx, params)
	if err != nil {
		return nil, err
	}
	return r.toDomainRules(rows)
}

// GetEnabledByEventType retrieves enabled rules that trigger on a specific event type.
func (r *RuleRepository) GetEnabledByEventType(ctx context.Context, userID uuid.UUID, eventType string) ([]*domain.AutomationRule, error) {
	// First get all event-triggered rules, then filter by event type
	params := db.GetEnabledAutomationRulesByTriggerTypeParams{
		UserID:      toPgUUID(userID),
		TriggerType: string(domain.TriggerTypeEvent),
	}
	rows, err := r.queries.GetEnabledAutomationRulesByTriggerType(ctx, params)
	if err != nil {
		return nil, err
	}

	allRules, err := r.toDomainRules(rows)
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
func (r *RuleRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.queries.CountAutomationRulesByUserID(ctx, toPgUUID(userID))
}

// Helper methods

func (r *RuleRepository) toDomainRule(row db.AutomationRule) (*domain.AutomationRule, error) {
	var triggerConfig map[string]any
	if err := json.Unmarshal(row.TriggerConfig, &triggerConfig); err != nil {
		return nil, err
	}

	var conditions []types.RuleCondition
	if err := json.Unmarshal(row.Conditions, &conditions); err != nil {
		return nil, err
	}

	var actions []types.RuleAction
	if err := json.Unmarshal(row.Actions, &actions); err != nil {
		return nil, err
	}

	rule := &domain.AutomationRule{
		ID:                fromPgUUID(row.ID),
		UserID:            fromPgUUID(row.UserID),
		Name:              row.Name,
		Description:       fromPgText(row.Description),
		Enabled:           row.Enabled,
		Priority:          int(row.Priority),
		TriggerType:       domain.TriggerType(row.TriggerType),
		TriggerConfig:     triggerConfig,
		Conditions:        conditions,
		ConditionOperator: domain.ConditionOperator(row.ConditionOperator),
		Actions:           actions,
		CooldownSeconds:   int(row.CooldownSeconds),
		Tags:              row.Tags,
		CreatedAt:         fromPgTimestamp(row.CreatedAt),
		UpdatedAt:         fromPgTimestamp(row.UpdatedAt),
	}

	if row.MaxExecutionsPerHour.Valid {
		maxExec := int(row.MaxExecutionsPerHour.Int32)
		rule.MaxExecutionsPerHour = &maxExec
	}
	if row.LastTriggeredAt.Valid {
		lastTriggered := row.LastTriggeredAt.Time
		rule.LastTriggeredAt = &lastTriggered
	}

	return rule, nil
}

func (r *RuleRepository) toDomainRules(rows []db.AutomationRule) ([]*domain.AutomationRule, error) {
	rules := make([]*domain.AutomationRule, 0, len(rows))
	for _, row := range rows {
		rule, err := r.toDomainRule(row)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// Type conversion helpers

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return id.Bytes
}

func toPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func fromPgText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func toPgTimestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func fromPgTimestamp(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}
