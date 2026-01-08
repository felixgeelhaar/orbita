package grpc

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/engine/types"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// GRPCServer is implemented by plugin-side gRPC servers.
// Each engine type has its own server implementation that wraps
// the actual engine implementation and handles gRPC communication.

// Ensure plugins implement the GRPCPlugin interface.
var _ plugin.GRPCPlugin = (*SchedulerPlugin)(nil)
var _ plugin.GRPCPlugin = (*PriorityPlugin)(nil)
var _ plugin.GRPCPlugin = (*ClassifierPlugin)(nil)
var _ plugin.GRPCPlugin = (*AutomationPlugin)(nil)

// GRPCServer returns the gRPC server for scheduler plugins.
func (p *SchedulerPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// Registration will use generated proto code when available
	// For now, we document the expected interface
	return nil
}

// GRPCClient returns the gRPC client for scheduler plugins.
func (p *SchedulerPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &SchedulerGRPCClient{conn: c}, nil
}

// GRPCServer returns the gRPC server for priority plugins.
func (p *PriorityPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

// GRPCClient returns the gRPC client for priority plugins.
func (p *PriorityPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &PriorityGRPCClient{conn: c}, nil
}

// GRPCServer returns the gRPC server for classifier plugins.
func (p *ClassifierPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

// GRPCClient returns the gRPC client for classifier plugins.
func (p *ClassifierPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &ClassifierGRPCClient{conn: c}, nil
}

// GRPCServer returns the gRPC server for automation plugins.
func (p *AutomationPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

// GRPCClient returns the gRPC client for automation plugins.
func (p *AutomationPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &AutomationGRPCClient{conn: c}, nil
}

// BaseEngineServer provides common engine functionality for gRPC servers.
type BaseEngineServer struct {
	engine sdk.Engine
}

// NewBaseEngineServer creates a new base engine server.
func NewBaseEngineServer(engine sdk.Engine) *BaseEngineServer {
	return &BaseEngineServer{engine: engine}
}

// Metadata returns the engine metadata.
func (s *BaseEngineServer) Metadata() sdk.EngineMetadata {
	return s.engine.Metadata()
}

// Type returns the engine type.
func (s *BaseEngineServer) Type() sdk.EngineType {
	return s.engine.Type()
}

// ConfigSchema returns the configuration schema.
func (s *BaseEngineServer) ConfigSchema() sdk.ConfigSchema {
	return s.engine.ConfigSchema()
}

// Initialize initializes the engine.
func (s *BaseEngineServer) Initialize(ctx context.Context, config sdk.EngineConfig) error {
	return s.engine.Initialize(ctx, config)
}

// HealthCheck returns the health status.
func (s *BaseEngineServer) HealthCheck(ctx context.Context) sdk.HealthStatus {
	return s.engine.HealthCheck(ctx)
}

// Shutdown shuts down the engine.
func (s *BaseEngineServer) Shutdown(ctx context.Context) error {
	return s.engine.Shutdown(ctx)
}

// SchedulerGRPCServer wraps a scheduler engine for gRPC serving.
type SchedulerGRPCServer struct {
	BaseEngineServer
	impl types.SchedulerEngine
}

// NewSchedulerGRPCServer creates a new scheduler gRPC server.
func NewSchedulerGRPCServer(impl types.SchedulerEngine) *SchedulerGRPCServer {
	return &SchedulerGRPCServer{
		BaseEngineServer: *NewBaseEngineServer(impl),
		impl:             impl,
	}
}

// ScheduleTasks handles the ScheduleTasks RPC.
func (s *SchedulerGRPCServer) ScheduleTasks(ctx *sdk.ExecutionContext, input types.ScheduleTasksInput) (*types.ScheduleTasksOutput, error) {
	return s.impl.ScheduleTasks(ctx, input)
}

// FindOptimalSlot handles the FindOptimalSlot RPC.
func (s *SchedulerGRPCServer) FindOptimalSlot(ctx *sdk.ExecutionContext, input types.FindSlotInput) (*types.TimeSlot, error) {
	return s.impl.FindOptimalSlot(ctx, input)
}

// RescheduleConflicts handles the RescheduleConflicts RPC.
func (s *SchedulerGRPCServer) RescheduleConflicts(ctx *sdk.ExecutionContext, input types.RescheduleInput) (*types.RescheduleOutput, error) {
	return s.impl.RescheduleConflicts(ctx, input)
}

// CalculateUtilization handles the CalculateUtilization RPC.
func (s *SchedulerGRPCServer) CalculateUtilization(ctx *sdk.ExecutionContext, input types.UtilizationInput) (*types.UtilizationOutput, error) {
	return s.impl.CalculateUtilization(ctx, input)
}

// PriorityGRPCServer wraps a priority engine for gRPC serving.
type PriorityGRPCServer struct {
	BaseEngineServer
	impl types.PriorityEngine
}

// NewPriorityGRPCServer creates a new priority gRPC server.
func NewPriorityGRPCServer(impl types.PriorityEngine) *PriorityGRPCServer {
	return &PriorityGRPCServer{
		BaseEngineServer: *NewBaseEngineServer(impl),
		impl:             impl,
	}
}

// CalculatePriority handles the CalculatePriority RPC.
func (s *PriorityGRPCServer) CalculatePriority(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityOutput, error) {
	return s.impl.CalculatePriority(ctx, input)
}

// BatchCalculate handles the BatchCalculate RPC.
func (s *PriorityGRPCServer) BatchCalculate(ctx *sdk.ExecutionContext, inputs []types.PriorityInput) ([]types.PriorityOutput, error) {
	return s.impl.BatchCalculate(ctx, inputs)
}

// ExplainFactors handles the ExplainFactors RPC.
func (s *PriorityGRPCServer) ExplainFactors(ctx *sdk.ExecutionContext, input types.PriorityInput) (*types.PriorityExplanation, error) {
	return s.impl.ExplainFactors(ctx, input)
}

// ClassifierGRPCServer wraps a classifier engine for gRPC serving.
type ClassifierGRPCServer struct {
	BaseEngineServer
	impl types.ClassifierEngine
}

// NewClassifierGRPCServer creates a new classifier gRPC server.
func NewClassifierGRPCServer(impl types.ClassifierEngine) *ClassifierGRPCServer {
	return &ClassifierGRPCServer{
		BaseEngineServer: *NewBaseEngineServer(impl),
		impl:             impl,
	}
}

// Classify handles the Classify RPC.
func (s *ClassifierGRPCServer) Classify(ctx *sdk.ExecutionContext, input types.ClassifyInput) (*types.ClassifyOutput, error) {
	return s.impl.Classify(ctx, input)
}

// BatchClassify handles the BatchClassify RPC.
func (s *ClassifierGRPCServer) BatchClassify(ctx *sdk.ExecutionContext, inputs []types.ClassifyInput) ([]types.ClassifyOutput, error) {
	return s.impl.BatchClassify(ctx, inputs)
}

// GetCategories handles the GetCategories RPC.
func (s *ClassifierGRPCServer) GetCategories(ctx *sdk.ExecutionContext) ([]types.Category, error) {
	return s.impl.GetCategories(ctx)
}

// AutomationGRPCServer wraps an automation engine for gRPC serving.
type AutomationGRPCServer struct {
	BaseEngineServer
	impl types.AutomationEngine
}

// NewAutomationGRPCServer creates a new automation gRPC server.
func NewAutomationGRPCServer(impl types.AutomationEngine) *AutomationGRPCServer {
	return &AutomationGRPCServer{
		BaseEngineServer: *NewBaseEngineServer(impl),
		impl:             impl,
	}
}

// Evaluate handles the Evaluate RPC.
func (s *AutomationGRPCServer) Evaluate(ctx *sdk.ExecutionContext, input types.AutomationInput) (*types.AutomationOutput, error) {
	return s.impl.Evaluate(ctx, input)
}

// ValidateRule handles the ValidateRule RPC.
func (s *AutomationGRPCServer) ValidateRule(ctx *sdk.ExecutionContext, rule types.AutomationRule) error {
	return s.impl.ValidateRule(ctx, rule)
}

// GetSupportedTriggers handles the GetSupportedTriggers RPC.
func (s *AutomationGRPCServer) GetSupportedTriggers(ctx *sdk.ExecutionContext) ([]types.TriggerDefinition, error) {
	return s.impl.GetSupportedTriggers(ctx)
}

// GetSupportedActions handles the GetSupportedActions RPC.
func (s *AutomationGRPCServer) GetSupportedActions(ctx *sdk.ExecutionContext) ([]types.ActionDefinition, error) {
	return s.impl.GetSupportedActions(ctx)
}
