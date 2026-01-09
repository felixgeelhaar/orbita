package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RuleFilter specifies criteria for filtering rules.
type RuleFilter struct {
	UserID      uuid.UUID
	Enabled     *bool
	TriggerType *TriggerType
	Tags        []string
	Limit       int
	Offset      int
}

// ExecutionFilter specifies criteria for filtering executions.
type ExecutionFilter struct {
	UserID     uuid.UUID
	RuleID     *uuid.UUID
	Status     *ExecutionStatus
	StartAfter *time.Time
	StartBefore *time.Time
	Limit      int
	Offset     int
}

// PendingActionFilter specifies criteria for filtering pending actions.
type PendingActionFilter struct {
	UserID         uuid.UUID
	RuleID         *uuid.UUID
	Status         *PendingActionStatus
	ScheduledBefore *time.Time
	Limit          int
	Offset         int
}

// RuleRepository defines the interface for automation rule persistence.
type RuleRepository interface {
	// Create creates a new automation rule.
	Create(ctx context.Context, rule *AutomationRule) error

	// Update updates an existing automation rule.
	Update(ctx context.Context, rule *AutomationRule) error

	// Delete deletes an automation rule by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves a rule by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*AutomationRule, error)

	// GetByUserID retrieves all rules for a user.
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*AutomationRule, error)

	// List retrieves rules matching the filter.
	List(ctx context.Context, filter RuleFilter) ([]*AutomationRule, int64, error)

	// GetEnabledByTriggerType retrieves enabled rules by trigger type.
	GetEnabledByTriggerType(ctx context.Context, userID uuid.UUID, triggerType TriggerType) ([]*AutomationRule, error)

	// GetEnabledByEventType retrieves enabled rules that trigger on a specific event type.
	GetEnabledByEventType(ctx context.Context, userID uuid.UUID, eventType string) ([]*AutomationRule, error)

	// CountByUserID counts rules for a user.
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// ExecutionRepository defines the interface for rule execution persistence.
type ExecutionRepository interface {
	// Create creates a new execution record.
	Create(ctx context.Context, execution *RuleExecution) error

	// Update updates an execution record.
	Update(ctx context.Context, execution *RuleExecution) error

	// GetByID retrieves an execution by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*RuleExecution, error)

	// GetByRuleID retrieves executions for a rule.
	GetByRuleID(ctx context.Context, ruleID uuid.UUID, limit int) ([]*RuleExecution, error)

	// List retrieves executions matching the filter.
	List(ctx context.Context, filter ExecutionFilter) ([]*RuleExecution, int64, error)

	// CountByRuleIDSince counts executions for a rule since a given time.
	CountByRuleIDSince(ctx context.Context, ruleID uuid.UUID, since time.Time) (int64, error)

	// GetLatestByRuleID gets the most recent execution for a rule.
	GetLatestByRuleID(ctx context.Context, ruleID uuid.UUID) (*RuleExecution, error)

	// DeleteOlderThan deletes executions older than a given time.
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// PendingActionRepository defines the interface for pending action persistence.
type PendingActionRepository interface {
	// Create creates a new pending action.
	Create(ctx context.Context, action *PendingAction) error

	// Update updates a pending action.
	Update(ctx context.Context, action *PendingAction) error

	// GetByID retrieves a pending action by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*PendingAction, error)

	// GetDue retrieves pending actions that are due for execution.
	GetDue(ctx context.Context, limit int) ([]*PendingAction, error)

	// GetByRuleID retrieves pending actions for a rule.
	GetByRuleID(ctx context.Context, ruleID uuid.UUID) ([]*PendingAction, error)

	// GetByExecutionID retrieves pending actions for an execution.
	GetByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*PendingAction, error)

	// List retrieves pending actions matching the filter.
	List(ctx context.Context, filter PendingActionFilter) ([]*PendingAction, int64, error)

	// CancelByRuleID cancels all pending actions for a rule.
	CancelByRuleID(ctx context.Context, ruleID uuid.UUID) error

	// DeleteExecuted deletes executed actions older than a given time.
	DeleteExecuted(ctx context.Context, before time.Time) (int64, error)
}
