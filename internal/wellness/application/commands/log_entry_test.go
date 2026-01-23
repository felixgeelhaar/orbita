package commands

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/wellness/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockEntryRepo is a mock implementation of WellnessEntryRepository.
type mockEntryRepo struct {
	mock.Mock
}

func (m *mockEntryRepo) Create(ctx context.Context, entry *domain.WellnessEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *mockEntryRepo) Update(ctx context.Context, entry *domain.WellnessEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *mockEntryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.WellnessEntry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WellnessEntry), args.Error(1)
}

func (m *mockEntryRepo) GetByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]*domain.WellnessEntry, error) {
	args := m.Called(ctx, userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WellnessEntry), args.Error(1)
}

func (m *mockEntryRepo) GetByUserAndType(ctx context.Context, userID uuid.UUID, wellnessType domain.WellnessType, limit int) ([]*domain.WellnessEntry, error) {
	args := m.Called(ctx, userID, wellnessType, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WellnessEntry), args.Error(1)
}

func (m *mockEntryRepo) GetByUserDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*domain.WellnessEntry, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WellnessEntry), args.Error(1)
}

func (m *mockEntryRepo) GetLatestByType(ctx context.Context, userID uuid.UUID) (map[domain.WellnessType]*domain.WellnessEntry, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[domain.WellnessType]*domain.WellnessEntry), args.Error(1)
}

func (m *mockEntryRepo) GetAverageByType(ctx context.Context, userID uuid.UUID, wellnessType domain.WellnessType, start, end time.Time) (float64, error) {
	args := m.Called(ctx, userID, wellnessType, start, end)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockEntryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// mockGoalRepo is a mock implementation of WellnessGoalRepository.
type mockGoalRepo struct {
	mock.Mock
}

func (m *mockGoalRepo) Create(ctx context.Context, goal *domain.WellnessGoal) error {
	args := m.Called(ctx, goal)
	return args.Error(0)
}

func (m *mockGoalRepo) Update(ctx context.Context, goal *domain.WellnessGoal) error {
	args := m.Called(ctx, goal)
	return args.Error(0)
}

func (m *mockGoalRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.WellnessGoal, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WellnessGoal), args.Error(1)
}

func (m *mockGoalRepo) GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.WellnessGoal, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WellnessGoal), args.Error(1)
}

func (m *mockGoalRepo) GetByUserAndType(ctx context.Context, userID uuid.UUID, wellnessType domain.WellnessType) (*domain.WellnessGoal, error) {
	args := m.Called(ctx, userID, wellnessType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WellnessGoal), args.Error(1)
}

func (m *mockGoalRepo) GetActiveByUser(ctx context.Context, userID uuid.UUID) ([]*domain.WellnessGoal, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WellnessGoal), args.Error(1)
}

func (m *mockGoalRepo) GetAchievedByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.WellnessGoal, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WellnessGoal), args.Error(1)
}

func (m *mockGoalRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestLogWellnessEntryHandler_Success(t *testing.T) {
	entryRepo := new(mockEntryRepo)
	goalRepo := new(mockGoalRepo)

	userID := uuid.New()

	entryRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.WellnessEntry")).Return(nil)
	goalRepo.On("GetByUserAndType", mock.Anything, userID, domain.WellnessTypeMood).Return(nil, nil)

	handler := NewLogWellnessEntryHandler(entryRepo, goalRepo)

	result, err := handler.Handle(context.Background(), LogWellnessEntryCommand{
		UserID: userID,
		Type:   domain.WellnessTypeMood,
		Value:  7,
		Notes:  "Feeling good",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.EntryID)
	assert.Equal(t, domain.WellnessTypeMood, result.Type)
	assert.Equal(t, 7, result.Value)
	assert.Equal(t, domain.WellnessSourceManual, result.Source)

	entryRepo.AssertExpectations(t)
}

func TestLogWellnessEntryHandler_UpdatesGoal(t *testing.T) {
	entryRepo := new(mockEntryRepo)
	goalRepo := new(mockGoalRepo)

	userID := uuid.New()
	goal, _ := domain.NewWellnessGoal(userID, domain.WellnessTypeHydration, 8, domain.GoalFrequencyDaily)

	entryRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.WellnessEntry")).Return(nil)
	goalRepo.On("GetByUserAndType", mock.Anything, userID, domain.WellnessTypeHydration).Return(goal, nil)
	goalRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.WellnessGoal")).Return(nil)

	handler := NewLogWellnessEntryHandler(entryRepo, goalRepo)

	_, err := handler.Handle(context.Background(), LogWellnessEntryCommand{
		UserID: userID,
		Type:   domain.WellnessTypeHydration,
		Value:  2,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, goal.Current)

	entryRepo.AssertExpectations(t)
	goalRepo.AssertExpectations(t)
}

func TestLogWellnessEntryHandler_ValidationError(t *testing.T) {
	entryRepo := new(mockEntryRepo)
	goalRepo := new(mockGoalRepo)

	handler := NewLogWellnessEntryHandler(entryRepo, goalRepo)

	_, err := handler.Handle(context.Background(), LogWellnessEntryCommand{
		UserID: uuid.Nil, // Invalid
		Type:   domain.WellnessTypeMood,
		Value:  7,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user ID")
}

func TestWellnessCheckinHandler_Success(t *testing.T) {
	entryRepo := new(mockEntryRepo)
	goalRepo := new(mockGoalRepo)

	userID := uuid.New()
	mood := 7
	energy := 8
	sleep := 7

	entryRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.WellnessEntry")).Return(nil)
	goalRepo.On("GetByUserAndType", mock.Anything, userID, mock.AnythingOfType("domain.WellnessType")).Return(nil, nil)

	logHandler := NewLogWellnessEntryHandler(entryRepo, goalRepo)
	handler := NewWellnessCheckinHandler(logHandler)

	result, err := handler.Handle(context.Background(), WellnessCheckinCommand{
		UserID: userID,
		Mood:   &mood,
		Energy: &energy,
		Sleep:  &sleep,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.EntriesLogged)
	assert.Len(t, result.Entries, 3)

	entryRepo.AssertExpectations(t)
}

func TestWellnessCheckinHandler_PartialCheckin(t *testing.T) {
	entryRepo := new(mockEntryRepo)
	goalRepo := new(mockGoalRepo)

	userID := uuid.New()
	mood := 6

	entryRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.WellnessEntry")).Return(nil)
	goalRepo.On("GetByUserAndType", mock.Anything, userID, domain.WellnessTypeMood).Return(nil, nil)

	logHandler := NewLogWellnessEntryHandler(entryRepo, goalRepo)
	handler := NewWellnessCheckinHandler(logHandler)

	result, err := handler.Handle(context.Background(), WellnessCheckinCommand{
		UserID: userID,
		Mood:   &mood,
		// Other fields are nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.EntriesLogged)

	entryRepo.AssertExpectations(t)
}
