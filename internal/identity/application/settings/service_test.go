package settings

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository is a test double for the Repository interface.
type mockRepository struct {
	calendarIDs   map[uuid.UUID]string
	deleteMissing map[uuid.UUID]bool
	err           error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		calendarIDs:   make(map[uuid.UUID]string),
		deleteMissing: make(map[uuid.UUID]bool),
	}
}

func (m *mockRepository) GetCalendarID(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.calendarIDs[userID], nil
}

func (m *mockRepository) SetCalendarID(ctx context.Context, userID uuid.UUID, calendarID string) error {
	if m.err != nil {
		return m.err
	}
	m.calendarIDs[userID] = calendarID
	return nil
}

func (m *mockRepository) GetDeleteMissing(ctx context.Context, userID uuid.UUID) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.deleteMissing[userID], nil
}

func (m *mockRepository) SetDeleteMissing(ctx context.Context, userID uuid.UUID, deleteMissing bool) error {
	if m.err != nil {
		return m.err
	}
	m.deleteMissing[userID] = deleteMissing
	return nil
}

func TestNewService(t *testing.T) {
	repo := newMockRepository()
	service := NewService(repo)
	assert.NotNil(t, service)
}

func TestService_GetCalendarID(t *testing.T) {
	repo := newMockRepository()
	service := NewService(repo)
	ctx := context.Background()
	userID := uuid.New()

	// Get when not set
	calID, err := service.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, calID)

	// Set and get
	repo.calendarIDs[userID] = "test-calendar"
	calID, err = service.GetCalendarID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "test-calendar", calID)
}

func TestService_GetCalendarID_Error(t *testing.T) {
	repo := newMockRepository()
	repo.err = errors.New("database error")
	service := NewService(repo)
	ctx := context.Background()

	_, err := service.GetCalendarID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Equal(t, "database error", err.Error())
}

func TestService_SetCalendarID(t *testing.T) {
	repo := newMockRepository()
	service := NewService(repo)
	ctx := context.Background()
	userID := uuid.New()

	err := service.SetCalendarID(ctx, userID, "my-calendar")
	require.NoError(t, err)
	assert.Equal(t, "my-calendar", repo.calendarIDs[userID])
}

func TestService_SetCalendarID_Error(t *testing.T) {
	repo := newMockRepository()
	repo.err = errors.New("write error")
	service := NewService(repo)
	ctx := context.Background()

	err := service.SetCalendarID(ctx, uuid.New(), "calendar")
	assert.Error(t, err)
	assert.Equal(t, "write error", err.Error())
}

func TestService_GetDeleteMissing(t *testing.T) {
	repo := newMockRepository()
	service := NewService(repo)
	ctx := context.Background()
	userID := uuid.New()

	// Default is false
	del, err := service.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.False(t, del)

	// Set to true
	repo.deleteMissing[userID] = true
	del, err = service.GetDeleteMissing(ctx, userID)
	require.NoError(t, err)
	assert.True(t, del)
}

func TestService_GetDeleteMissing_Error(t *testing.T) {
	repo := newMockRepository()
	repo.err = errors.New("read error")
	service := NewService(repo)
	ctx := context.Background()

	_, err := service.GetDeleteMissing(ctx, uuid.New())
	assert.Error(t, err)
	assert.Equal(t, "read error", err.Error())
}

func TestService_SetDeleteMissing(t *testing.T) {
	repo := newMockRepository()
	service := NewService(repo)
	ctx := context.Background()
	userID := uuid.New()

	// Set to true
	err := service.SetDeleteMissing(ctx, userID, true)
	require.NoError(t, err)
	assert.True(t, repo.deleteMissing[userID])

	// Set to false
	err = service.SetDeleteMissing(ctx, userID, false)
	require.NoError(t, err)
	assert.False(t, repo.deleteMissing[userID])
}

func TestService_SetDeleteMissing_Error(t *testing.T) {
	repo := newMockRepository()
	repo.err = errors.New("write error")
	service := NewService(repo)
	ctx := context.Background()

	err := service.SetDeleteMissing(ctx, uuid.New(), true)
	assert.Error(t, err)
	assert.Equal(t, "write error", err.Error())
}

func TestService_MultipleUsers(t *testing.T) {
	repo := newMockRepository()
	service := NewService(repo)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// Set different settings for each user
	require.NoError(t, service.SetCalendarID(ctx, user1, "user1-cal"))
	require.NoError(t, service.SetCalendarID(ctx, user2, "user2-cal"))
	require.NoError(t, service.SetDeleteMissing(ctx, user1, true))
	require.NoError(t, service.SetDeleteMissing(ctx, user2, false))

	// Verify isolation
	cal1, err := service.GetCalendarID(ctx, user1)
	require.NoError(t, err)
	assert.Equal(t, "user1-cal", cal1)

	cal2, err := service.GetCalendarID(ctx, user2)
	require.NoError(t, err)
	assert.Equal(t, "user2-cal", cal2)

	del1, err := service.GetDeleteMissing(ctx, user1)
	require.NoError(t, err)
	assert.True(t, del1)

	del2, err := service.GetDeleteMissing(ctx, user2)
	require.NoError(t, err)
	assert.False(t, del2)
}
