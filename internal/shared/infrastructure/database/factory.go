package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds database configuration.
type Config struct {
	// Driver specifies the database driver to use.
	// If empty or "auto", it will be detected from the URL.
	Driver Driver

	// URL is the connection string for PostgreSQL.
	// Example: "postgres://user:pass@localhost:5432/dbname"
	URL string

	// SQLitePath is the path to the SQLite database file.
	// Used when Driver is DriverSQLite.
	// Defaults to ~/.orbita/data.db
	SQLitePath string

	// MaxConns is the maximum number of connections (PostgreSQL only).
	MaxConns int
}

// NewConnection creates a database connection based on configuration.
// This is the main factory function for creating database connections.
func NewConnection(ctx context.Context, cfg Config) (Connection, error) {
	driver := cfg.Driver
	if driver == "" || driver == "auto" {
		driver = DetectDriver(cfg.URL)
	}

	switch driver {
	case DriverPostgres:
		return newPostgresConnection(ctx, cfg)
	case DriverSQLite:
		return newSQLiteConnection(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

// DefaultSQLitePath returns the default SQLite database path.
func DefaultSQLitePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".orbita", "data.db")
}

// DefaultLocalConfig returns configuration for local SQLite mode.
func DefaultLocalConfig() Config {
	return Config{
		Driver:     DriverSQLite,
		SQLitePath: DefaultSQLitePath(),
	}
}

// EnsureDirectory creates the parent directory for a file path if it doesn't exist.
func EnsureDirectory(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// newPostgresConnection creates a PostgreSQL connection.
// This is a forward declaration - the actual implementation is in postgres/connection.go
// and will be wired in at build time.
var newPostgresConnection func(ctx context.Context, cfg Config) (Connection, error)

// newSQLiteConnection creates a SQLite connection.
// This is a forward declaration - the actual implementation is in sqlite/connection.go
// and will be wired in at build time.
var newSQLiteConnection func(ctx context.Context, cfg Config) (Connection, error)

// RegisterPostgresDriver registers the PostgreSQL connection factory.
func RegisterPostgresDriver(fn func(ctx context.Context, cfg Config) (Connection, error)) {
	newPostgresConnection = fn
}

// RegisterSQLiteDriver registers the SQLite connection factory.
func RegisterSQLiteDriver(fn func(ctx context.Context, cfg Config) (Connection, error)) {
	newSQLiteConnection = fn
}
