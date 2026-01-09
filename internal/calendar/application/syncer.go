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

// CalendarEvent represents an event from an external calendar.
type CalendarEvent struct {
	ID          string
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	IsAllDay    bool
	IsRecurring bool
	Organizer   string
	Attendees   []string
	Status      string // confirmed, tentative, cancelled
	IsOrbitaEvent bool  // true if this event was created by Orbita
}

// ImportResult describes the outcome of an import run.
type ImportResult struct {
	Imported   int
	Skipped    int // Events that were already imported or created by Orbita
	Failed     int
	Conflicts  int // Events that conflict with existing schedule blocks
}

// ImportOptions configures the import behavior.
type ImportOptions struct {
	SkipOrbitaEvents bool      // Skip events created by Orbita (default: true)
	IncludeAllDay    bool      // Include all-day events
	ConflictMode     string    // "skip", "merge", "replace" (default: "skip")
	BlockType        string    // Type to assign to imported blocks (default: "meeting")
	CalendarID       string    // Calendar to import from
}

// Calendar represents an external calendar.
type Calendar struct {
	ID      string
	Name    string
	Primary bool
}

// Syncer syncs blocks into an external calendar.
type Syncer interface {
	Sync(ctx context.Context, userID uuid.UUID, blocks []TimeBlock) (*SyncResult, error)
}

// Importer imports events from an external calendar.
type Importer interface {
	// ListEvents retrieves events from the calendar within the time range.
	ListEvents(ctx context.Context, userID uuid.UUID, start, end time.Time, onlyOrbitaEvents bool) ([]CalendarEvent, error)

	// ListCalendars returns available calendars for the user.
	ListCalendars(ctx context.Context, userID uuid.UUID) ([]Calendar, error)
}

// BidirectionalSyncer supports both push and pull sync operations.
type BidirectionalSyncer interface {
	Syncer
	Importer
}
