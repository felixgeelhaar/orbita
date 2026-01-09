// Package focusmode provides the Focus Mode orbit for deep work and distraction-free productivity.
package focusmode

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

const (
	// OrbitID is the unique identifier for the focus mode orbit.
	OrbitID = "orbita.focusmode"

	// OrbitVersion is the current version of the focus mode orbit.
	OrbitVersion = "1.0.0"
)

// Orbit implements the Focus Mode orbit.
type Orbit struct {
	ctx sdk.Context
}

// New creates a new Focus Mode Orbit instance.
func New() *Orbit {
	return &Orbit{}
}

// Metadata returns the orbit's metadata.
func (o *Orbit) Metadata() sdk.Metadata {
	return sdk.Metadata{
		ID:          OrbitID,
		Name:        "Focus Mode Pro",
		Version:     OrbitVersion,
		Author:      "Orbita",
		Description: "Deep work sessions with Pomodoro-style timers, focus tracking, and distraction insights",
		License:     "Proprietary",
		Homepage:    "https://orbita.app/orbits/focusmode",
		Tags:        []string{"productivity", "focus", "deep-work", "pomodoro", "time-tracking"},
	}
}

// RequiredCapabilities returns the capabilities required by this orbit.
func (o *Orbit) RequiredCapabilities() []sdk.Capability {
	return []sdk.Capability{
		sdk.CapReadTasks,       // To link focus sessions to tasks
		sdk.CapReadSchedule,    // To check scheduled focus time
		sdk.CapReadStorage,     // To read focus session data
		sdk.CapWriteStorage,    // To save focus session data
		sdk.CapSubscribeEvents, // To react to task events
		sdk.CapRegisterTools,   // To register MCP tools
	}
}

// Initialize initializes the orbit with the provided context.
func (o *Orbit) Initialize(ctx sdk.Context) error {
	o.ctx = ctx
	o.ctx.Logger().Info("focus mode orbit initialized")
	return nil
}

// Shutdown gracefully shuts down the orbit.
func (o *Orbit) Shutdown(ctx context.Context) error {
	if o.ctx != nil {
		o.ctx.Logger().Info("focus mode orbit shutting down")
	}
	return nil
}

// RegisterTools registers MCP tools for the focus mode orbit.
func (o *Orbit) RegisterTools(registry sdk.ToolRegistry) error {
	return registerTools(registry, o)
}

// RegisterCommands registers CLI commands for the focus mode orbit.
func (o *Orbit) RegisterCommands(registry sdk.CommandRegistry) error {
	// No CLI commands for now, all interaction via MCP tools
	return nil
}

// SubscribeEvents subscribes to domain events.
func (o *Orbit) SubscribeEvents(bus sdk.EventBus) error {
	// Subscribe to task completion to end linked focus sessions
	if err := bus.Subscribe("core.task.completed", o.onTaskCompleted); err != nil {
		return err
	}

	// Subscribe to schedule block events to start/end focus sessions
	if err := bus.Subscribe("scheduling.block.started", o.onBlockStarted); err != nil {
		return err
	}

	return nil
}

// Event handlers

func (o *Orbit) onTaskCompleted(ctx context.Context, event sdk.DomainEvent) error {
	o.ctx.Logger().Debug("task completed event received",
		"event_type", event.Type,
		"payload", event.Payload,
	)
	// Could auto-end focus sessions linked to completed tasks
	return nil
}

func (o *Orbit) onBlockStarted(ctx context.Context, event sdk.DomainEvent) error {
	o.ctx.Logger().Debug("schedule block started event received",
		"event_type", event.Type,
		"payload", event.Payload,
	)
	// Could auto-start focus sessions for deep work blocks
	return nil
}

// Context returns the orbit's context (for tools).
func (o *Orbit) Context() sdk.Context {
	return o.ctx
}
