// Package wellness provides the Wellness Sync orbit for health and wellness tracking.
package wellness

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

const (
	// OrbitID is the unique identifier for the wellness orbit.
	OrbitID = "orbita.wellness"

	// OrbitVersion is the current version of the wellness orbit.
	OrbitVersion = "1.0.0"
)

// Orbit implements the Wellness Sync orbit.
type Orbit struct {
	ctx sdk.Context
}

// New creates a new Wellness Orbit instance.
func New() *Orbit {
	return &Orbit{}
}

// Metadata returns the orbit's metadata.
func (o *Orbit) Metadata() sdk.Metadata {
	return sdk.Metadata{
		ID:          OrbitID,
		Name:        "Wellness Sync",
		Version:     OrbitVersion,
		Author:      "Orbita",
		Description: "Health and wellness tracking with mood, energy, sleep, stress, and more",
		License:     "Proprietary",
		Homepage:    "https://orbita.app/orbits/wellness",
		Tags:        []string{"health", "wellness", "tracking", "habits"},
	}
}

// RequiredCapabilities returns the capabilities required by this orbit.
func (o *Orbit) RequiredCapabilities() []sdk.Capability {
	return []sdk.Capability{
		sdk.CapReadTasks,       // To correlate wellness with productivity
		sdk.CapReadSchedule,    // To correlate wellness with schedule
		sdk.CapReadStorage,     // To read wellness data
		sdk.CapWriteStorage,    // To write wellness entries
		sdk.CapSubscribeEvents, // To react to domain events
		sdk.CapRegisterTools,   // To register MCP tools
	}
}

// Initialize initializes the orbit with the provided context.
func (o *Orbit) Initialize(ctx sdk.Context) error {
	o.ctx = ctx
	o.ctx.Logger().Info("wellness orbit initialized")
	return nil
}

// Shutdown gracefully shuts down the orbit.
func (o *Orbit) Shutdown(ctx context.Context) error {
	if o.ctx != nil {
		o.ctx.Logger().Info("wellness orbit shutting down")
	}
	return nil
}

// RegisterTools registers MCP tools for the wellness orbit.
func (o *Orbit) RegisterTools(registry sdk.ToolRegistry) error {
	return registerTools(registry, o)
}

// RegisterCommands registers CLI commands for the wellness orbit.
func (o *Orbit) RegisterCommands(registry sdk.CommandRegistry) error {
	// No CLI commands for now, all interaction via MCP tools
	return nil
}

// SubscribeEvents subscribes to domain events.
func (o *Orbit) SubscribeEvents(bus sdk.EventBus) error {
	// Subscribe to habit completion events to suggest wellness logging
	if err := bus.Subscribe("habits.habit.completed", o.onHabitCompleted); err != nil {
		return err
	}

	// Subscribe to task completion events to correlate productivity
	if err := bus.Subscribe("core.task.completed", o.onTaskCompleted); err != nil {
		return err
	}

	return nil
}

// Event handlers

func (o *Orbit) onHabitCompleted(ctx context.Context, event sdk.DomainEvent) error {
	o.ctx.Logger().Debug("habit completed event received",
		"event_type", event.Type,
		"payload", event.Payload,
	)
	// Could trigger wellness check-in suggestion
	return nil
}

func (o *Orbit) onTaskCompleted(ctx context.Context, event sdk.DomainEvent) error {
	o.ctx.Logger().Debug("task completed event received",
		"event_type", event.Type,
		"payload", event.Payload,
	)
	// Could track productivity correlation with wellness
	return nil
}

// Context returns the orbit's context (for tools).
func (o *Orbit) Context() sdk.Context {
	return o.ctx
}
