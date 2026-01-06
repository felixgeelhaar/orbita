package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InboxRepository handles persistence for inbox items.
type InboxRepository interface {
	Save(ctx context.Context, item InboxItem) error
	ListByUser(ctx context.Context, userID uuid.UUID, includePromoted bool) ([]InboxItem, error)
	FindByID(ctx context.Context, userID, id uuid.UUID) (*InboxItem, error)
	MarkPromoted(ctx context.Context, id uuid.UUID, promotedTo string, promotedID uuid.UUID, promotedAt time.Time) error
}
