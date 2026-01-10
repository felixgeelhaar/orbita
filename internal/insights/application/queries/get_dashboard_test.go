package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockSnapshotRepo is a mock implementation of domain.SnapshotRepository.
type mockSnapshotRepo struct {
	mock.Mock
}

func (m *mockSnapshotRepo) Save(ctx context.Context, snapshot *domain.ProductivitySnapshot) error {
	args := m.Called(ctx, snapshot)
	return args.Error(0)
}

func (m *mockSnapshotRepo) GetByDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.ProductivitySnapshot, error) {
	args := m.Called(ctx, userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProductivitySnapshot), args.Error(1)
}

func (m *mockSnapshotRepo) GetDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivitySnapshot, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProductivitySnapshot), args.Error(1)
}

func (m *mockSnapshotRepo) GetLatest(ctx context.Context, userID uuid.UUID) (*domain.ProductivitySnapshot, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProductivitySnapshot), args.Error(1)
}

func (m *mockSnapshotRepo) GetRecent(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivitySnapshot, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProductivitySnapshot), args.Error(1)
}

func (m *mockSnapshotRepo) GetAverageScore(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	args := m.Called(ctx, userID, start, end)
	return args.Int(0), args.Error(1)
}

// mockSessionRepo is a mock implementation of domain.SessionRepository.
type mockSessionRepo struct {
	mock.Mock
}

func (m *mockSessionRepo) Create(ctx context.Context, session *domain.TimeSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockSessionRepo) Update(ctx context.Context, session *domain.TimeSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.TimeSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TimeSession), args.Error(1)
}

func (m *mockSessionRepo) GetActive(ctx context.Context, userID uuid.UUID) (*domain.TimeSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TimeSession), args.Error(1)
}

func (m *mockSessionRepo) GetByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.TimeSession, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TimeSession), args.Error(1)
}

func (m *mockSessionRepo) GetByType(ctx context.Context, userID uuid.UUID, sessionType domain.SessionType, limit int) ([]*domain.TimeSession, error) {
	args := m.Called(ctx, userID, sessionType, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TimeSession), args.Error(1)
}

func (m *mockSessionRepo) GetTotalFocusMinutes(ctx context.Context, userID uuid.UUID, start, end time.Time) (int, error) {
	args := m.Called(ctx, userID, start, end)
	return args.Int(0), args.Error(1)
}

func (m *mockSessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

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

func createTestSnapshot(userID uuid.UUID, date time.Time, score int) *domain.ProductivitySnapshot {
	return &domain.ProductivitySnapshot{
		ID:                 uuid.New(),
		UserID:             userID,
		SnapshotDate:       date,
		ProductivityScore:  score,
		TasksCompleted:     5,
		HabitsCompleted:    3,
		TotalFocusMinutes:  120,
		TaskCompletionRate: 0.8,
		HabitCompletionRate: 0.75,
		PeakHours:          []domain.PeakHour{{Hour: 10, Completions: 5}},
		TimeByCategory:     map[string]int{"work": 60},
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

func createTestWeeklySummary(userID uuid.UUID, weekStart time.Time) *domain.WeeklySummary {
	return &domain.WeeklySummary{
		ID:                        uuid.New(),
		UserID:                    userID,
		WeekStart:                 weekStart,
		WeekEnd:                   weekStart.AddDate(0, 0, 6),
		TotalTasksCompleted:       25,
		TotalHabitsCompleted:      15,
		TotalBlocksCompleted:      20,
		TotalFocusMinutes:         600,
		AvgDailyProductivityScore: 75.0,
		AvgDailyFocusMinutes:      86,
		CreatedAt:                 time.Now(),
	}
}

func createTestSession(userID uuid.UUID, title string) *domain.TimeSession {
	return domain.NewTimeSession(userID, domain.SessionTypeFocus, title)
}

func TestGetDashboardHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("returns complete dashboard with all data", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		handler := NewGetDashboardHandler(snapshotRepo, sessionRepo, summaryRepo, goalRepo)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		snapshot := createTestSnapshot(userID, today, 80)
		weeklySummary := createTestWeeklySummary(userID, today.AddDate(0, 0, -int(today.Weekday())+1))
		session := createTestSession(userID, "Focus Session")
		goal, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 5, domain.PeriodTypeDaily)
		goals := []*domain.ProductivityGoal{goal}
		snapshots := []*domain.ProductivitySnapshot{snapshot}

		snapshotRepo.On("GetByDate", mock.Anything, userID, mock.Anything).Return(snapshot, nil)
		summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(weeklySummary, nil)
		sessionRepo.On("GetActive", mock.Anything, userID).Return(session, nil)
		goalRepo.On("GetActive", mock.Anything, userID).Return(goals, nil)
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(snapshots, nil)
		snapshotRepo.On("GetAverageScore", mock.Anything, userID, mock.Anything, mock.Anything).Return(75, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(300, nil)

		query := GetDashboardQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Today)
		assert.Equal(t, 80, result.Today.ProductivityScore)
		assert.NotNil(t, result.ThisWeek)
		assert.NotNil(t, result.ActiveSession)
		assert.Len(t, result.ActiveGoals, 1)
		assert.Len(t, result.RecentSnapshots, 1)
		assert.Equal(t, 75, result.AvgProductivityScore)
		assert.Equal(t, 300, result.TotalFocusThisWeek)

		snapshotRepo.AssertExpectations(t)
		sessionRepo.AssertExpectations(t)
		summaryRepo.AssertExpectations(t)
		goalRepo.AssertExpectations(t)
	})

	t.Run("returns dashboard with empty data when no records exist", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		handler := NewGetDashboardHandler(snapshotRepo, sessionRepo, summaryRepo, goalRepo)

		snapshotRepo.On("GetByDate", mock.Anything, userID, mock.Anything).Return(nil, errors.New("not found"))
		summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, errors.New("not found"))
		sessionRepo.On("GetActive", mock.Anything, userID).Return(nil, errors.New("not found"))
		goalRepo.On("GetActive", mock.Anything, userID).Return([]*domain.ProductivityGoal{}, nil)
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil)
		snapshotRepo.On("GetAverageScore", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)

		query := GetDashboardQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.Today)
		assert.Nil(t, result.ThisWeek)
		assert.Nil(t, result.ActiveSession)
		assert.Empty(t, result.ActiveGoals)
		assert.Empty(t, result.RecentSnapshots)
		assert.Equal(t, 0, result.AvgProductivityScore)
		assert.Equal(t, 0, result.TotalFocusThisWeek)
	})

	t.Run("handles partial data availability", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		handler := NewGetDashboardHandler(snapshotRepo, sessionRepo, summaryRepo, goalRepo)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		snapshot := createTestSnapshot(userID, today, 65)

		snapshotRepo.On("GetByDate", mock.Anything, userID, mock.Anything).Return(snapshot, nil)
		summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, errors.New("not found"))
		sessionRepo.On("GetActive", mock.Anything, userID).Return(nil, nil)
		goalRepo.On("GetActive", mock.Anything, userID).Return(nil, errors.New("repo error"))
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil)
		snapshotRepo.On("GetAverageScore", mock.Anything, userID, mock.Anything, mock.Anything).Return(65, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(180, nil)

		query := GetDashboardQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Today)
		assert.Nil(t, result.ThisWeek)
		assert.Nil(t, result.ActiveSession)
		assert.Empty(t, result.ActiveGoals)
		assert.Equal(t, 65, result.AvgProductivityScore)
		assert.Equal(t, 180, result.TotalFocusThisWeek)
	})

	t.Run("returns multiple active goals", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		handler := NewGetDashboardHandler(snapshotRepo, sessionRepo, summaryRepo, goalRepo)

		goal1, _ := domain.NewProductivityGoal(userID, domain.GoalTypeDailyTasks, 5, domain.PeriodTypeDaily)
		goal2, _ := domain.NewProductivityGoal(userID, domain.GoalTypeWeeklyFocusMinutes, 300, domain.PeriodTypeWeekly)
		goal3, _ := domain.NewProductivityGoal(userID, domain.GoalTypeMonthlyTasks, 20, domain.PeriodTypeMonthly)
		goals := []*domain.ProductivityGoal{goal1, goal2, goal3}

		snapshotRepo.On("GetByDate", mock.Anything, userID, mock.Anything).Return(nil, nil)
		summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, nil)
		sessionRepo.On("GetActive", mock.Anything, userID).Return(nil, nil)
		goalRepo.On("GetActive", mock.Anything, userID).Return(goals, nil)
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil)
		snapshotRepo.On("GetAverageScore", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)

		query := GetDashboardQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.ActiveGoals, 3)
		assert.Equal(t, domain.GoalTypeDailyTasks, result.ActiveGoals[0].GoalType)
		assert.Equal(t, domain.GoalTypeWeeklyFocusMinutes, result.ActiveGoals[1].GoalType)
		assert.Equal(t, domain.GoalTypeMonthlyTasks, result.ActiveGoals[2].GoalType)
	})

	t.Run("returns recent snapshots for 7 days", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		handler := NewGetDashboardHandler(snapshotRepo, sessionRepo, summaryRepo, goalRepo)

		now := time.Now()
		var snapshots []*domain.ProductivitySnapshot
		for i := 0; i < 7; i++ {
			date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -i)
			snapshots = append(snapshots, createTestSnapshot(userID, date, 70+i))
		}

		snapshotRepo.On("GetByDate", mock.Anything, userID, mock.Anything).Return(snapshots[0], nil)
		summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, nil)
		sessionRepo.On("GetActive", mock.Anything, userID).Return(nil, nil)
		goalRepo.On("GetActive", mock.Anything, userID).Return([]*domain.ProductivityGoal{}, nil)
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(snapshots, nil)
		snapshotRepo.On("GetAverageScore", mock.Anything, userID, mock.Anything, mock.Anything).Return(73, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)

		query := GetDashboardQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.RecentSnapshots, 7)
		assert.Equal(t, 73, result.AvgProductivityScore)
	})
}

func TestStartOfWeek(t *testing.T) {
	t.Run("returns Monday for a Wednesday", func(t *testing.T) {
		// Wednesday Jan 8, 2025
		wed := time.Date(2025, 1, 8, 15, 30, 0, 0, time.UTC)
		monday := startOfWeek(wed)

		// Should be Monday Jan 6, 2025
		assert.Equal(t, time.Monday, monday.Weekday())
		assert.Equal(t, 6, monday.Day())
		assert.Equal(t, 0, monday.Hour())
		assert.Equal(t, 0, monday.Minute())
	})

	t.Run("returns same day for Monday", func(t *testing.T) {
		// Monday Jan 6, 2025
		mon := time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)
		monday := startOfWeek(mon)

		assert.Equal(t, time.Monday, monday.Weekday())
		assert.Equal(t, 6, monday.Day())
	})

	t.Run("returns previous Monday for Sunday", func(t *testing.T) {
		// Sunday Jan 12, 2025
		sun := time.Date(2025, 1, 12, 23, 59, 0, 0, time.UTC)
		monday := startOfWeek(sun)

		assert.Equal(t, time.Monday, monday.Weekday())
		assert.Equal(t, 6, monday.Day())
	})
}

func TestNewGetDashboardHandler(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	sessionRepo := new(mockSessionRepo)
	summaryRepo := new(mockSummaryRepo)
	goalRepo := new(mockGoalRepo)

	handler := NewGetDashboardHandler(snapshotRepo, sessionRepo, summaryRepo, goalRepo)

	require.NotNil(t, handler)
}
