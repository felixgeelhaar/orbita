package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/internal/orbit/registry"
	"github.com/felixgeelhaar/orbita/internal/orbit/runtime"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
)

// OrbitToolBridge collects tools from orbits and registers them with the MCP server.
type OrbitToolBridge struct {
	mu      sync.RWMutex
	tools   map[string]*orbitToolEntry
	orbitID string
}

// orbitToolEntry holds a registered orbit tool.
type orbitToolEntry struct {
	name     string
	fullName string // {orbit_id}.{name}
	handler  sdk.ToolHandler
	schema   sdk.ToolSchema
}

// NewOrbitToolBridge creates a new orbit tool bridge for a specific orbit.
func NewOrbitToolBridge(orbitID string) *OrbitToolBridge {
	return &OrbitToolBridge{
		tools:   make(map[string]*orbitToolEntry),
		orbitID: orbitID,
	}
}

// RegisterTool implements sdk.ToolRegistry.
func (b *OrbitToolBridge) RegisterTool(name string, handler sdk.ToolHandler, schema sdk.ToolSchema) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	fullName := fmt.Sprintf("%s.%s", b.orbitID, name)

	if _, exists := b.tools[fullName]; exists {
		return fmt.Errorf("tool %s already registered", fullName)
	}

	b.tools[fullName] = &orbitToolEntry{
		name:     name,
		fullName: fullName,
		handler:  handler,
		schema:   schema,
	}

	return nil
}

// GetTools returns all registered tools.
func (b *OrbitToolBridge) GetTools() []*orbitToolEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	tools := make([]*orbitToolEntry, 0, len(b.tools))
	for _, tool := range b.tools {
		tools = append(tools, tool)
	}
	return tools
}

// OrbitDependencies holds dependencies for orbit tool registration.
type OrbitDependencies struct {
	Registry *registry.Registry
	Sandbox  *runtime.Sandbox
	Executor *runtime.Executor
	UserID   uuid.UUID // Default user ID for tools (can be overridden per-request)
}

// registerOrbitTools registers all tools from loaded orbits with the MCP server.
func registerOrbitTools(srv *mcp.Server, deps ToolDependencies, orbitDeps OrbitDependencies) error {
	if orbitDeps.Registry == nil {
		// No orbit registry configured, skip orbit tools
		return nil
	}

	// Get all loaded orbits
	entries := orbitDeps.Registry.List()

	for _, entry := range entries {
		if entry.Status != registry.StatusReady || entry.Orbit == nil {
			continue
		}

		orbitID := entry.Manifest.ID

		// Create a tool bridge for this orbit
		bridge := NewOrbitToolBridge(orbitID)

		// Let the orbit register its tools
		if err := entry.Orbit.RegisterTools(bridge); err != nil {
			// Log warning but continue with other orbits
			continue
		}

		// Register each tool with the MCP server
		for _, tool := range bridge.GetTools() {
			if err := registerOrbitToolWithMCP(srv, tool, orbitDeps); err != nil {
				// Log warning but continue
				continue
			}
		}
	}

	return nil
}

// registerOrbitToolWithMCP registers a single orbit tool with the MCP server.
func registerOrbitToolWithMCP(srv *mcp.Server, tool *orbitToolEntry, deps OrbitDependencies) error {
	// Create the MCP tool handler wrapper
	// Note: The MCP library infers schema from the input struct type.
	// For dynamic orbit tools, we use map[string]any which provides flexibility.
	// The description provides context about expected parameters.
	handler := tool.handler

	// Build a description that includes parameter info from the schema
	description := tool.schema.Description
	if len(tool.schema.Properties) > 0 {
		description += " Parameters: "
		first := true
		for name, prop := range tool.schema.Properties {
			if !first {
				description += ", "
			}
			first = false
			description += name
			if prop.Type != "" {
				description += " (" + prop.Type + ")"
			}
		}
	}

	srv.Tool(tool.fullName).
		Description(description).
		Handler(func(ctx context.Context, input map[string]any) (any, error) {
			return handler(ctx, input)
		})

	return nil
}

// RegisterOrbitToolsDynamic registers orbit tools dynamically as orbits are loaded.
// This should be called when the MCP server starts and whenever new orbits are loaded.
func RegisterOrbitToolsDynamic(srv *mcp.Server, deps OrbitDependencies) error {
	if deps.Registry == nil {
		return nil
	}

	// Get all ready orbits
	entries := deps.Registry.List()

	for _, entry := range entries {
		if entry.Status != registry.StatusReady || entry.Orbit == nil {
			continue
		}

		if err := RegisterOrbitToolsForOrbit(srv, entry.Orbit, deps); err != nil {
			// Log but continue
			continue
		}
	}

	return nil
}

// RegisterOrbitToolsForOrbit registers tools from a specific orbit.
func RegisterOrbitToolsForOrbit(srv *mcp.Server, orbit sdk.Orbit, deps OrbitDependencies) error {
	meta := orbit.Metadata()

	// Create a tool bridge for this orbit
	bridge := NewOrbitToolBridge(meta.ID)

	// Let the orbit register its tools
	if err := orbit.RegisterTools(bridge); err != nil {
		return fmt.Errorf("failed to register tools for orbit %s: %w", meta.ID, err)
	}

	// Register each tool with the MCP server
	for _, tool := range bridge.GetTools() {
		if err := registerOrbitToolWithMCP(srv, tool, deps); err != nil {
			return fmt.Errorf("failed to register MCP tool %s: %w", tool.fullName, err)
		}
	}

	return nil
}

// OrbitToolInfo provides information about a registered orbit tool.
type OrbitToolInfo struct {
	OrbitID     string   `json:"orbit_id"`
	ToolName    string   `json:"tool_name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	Properties  []string `json:"properties"`
	Required    []string `json:"required"`
}

// ListOrbitTools returns information about all registered orbit tools.
func ListOrbitTools(deps OrbitDependencies) []OrbitToolInfo {
	if deps.Registry == nil {
		return nil
	}

	var tools []OrbitToolInfo

	entries := deps.Registry.List()
	for _, entry := range entries {
		if entry.Status != registry.StatusReady || entry.Orbit == nil {
			continue
		}

		orbitID := entry.Manifest.ID
		bridge := NewOrbitToolBridge(orbitID)

		if err := entry.Orbit.RegisterTools(bridge); err != nil {
			continue
		}

		for _, tool := range bridge.GetTools() {
			props := make([]string, 0, len(tool.schema.Properties))
			for name := range tool.schema.Properties {
				props = append(props, name)
			}

			tools = append(tools, OrbitToolInfo{
				OrbitID:     orbitID,
				ToolName:    tool.name,
				FullName:    tool.fullName,
				Description: tool.schema.Description,
				Properties:  props,
				Required:    tool.schema.Required,
			})
		}
	}

	return tools
}
