package app

import (
	"context"
	"fmt"
	"log/slog"

	billingApp "github.com/felixgeelhaar/orbita/internal/billing/application"
	billingPersistence "github.com/felixgeelhaar/orbita/internal/billing/infrastructure/persistence"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	marketplaceDomain "github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	marketplaceQueries "github.com/felixgeelhaar/orbita/internal/marketplace/application/queries"
	marketplacePersistence "github.com/felixgeelhaar/orbita/internal/marketplace/infrastructure/persistence"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	"github.com/felixgeelhaar/orbita/internal/engine/builtin"
	"github.com/felixgeelhaar/orbita/internal/engine/registry"
	"github.com/felixgeelhaar/orbita/internal/engine/runtime"
	orbitAPI "github.com/felixgeelhaar/orbita/internal/orbit/api"
	"github.com/felixgeelhaar/orbita/internal/orbit/builtin/focusmode"
	"github.com/felixgeelhaar/orbita/internal/orbit/builtin/idealweek"
	"github.com/felixgeelhaar/orbita/internal/orbit/builtin/wellness"
	orbitRegistry "github.com/felixgeelhaar/orbita/internal/orbit/registry"
	orbitRuntime "github.com/felixgeelhaar/orbita/internal/orbit/runtime"
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
	"github.com/redis/go-redis/v9"
)

// Container holds all application dependencies.
type Container struct {
	Config *config.Config
	Logger *slog.Logger

	// Database
	DB *pgxpool.Pool

	// Redis
	RedisClient *redis.Client

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

	// Orbit SDK
	OrbitRegistry *orbitRegistry.Registry
	OrbitSandbox  *orbitRuntime.Sandbox
	OrbitExecutor *orbitRuntime.Executor

	// Marketplace
	MarketplacePackageRepo   marketplaceDomain.PackageRepository
	MarketplaceVersionRepo   marketplaceDomain.VersionRepository
	MarketplacePublisherRepo marketplaceDomain.PublisherRepository
	ListMarketplacePackages  *marketplaceQueries.ListPackagesHandler
	SearchMarketplacePackages *marketplaceQueries.SearchPackagesHandler
	GetMarketplacePackage    *marketplaceQueries.GetPackageHandler
	GetMarketplaceFeatured   *marketplaceQueries.GetFeaturedHandler
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

	// Connect to Redis (optional in development)
	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			if !cfg.IsDevelopment() {
				pool.Close()
				return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
			}
			logger.Warn("invalid Redis URL, orbit storage will use in-memory fallback", "error", err)
		} else {
			redisClient := redis.NewClient(opt)
			if err := redisClient.Ping(ctx).Err(); err != nil {
				if !cfg.IsDevelopment() {
					pool.Close()
					return nil, fmt.Errorf("failed to connect to Redis: %w", err)
				}
				logger.Warn("Redis not available, orbit storage will use in-memory fallback", "error", err)
			} else {
				c.RedisClient = redisClient
				logger.Info("connected to Redis")
			}
		}
	}

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

	// Create marketplace repositories
	c.MarketplacePackageRepo = marketplacePersistence.NewPostgresPackageRepository(pool)
	c.MarketplaceVersionRepo = marketplacePersistence.NewPostgresVersionRepository(pool)
	c.MarketplacePublisherRepo = marketplacePersistence.NewPostgresPublisherRepository(pool)

	// Create marketplace query handlers
	c.ListMarketplacePackages = marketplaceQueries.NewListPackagesHandler(c.MarketplacePackageRepo)
	c.SearchMarketplacePackages = marketplaceQueries.NewSearchPackagesHandler(c.MarketplacePackageRepo)
	c.GetMarketplacePackage = marketplaceQueries.NewGetPackageHandler(c.MarketplacePackageRepo, c.MarketplaceVersionRepo, c.MarketplacePublisherRepo)
	c.GetMarketplaceFeatured = marketplaceQueries.NewGetFeaturedHandler(c.MarketplacePackageRepo)

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

	// Create orbit registry and register built-in orbits
	c.OrbitRegistry = orbitRegistry.NewRegistry(logger, c.BillingService)

	// Register built-in orbits
	wellnessOrbit := wellness.New()
	if err := c.OrbitRegistry.RegisterBuiltin(wellnessOrbit); err != nil {
		logger.Warn("failed to register wellness orbit", "error", err)
	}

	idealweekOrbit := idealweek.New()
	if err := c.OrbitRegistry.RegisterBuiltin(idealweekOrbit); err != nil {
		logger.Warn("failed to register ideal week orbit", "error", err)
	}

	focusmodeOrbit := focusmode.New()
	if err := c.OrbitRegistry.RegisterBuiltin(focusmodeOrbit); err != nil {
		logger.Warn("failed to register focus mode orbit", "error", err)
	}

	// Discover and load orbits from filesystem
	orbitSearchPaths := cfg.OrbitSearchPaths
	if len(orbitSearchPaths) == 0 {
		// Use default search paths if none configured
		orbitSearchPaths = orbitRegistry.DefaultOrbitSearchPaths()
	}

	orbitDiscovery := orbitRegistry.NewDiscovery(orbitSearchPaths, logger)
	discoveredOrbits, err := orbitDiscovery.Discover()
	if err != nil {
		logger.Warn("orbit discovery failed", "error", err)
	} else if len(discoveredOrbits) > 0 {
		logger.Info("discovered orbits from filesystem",
			"count", len(discoveredOrbits),
			"paths", orbitSearchPaths,
		)
		for _, discovered := range discoveredOrbits {
			// Check if already registered (built-in takes precedence)
			if c.OrbitRegistry.Has(discovered.Manifest.ID) {
				logger.Debug("skipping discovered orbit, already registered",
					"orbit_id", discovered.Manifest.ID,
				)
				continue
			}

			// Register the manifest for filesystem orbits
			// Note: Filesystem orbits use manifest-only registration since they
			// don't have in-process implementations. The Orbit interface implementation
			// would need to be loaded via a plugin mechanism (future enhancement).
			if err := c.OrbitRegistry.RegisterManifest(discovered.Manifest, discovered.Path); err != nil {
				logger.Warn("failed to register discovered orbit",
					"orbit_id", discovered.Manifest.ID,
					"path", discovered.Path,
					"error", err,
				)
			} else {
				logger.Info("registered discovered orbit",
					"orbit_id", discovered.Manifest.ID,
					"path", discovered.Path,
				)
			}
		}
	}

	// Create API factories for orbit sandbox
	apiFactories := &orbitAPI.APIFactories{
		TaskHandler:     c.ListTasksHandler,
		HabitHandler:    c.ListHabitsHandler,
		ScheduleHandler: c.GetScheduleHandler,
		MeetingHandler:  c.ListMeetingsHandler,
		InboxHandler:    c.ListInboxItemsHandler,
		RedisClient:     c.RedisClient, // nil in development mode (uses in-memory storage)
	}

	// Create orbit sandbox with full API factory integration
	c.OrbitSandbox = orbitRuntime.NewSandbox(orbitRuntime.SandboxConfig{
		Logger:             logger,
		Registry:           c.OrbitRegistry,
		TaskAPIFactory:     apiFactories.TaskAPIFactory(),
		HabitAPIFactory:    apiFactories.HabitAPIFactory(),
		ScheduleAPIFactory: apiFactories.ScheduleAPIFactory(),
		MeetingAPIFactory:  apiFactories.MeetingAPIFactory(),
		InboxAPIFactory:    apiFactories.InboxAPIFactory(),
		StorageAPIFactory:  apiFactories.StorageAPIFactory(),
		MetricsFactory:     orbitAPI.NoopMetricsFactory(),
	})

	// Create orbit executor
	c.OrbitExecutor = orbitRuntime.NewExecutor(orbitRuntime.ExecutorConfig{
		Sandbox:  c.OrbitSandbox,
		Registry: c.OrbitRegistry,
		Logger:   logger,
	})

	logger.Info("registered orbits",
		"wellness", wellness.OrbitID,
		"idealweek", idealweek.OrbitID,
		"focusmode", focusmode.OrbitID,
	)

	return c, nil
}

// Close cleans up all resources.
func (c *Container) Close() {
	// Shutdown all orbits via registry
	if c.OrbitRegistry != nil {
		ctx := context.Background()
		if err := c.OrbitRegistry.Shutdown(ctx); err != nil {
			c.Logger.Warn("error shutting down orbits", "error", err)
		}
	}

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

	if c.RedisClient != nil {
		if err := c.RedisClient.Close(); err != nil {
			c.Logger.Warn("error closing Redis connection", "error", err)
		} else {
			c.Logger.Info("Redis connection closed")
		}
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
