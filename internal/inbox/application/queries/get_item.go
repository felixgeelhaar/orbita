package queries

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/google/uuid"
)

// ErrInboxItemNotFound is returned when an inbox item is not found.
var ErrInboxItemNotFound = errors.New("inbox item not found")

// GetInboxItemQuery contains the parameters for getting a single inbox item.
type GetInboxItemQuery struct {
	ItemID uuid.UUID
	UserID uuid.UUID // For authorization check
}

// GetInboxItemHandler handles the GetInboxItemQuery.
type GetInboxItemHandler struct {
	repo domain.InboxRepository
}

// NewGetInboxItemHandler creates a new GetInboxItemHandler.
func NewGetInboxItemHandler(repo domain.InboxRepository) *GetInboxItemHandler {
	return &GetInboxItemHandler{repo: repo}
}

// Handle executes the GetInboxItemQuery.
func (h *GetInboxItemHandler) Handle(ctx context.Context, query GetInboxItemQuery) (*InboxItemDTO, error) {
	item, err := h.repo.FindByID(ctx, query.UserID, query.ItemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrInboxItemNotFound
	}

	var promotedAt *string
	if item.PromotedAt != nil {
		val := item.PromotedAt.Format(time.RFC3339)
		promotedAt = &val
	}

	dto := InboxItemDTO{
		ID:             item.ID,
		Content:        item.Content,
		Tags:           item.Tags,
		Source:         item.Source,
		Classification: item.Classification,
		CapturedAt:     item.CapturedAt.Format(time.RFC3339),
		Promoted:       item.Promoted,
		PromotedTo:     item.PromotedTo,
		PromotedAt:     promotedAt,
	}

	return &dto, nil
}
