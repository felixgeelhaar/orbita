package app

import (
	"context"
	"fmt"
	"log/slog"

	billingApp "github.com/felixgeelhaar/orbita/internal/billing/application"
	billingPersistence "github.com/felixgeelhaar/orbita/internal/billing/infrastructure/persistence"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	"github.com/felixgeelhaar/orbita/internal/engine/builtin"
	"github.com/felixgeelhaar/orbita/internal/engine/registry"
	"github.com/felixgeelhaar/orbita/internal/engine/runtime"
	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	habitPersistence "github.com/felixgeelhaar/orbita/internal/habits/infrastructure/persistence"
	identityOAuth "github.com/felixgeelhaar/orbita/internal/identity/application/oauth"
	identitySettings "github.com/felixgeelhaar/orbita/internal/identity/application/settings"
	identityPersistence "github.com/felixgeelhaar/orbita/internal/identity/infrastructure/persistence"
	inboxCommands "github.com/felixgeelhaar/orbita/internal/inbox/application/commands"
	inboxQueries "github.com/felixgeelhaar/orbita/internal/inbox/application/queries"
	inboxPersistence "github.com/felixgeelhaar/orbita/internal/inbox/persistence"
	inboxServices "github.com/felixgeelhaar/orbita/internal/inbox/services"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	meetingPersistence "github.com/felixgeelhaar/orbita/internal/meetings/infrastructure/persistence"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/infrastructure/persistence"
	scheduleCommands "github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	schedulerServices "github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	schedulePersistence "github.com/felixgeelhaar/orbita/internal/scheduling/infrastructure/persistence"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	sharedCrypto "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/crypto"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/felixgeelhaar/orbita/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Container holds all application dependencies.
type Container struct {
	Config *config.Config
	Logger *slog.Logger

	// Database
	DB *pgxpool.Pool

	// Repositories
	TaskRepo              *persistence.PostgresTaskRepository
	HabitRepo             *habitPersistence.PostgresHabitRepository
	MeetingRepo           *meetingPersistence.PostgresMeetingRepository
	EntitlementRepo       *billingPersistence.PostgresEntitlementRepository
	SubscriptionRepo      *billingPersistence.PostgresSubscriptionRepository
	ScheduleRepo          *schedulePersistence.PostgresScheduleRepository
	RescheduleAttemptRepo *schedulePersistence.PostgresRescheduleAttemptRepository
	OAuthTokenRepo        *identityPersistence.OAuthTokenRepository
	SettingsRepo          *identityPersistence.SettingsRepository
	OutboxRepo            outbox.Repository

	// Publishers
	EventPublisher eventbus.Publisher

	// Unit of Work
	UnitOfWork sharedApplication.UnitOfWork

	// Task Command Handlers
	CreateTaskHandler   *commands.CreateTaskHandler
	CompleteTaskHandler *commands.CompleteTaskHandler
	ArchiveTaskHandler  *commands.ArchiveTaskHandler

	// Task Query Handlers
	ListTasksHandler *queries.ListTasksHandler

	// Habit Command Handlers
	CreateHabitHandler          *habitCommands.CreateHabitHandler
	LogCompletionHandler        *habitCommands.LogCompletionHandler
	ArchiveHabitHandler         *habitCommands.ArchiveHabitHandler
	AdjustHabitFrequencyHandler *habitCommands.AdjustHabitFrequencyHandler

	// Habit Query Handlers
	ListHabitsHandler *habitQueries.ListHabitsHandler

	// Meeting Command Handlers
	CreateMeetingHandler        *meetingCommands.CreateMeetingHandler
	UpdateMeetingHandler        *meetingCommands.UpdateMeetingHandler
	ArchiveMeetingHandler       *meetingCommands.ArchiveMeetingHandler
	MarkMeetingHeldHandler      *meetingCommands.MarkMeetingHeldHandler
	AdjustMeetingCadenceHandler *meetingCommands.AdjustMeetingCadenceHandler

	// Meeting Query Handlers
	ListMeetingsHandler          *meetingQueries.ListMeetingsHandler
	ListMeetingCandidatesHandler *meetingQueries.ListMeetingCandidatesHandler

	// Schedule Command Handlers
	AddBlockHandler        *scheduleCommands.AddBlockHandler
	CompleteBlockHandler   *scheduleCommands.CompleteBlockHandler
	RemoveBlockHandler     *scheduleCommands.RemoveBlockHandler
	RescheduleBlockHandler *scheduleCommands.RescheduleBlockHandler
	AutoScheduleHandler   *scheduleCommands.AutoScheduleHandler
	AutoRescheduleHandler *scheduleCommands.AutoRescheduleHandler

	// Scheduler Engine
	SchedulerEngine *schedulerServices.SchedulerEngine

	// Auth
	AuthService     *identityOAuth.Service
	SettingsService *identitySettings.Service
	BillingService  *billingApp.Service

	// Calendar Sync
	CalendarSyncer calendarApp.Syncer

	// Schedule Query Handlers
	GetScheduleHandler            *scheduleQueries.GetScheduleHandler
	FindAvailableSlotsHandler     *scheduleQueries.FindAvailableSlotsHandler
	ListRescheduleAttemptsHandler *scheduleQueries.ListRescheduleAttemptsHandler

	// Inbox
	InboxRepo               *inboxPersistence.PostgresInboxRepository
	InboxClassifier         *inboxServices.Classifier
	CaptureInboxItemHandler *inboxCommands.CaptureInboxItemHandler
	PromoteInboxItemHandler *inboxCommands.PromoteInboxItemHandler
	ListInboxItemsHandler   *inboxQueries.ListInboxItemsHandler

	// Outbox Processor
	OutboxProcessor *outbox.Processor

	// Engine SDK
	EngineRegistry *registry.Registry
	EngineExecutor *runtime.Executor
}

// NewContainer creates and wires all dependencies.
func NewContainer(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Container, error) {
	c := &Container{
		Config: cfg,
		Logger: logger,
	}

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	c.DB = pool
	logger.Info("connected to database")

	// Create repositories
	c.TaskRepo = persistence.NewPostgresTaskRepository(pool)
	c.HabitRepo = habitPersistence.NewPostgresHabitRepository(pool)
	c.MeetingRepo = meetingPersistence.NewPostgresMeetingRepository(pool)
	c.EntitlementRepo = billingPersistence.NewPostgresEntitlementRepository(pool)
	c.SubscriptionRepo = billingPersistence.NewPostgresSubscriptionRepository(pool)
	c.ScheduleRepo = schedulePersistence.NewPostgresScheduleRepository(pool)
	c.RescheduleAttemptRepo = schedulePersistence.NewPostgresRescheduleAttemptRepository(pool)
	c.OAuthTokenRepo = identityPersistence.NewOAuthTokenRepository(pool)
	c.SettingsRepo = identityPersistence.NewSettingsRepository(pool)
	c.OutboxRepo = outbox.NewPostgresRepository(pool)
	c.UnitOfWork = sharedPersistence.NewPostgresUnitOfWork(pool)
	c.InboxRepo = inboxPersistence.NewPostgresInboxRepository(pool)
	c.InboxClassifier = inboxServices.NewClassifier()

	// Create event publisher
	publisher, err := eventbus.NewRabbitMQPublisher(cfg.RabbitMQURL, logger)
	if err != nil {
		// Fall back to noop publisher in development
		if cfg.IsDevelopment() {
			logger.Warn("RabbitMQ not available, using noop publisher")
			c.EventPublisher = eventbus.NewNoopPublisher(logger)
		} else {
			pool.Close()
			return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
		}
	} else {
		c.EventPublisher = publisher
	}

	// Create task command handlers
	c.CreateTaskHandler = commands.NewCreateTaskHandler(c.TaskRepo, c.OutboxRepo, c.UnitOfWork)
	c.CompleteTaskHandler = commands.NewCompleteTaskHandler(c.TaskRepo, c.OutboxRepo, c.UnitOfWork)
	c.ArchiveTaskHandler = commands.NewArchiveTaskHandler(c.TaskRepo, c.OutboxRepo, c.UnitOfWork)

	// Create task query handlers
	c.ListTasksHandler = queries.NewListTasksHandler(c.TaskRepo)

	// Create habit command handlers
	c.CreateHabitHandler = habitCommands.NewCreateHabitHandler(c.HabitRepo, c.OutboxRepo, c.UnitOfWork)
	c.LogCompletionHandler = habitCommands.NewLogCompletionHandler(c.HabitRepo, c.OutboxRepo, c.UnitOfWork)
	c.ArchiveHabitHandler = habitCommands.NewArchiveHabitHandler(c.HabitRepo, c.OutboxRepo, c.UnitOfWork)
	c.AdjustHabitFrequencyHandler = habitCommands.NewAdjustHabitFrequencyHandler(c.HabitRepo, c.OutboxRepo, c.UnitOfWork)

	// Create habit query handlers
	c.ListHabitsHandler = habitQueries.NewListHabitsHandler(c.HabitRepo)

	// Create meeting command handlers
	c.CreateMeetingHandler = meetingCommands.NewCreateMeetingHandler(c.MeetingRepo, c.OutboxRepo, c.UnitOfWork)
	c.UpdateMeetingHandler = meetingCommands.NewUpdateMeetingHandler(c.MeetingRepo, c.OutboxRepo, c.UnitOfWork)
	c.ArchiveMeetingHandler = meetingCommands.NewArchiveMeetingHandler(c.MeetingRepo, c.OutboxRepo, c.UnitOfWork)
	c.MarkMeetingHeldHandler = meetingCommands.NewMarkMeetingHeldHandler(c.MeetingRepo, c.UnitOfWork)
	c.AdjustMeetingCadenceHandler = meetingCommands.NewAdjustMeetingCadenceHandler(c.MeetingRepo, c.OutboxRepo, c.UnitOfWork)

	// Create meeting query handlers
	c.ListMeetingsHandler = meetingQueries.NewListMeetingsHandler(c.MeetingRepo)
	c.ListMeetingCandidatesHandler = meetingQueries.NewListMeetingCandidatesHandler(c.MeetingRepo)

	// Create inbox handlers
	c.CaptureInboxItemHandler = inboxCommands.NewCaptureInboxItemHandler(c.InboxRepo, c.InboxClassifier, c.UnitOfWork)
	c.ListInboxItemsHandler = inboxQueries.NewListInboxItemsHandler(c.InboxRepo)
	c.PromoteInboxItemHandler = inboxCommands.NewPromoteInboxItemHandler(
		c.InboxRepo,
		c.CreateTaskHandler,
		c.CreateHabitHandler,
		c.CreateMeetingHandler,
	)

	// Create scheduler engine
	c.SchedulerEngine = schedulerServices.NewSchedulerEngine(schedulerServices.DefaultSchedulerConfig())

	// Create schedule command handlers
	c.AddBlockHandler = scheduleCommands.NewAddBlockHandler(c.ScheduleRepo, c.OutboxRepo, c.UnitOfWork)
	c.CompleteBlockHandler = scheduleCommands.NewCompleteBlockHandler(c.ScheduleRepo, c.OutboxRepo, c.UnitOfWork)
	c.RemoveBlockHandler = scheduleCommands.NewRemoveBlockHandler(c.ScheduleRepo, c.OutboxRepo, c.UnitOfWork)
	c.RescheduleBlockHandler = scheduleCommands.NewRescheduleBlockHandler(c.ScheduleRepo, c.OutboxRepo, c.UnitOfWork)
	c.AutoScheduleHandler = scheduleCommands.NewAutoScheduleHandler(c.ScheduleRepo, c.OutboxRepo, c.UnitOfWork, c.SchedulerEngine, logger)
	c.AutoRescheduleHandler = scheduleCommands.NewAutoRescheduleHandler(c.ScheduleRepo, c.RescheduleAttemptRepo, c.OutboxRepo, c.UnitOfWork, c.SchedulerEngine)

	// Create schedule query handlers
	c.GetScheduleHandler = scheduleQueries.NewGetScheduleHandler(c.ScheduleRepo)
	c.FindAvailableSlotsHandler = scheduleQueries.NewFindAvailableSlotsHandler(c.ScheduleRepo)
	c.ListRescheduleAttemptsHandler = scheduleQueries.NewListRescheduleAttemptsHandler(c.RescheduleAttemptRepo)

	// Create settings service
	c.SettingsService = identitySettings.NewService(c.SettingsRepo)
	c.BillingService = billingApp.NewService(c.EntitlementRepo, c.SubscriptionRepo)

	// Create auth service if configured
	scopes := identityOAuth.ScopesFromEnv(cfg.OAuthScopes)
	if cfg.OAuthProvider != "" && cfg.OAuthClientID != "" && cfg.OAuthClientSecret != "" && cfg.OAuthAuthURL != "" && cfg.OAuthTokenURL != "" && cfg.OAuthRedirectURL != "" {
		encrypter, err := sharedCrypto.NewAESGCMFromBase64Key(cfg.EncryptionKey)
		if err != nil {
			logger.Warn("auth encryption not configured", "error", err)
		} else {
			service, err := identityOAuth.NewService(
				cfg.OAuthProvider,
				cfg.OAuthClientID,
				cfg.OAuthClientSecret,
				cfg.OAuthAuthURL,
				cfg.OAuthTokenURL,
				cfg.OAuthRedirectURL,
				scopes,
				c.OAuthTokenRepo,
				encrypter,
			)
			if err != nil {
				logger.Warn("failed to initialize auth service", "error", err)
			} else {
				c.AuthService = service
			}
		}
	}

	// Create calendar syncer if provider is supported
	if c.AuthService != nil && cfg.OAuthProvider == "google" {
		syncer := googleCalendar.NewSyncer(c.AuthService, logger)
		if cfg.CalendarDeleteMissing {
			syncer.WithDeleteMissing(true)
		}
		if cfg.CalendarID != "" {
			syncer.WithCalendarID(cfg.CalendarID)
		}
		c.CalendarSyncer = syncer
	}

	// Create outbox processor
	processorConfig := outbox.ProcessorConfig{
		PollInterval: cfg.OutboxPollInterval,
		BatchSize:    cfg.OutboxBatchSize,
		MaxRetries:   cfg.OutboxMaxRetries,
	}
	c.OutboxProcessor = outbox.NewProcessor(c.OutboxRepo, c.EventPublisher, processorConfig, logger)

	// Create engine registry and register built-in engines
	c.EngineRegistry = registry.NewRegistry(logger)

	// Register built-in engines
	if err := c.EngineRegistry.RegisterBuiltin(builtin.NewDefaultSchedulerEngine()); err != nil {
		logger.Warn("failed to register default scheduler engine", "error", err)
	}
	if err := c.EngineRegistry.RegisterBuiltin(builtin.NewDefaultPriorityEngine()); err != nil {
		logger.Warn("failed to register default priority engine", "error", err)
	}
	if err := c.EngineRegistry.RegisterBuiltin(builtin.NewDefaultClassifierEngine()); err != nil {
		logger.Warn("failed to register default classifier engine", "error", err)
	}
	if err := c.EngineRegistry.RegisterBuiltin(builtin.NewDefaultAutomationEngine()); err != nil {
		logger.Warn("failed to register default automation engine", "error", err)
	}

	// Create engine executor with circuit breaker
	executorConfig := runtime.DefaultExecutorConfig()
	metricsCollector := runtime.NewMetricsCollector()
	c.EngineExecutor = runtime.NewExecutor(c.EngineRegistry, metricsCollector, logger, executorConfig)

	logger.Info("registered engines", "count", c.EngineRegistry.Count())

	return c, nil
}

// Close cleans up all resources.
func (c *Container) Close() {
	// Shutdown all engines via registry
	if c.EngineRegistry != nil {
		ctx := context.Background()
		if err := c.EngineRegistry.ShutdownAll(ctx); err != nil {
			c.Logger.Warn("error shutting down engines", "error", err)
		}
	}

	if c.OutboxProcessor != nil {
		c.OutboxProcessor.Stop()
	}

	if c.EventPublisher != nil {
		c.EventPublisher.Close()
	}

	if c.DB != nil {
		c.DB.Close()
		c.Logger.Info("database connection closed")
	}
}

// NewDevelopmentContainer creates a container for local development without external services.
func NewDevelopmentContainer(logger *slog.Logger) *Container {
	c := &Container{
		Config: &config.Config{AppEnv: "development"},
		Logger: logger,
	}

	// Use in-memory repositories
	c.OutboxRepo = outbox.NewInMemoryRepository()
	c.EventPublisher = eventbus.NewNoopPublisher(logger)

	// Note: TaskRepo requires a database connection
	// This container is useful for testing CLI structure without DB

	return c
}
