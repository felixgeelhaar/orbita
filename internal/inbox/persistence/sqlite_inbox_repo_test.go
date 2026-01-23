package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupSQLiteTestDB creates an in-memory SQLite database with the schema applied.
func setupSQLiteTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Open in-memory database
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Read and execute the schema
	schemaPath := filepath.Join("..", "..", "..", "migrations", "sqlite", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "Failed to read SQLite schema file")

	_, err = sqlDB.Exec(string(schema))
	require.NoError(t, err, "Failed to apply SQLite schema")

	return sqlDB
}

// createTestUser creates a user in the database for foreign key constraints.
func createTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
	t.Helper()

	queries := db.New(sqlDB)
	_, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        userID.String(),
		Email:     "test-" + userID.String()[:8] + "@example.com",
		Name:      "Test User",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
	require.NoError(t, err)
}

func TestSQLiteInboxRepository_Save(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create a new inbox item
	item := domain.InboxItem{
		ID:             uuid.New(),
		UserID:         userID,
		Content:        "Test inbox item content",
		Metadata:       domain.InboxMetadata{"key": "value"},
		Tags:           []string{"tag1", "tag2"},
		Source:         "cli",
		Classification: "task",
		CapturedAt:     time.Now().Truncate(time.Second),
	}

	// Save it
	err := repo.Save(ctx, item)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.FindByID(ctx, userID, item.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, item.ID, found.ID)
	assert.Equal(t, "Test inbox item content", found.Content)
	assert.Equal(t, userID, found.UserID)
	assert.Equal(t, "cli", found.Source)
	assert.Equal(t, "task", found.Classification)
}

func TestSQLiteInboxRepository_ListByUser(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	otherUserID := uuid.New()
	createTestUser(t, sqlDB, otherUserID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create items for the user
	item1 := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     userID,
		Content:    "Item 1",
		Source:     "cli",
		CapturedAt: time.Now().Add(-2 * time.Hour).Truncate(time.Second),
	}
	item2 := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     userID,
		Content:    "Item 2",
		Source:     "cli",
		CapturedAt: time.Now().Add(-1 * time.Hour).Truncate(time.Second),
	}
	item3 := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     otherUserID,
		Content:    "Other User Item",
		Source:     "cli",
		CapturedAt: time.Now().Truncate(time.Second),
	}

	require.NoError(t, repo.Save(ctx, item1))
	require.NoError(t, repo.Save(ctx, item2))
	require.NoError(t, repo.Save(ctx, item3))

	// List items for the user
	items, err := repo.ListByUser(ctx, userID, false)
	require.NoError(t, err)
	assert.Len(t, items, 2)

	// Verify we got the right items (ordered by captured_at DESC)
	assert.Equal(t, item2.ID, items[0].ID) // More recent first
	assert.Equal(t, item1.ID, items[1].ID)
}

func TestSQLiteInboxRepository_ListByUser_ExcludesPromoted(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create regular and promoted items
	regularItem := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     userID,
		Content:    "Regular Item",
		Source:     "cli",
		CapturedAt: time.Now().Truncate(time.Second),
	}
	promotedItem := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     userID,
		Content:    "Promoted Item",
		Source:     "cli",
		CapturedAt: time.Now().Add(-1 * time.Hour).Truncate(time.Second),
	}

	require.NoError(t, repo.Save(ctx, regularItem))
	require.NoError(t, repo.Save(ctx, promotedItem))

	// Mark second item as promoted
	promotedAt := time.Now()
	err := repo.MarkPromoted(ctx, promotedItem.ID, "task", uuid.New(), promotedAt)
	require.NoError(t, err)

	// List without promoted - should only get regular item
	items, err := repo.ListByUser(ctx, userID, false)
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, regularItem.ID, items[0].ID)

	// List with promoted - should get both
	itemsWithPromoted, err := repo.ListByUser(ctx, userID, true)
	require.NoError(t, err)
	assert.Len(t, itemsWithPromoted, 2)
}

func TestSQLiteInboxRepository_FindByID(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create an item
	item := domain.InboxItem{
		ID:             uuid.New(),
		UserID:         userID,
		Content:        "Test content",
		Metadata:       domain.InboxMetadata{"foo": "bar"},
		Tags:           []string{"urgent"},
		Source:         "email",
		Classification: "meeting",
		CapturedAt:     time.Now().Truncate(time.Second),
	}

	require.NoError(t, repo.Save(ctx, item))

	// Find by ID
	found, err := repo.FindByID(ctx, userID, item.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, item.ID, found.ID)
	assert.Equal(t, "Test content", found.Content)
	assert.Equal(t, "bar", found.Metadata["foo"])
	assert.Contains(t, found.Tags, "urgent")
	assert.Equal(t, "email", found.Source)
	assert.Equal(t, "meeting", found.Classification)
}

func TestSQLiteInboxRepository_FindByID_NotFound(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, userID, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestSQLiteInboxRepository_FindByID_WrongUser(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	otherUserID := uuid.New()
	createTestUser(t, sqlDB, otherUserID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create an item for userID
	item := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     userID,
		Content:    "User's item",
		Source:     "cli",
		CapturedAt: time.Now().Truncate(time.Second),
	}
	require.NoError(t, repo.Save(ctx, item))

	// Try to find it as otherUserID
	found, err := repo.FindByID(ctx, otherUserID, item.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestSQLiteInboxRepository_MarkPromoted(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create an item
	item := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     userID,
		Content:    "Item to promote",
		Source:     "cli",
		CapturedAt: time.Now().Truncate(time.Second),
	}
	require.NoError(t, repo.Save(ctx, item))

	// Mark as promoted
	promotedID := uuid.New()
	promotedAt := time.Now().Truncate(time.Second)
	err := repo.MarkPromoted(ctx, item.ID, "task", promotedID, promotedAt)
	require.NoError(t, err)

	// Verify promotion
	found, err := repo.FindByID(ctx, userID, item.ID)
	require.NoError(t, err)
	assert.True(t, found.Promoted)
	assert.Equal(t, "task", found.PromotedTo)
	assert.Equal(t, promotedID, found.PromotedID)
	require.NotNil(t, found.PromotedAt)
	assert.Equal(t, promotedAt.Unix(), found.PromotedAt.Unix())
}

func TestSQLiteInboxRepository_WithMetadata(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create an item with complex metadata
	item := domain.InboxItem{
		ID:     uuid.New(),
		UserID: userID,
		Content: "Item with metadata",
		Metadata: domain.InboxMetadata{
			"subject": "Meeting Notes",
			"from":    "alice@example.com",
			"date":    "2024-01-15",
		},
		Source:     "email",
		CapturedAt: time.Now().Truncate(time.Second),
	}

	require.NoError(t, repo.Save(ctx, item))

	// Verify metadata is persisted correctly
	found, err := repo.FindByID(ctx, userID, item.ID)
	require.NoError(t, err)

	assert.Equal(t, "Meeting Notes", found.Metadata["subject"])
	assert.Equal(t, "alice@example.com", found.Metadata["from"])
	assert.Equal(t, "2024-01-15", found.Metadata["date"])
}

func TestSQLiteInboxRepository_WithEmptyMetadata(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create an item without metadata
	item := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     userID,
		Content:    "Item without metadata",
		Source:     "cli",
		CapturedAt: time.Now().Truncate(time.Second),
	}

	require.NoError(t, repo.Save(ctx, item))

	// Verify item is retrieved correctly
	found, err := repo.FindByID(ctx, userID, item.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
}

func TestSQLiteInboxRepository_WithTags(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Create an item with multiple tags
	item := domain.InboxItem{
		ID:         uuid.New(),
		UserID:     userID,
		Content:    "Tagged item",
		Tags:       []string{"work", "urgent", "review"},
		Source:     "cli",
		CapturedAt: time.Now().Truncate(time.Second),
	}

	require.NoError(t, repo.Save(ctx, item))

	// Verify tags are persisted correctly
	found, err := repo.FindByID(ctx, userID, item.ID)
	require.NoError(t, err)

	assert.Len(t, found.Tags, 3)
	assert.Contains(t, found.Tags, "work")
	assert.Contains(t, found.Tags, "urgent")
	assert.Contains(t, found.Tags, "review")
}

func TestSQLiteInboxRepository_EmptyList(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// List items for user with no items
	items, err := repo.ListByUser(ctx, userID, false)
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestSQLiteInboxRepository_AllSources(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Test different sources
	sources := []string{"cli", "email", "api", "siri", "mcp"}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			item := domain.InboxItem{
				ID:         uuid.New(),
				UserID:     userID,
				Content:    "Item from " + source,
				Source:     source,
				CapturedAt: time.Now().Truncate(time.Second),
			}

			err := repo.Save(ctx, item)
			require.NoError(t, err)

			found, err := repo.FindByID(ctx, userID, item.ID)
			require.NoError(t, err)
			assert.Equal(t, source, found.Source)
		})
	}
}

func TestSQLiteInboxRepository_AllClassifications(t *testing.T) {
	sqlDB := setupSQLiteTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createTestUser(t, sqlDB, userID)

	repo := NewSQLiteInboxRepository(sqlDB)
	ctx := context.Background()

	// Test different classifications
	classifications := []string{"task", "meeting", "habit", "note", "reminder"}

	for _, classification := range classifications {
		t.Run(classification, func(t *testing.T) {
			item := domain.InboxItem{
				ID:             uuid.New(),
				UserID:         userID,
				Content:        "Item classified as " + classification,
				Source:         "cli",
				Classification: classification,
				CapturedAt:     time.Now().Truncate(time.Second),
			}

			err := repo.Save(ctx, item)
			require.NoError(t, err)

			found, err := repo.FindByID(ctx, userID, item.ID)
			require.NoError(t, err)
			assert.Equal(t, classification, found.Classification)
		})
	}
}
