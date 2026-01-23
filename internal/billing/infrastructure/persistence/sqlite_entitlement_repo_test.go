package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func setupBillingTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Read and execute the schema
	schemaPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read SQLite schema file")

	_, err = db.Exec(string(schema))
	require.NoError(t, err)

	return db
}

func TestSQLiteEntitlementRepository_Set(t *testing.T) {
	db := setupBillingTestDB(t)
	defer db.Close()

	repo := NewSQLiteEntitlementRepository(db)
	userID := uuid.New()

	// Test setting an active entitlement
	err := repo.Set(context.Background(), userID, "smart-habits", true, "manual")
	require.NoError(t, err)

	// Verify it was set correctly
	active, err := repo.IsActive(context.Background(), userID, "smart-habits")
	require.NoError(t, err)
	assert.True(t, active)

	// Test updating to inactive
	err = repo.Set(context.Background(), userID, "smart-habits", false, "manual")
	require.NoError(t, err)

	active, err = repo.IsActive(context.Background(), userID, "smart-habits")
	require.NoError(t, err)
	assert.False(t, active)
}

func TestSQLiteEntitlementRepository_List(t *testing.T) {
	db := setupBillingTestDB(t)
	defer db.Close()

	repo := NewSQLiteEntitlementRepository(db)
	userID := uuid.New()

	// Set multiple entitlements
	err := repo.Set(context.Background(), userID, "smart-habits", true, "stripe")
	require.NoError(t, err)

	err = repo.Set(context.Background(), userID, "ai-inbox", true, "manual")
	require.NoError(t, err)

	err = repo.Set(context.Background(), userID, "auto-rescheduler", false, "manual")
	require.NoError(t, err)

	// List entitlements
	entitlements, err := repo.List(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, entitlements, 3)

	// Verify ordering (alphabetical by module)
	assert.Equal(t, "ai-inbox", entitlements[0].Module)
	assert.True(t, entitlements[0].Active)

	assert.Equal(t, "auto-rescheduler", entitlements[1].Module)
	assert.False(t, entitlements[1].Active)

	assert.Equal(t, "smart-habits", entitlements[2].Module)
	assert.True(t, entitlements[2].Active)
}

func TestSQLiteEntitlementRepository_IsActive(t *testing.T) {
	db := setupBillingTestDB(t)
	defer db.Close()

	repo := NewSQLiteEntitlementRepository(db)
	userID := uuid.New()

	// Test non-existent entitlement
	active, err := repo.IsActive(context.Background(), userID, "non-existent")
	require.NoError(t, err)
	assert.False(t, active)

	// Test active entitlement
	err = repo.Set(context.Background(), userID, "smart-habits", true, "manual")
	require.NoError(t, err)

	active, err = repo.IsActive(context.Background(), userID, "smart-habits")
	require.NoError(t, err)
	assert.True(t, active)

	// Test inactive entitlement
	err = repo.Set(context.Background(), userID, "ai-inbox", false, "manual")
	require.NoError(t, err)

	active, err = repo.IsActive(context.Background(), userID, "ai-inbox")
	require.NoError(t, err)
	assert.False(t, active)
}

func TestSQLiteEntitlementRepository_TrialingStatus(t *testing.T) {
	db := setupBillingTestDB(t)
	defer db.Close()

	repo := NewSQLiteEntitlementRepository(db)
	userID := uuid.New()

	// smart-habits is pre-seeded, just add user_entitlement with trialing status
	_, err := db.Exec(`
		INSERT INTO user_entitlements (user_id, entitlement_id, status, created_at, updated_at)
		VALUES (?, 'smart-habits', 'trialing', datetime('now'), datetime('now'))
	`, userID.String())
	require.NoError(t, err)

	// Trialing should be treated as active
	active, err := repo.IsActive(context.Background(), userID, "smart-habits")
	require.NoError(t, err)
	assert.True(t, active)
}

func TestSQLiteEntitlementRepository_MultipleUsers(t *testing.T) {
	db := setupBillingTestDB(t)
	defer db.Close()

	repo := NewSQLiteEntitlementRepository(db)
	user1 := uuid.New()
	user2 := uuid.New()

	// Set entitlements for different users
	err := repo.Set(context.Background(), user1, "smart-habits", true, "manual")
	require.NoError(t, err)

	err = repo.Set(context.Background(), user2, "smart-habits", false, "manual")
	require.NoError(t, err)

	// Verify they are independent
	active1, err := repo.IsActive(context.Background(), user1, "smart-habits")
	require.NoError(t, err)
	assert.True(t, active1)

	active2, err := repo.IsActive(context.Background(), user2, "smart-habits")
	require.NoError(t, err)
	assert.False(t, active2)
}
