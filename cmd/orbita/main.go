package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	cliAuth "github.com/felixgeelhaar/orbita/adapter/cli/auth"
	"github.com/felixgeelhaar/orbita/adapter/cli/automation"
	cliBilling "github.com/felixgeelhaar/orbita/adapter/cli/billing"
	"github.com/felixgeelhaar/orbita/adapter/cli/habit"
	"github.com/felixgeelhaar/orbita/adapter/cli/inbox"
	"github.com/felixgeelhaar/orbita/adapter/cli/insights"
	"github.com/felixgeelhaar/orbita/adapter/cli/license"
	"github.com/felixgeelhaar/orbita/adapter/cli/mcp"
	"github.com/felixgeelhaar/orbita/adapter/cli/meeting"
	"github.com/felixgeelhaar/orbita/adapter/cli/schedule"
	cliSettings "github.com/felixgeelhaar/orbita/adapter/cli/settings"
	"github.com/felixgeelhaar/orbita/adapter/cli/task"
	"github.com/felixgeelhaar/orbita/internal/app"
	"github.com/felixgeelhaar/orbita/pkg/config"
	"github.com/google/uuid"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// In development without .env, use defaults
		logger.Warn("failed to load config, using development mode", "error", err)
		cfg = &config.Config{AppEnv: "development"}
	}

	// Update logger level based on config
	if cfg.IsDevelopment() {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	cli.SetLogger(logger)

	// Initialize container based on mode
	var cliApp *cli.App
	var container *app.Container

	if cfg.IsLocalMode() {
		// Use SQLite local mode (zero-config, no external services)
		logger.Info("starting in local mode with SQLite", "database", cfg.SQLitePath)
		container, err = app.NewLocalContainer(ctx, cfg, logger)
		if err != nil {
			logger.Error("failed to initialize local container", "error", err)
			os.Exit(1)
		}
	} else {
		// Use full PostgreSQL mode with external services
		container, err = app.NewContainer(ctx, cfg, logger)
	}

	if err != nil {
		if cfg.IsDevelopment() {
			logger.Warn("failed to initialize container, running in limited mode", "error", err)
			// In development, allow CLI to run without database
			cliApp = nil
		} else {
			logger.Error("failed to initialize container", "error", err)
			os.Exit(1)
		}
	} else {
		defer container.Close()

		// Start outbox processor in background (optional in CLI, not available in local mode)
		if cfg.OutboxProcessorEnabled && container.OutboxProcessor != nil {
			go container.OutboxProcessor.Start(ctx)
		} else if container.OutboxProcessor == nil {
			logger.Debug("outbox processor not available in local mode")
		} else {
			logger.Info("outbox processor disabled in CLI")
		}

		// Start calendar import worker in background (for automatic external calendar sync)
		if container.CalendarImportWorker != nil {
			go container.CalendarImportWorker.Run(ctx)
			logger.Info("calendar import worker started")
		}

		// Create CLI app with handlers
		cliApp = cli.NewApp(
			container.CreateTaskHandler,
			container.CompleteTaskHandler,
			container.ArchiveTaskHandler,
			container.ListTasksHandler,
			container.CreateHabitHandler,
			container.LogCompletionHandler,
			container.ArchiveHabitHandler,
			container.AdjustHabitFrequencyHandler,
			container.ListHabitsHandler,
			container.CreateMeetingHandler,
			container.UpdateMeetingHandler,
			container.ArchiveMeetingHandler,
			container.MarkMeetingHeldHandler,
			container.AdjustMeetingCadenceHandler,
			container.ListMeetingsHandler,
			container.ListMeetingCandidatesHandler,
			container.AddBlockHandler,
			container.CompleteBlockHandler,
			container.RemoveBlockHandler,
			container.RescheduleBlockHandler,
			container.AutoScheduleHandler,
			container.AutoRescheduleHandler,
			container.GetScheduleHandler,
			container.FindAvailableSlotsHandler,
			container.ListRescheduleAttemptsHandler,
			container.CaptureInboxItemHandler,
			container.PromoteInboxItemHandler,
			container.ListInboxItemsHandler,
			container.BillingService,
		)

		userID, err := uuid.Parse(cfg.UserID)
		if err != nil {
			logger.Error("invalid ORBITA_USER_ID", "error", err)
			os.Exit(1)
		}
		cliApp.SetCurrentUserID(userID)

		if container.AuthService != nil {
			cliAuth.SetService(container.AuthService)
		}
		if container.CalendarSyncer != nil {
			cliApp.SetCalendarSyncer(container.CalendarSyncer)
		}

		// Wire calendar multi-provider infrastructure
		if container.ConnectedCalendarRepo != nil {
			cliAuth.SetCalendarRepo(container.ConnectedCalendarRepo)
			cliApp.SetCalendarRepo(container.ConnectedCalendarRepo)
		}
		if container.ProviderRegistry != nil {
			cliAuth.SetProviderRegistry(container.ProviderRegistry)
			cliApp.SetProviderRegistry(container.ProviderRegistry)
		}
		if container.SyncCoordinator != nil {
			cliAuth.SetSyncCoordinator(container.SyncCoordinator)
			cliApp.SetSyncCoordinator(container.SyncCoordinator)
		}
		if container.SettingsService != nil {
			cliApp.SetSettingsService(container.SettingsService)
		}
		if container.BillingService != nil {
			cliApp.SetBillingService(container.BillingService)
		}
		if container.EngineRegistry != nil {
			cliApp.SetEngineRegistry(container.EngineRegistry)
		}
		if container.EngineExecutor != nil {
			cliApp.SetEngineExecutor(container.EngineExecutor)
		}
		if container.AutomationService != nil {
			cliApp.SetAutomationService(container.AutomationService)
		}
		if container.InsightsService != nil {
			insights.SetService(container.InsightsService)
			cliApp.SetInsightsService(container.InsightsService)
		}

		// Set license service for local mode
		if container.LicenseService != nil {
			license.SetLicenseService(container.LicenseService)
		}
	}

	// Set the CLI app
	cli.SetApp(cliApp)

	// Register commands
	cli.AddCommand(task.Cmd)
	cli.AddCommand(habit.Cmd)
	cli.AddCommand(inbox.Cmd)
	cli.AddCommand(meeting.Cmd)
	cli.AddCommand(mcp.Cmd)
	cli.AddCommand(schedule.Cmd)
	cli.AddCommand(cliBilling.Cmd)
	cli.AddCommand(cliAuth.Cmd)
	cli.AddCommand(cliSettings.Cmd)
	cli.AddCommand(automation.Cmd)
	cli.AddCommand(insights.Cmd)
	cli.AddCommand(license.Cmd)
	cli.AddCommand(license.UpgradeCmd) // Also add at root level for convenience

	// Execute CLI
	cli.Execute()
}
