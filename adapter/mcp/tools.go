package mcp

import (
	"errors"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/adapter/cli"
	identityOAuth "github.com/felixgeelhaar/orbita/internal/identity/application/oauth"
)

// ToolDependencies provides handlers and context for MCP tools.
type ToolDependencies struct {
	App         *cli.App
	AuthService *identityOAuth.Service
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

	return nil
}
