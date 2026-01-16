-- Automation Rules

-- name: GetAutomationRuleByID :one
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE id = ?;

-- name: GetAutomationRulesByUserID :many
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE user_id = ?
ORDER BY priority DESC, created_at DESC;

-- name: GetEnabledAutomationRulesByUserID :many
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE user_id = ? AND enabled = 1
ORDER BY priority DESC, created_at DESC;

-- name: GetEnabledAutomationRulesByTriggerType :many
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE user_id = ? AND enabled = 1 AND trigger_type = ?
ORDER BY priority DESC, created_at DESC;

-- name: CountAutomationRulesByUserID :one
SELECT COUNT(*) FROM automation_rules WHERE user_id = ?;

-- name: CreateAutomationRule :exec
INSERT INTO automation_rules (
    id, user_id, name, description, enabled, priority,
    trigger_type, trigger_config, conditions, condition_operator,
    actions, cooldown_seconds, max_executions_per_hour, tags,
    created_at, updated_at, last_triggered_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateAutomationRule :exec
UPDATE automation_rules
SET name = ?,
    description = ?,
    enabled = ?,
    priority = ?,
    trigger_type = ?,
    trigger_config = ?,
    conditions = ?,
    condition_operator = ?,
    actions = ?,
    cooldown_seconds = ?,
    max_executions_per_hour = ?,
    tags = ?,
    updated_at = ?,
    last_triggered_at = ?
WHERE id = ?;

-- name: DeleteAutomationRule :exec
DELETE FROM automation_rules WHERE id = ?;

-- Automation Rule Executions

-- name: GetAutomationRuleExecutionByID :one
SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
       status, actions_executed, error_message, error_details,
       started_at, completed_at, duration_ms, skip_reason
FROM automation_rule_executions
WHERE id = ?;

-- name: GetAutomationRuleExecutionsByRuleID :many
SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
       status, actions_executed, error_message, error_details,
       started_at, completed_at, duration_ms, skip_reason
FROM automation_rule_executions
WHERE rule_id = ?
ORDER BY started_at DESC
LIMIT ?;

-- name: GetLatestAutomationRuleExecution :one
SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
       status, actions_executed, error_message, error_details,
       started_at, completed_at, duration_ms, skip_reason
FROM automation_rule_executions
WHERE rule_id = ?
ORDER BY started_at DESC
LIMIT 1;

-- name: CountAutomationRuleExecutionsSince :one
SELECT COUNT(*)
FROM automation_rule_executions
WHERE rule_id = ? AND started_at >= ?;

-- name: CreateAutomationRuleExecution :exec
INSERT INTO automation_rule_executions (
    id, rule_id, user_id, trigger_event_type, trigger_event_payload,
    status, actions_executed, error_message, error_details,
    started_at, completed_at, duration_ms, skip_reason
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateAutomationRuleExecution :exec
UPDATE automation_rule_executions
SET status = ?,
    actions_executed = ?,
    error_message = ?,
    error_details = ?,
    completed_at = ?,
    duration_ms = ?,
    skip_reason = ?
WHERE id = ?;

-- name: DeleteAutomationRuleExecutionsOlderThan :exec
DELETE FROM automation_rule_executions
WHERE completed_at < ?;

-- Automation Pending Actions

-- name: GetAutomationPendingActionByID :one
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE id = ?;

-- name: GetDueAutomationPendingActions :many
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE status = 'pending' AND scheduled_for <= datetime('now')
ORDER BY scheduled_for ASC
LIMIT ?;

-- name: GetAutomationPendingActionsByRuleID :many
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE rule_id = ?
ORDER BY scheduled_for DESC;

-- name: GetAutomationPendingActionsByExecutionID :many
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE execution_id = ?
ORDER BY scheduled_for ASC;

-- name: CreateAutomationPendingAction :exec
INSERT INTO automation_pending_actions (
    id, execution_id, rule_id, user_id, action_type, action_params,
    scheduled_for, status, executed_at, result, error_message,
    retry_count, max_retries, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateAutomationPendingAction :exec
UPDATE automation_pending_actions
SET status = ?,
    executed_at = ?,
    result = ?,
    error_message = ?,
    retry_count = ?
WHERE id = ?;

-- name: CancelAutomationPendingActionsByRuleID :exec
UPDATE automation_pending_actions
SET status = 'cancelled'
WHERE rule_id = ? AND status = 'pending';

-- name: DeleteExecutedAutomationPendingActions :exec
DELETE FROM automation_pending_actions
WHERE status IN ('executed', 'cancelled', 'failed')
  AND executed_at < ?;
