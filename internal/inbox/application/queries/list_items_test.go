package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockInboxRepo is a mock implementation of domain.InboxRepository.
type mockInboxRepo struct {
	mock.Mock
}

func (m *mockInboxRepo) Save(ctx context.Context, item domain.InboxItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *mockInboxRepo) ListByUser(ctx context.Context, userID uuid.UUID, includePromoted bool) ([]domain.InboxItem, error) {
	args := m.Called(ctx, userID, includePromoted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.InboxItem), args.Error(1)
}

func (m *mockInboxRepo) FindByID(ctx context.Context, userID, id uuid.UUID) (*domain.InboxItem, error) {
	args := m.Called(ctx, userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.InboxItem), args.Error(1)
}

func (m *mockInboxRepo) MarkPromoted(ctx context.Context, id uuid.UUID, promotedTo string, promotedID uuid.UUID, promotedAt time.Time) error {
	args := m.Called(ctx, id, promotedTo, promotedID, promotedAt)
	return args.Error(0)
}

func TestListInboxItemsHandler_Handle(t *testing.T) {
	userID := uuid.New()

	t.Run("successfully lists items", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewListInboxItemsHandler(repo)

		now := time.Now()
		items := []domain.InboxItem{
			{
				ID:             uuid.New(),
				UserID:         userID,
				Content:        "First item",
				Tags:           []string{"work"},
				Source:         "cli",
				Classification: "task",
				CapturedAt:     now,
				Promoted:       false,
			},
			{
				ID:             uuid.New(),
				UserID:         userID,
				Content:        "Second item",
				Tags:           []string{"personal"},
				Source:         "mcp",
				Classification: "note",
				CapturedAt:     now.Add(-time.Hour),
				Promoted:       false,
			},
		}

		repo.On("ListByUser", mock.Anything, userID, false).Return(items, nil)

		query := ListInboxItemsQuery{
			UserID:          userID,
			IncludePromoted: false,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "First item", result[0].Content)
		assert.Equal(t, "work", result[0].Tags[0])
		assert.Equal(t, "cli", result[0].Source)
		assert.Equal(t, "Second item", result[1].Content)

		repo.AssertExpectations(t)
	})

	t.Run("includes promoted items when requested", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewListInboxItemsHandler(repo)

		now := time.Now()
		promotedAt := now.Add(-30 * time.Minute)
		items := []domain.InboxItem{
			{
				ID:             uuid.New(),
				UserID:         userID,
				Content:        "Active item",
				CapturedAt:     now,
				Promoted:       false,
			},
			{
				ID:             uuid.New(),
				UserID:         userID,
				Content:        "Promoted item",
				CapturedAt:     now.Add(-time.Hour),
				Promoted:       true,
				PromotedTo:     "task",
				PromotedAt:     &promotedAt,
			},
		}

		repo.On("ListByUser", mock.Anything, userID, true).Return(items, nil)

		query := ListInboxItemsQuery{
			UserID:          userID,
			IncludePromoted: true,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.False(t, result[0].Promoted)
		assert.True(t, result[1].Promoted)
		assert.Equal(t, "task", result[1].PromotedTo)
		require.NotNil(t, result[1].PromotedAt)

		repo.AssertExpectations(t)
	})

	t.Run("returns empty list when no items", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewListInboxItemsHandler(repo)

		repo.On("ListByUser", mock.Anything, userID, false).Return([]domain.InboxItem{}, nil)

		query := ListInboxItemsQuery{
			UserID:          userID,
			IncludePromoted: false,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		assert.Empty(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("formats dates as RFC3339", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewListInboxItemsHandler(repo)

		capturedAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		promotedAt := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)
		items := []domain.InboxItem{
			{
				ID:         uuid.New(),
				UserID:     userID,
				Content:    "Test item",
				CapturedAt: capturedAt,
				Promoted:   true,
				PromotedAt: &promotedAt,
			},
		}

		repo.On("ListByUser", mock.Anything, userID, true).Return(items, nil)

		query := ListInboxItemsQuery{
			UserID:          userID,
			IncludePromoted: true,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "2024-01-15T10:30:00Z", result[0].CapturedAt)
		require.NotNil(t, result[0].PromotedAt)
		assert.Equal(t, "2024-01-15T11:00:00Z", *result[0].PromotedAt)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewListInboxItemsHandler(repo)

		repo.On("ListByUser", mock.Anything, userID, false).Return(nil, errors.New("database error"))

		query := ListInboxItemsQuery{
			UserID:          userID,
			IncludePromoted: false,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestNewListInboxItemsHandler(t *testing.T) {
	repo := new(mockInboxRepo)
	handler := NewListInboxItemsHandler(repo)

	require.NotNil(t, handler)
}
