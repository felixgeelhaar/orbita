package application

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TimeBlock is a simplified block for calendar sync.
type TimeBlock struct {
	ID        uuid.UUID
	Title     string
	BlockType string
	StartTime time.Time
	EndTime   time.Time
	Completed bool
	Missed    bool
}

// SyncResult describes the outcome of a sync run.
type SyncResult struct {
	Created int
	Updated int
	Failed  int
	Deleted int
}

// Syncer syncs blocks into an external calendar.
type Syncer interface {
	Sync(ctx context.Context, userID uuid.UUID, blocks []TimeBlock) (*SyncResult, error)
}
