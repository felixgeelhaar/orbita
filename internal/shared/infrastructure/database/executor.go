package database

import (
	"context"
	"database/sql"
)

// Row represents a single result row.
// This interface abstracts pgx.Row and *sql.Row.
type Row interface {
	Scan(dest ...any) error
}

// Rows represents multiple result rows.
// This interface abstracts pgx.Rows and *sql.Rows.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// Result represents the result of an Exec operation.
type Result interface {
	RowsAffected() (int64, error)
	LastInsertId() (int64, error)
}

// Executor is the database executor interface used by all repositories.
// It provides a unified interface for executing queries regardless of the underlying driver.
type Executor interface {
	// Exec executes a query that doesn't return rows (INSERT, UPDATE, DELETE).
	Exec(ctx context.Context, query string, args ...any) (Result, error)

	// QueryRow executes a query that returns at most one row.
	QueryRow(ctx context.Context, query string, args ...any) Row

	// Query executes a query that returns multiple rows.
	Query(ctx context.Context, query string, args ...any) (Rows, error)
}

// Transaction wraps Executor with Commit/Rollback capabilities.
type Transaction interface {
	Executor
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Connection represents a database connection that can create transactions.
type Connection interface {
	Executor
	// BeginTx starts a new transaction.
	BeginTx(ctx context.Context) (Transaction, error)
	// Close closes the database connection.
	Close() error
	// Ping verifies the connection is still alive.
	Ping(ctx context.Context) error
	// Driver returns the driver type for this connection.
	Driver() Driver
}

// sqlResult wraps sql.Result to implement our Result interface.
type sqlResult struct {
	result sql.Result
}

func (r *sqlResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

func (r *sqlResult) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

// WrapSQLResult wraps a sql.Result to implement the Result interface.
func WrapSQLResult(r sql.Result) Result {
	return &sqlResult{result: r}
}

// sqlRows wraps sql.Rows to implement our Rows interface.
type sqlRows struct {
	rows *sql.Rows
}

func (r *sqlRows) Next() bool {
	return r.rows.Next()
}

func (r *sqlRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRows) Close() error {
	return r.rows.Close()
}

func (r *sqlRows) Err() error {
	return r.rows.Err()
}

// WrapSQLRows wraps sql.Rows to implement the Rows interface.
func WrapSQLRows(r *sql.Rows) Rows {
	return &sqlRows{rows: r}
}
