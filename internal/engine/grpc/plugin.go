// Package grpc provides gRPC-based plugin communication for Orbita engines.
// It uses HashiCorp's go-plugin library for process isolation and management.
package grpc

import (
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/hashicorp/go-plugin"
)

// HandshakeConfig is used to verify that the plugin is compatible.
// Both the core and plugins must use the same handshake configuration.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ORBITA_ENGINE_PLUGIN",
	MagicCookieValue: "orbita-engine-v1",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"scheduler":  &SchedulerPlugin{},
	"priority":   &PriorityPlugin{},
	"classifier": &ClassifierPlugin{},
	"automation": &AutomationPlugin{},
}

// PluginMapForEngine returns a plugin map for a specific engine type.
func PluginMapForEngine(engineType sdk.EngineType) map[string]plugin.Plugin {
	switch engineType {
	case sdk.EngineTypeScheduler:
		return map[string]plugin.Plugin{"engine": &SchedulerPlugin{}}
	case sdk.EngineTypePriority:
		return map[string]plugin.Plugin{"engine": &PriorityPlugin{}}
	case sdk.EngineTypeClassifier:
		return map[string]plugin.Plugin{"engine": &ClassifierPlugin{}}
	case sdk.EngineTypeAutomation:
		return map[string]plugin.Plugin{"engine": &AutomationPlugin{}}
	default:
		return nil
	}
}

// SchedulerPlugin is the plugin.Plugin implementation for scheduler engines.
type SchedulerPlugin struct {
	plugin.Plugin
	// Impl is the concrete implementation (plugin-side).
	Impl SchedulerEnginePlugin
}

// PriorityPlugin is the plugin.Plugin implementation for priority engines.
type PriorityPlugin struct {
	plugin.Plugin
	// Impl is the concrete implementation (plugin-side).
	Impl PriorityEnginePlugin
}

// ClassifierPlugin is the plugin.Plugin implementation for classifier engines.
type ClassifierPlugin struct {
	plugin.Plugin
	// Impl is the concrete implementation (plugin-side).
	Impl ClassifierEnginePlugin
}

// AutomationPlugin is the plugin.Plugin implementation for automation engines.
type AutomationPlugin struct {
	plugin.Plugin
	// Impl is the concrete implementation (plugin-side).
	Impl AutomationEnginePlugin
}

// SchedulerEnginePlugin is the interface for scheduler engine plugins.
type SchedulerEnginePlugin interface {
	sdk.Engine
	ScheduleTasks(ctx *sdk.ExecutionContext, input ScheduleTasksInput) (*ScheduleTasksOutput, error)
	FindOptimalSlot(ctx *sdk.ExecutionContext, input FindSlotInput) (*TimeSlot, error)
	RescheduleConflicts(ctx *sdk.ExecutionContext, input RescheduleInput) (*RescheduleOutput, error)
	CalculateUtilization(ctx *sdk.ExecutionContext, input UtilizationInput) (*UtilizationOutput, error)
}

// PriorityEnginePlugin is the interface for priority engine plugins.
type PriorityEnginePlugin interface {
	sdk.Engine
	CalculatePriority(ctx *sdk.ExecutionContext, input PriorityInput) (*PriorityOutput, error)
	BatchCalculate(ctx *sdk.ExecutionContext, inputs []PriorityInput) ([]PriorityOutput, error)
	ExplainFactors(ctx *sdk.ExecutionContext, input PriorityInput) (*PriorityExplanation, error)
}

// ClassifierEnginePlugin is the interface for classifier engine plugins.
type ClassifierEnginePlugin interface {
	sdk.Engine
	Classify(ctx *sdk.ExecutionContext, input ClassifyInput) (*ClassifyOutput, error)
	BatchClassify(ctx *sdk.ExecutionContext, inputs []ClassifyInput) ([]ClassifyOutput, error)
	GetCategories(ctx *sdk.ExecutionContext) ([]Category, error)
}

// AutomationEnginePlugin is the interface for automation engine plugins.
type AutomationEnginePlugin interface {
	sdk.Engine
	Evaluate(ctx *sdk.ExecutionContext, input AutomationInput) (*AutomationOutput, error)
	ValidateRule(ctx *sdk.ExecutionContext, rule AutomationRule) error
	GetSupportedTriggers(ctx *sdk.ExecutionContext) ([]TriggerDefinition, error)
	GetSupportedActions(ctx *sdk.ExecutionContext) ([]ActionDefinition, error)
}
