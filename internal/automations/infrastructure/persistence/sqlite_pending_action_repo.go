package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// SQLitePendingActionRepository implements domain.PendingActionRepository using SQLite.
type SQLitePendingActionRepository struct {
	db *sql.DB
}

// NewSQLitePendingActionRepository creates a new SQLite pending action repository.
func NewSQLitePendingActionRepository(db *sql.DB) *SQLitePendingActionRepository {
	return &SQLitePendingActionRepository{db: db}
}

// Create creates a new pending action.
func (r *SQLitePendingActionRepository) Create(ctx context.Context, action *domain.PendingAction) error {
	actionParams, err := json.Marshal(action.ActionParams)
	if err != nil {
		return err
	}
	result, err := json.Marshal(action.Result)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO automation_pending_actions (
			id, execution_id, rule_id, user_id, action_type, action_params,
			scheduled_for, status, executed_at, result, error_message,
			retry_count, max_retries, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var executedAt sql.NullString
	if action.ExecutedAt != nil {
		executedAt = sql.NullString{String: action.ExecutedAt.Format(time.RFC3339), Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		action.ID.String(),
		action.ExecutionID.String(),
		action.RuleID.String(),
		action.UserID.String(),
		action.ActionType,
		string(actionParams),
		action.ScheduledFor.Format(time.RFC3339),
		string(action.Status),
		executedAt,
		string(result),
		action.ErrorMessage,
		action.RetryCount,
		action.MaxRetries,
		action.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// Update updates a pending action.
func (r *SQLitePendingActionRepository) Update(ctx context.Context, action *domain.PendingAction) error {
	result, err := json.Marshal(action.Result)
	if err != nil {
		return err
	}

	query := `
		UPDATE automation_pending_actions SET
			status = ?, executed_at = ?, result = ?, error_message = ?, retry_count = ?
		WHERE id = ?
	`

	var executedAt sql.NullString
	if action.ExecutedAt != nil {
		executedAt = sql.NullString{String: action.ExecutedAt.Format(time.RFC3339), Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		string(action.Status),
		executedAt,
		string(result),
		action.ErrorMessage,
		action.RetryCount,
		action.ID.String(),
	)
	return err
}

// GetByID retrieves a pending action by ID.
func (r *SQLitePendingActionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PendingAction, error) {
	query := `
		SELECT id, execution_id, rule_id, user_id, action_type, action_params,
			scheduled_for, status, executed_at, result, error_message,
			retry_count, max_retries, created_at
		FROM automation_pending_actions
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanPendingAction(row)
}

// GetDue retrieves pending actions that are due for execution.
func (r *SQLitePendingActionRepository) GetDue(ctx context.Context, limit int) ([]*domain.PendingAction, error) {
	query := `
		SELECT id, execution_id, rule_id, user_id, action_type, action_params,
			scheduled_for, status, executed_at, result, error_message,
			retry_count, max_retries, created_at
		FROM automation_pending_actions
		WHERE status = 'pending' AND scheduled_for <= ?
		ORDER BY scheduled_for ASC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, time.Now().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanPendingActions(rows)
}

// GetByRuleID retrieves pending actions for a rule.
func (r *SQLitePendingActionRepository) GetByRuleID(ctx context.Context, ruleID uuid.UUID) ([]*domain.PendingAction, error) {
	query := `
		SELECT id, execution_id, rule_id, user_id, action_type, action_params,
			scheduled_for, status, executed_at, result, error_message,
			retry_count, max_retries, created_at
		FROM automation_pending_actions
		WHERE rule_id = ?
		ORDER BY scheduled_for ASC
	`
	rows, err := r.db.QueryContext(ctx, query, ruleID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanPendingActions(rows)
}

// GetByExecutionID retrieves pending actions for an execution.
func (r *SQLitePendingActionRepository) GetByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*domain.PendingAction, error) {
	query := `
		SELECT id, execution_id, rule_id, user_id, action_type, action_params,
			scheduled_for, status, executed_at, result, error_message,
			retry_count, max_retries, created_at
		FROM automation_pending_actions
		WHERE execution_id = ?
		ORDER BY scheduled_for ASC
	`
	rows, err := r.db.QueryContext(ctx, query, executionID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanPendingActions(rows)
}

// List retrieves pending actions matching the filter.
func (r *SQLitePendingActionRepository) List(ctx context.Context, filter domain.PendingActionFilter) ([]*domain.PendingAction, int64, error) {
	query := `
		SELECT id, execution_id, rule_id, user_id, action_type, action_params,
			scheduled_for, status, executed_at, result, error_message,
			retry_count, max_retries, created_at
		FROM automation_pending_actions
		WHERE user_id = ?
	`
	countQuery := `SELECT COUNT(*) FROM automation_pending_actions WHERE user_id = ?`
	args := []any{filter.UserID.String()}
	countArgs := []any{filter.UserID.String()}

	if filter.RuleID != nil {
		query += " AND rule_id = ?"
		countQuery += " AND rule_id = ?"
		args = append(args, filter.RuleID.String())
		countArgs = append(countArgs, filter.RuleID.String())
	}
	if filter.Status != nil {
		query += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, string(*filter.Status))
		countArgs = append(countArgs, string(*filter.Status))
	}
	if filter.ScheduledBefore != nil {
		query += " AND scheduled_for <= ?"
		countQuery += " AND scheduled_for <= ?"
		args = append(args, filter.ScheduledBefore.Format(time.RFC3339))
		countArgs = append(countArgs, filter.ScheduledBefore.Format(time.RFC3339))
	}

	query += " ORDER BY scheduled_for ASC"

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

	actions, err := r.scanPendingActions(rows)
	if err != nil {
		return nil, 0, err
	}
	return actions, total, nil
}

// CancelByRuleID cancels all pending actions for a rule.
func (r *SQLitePendingActionRepository) CancelByRuleID(ctx context.Context, ruleID uuid.UUID) error {
	query := `UPDATE automation_pending_actions SET status = 'cancelled' WHERE rule_id = ? AND status = 'pending'`
	_, err := r.db.ExecContext(ctx, query, ruleID.String())
	return err
}

// DeleteExecuted deletes executed actions older than a given time.
func (r *SQLitePendingActionRepository) DeleteExecuted(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM automation_pending_actions WHERE status = 'executed' AND executed_at < ?`
	result, err := r.db.ExecContext(ctx, query, before.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Helper methods

func (r *SQLitePendingActionRepository) scanPendingAction(row *sql.Row) (*domain.PendingAction, error) {
	var action domain.PendingAction
	var idStr, executionIDStr, ruleIDStr, userIDStr string
	var actionParamsStr, resultStr string
	var scheduledForStr, createdAtStr string
	var executedAtStr sql.NullString
	var errorMessage sql.NullString

	err := row.Scan(
		&idStr,
		&executionIDStr,
		&ruleIDStr,
		&userIDStr,
		&action.ActionType,
		&actionParamsStr,
		&scheduledForStr,
		&action.Status,
		&executedAtStr,
		&resultStr,
		&errorMessage,
		&action.RetryCount,
		&action.MaxRetries,
		&createdAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	action.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	action.ExecutionID, err = uuid.Parse(executionIDStr)
	if err != nil {
		return nil, err
	}
	action.RuleID, err = uuid.Parse(ruleIDStr)
	if err != nil {
		return nil, err
	}
	action.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	action.ErrorMessage = errorMessage.String

	if actionParamsStr != "" && actionParamsStr != "{}" {
		if err := json.Unmarshal([]byte(actionParamsStr), &action.ActionParams); err != nil {
			return nil, err
		}
	}

	if resultStr != "" && resultStr != "{}" && resultStr != "null" {
		if err := json.Unmarshal([]byte(resultStr), &action.Result); err != nil {
			return nil, err
		}
	}

	action.ScheduledFor, err = time.Parse(time.RFC3339, scheduledForStr)
	if err != nil {
		return nil, err
	}

	action.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	if executedAtStr.Valid {
		executedAt, err := time.Parse(time.RFC3339, executedAtStr.String)
		if err == nil {
			action.ExecutedAt = &executedAt
		}
	}

	return &action, nil
}

func (r *SQLitePendingActionRepository) scanPendingActions(rows *sql.Rows) ([]*domain.PendingAction, error) {
	var actions []*domain.PendingAction
	for rows.Next() {
		var action domain.PendingAction
		var idStr, executionIDStr, ruleIDStr, userIDStr string
		var actionParamsStr, resultStr string
		var scheduledForStr, createdAtStr string
		var executedAtStr sql.NullString
		var errorMessage sql.NullString

		err := rows.Scan(
			&idStr,
			&executionIDStr,
			&ruleIDStr,
			&userIDStr,
			&action.ActionType,
			&actionParamsStr,
			&scheduledForStr,
			&action.Status,
			&executedAtStr,
			&resultStr,
			&errorMessage,
			&action.RetryCount,
			&action.MaxRetries,
			&createdAtStr,
		)
		if err != nil {
			return nil, err
		}

		action.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		action.ExecutionID, err = uuid.Parse(executionIDStr)
		if err != nil {
			return nil, err
		}
		action.RuleID, err = uuid.Parse(ruleIDStr)
		if err != nil {
			return nil, err
		}
		action.UserID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, err
		}

		action.ErrorMessage = errorMessage.String

		if actionParamsStr != "" && actionParamsStr != "{}" {
			if err := json.Unmarshal([]byte(actionParamsStr), &action.ActionParams); err != nil {
				return nil, err
			}
		}

		if resultStr != "" && resultStr != "{}" && resultStr != "null" {
			if err := json.Unmarshal([]byte(resultStr), &action.Result); err != nil {
				return nil, err
			}
		}

		action.ScheduledFor, err = time.Parse(time.RFC3339, scheduledForStr)
		if err != nil {
			return nil, err
		}

		action.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, err
		}

		if executedAtStr.Valid {
			executedAt, err := time.Parse(time.RFC3339, executedAtStr.String)
			if err == nil {
				action.ExecutedAt = &executedAt
			}
		}

		actions = append(actions, &action)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return actions, nil
}
