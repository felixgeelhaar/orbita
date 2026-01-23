package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/automations/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockActionHandler for testing
type mockActionHandler struct {
	actionType string
	execFunc   func(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error)
}

func newMockActionHandler(actionType string) *mockActionHandler {
	return &mockActionHandler{
		actionType: actionType,
	}
}

func (m *mockActionHandler) ActionType() string {
	return m.actionType
}

func (m *mockActionHandler) Execute(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, userID, target, params)
	}
	return map[string]any{"success": true}, nil
}

func TestActionExecutor_ExecutePending_NoActions(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()
	executor := NewActionExecutor(pendingRepo, testLogger())

	result, err := executor.ExecutePending(context.Background(), 100)

	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalProcessed)
	assert.Equal(t, 0, result.SuccessCount)
}

func TestActionExecutor_ExecutePending_Success(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()

	userID := uuid.New()
	ruleID := uuid.New()
	executionID := uuid.New()

	// Create a pending action that is due
	action := domain.NewPendingAction(
		executionID,
		ruleID,
		userID,
		"notification.send",
		map[string]any{
			"title": "Test Notification",
			"body":  "This is a test",
		},
		time.Now().Add(-1*time.Minute), // Due in the past
	)
	_ = pendingRepo.Create(context.Background(), action)

	executor := NewActionExecutor(pendingRepo, testLogger())

	// Register a handler
	handler := newMockActionHandler("notification.send")
	handler.execFunc = func(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error) {
		return map[string]any{"notification_id": uuid.New().String()}, nil
	}
	executor.RegisterHandler(handler)

	result, err := executor.ExecutePending(context.Background(), 100)

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalProcessed)
	assert.Equal(t, 1, result.SuccessCount)
	assert.Equal(t, 0, result.FailedCount)

	// Verify action was marked as executed
	updatedAction, _ := pendingRepo.GetByID(context.Background(), action.ID)
	assert.Equal(t, domain.PendingActionStatusExecuted, updatedAction.Status)
}

func TestActionExecutor_ExecutePending_NoHandler(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()

	userID := uuid.New()
	ruleID := uuid.New()
	executionID := uuid.New()

	// Create a pending action with an unknown action type
	action := domain.NewPendingAction(
		executionID,
		ruleID,
		userID,
		"unknown.action",
		map[string]any{},
		time.Now().Add(-1*time.Minute),
	)
	_ = pendingRepo.Create(context.Background(), action)

	executor := NewActionExecutor(pendingRepo, testLogger())

	result, err := executor.ExecutePending(context.Background(), 100)

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalProcessed)
	assert.Equal(t, 0, result.SuccessCount)
	assert.Equal(t, 1, result.FailedCount)
	assert.Contains(t, result.Results[0].Error, "no handler")
}

func TestActionExecutor_ExecutePending_HandlerError(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()

	userID := uuid.New()
	ruleID := uuid.New()
	executionID := uuid.New()

	action := domain.NewPendingAction(
		executionID,
		ruleID,
		userID,
		"failing.action",
		map[string]any{},
		time.Now().Add(-1*time.Minute),
	)
	_ = pendingRepo.Create(context.Background(), action)

	executor := NewActionExecutor(pendingRepo, testLogger())

	// Register a handler that fails
	handler := newMockActionHandler("failing.action")
	handler.execFunc = func(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error) {
		return nil, errors.New("handler error")
	}
	executor.RegisterHandler(handler)

	result, err := executor.ExecutePending(context.Background(), 100)

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalProcessed)
	assert.Equal(t, 1, result.RetryCount) // First failure goes to retry
	assert.Contains(t, result.Results[0].Error, "handler error")

	// Action should have retry count incremented
	updatedAction, _ := pendingRepo.GetByID(context.Background(), action.ID)
	assert.Equal(t, 1, updatedAction.RetryCount)
}

func TestActionExecutor_ExecuteAction_NotFound(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()
	executor := NewActionExecutor(pendingRepo, testLogger())

	_, err := executor.ExecuteAction(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestActionExecutor_ExecuteAction_Success(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()

	userID := uuid.New()
	ruleID := uuid.New()
	executionID := uuid.New()

	action := domain.NewPendingAction(
		executionID,
		ruleID,
		userID,
		"test.action",
		map[string]any{"key": "value"},
		time.Now().Add(-1*time.Minute),
	)
	_ = pendingRepo.Create(context.Background(), action)

	executor := NewActionExecutor(pendingRepo, testLogger())

	handler := newMockActionHandler("test.action")
	handler.execFunc = func(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error) {
		return map[string]any{"received_key": params["key"]}, nil
	}
	executor.RegisterHandler(handler)

	result, err := executor.ExecuteAction(context.Background(), action.ID)

	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "value", result.Result["received_key"])
}

func TestActionExecutor_CancelPendingForRule(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()

	userID := uuid.New()
	ruleID := uuid.New()
	executionID := uuid.New()

	// Create multiple pending actions for the same rule
	for i := 0; i < 3; i++ {
		action := domain.NewPendingAction(
			executionID,
			ruleID,
			userID,
			"test.action",
			map[string]any{},
			time.Now().Add(1*time.Hour), // Future execution
		)
		_ = pendingRepo.Create(context.Background(), action)
	}

	executor := NewActionExecutor(pendingRepo, testLogger())

	cancelled, err := executor.CancelPendingForRule(context.Background(), ruleID)

	require.NoError(t, err)
	assert.Equal(t, 3, cancelled)

	// Verify all actions are cancelled
	actions, _ := pendingRepo.GetByRuleID(context.Background(), ruleID)
	for _, action := range actions {
		assert.Equal(t, domain.PendingActionStatusCancelled, action.Status)
	}
}

func TestActionExecutor_MultipleActionsInOrder(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()

	userID := uuid.New()
	ruleID := uuid.New()
	executionID := uuid.New()

	// Create multiple due actions
	for i := 0; i < 5; i++ {
		action := domain.NewPendingAction(
			executionID,
			ruleID,
			userID,
			"batch.action",
			map[string]any{"index": i},
			time.Now().Add(-1*time.Minute),
		)
		_ = pendingRepo.Create(context.Background(), action)
	}

	executor := NewActionExecutor(pendingRepo, testLogger())

	executedIndices := []int{}
	handler := newMockActionHandler("batch.action")
	handler.execFunc = func(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error) {
		if idx, ok := params["index"].(int); ok {
			executedIndices = append(executedIndices, idx)
		}
		return map[string]any{"success": true}, nil
	}
	executor.RegisterHandler(handler)

	result, err := executor.ExecutePending(context.Background(), 100)

	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalProcessed)
	assert.Equal(t, 5, result.SuccessCount)
	assert.Len(t, executedIndices, 5)
}

func TestNotificationActionHandler_Execute(t *testing.T) {
	handler := NewNotificationActionHandler(testLogger())

	assert.Equal(t, "notification.send", handler.ActionType())

	result, err := handler.Execute(context.Background(), uuid.New(), "", map[string]any{
		"title":    "Test Title",
		"body":     "Test Body",
		"priority": "high",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result["notification_id"])
	assert.NotNil(t, result["delivered_at"])
}

func TestNotificationActionHandler_Execute_MissingTitle(t *testing.T) {
	handler := NewNotificationActionHandler(testLogger())

	_, err := handler.Execute(context.Background(), uuid.New(), "", map[string]any{
		"body": "Body without title",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestLogActionHandler_Execute(t *testing.T) {
	handler := NewLogActionHandler(testLogger())

	assert.Equal(t, "debug.log", handler.ActionType())

	result, err := handler.Execute(context.Background(), uuid.New(), "", map[string]any{
		"message": "Test log message",
		"level":   "info",
	})

	require.NoError(t, err)
	assert.NotNil(t, result["logged_at"])
}

func TestActionExecutor_RetryMechanism(t *testing.T) {
	pendingRepo := newMockPendingActionRepo()

	userID := uuid.New()
	ruleID := uuid.New()
	executionID := uuid.New()

	action := domain.NewPendingAction(
		executionID,
		ruleID,
		userID,
		"retry.action",
		map[string]any{},
		time.Now().Add(-1*time.Minute),
	)
	action.MaxRetries = 3
	_ = pendingRepo.Create(context.Background(), action)

	executor := NewActionExecutor(pendingRepo, testLogger())

	failCount := 0
	handler := newMockActionHandler("retry.action")
	handler.execFunc = func(ctx context.Context, userID uuid.UUID, target string, params map[string]any) (map[string]any, error) {
		failCount++
		return nil, errors.New("transient error")
	}
	executor.RegisterHandler(handler)

	// First execution attempt
	result1, _ := executor.ExecutePending(context.Background(), 100)
	assert.Equal(t, 1, result1.RetryCount)

	// Update action's scheduled time to make it due again
	action.ScheduledFor = time.Now().Add(-1 * time.Minute)
	_ = pendingRepo.Update(context.Background(), action)

	// Second attempt
	result2, _ := executor.ExecutePending(context.Background(), 100)
	assert.Equal(t, 1, result2.RetryCount)

	// Third attempt
	action.ScheduledFor = time.Now().Add(-1 * time.Minute)
	_ = pendingRepo.Update(context.Background(), action)
	result3, _ := executor.ExecutePending(context.Background(), 100)
	assert.Equal(t, 1, result3.FailedCount) // After max retries, it's failed

	// Verify final state
	updatedAction, _ := pendingRepo.GetByID(context.Background(), action.ID)
	assert.Equal(t, domain.PendingActionStatusFailed, updatedAction.Status)
	assert.Equal(t, 3, updatedAction.RetryCount)
}
