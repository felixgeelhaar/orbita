package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/insights/application/commands"
	"github.com/felixgeelhaar/orbita/internal/insights/application/queries"
	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations

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

type mockGoalRepo struct {
	mock.Mock
}

func (m *mockGoalRepo) Create(ctx context.Context, goal *domain.ProductivityGoal) error {
	args := m.Called(ctx, goal)
	return args.Error(0)
}

func (m *mockGoalRepo) Update(ctx context.Context, goal *domain.ProductivityGoal) error {
	args := m.Called(ctx, goal)
	return args.Error(0)
}

func (m *mockGoalRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProductivityGoal, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProductivityGoal), args.Error(1)
}

func (m *mockGoalRepo) GetActive(ctx context.Context, userID uuid.UUID) ([]*domain.ProductivityGoal, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProductivityGoal), args.Error(1)
}

func (m *mockGoalRepo) GetByPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.ProductivityGoal, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProductivityGoal), args.Error(1)
}

func (m *mockGoalRepo) GetAchieved(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.ProductivityGoal, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProductivityGoal), args.Error(1)
}

func (m *mockGoalRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockDataSource struct {
	mock.Mock
}

func (m *mockDataSource) GetTaskStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.TaskStats, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TaskStats), args.Error(1)
}

func (m *mockDataSource) GetBlockStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.BlockStats, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.BlockStats), args.Error(1)
}

func (m *mockDataSource) GetHabitStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*domain.HabitStats, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.HabitStats), args.Error(1)
}

func (m *mockDataSource) GetPeakHours(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]domain.PeakHour, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PeakHour), args.Error(1)
}

func (m *mockDataSource) GetTimeByCategory(ctx context.Context, userID uuid.UUID, start, end time.Time) (map[string]int, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

// Tests

func TestNewService(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	sessionRepo := new(mockSessionRepo)
	summaryRepo := new(mockSummaryRepo)
	goalRepo := new(mockGoalRepo)
	dataSource := new(mockDataSource)

	svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

	require.NotNil(t, svc)
}

func TestService_StartSession(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully starts session", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		sessionRepo.On("GetActive", mock.Anything, userID).Return(nil, commands.ErrNotFound)
		sessionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(nil)

		cmd := commands.StartSessionCommand{
			UserID:      userID,
			SessionType: domain.SessionTypeFocus,
			Title:       "Deep work",
		}

		session, err := svc.StartSession(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, domain.SessionTypeFocus, session.SessionType)
		sessionRepo.AssertExpectations(t)
	})

	t.Run("fails when session already active", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		existingSession := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Existing")
		sessionRepo.On("GetActive", mock.Anything, userID).Return(existingSession, nil)

		cmd := commands.StartSessionCommand{
			UserID:      userID,
			SessionType: domain.SessionTypeFocus,
			Title:       "New session",
		}

		session, err := svc.StartSession(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, session)
		sessionRepo.AssertExpectations(t)
	})
}

func TestService_EndSession(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully ends session", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		activeSession := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Active session")
		sessionRepo.On("GetActive", mock.Anything, userID).Return(activeSession, nil)
		sessionRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.TimeSession")).Return(nil)

		cmd := commands.EndSessionCommand{
			UserID: userID,
			Notes:  "Session completed",
		}

		session, err := svc.EndSession(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, session)
		sessionRepo.AssertExpectations(t)
	})

	t.Run("fails when no active session", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		sessionRepo.On("GetActive", mock.Anything, userID).Return(nil, commands.ErrNoActiveSession)

		cmd := commands.EndSessionCommand{
			UserID: userID,
		}

		session, err := svc.EndSession(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, session)
		sessionRepo.AssertExpectations(t)
	})
}

func TestService_ComputeSnapshot(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully computes snapshot", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		today := time.Now().Truncate(24 * time.Hour)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(&domain.TaskStats{
			Created:   5,
			Completed: 3,
		}, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(&domain.BlockStats{
			Scheduled: 4,
			Completed: 3,
		}, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(&domain.HabitStats{
			Due:       3,
			Completed: 2,
		}, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(120, nil)
		sessionRepo.On("GetByDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.TimeSession{}, nil)
		dataSource.On("GetPeakHours", mock.Anything, userID, mock.Anything, mock.Anything).Return([]domain.PeakHour{}, nil)
		dataSource.On("GetTimeByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(map[string]int{}, nil)
		snapshotRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.ProductivitySnapshot")).Return(nil)

		cmd := commands.ComputeSnapshotCommand{
			UserID: userID,
			Date:   today,
		}

		snapshot, err := svc.ComputeSnapshot(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, snapshot)
		dataSource.AssertExpectations(t)
		sessionRepo.AssertExpectations(t)
		snapshotRepo.AssertExpectations(t)
	})
}

func TestService_CreateGoal(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully creates goal", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		goalRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(nil)

		cmd := commands.CreateGoalCommand{
			UserID:      userID,
			GoalType:    domain.GoalTypeDailyTasks,
			TargetValue: 5,
			PeriodType:  domain.PeriodTypeDaily,
		}

		goal, err := svc.CreateGoal(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, goal)
		assert.Equal(t, domain.GoalTypeDailyTasks, goal.GoalType)
		goalRepo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		goalRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.ProductivityGoal")).Return(errors.New("db error"))

		cmd := commands.CreateGoalCommand{
			UserID:      userID,
			GoalType:    domain.GoalTypeDailyTasks,
			TargetValue: 5,
			PeriodType:  domain.PeriodTypeDaily,
		}

		goal, err := svc.CreateGoal(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, goal)
		goalRepo.AssertExpectations(t)
	})
}

func TestService_GetDashboard(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully gets dashboard", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		// GetDashboard calls many methods - set up all needed mocks
		snapshotRepo.On("GetByDate", mock.Anything, userID, mock.Anything).Return(nil, commands.ErrNotFound)
		summaryRepo.On("GetByWeek", mock.Anything, userID, mock.Anything).Return(nil, commands.ErrNotFound)
		sessionRepo.On("GetActive", mock.Anything, userID).Return(nil, commands.ErrNotFound)
		goalRepo.On("GetActive", mock.Anything, userID).Return([]*domain.ProductivityGoal{}, nil)
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.ProductivitySnapshot{}, nil)
		snapshotRepo.On("GetAverageScore", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)

		query := queries.GetDashboardQuery{
			UserID: userID,
		}

		result, err := svc.GetDashboard(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		snapshotRepo.AssertExpectations(t)
		sessionRepo.AssertExpectations(t)
		summaryRepo.AssertExpectations(t)
		goalRepo.AssertExpectations(t)
	})
}

func TestService_GetTrends(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully gets trends", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		now := time.Now()
		snapshots := []*domain.ProductivitySnapshot{
			{UserID: userID, SnapshotDate: now.AddDate(0, 0, -1), ProductivityScore: 70},
			{UserID: userID, SnapshotDate: now, ProductivityScore: 75},
		}
		snapshotRepo.On("GetDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(snapshots, nil)

		query := queries.GetTrendsQuery{
			UserID: userID,
			Days:   7,
		}

		result, err := svc.GetTrends(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		snapshotRepo.AssertExpectations(t)
	})
}

func TestService_GetActiveGoals(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully gets active goals", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		goals := []*domain.ProductivityGoal{
			{UserID: userID, GoalType: domain.GoalTypeDailyTasks, TargetValue: 5},
		}
		goalRepo.On("GetActive", mock.Anything, userID).Return(goals, nil)

		query := queries.GetActiveGoalsQuery{
			UserID: userID,
		}

		result, err := svc.GetActiveGoals(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		goalRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no goals", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		goalRepo.On("GetActive", mock.Anything, userID).Return([]*domain.ProductivityGoal{}, nil)

		query := queries.GetActiveGoalsQuery{
			UserID: userID,
		}

		result, err := svc.GetActiveGoals(context.Background(), query)

		require.NoError(t, err)
		require.Empty(t, result)
		goalRepo.AssertExpectations(t)
	})
}

func TestService_GetAchievedGoals(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully gets achieved goals", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		summaryRepo := new(mockSummaryRepo)
		goalRepo := new(mockGoalRepo)
		dataSource := new(mockDataSource)

		svc := NewService(snapshotRepo, sessionRepo, summaryRepo, goalRepo, dataSource)

		goals := []*domain.ProductivityGoal{
			{UserID: userID, GoalType: domain.GoalTypeDailyTasks, TargetValue: 5, CurrentValue: 5},
		}
		goalRepo.On("GetAchieved", mock.Anything, userID, 10).Return(goals, nil)

		query := queries.GetAchievedGoalsQuery{
			UserID: userID,
			Limit:  10,
		}

		result, err := svc.GetAchievedGoals(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		goalRepo.AssertExpectations(t)
	})
}
