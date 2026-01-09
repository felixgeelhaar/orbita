-- Automation Rules

-- name: GetAutomationRuleByID :one
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE id = $1;

-- name: GetAutomationRulesByUserID :many
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE user_id = $1
ORDER BY priority DESC, created_at DESC;

-- name: GetEnabledAutomationRulesByUserID :many
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE user_id = $1 AND enabled = TRUE
ORDER BY priority DESC, created_at DESC;

-- name: GetEnabledAutomationRulesByTriggerType :many
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE user_id = $1 AND enabled = TRUE AND trigger_type = $2
ORDER BY priority DESC, created_at DESC;

-- name: ListAutomationRules :many
SELECT id, user_id, name, description, enabled, priority,
       trigger_type, trigger_config, conditions, condition_operator,
       actions, cooldown_seconds, max_executions_per_hour, tags,
       created_at, updated_at, last_triggered_at
FROM automation_rules
WHERE user_id = $1
  AND ($2::boolean IS NULL OR enabled = $2)
  AND ($3::varchar IS NULL OR trigger_type = $3)
  AND ($4::text[] IS NULL OR tags && $4)
ORDER BY priority DESC, created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountAutomationRules :one
SELECT COUNT(*)
FROM automation_rules
WHERE user_id = $1
  AND ($2::boolean IS NULL OR enabled = $2)
  AND ($3::varchar IS NULL OR trigger_type = $3)
  AND ($4::text[] IS NULL OR tags && $4);

-- name: CountAutomationRulesByUserID :one
SELECT COUNT(*) FROM automation_rules WHERE user_id = $1;

-- name: CreateAutomationRule :exec
INSERT INTO automation_rules (
    id, user_id, name, description, enabled, priority,
    trigger_type, trigger_config, conditions, condition_operator,
    actions, cooldown_seconds, max_executions_per_hour, tags,
    created_at, updated_at, last_triggered_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
);

-- name: UpdateAutomationRule :exec
UPDATE automation_rules
SET name = $2,
    description = $3,
    enabled = $4,
    priority = $5,
    trigger_type = $6,
    trigger_config = $7,
    conditions = $8,
    condition_operator = $9,
    actions = $10,
    cooldown_seconds = $11,
    max_executions_per_hour = $12,
    tags = $13,
    updated_at = $14,
    last_triggered_at = $15
WHERE id = $1;

-- name: DeleteAutomationRule :exec
DELETE FROM automation_rules WHERE id = $1;

-- Automation Rule Executions

-- name: GetAutomationRuleExecutionByID :one
SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
       status, actions_executed, error_message, error_details,
       started_at, completed_at, duration_ms, skip_reason
FROM automation_rule_executions
WHERE id = $1;

-- name: GetAutomationRuleExecutionsByRuleID :many
SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
       status, actions_executed, error_message, error_details,
       started_at, completed_at, duration_ms, skip_reason
FROM automation_rule_executions
WHERE rule_id = $1
ORDER BY started_at DESC
LIMIT $2;

-- name: GetLatestAutomationRuleExecution :one
SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
       status, actions_executed, error_message, error_details,
       started_at, completed_at, duration_ms, skip_reason
FROM automation_rule_executions
WHERE rule_id = $1
ORDER BY started_at DESC
LIMIT 1;

-- name: ListAutomationRuleExecutions :many
SELECT id, rule_id, user_id, trigger_event_type, trigger_event_payload,
       status, actions_executed, error_message, error_details,
       started_at, completed_at, duration_ms, skip_reason
FROM automation_rule_executions
WHERE user_id = $1
  AND ($2::uuid IS NULL OR rule_id = $2)
  AND ($3::varchar IS NULL OR status = $3)
  AND ($4::timestamptz IS NULL OR started_at >= $4)
  AND ($5::timestamptz IS NULL OR started_at <= $5)
ORDER BY started_at DESC
LIMIT $6 OFFSET $7;

-- name: CountAutomationRuleExecutions :one
SELECT COUNT(*)
FROM automation_rule_executions
WHERE user_id = $1
  AND ($2::uuid IS NULL OR rule_id = $2)
  AND ($3::varchar IS NULL OR status = $3)
  AND ($4::timestamptz IS NULL OR started_at >= $4)
  AND ($5::timestamptz IS NULL OR started_at <= $5);

-- name: CountAutomationRuleExecutionsSince :one
SELECT COUNT(*)
FROM automation_rule_executions
WHERE rule_id = $1 AND started_at >= $2;

-- name: CreateAutomationRuleExecution :exec
INSERT INTO automation_rule_executions (
    id, rule_id, user_id, trigger_event_type, trigger_event_payload,
    status, actions_executed, error_message, error_details,
    started_at, completed_at, duration_ms, skip_reason
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: UpdateAutomationRuleExecution :exec
UPDATE automation_rule_executions
SET status = $2,
    actions_executed = $3,
    error_message = $4,
    error_details = $5,
    completed_at = $6,
    duration_ms = $7,
    skip_reason = $8
WHERE id = $1;

-- name: DeleteAutomationRuleExecutionsOlderThan :execrows
DELETE FROM automation_rule_executions
WHERE completed_at < $1;

-- Automation Pending Actions

-- name: GetAutomationPendingActionByID :one
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE id = $1;

-- name: GetDueAutomationPendingActions :many
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE status = 'pending' AND scheduled_for <= NOW()
ORDER BY scheduled_for ASC
LIMIT $1;

-- name: GetAutomationPendingActionsByRuleID :many
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE rule_id = $1
ORDER BY scheduled_for DESC;

-- name: GetAutomationPendingActionsByExecutionID :many
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE execution_id = $1
ORDER BY scheduled_for ASC;

-- name: ListAutomationPendingActions :many
SELECT id, execution_id, rule_id, user_id, action_type, action_params,
       scheduled_for, status, executed_at, result, error_message,
       retry_count, max_retries, created_at
FROM automation_pending_actions
WHERE user_id = $1
  AND ($2::uuid IS NULL OR rule_id = $2)
  AND ($3::varchar IS NULL OR status = $3)
  AND ($4::timestamptz IS NULL OR scheduled_for <= $4)
ORDER BY scheduled_for DESC
LIMIT $5 OFFSET $6;

-- name: CountAutomationPendingActions :one
SELECT COUNT(*)
FROM automation_pending_actions
WHERE user_id = $1
  AND ($2::uuid IS NULL OR rule_id = $2)
  AND ($3::varchar IS NULL OR status = $3)
  AND ($4::timestamptz IS NULL OR scheduled_for <= $4);

-- name: CreateAutomationPendingAction :exec
INSERT INTO automation_pending_actions (
    id, execution_id, rule_id, user_id, action_type, action_params,
    scheduled_for, status, executed_at, result, error_message,
    retry_count, max_retries, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
);

-- name: UpdateAutomationPendingAction :exec
UPDATE automation_pending_actions
SET status = $2,
    executed_at = $3,
    result = $4,
    error_message = $5,
    retry_count = $6
WHERE id = $1;

-- name: CancelAutomationPendingActionsByRuleID :exec
UPDATE automation_pending_actions
SET status = 'cancelled'
WHERE rule_id = $1 AND status = 'pending';

-- name: DeleteExecutedAutomationPendingActions :execrows
DELETE FROM automation_pending_actions
WHERE status IN ('executed', 'cancelled', 'failed')
  AND executed_at < $1;
