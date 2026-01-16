package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	automationApp "github.com/felixgeelhaar/orbita/internal/automations/application"
	automationPersistence "github.com/felixgeelhaar/orbita/internal/automations/infrastructure/persistence"
	db "github.com/felixgeelhaar/orbita/db/generated/postgres"
	insightsApp "github.com/felixgeelhaar/orbita/internal/insights/application"
	insightsPersistence "github.com/felixgeelhaar/orbita/internal/insights/infrastructure/persistence"
	billingApp "github.com/felixgeelhaar/orbita/internal/billing/application"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	billingPersistence "github.com/felixgeelhaar/orbita/internal/billing/infrastructure/persistence"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	calendarSubs "github.com/felixgeelhaar/orbita/internal/calendar/application/subscribers"
	calendarWorkers "github.com/felixgeelhaar/orbita/internal/calendar/application/workers"
	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	calendarPersistence "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/persistence"
	calendarSetup "github.com/felixgeelhaar/orbita/internal/calendar/setup"
	licensingApp "github.com/felixgeelhaar/orbita/internal/licensing/application"
	licensingCrypto "github.com/felixgeelhaar/orbita/internal/licensing/infrastructure/crypto"
	licensingPersistence "github.com/felixgeelhaar/orbita/internal/licensing/infrastructure/persistence"
	marketplaceDomain "github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	marketplaceQueries "github.com/felixgeelhaar/orbita/internal/marketplace/application/queries"
	marketplacePersistence "github.com/felixgeelhaar/orbita/internal/marketplace/infrastructure/persistence"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	"github.com/felixgeelhaar/orbita/internal/engine/builtin"
	"github.com/felixgeelhaar/orbita/internal/engine/registry"
	"github.com/felixgeelhaar/orbita/internal/engine/runtime"
	habitsDomain "github.com/felixgeelhaar/orbita/internal/habits/domain"
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
	meetingsDomain "github.com/felixgeelhaar/orbita/internal/meetings/domain"
	meetingPersistence "github.com/felixgeelhaar/orbita/internal/meetings/infrastructure/persistence"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/productivity/infrastructure/persistence"
	scheduleCommands "github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	schedulerServices "github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	scheduleSubs "github.com/felixgeelhaar/orbita/internal/scheduling/application/subscribers"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	schedulePersistence "github.com/felixgeelhaar/orbita/internal/scheduling/infrastructure/persistence"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	sharedCrypto "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/crypto"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/database"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/migrations"
	_ "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/database/sqlite" // Register SQLite driver
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
	DB       *pgxpool.Pool
	DBConn   database.Connection // Abstract connection for driver-agnostic access
	DBDriver database.Driver

	// Redis
	RedisClient *redis.Client

	// Repositories (use interfaces for driver-agnostic access)
	TaskRepo              task.Repository
	HabitRepo             habitsDomain.Repository
	MeetingRepo           meetingsDomain.Repository
	EntitlementRepo       *billingPersistence.PostgresEntitlementRepository
	SubscriptionRepo      *billingPersistence.PostgresSubscriptionRepository
	ScheduleRepo          schedulingDomain.ScheduleRepository
	RescheduleAttemptRepo *schedulePersistence.PostgresRescheduleAttemptRepository
	OAuthTokenRepo        *identityPersistence.OAuthTokenRepository
	SettingsRepo          identitySettings.Repository
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
	GetTaskHandler   *queries.GetTaskHandler

	// Habit Command Handlers
	CreateHabitHandler          *habitCommands.CreateHabitHandler
	LogCompletionHandler        *habitCommands.LogCompletionHandler
	ArchiveHabitHandler         *habitCommands.ArchiveHabitHandler
	AdjustHabitFrequencyHandler *habitCommands.AdjustHabitFrequencyHandler

	// Habit Query Handlers
	ListHabitsHandler *habitQueries.ListHabitsHandler
	GetHabitHandler   *habitQueries.GetHabitHandler

	// Meeting Command Handlers
	CreateMeetingHandler        *meetingCommands.CreateMeetingHandler
	UpdateMeetingHandler        *meetingCommands.UpdateMeetingHandler
	ArchiveMeetingHandler       *meetingCommands.ArchiveMeetingHandler
	MarkMeetingHeldHandler      *meetingCommands.MarkMeetingHeldHandler
	AdjustMeetingCadenceHandler *meetingCommands.AdjustMeetingCadenceHandler

	// Meeting Query Handlers
	ListMeetingsHandler          *meetingQueries.ListMeetingsHandler
	GetMeetingHandler            *meetingQueries.GetMeetingHandler
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
	BillingService  billingDomain.BillingService

	// Licensing (local mode)
	LicenseService *licensingApp.Service

	// Calendar Sync
	CalendarSyncer       calendarApp.Syncer
	CalendarImporter     calendarApp.Importer
	SyncStateRepo        calendarDomain.SyncStateRepository
	CalendarImportWorker *calendarWorkers.CalendarImportWorker
	ConflictResolver     *schedulerServices.ConflictResolver
	ProviderRegistry     *calendarApp.ProviderRegistry
	SyncCoordinator      *calendarApp.SyncCoordinator
	ConnectedCalendarRepo calendarDomain.ConnectedCalendarRepository

	// Event Subscribers
	SchedulingSubscriber   *scheduleSubs.SchedulingSubscriber
	CalendarSyncSubscriber *calendarSubs.CalendarSyncSubscriber
	InProcessEventBus      *eventbus.InProcessEventBus

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
	GetInboxItemHandler     *inboxQueries.GetInboxItemHandler

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

	// Automations
	AutomationService *automationApp.Service

	// Insights
	InsightsService *insightsApp.Service
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
	c.TaskRepo = persistence.NewPostgresTaskRepositoryFromPool(pool)
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
	c.GetTaskHandler = queries.NewGetTaskHandler(c.TaskRepo)

	// Create habit command handlers
	c.CreateHabitHandler = habitCommands.NewCreateHabitHandler(c.HabitRepo, c.OutboxRepo, c.UnitOfWork)
	c.LogCompletionHandler = habitCommands.NewLogCompletionHandler(c.HabitRepo, c.OutboxRepo, c.UnitOfWork)
	c.ArchiveHabitHandler = habitCommands.NewArchiveHabitHandler(c.HabitRepo, c.OutboxRepo, c.UnitOfWork)
	c.AdjustHabitFrequencyHandler = habitCommands.NewAdjustHabitFrequencyHandler(c.HabitRepo, c.OutboxRepo, c.UnitOfWork)

	// Create habit query handlers
	c.ListHabitsHandler = habitQueries.NewListHabitsHandler(c.HabitRepo)
	c.GetHabitHandler = habitQueries.NewGetHabitHandler(c.HabitRepo)

	// Create meeting command handlers
	c.CreateMeetingHandler = meetingCommands.NewCreateMeetingHandler(c.MeetingRepo, c.OutboxRepo, c.UnitOfWork)
	c.UpdateMeetingHandler = meetingCommands.NewUpdateMeetingHandler(c.MeetingRepo, c.OutboxRepo, c.UnitOfWork)
	c.ArchiveMeetingHandler = meetingCommands.NewArchiveMeetingHandler(c.MeetingRepo, c.OutboxRepo, c.UnitOfWork)
	c.MarkMeetingHeldHandler = meetingCommands.NewMarkMeetingHeldHandler(c.MeetingRepo, c.UnitOfWork)
	c.AdjustMeetingCadenceHandler = meetingCommands.NewAdjustMeetingCadenceHandler(c.MeetingRepo, c.OutboxRepo, c.UnitOfWork)

	// Create meeting query handlers
	c.ListMeetingsHandler = meetingQueries.NewListMeetingsHandler(c.MeetingRepo)
	c.GetMeetingHandler = meetingQueries.NewGetMeetingHandler(c.MeetingRepo)
	c.ListMeetingCandidatesHandler = meetingQueries.NewListMeetingCandidatesHandler(c.MeetingRepo)

	// Create inbox handlers
	c.CaptureInboxItemHandler = inboxCommands.NewCaptureInboxItemHandler(c.InboxRepo, c.InboxClassifier, c.UnitOfWork)
	c.ListInboxItemsHandler = inboxQueries.NewListInboxItemsHandler(c.InboxRepo)
	c.GetInboxItemHandler = inboxQueries.NewGetInboxItemHandler(c.InboxRepo)
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

	// Create automation repositories and service
	automationQueries := db.New(pool)
	automationRuleRepo := automationPersistence.NewRuleRepository(automationQueries)
	automationExecRepo := automationPersistence.NewExecutionRepository(automationQueries)
	automationPendingRepo := automationPersistence.NewPendingActionRepository(automationQueries)
	c.AutomationService = automationApp.NewService(automationRuleRepo, automationExecRepo, automationPendingRepo)

	// Create insights repositories and service
	insightsQueries := db.New(pool)
	snapshotRepo := insightsPersistence.NewSnapshotRepository(insightsQueries)
	sessionRepo := insightsPersistence.NewSessionRepository(insightsQueries)
	summaryRepo := insightsPersistence.NewSummaryRepository(insightsQueries)
	goalRepo := insightsPersistence.NewGoalRepository(insightsQueries)
	analyticsDataSource := insightsPersistence.NewAnalyticsDataSource(insightsQueries)
	c.InsightsService = insightsApp.NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, analyticsDataSource)

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

	// Create connected calendar repository
	c.ConnectedCalendarRepo = calendarPersistence.NewPostgresConnectedCalendarRepository(pool)

	// Create provider registry and register available providers
	c.ProviderRegistry = calendarApp.NewProviderRegistry()
	providerConfig := calendarSetup.ProviderConfig{
		Logger: logger,
	}
	if c.AuthService != nil {
		providerConfig.GoogleOAuth = c.AuthService
	}
	calendarSetup.RegisterProviders(c.ProviderRegistry, providerConfig)

	// Create sync coordinator
	c.SyncCoordinator = calendarApp.NewSyncCoordinator(c.ProviderRegistry, c.ConnectedCalendarRepo)

	// Create calendar syncer if provider is supported (legacy single-provider mode)
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
		ListTaskHandler:    c.ListTasksHandler,
		GetTaskHandler:     c.GetTaskHandler,
		ListHabitHandler:   c.ListHabitsHandler,
		GetHabitHandler:    c.GetHabitHandler,
		ScheduleHandler:    c.GetScheduleHandler,
		ListMeetingHandler: c.ListMeetingsHandler,
		GetMeetingHandler:  c.GetMeetingHandler,
		ListInboxHandler:   c.ListInboxItemsHandler,
		GetInboxHandler:    c.GetInboxItemHandler,
		RedisClient:        c.RedisClient, // nil in development mode (uses in-memory storage)
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

	// Stop calendar import worker
	if c.CalendarImportWorker != nil && c.CalendarImportWorker.IsRunning() {
		c.CalendarImportWorker.Stop()
		c.Logger.Info("calendar import worker stopped")
	}

	if c.OutboxProcessor != nil {
		c.OutboxProcessor.Stop()
	}

	if c.EventPublisher != nil {
		if err := c.EventPublisher.Close(); err != nil {
			c.Logger.Warn("error closing event publisher", "error", err)
		}
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
		c.Logger.Info("PostgreSQL connection closed")
	}

	// Close SQLite connection if using local mode
	if c.DBConn != nil && c.DBDriver == database.DriverSQLite {
		if err := c.DBConn.Close(); err != nil {
			c.Logger.Warn("error closing SQLite connection", "error", err)
		} else {
			c.Logger.Info("SQLite connection closed")
		}
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

// NewLocalContainer creates a container for local mode with SQLite.
// This provides zero-config operation without requiring PostgreSQL, Redis, or RabbitMQ.
func NewLocalContainer(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Container, error) {
	c := &Container{
		Config: cfg,
		Logger: logger,
	}

	// Initialize SQLite database
	conn, err := initSQLiteConnection(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SQLite: %w", err)
	}

	// Create repository factory
	factory := NewRepositoryFactory(conn)

	// Create repositories using factory
	taskRepo, err := factory.TaskRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create task repository: %w", err)
	}
	c.TaskRepo = taskRepo

	habitRepo, err := factory.HabitRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create habit repository: %w", err)
	}
	c.HabitRepo = habitRepo

	meetingRepo, err := factory.MeetingRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create meeting repository: %w", err)
	}
	c.MeetingRepo = meetingRepo

	scheduleRepo, err := factory.ScheduleRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule repository: %w", err)
	}
	c.ScheduleRepo = scheduleRepo

	settingsRepo, err := factory.SettingsRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create settings repository: %w", err)
	}
	c.SettingsRepo = settingsRepo

	outboxRepo, err := factory.OutboxRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox repository: %w", err)
	}
	c.OutboxRepo = outboxRepo

	// Use noop publisher for local mode (no RabbitMQ)
	c.EventPublisher = eventbus.NewNoopPublisher(logger)

	// Create unit of work for SQLite
	c.UnitOfWork = sharedPersistence.NewSQLiteUnitOfWork(conn.DB())

	// Create task command handlers
	c.CreateTaskHandler = commands.NewCreateTaskHandler(taskRepo, outboxRepo, c.UnitOfWork)
	c.CompleteTaskHandler = commands.NewCompleteTaskHandler(taskRepo, outboxRepo, c.UnitOfWork)
	c.ArchiveTaskHandler = commands.NewArchiveTaskHandler(taskRepo, outboxRepo, c.UnitOfWork)

	// Create task query handlers
	c.ListTasksHandler = queries.NewListTasksHandler(taskRepo)
	c.GetTaskHandler = queries.NewGetTaskHandler(taskRepo)

	// Create habit command handlers
	c.CreateHabitHandler = habitCommands.NewCreateHabitHandler(habitRepo, outboxRepo, c.UnitOfWork)
	c.LogCompletionHandler = habitCommands.NewLogCompletionHandler(habitRepo, outboxRepo, c.UnitOfWork)
	c.ArchiveHabitHandler = habitCommands.NewArchiveHabitHandler(habitRepo, outboxRepo, c.UnitOfWork)
	c.AdjustHabitFrequencyHandler = habitCommands.NewAdjustHabitFrequencyHandler(habitRepo, outboxRepo, c.UnitOfWork)

	// Create habit query handlers
	c.ListHabitsHandler = habitQueries.NewListHabitsHandler(habitRepo)
	c.GetHabitHandler = habitQueries.NewGetHabitHandler(habitRepo)

	// Create meeting command handlers
	c.CreateMeetingHandler = meetingCommands.NewCreateMeetingHandler(meetingRepo, outboxRepo, c.UnitOfWork)
	c.UpdateMeetingHandler = meetingCommands.NewUpdateMeetingHandler(meetingRepo, outboxRepo, c.UnitOfWork)
	c.ArchiveMeetingHandler = meetingCommands.NewArchiveMeetingHandler(meetingRepo, outboxRepo, c.UnitOfWork)
	c.MarkMeetingHeldHandler = meetingCommands.NewMarkMeetingHeldHandler(meetingRepo, c.UnitOfWork)
	c.AdjustMeetingCadenceHandler = meetingCommands.NewAdjustMeetingCadenceHandler(meetingRepo, outboxRepo, c.UnitOfWork)

	// Create meeting query handlers
	c.ListMeetingsHandler = meetingQueries.NewListMeetingsHandler(meetingRepo)
	c.GetMeetingHandler = meetingQueries.NewGetMeetingHandler(meetingRepo)
	c.ListMeetingCandidatesHandler = meetingQueries.NewListMeetingCandidatesHandler(meetingRepo)

	// Create scheduler engine
	c.SchedulerEngine = schedulerServices.NewSchedulerEngine(schedulerServices.DefaultSchedulerConfig())

	// Create schedule command handlers
	c.AddBlockHandler = scheduleCommands.NewAddBlockHandler(scheduleRepo, outboxRepo, c.UnitOfWork)
	c.CompleteBlockHandler = scheduleCommands.NewCompleteBlockHandler(scheduleRepo, outboxRepo, c.UnitOfWork)
	c.RemoveBlockHandler = scheduleCommands.NewRemoveBlockHandler(scheduleRepo, outboxRepo, c.UnitOfWork)
	c.RescheduleBlockHandler = scheduleCommands.NewRescheduleBlockHandler(scheduleRepo, outboxRepo, c.UnitOfWork)
	c.AutoScheduleHandler = scheduleCommands.NewAutoScheduleHandler(scheduleRepo, outboxRepo, c.UnitOfWork, c.SchedulerEngine, logger)

	// Create schedule query handlers
	c.GetScheduleHandler = scheduleQueries.NewGetScheduleHandler(scheduleRepo)
	c.FindAvailableSlotsHandler = scheduleQueries.NewFindAvailableSlotsHandler(scheduleRepo)

	// Create conflict resolver
	conflictConfig := schedulerServices.ConflictResolverConfig{
		Strategy: schedulingDomain.ConflictResolutionStrategy(cfg.CalendarConflictStrategy),
	}
	c.ConflictResolver = schedulerServices.NewConflictResolver(scheduleRepo, c.SchedulerEngine, conflictConfig, logger)

	// Create sync state repository for SQLite
	c.SyncStateRepo = calendarPersistence.NewSQLiteSyncStateRepository(conn.DB())

	// Create connected calendar repository for SQLite
	connectedCalendarRepo, err := factory.ConnectedCalendarRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to create connected calendar repository: %w", err)
	}
	c.ConnectedCalendarRepo = connectedCalendarRepo

	// Create provider registry and register available providers
	c.ProviderRegistry = calendarApp.NewProviderRegistry()
	providerConfig := calendarSetup.ProviderConfig{
		Logger: logger,
	}
	// Note: OAuth providers will be registered when user configures them via CLI
	// For local mode, we don't auto-configure OAuth from environment
	calendarSetup.RegisterProviders(c.ProviderRegistry, providerConfig)

	// Create sync coordinator
	c.SyncCoordinator = calendarApp.NewSyncCoordinator(c.ProviderRegistry, c.ConnectedCalendarRepo)

	// Create in-process event bus for local mode (no RabbitMQ)
	c.InProcessEventBus = eventbus.NewInProcessEventBus(logger)

	// Create scheduling subscriber (auto-schedule tasks/habits/meetings)
	c.SchedulingSubscriber = scheduleSubs.NewSchedulingSubscriber(
		c.AutoScheduleHandler,
		taskRepo,
		habitRepo,
		meetingRepo,
		logger,
	)

	// Enable/disable based on config
	c.SchedulingSubscriber.SetEnabled(
		cfg.CalendarAutoScheduleTasks || cfg.CalendarAutoScheduleHabits || cfg.CalendarAutoScheduleMeetings,
	)

	// Register scheduling subscriber with event bus
	c.InProcessEventBus.RegisterConsumer(c.SchedulingSubscriber)

	// Create calendar sync subscriber (syncs scheduled blocks to external calendar)
	// Note: In local mode, CalendarSyncer is nil unless user has configured Google OAuth
	if c.CalendarSyncer != nil {
		c.CalendarSyncSubscriber = calendarSubs.NewCalendarSyncSubscriber(
			c.CalendarSyncer,
			scheduleRepo,
			logger,
		)
		c.InProcessEventBus.RegisterConsumer(c.CalendarSyncSubscriber)
		logger.Info("calendar sync subscriber enabled")
	}

	// Create calendar import worker (imports external events and handles conflicts)
	if cfg.CalendarSyncEnabled && c.CalendarImporter != nil {
		// Create conflict handler adapter for calendar import worker
		conflictHandler := schedulerServices.NewConflictHandlerAdapter(
			c.ConflictResolver,
			scheduleRepo,
			logger,
		)

		workerConfig := calendarWorkers.CalendarImportWorkerConfig{
			Interval:         cfg.CalendarSyncInterval,
			LookAheadDays:    cfg.CalendarSyncLookAheadDays,
			MaxSyncErrors:    5,
			BatchSize:        10,
			SkipOrbitaEvents: true,
		}
		c.CalendarImportWorker = calendarWorkers.NewCalendarImportWorker(
			c.CalendarImporter,
			c.SyncStateRepo,
			conflictHandler,
			workerConfig,
			logger,
		)
		logger.Info("calendar import worker configured",
			"interval", cfg.CalendarSyncInterval,
			"look_ahead_days", cfg.CalendarSyncLookAheadDays,
		)
	}

	// Create settings service
	c.SettingsService = identitySettings.NewService(settingsRepo)

	// Create license service for local mode
	licenseRepo := licensingPersistence.NewFileRepository(cfg.LicenseFilePath())
	licenseVerifier, err := licensingCrypto.NewVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to create license verifier: %w", err)
	}
	c.LicenseService = licensingApp.NewService(licenseRepo, licenseVerifier, logger)

	// Create LocalBillingService that wraps the license service
	c.BillingService = licensingApp.NewLocalBillingService(c.LicenseService)

	// Store connection for Close
	c.DBConn = conn
	c.DBDriver = database.DriverSQLite

	logger.Info("local mode container initialized",
		"database", cfg.SQLitePath,
		"driver", "sqlite",
	)

	return c, nil
}

// sqliteConnection is a type that implements database.Connection and exposes DB()
type sqliteConnection interface {
	database.Connection
	DB() *sql.DB
}

// initSQLiteConnection initializes the SQLite database connection with auto-migration.
func initSQLiteConnection(ctx context.Context, cfg *config.Config, logger *slog.Logger) (sqliteConnection, error) {
	// Create SQLite connection
	conn, err := database.NewConnection(ctx, database.Config{
		Driver:     database.DriverSQLite,
		SQLitePath: cfg.SQLitePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite connection: %w", err)
	}

	// Type assert to get SQLite-specific connection with DB() method
	sqliteConn, ok := conn.(sqliteConnection)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("expected SQLite connection with DB() method, got %T", conn)
	}

	// Run auto-migrations for SQLite
	if err := runSQLiteMigrations(ctx, sqliteConn.DB(), logger); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Ensure local user exists
	if err := ensureLocalUserExists(ctx, sqliteConn.DB(), cfg.UserID, logger); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ensure local user exists: %w", err)
	}

	return sqliteConn, nil
}

// runSQLiteMigrations applies SQLite schema migrations.
func runSQLiteMigrations(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	logger.Info("running SQLite migrations")
	if err := migrations.RunSQLiteMigrations(ctx, db); err != nil {
		return err
	}
	logger.Info("SQLite migrations completed successfully")
	return nil
}

// ensureLocalUserExists creates the local user in SQLite if they don't exist.
func ensureLocalUserExists(ctx context.Context, db *sql.DB, userID string, logger *slog.Logger) error {
	// Check if user exists
	var exists int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM users WHERE id = ?", userID).Scan(&exists)
	if err == nil {
		// User already exists
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// Create the local user
	now := time.Now().Format(time.RFC3339)
	_, err = db.ExecContext(ctx,
		"INSERT INTO users (id, email, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		userID, "local@orbita.local", "Local User", now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create local user: %w", err)
	}

	logger.Info("created local user", "user_id", userID)
	return nil
}
