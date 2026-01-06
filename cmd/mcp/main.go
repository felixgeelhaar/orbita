package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/felixgeelhaar/orbita/internal/app"
	mcpinternal "github.com/felixgeelhaar/orbita/internal/mcp"
	"github.com/felixgeelhaar/orbita/pkg/config"
	"github.com/google/uuid"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.IsDevelopment() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	container, err := app.NewContainer(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize container", "error", err)
		os.Exit(1)
	}
	defer container.Close()

	userID, err := uuid.Parse(cfg.UserID)
	if err != nil {
		logger.Error("invalid ORBITA_USER_ID", "error", err)
		os.Exit(1)
	}

	cliApp := mcpinternal.NewCLIApp(container, userID)

	if err := mcpinternal.Serve(ctx, cfg, cliApp, container.AuthService, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("mcp server error", "error", err)
		os.Exit(1)
	}
}
