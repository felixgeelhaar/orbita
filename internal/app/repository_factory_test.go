package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/productivity/domain/task"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// mockSQLiteConnection implements database.Connection for testing.
type mockSQLiteConnection struct {
	db *sql.DB
}

func (m *mockSQLiteConnection) Driver() database.Driver {
	return database.DriverSQLite
}

func (m *mockSQLiteConnection) DB() *sql.DB {
	return m.db
}

func (m *mockSQLiteConnection) Close() error {
	return m.db.Close()
}

func (m *mockSQLiteConnection) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

func (m *mockSQLiteConnection) BeginTx(ctx context.Context) (database.Transaction, error) {
	return nil, nil // Not needed for this test
}

func (m *mockSQLiteConnection) Exec(ctx context.Context, query string, args ...any) (database.Result, error) {
	return nil, nil
}

func (m *mockSQLiteConnection) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return nil
}

func (m *mockSQLiteConnection) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	return nil, nil
}

// setupTestDB creates an in-memory SQLite database with schema.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	schemaPath := filepath.Join("..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	_, err = sqlDB.Exec(string(schema))
	require.NoError(t, err)

	return sqlDB
}

func createUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
	t.Helper()

	queries := db.New(sqlDB)
	_, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        userID.String(),
		Email:     "test-" + userID.String()[:8] + "@example.com",
		Name:      "Test User",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
	require.NoError(t, err)
}

func TestRepositoryFactory_TaskRepository_SQLite(t *testing.T) {
	sqlDB := setupTestDB(t)
	defer sqlDB.Close()

	// Create a mock connection that exposes the DB() method
	conn := &mockSQLiteConnection{db: sqlDB}

	// Create the factory
	factory := NewRepositoryFactory(conn)

	// Get the task repository
	taskRepo, err := factory.TaskRepository()
	require.NoError(t, err)
	require.NotNil(t, taskRepo)

	// Create a user (needed for foreign key)
	userID := uuid.New()
	createUser(t, sqlDB, userID)

	// Test the repository works
	ctx := context.Background()
	newTask, err := task.NewTask(userID, "Factory Test Task")
	require.NoError(t, err)

	err = taskRepo.Save(ctx, newTask)
	require.NoError(t, err)

	found, err := taskRepo.FindByID(ctx, newTask.ID())
	require.NoError(t, err)
	assert.Equal(t, "Factory Test Task", found.Title())
}

func TestRepositoryFactory_Driver(t *testing.T) {
	sqlDB := setupTestDB(t)
	defer sqlDB.Close()

	conn := &mockSQLiteConnection{db: sqlDB}
	factory := NewRepositoryFactory(conn)

	assert.Equal(t, database.DriverSQLite, factory.Driver())
}

func TestRepositoryFactory_Connection(t *testing.T) {
	sqlDB := setupTestDB(t)
	defer sqlDB.Close()

	conn := &mockSQLiteConnection{db: sqlDB}
	factory := NewRepositoryFactory(conn)

	assert.Equal(t, conn, factory.Connection())
}
