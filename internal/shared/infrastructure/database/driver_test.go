package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectDriver(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected Driver
	}{
		{
			name:     "empty URL defaults to SQLite",
			url:      "",
			expected: DriverSQLite,
		},
		{
			name:     "postgres:// scheme",
			url:      "postgres://user:pass@localhost:5432/dbname",
			expected: DriverPostgres,
		},
		{
			name:     "postgresql:// scheme",
			url:      "postgresql://user:pass@localhost:5432/dbname",
			expected: DriverPostgres,
		},
		{
			name:     "sqlite:// scheme",
			url:      "sqlite:///path/to/db.sqlite",
			expected: DriverSQLite,
		},
		{
			name:     "file: scheme",
			url:      "file:/path/to/db.sqlite",
			expected: DriverSQLite,
		},
		{
			name:     ".db extension",
			url:      "/path/to/data.db",
			expected: DriverSQLite,
		},
		{
			name:     ".sqlite extension",
			url:      "/path/to/data.sqlite",
			expected: DriverSQLite,
		},
		{
			name:     ".sqlite3 extension",
			url:      "/path/to/data.sqlite3",
			expected: DriverSQLite,
		},
		{
			name:     "unknown defaults to PostgreSQL",
			url:      "mysql://user:pass@localhost/db",
			expected: DriverPostgres,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectDriver(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDriver_String(t *testing.T) {
	assert.Equal(t, "postgres", DriverPostgres.String())
	assert.Equal(t, "sqlite", DriverSQLite.String())
}

func TestDriver_IsValid(t *testing.T) {
	assert.True(t, DriverPostgres.IsValid())
	assert.True(t, DriverSQLite.IsValid())
	assert.False(t, Driver("mysql").IsValid())
	assert.False(t, Driver("").IsValid())
}
