package database

import "strings"

// Driver represents a database backend type.
type Driver string

const (
	// DriverPostgres represents PostgreSQL database.
	DriverPostgres Driver = "postgres"
	// DriverSQLite represents SQLite database.
	DriverSQLite Driver = "sqlite"
)

// String returns the string representation of the driver.
func (d Driver) String() string {
	return string(d)
}

// DetectDriver parses a connection string and returns the driver type.
// Returns DriverSQLite for empty URLs to enable zero-config local mode.
func DetectDriver(url string) Driver {
	if url == "" {
		return DriverSQLite
	}

	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return DriverPostgres
	}

	if strings.HasPrefix(url, "sqlite://") ||
		strings.HasPrefix(url, "file:") ||
		strings.HasSuffix(url, ".db") ||
		strings.HasSuffix(url, ".sqlite") ||
		strings.HasSuffix(url, ".sqlite3") {
		return DriverSQLite
	}

	// Default to PostgreSQL for backward compatibility
	return DriverPostgres
}

// IsValid returns true if the driver is a known type.
func (d Driver) IsValid() bool {
	switch d {
	case DriverPostgres, DriverSQLite:
		return true
	default:
		return false
	}
}
