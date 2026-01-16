package persistence

import (
	"context"
	"encoding/json"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/postgres"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/convert"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ExecutionRepository implements domain.ExecutionRepository using PostgreSQL.
type ExecutionRepository struct {
	queries *db.Queries
}

// NewExecutionRepository creates a new PostgreSQL execution repository.
func NewExecutionRepository(queries *db.Queries) *ExecutionRepository {
	return &ExecutionRepository{queries: queries}
}

// Create creates a new execution record.
func (r *ExecutionRepository) Create(ctx context.Context, execution *domain.RuleExecution) error {
	triggerPayload, err := json.Marshal(execution.TriggerEventPayload)
	if err != nil {
		return err
	}
	actionsExecuted, err := json.Marshal(execution.ActionsExecuted)
	if err != nil {
		return err
	}
	errorDetails, err := json.Marshal(execution.ErrorDetails)
	if err != nil {
		return err
	}

	params := db.CreateAutomationRuleExecutionParams{
		ID:                  toPgUUID(execution.ID),
		RuleID:              toPgUUID(execution.RuleID),
		UserID:              toPgUUID(execution.UserID),
		TriggerEventType:    toPgText(execution.TriggerEventType),
		TriggerEventPayload: triggerPayload,
		Status:              string(execution.Status),
		ActionsExecuted:     actionsExecuted,
		ErrorMessage:        toPgText(execution.ErrorMessage),
		ErrorDetails:        errorDetails,
		StartedAt:           toPgTimestamp(execution.StartedAt),
		SkipReason:          toPgText(execution.SkipReason),
	}

	if execution.CompletedAt != nil {
		params.CompletedAt = toPgTimestamp(*execution.CompletedAt)
	}
	if execution.DurationMs != nil {
		params.DurationMs = pgtype.Int4{Int32: convert.IntToInt32Safe(*execution.DurationMs), Valid: true}
	}

	return r.queries.CreateAutomationRuleExecution(ctx, params)
}

// Update updates an execution record.
func (r *ExecutionRepository) Update(ctx context.Context, execution *domain.RuleExecution) error {
	actionsExecuted, err := json.Marshal(execution.ActionsExecuted)
	if err != nil {
		return err
	}
	errorDetails, err := json.Marshal(execution.ErrorDetails)
	if err != nil {
		return err
	}

	params := db.UpdateAutomationRuleExecutionParams{
		ID:              toPgUUID(execution.ID),
		Status:          string(execution.Status),
		ActionsExecuted: actionsExecuted,
		ErrorMessage:    toPgText(execution.ErrorMessage),
		ErrorDetails:    errorDetails,
		SkipReason:      toPgText(execution.SkipReason),
	}

	if execution.CompletedAt != nil {
		params.CompletedAt = toPgTimestamp(*execution.CompletedAt)
	}
	if execution.DurationMs != nil {
		params.DurationMs = pgtype.Int4{Int32: convert.IntToInt32Safe(*execution.DurationMs), Valid: true}
	}

	return r.queries.UpdateAutomationRuleExecution(ctx, params)
}

// GetByID retrieves an execution by ID.
func (r *ExecutionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RuleExecution, error) {
	row, err := r.queries.GetAutomationRuleExecutionByID(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrExecutionNotFound
		}
		return nil, err
	}
	return r.toDomainExecution(row)
}

// GetByRuleID retrieves executions for a rule.
func (r *ExecutionRepository) GetByRuleID(ctx context.Context, ruleID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	params := db.GetAutomationRuleExecutionsByRuleIDParams{
		RuleID: toPgUUID(ruleID),
		Limit:  convert.IntToInt32Safe(limit),
	}
	rows, err := r.queries.GetAutomationRuleExecutionsByRuleID(ctx, params)
	if err != nil {
		return nil, err
	}
	return r.toDomainExecutions(rows)
}

// List retrieves executions matching the filter.
func (r *ExecutionRepository) List(ctx context.Context, filter domain.ExecutionFilter) ([]*domain.RuleExecution, int64, error) {
	params := db.ListAutomationRuleExecutionsParams{
		UserID: toPgUUID(filter.UserID),
		Limit:  convert.IntToInt32Safe(filter.Limit),
		Offset: convert.IntToInt32Safe(filter.Offset),
	}

	if filter.RuleID != nil {
		params.Column2 = toPgUUID(*filter.RuleID)
	}
	if filter.Status != nil {
		params.Column3 = string(*filter.Status)
	}
	if filter.StartAfter != nil {
		params.Column4 = toPgTimestamp(*filter.StartAfter)
	}
	if filter.StartBefore != nil {
		params.Column5 = toPgTimestamp(*filter.StartBefore)
	}

	rows, err := r.queries.ListAutomationRuleExecutions(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	countParams := db.CountAutomationRuleExecutionsParams{
		UserID:  params.UserID,
		Column2: params.Column2,
		Column3: params.Column3,
		Column4: params.Column4,
		Column5: params.Column5,
	}
	total, err := r.queries.CountAutomationRuleExecutions(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}

	executions, err := r.toDomainExecutions(rows)
	if err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// CountByRuleIDSince counts executions for a rule since a given time.
func (r *ExecutionRepository) CountByRuleIDSince(ctx context.Context, ruleID uuid.UUID, since time.Time) (int64, error) {
	params := db.CountAutomationRuleExecutionsSinceParams{
		RuleID:    toPgUUID(ruleID),
		StartedAt: toPgTimestamp(since),
	}
	return r.queries.CountAutomationRuleExecutionsSince(ctx, params)
}

// GetLatestByRuleID gets the most recent execution for a rule.
func (r *ExecutionRepository) GetLatestByRuleID(ctx context.Context, ruleID uuid.UUID) (*domain.RuleExecution, error) {
	row, err := r.queries.GetLatestAutomationRuleExecution(ctx, toPgUUID(ruleID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No executions yet is not an error
		}
		return nil, err
	}
	return r.toDomainExecution(row)
}

// DeleteOlderThan deletes executions older than a given time.
func (r *ExecutionRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return r.queries.DeleteAutomationRuleExecutionsOlderThan(ctx, toPgTimestamp(before))
}

// Helper methods

func (r *ExecutionRepository) toDomainExecution(row db.AutomationRuleExecution) (*domain.RuleExecution, error) {
	var triggerPayload map[string]any
	if len(row.TriggerEventPayload) > 0 {
		if err := json.Unmarshal(row.TriggerEventPayload, &triggerPayload); err != nil {
			return nil, err
		}
	}

	var actionsExecuted []domain.ActionResult
	if len(row.ActionsExecuted) > 0 {
		if err := json.Unmarshal(row.ActionsExecuted, &actionsExecuted); err != nil {
			return nil, err
		}
	}

	var errorDetails map[string]any
	if len(row.ErrorDetails) > 0 {
		if err := json.Unmarshal(row.ErrorDetails, &errorDetails); err != nil {
			return nil, err
		}
	}

	execution := &domain.RuleExecution{
		ID:                  fromPgUUID(row.ID),
		RuleID:              fromPgUUID(row.RuleID),
		UserID:              fromPgUUID(row.UserID),
		TriggerEventType:    fromPgText(row.TriggerEventType),
		TriggerEventPayload: triggerPayload,
		Status:              domain.ExecutionStatus(row.Status),
		ActionsExecuted:     actionsExecuted,
		ErrorMessage:        fromPgText(row.ErrorMessage),
		ErrorDetails:        errorDetails,
		StartedAt:           fromPgTimestamp(row.StartedAt),
		SkipReason:          fromPgText(row.SkipReason),
	}

	if row.CompletedAt.Valid {
		completedAt := row.CompletedAt.Time
		execution.CompletedAt = &completedAt
	}
	if row.DurationMs.Valid {
		durationMs := int(row.DurationMs.Int32)
		execution.DurationMs = &durationMs
	}

	return execution, nil
}

func (r *ExecutionRepository) toDomainExecutions(rows []db.AutomationRuleExecution) ([]*domain.RuleExecution, error) {
	executions := make([]*domain.RuleExecution, 0, len(rows))
	for _, row := range rows {
		execution, err := r.toDomainExecution(row)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}
	return executions, nil
}
