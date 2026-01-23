package app

import (
	"database/sql"
	"fmt"

	db "github.com/felixgeelhaar/orbita/db/generated/postgres"
	automationsDomain "github.com/felixgeelhaar/orbita/internal/automations/domain"
	automationsPersistence "github.com/felixgeelhaar/orbita/internal/automations/infrastructure/persistence"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	billingPersistence "github.com/felixgeelhaar/orbita/internal/billing/infrastructure/persistence"
	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	calendarPersistence "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/persistence"
	habitsDomain "github.com/felixgeelhaar/orbita/internal/habits/domain"
	habitsPersistence "github.com/felixgeelhaar/orbita/internal/habits/infrastructure/persistence"
	settingsApp "github.com/felixgeelhaar/orbita/internal/identity/application/settings"
	identityDomain "github.com/felixgeelhaar/orbita/internal/identity/domain"
	identityPersistence "github.com/felixgeelhaar/orbita/internal/identity/infrastructure/persistence"
	inboxDomain "github.com/felixgeelhaar/orbita/internal/inbox/domain"
	inboxPersistence "github.com/felixgeelhaar/orbita/internal/inbox/persistence"
	insightsDomain "github.com/felixgeelhaar/orbita/internal/insights/domain"
	insightsPersistence "github.com/felixgeelhaar/orbita/internal/insights/infrastructure/persistence"
	meetingsDomain "github.com/felixgeelhaar/orbita/internal/meetings/domain"
	meetingsPersistence "github.com/felixgeelhaar/orbita/internal/meetings/infrastructure/persistence"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	productivityPersistence "github.com/felixgeelhaar/orbita/internal/productivity/infrastructure/persistence"
	projectsDomain "github.com/felixgeelhaar/orbita/internal/projects/domain"
	projectsPersistence "github.com/felixgeelhaar/orbita/internal/projects/infrastructure/persistence"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	schedulingPersistence "github.com/felixgeelhaar/orbita/internal/scheduling/infrastructure/persistence"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/database"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RepositoryFactory creates repositories based on the database driver.
type RepositoryFactory struct {
	conn   database.Connection
	driver database.Driver
}

// NewRepositoryFactory creates a new repository factory.
func NewRepositoryFactory(conn database.Connection) *RepositoryFactory {
	return &RepositoryFactory{
		conn:   conn,
		driver: conn.Driver(),
	}
}

// TaskRepository creates a task repository for the configured driver.
func (f *RepositoryFactory) TaskRepository() (task.Repository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return productivityPersistence.NewPostgresTaskRepositoryFromPool(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return productivityPersistence.NewSQLiteTaskRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// HabitRepository creates a habit repository for the configured driver.
func (f *RepositoryFactory) HabitRepository() (habitsDomain.Repository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return habitsPersistence.NewPostgresHabitRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return habitsPersistence.NewSQLiteHabitRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// ScheduleRepository creates a schedule repository for the configured driver.
func (f *RepositoryFactory) ScheduleRepository() (schedulingDomain.ScheduleRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return schedulingPersistence.NewPostgresScheduleRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return schedulingPersistence.NewSQLiteScheduleRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// MeetingRepository creates a meeting repository for the configured driver.
func (f *RepositoryFactory) MeetingRepository() (meetingsDomain.Repository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return meetingsPersistence.NewPostgresMeetingRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return meetingsPersistence.NewSQLiteMeetingRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// UserRepository creates a user repository for the configured driver.
func (f *RepositoryFactory) UserRepository() (identityDomain.UserRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return identityPersistence.NewPostgresUserRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return identityPersistence.NewSQLiteUserRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// SettingsRepository creates a settings repository for the configured driver.
func (f *RepositoryFactory) SettingsRepository() (settingsApp.Repository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return identityPersistence.NewSettingsRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return identityPersistence.NewSQLiteSettingsRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// OutboxRepository creates an outbox repository for the configured driver.
func (f *RepositoryFactory) OutboxRepository() (outbox.Repository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return outbox.NewPostgresRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return outbox.NewSQLiteRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// InboxRepository creates an inbox repository for the configured driver.
func (f *RepositoryFactory) InboxRepository() (inboxDomain.InboxRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return inboxPersistence.NewPostgresInboxRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return inboxPersistence.NewSQLiteInboxRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// ConnectedCalendarRepository creates a connected calendar repository for the configured driver.
func (f *RepositoryFactory) ConnectedCalendarRepository() (calendarDomain.ConnectedCalendarRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return calendarPersistence.NewPostgresConnectedCalendarRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return calendarPersistence.NewSQLiteConnectedCalendarRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// SyncStateRepository creates a sync state repository for the configured driver.
func (f *RepositoryFactory) SyncStateRepository() (calendarDomain.SyncStateRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return calendarPersistence.NewPostgresSyncStateRepository(pool), nil

	case database.DriverSQLite:
		db, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return calendarPersistence.NewSQLiteSyncStateRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// ProjectRepository creates a project repository for the configured driver.
func (f *RepositoryFactory) ProjectRepository() (projectsDomain.Repository, error) {
	switch f.driver {
	case database.DriverPostgres:
		// TODO: Implement PostgreSQL project repository
		return nil, fmt.Errorf("postgres project repository not yet implemented")

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return projectsPersistence.NewSQLiteProjectRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// RuleRepository creates an automation rule repository for the configured driver.
func (f *RepositoryFactory) RuleRepository() (automationsDomain.RuleRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		queries := db.New(pool)
		return automationsPersistence.NewRuleRepository(queries), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return automationsPersistence.NewSQLiteRuleRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// ExecutionRepository creates an automation execution repository for the configured driver.
func (f *RepositoryFactory) ExecutionRepository() (automationsDomain.ExecutionRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		queries := db.New(pool)
		return automationsPersistence.NewExecutionRepository(queries), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return automationsPersistence.NewSQLiteExecutionRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// PendingActionRepository creates an automation pending action repository for the configured driver.
func (f *RepositoryFactory) PendingActionRepository() (automationsDomain.PendingActionRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		queries := db.New(pool)
		return automationsPersistence.NewPendingActionRepository(queries), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return automationsPersistence.NewSQLitePendingActionRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// SnapshotRepository creates a productivity snapshot repository for the configured driver.
func (f *RepositoryFactory) SnapshotRepository() (insightsDomain.SnapshotRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		queries := db.New(pool)
		return insightsPersistence.NewSnapshotRepository(queries), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return insightsPersistence.NewSQLiteSnapshotRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// SessionRepository creates a time session repository for the configured driver.
func (f *RepositoryFactory) SessionRepository() (insightsDomain.SessionRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		queries := db.New(pool)
		return insightsPersistence.NewSessionRepository(queries), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return insightsPersistence.NewSQLiteSessionRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// SummaryRepository creates a weekly summary repository for the configured driver.
func (f *RepositoryFactory) SummaryRepository() (insightsDomain.SummaryRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		queries := db.New(pool)
		return insightsPersistence.NewSummaryRepository(queries), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return insightsPersistence.NewSQLiteSummaryRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// GoalRepository creates a productivity goal repository for the configured driver.
func (f *RepositoryFactory) GoalRepository() (insightsDomain.GoalRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		queries := db.New(pool)
		return insightsPersistence.NewGoalRepository(queries), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return insightsPersistence.NewSQLiteGoalRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// AnalyticsDataSource creates an analytics data source for the configured driver.
func (f *RepositoryFactory) AnalyticsDataSource() (insightsDomain.AnalyticsDataSource, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		queries := db.New(pool)
		return insightsPersistence.NewAnalyticsDataSource(queries), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return insightsPersistence.NewSQLiteAnalyticsDataSource(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// RescheduleAttemptRepository creates a reschedule attempt repository for the configured driver.
func (f *RepositoryFactory) RescheduleAttemptRepository() (schedulingDomain.RescheduleAttemptRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return schedulingPersistence.NewPostgresRescheduleAttemptRepository(pool), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return schedulingPersistence.NewSQLiteRescheduleAttemptRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// EntitlementRepository creates an entitlement repository for the configured driver.
func (f *RepositoryFactory) EntitlementRepository() (billingDomain.EntitlementRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return billingPersistence.NewPostgresEntitlementRepository(pool), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return billingPersistence.NewSQLiteEntitlementRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// SubscriptionRepository creates a subscription repository for the configured driver.
func (f *RepositoryFactory) SubscriptionRepository() (billingDomain.SubscriptionRepository, error) {
	switch f.driver {
	case database.DriverPostgres:
		pool, err := f.getPostgresPool()
		if err != nil {
			return nil, err
		}
		return billingPersistence.NewPostgresSubscriptionRepository(pool), nil

	case database.DriverSQLite:
		sqliteDB, err := f.getSQLiteDB()
		if err != nil {
			return nil, err
		}
		return billingPersistence.NewSQLiteSubscriptionRepository(sqliteDB), nil

	default:
		return nil, fmt.Errorf("unsupported driver: %s", f.driver)
	}
}

// Helper methods to get underlying database connections

func (f *RepositoryFactory) getPostgresPool() (*pgxpool.Pool, error) {
	pgConn, ok := f.conn.(interface{ Pool() *pgxpool.Pool })
	if !ok {
		return nil, fmt.Errorf("postgres connection does not expose Pool()")
	}
	return pgConn.Pool(), nil
}

func (f *RepositoryFactory) getSQLiteDB() (*sql.DB, error) {
	sqliteConn, ok := f.conn.(interface{ DB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("sqlite connection does not expose DB()")
	}
	return sqliteConn.DB(), nil
}

// Driver returns the database driver type.
func (f *RepositoryFactory) Driver() database.Driver {
	return f.driver
}

// Connection returns the underlying database connection.
func (f *RepositoryFactory) Connection() database.Connection {
	return f.conn
}
