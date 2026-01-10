package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/felixgeelhaar/orbita/db/generated"
	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/convert"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PendingActionRepository implements domain.PendingActionRepository using PostgreSQL.
type PendingActionRepository struct {
	queries *db.Queries
}

// NewPendingActionRepository creates a new PostgreSQL pending action repository.
func NewPendingActionRepository(queries *db.Queries) *PendingActionRepository {
	return &PendingActionRepository{queries: queries}
}

// Create creates a new pending action.
func (r *PendingActionRepository) Create(ctx context.Context, action *domain.PendingAction) error {
	actionParams, err := json.Marshal(action.ActionParams)
	if err != nil {
		return err
	}
	result, err := json.Marshal(action.Result)
	if err != nil {
		return err
	}

	params := db.CreateAutomationPendingActionParams{
		ID:           toPgUUID(action.ID),
		ExecutionID:  toPgUUID(action.ExecutionID),
		RuleID:       toPgUUID(action.RuleID),
		UserID:       toPgUUID(action.UserID),
		ActionType:   action.ActionType,
		ActionParams: actionParams,
		ScheduledFor: toPgTimestamp(action.ScheduledFor),
		Status:       string(action.Status),
		ErrorMessage: toPgText(action.ErrorMessage),
		RetryCount:   convert.IntToInt32Safe(action.RetryCount),
		MaxRetries:   convert.IntToInt32Safe(action.MaxRetries),
		CreatedAt:    toPgTimestamp(action.CreatedAt),
		Result:       result,
	}

	if action.ExecutedAt != nil {
		params.ExecutedAt = toPgTimestamp(*action.ExecutedAt)
	}

	return r.queries.CreateAutomationPendingAction(ctx, params)
}

// Update updates a pending action.
func (r *PendingActionRepository) Update(ctx context.Context, action *domain.PendingAction) error {
	result, err := json.Marshal(action.Result)
	if err != nil {
		return err
	}

	params := db.UpdateAutomationPendingActionParams{
		ID:           toPgUUID(action.ID),
		Status:       string(action.Status),
		ErrorMessage: toPgText(action.ErrorMessage),
		RetryCount:   convert.IntToInt32Safe(action.RetryCount),
		Result:       result,
	}

	if action.ExecutedAt != nil {
		params.ExecutedAt = toPgTimestamp(*action.ExecutedAt)
	}

	return r.queries.UpdateAutomationPendingAction(ctx, params)
}

// GetByID retrieves a pending action by ID.
func (r *PendingActionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PendingAction, error) {
	row, err := r.queries.GetAutomationPendingActionByID(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomainPendingAction(row)
}

// GetDue retrieves pending actions that are due for execution.
func (r *PendingActionRepository) GetDue(ctx context.Context, limit int) ([]*domain.PendingAction, error) {
	rows, err := r.queries.GetDueAutomationPendingActions(ctx, convert.IntToInt32Safe(limit))
	if err != nil {
		return nil, err
	}
	return r.toDomainPendingActions(rows)
}

// GetByRuleID retrieves pending actions for a rule.
func (r *PendingActionRepository) GetByRuleID(ctx context.Context, ruleID uuid.UUID) ([]*domain.PendingAction, error) {
	rows, err := r.queries.GetAutomationPendingActionsByRuleID(ctx, toPgUUID(ruleID))
	if err != nil {
		return nil, err
	}
	return r.toDomainPendingActions(rows)
}

// GetByExecutionID retrieves pending actions for an execution.
func (r *PendingActionRepository) GetByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*domain.PendingAction, error) {
	rows, err := r.queries.GetAutomationPendingActionsByExecutionID(ctx, toPgUUID(executionID))
	if err != nil {
		return nil, err
	}
	return r.toDomainPendingActions(rows)
}

// List retrieves pending actions matching the filter.
func (r *PendingActionRepository) List(ctx context.Context, filter domain.PendingActionFilter) ([]*domain.PendingAction, int64, error) {
	params := db.ListAutomationPendingActionsParams{
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
	if filter.ScheduledBefore != nil {
		params.Column4 = toPgTimestamp(*filter.ScheduledBefore)
	}

	rows, err := r.queries.ListAutomationPendingActions(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	countParams := db.CountAutomationPendingActionsParams{
		UserID:  params.UserID,
		Column2: params.Column2,
		Column3: params.Column3,
		Column4: params.Column4,
	}
	total, err := r.queries.CountAutomationPendingActions(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}

	actions, err := r.toDomainPendingActions(rows)
	if err != nil {
		return nil, 0, err
	}

	return actions, total, nil
}

// CancelByRuleID cancels all pending actions for a rule.
func (r *PendingActionRepository) CancelByRuleID(ctx context.Context, ruleID uuid.UUID) error {
	return r.queries.CancelAutomationPendingActionsByRuleID(ctx, toPgUUID(ruleID))
}

// DeleteExecuted deletes executed actions older than a given time.
func (r *PendingActionRepository) DeleteExecuted(ctx context.Context, before time.Time) (int64, error) {
	return r.queries.DeleteExecutedAutomationPendingActions(ctx, toPgTimestamp(before))
}

// Helper methods

func (r *PendingActionRepository) toDomainPendingAction(row db.AutomationPendingAction) (*domain.PendingAction, error) {
	var actionParams map[string]any
	if len(row.ActionParams) > 0 {
		if err := json.Unmarshal(row.ActionParams, &actionParams); err != nil {
			return nil, err
		}
	}

	var result map[string]any
	if len(row.Result) > 0 {
		if err := json.Unmarshal(row.Result, &result); err != nil {
			return nil, err
		}
	}

	action := &domain.PendingAction{
		ID:           fromPgUUID(row.ID),
		ExecutionID:  fromPgUUID(row.ExecutionID),
		RuleID:       fromPgUUID(row.RuleID),
		UserID:       fromPgUUID(row.UserID),
		ActionType:   row.ActionType,
		ActionParams: actionParams,
		ScheduledFor: fromPgTimestamp(row.ScheduledFor),
		Status:       domain.PendingActionStatus(row.Status),
		Result:       result,
		ErrorMessage: fromPgText(row.ErrorMessage),
		RetryCount:   int(row.RetryCount),
		MaxRetries:   int(row.MaxRetries),
		CreatedAt:    fromPgTimestamp(row.CreatedAt),
	}

	if row.ExecutedAt.Valid {
		executedAt := row.ExecutedAt.Time
		action.ExecutedAt = &executedAt
	}

	return action, nil
}

func (r *PendingActionRepository) toDomainPendingActions(rows []db.AutomationPendingAction) ([]*domain.PendingAction, error) {
	actions := make([]*domain.PendingAction, 0, len(rows))
	for _, row := range rows {
		action, err := r.toDomainPendingAction(row)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}
