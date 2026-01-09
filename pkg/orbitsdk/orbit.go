// Package orbitsdk provides the public SDK for developing Orbita orbit modules.
// Orbits are capability-restricted feature modules that extend Orbita's functionality
// without modifying core domain entities.
//
// # Getting Started
//
// To create an orbit, implement the Orbit interface:
//
//	type MyOrbit struct {
//		ctx orbitsdk.Context
//	}
//
//	func (o *MyOrbit) Metadata() orbitsdk.Metadata {
//		return orbitsdk.Metadata{
//			ID:      "mycompany.myorbit",
//			Name:    "My Orbit",
//			Version: "1.0.0",
//		}
//	}
//
//	func (o *MyOrbit) RequiredCapabilities() []orbitsdk.Capability {
//		return []orbitsdk.Capability{
//			orbitsdk.CapReadTasks,
//			orbitsdk.CapWriteStorage,
//		}
//	}
//
//	func (o *MyOrbit) Initialize(ctx orbitsdk.Context) error {
//		o.ctx = ctx
//		return nil
//	}
//
// # Capabilities
//
// Orbits declare their required capabilities in the manifest. Available capabilities:
//   - read:tasks, read:habits, read:schedule, read:meetings, read:inbox - Domain data access
//   - read:storage, write:storage - Orbit-specific key-value storage
//   - subscribe:events, publish:events - Event bus access
//   - register:tools, register:commands - Extension registration
//
// # Extension Points
//
// Orbits can register MCP tools and CLI commands through the provided registries.
// All tool names are automatically namespaced with the orbit ID.
package orbitsdk

import (
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
)

// Re-export core types from internal SDK

// Orbit defines the interface that all orbit modules must implement.
type Orbit = sdk.Orbit

// Context provides the sandboxed runtime environment for orbits.
type Context = sdk.Context

// Metadata contains orbit identity and version information.
type Metadata = sdk.Metadata

// Capability represents a permission that an orbit can request.
type Capability = sdk.Capability

// CapabilitySet is a set of capabilities for efficient lookup.
type CapabilitySet = sdk.CapabilitySet

// Re-export capability constants
const (
	CapReadTasks       = sdk.CapReadTasks
	CapReadHabits      = sdk.CapReadHabits
	CapReadSchedule    = sdk.CapReadSchedule
	CapReadMeetings    = sdk.CapReadMeetings
	CapReadInbox       = sdk.CapReadInbox
	CapReadUser        = sdk.CapReadUser
	CapWriteStorage    = sdk.CapWriteStorage
	CapReadStorage     = sdk.CapReadStorage
	CapSubscribeEvents = sdk.CapSubscribeEvents
	CapPublishEvents   = sdk.CapPublishEvents
	CapRegisterTools   = sdk.CapRegisterTools
	CapRegisterCommands = sdk.CapRegisterCommands
)

// Re-export capability functions
var (
	AllCapabilities      = sdk.AllCapabilities
	ValidCapabilities    = sdk.ValidCapabilities
	ValidateCapabilities = sdk.ValidateCapabilities
	ParseCapability      = sdk.ParseCapability
	ParseCapabilities    = sdk.ParseCapabilities
	NewCapabilitySet     = sdk.NewCapabilitySet
)

// Re-export extension point types

// ToolRegistry allows orbits to register MCP tools.
type ToolRegistry = sdk.ToolRegistry

// ToolHandler is the function signature for MCP tool handlers.
type ToolHandler = sdk.ToolHandler

// ToolSchema defines the JSON schema for a tool's input parameters.
type ToolSchema = sdk.ToolSchema

// PropertySchema defines a single property in a tool schema.
type PropertySchema = sdk.PropertySchema

// CommandRegistry allows orbits to register CLI commands.
type CommandRegistry = sdk.CommandRegistry

// CommandHandler is the function signature for CLI command handlers.
type CommandHandler = sdk.CommandHandler

// CommandConfig defines configuration for a CLI command.
type CommandConfig = sdk.CommandConfig

// ArgConfig defines a positional argument.
type ArgConfig = sdk.ArgConfig

// FlagConfig defines a command flag.
type FlagConfig = sdk.FlagConfig

// EventBus allows orbits to subscribe to domain events.
type EventBus = sdk.EventBus

// EventHandler processes domain events.
type EventHandler = sdk.EventHandler

// DomainEvent represents an event from the core domain.
type DomainEvent = sdk.DomainEvent

// OrbitEvent represents an event published by an orbit.
type OrbitEvent = sdk.OrbitEvent

// Re-export error types
var (
	ErrCapabilityNotGranted = sdk.ErrCapabilityNotGranted
	ErrInvalidCapability    = sdk.ErrInvalidCapability
	ErrOrbitNotFound        = sdk.ErrOrbitNotFound
	ErrOrbitAlreadyLoaded   = sdk.ErrOrbitAlreadyLoaded
	ErrOrbitNotEntitled     = sdk.ErrOrbitNotEntitled
	ErrCapabilityMismatch   = sdk.ErrCapabilityMismatch
	ErrMissingID            = sdk.ErrMissingID
)
