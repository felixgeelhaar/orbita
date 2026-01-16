package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // Pure Go SQLite driver

	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/database"
)

func init() {
	database.RegisterSQLiteDriver(NewConnection)
}

// Connection wraps sql.DB to implement database.Connection for SQLite.
type Connection struct {
	db *sql.DB
}

// NewConnection creates a new SQLite connection.
func NewConnection(ctx context.Context, cfg database.Config) (database.Connection, error) {
	path := cfg.SQLitePath
	if path == "" {
		path = database.DefaultSQLitePath()
	}

	// Ensure the directory exists
	if err := database.EnsureDirectory(path); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Build DSN with pragmas for optimal SQLite performance
	// - journal_mode=WAL: Write-Ahead Logging for better concurrency
	// - foreign_keys=ON: Enforce foreign key constraints
	// - busy_timeout=5000: Wait 5s on lock instead of failing immediately
	// - synchronous=NORMAL: Good balance of safety and speed
	dsn := path
	if !strings.Contains(dsn, "?") {
		dsn += "?"
	} else {
		dsn += "&"
	}
	dsn += "_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// SQLite doesn't support multiple writers, so limit connections
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	return &Connection{db: db}, nil
}

// DB returns the underlying sql.DB.
// This is useful for advanced operations or testing.
func (c *Connection) DB() *sql.DB {
	return c.db
}

// Driver returns the driver type.
func (c *Connection) Driver() database.Driver {
	return database.DriverSQLite
}

// Close closes the database connection.
func (c *Connection) Close() error {
	return c.db.Close()
}

// Ping verifies the connection is still alive.
func (c *Connection) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// BeginTx starts a new transaction.
func (c *Connection) BeginTx(ctx context.Context) (database.Transaction, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx: tx}, nil
}

// Exec executes a query that doesn't return rows.
func (c *Connection) Exec(ctx context.Context, query string, args ...any) (database.Result, error) {
	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return database.WrapSQLResult(result), nil
}

// QueryRow executes a query that returns at most one row.
func (c *Connection) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// Query executes a query that returns multiple rows.
func (c *Connection) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return database.WrapSQLRows(rows), nil
}

// Transaction wraps sql.Tx to implement database.Transaction.
type Transaction struct {
	tx *sql.Tx
}

// Commit commits the transaction.
func (t *Transaction) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction.
func (t *Transaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

// Exec executes a query that doesn't return rows.
func (t *Transaction) Exec(ctx context.Context, query string, args ...any) (database.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return database.WrapSQLResult(result), nil
}

// QueryRow executes a query that returns at most one row.
func (t *Transaction) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

// Query executes a query that returns multiple rows.
func (t *Transaction) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return database.WrapSQLRows(rows), nil
}
