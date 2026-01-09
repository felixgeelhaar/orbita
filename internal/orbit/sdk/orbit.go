// Package sdk defines the core interfaces for Orbita's Orbit module system.
// Orbits are feature modules that extend Orbita's capabilities through
// capability-restricted in-process execution.
package sdk

import (
	"context"
)

// Orbit defines the interface that all orbit modules must implement.
// Orbits can register MCP tools, CLI commands, and subscribe to events,
// but cannot add new domain entities or access the database directly.
type Orbit interface {
	// Metadata returns the orbit's identity and version information.
	Metadata() Metadata

	// RequiredCapabilities returns the list of capabilities this orbit needs.
	// These are validated at load time against the manifest declaration.
	RequiredCapabilities() []Capability

	// Initialize is called when the orbit is loaded. The context provides
	// sandboxed access to domain data and orbit-specific storage.
	Initialize(ctx Context) error

	// Shutdown is called when the orbit is being unloaded.
	Shutdown(ctx context.Context) error

	// RegisterTools registers MCP tools with the tool registry.
	// Tool names are automatically namespaced with the orbit ID.
	RegisterTools(registry ToolRegistry) error

	// RegisterCommands registers CLI commands with the command registry.
	// Commands appear under: orbita <orbit-name> <command>
	RegisterCommands(registry CommandRegistry) error

	// SubscribeEvents sets up event subscriptions for this orbit.
	SubscribeEvents(bus EventBus) error
}

// ToolRegistry allows orbits to register MCP tools.
type ToolRegistry interface {
	// RegisterTool registers a new MCP tool.
	// The tool name will be prefixed with the orbit ID: {orbit_id}.{name}
	RegisterTool(name string, handler ToolHandler, schema ToolSchema) error
}

// ToolHandler is the function signature for MCP tool handlers.
type ToolHandler func(ctx context.Context, input map[string]any) (any, error)

// ToolSchema defines the JSON schema for a tool's input parameters.
type ToolSchema struct {
	Description string                    `json:"description"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
	Required    []string                  `json:"required,omitempty"`
}

// PropertySchema defines a single property in a tool schema.
type PropertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []any    `json:"enum,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
}

// CommandRegistry allows orbits to register CLI commands.
type CommandRegistry interface {
	// RegisterCommand registers a CLI command.
	// Commands are added under: orbita <orbit-name> <command>
	RegisterCommand(name string, handler CommandHandler, config CommandConfig) error
}

// CommandHandler is the function signature for CLI command handlers.
type CommandHandler func(ctx context.Context, args []string, flags map[string]string) error

// CommandConfig defines configuration for a CLI command.
type CommandConfig struct {
	Short string
	Long  string
	Args  []ArgConfig
	Flags []FlagConfig
}

// ArgConfig defines a positional argument.
type ArgConfig struct {
	Name     string
	Required bool
}

// FlagConfig defines a command flag.
type FlagConfig struct {
	Name      string
	Short     string
	Usage     string
	Default   string
	Required  bool
	IsBool    bool
}

// EventBus allows orbits to subscribe to domain events and publish orbit-specific events.
type EventBus interface {
	// Subscribe registers a handler for a specific event type.
	// Event types follow the pattern: domain.entity.action (e.g., "tasks.task.completed")
	Subscribe(eventType string, handler EventHandler) error

	// Publish publishes an orbit-specific event.
	// Event types are automatically prefixed with the orbit ID.
	Publish(ctx context.Context, event OrbitEvent) error
}

// EventHandler processes domain events.
type EventHandler func(ctx context.Context, event DomainEvent) error

// DomainEvent represents an event from the core domain.
type DomainEvent struct {
	Type      string         `json:"type"`
	Timestamp int64          `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

// OrbitEvent represents an event published by an orbit.
type OrbitEvent struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}
