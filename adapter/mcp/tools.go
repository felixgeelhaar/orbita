package mcp

import (
	"errors"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/adapter/cli"
	identityOAuth "github.com/felixgeelhaar/orbita/internal/identity/application/oauth"
	"github.com/felixgeelhaar/orbita/internal/orbit/registry"
	"github.com/felixgeelhaar/orbita/internal/orbit/runtime"
	"github.com/google/uuid"
)

// ToolDependencies provides handlers and context for MCP tools.
type ToolDependencies struct {
	App         *cli.App
	AuthService *identityOAuth.Service

	// Orbit system dependencies (optional)
	OrbitRegistry *registry.Registry
	OrbitSandbox  *runtime.Sandbox
	OrbitExecutor *runtime.Executor
	DefaultUserID uuid.UUID // Default user ID for orbit tool execution
}

// RegisterCLITools registers MCP tools that mirror CLI functionality.
func RegisterCLITools(srv *mcp.Server, deps ToolDependencies) error {
	if srv == nil {
		return errors.New("server is required")
	}
	if deps.App == nil {
		return errors.New("app is required")
	}

	if err := registerCoreTools(srv, deps); err != nil {
		return err
	}
	if err := registerTaskTools(srv, deps); err != nil {
		return err
	}
	if err := registerHabitTools(srv, deps); err != nil {
		return err
	}
	if err := registerMeetingTools(srv, deps); err != nil {
		return err
	}
	if err := registerScheduleTools(srv, deps); err != nil {
		return err
	}
	if err := registerInboxTools(srv, deps); err != nil {
		return err
	}
	if err := registerBillingTools(srv, deps); err != nil {
		return err
	}
	if err := registerSettingsTools(srv, deps); err != nil {
		return err
	}
	if err := registerAuthTools(srv, deps); err != nil {
		return err
	}
	if err := registerEngineTools(srv, deps); err != nil {
		return err
	}
	if err := registerInsightsTools(srv, deps); err != nil {
		return err
	}
	if err := registerAutomationTools(srv, deps); err != nil {
		return err
	}
	if err := registerSearchTools(srv, deps); err != nil {
		return err
	}
	if err := registerIdealWeekTools(srv, deps); err != nil {
		return err
	}
	if err := registerWellnessTools(srv, deps); err != nil {
		return err
	}
	if err := registerCalendarTools(srv, deps); err != nil {
		return err
	}

	// Register orbit tools if orbit system is configured
	if deps.OrbitRegistry != nil {
		orbitDeps := OrbitDependencies{
			Registry: deps.OrbitRegistry,
			Sandbox:  deps.OrbitSandbox,
			Executor: deps.OrbitExecutor,
			UserID:   deps.DefaultUserID,
		}
		if err := registerOrbitTools(srv, deps, orbitDeps); err != nil {
			// Log warning but don't fail - orbits are optional
			// This allows the MCP server to start even if orbit registration fails
		}
	}

	return nil
}
