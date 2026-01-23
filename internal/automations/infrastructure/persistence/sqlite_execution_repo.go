package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// SQLiteExecutionRepository implements domain.ExecutionRepository using SQLite.
type SQLiteExecutionRepository struct {
	db *sql.DB
}

// NewSQLiteExecutionRepository creates a new SQLite execution repository.
func NewSQLiteExecutionRepository(db *sql.DB) *SQLiteExecutionRepository {
	return &SQLiteExecutionRepository{db: db}
}

// Create creates a new execution record.
func (r *SQLiteExecutionRepository) Create(ctx context.Context, execution *domain.RuleExecution) error {
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

	query := `
		INSERT INTO automation_rule_executions (
			id, rule_id, user_id, trigger_event_type, trigger_event_payload,
			status, actions_executed, error_message, error_details,
			started_at, completed_at, duration_ms, skip_reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var completedAt sql.NullString
	if execution.CompletedAt != nil {
		completedAt = sql.NullString{String: execution.CompletedAt.Format(time.RFC3339), Valid: true}
	}

	var durationMs sql.NullInt32
	if execution.DurationMs != nil {
		durationMs = sql.NullInt32{Int32: int32(*execution.DurationMs), Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		execution.ID.String(),
		execution.RuleID.String(),
		execution.UserID.String(),
		execution.TriggerEventType,
		string(triggerPayload),
		string(execution.Status),
		string(actionsExecuted),
		execution.ErrorMessage,
		string(errorDetails),
		execution.StartedAt.Format(time.RFC3339),
		completedAt,
		durationMs,
		execution.SkipReason,
	)
	return err
}

// Update updates an execution record.
func (r *SQLiteExecutionRepository) Update(ctx context.Context, execution *domain.RuleExecution) error {
	actionsExecuted, err := json.Marshal(execution.ActionsExecuted)
	if err != nil {
		return err
	}
	errorDetails, err := json.Marshal(execution.ErrorDetails)
	if err != nil {
		return err
	}

	query := `
		UPDATE automation_rule_executions SET
			status = ?, actions_executed = ?, error_message = ?, error_details = ?,
			completed_at = ?, duration_ms = ?, skip_reason = ?
		WHERE id = ?
	`

	var completedAt sql.NullString
	if execution.CompletedAt != nil {
		completedAt = sql.NullString{String: execution.CompletedAt.Format(time.RFC3339), Valid: true}
	}

	var durationMs sql.NullInt32
	if execution.DurationMs != nil {
		durationMs = sql.NullInt32{Int32: int32(*execution.DurationMs), Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		string(execution.Status),
		string(actionsExecuted),
		execution.ErrorMessage,
		string(errorDetails),
		completedAt,
		durationMs,
		execution.SkipReason,
		execution.ID.String(),
	)
	return err
}

// GetByID retrieves an execution by ID.
func (r *SQLiteExecutionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RuleExecution, error) {
	query := `
		SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
			status, actions_executed, error_message, error_details,
			started_at, completed_at, duration_ms, skip_reason
		FROM automation_rule_executions
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanExecution(row)
}

// GetByRuleID retrieves executions for a rule.
func (r *SQLiteExecutionRepository) GetByRuleID(ctx context.Context, ruleID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	query := `
		SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
			status, actions_executed, error_message, error_details,
			started_at, completed_at, duration_ms, skip_reason
		FROM automation_rule_executions
		WHERE rule_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, ruleID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanExecutions(rows)
}

// List retrieves executions matching the filter.
func (r *SQLiteExecutionRepository) List(ctx context.Context, filter domain.ExecutionFilter) ([]*domain.RuleExecution, int64, error) {
	query := `
		SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
			status, actions_executed, error_message, error_details,
			started_at, completed_at, duration_ms, skip_reason
		FROM automation_rule_executions
		WHERE user_id = ?
	`
	countQuery := `SELECT COUNT(*) FROM automation_rule_executions WHERE user_id = ?`
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
	if filter.StartAfter != nil {
		query += " AND started_at >= ?"
		countQuery += " AND started_at >= ?"
		args = append(args, filter.StartAfter.Format(time.RFC3339))
		countArgs = append(countArgs, filter.StartAfter.Format(time.RFC3339))
	}
	if filter.StartBefore != nil {
		query += " AND started_at <= ?"
		countQuery += " AND started_at <= ?"
		args = append(args, filter.StartBefore.Format(time.RFC3339))
		countArgs = append(countArgs, filter.StartBefore.Format(time.RFC3339))
	}

	query += " ORDER BY started_at DESC"

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

	executions, err := r.scanExecutions(rows)
	if err != nil {
		return nil, 0, err
	}
	return executions, total, nil
}

// CountByRuleIDSince counts executions for a rule since a given time.
func (r *SQLiteExecutionRepository) CountByRuleIDSince(ctx context.Context, ruleID uuid.UUID, since time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM automation_rule_executions WHERE rule_id = ? AND started_at >= ?`
	var count int64
	err := r.db.QueryRowContext(ctx, query, ruleID.String(), since.Format(time.RFC3339)).Scan(&count)
	return count, err
}

// GetLatestByRuleID gets the most recent execution for a rule.
func (r *SQLiteExecutionRepository) GetLatestByRuleID(ctx context.Context, ruleID uuid.UUID) (*domain.RuleExecution, error) {
	query := `
		SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
			status, actions_executed, error_message, error_details,
			started_at, completed_at, duration_ms, skip_reason
		FROM automation_rule_executions
		WHERE rule_id = ?
		ORDER BY started_at DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, ruleID.String())
	execution, err := r.scanExecution(row)
	if err != nil {
		if errors.Is(err, domain.ErrExecutionNotFound) {
			return nil, nil // No executions yet is not an error
		}
		return nil, err
	}
	return execution, nil
}

// DeleteOlderThan deletes executions older than a given time.
func (r *SQLiteExecutionRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM automation_rule_executions WHERE started_at < ?`
	result, err := r.db.ExecContext(ctx, query, before.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Helper methods

func (r *SQLiteExecutionRepository) scanExecution(row *sql.Row) (*domain.RuleExecution, error) {
	var execution domain.RuleExecution
	var idStr, ruleIDStr, userIDStr string
	var triggerEventType sql.NullString
	var triggerPayloadStr, actionsExecutedStr string
	var errorMessage sql.NullString
	var errorDetailsStr string
	var startedAtStr string
	var completedAtStr sql.NullString
	var durationMs sql.NullInt32
	var skipReason sql.NullString

	err := row.Scan(
		&idStr,
		&ruleIDStr,
		&userIDStr,
		&triggerEventType,
		&triggerPayloadStr,
		&execution.Status,
		&actionsExecutedStr,
		&errorMessage,
		&errorDetailsStr,
		&startedAtStr,
		&completedAtStr,
		&durationMs,
		&skipReason,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrExecutionNotFound
		}
		return nil, err
	}

	execution.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	execution.RuleID, err = uuid.Parse(ruleIDStr)
	if err != nil {
		return nil, err
	}
	execution.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	execution.TriggerEventType = triggerEventType.String
	execution.ErrorMessage = errorMessage.String
	execution.SkipReason = skipReason.String

	if triggerPayloadStr != "" && triggerPayloadStr != "{}" {
		if err := json.Unmarshal([]byte(triggerPayloadStr), &execution.TriggerEventPayload); err != nil {
			return nil, err
		}
	}

	if actionsExecutedStr != "" && actionsExecutedStr != "[]" {
		if err := json.Unmarshal([]byte(actionsExecutedStr), &execution.ActionsExecuted); err != nil {
			return nil, err
		}
	}

	if errorDetailsStr != "" && errorDetailsStr != "{}" {
		if err := json.Unmarshal([]byte(errorDetailsStr), &execution.ErrorDetails); err != nil {
			return nil, err
		}
	}

	execution.StartedAt, err = time.Parse(time.RFC3339, startedAtStr)
	if err != nil {
		return nil, err
	}

	if completedAtStr.Valid {
		completedAt, err := time.Parse(time.RFC3339, completedAtStr.String)
		if err == nil {
			execution.CompletedAt = &completedAt
		}
	}

	if durationMs.Valid {
		dur := int(durationMs.Int32)
		execution.DurationMs = &dur
	}

	return &execution, nil
}

func (r *SQLiteExecutionRepository) scanExecutions(rows *sql.Rows) ([]*domain.RuleExecution, error) {
	var executions []*domain.RuleExecution
	for rows.Next() {
		var execution domain.RuleExecution
		var idStr, ruleIDStr, userIDStr string
		var triggerEventType sql.NullString
		var triggerPayloadStr, actionsExecutedStr string
		var errorMessage sql.NullString
		var errorDetailsStr string
		var startedAtStr string
		var completedAtStr sql.NullString
		var durationMs sql.NullInt32
		var skipReason sql.NullString

		err := rows.Scan(
			&idStr,
			&ruleIDStr,
			&userIDStr,
			&triggerEventType,
			&triggerPayloadStr,
			&execution.Status,
			&actionsExecutedStr,
			&errorMessage,
			&errorDetailsStr,
			&startedAtStr,
			&completedAtStr,
			&durationMs,
			&skipReason,
		)
		if err != nil {
			return nil, err
		}

		execution.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		execution.RuleID, err = uuid.Parse(ruleIDStr)
		if err != nil {
			return nil, err
		}
		execution.UserID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, err
		}

		execution.TriggerEventType = triggerEventType.String
		execution.ErrorMessage = errorMessage.String
		execution.SkipReason = skipReason.String

		if triggerPayloadStr != "" && triggerPayloadStr != "{}" {
			if err := json.Unmarshal([]byte(triggerPayloadStr), &execution.TriggerEventPayload); err != nil {
				return nil, err
			}
		}

		if actionsExecutedStr != "" && actionsExecutedStr != "[]" {
			if err := json.Unmarshal([]byte(actionsExecutedStr), &execution.ActionsExecuted); err != nil {
				return nil, err
			}
		}

		if errorDetailsStr != "" && errorDetailsStr != "{}" {
			if err := json.Unmarshal([]byte(errorDetailsStr), &execution.ErrorDetails); err != nil {
				return nil, err
			}
		}

		execution.StartedAt, err = time.Parse(time.RFC3339, startedAtStr)
		if err != nil {
			return nil, err
		}

		if completedAtStr.Valid {
			completedAt, err := time.Parse(time.RFC3339, completedAtStr.String)
			if err == nil {
				execution.CompletedAt = &completedAt
			}
		}

		if durationMs.Valid {
			dur := int(durationMs.Int32)
			execution.DurationMs = &dur
		}

		executions = append(executions, &execution)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return executions, nil
}
