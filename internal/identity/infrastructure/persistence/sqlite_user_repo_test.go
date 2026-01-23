package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupUserTestDB creates an in-memory SQLite database with the schema applied.
func setupUserTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Open in-memory database
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Read and execute the schema
	schemaPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read SQLite schema file")

	_, err = sqlDB.Exec(string(schema))
	require.NoError(t, err, "Failed to apply SQLite schema")

	return sqlDB
}

func TestSQLiteUserRepository_Save_NewUser(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	email, err := domain.NewEmail("test@example.com")
	require.NoError(t, err)
	name, err := domain.NewName("Test User")
	require.NoError(t, err)

	user := domain.NewUser(email, name)

	// Save new user
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Verify user was saved
	found, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	assert.Equal(t, user.ID(), found.ID())
	assert.Equal(t, email.String(), found.Email().String())
	assert.Equal(t, name.String(), found.Name().String())
}

func TestSQLiteUserRepository_Save_UpdateUser(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	email, err := domain.NewEmail("update@example.com")
	require.NoError(t, err)
	name, err := domain.NewName("Original Name")
	require.NoError(t, err)

	user := domain.NewUser(email, name)
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Update the user's name
	newName, err := domain.NewName("Updated Name")
	require.NoError(t, err)
	user.UpdateName(newName)

	// Save updated user
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name().String())
}

func TestSQLiteUserRepository_FindByID_NotFound(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	// Try to find non-existent user
	_, err := repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestSQLiteUserRepository_FindByEmail(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	email, err := domain.NewEmail("findbyemail@example.com")
	require.NoError(t, err)
	name, err := domain.NewName("Find By Email User")
	require.NoError(t, err)

	user := domain.NewUser(email, name)
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Find by email
	found, err := repo.FindByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, user.ID(), found.ID())
	assert.Equal(t, email.String(), found.Email().String())
}

func TestSQLiteUserRepository_FindByEmail_NotFound(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	email, err := domain.NewEmail("notfound@example.com")
	require.NoError(t, err)

	// Try to find non-existent user by email
	_, err = repo.FindByEmail(ctx, email)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestSQLiteUserRepository_Delete(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	email, err := domain.NewEmail("delete@example.com")
	require.NoError(t, err)
	name, err := domain.NewName("Delete User")
	require.NoError(t, err)

	user := domain.NewUser(email, name)
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Delete the user
	err = repo.Delete(ctx, user.ID())
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.FindByID(ctx, user.ID())
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestSQLiteUserRepository_ExistsByEmail_True(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	email, err := domain.NewEmail("exists@example.com")
	require.NoError(t, err)
	name, err := domain.NewName("Exists User")
	require.NoError(t, err)

	user := domain.NewUser(email, name)
	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Check exists
	exists, err := repo.ExistsByEmail(ctx, email)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSQLiteUserRepository_ExistsByEmail_False(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	email, err := domain.NewEmail("notexists@example.com")
	require.NoError(t, err)

	// Check exists for non-existent user
	exists, err := repo.ExistsByEmail(ctx, email)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSQLiteUserRepository_MultipleUsers(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	// Create multiple users
	users := make([]*domain.User, 3)
	for i := 0; i < 3; i++ {
		email, err := domain.NewEmail("user" + string(rune('0'+i)) + "@example.com")
		require.NoError(t, err)
		name, err := domain.NewName("User " + string(rune('0'+i)))
		require.NoError(t, err)

		users[i] = domain.NewUser(email, name)
		err = repo.Save(ctx, users[i])
		require.NoError(t, err)
	}

	// Verify all users can be found
	for _, user := range users {
		found, err := repo.FindByID(ctx, user.ID())
		require.NoError(t, err)
		assert.Equal(t, user.ID(), found.ID())
	}
}

func TestSQLiteUserRepository_PreservesTimestamps(t *testing.T) {
	sqlDB := setupUserTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteUserRepository(sqlDB)
	ctx := context.Background()

	email, err := domain.NewEmail("timestamps@example.com")
	require.NoError(t, err)
	name, err := domain.NewName("Timestamps User")
	require.NoError(t, err)

	user := domain.NewUser(email, name)
	originalCreatedAt := user.CreatedAt()

	err = repo.Save(ctx, user)
	require.NoError(t, err)

	// Retrieve and check timestamps are preserved
	found, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)

	// CreatedAt should be preserved (within 1 second tolerance for parsing)
	assert.WithinDuration(t, originalCreatedAt, found.CreatedAt(), 1000000000) // 1 second
}
