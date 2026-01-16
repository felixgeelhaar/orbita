package app

import (
	"database/sql"
	"fmt"

	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	calendarPersistence "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/persistence"
	habitsDomain "github.com/felixgeelhaar/orbita/internal/habits/domain"
	habitsPersistence "github.com/felixgeelhaar/orbita/internal/habits/infrastructure/persistence"
	settingsApp "github.com/felixgeelhaar/orbita/internal/identity/application/settings"
	identityPersistence "github.com/felixgeelhaar/orbita/internal/identity/infrastructure/persistence"
	meetingsDomain "github.com/felixgeelhaar/orbita/internal/meetings/domain"
	meetingsPersistence "github.com/felixgeelhaar/orbita/internal/meetings/infrastructure/persistence"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	productivityPersistence "github.com/felixgeelhaar/orbita/internal/productivity/infrastructure/persistence"
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
