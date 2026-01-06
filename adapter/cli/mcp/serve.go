package mcp

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/felixgeelhaar/orbita/internal/app"
	mcpinternal "github.com/felixgeelhaar/orbita/internal/mcp"
	"github.com/felixgeelhaar/orbita/pkg/config"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		logger := newServerLogger(cmd.OutOrStdout(), cfg.IsDevelopment())

		container, err := app.NewContainer(ctx, cfg, logger)
		if err != nil {
			return err
		}
		defer container.Close()

		userID, err := uuid.Parse(cfg.UserID)
		if err != nil {
			return err
		}

		cliApp := mcpinternal.NewCLIApp(container, userID)
		err = mcpinternal.Serve(ctx, cfg, cliApp, container.AuthService, logger)
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	},
}

func newServerLogger(out io.Writer, debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{
		Level: level,
	}))
}
