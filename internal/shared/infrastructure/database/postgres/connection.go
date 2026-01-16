package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/database"
)

func init() {
	database.RegisterPostgresDriver(NewConnection)
}

// Connection wraps pgxpool.Pool to implement database.Connection.
type Connection struct {
	pool *pgxpool.Pool
}

// NewConnection creates a new PostgreSQL connection.
func NewConnection(ctx context.Context, cfg database.Config) (database.Connection, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("database URL is required for PostgreSQL")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolConfig.MaxConns = int32(cfg.MaxConns)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &Connection{pool: pool}, nil
}

// Pool returns the underlying pgxpool.Pool.
// This is useful for backward compatibility during migration.
func (c *Connection) Pool() *pgxpool.Pool {
	return c.pool
}

// Driver returns the driver type.
func (c *Connection) Driver() database.Driver {
	return database.DriverPostgres
}

// Close closes the connection pool.
func (c *Connection) Close() error {
	c.pool.Close()
	return nil
}

// Ping verifies the connection is still alive.
func (c *Connection) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// BeginTx starts a new transaction.
func (c *Connection) BeginTx(ctx context.Context) (database.Transaction, error) {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx: tx}, nil
}

// Exec executes a query that doesn't return rows.
func (c *Connection) Exec(ctx context.Context, query string, args ...any) (database.Result, error) {
	tag, err := c.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{tag: tag}, nil
}

// QueryRow executes a query that returns at most one row.
func (c *Connection) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return c.pool.QueryRow(ctx, query, args...)
}

// Query executes a query that returns multiple rows.
func (c *Connection) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	rows, err := c.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{rows: rows}, nil
}

// Transaction wraps pgx.Tx to implement database.Transaction.
type Transaction struct {
	tx pgx.Tx
}

// Commit commits the transaction.
func (t *Transaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction.
func (t *Transaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// Exec executes a query that doesn't return rows.
func (t *Transaction) Exec(ctx context.Context, query string, args ...any) (database.Result, error) {
	tag, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{tag: tag}, nil
}

// QueryRow executes a query that returns at most one row.
func (t *Transaction) QueryRow(ctx context.Context, query string, args ...any) database.Row {
	return t.tx.QueryRow(ctx, query, args...)
}

// Query executes a query that returns multiple rows.
func (t *Transaction) Query(ctx context.Context, query string, args ...any) (database.Rows, error) {
	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{rows: rows}, nil
}

// pgxResult wraps pgx command tag to implement database.Result.
type pgxResult struct {
	tag pgconn.CommandTag
}

func (r *pgxResult) RowsAffected() (int64, error) {
	return r.tag.RowsAffected(), nil
}

func (r *pgxResult) LastInsertId() (int64, error) {
	// PostgreSQL doesn't support LastInsertId via CommandTag.
	// Use RETURNING clause in queries instead.
	return 0, fmt.Errorf("LastInsertId not supported in PostgreSQL; use RETURNING clause")
}

// pgxRows wraps pgx.Rows to implement database.Rows.
type pgxRows struct {
	rows pgx.Rows
}

func (r *pgxRows) Next() bool {
	return r.rows.Next()
}

func (r *pgxRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *pgxRows) Close() error {
	r.rows.Close()
	return nil
}

func (r *pgxRows) Err() error {
	return r.rows.Err()
}
