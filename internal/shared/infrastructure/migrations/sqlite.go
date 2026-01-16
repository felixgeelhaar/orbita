package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed sqlite/*.sql
var sqliteFS embed.FS

// RunSQLiteMigrations executes all SQLite migrations in order.
func RunSQLiteMigrations(ctx context.Context, db *sql.DB) error {
	// Read all migration files
	entries, err := sqliteFS.ReadDir("sqlite")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort .up.sql files
	var upFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	// Execute each migration in order
	for _, file := range upFiles {
		migration, err := sqliteFS.ReadFile("sqlite/" + file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		// Execute the migration (CREATE TABLE IF NOT EXISTS is idempotent)
		if _, err := db.ExecContext(ctx, string(migration)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}
