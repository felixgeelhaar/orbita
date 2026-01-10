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

func TestGetInboxItemHandler_Handle(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	t.Run("successfully returns item", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewGetInboxItemHandler(repo)

		now := time.Now()
		item := &domain.InboxItem{
			ID:             itemID,
			UserID:         userID,
			Content:        "Test item content",
			Tags:           []string{"important", "work"},
			Source:         "cli",
			Classification: "task",
			CapturedAt:     now,
			Promoted:       false,
		}

		repo.On("FindByID", mock.Anything, userID, itemID).Return(item, nil)

		query := GetInboxItemQuery{
			ItemID: itemID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, itemID, result.ID)
		assert.Equal(t, "Test item content", result.Content)
		assert.Contains(t, result.Tags, "important")
		assert.Contains(t, result.Tags, "work")
		assert.Equal(t, "cli", result.Source)
		assert.Equal(t, "task", result.Classification)
		assert.False(t, result.Promoted)
		assert.Nil(t, result.PromotedAt)

		repo.AssertExpectations(t)
	})

	t.Run("returns promoted item with promotion details", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewGetInboxItemHandler(repo)

		now := time.Now()
		promotedAt := now.Add(-time.Hour)
		promotedID := uuid.New()
		item := &domain.InboxItem{
			ID:             itemID,
			UserID:         userID,
			Content:        "Promoted item",
			CapturedAt:     now.Add(-2 * time.Hour),
			Promoted:       true,
			PromotedTo:     "task",
			PromotedID:     promotedID,
			PromotedAt:     &promotedAt,
		}

		repo.On("FindByID", mock.Anything, userID, itemID).Return(item, nil)

		query := GetInboxItemQuery{
			ItemID: itemID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Promoted)
		assert.Equal(t, "task", result.PromotedTo)
		require.NotNil(t, result.PromotedAt)

		repo.AssertExpectations(t)
	})

	t.Run("formats dates as RFC3339", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewGetInboxItemHandler(repo)

		capturedAt := time.Date(2024, 6, 20, 14, 30, 0, 0, time.UTC)
		promotedAt := time.Date(2024, 6, 20, 15, 45, 0, 0, time.UTC)
		item := &domain.InboxItem{
			ID:         itemID,
			UserID:     userID,
			Content:    "Test item",
			CapturedAt: capturedAt,
			Promoted:   true,
			PromotedAt: &promotedAt,
		}

		repo.On("FindByID", mock.Anything, userID, itemID).Return(item, nil)

		query := GetInboxItemQuery{
			ItemID: itemID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		require.NoError(t, err)
		assert.Equal(t, "2024-06-20T14:30:00Z", result.CapturedAt)
		require.NotNil(t, result.PromotedAt)
		assert.Equal(t, "2024-06-20T15:45:00Z", *result.PromotedAt)

		repo.AssertExpectations(t)
	})

	t.Run("returns ErrInboxItemNotFound when item is nil", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewGetInboxItemHandler(repo)

		repo.On("FindByID", mock.Anything, userID, itemID).Return(nil, nil)

		query := GetInboxItemQuery{
			ItemID: itemID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.ErrorIs(t, err, ErrInboxItemNotFound)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})

	t.Run("fails when repository error", func(t *testing.T) {
		repo := new(mockInboxRepo)
		handler := NewGetInboxItemHandler(repo)

		repo.On("FindByID", mock.Anything, userID, itemID).Return(nil, errors.New("database error"))

		query := GetInboxItemQuery{
			ItemID: itemID,
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)

		repo.AssertExpectations(t)
	})
}

func TestNewGetInboxItemHandler(t *testing.T) {
	repo := new(mockInboxRepo)
	handler := NewGetInboxItemHandler(repo)

	require.NotNil(t, handler)
}
