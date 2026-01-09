package api

import (
	"context"
	"time"

	inboxQueries "github.com/felixgeelhaar/orbita/internal/inbox/application/queries"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
)

// InboxAPIImpl implements sdk.InboxAPI with capability checking.
type InboxAPIImpl struct {
	listHandler  *inboxQueries.ListInboxItemsHandler
	getHandler   *inboxQueries.GetInboxItemHandler
	userID       uuid.UUID
	capabilities sdk.CapabilitySet
}

// NewInboxAPI creates a new InboxAPI implementation.
func NewInboxAPI(
	listHandler *inboxQueries.ListInboxItemsHandler,
	getHandler *inboxQueries.GetInboxItemHandler,
	userID uuid.UUID,
	caps sdk.CapabilitySet,
) *InboxAPIImpl {
	return &InboxAPIImpl{
		listHandler:  listHandler,
		getHandler:   getHandler,
		userID:       userID,
		capabilities: caps,
	}
}

func (a *InboxAPIImpl) checkCapability() error {
	if !a.capabilities.Has(sdk.CapReadInbox) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

// List returns all inbox items.
func (a *InboxAPIImpl) List(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	items, err := a.listHandler.Handle(ctx, inboxQueries.ListInboxItemsQuery{
		UserID: a.userID,
	})
	if err != nil {
		return nil, err
	}

	return toInboxSDKDTOs(items), nil
}

// Get returns a single inbox item by ID.
func (a *InboxAPIImpl) Get(ctx context.Context, id string) (*sdk.InboxItemDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	itemID, err := uuid.Parse(id)
	if err != nil {
		return nil, sdk.ErrResourceNotFound
	}

	item, err := a.getHandler.Handle(ctx, inboxQueries.GetInboxItemQuery{
		ItemID: itemID,
		UserID: a.userID,
	})
	if err != nil {
		if err == inboxQueries.ErrInboxItemNotFound {
			return nil, sdk.ErrResourceNotFound
		}
		return nil, err
	}

	dto := toInboxSDKDTO(*item)
	return &dto, nil
}

// GetPending returns items that haven't been promoted.
func (a *InboxAPIImpl) GetPending(ctx context.Context) ([]sdk.InboxItemDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	items, err := a.listHandler.Handle(ctx, inboxQueries.ListInboxItemsQuery{
		UserID: a.userID,
	})
	if err != nil {
		return nil, err
	}

	// Filter to pending items
	var pending []inboxQueries.InboxItemDTO
	for _, item := range items {
		if !item.Promoted {
			pending = append(pending, item)
		}
	}

	return toInboxSDKDTOs(pending), nil
}

// GetByClassification returns items with a specific classification.
func (a *InboxAPIImpl) GetByClassification(ctx context.Context, classification string) ([]sdk.InboxItemDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	items, err := a.listHandler.Handle(ctx, inboxQueries.ListInboxItemsQuery{
		UserID: a.userID,
	})
	if err != nil {
		return nil, err
	}

	// Filter by classification
	var filtered []inboxQueries.InboxItemDTO
	for _, item := range items {
		if item.Classification == classification {
			filtered = append(filtered, item)
		}
	}

	return toInboxSDKDTOs(filtered), nil
}

func toInboxSDKDTOs(items []inboxQueries.InboxItemDTO) []sdk.InboxItemDTO {
	result := make([]sdk.InboxItemDTO, len(items))
	for i, item := range items {
		result[i] = toInboxSDKDTO(item)
	}
	return result
}

func toInboxSDKDTO(item inboxQueries.InboxItemDTO) sdk.InboxItemDTO {
	// Parse CapturedAt string to time.Time
	capturedAt, _ := time.Parse(time.RFC3339, item.CapturedAt)
	return sdk.InboxItemDTO{
		ID:             item.ID.String(),
		Content:        item.Content,
		Source:         item.Source,
		Classification: item.Classification,
		Promoted:       item.Promoted,
		CreatedAt:      capturedAt,
	}
}
