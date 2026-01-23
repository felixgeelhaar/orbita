package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteRescheduleAttemptRepository_Create(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	// Create a schedule first
	scheduleRepo := NewSQLiteScheduleRepository(sqlDB)
	scheduleDate := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, scheduleDate)

	startTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 9, 0, 0, 0, time.UTC)
	endTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 10, 0, 0, 0, time.UTC)
	block, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test Block", startTime, endTime)
	require.NoError(t, err)

	err = scheduleRepo.Save(context.Background(), schedule)
	require.NoError(t, err)

	// Now test reschedule attempt repository
	repo := NewSQLiteRescheduleAttemptRepository(sqlDB)
	ctx := context.Background()

	attempt := domain.RescheduleAttempt{
		ID:            uuid.New(),
		UserID:        userID,
		ScheduleID:    schedule.ID(),
		BlockID:       block.ID(),
		AttemptType:   domain.RescheduleAttemptAutoMissed,
		Success:       true,
		FailureReason: "",
		OldStart:      startTime,
		OldEnd:        endTime,
		NewStart:      timePtr(startTime.Add(time.Hour)),
		NewEnd:        timePtr(endTime.Add(time.Hour)),
		AttemptedAt:   time.Now(),
	}

	err = repo.Create(ctx, attempt)
	require.NoError(t, err)
}

func TestSQLiteRescheduleAttemptRepository_Create_WithFailure(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	// Create a schedule first
	scheduleRepo := NewSQLiteScheduleRepository(sqlDB)
	scheduleDate := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, scheduleDate)

	startTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 9, 0, 0, 0, time.UTC)
	endTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 10, 0, 0, 0, time.UTC)
	block, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test Block", startTime, endTime)
	require.NoError(t, err)

	err = scheduleRepo.Save(context.Background(), schedule)
	require.NoError(t, err)

	repo := NewSQLiteRescheduleAttemptRepository(sqlDB)
	ctx := context.Background()

	attempt := domain.RescheduleAttempt{
		ID:            uuid.New(),
		UserID:        userID,
		ScheduleID:    schedule.ID(),
		BlockID:       block.ID(),
		AttemptType:   domain.RescheduleAttemptAutoMissed,
		Success:       false,
		FailureReason: "No available slots",
		OldStart:      startTime,
		OldEnd:        endTime,
		NewStart:      nil, // No new time on failure
		NewEnd:        nil,
		AttemptedAt:   time.Now(),
	}

	err = repo.Create(ctx, attempt)
	require.NoError(t, err)
}

func TestSQLiteRescheduleAttemptRepository_ListByUserAndDate(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	// Create a schedule
	scheduleRepo := NewSQLiteScheduleRepository(sqlDB)
	scheduleDate := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, scheduleDate)

	startTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 9, 0, 0, 0, time.UTC)
	endTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 10, 0, 0, 0, time.UTC)
	block, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test Block", startTime, endTime)
	require.NoError(t, err)

	err = scheduleRepo.Save(context.Background(), schedule)
	require.NoError(t, err)

	repo := NewSQLiteRescheduleAttemptRepository(sqlDB)
	ctx := context.Background()

	// Create multiple attempts
	attempt1 := domain.RescheduleAttempt{
		ID:          uuid.New(),
		UserID:      userID,
		ScheduleID:  schedule.ID(),
		BlockID:     block.ID(),
		AttemptType: domain.RescheduleAttemptAutoMissed,
		Success:     true,
		OldStart:    startTime,
		OldEnd:      endTime,
		NewStart:    timePtr(startTime.Add(time.Hour)),
		NewEnd:      timePtr(endTime.Add(time.Hour)),
		AttemptedAt: time.Now().Add(-time.Hour),
	}
	require.NoError(t, repo.Create(ctx, attempt1))

	attempt2 := domain.RescheduleAttempt{
		ID:          uuid.New(),
		UserID:      userID,
		ScheduleID:  schedule.ID(),
		BlockID:     block.ID(),
		AttemptType: domain.RescheduleAttemptManual,
		Success:     true,
		OldStart:    startTime.Add(time.Hour),
		OldEnd:      endTime.Add(time.Hour),
		NewStart:    timePtr(startTime.Add(2 * time.Hour)),
		NewEnd:      timePtr(endTime.Add(2 * time.Hour)),
		AttemptedAt: time.Now(),
	}
	require.NoError(t, repo.Create(ctx, attempt2))

	// List attempts for the schedule date
	attempts, err := repo.ListByUserAndDate(ctx, userID, scheduleDate)
	require.NoError(t, err)
	assert.Len(t, attempts, 2)

	// Verify ordering by attempted_at
	assert.True(t, attempts[0].AttemptedAt.Before(attempts[1].AttemptedAt))
}

func TestSQLiteRescheduleAttemptRepository_ListByUserAndDate_Empty(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	repo := NewSQLiteRescheduleAttemptRepository(sqlDB)
	ctx := context.Background()

	attempts, err := repo.ListByUserAndDate(ctx, userID, time.Now())
	require.NoError(t, err)
	assert.Empty(t, attempts)
}

func TestSQLiteRescheduleAttemptRepository_AttemptTypes(t *testing.T) {
	sqlDB := setupScheduleTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createScheduleTestUser(t, sqlDB, userID)

	// Create a schedule
	scheduleRepo := NewSQLiteScheduleRepository(sqlDB)
	scheduleDate := time.Now().Truncate(24 * time.Hour)
	schedule := domain.NewSchedule(userID, scheduleDate)

	startTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 9, 0, 0, 0, time.UTC)
	endTime := time.Date(scheduleDate.Year(), scheduleDate.Month(), scheduleDate.Day(), 10, 0, 0, 0, time.UTC)
	block, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Test Block", startTime, endTime)
	require.NoError(t, err)

	err = scheduleRepo.Save(context.Background(), schedule)
	require.NoError(t, err)

	repo := NewSQLiteRescheduleAttemptRepository(sqlDB)
	ctx := context.Background()

	testCases := []domain.RescheduleAttemptType{
		domain.RescheduleAttemptAutoMissed,
		domain.RescheduleAttemptManual,
		domain.RescheduleAttemptAutoConflict,
	}

	for i, attemptType := range testCases {
		attempt := domain.RescheduleAttempt{
			ID:          uuid.New(),
			UserID:      userID,
			ScheduleID:  schedule.ID(),
			BlockID:     block.ID(),
			AttemptType: attemptType,
			Success:     true,
			OldStart:    startTime.Add(time.Duration(i) * time.Hour),
			OldEnd:      endTime.Add(time.Duration(i) * time.Hour),
			NewStart:    timePtr(startTime.Add(time.Duration(i+1) * time.Hour)),
			NewEnd:      timePtr(endTime.Add(time.Duration(i+1) * time.Hour)),
			AttemptedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		require.NoError(t, repo.Create(ctx, attempt))
	}

	attempts, err := repo.ListByUserAndDate(ctx, userID, scheduleDate)
	require.NoError(t, err)
	assert.Len(t, attempts, 3)

	// Verify all attempt types were saved correctly
	foundTypes := make(map[domain.RescheduleAttemptType]bool)
	for _, attempt := range attempts {
		foundTypes[attempt.AttemptType] = true
	}

	for _, tc := range testCases {
		assert.True(t, foundTypes[tc], "Attempt type %s should be present", tc)
	}
}

func TestBoolToInt_RescheduleAttempt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}

// timePtr returns a pointer to the given time.
func timePtr(t time.Time) *time.Time {
	return &t
}
