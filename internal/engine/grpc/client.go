package grpc

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"google.golang.org/grpc"
)

// GRPCClient interfaces are implemented by host-side gRPC clients.
// These wrap the gRPC client connections and translate between
// Go types and protobuf messages.

// BaseGRPCClient provides common engine functionality for gRPC clients.
type BaseGRPCClient struct {
	conn *grpc.ClientConn
}

// SchedulerGRPCClient is the gRPC client for scheduler engines.
type SchedulerGRPCClient struct {
	conn *grpc.ClientConn
}

// Metadata returns the engine metadata.
func (c *SchedulerGRPCClient) Metadata() sdk.EngineMetadata {
	// Will call gRPC Metadata RPC when proto is generated
	return sdk.EngineMetadata{}
}

// Type returns the engine type.
func (c *SchedulerGRPCClient) Type() sdk.EngineType {
	return sdk.EngineTypeScheduler
}

// ConfigSchema returns the configuration schema.
func (c *SchedulerGRPCClient) ConfigSchema() sdk.ConfigSchema {
	// Will call gRPC ConfigSchema RPC when proto is generated
	return sdk.ConfigSchema{}
}

// Initialize initializes the engine.
func (c *SchedulerGRPCClient) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	// Will call gRPC Initialize RPC when proto is generated
	return nil
}

// HealthCheck returns the health status.
func (c *SchedulerGRPCClient) HealthCheck(ctx context.Context) sdk.HealthStatus {
	// Will call gRPC HealthCheck RPC when proto is generated
	return sdk.HealthStatus{Healthy: true}
}

// Shutdown shuts down the engine.
func (c *SchedulerGRPCClient) Shutdown(ctx context.Context) error {
	// Will call gRPC Shutdown RPC when proto is generated
	return nil
}

// ScheduleTasks schedules tasks.
func (c *SchedulerGRPCClient) ScheduleTasks(ctx *sdk.ExecutionContext, input types.ScheduleTasksInput) (*types.ScheduleTasksOutput, error) {
	// Will call gRPC ScheduleTasks RPC when proto is generated
	return &types.ScheduleTasksOutput{}, nil
}

// FindOptimalSlot finds optimal slot.
func (c *SchedulerGRPCClient) FindOptimalSlot(ctx *sdk.ExecutionContext, input types.FindSlotInput) (*types.TimeSlot, error) {
	// Will call gRPC FindOptimalSlot RPC when proto is generated
	return &types.TimeSlot{}, nil
}

// RescheduleConflicts handles conflicts.
func (c *SchedulerGRPCClient) RescheduleConflicts(ctx *sdk.ExecutionContext, input types.RescheduleInput) (*types.RescheduleOutput, error) {
	// Will call gRPC RescheduleConflicts RPC when proto is generated
	return &types.RescheduleOutput{}, nil
}

// CalculateUtilization calculates utilization.
func (c *SchedulerGRPCClient) CalculateUtilization(ctx *sdk.ExecutionContext, input types.UtilizationInput) (*types.UtilizationOutput, error) {
	// Will call gRPC CalculateUtilization RPC when proto is generated
	return &types.UtilizationOutput{}, nil
}

// PriorityGRPCClient is the gRPC client for priority engines.
type PriorityGRPCClient struct {
	conn *grpc.ClientConn
}

// Metadata returns the engine metadata.
func (c *PriorityGRPCClient) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{}
}

// Type returns the engine type.
func (c *PriorityGRPCClient) Type() sdk.EngineType {
	return sdk.EngineTypePriority
}

// ConfigSchema returns the configuration schema.
func (c *PriorityGRPCClient) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{}
}

// Initialize initializes the engine.
func (c *PriorityGRPCClient) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	return nil
}

// HealthCheck returns the health status.
func (c *PriorityGRPCClient) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{Healthy: true}
}

// Shutdown shuts down the engine.
func (c *PriorityGRPCClient) Shutdown(ctx context.Context) error {
	return nil
}

// CalculatePriority calculates priority.
func (c *PriorityGRPCClient) CalculatePriority(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityOutput, error) {
	return &types.PriorityOutput{}, nil
}

// BatchCalculate batch calculates priorities.
func (c *PriorityGRPCClient) BatchCalculate(ctx *sdk.ExecutionContext, inputs []types.PriorityInput) ([]types.PriorityOutput, error) {
	return []types.PriorityOutput{}, nil
}

// ExplainFactors explains priority factors.
func (c *PriorityGRPCClient) ExplainFactors(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityExplanation, error) {
	return &types.PriorityExplanation{}, nil
}

// ClassifierGRPCClient is the gRPC client for classifier engines.
type ClassifierGRPCClient struct {
	conn *grpc.ClientConn
}

// Metadata returns the engine metadata.
func (c *ClassifierGRPCClient) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{}
}

// Type returns the engine type.
func (c *ClassifierGRPCClient) Type() sdk.EngineType {
	return sdk.EngineTypeClassifier
}

// ConfigSchema returns the configuration schema.
func (c *ClassifierGRPCClient) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{}
}

// Initialize initializes the engine.
func (c *ClassifierGRPCClient) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	return nil
}

// HealthCheck returns the health status.
func (c *ClassifierGRPCClient) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{Healthy: true}
}

// Shutdown shuts down the engine.
func (c *ClassifierGRPCClient) Shutdown(ctx context.Context) error {
	return nil
}

// Classify classifies content.
func (c *ClassifierGRPCClient) Classify(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
	return &types.ClassifyOutput{}, nil
}

// BatchClassify batch classifies content.
func (c *ClassifierGRPCClient) BatchClassify(ctx *sdk.ExecutionContext, inputs []types.ClassifyInput) ([]types.ClassifyOutput, error) {
	return []types.ClassifyOutput{}, nil
}

// GetCategories returns categories.
func (c *ClassifierGRPCClient) GetCategories(ctx *sdk.ExecutionContext) ([]types.Category, error) {
	return []types.Category{}, nil
}

// AutomationGRPCClient is the gRPC client for automation engines.
type AutomationGRPCClient struct {
	conn *grpc.ClientConn
}

// Metadata returns the engine metadata.
func (c *AutomationGRPCClient) Metadata() sdk.EngineMetadata {
	return sdk.EngineMetadata{}
}

// Type returns the engine type.
func (c *AutomationGRPCClient) Type() sdk.EngineType {
	return sdk.EngineTypeAutomation
}

// ConfigSchema returns the configuration schema.
func (c *AutomationGRPCClient) ConfigSchema() sdk.ConfigSchema {
	return sdk.ConfigSchema{}
}

// Initialize initializes the engine.
func (c *AutomationGRPCClient) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	return nil
}

// HealthCheck returns the health status.
func (c *AutomationGRPCClient) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return sdk.HealthStatus{Healthy: true}
}

// Shutdown shuts down the engine.
func (c *AutomationGRPCClient) Shutdown(ctx context.Context) error {
	return nil
}

// Evaluate evaluates automation rules.
func (c *AutomationGRPCClient) Evaluate(ctx *sdk.ExecutionContext, input types.AutomationInput) (*types.AutomationOutput, error) {
	return &types.AutomationOutput{}, nil
}

// ValidateRule validates an automation rule.
func (c *AutomationGRPCClient) ValidateRule(ctx *sdk.ExecutionContext, rule types.AutomationRule) error {
	return nil
}

// GetSupportedTriggers returns supported triggers.
func (c *AutomationGRPCClient) GetSupportedTriggers(ctx *sdk.ExecutionContext) ([]types.TriggerDefinition, error) {
	return []types.TriggerDefinition{}, nil
}

// GetSupportedActions returns supported actions.
func (c *AutomationGRPCClient) GetSupportedActions(ctx *sdk.ExecutionContext) ([]types.ActionDefinition, error) {
	return []types.ActionDefinition{}, nil
}

// Verify interface compliance at compile time.
var (
	_ types.SchedulerEngine  = (*SchedulerGRPCClient)(nil)
	_ types.PriorityEngine   = (*PriorityGRPCClient)(nil)
	_ types.ClassifierEngine = (*ClassifierGRPCClient)(nil)
	_ types.AutomationEngine = (*AutomationGRPCClient)(nil)
)
