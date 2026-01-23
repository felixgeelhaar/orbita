package commands

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockSummaryRepo is a mock implementation of domain.SummaryRepository.
type mockSummaryRepo struct {
	mock.Mock
}

func (m *mockSummaryRepo) Save(ctx context.Context, summary *domain.WeeklySummary) error {
	args := m.Called(ctx, summary)
	return args.Error(0)
}

func (m *mockSummaryRepo) GetByWeek(ctx context.Context, userID uuid.UUID, weekStart time.Time) (*domain.WeeklySummary, error) {
	args := m.Called(ctx, userID, weekStart)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WeeklySummary), args.Error(1)
}

func (m *mockSummaryRepo) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.WeeklySummary, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WeeklySummary), args.Error(1)
}

func (m *mockSummaryRepo) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.WeeklySummary, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WeeklySummary), args.Error(1)
}

// Helper to create test snapshot
func createTestSnapshotForSummary(userID uuid.UUID, date time.Time, score int) *domain.ProductivitySnapshot {
	snapshot := domain.NewProductivitySnapshot(userID, date)
	snapshot.ProductivityScore = score
	snapshot.TasksCompleted = 5
	snapshot.HabitsCompleted = 3
	snapshot.BlocksCompleted = 4
	snapshot.TotalFocusMinutes = 120
	snapshot.LongestStreak = 7
	return snapshot
}

func TestComputeWeeklySummaryHandler_Success(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	summaryRepo := new(mockSummaryRepo)
	sessionRepo := new(mockSessionRepo)

	userID := uuid.New()
	weekStart := normalizeToMonday(time.Now().AddDate(0, 0, -7))

	// Create week's worth of snapshots
	snapshots := make([]*domain.ProductivitySnapshot, 7)
	for i := 0; i < 7; i++ {
		date := weekStart.AddDate(0, 0, i)
		snapshots[i] = createTestSnapshotForSummary(userID, date, 60+i*5)
	}

	snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(snapshots, nil)
	summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, nil) // No previous summary
	summaryRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.WeeklySummary")).Return(nil)

	handler := NewComputeWeeklySummaryHandler(snapshotRepo, summaryRepo, sessionRepo)

	result, err := handler.Handle(context.Background(), ComputeWeeklySummaryCommand{
		UserID:    userID,
		WeekStart: weekStart,
	})

	require.NoError(t, err)
	assert.Equal(t, 7, result.DaysWithData)
	assert.NotNil(t, result.Summary)
	assert.Equal(t, 35, result.Summary.TotalTasksCompleted)  // 5 * 7
	assert.Equal(t, 21, result.Summary.TotalHabitsCompleted) // 3 * 7
	assert.Equal(t, 28, result.Summary.TotalBlocksCompleted) // 4 * 7
	assert.Equal(t, 840, result.Summary.TotalFocusMinutes)   // 120 * 7

	snapshotRepo.AssertExpectations(t)
	summaryRepo.AssertExpectations(t)
}

func TestComputeWeeklySummaryHandler_NoData(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	summaryRepo := new(mockSummaryRepo)
	sessionRepo := new(mockSessionRepo)

	userID := uuid.New()
	weekStart := normalizeToMonday(time.Now())

	snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil)
	summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, nil)
	summaryRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.WeeklySummary")).Return(nil)

	handler := NewComputeWeeklySummaryHandler(snapshotRepo, summaryRepo, sessionRepo)

	result, err := handler.Handle(context.Background(), ComputeWeeklySummaryCommand{
		UserID:    userID,
		WeekStart: weekStart,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.DaysWithData)
	assert.NotNil(t, result.Summary)
	assert.Equal(t, 0, result.Summary.TotalTasksCompleted)

	snapshotRepo.AssertExpectations(t)
}

func TestComputeWeeklySummaryHandler_PartialWeek(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	summaryRepo := new(mockSummaryRepo)
	sessionRepo := new(mockSessionRepo)

	userID := uuid.New()
	weekStart := normalizeToMonday(time.Now().AddDate(0, 0, -14))

	// Only 3 days of data
	snapshots := make([]*domain.ProductivitySnapshot, 3)
	for i := 0; i < 3; i++ {
		date := weekStart.AddDate(0, 0, i)
		snapshots[i] = createTestSnapshotForSummary(userID, date, 70)
	}

	snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(snapshots, nil)
	summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, nil)
	summaryRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.WeeklySummary")).Return(nil)

	handler := NewComputeWeeklySummaryHandler(snapshotRepo, summaryRepo, sessionRepo)

	result, err := handler.Handle(context.Background(), ComputeWeeklySummaryCommand{
		UserID:    userID,
		WeekStart: weekStart,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.DaysWithData)
	assert.True(t, result.IsComplete) // Past week is complete

	snapshotRepo.AssertExpectations(t)
}

func TestComputeWeeklySummaryHandler_CalculatesTrends(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	summaryRepo := new(mockSummaryRepo)
	sessionRepo := new(mockSessionRepo)

	userID := uuid.New()
	currentWeekStart := normalizeToMonday(time.Now().AddDate(0, 0, -7))
	previousWeekStart := normalizeToMonday(time.Now().AddDate(0, 0, -14))

	// Previous week summary with lower score
	previousSummary := domain.NewWeeklySummary(userID, previousWeekStart)
	previousSummary.SetAverages(60.0, 100)
	previousSummary.SetTotals(30, 20, 25, 700)

	// Current week with higher scores
	snapshots := make([]*domain.ProductivitySnapshot, 7)
	for i := 0; i < 7; i++ {
		date := currentWeekStart.AddDate(0, 0, i)
		snapshots[i] = createTestSnapshotForSummary(userID, date, 80)
	}

	snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(snapshots, nil)
	summaryRepo.On("GetByWeek", mock.Anything, userID, previousWeekStart).Return(previousSummary, nil)
	summaryRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.WeeklySummary")).Return(nil)

	handler := NewComputeWeeklySummaryHandler(snapshotRepo, summaryRepo, sessionRepo)

	result, err := handler.Handle(context.Background(), ComputeWeeklySummaryCommand{
		UserID:    userID,
		WeekStart: currentWeekStart,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.ProductivityTrend)
	assert.Contains(t, result.ProductivityTrend, "improved")

	snapshotRepo.AssertExpectations(t)
	summaryRepo.AssertExpectations(t)
}

func TestComputeWeeklySummaryHandler_FindsBestWorstDays(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	summaryRepo := new(mockSummaryRepo)
	sessionRepo := new(mockSessionRepo)

	userID := uuid.New()
	weekStart := normalizeToMonday(time.Now().AddDate(0, 0, -7))

	// Create varied scores
	scores := []int{50, 70, 90, 60, 80, 40, 75}
	snapshots := make([]*domain.ProductivitySnapshot, 7)
	for i, score := range scores {
		date := weekStart.AddDate(0, 0, i)
		snapshots[i] = createTestSnapshotForSummary(userID, date, score)
	}

	snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(snapshots, nil)
	summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, nil)
	summaryRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.WeeklySummary")).Return(nil)

	handler := NewComputeWeeklySummaryHandler(snapshotRepo, summaryRepo, sessionRepo)

	result, err := handler.Handle(context.Background(), ComputeWeeklySummaryCommand{
		UserID:    userID,
		WeekStart: weekStart,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Summary.MostProductiveDay)
	assert.NotNil(t, result.Summary.LeastProductiveDay)

	snapshotRepo.AssertExpectations(t)
}

func TestComputeCurrentWeekSummaryHandler_Success(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	summaryRepo := new(mockSummaryRepo)
	sessionRepo := new(mockSessionRepo)

	userID := uuid.New()

	snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil)
	summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, nil)
	summaryRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.WeeklySummary")).Return(nil)

	handler := NewComputeCurrentWeekSummaryHandler(snapshotRepo, summaryRepo, sessionRepo)

	result, err := handler.Handle(context.Background(), ComputeCurrentWeekSummaryCommand{
		UserID: userID,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Summary)
	assert.False(t, result.IsComplete) // Current week is not complete

	snapshotRepo.AssertExpectations(t)
}

func TestNormalizeToMonday(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Weekday
	}{
		{
			name:     "Monday stays Monday",
			input:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			expected: time.Monday,
		},
		{
			name:     "Wednesday goes to Monday",
			input:    time.Date(2024, 1, 17, 12, 0, 0, 0, time.UTC),
			expected: time.Monday,
		},
		{
			name:     "Sunday goes to Monday of same week",
			input:    time.Date(2024, 1, 21, 12, 0, 0, 0, time.UTC),
			expected: time.Monday,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeToMonday(tc.input)
			assert.Equal(t, tc.expected, result.Weekday())
		})
	}
}
