package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/felixgeelhaar/orbita/internal/inbox/services"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/google/uuid"
)

// CaptureInboxItemCommand contains capture data.
type CaptureInboxItemCommand struct {
	UserID   uuid.UUID
	Content  string
	Metadata domain.InboxMetadata
	Tags     []string
	Source   string
}

// CaptureInboxItemResult returns the saved ID.
type CaptureInboxItemResult struct {
	ItemID uuid.UUID
}

// CaptureInboxItemHandler persists inbox items.
type CaptureInboxItemHandler struct {
	repo        domain.InboxRepository
	classifier  *services.Classifier
	uow         sharedApplication.UnitOfWork
}

// NewCaptureInboxItemHandler builds a handler.
func NewCaptureInboxItemHandler(repo domain.InboxRepository, classifier *services.Classifier, uow sharedApplication.UnitOfWork) *CaptureInboxItemHandler {
	return &CaptureInboxItemHandler{repo: repo, classifier: classifier, uow: uow}
}

// Handle saves the inbox item.
func (h *CaptureInboxItemHandler) Handle(ctx context.Context, cmd CaptureInboxItemCommand) (*CaptureInboxItemResult, error) {
	var result *CaptureInboxItemResult
	err := sharedApplication.WithUnitOfWork(ctx, h.uow, func(txCtx context.Context) error {
		now := time.Now().UTC()
		itemID := uuid.New()
		classification := h.classifier.Classify(cmd.Content, cmd.Metadata)

		item := domain.InboxItem{
			ID:             itemID,
			UserID:         cmd.UserID,
			Content:        cmd.Content,
			Metadata:       cmd.Metadata,
			Tags:           cmd.Tags,
			Source:         cmd.Source,
			Classification: classification,
			CapturedAt:     now,
		}

		if err := h.repo.Save(txCtx, item); err != nil {
			return err
		}
		result = &CaptureInboxItemResult{ItemID: itemID}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
