package mcp

import (
	"context"
	"errors"
	"log/slog"

	mcpgo "github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/mcp-go/middleware"
	"github.com/felixgeelhaar/orbita/adapter/cli"
	mcplocal "github.com/felixgeelhaar/orbita/adapter/mcp"
	identityOAuth "github.com/felixgeelhaar/orbita/internal/identity/application/oauth"
	"github.com/felixgeelhaar/orbita/pkg/config"
)

// Serve starts an MCP server that mirrors CLI behavior and blocks until the context is canceled.
func Serve(ctx context.Context, cfg *config.Config, cliApp *cli.App, authService *identityOAuth.Service, logger *slog.Logger) error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if cliApp == nil {
		return errors.New("CLI app is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	srv := mcpgo.NewServer(mcpgo.ServerInfo{
		Name:    "orbita-mcp",
		Version: "1.0.0",
		Capabilities: mcpgo.Capabilities{
			Tools:     true,
			Resources: true,
			Prompts:   true,
		},
	})

	deps := mcplocal.ToolDependencies{
		App:         cliApp,
		AuthService: authService,
	}

	// Register CLI tools
	if err := mcplocal.RegisterCLITools(srv, deps); err != nil {
		return err
	}

	// Register MCP resources
	if err := mcplocal.RegisterResources(srv, deps); err != nil {
		logger.Warn("failed to register MCP resources", "error", err)
		// Continue - resources are optional enhancements
	}

	// Register MCP prompts
	if err := mcplocal.RegisterPrompts(srv, deps); err != nil {
		logger.Warn("failed to register MCP prompts", "error", err)
		// Continue - prompts are optional enhancements
	}

	adapter := mcpLogger{logger: logger}
	stack := middleware.DefaultStack(adapter)

	if cfg.MCPAuthToken != "" {
		authenticator := middleware.BearerTokenAuthenticator(middleware.StaticTokens(map[string]*middleware.Identity{
			cfg.MCPAuthToken: {ID: "mcp", Name: "mcp"},
		}))
		stack = append([]middleware.Middleware{middleware.Auth(authenticator, middleware.WithAuthLogger(adapter))}, stack...)
	} else {
		logger.Warn("MCP auth token not set; requests will be unauthenticated")
	}

	logger.Info("mcp server listening", "addr", cfg.MCPAddr)
	return mcpgo.ServeHTTPWithMiddleware(ctx, srv, cfg.MCPAddr, nil, mcpgo.WithMiddleware(stack...))
}

type mcpLogger struct {
	logger *slog.Logger
}

func (l mcpLogger) Info(msg string, fields ...middleware.Field) {
	l.logger.Info(msg, fieldsToArgs(fields)...)
}

func (l mcpLogger) Error(msg string, fields ...middleware.Field) {
	l.logger.Error(msg, fieldsToArgs(fields)...)
}

func (l mcpLogger) Debug(msg string, fields ...middleware.Field) {
	l.logger.Debug(msg, fieldsToArgs(fields)...)
}

func (l mcpLogger) Warn(msg string, fields ...middleware.Field) {
	l.logger.Warn(msg, fieldsToArgs(fields)...)
}

func fieldsToArgs(fields []middleware.Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	return args
}
