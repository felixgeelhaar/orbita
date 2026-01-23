package persistence

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create a simple test table
	_, err = db.Exec(`CREATE TABLE test_data (id INTEGER PRIMARY KEY, value TEXT)`)
	require.NoError(t, err)

	return db
}

func TestNewSQLiteUnitOfWork(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	assert.NotNil(t, uow)
}

func TestSQLiteUnitOfWork_Begin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	ctx := context.Background()

	// Begin a transaction
	txCtx, err := uow.Begin(ctx)
	require.NoError(t, err)
	require.NotNil(t, txCtx)

	// Verify transaction is in context
	info, ok := SQLiteTxInfoFromContext(txCtx)
	assert.True(t, ok)
	assert.NotNil(t, info.Tx)
	assert.True(t, info.Owned)

	// Rollback to clean up
	err = uow.Rollback(txCtx)
	require.NoError(t, err)
}

func TestSQLiteUnitOfWork_NestedTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	ctx := context.Background()

	// Begin first (outer) transaction
	outerCtx, err := uow.Begin(ctx)
	require.NoError(t, err)

	outerInfo, ok := SQLiteTxInfoFromContext(outerCtx)
	require.True(t, ok)
	assert.True(t, outerInfo.Owned)

	// Begin second (inner) transaction - should reuse the outer one
	innerCtx, err := uow.Begin(outerCtx)
	require.NoError(t, err)

	innerInfo, ok := SQLiteTxInfoFromContext(innerCtx)
	require.True(t, ok)
	assert.False(t, innerInfo.Owned) // Inner should NOT own the transaction
	assert.Equal(t, outerInfo.Tx, innerInfo.Tx) // Same transaction object

	// Commit inner (should be no-op since it doesn't own)
	err = uow.Commit(innerCtx)
	require.NoError(t, err)

	// Rollback outer
	err = uow.Rollback(outerCtx)
	require.NoError(t, err)
}

func TestSQLiteUnitOfWork_CommitPersistsData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	ctx := context.Background()

	// Begin transaction
	txCtx, err := uow.Begin(ctx)
	require.NoError(t, err)

	// Get transaction and insert data
	info, ok := SQLiteTxInfoFromContext(txCtx)
	require.True(t, ok)

	_, err = info.Tx.Exec(`INSERT INTO test_data (value) VALUES ('test_value')`)
	require.NoError(t, err)

	// Commit
	err = uow.Commit(txCtx)
	require.NoError(t, err)

	// Verify data persisted
	var value string
	err = db.QueryRow(`SELECT value FROM test_data WHERE value = 'test_value'`).Scan(&value)
	require.NoError(t, err)
	assert.Equal(t, "test_value", value)
}

func TestSQLiteUnitOfWork_RollbackDiscardsData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	ctx := context.Background()

	// Begin transaction
	txCtx, err := uow.Begin(ctx)
	require.NoError(t, err)

	// Get transaction and insert data
	info, ok := SQLiteTxInfoFromContext(txCtx)
	require.True(t, ok)

	_, err = info.Tx.Exec(`INSERT INTO test_data (value) VALUES ('rollback_value')`)
	require.NoError(t, err)

	// Rollback
	err = uow.Rollback(txCtx)
	require.NoError(t, err)

	// Verify data was NOT persisted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM test_data WHERE value = 'rollback_value'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSQLiteUnitOfWork_CommitWithoutTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	ctx := context.Background()

	// Try to commit without starting a transaction
	err := uow.Commit(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no transaction in context")
}

func TestSQLiteUnitOfWork_RollbackWithoutTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	ctx := context.Background()

	// Try to rollback without starting a transaction
	err := uow.Rollback(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no transaction in context")
}

func TestSQLiteTxInfoFromContext_Empty(t *testing.T) {
	ctx := context.Background()

	// Should return false when no transaction in context
	info, ok := SQLiteTxInfoFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, info.Tx)
}

func TestSQLiteTxInfoFromContext_NilTx(t *testing.T) {
	// Create context with nil transaction
	ctx := WithSQLiteTx(context.Background(), nil, true)

	// Should return false when transaction is nil
	info, ok := SQLiteTxInfoFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, info.Tx)
}

func TestWithSQLiteTx(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Start a real transaction
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	// Store in context with owned=true
	txCtx := WithSQLiteTx(ctx, tx, true)
	info, ok := SQLiteTxInfoFromContext(txCtx)
	require.True(t, ok)
	assert.Equal(t, tx, info.Tx)
	assert.True(t, info.Owned)

	// Store in new context with owned=false
	notOwnedCtx := WithSQLiteTx(ctx, tx, false)
	info2, ok := SQLiteTxInfoFromContext(notOwnedCtx)
	require.True(t, ok)
	assert.Equal(t, tx, info2.Tx)
	assert.False(t, info2.Owned)
}

func TestSQLiteUnitOfWork_CommitNotOwned(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	ctx := context.Background()

	// Begin outer transaction
	outerCtx, err := uow.Begin(ctx)
	require.NoError(t, err)

	// Begin inner (not owned)
	innerCtx, err := uow.Begin(outerCtx)
	require.NoError(t, err)

	// Commit on inner should be no-op (returns nil)
	err = uow.Commit(innerCtx)
	require.NoError(t, err)

	// Outer transaction should still be active
	info, ok := SQLiteTxInfoFromContext(outerCtx)
	require.True(t, ok)

	// We can still use the transaction
	_, err = info.Tx.Exec(`INSERT INTO test_data (value) VALUES ('still_active')`)
	require.NoError(t, err)

	// Clean up
	err = uow.Rollback(outerCtx)
	require.NoError(t, err)
}

func TestSQLiteUnitOfWork_RollbackNotOwned(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := NewSQLiteUnitOfWork(db)
	ctx := context.Background()

	// Begin outer transaction
	outerCtx, err := uow.Begin(ctx)
	require.NoError(t, err)

	// Begin inner (not owned)
	innerCtx, err := uow.Begin(outerCtx)
	require.NoError(t, err)

	// Rollback on inner should be no-op (returns nil)
	err = uow.Rollback(innerCtx)
	require.NoError(t, err)

	// Outer transaction should still be active
	info, ok := SQLiteTxInfoFromContext(outerCtx)
	require.True(t, ok)

	// We can still use the transaction
	_, err = info.Tx.Exec(`INSERT INTO test_data (value) VALUES ('still_active_after_inner_rollback')`)
	require.NoError(t, err)

	// Clean up
	err = uow.Rollback(outerCtx)
	require.NoError(t, err)
}
