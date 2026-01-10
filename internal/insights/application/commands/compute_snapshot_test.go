package commands

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

// mockDataSource is a mock implementation of domain.AnalyticsDataSource.
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

func createCompletedFocusSession(userID uuid.UUID) *domain.TimeSession {
	session := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Focus")
	_ = session.Complete()
	return session
}

func TestComputeSnapshotHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully computes snapshot with all data", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		now := time.Now()
		date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		taskStats := &domain.TaskStats{
			Created:         10,
			Completed:       8,
			Overdue:         1,
			AvgDurationMins: 45,
		}
		blockStats := &domain.BlockStats{
			Scheduled:        5,
			Completed:        4,
			Missed:           1,
			ScheduledMinutes: 300,
			CompletedMinutes: 240,
		}
		habitStats := &domain.HabitStats{
			Due:           5,
			Completed:     4,
			LongestStreak: 10,
		}
		peakHours := []domain.PeakHour{
			{Hour: 10, Completions: 5},
			{Hour: 14, Completions: 3},
		}
		timeByCategory := map[string]int{
			"work":     180,
			"personal": 60,
		}
		focusSessions := []*domain.TimeSession{
			createCompletedFocusSession(userID),
			createCompletedFocusSession(userID),
		}

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(taskStats, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(blockStats, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(habitStats, nil)
		dataSource.On("GetPeakHours", mock.Anything, userID, mock.Anything, mock.Anything).Return(peakHours, nil)
		dataSource.On("GetTimeByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(timeByCategory, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(120, nil)
		sessionRepo.On("GetByDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(focusSessions, nil)
		snapshotRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.ProductivitySnapshot")).Return(nil)

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   date,
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, snapshot)
		assert.Equal(t, userID, snapshot.UserID)
		assert.Equal(t, 10, snapshot.TasksCreated)
		assert.Equal(t, 8, snapshot.TasksCompleted)
		assert.Equal(t, 1, snapshot.TasksOverdue)
		assert.Equal(t, 5, snapshot.BlocksScheduled)
		assert.Equal(t, 4, snapshot.BlocksCompleted)
		assert.Equal(t, 5, snapshot.HabitsDue)
		assert.Equal(t, 4, snapshot.HabitsCompleted)
		assert.Equal(t, 2, snapshot.FocusSessions)
		assert.Equal(t, 120, snapshot.TotalFocusMinutes)
		assert.Len(t, snapshot.PeakHours, 2)
		assert.Equal(t, 180, snapshot.TimeByCategory["work"])

		snapshotRepo.AssertExpectations(t)
		sessionRepo.AssertExpectations(t)
		dataSource.AssertExpectations(t)
	})

	t.Run("computes snapshot with nil stats", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetPeakHours", mock.Anything, userID, mock.Anything, mock.Anything).Return([]domain.PeakHour{}, nil)
		dataSource.On("GetTimeByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(map[string]int{}, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)
		sessionRepo.On("GetByDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.TimeSession{}, nil)
		snapshotRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.ProductivitySnapshot")).Return(nil)

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, snapshot)
		assert.Equal(t, 0, snapshot.TasksCreated)
		assert.Equal(t, 0, snapshot.BlocksScheduled)
		assert.Equal(t, 0, snapshot.HabitsDue)
		assert.Equal(t, 0, snapshot.FocusSessions)

		snapshotRepo.AssertExpectations(t)
	})

	t.Run("fails when GetTaskStats returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("fails when GetBlockStats returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
	})

	t.Run("fails when GetHabitStats returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
	})

	t.Run("fails when GetTotalFocusMinutes returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, errors.New("database error"))

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
	})

	t.Run("fails when GetByDateRange returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)
		sessionRepo.On("GetByDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
	})

	t.Run("fails when GetPeakHours returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)
		sessionRepo.On("GetByDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.TimeSession{}, nil)
		dataSource.On("GetPeakHours", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
	})

	t.Run("fails when GetTimeByCategory returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)
		sessionRepo.On("GetByDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.TimeSession{}, nil)
		dataSource.On("GetPeakHours", mock.Anything, userID, mock.Anything, mock.Anything).Return([]domain.PeakHour{}, nil)
		dataSource.On("GetTimeByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
	})

	t.Run("fails when Save returns error", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(0, nil)
		sessionRepo.On("GetByDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return([]*domain.TimeSession{}, nil)
		dataSource.On("GetPeakHours", mock.Anything, userID, mock.Anything, mock.Anything).Return([]domain.PeakHour{}, nil)
		dataSource.On("GetTimeByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(map[string]int{}, nil)
		snapshotRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.ProductivitySnapshot")).Return(errors.New("save error"))

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
		assert.Contains(t, err.Error(), "save error")
	})

	t.Run("counts only completed focus sessions", func(t *testing.T) {
		snapshotRepo := new(mockSnapshotRepo)
		sessionRepo := new(mockSessionRepo)
		dataSource := new(mockDataSource)
		handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

		// Mix of focus sessions with different statuses
		completedFocus := createCompletedFocusSession(userID)
		activeFocus := domain.NewTimeSession(userID, domain.SessionTypeFocus, "Active Focus")
		taskSession := domain.NewTimeSession(userID, domain.SessionTypeTask, "Task")
		_ = taskSession.Complete()

		sessions := []*domain.TimeSession{completedFocus, activeFocus, taskSession}

		dataSource.On("GetTaskStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetBlockStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetHabitStats", mock.Anything, userID, mock.Anything, mock.Anything).Return(nil, nil)
		dataSource.On("GetPeakHours", mock.Anything, userID, mock.Anything, mock.Anything).Return([]domain.PeakHour{}, nil)
		dataSource.On("GetTimeByCategory", mock.Anything, userID, mock.Anything, mock.Anything).Return(map[string]int{}, nil)
		sessionRepo.On("GetTotalFocusMinutes", mock.Anything, userID, mock.Anything, mock.Anything).Return(60, nil)
		sessionRepo.On("GetByDateRange", mock.Anything, userID, mock.Anything, mock.Anything).Return(sessions, nil)
		snapshotRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.ProductivitySnapshot")).Return(nil)

		cmd := ComputeSnapshotCommand{
			UserID: userID,
			Date:   time.Now(),
		}

		snapshot, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, snapshot)
		assert.Equal(t, 1, snapshot.FocusSessions) // Only the completed focus session
	})
}

func TestNewComputeSnapshotHandler(t *testing.T) {
	snapshotRepo := new(mockSnapshotRepo)
	sessionRepo := new(mockSessionRepo)
	dataSource := new(mockDataSource)

	handler := NewComputeSnapshotHandler(snapshotRepo, sessionRepo, dataSource)

	require.NotNil(t, handler)
}
