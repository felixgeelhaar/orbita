package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
)

// ActionHandler handles execution of a specific action type.
type ActionHandler interface {
	// ActionType returns the action type this handler supports.
	ActionType() string

	// Execute executes the action with the given parameters.
	Execute(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error)
}

// ActionExecutor executes pending automation actions.
type ActionExecutor struct {
	pendingRepo domain.PendingActionRepository
	handlers    map[string]ActionHandler
	logger      *slog.Logger
}

// NewActionExecutor creates a new action executor.
func NewActionExecutor(
	pendingRepo domain.PendingActionRepository,
	logger *slog.Logger,
) *ActionExecutor {
	return &ActionExecutor{
		pendingRepo: pendingRepo,
		handlers:    make(map[string]ActionHandler),
		logger:      logger,
	}
}

// RegisterHandler registers an action handler.
func (e *ActionExecutor) RegisterHandler(handler ActionHandler) {
	e.handlers[handler.ActionType()] = handler
}

// ExecutionResult contains the results of executing pending actions.
type ExecutionResult struct {
	TotalProcessed int
	SuccessCount   int
	FailedCount    int
	RetryCount     int
	Results        []ActionExecutionResult
}

// ActionExecutionResult contains the result of a single action execution.
type ActionExecutionResult struct {
	ActionID   uuid.UUID
	ActionType string
	Status     string
	Result     map[string]any
	Error      string
	Duration   time.Duration
}

// ExecutePending executes all pending actions that are due.
func (e *ActionExecutor) ExecutePending(ctx context.Context, limit int) (*ExecutionResult, error) {
	// Get pending actions that are due
	actions, err := e.pendingRepo.GetDue(ctx, limit)
	if err != nil {
		return nil, err
	}

	result := &ExecutionResult{
		TotalProcessed: len(actions),
		Results:        make([]ActionExecutionResult, 0, len(actions)),
	}

	for _, action := range actions {
		execResult := e.executeAction(ctx, action)
		result.Results = append(result.Results, execResult)

		switch execResult.Status {
		case "success":
			result.SuccessCount++
		case "failed":
			result.FailedCount++
		case "retry":
			result.RetryCount++
		}
	}

	return result, nil
}

// ExecuteAction executes a single pending action by ID.
func (e *ActionExecutor) ExecuteAction(ctx context.Context, actionID uuid.UUID) (*ActionExecutionResult, error) {
	action, err := e.pendingRepo.GetByID(ctx, actionID)
	if err != nil {
		return nil, err
	}
	if action == nil {
		return nil, fmt.Errorf("pending action not found: %s", actionID)
	}

	result := e.executeAction(ctx, action)
	return &result, nil
}

func (e *ActionExecutor) executeAction(ctx context.Context, action *domain.PendingAction) ActionExecutionResult {
	startTime := time.Now()

	result := ActionExecutionResult{
		ActionID:   action.ID,
		ActionType: action.ActionType,
	}

	// Find handler for this action type
	handler, ok := e.handlers[action.ActionType]
	if !ok {
		e.logger.Warn("no handler for action type", "action_type", action.ActionType)
		action.Fail(fmt.Sprintf("no handler for action type: %s", action.ActionType))
		if err := e.pendingRepo.Update(ctx, action); err != nil {
			e.logger.Error("failed to update action", "action_id", action.ID, "error", err)
		}
		result.Status = "failed"
		result.Error = fmt.Sprintf("no handler for action type: %s", action.ActionType)
		result.Duration = time.Since(startTime)
		return result
	}

	// Execute the action
	actionResult, err := handler.Execute(ctx, action.UserID, "", action.ActionParams)
	result.Duration = time.Since(startTime)

	if err != nil {
		e.logger.Error("action execution failed",
			"action_id", action.ID,
			"action_type", action.ActionType,
			"error", err,
		)

		action.Fail(err.Error())
		if updateErr := e.pendingRepo.Update(ctx, action); updateErr != nil {
			e.logger.Error("failed to update action", "action_id", action.ID, "error", updateErr)
		}

		if action.CanRetry() {
			result.Status = "retry"
		} else {
			result.Status = "failed"
		}
		result.Error = err.Error()
		return result
	}

	// Success
	action.Execute(actionResult)
	if err := e.pendingRepo.Update(ctx, action); err != nil {
		e.logger.Error("failed to update action", "action_id", action.ID, "error", err)
	}

	e.logger.Info("action executed successfully",
		"action_id", action.ID,
		"action_type", action.ActionType,
		"duration_ms", result.Duration.Milliseconds(),
	)

	result.Status = "success"
	result.Result = actionResult
	return result
}

// CancelPendingForRule cancels all pending actions for a rule.
func (e *ActionExecutor) CancelPendingForRule(ctx context.Context, ruleID uuid.UUID) (int, error) {
	actions, err := e.pendingRepo.GetByRuleID(ctx, ruleID)
	if err != nil {
		return 0, err
	}

	cancelled := 0
	for _, action := range actions {
		if action.Status == domain.PendingActionStatusPending {
			action.Cancel()
			if err := e.pendingRepo.Update(ctx, action); err != nil {
				e.logger.Error("failed to cancel action", "action_id", action.ID, "error", err)
				continue
			}
			cancelled++
		}
	}

	return cancelled, nil
}

// Standard action handlers

// NotificationActionHandler handles notification actions.
type NotificationActionHandler struct {
	logger *slog.Logger
}

// NewNotificationActionHandler creates a new notification action handler.
func NewNotificationActionHandler(logger *slog.Logger) *NotificationActionHandler {
	return &NotificationActionHandler{logger: logger}
}

// ActionType returns the action type.
func (h *NotificationActionHandler) ActionType() string {
	return "notification.send"
}

// Execute sends a notification.
func (h *NotificationActionHandler) Execute(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error) {
	title, _ := params["title"].(string)
	body, _ := params["body"].(string)
	priority, _ := params["priority"].(string)

	if title == "" {
		return nil, fmt.Errorf("notification title is required")
	}

	h.logger.Info("sending notification",
		"user_id", userID,
		"title", title,
		"body", body,
		"priority", priority,
	)

	// In a real implementation, this would send via push notification, email, etc.
	return map[string]any{
		"notification_id": uuid.New().String(),
		"delivered_at":    time.Now(),
	}, nil
}

// LogActionHandler handles log actions (for debugging/testing).
type LogActionHandler struct {
	logger *slog.Logger
}

// NewLogActionHandler creates a new log action handler.
func NewLogActionHandler(logger *slog.Logger) *LogActionHandler {
	return &LogActionHandler{logger: logger}
}

// ActionType returns the action type.
func (h *LogActionHandler) ActionType() string {
	return "debug.log"
}

// Execute logs the action parameters.
func (h *LogActionHandler) Execute(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error) {
	message, _ := params["message"].(string)
	level, _ := params["level"].(string)

	switch level {
	case "error":
		h.logger.Error(message, "user_id", userID, "params", params)
	case "warn":
		h.logger.Warn(message, "user_id", userID, "params", params)
	case "debug":
		h.logger.Debug(message, "user_id", userID, "params", params)
	default:
		h.logger.Info(message, "user_id", userID, "params", params)
	}

	return map[string]any{
		"logged_at": time.Now(),
	}, nil
}
