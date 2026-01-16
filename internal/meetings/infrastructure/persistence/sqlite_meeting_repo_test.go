package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupMeetingTestDB creates an in-memory SQLite database with the schema applied.
func setupMeetingTestDB(t *testing.T) *sql.DB {
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

// createMeetingTestUser creates a user in the database for foreign key constraints.
func createMeetingTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
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

func TestSQLiteMeetingRepository_Save_Create(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// Create a new meeting
	meeting, err := domain.NewMeeting(userID, "Weekly Standup", domain.CadenceWeekly, 7, 30*time.Minute, 10*time.Hour)
	require.NoError(t, err)

	// Save it
	err = repo.Save(ctx, meeting)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, meeting.ID(), found.ID())
	assert.Equal(t, "Weekly Standup", found.Name())
	assert.Equal(t, userID, found.UserID())
	assert.Equal(t, domain.CadenceWeekly, found.Cadence())
	assert.Equal(t, 30*time.Minute, found.Duration())
}

func TestSQLiteMeetingRepository_Save_Update(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// Create and save a meeting
	meeting, err := domain.NewMeeting(userID, "Original Meeting", domain.CadenceWeekly, 7, 30*time.Minute, 10*time.Hour)
	require.NoError(t, err)
	err = repo.Save(ctx, meeting)
	require.NoError(t, err)

	// Reload, modify, and save again
	found, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)

	err = found.SetName("Updated Meeting")
	require.NoError(t, err)
	err = found.SetCadence(domain.CadenceBiweekly, 14)
	require.NoError(t, err)

	err = repo.Save(ctx, found)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated Meeting", updated.Name())
	assert.Equal(t, domain.CadenceBiweekly, updated.Cadence())
	assert.Equal(t, 14, updated.CadenceDays())
}

func TestSQLiteMeetingRepository_FindByID_NotFound(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, uuid.New())
	assert.NoError(t, err) // Returns nil, nil for not found
	assert.Nil(t, found)
}

func TestSQLiteMeetingRepository_FindByUserID(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	otherUserID := uuid.New()
	createMeetingTestUser(t, sqlDB, otherUserID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// Create meetings for the user
	meeting1, err := domain.NewMeeting(userID, "Meeting 1", domain.CadenceWeekly, 7, 30*time.Minute, 9*time.Hour)
	require.NoError(t, err)
	meeting2, err := domain.NewMeeting(userID, "Meeting 2", domain.CadenceBiweekly, 14, 45*time.Minute, 14*time.Hour)
	require.NoError(t, err)
	meeting3, err := domain.NewMeeting(otherUserID, "Other User Meeting", domain.CadenceMonthly, 30, 60*time.Minute, 10*time.Hour)
	require.NoError(t, err)

	require.NoError(t, repo.Save(ctx, meeting1))
	require.NoError(t, repo.Save(ctx, meeting2))
	require.NoError(t, repo.Save(ctx, meeting3))

	// Find meetings for the user
	meetings, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, meetings, 2)

	// Verify we got the right meetings
	meetingIDs := make(map[uuid.UUID]bool)
	for _, m := range meetings {
		meetingIDs[m.ID()] = true
	}
	assert.True(t, meetingIDs[meeting1.ID()])
	assert.True(t, meetingIDs[meeting2.ID()])
	assert.False(t, meetingIDs[meeting3.ID()])
}

func TestSQLiteMeetingRepository_FindActiveByUserID(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// Create meetings with different archived status
	activeMeeting, err := domain.NewMeeting(userID, "Active Meeting", domain.CadenceWeekly, 7, 30*time.Minute, 10*time.Hour)
	require.NoError(t, err)
	archivedMeeting, err := domain.NewMeeting(userID, "Archived Meeting", domain.CadenceBiweekly, 14, 45*time.Minute, 14*time.Hour)
	require.NoError(t, err)
	archivedMeeting.Archive()

	require.NoError(t, repo.Save(ctx, activeMeeting))
	require.NoError(t, repo.Save(ctx, archivedMeeting))

	// Find only active meetings
	meetings, err := repo.FindActiveByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, meetings, 1)
	assert.Equal(t, activeMeeting.ID(), meetings[0].ID())
	assert.False(t, meetings[0].IsArchived())
}

func TestSQLiteMeetingRepository_MarkHeld(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// Create a meeting
	meeting, err := domain.NewMeeting(userID, "Recurring Meeting", domain.CadenceWeekly, 7, 30*time.Minute, 10*time.Hour)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, meeting))

	// Mark it as held
	heldTime := time.Now().Truncate(time.Second)
	err = meeting.MarkHeld(heldTime)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, meeting))

	// Verify last_held_at is persisted
	found, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	require.NotNil(t, found.LastHeldAt())
	assert.Equal(t, heldTime.Unix(), found.LastHeldAt().Unix())
}

func TestSQLiteMeetingRepository_Archive(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// Create a meeting
	meeting, err := domain.NewMeeting(userID, "Meeting to Archive", domain.CadenceWeekly, 7, 30*time.Minute, 10*time.Hour)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, meeting))
	assert.False(t, meeting.IsArchived())

	// Archive it
	meeting.Archive()
	require.NoError(t, repo.Save(ctx, meeting))

	// Verify it's archived
	found, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	assert.True(t, found.IsArchived())
}

func TestSQLiteMeetingRepository_DifferentCadences(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	testCases := []struct {
		name     string
		cadence  domain.Cadence
		days     int
		duration time.Duration
	}{
		{"Weekly", domain.CadenceWeekly, 7, 30 * time.Minute},
		{"Biweekly", domain.CadenceBiweekly, 14, 45 * time.Minute},
		{"Monthly", domain.CadenceMonthly, 30, 60 * time.Minute},
		{"Custom 10 days", domain.CadenceCustom, 10, 20 * time.Minute},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			meeting, err := domain.NewMeeting(userID, tc.name+" Meeting", tc.cadence, tc.days, tc.duration, 10*time.Hour)
			require.NoError(t, err)

			err = repo.Save(ctx, meeting)
			require.NoError(t, err)

			found, err := repo.FindByID(ctx, meeting.ID())
			require.NoError(t, err)
			assert.Equal(t, tc.cadence, found.Cadence())
			assert.Equal(t, tc.duration, found.Duration())
		})
	}
}

func TestSQLiteMeetingRepository_PreferredTime(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// Create meeting with preferred time
	preferredTime := 10 * time.Hour // 10:00 AM
	meeting, err := domain.NewMeeting(userID, "Morning Meeting", domain.CadenceWeekly, 7, 30*time.Minute, preferredTime)
	require.NoError(t, err)

	err = repo.Save(ctx, meeting)
	require.NoError(t, err)

	// Verify preferred time is persisted
	found, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	assert.Equal(t, preferredTime, found.PreferredTime())
}

func TestSQLiteMeetingRepository_FullCRUDCycle(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createMeetingTestUser(t, sqlDB, userID)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// CREATE
	meeting, err := domain.NewMeeting(userID, "Full Cycle Meeting", domain.CadenceWeekly, 7, 45*time.Minute, 14*time.Hour)
	require.NoError(t, err)

	err = repo.Save(ctx, meeting)
	require.NoError(t, err)

	// READ
	found, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	assert.Equal(t, "Full Cycle Meeting", found.Name())
	assert.Equal(t, domain.CadenceWeekly, found.Cadence())
	assert.Equal(t, 45*time.Minute, found.Duration())
	assert.Equal(t, 14*time.Hour, found.PreferredTime())
	assert.Nil(t, found.LastHeldAt())
	assert.False(t, found.IsArchived())

	// UPDATE - Mark as held
	heldTime := time.Now().Truncate(time.Second)
	err = found.MarkHeld(heldTime)
	require.NoError(t, err)
	err = repo.Save(ctx, found)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	require.NotNil(t, updated.LastHeldAt())
	assert.Equal(t, heldTime.Unix(), updated.LastHeldAt().Unix())

	// UPDATE - Change cadence
	err = updated.SetCadence(domain.CadenceBiweekly, 14)
	require.NoError(t, err)
	err = repo.Save(ctx, updated)
	require.NoError(t, err)

	// Verify cadence update
	withNewCadence, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	assert.Equal(t, domain.CadenceBiweekly, withNewCadence.Cadence())
	assert.Equal(t, 14, withNewCadence.CadenceDays())

	// UPDATE - Archive
	withNewCadence.Archive()
	err = repo.Save(ctx, withNewCadence)
	require.NoError(t, err)

	// Verify archive
	archived, err := repo.FindByID(ctx, meeting.ID())
	require.NoError(t, err)
	assert.True(t, archived.IsArchived())

	// Verify it's excluded from active list
	active, err := repo.FindActiveByUserID(ctx, userID)
	require.NoError(t, err)
	for _, m := range active {
		assert.NotEqual(t, meeting.ID(), m.ID(), "Archived meeting should not appear in active list")
	}
}

func TestSQLiteMeetingRepository_MultipleUsers(t *testing.T) {
	sqlDB := setupMeetingTestDB(t)
	defer sqlDB.Close()

	user1 := uuid.New()
	user2 := uuid.New()
	createMeetingTestUser(t, sqlDB, user1)
	createMeetingTestUser(t, sqlDB, user2)

	repo := NewSQLiteMeetingRepository(sqlDB)
	ctx := context.Background()

	// Create meetings for both users
	user1Meeting1, err := domain.NewMeeting(user1, "User1 Meeting A", domain.CadenceWeekly, 7, 30*time.Minute, 9*time.Hour)
	require.NoError(t, err)
	user1Meeting2, err := domain.NewMeeting(user1, "User1 Meeting B", domain.CadenceBiweekly, 14, 45*time.Minute, 14*time.Hour)
	require.NoError(t, err)
	user2Meeting1, err := domain.NewMeeting(user2, "User2 Meeting X", domain.CadenceMonthly, 30, 60*time.Minute, 10*time.Hour)
	require.NoError(t, err)

	require.NoError(t, repo.Save(ctx, user1Meeting1))
	require.NoError(t, repo.Save(ctx, user1Meeting2))
	require.NoError(t, repo.Save(ctx, user2Meeting1))

	// Verify user1 gets only their meetings
	meetings1, err := repo.FindByUserID(ctx, user1)
	require.NoError(t, err)
	assert.Len(t, meetings1, 2)

	// Verify user2 gets only their meetings
	meetings2, err := repo.FindByUserID(ctx, user2)
	require.NoError(t, err)
	assert.Len(t, meetings2, 1)
}

func TestBoolToInt64(t *testing.T) {
	assert.Equal(t, int64(1), boolToInt64(true))
	assert.Equal(t, int64(0), boolToInt64(false))
}
