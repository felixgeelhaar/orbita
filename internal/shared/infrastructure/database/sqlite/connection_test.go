package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/database"
)

func TestNewConnection_InMemory(t *testing.T) {
	ctx := context.Background()

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "orbita-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := database.Config{
		Driver:     database.DriverSQLite,
		SQLitePath: filepath.Join(tmpDir, "test.db"),
	}

	conn, err := NewConnection(ctx, cfg)
	require.NoError(t, err)
	defer conn.Close()

	// Verify connection works
	err = conn.Ping(ctx)
	assert.NoError(t, err)

	// Verify driver type
	assert.Equal(t, database.DriverSQLite, conn.Driver())
}

func TestConnection_ExecAndQuery(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "orbita-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := database.Config{
		SQLitePath: filepath.Join(tmpDir, "test.db"),
	}

	conn, err := NewConnection(ctx, cfg)
	require.NoError(t, err)
	defer conn.Close()

	// Create a test table
	_, err = conn.Exec(ctx, `CREATE TABLE test (id TEXT PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)

	// Insert data
	result, err := conn.Exec(ctx, `INSERT INTO test (id, name) VALUES (?, ?)`, "1", "Alice")
	require.NoError(t, err)

	rowsAffected, err := result.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Query single row
	row := conn.QueryRow(ctx, `SELECT id, name FROM test WHERE id = ?`, "1")
	var id, name string
	err = row.Scan(&id, &name)
	require.NoError(t, err)
	assert.Equal(t, "1", id)
	assert.Equal(t, "Alice", name)

	// Insert more data
	_, err = conn.Exec(ctx, `INSERT INTO test (id, name) VALUES (?, ?)`, "2", "Bob")
	require.NoError(t, err)

	// Query multiple rows
	rows, err := conn.Query(ctx, `SELECT id, name FROM test ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	var results []string
	for rows.Next() {
		var id, name string
		err := rows.Scan(&id, &name)
		require.NoError(t, err)
		results = append(results, name)
	}
	assert.NoError(t, rows.Err())
	assert.Equal(t, []string{"Alice", "Bob"}, results)
}

func TestConnection_Transaction(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "orbita-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := database.Config{
		SQLitePath: filepath.Join(tmpDir, "test.db"),
	}

	conn, err := NewConnection(ctx, cfg)
	require.NoError(t, err)
	defer conn.Close()

	// Create a test table
	_, err = conn.Exec(ctx, `CREATE TABLE test (id TEXT PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)

	// Test transaction commit
	tx, err := conn.BeginTx(ctx)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, `INSERT INTO test (id, name) VALUES (?, ?)`, "1", "Alice")
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Verify data persisted
	row := conn.QueryRow(ctx, `SELECT name FROM test WHERE id = ?`, "1")
	var name string
	err = row.Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "Alice", name)

	// Test transaction rollback
	tx2, err := conn.BeginTx(ctx)
	require.NoError(t, err)

	_, err = tx2.Exec(ctx, `INSERT INTO test (id, name) VALUES (?, ?)`, "2", "Bob")
	require.NoError(t, err)

	err = tx2.Rollback(ctx)
	require.NoError(t, err)

	// Verify data was rolled back
	row = conn.QueryRow(ctx, `SELECT COUNT(*) FROM test`)
	var count int
	err = row.Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count) // Only Alice should exist
}
