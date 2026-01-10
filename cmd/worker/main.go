package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/felixgeelhaar/orbita/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting orbita worker")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Update logger level based on config
	if cfg.IsDevelopment() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	// Connect to database
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database")

	// Create outbox repository
	outboxRepo := outbox.NewPostgresRepository(pool)

	// Create event publisher
	var publisher eventbus.Publisher
	rabbitPublisher, err := eventbus.NewRabbitMQPublisher(cfg.RabbitMQURL, logger)
	if err != nil {
		if cfg.IsDevelopment() {
			logger.Warn("RabbitMQ not available, using noop publisher", "error", err)
			publisher = eventbus.NewNoopPublisher(logger)
		} else {
			logger.Error("failed to connect to RabbitMQ", "error", err)
			os.Exit(1)
		}
	} else {
		publisher = rabbitPublisher
		defer rabbitPublisher.Close()
	}
	logger.Info("event publisher initialized")

	// Create outbox processor
	processorConfig := outbox.ProcessorConfig{
		PollInterval: cfg.OutboxPollInterval,
		BatchSize:    cfg.OutboxBatchSize,
		MaxRetries:   cfg.OutboxMaxRetries,
	}
	processor := outbox.NewProcessor(outboxRepo, publisher, processorConfig, logger)

	// Start processing
	logger.Info("starting outbox processor",
		"poll_interval", processorConfig.PollInterval,
		"batch_size", processorConfig.BatchSize,
		"max_retries", processorConfig.MaxRetries,
	)

	if err := processor.Start(ctx); err != nil {
		logger.Error("failed to start outbox processor", "error", err)
		os.Exit(1)
	}

	cleanupTicker := time.NewTicker(cfg.OutboxCleanupInterval)
	defer cleanupTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-cleanupTicker.C:
				deleted, err := outboxRepo.DeleteOld(ctx, cfg.OutboxRetentionDays)
				if err != nil {
					logger.Error("outbox cleanup failed", "error", err)
					continue
				}
				if deleted > 0 {
					logger.Info("outbox cleanup completed", "deleted", deleted, "retention_days", cfg.OutboxRetentionDays)
				}
			}
		}
	}()

	if cfg.WorkerHealthAddr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			stats := processor.GetStats()
			response := map[string]any{
				"status":            "ok",
				"running":           stats.IsRunning,
				"published":         stats.PublishedCount,
				"failed":            stats.FailedCount,
				"dead":              stats.DeadCount,
				"last_processed_at": stats.LastProcessedAt,
				"last_error_at":     stats.LastErrorAt,
				"last_error":        stats.LastError,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		})

		mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(checkCtx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status": "not_ready",
					"error":  err.Error(),
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ready"})
		})

		healthSrv := &http.Server{
			Addr:              cfg.WorkerHealthAddr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}

		go func() {
			logger.Info("health server starting", "addr", cfg.WorkerHealthAddr)
			if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("health server error", "error", err)
			}
		}()

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := healthSrv.Shutdown(shutdownCtx); err != nil {
				logger.Warn("health server shutdown error", "error", err)
			}
		}()
	}

	statsTicker := time.NewTicker(cfg.OutboxStatsInterval)
	defer statsTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-statsTicker.C:
				stats := processor.GetStats()
				logger.Info("outbox stats",
					"running", stats.IsRunning,
					"published", stats.PublishedCount,
					"failed", stats.FailedCount,
					"dead", stats.DeadCount,
					"lag_seconds", stats.LagSeconds,
					"oldest_message_at", stats.OldestMessageAt,
					"last_processed_at", stats.LastProcessedAt,
					"last_error_at", stats.LastErrorAt,
					"last_error", stats.LastError,
				)
			}
		}
	}()

	// Wait for shutdown
	<-ctx.Done()
	logger.Info("shutting down worker")

	processor.Stop()
	logger.Info("worker stopped")

	fmt.Println("Goodbye!")
}
