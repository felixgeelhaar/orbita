package queries

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/google/uuid"
)

// ListInboxItemsQuery holds params.
type ListInboxItemsQuery struct {
	UserID         uuid.UUID
	IncludePromoted bool
}

// InboxItemDTO is view model.
type InboxItemDTO struct {
	ID             uuid.UUID
	Content        string
	Tags           []string
	Source         string
	Classification string
	CapturedAt     string
	Promoted       bool
	PromotedTo     string
	PromotedAt     *string
}

// ListInboxItemsHandler returns items.
type ListInboxItemsHandler struct {
	repo domain.InboxRepository
}

// NewListInboxItemsHandler creates handler.
func NewListInboxItemsHandler(repo domain.InboxRepository) *ListInboxItemsHandler {
	return &ListInboxItemsHandler{repo: repo}
}

// Handle executes the query.
func (h *ListInboxItemsHandler) Handle(ctx context.Context, query ListInboxItemsQuery) ([]InboxItemDTO, error) {
	items, err := h.repo.ListByUser(ctx, query.UserID, query.IncludePromoted)
	if err != nil {
		return nil, err
	}

	dtos := make([]InboxItemDTO, len(items))
	for i, item := range items {
		var promotedAt *string
		if item.PromotedAt != nil {
			val := item.PromotedAt.Format(time.RFC3339)
			promotedAt = &val
		}
		dtos[i] = InboxItemDTO{
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
	}
	return dtos, nil
}
