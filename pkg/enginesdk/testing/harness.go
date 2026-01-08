// Package testing provides testing utilities for Orbita engine plugins.
//
// Example usage:
//
//	func TestMyEngine(t *testing.T) {
//		harness := testing.NewHarness(&MyEngine{})
//
//		// Test initialization
//		err := harness.Initialize(map[string]any{
//			"my_setting": "value",
//		})
//		require.NoError(t, err)
//
//		// Test priority calculation
//		result, err := harness.ExecutePriority(testing.PriorityInput{
//			ID:       uuid.New(),
//			Priority: 2,
//		})
//		require.NoError(t, err)
//		assert.Greater(t, result.Score, 0.0)
//	}
package testing

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/google/uuid"
)

// Harness provides a test harness for engine plugins.
type Harness struct {
	engine sdk.Engine
	config sdk.EngineConfig
	logger *slog.Logger
	userID uuid.UUID
}

// NewHarness creates a new test harness for an engine.
func NewHarness(engine sdk.Engine) *Harness {
	userID := uuid.New()
	return &Harness{
		engine: engine,
		config: sdk.NewEngineConfig(engine.Metadata().ID, userID, nil),
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		userID: userID,
	}
}

// WithLogger sets a custom logger.
func (h *Harness) WithLogger(logger *slog.Logger) *Harness {
	h.logger = logger
	return h
}

// WithUserID sets a custom user ID for execution context.
func (h *Harness) WithUserID(userID uuid.UUID) *Harness {
	h.userID = userID
	return h
}

// Initialize initializes the engine with the given configuration.
func (h *Harness) Initialize(config map[string]any) error {
	h.config = sdk.NewEngineConfig(h.engine.Metadata().ID, h.userID, config)
	return h.engine.Initialize(context.Background(), h.config)
}

// Shutdown shuts down the engine.
func (h *Harness) Shutdown() error {
	return h.engine.Shutdown(context.Background())
}

// HealthCheck checks engine health.
func (h *Harness) HealthCheck() sdk.HealthStatus {
	return h.engine.HealthCheck(context.Background())
}

// Metadata returns engine metadata.
func (h *Harness) Metadata() sdk.EngineMetadata {
	return h.engine.Metadata()
}

// ConfigSchema returns the configuration schema.
func (h *Harness) ConfigSchema() sdk.ConfigSchema {
	return h.engine.ConfigSchema()
}

// createContext creates an execution context for testing.
func (h *Harness) createContext() *sdk.ExecutionContext {
	ctx := sdk.NewExecutionContext(context.Background(), h.userID, h.engine.Metadata().ID)
	ctx.WithLogger(h.logger)
	return ctx
}

// Scheduler Engine Methods

// ExecuteScheduleTasks executes the ScheduleTasks operation.
func (h *Harness) ExecuteScheduleTasks(input types.ScheduleTasksInput) (*types.ScheduleTasksOutput, error) {
	scheduler, ok := h.engine.(types.SchedulerEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return scheduler.ScheduleTasks(h.createContext(), input)
}

// ExecuteFindOptimalSlot executes the FindOptimalSlot operation.
func (h *Harness) ExecuteFindOptimalSlot(input types.FindSlotInput) (*types.TimeSlot, error) {
	scheduler, ok := h.engine.(types.SchedulerEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return scheduler.FindOptimalSlot(h.createContext(), input)
}

// ExecuteRescheduleConflicts executes the RescheduleConflicts operation.
func (h *Harness) ExecuteRescheduleConflicts(input types.RescheduleInput) (*types.RescheduleOutput, error) {
	scheduler, ok := h.engine.(types.SchedulerEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return scheduler.RescheduleConflicts(h.createContext(), input)
}

// ExecuteCalculateUtilization executes the CalculateUtilization operation.
func (h *Harness) ExecuteCalculateUtilization(input types.UtilizationInput) (*types.UtilizationOutput, error) {
	scheduler, ok := h.engine.(types.SchedulerEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return scheduler.CalculateUtilization(h.createContext(), input)
}

// Priority Engine Methods

// ExecutePriority executes the CalculatePriority operation.
func (h *Harness) ExecutePriority(input types.PriorityInput) (*types.PriorityOutput, error) {
	priority, ok := h.engine.(types.PriorityEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return priority.CalculatePriority(h.createContext(), input)
}

// ExecuteBatchPriority executes the BatchCalculate operation.
func (h *Harness) ExecuteBatchPriority(inputs []types.PriorityInput) ([]types.PriorityOutput, error) {
	priority, ok := h.engine.(types.PriorityEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return priority.BatchCalculate(h.createContext(), inputs)
}

// ExecuteExplainFactors executes the ExplainFactors operation.
func (h *Harness) ExecuteExplainFactors(input types.PriorityInput) (*types.PriorityExplanation, error) {
	priority, ok := h.engine.(types.PriorityEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return priority.ExplainFactors(h.createContext(), input)
}

// Classifier Engine Methods

// ExecuteClassify executes the Classify operation.
func (h *Harness) ExecuteClassify(input types.ClassifyInput) (*types.ClassifyOutput, error) {
	classifier, ok := h.engine.(types.ClassifierEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return classifier.Classify(h.createContext(), input)
}

// ExecuteBatchClassify executes the BatchClassify operation.
func (h *Harness) ExecuteBatchClassify(inputs []types.ClassifyInput) ([]types.ClassifyOutput, error) {
	classifier, ok := h.engine.(types.ClassifierEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return classifier.BatchClassify(h.createContext(), inputs)
}

// ExecuteGetCategories executes the GetCategories operation.
func (h *Harness) ExecuteGetCategories() ([]types.Category, error) {
	classifier, ok := h.engine.(types.ClassifierEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return classifier.GetCategories(h.createContext())
}

// Automation Engine Methods

// ExecuteAutomation executes the Evaluate operation.
func (h *Harness) ExecuteAutomation(input types.AutomationInput) (*types.AutomationOutput, error) {
	automation, ok := h.engine.(types.AutomationEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return automation.Evaluate(h.createContext(), input)
}

// ExecuteValidateRule executes the ValidateRule operation.
func (h *Harness) ExecuteValidateRule(rule types.AutomationRule) error {
	automation, ok := h.engine.(types.AutomationEngine)
	if !ok {
		return ErrWrongEngineType
	}
	return automation.ValidateRule(h.createContext(), rule)
}

// ExecuteGetSupportedTriggers executes the GetSupportedTriggers operation.
func (h *Harness) ExecuteGetSupportedTriggers() ([]types.TriggerDefinition, error) {
	automation, ok := h.engine.(types.AutomationEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return automation.GetSupportedTriggers(h.createContext())
}

// ExecuteGetSupportedActions executes the GetSupportedActions operation.
func (h *Harness) ExecuteGetSupportedActions() ([]types.ActionDefinition, error) {
	automation, ok := h.engine.(types.AutomationEngine)
	if !ok {
		return nil, ErrWrongEngineType
	}
	return automation.GetSupportedActions(h.createContext())
}

// Test Data Helpers

// NewTestTask creates a test task for scheduler testing.
func NewTestTask(title string, priority int, duration time.Duration) types.SchedulableTask {
	due := time.Now().Add(7 * 24 * time.Hour)
	return types.SchedulableTask{
		ID:       uuid.New(),
		Title:    title,
		Priority: priority,
		Duration: duration,
		DueDate:  &due,
	}
}

// NewTestPriorityInput creates a test priority input.
func NewTestPriorityInput(priority int) types.PriorityInput {
	due := time.Now().Add(7 * 24 * time.Hour)
	return types.PriorityInput{
		ID:       uuid.New(),
		Priority: priority,
		DueDate:  &due,
		Duration: 30 * time.Minute,
	}
}

// NewTestClassifyInput creates a test classification input.
func NewTestClassifyInput(content string) types.ClassifyInput {
	return types.ClassifyInput{
		ID:      uuid.New(),
		Content: content,
	}
}

// NewTestAutomationEvent creates a test automation event.
func NewTestAutomationEvent(eventType string) types.AutomationEvent {
	return types.AutomationEvent{
		ID:           uuid.New(),
		Type:         eventType,
		EntityID:     uuid.New(),
		EntityType:   "task",
		Timestamp:    time.Now(),
		Data:         make(map[string]any),
		CurrentState: make(map[string]any),
	}
}

// NewTestAutomationRule creates a test automation rule.
func NewTestAutomationRule(name string, eventTypes ...string) types.AutomationRule {
	return types.AutomationRule{
		ID:      uuid.New(),
		Name:    name,
		Enabled: true,
		Trigger: types.RuleTrigger{
			Type:       "event",
			EventTypes: eventTypes,
		},
		Actions: []types.RuleAction{
			{
				Type:       "notification.send",
				Target:     "self",
				Parameters: map[string]any{"message": "Test notification"},
			},
		},
	}
}

// Error types
var (
	// ErrWrongEngineType is returned when the engine doesn't implement the expected interface.
	ErrWrongEngineType = &EngineTypeError{Message: "engine does not implement the required interface"}
)

// EngineTypeError represents an engine type mismatch error.
type EngineTypeError struct {
	Message string
}

func (e *EngineTypeError) Error() string {
	return e.Message
}
