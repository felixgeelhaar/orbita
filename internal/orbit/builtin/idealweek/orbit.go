// Package idealweek provides the Ideal Week Designer orbit for schedule planning.
package idealweek

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

const (
	// OrbitID is the unique identifier for the ideal week orbit.
	OrbitID = "orbita.idealweek"

	// OrbitVersion is the current version of the ideal week orbit.
	OrbitVersion = "1.0.0"
)

// Orbit implements the Ideal Week Designer orbit.
type Orbit struct {
	ctx sdk.Context
}

// New creates a new Ideal Week Orbit instance.
func New() *Orbit {
	return &Orbit{}
}

// Metadata returns the orbit's metadata.
func (o *Orbit) Metadata() sdk.Metadata {
	return sdk.Metadata{
		ID:          OrbitID,
		Name:        "Ideal Week Designer",
		Version:     OrbitVersion,
		Author:      "Orbita",
		Description: "Design and maintain your ideal week template for better time management",
		License:     "Proprietary",
		Homepage:    "https://orbita.app/orbits/idealweek",
		Tags:        []string{"scheduling", "planning", "time-management", "productivity"},
	}
}

// RequiredCapabilities returns the capabilities required by this orbit.
func (o *Orbit) RequiredCapabilities() []sdk.Capability {
	return []sdk.Capability{
		sdk.CapReadSchedule,    // To compare actual schedule with ideal
		sdk.CapReadTasks,       // To see task distribution
		sdk.CapReadStorage,     // To read ideal week templates
		sdk.CapWriteStorage,    // To save ideal week templates
		sdk.CapSubscribeEvents, // To react to schedule changes
		sdk.CapRegisterTools,   // To register MCP tools
	}
}

// Initialize initializes the orbit with the provided context.
func (o *Orbit) Initialize(ctx sdk.Context) error {
	o.ctx = ctx
	o.ctx.Logger().Info("ideal week orbit initialized")
	return nil
}

// Shutdown gracefully shuts down the orbit.
func (o *Orbit) Shutdown(ctx context.Context) error {
	if o.ctx != nil {
		o.ctx.Logger().Info("ideal week orbit shutting down")
	}
	return nil
}

// RegisterTools registers MCP tools for the ideal week orbit.
func (o *Orbit) RegisterTools(registry sdk.ToolRegistry) error {
	return registerTools(registry, o)
}

// RegisterCommands registers CLI commands for the ideal week orbit.
func (o *Orbit) RegisterCommands(registry sdk.CommandRegistry) error {
	// No CLI commands for now, all interaction via MCP tools
	return nil
}

// SubscribeEvents subscribes to domain events.
func (o *Orbit) SubscribeEvents(bus sdk.EventBus) error {
	// Subscribe to schedule events to track adherence
	if err := bus.Subscribe("scheduling.block.created", o.onBlockCreated); err != nil {
		return err
	}

	if err := bus.Subscribe("scheduling.block.completed", o.onBlockCompleted); err != nil {
		return err
	}

	return nil
}

// Event handlers

func (o *Orbit) onBlockCreated(ctx context.Context, event sdk.DomainEvent) error {
	o.ctx.Logger().Debug("schedule block created event received",
		"event_type", event.Type,
		"payload", event.Payload,
	)
	// Could track when blocks are created vs ideal week expectations
	return nil
}

func (o *Orbit) onBlockCompleted(ctx context.Context, event sdk.DomainEvent) error {
	o.ctx.Logger().Debug("schedule block completed event received",
		"event_type", event.Type,
		"payload", event.Payload,
	)
	// Could track adherence to ideal week
	return nil
}

// Context returns the orbit's context (for tools).
func (o *Orbit) Context() sdk.Context {
	return o.ctx
}
