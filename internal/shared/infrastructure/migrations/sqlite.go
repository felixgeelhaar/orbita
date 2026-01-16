package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
)

//go:embed sqlite/*.sql
var sqliteFS embed.FS

// RunSQLiteMigrations executes all SQLite migrations in order.
func RunSQLiteMigrations(ctx context.Context, db *sql.DB) error {
	// Read the initial schema file
	schema, err := sqliteFS.ReadFile("sqlite/000001_initial_schema.up.sql")
	if err != nil {
		return fmt.Errorf("failed to read SQLite schema: %w", err)
	}

	// Execute the schema (CREATE TABLE IF NOT EXISTS is idempotent)
	if _, err := db.ExecContext(ctx, string(schema)); err != nil {
		return fmt.Errorf("failed to execute SQLite schema: %w", err)
	}

	return nil
}
