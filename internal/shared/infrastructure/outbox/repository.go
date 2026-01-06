package outbox

import (
	"context"
	"time"
)

// Repository defines the interface for outbox persistence.
type Repository interface {
	// Save stores a new outbox message.
	Save(ctx context.Context, msg *Message) error

	// SaveBatch stores multiple outbox messages atomically.
	SaveBatch(ctx context.Context, msgs []*Message) error

	// GetUnpublished retrieves unpublished messages ordered by creation time.
	GetUnpublished(ctx context.Context, limit int) ([]*Message, error)

	// MarkPublished marks a message as successfully published.
	MarkPublished(ctx context.Context, id int64) error

	// MarkFailed records a publish failure with error message.
	MarkFailed(ctx context.Context, id int64, err string, nextRetryAt time.Time) error

	// MarkDead marks a message as dead-lettered.
	MarkDead(ctx context.Context, id int64, reason string) error

	// GetFailed retrieves failed messages eligible for retry.
	GetFailed(ctx context.Context, maxRetries, limit int) ([]*Message, error)

	// DeleteOld removes successfully published messages older than the retention period.
	DeleteOld(ctx context.Context, olderThanDays int) (int64, error)
}
