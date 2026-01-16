package domain

import (
	"time"

	"github.com/google/uuid"
)

// ConflictType represents the type of scheduling conflict.
type ConflictType string

const (
	// ConflictTypeOverlap indicates an external event overlaps an Orbita block.
	ConflictTypeOverlap ConflictType = "overlap"
	// ConflictTypeModified indicates an Orbita event was modified externally.
	ConflictTypeModified ConflictType = "modified"
	// ConflictTypeDeleted indicates an Orbita event was deleted externally.
	ConflictTypeDeleted ConflictType = "deleted"
	// ConflictTypeDoubleBooked indicates multiple blocks scheduled for same time.
	ConflictTypeDoubleBooked ConflictType = "double_booked"
)

// ConflictResolutionStrategy defines how conflicts should be resolved.
type ConflictResolutionStrategy string

const (
	// StrategyOrbitaWins means Orbita blocks always take priority.
	// External conflicts trigger reschedule of external items.
	StrategyOrbitaWins ConflictResolutionStrategy = "orbita_wins"

	// StrategyExternalWins means external events take priority.
	// Orbita blocks are rescheduled automatically.
	StrategyExternalWins ConflictResolutionStrategy = "external_wins"

	// StrategyManual means conflicts are marked for user review.
	// No auto-resolution is performed.
	StrategyManual ConflictResolutionStrategy = "manual"

	// StrategyTimeFirst means earlier scheduled item wins.
	// This is the default strategy.
	StrategyTimeFirst ConflictResolutionStrategy = "time_first"
)

// ConflictResolution describes how a conflict was resolved.
type ConflictResolution string

const (
	// ResolutionRescheduled means the conflicting item was rescheduled.
	ResolutionRescheduled ConflictResolution = "rescheduled"
	// ResolutionKept means both items were kept despite conflict.
	ResolutionKept ConflictResolution = "kept"
	// ResolutionRemoved means one item was removed.
	ResolutionRemoved ConflictResolution = "removed"
	// ResolutionPending means the conflict needs user review.
	ResolutionPending ConflictResolution = "pending"
)

// Conflict represents a scheduling conflict between events.
type Conflict struct {
	id              uuid.UUID
	userID          uuid.UUID
	conflictType    ConflictType
	orbitaBlockID   uuid.UUID
	orbitaBlockTime TimeRange
	externalEventID string
	externalTime    TimeRange
	resolution      ConflictResolution
	resolvedAt      *time.Time
	createdAt       time.Time
}

// TimeRange represents a time period with start and end.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// Overlaps checks if two time ranges overlap.
func (t TimeRange) Overlaps(other TimeRange) bool {
	return t.Start.Before(other.End) && other.Start.Before(t.End)
}

// Duration returns the duration of the time range.
func (t TimeRange) Duration() time.Duration {
	return t.End.Sub(t.Start)
}

// NewConflict creates a new conflict.
func NewConflict(
	userID uuid.UUID,
	conflictType ConflictType,
	orbitaBlockID uuid.UUID,
	orbitaBlockTime TimeRange,
	externalEventID string,
	externalTime TimeRange,
) *Conflict {
	return &Conflict{
		id:              uuid.New(),
		userID:          userID,
		conflictType:    conflictType,
		orbitaBlockID:   orbitaBlockID,
		orbitaBlockTime: orbitaBlockTime,
		externalEventID: externalEventID,
		externalTime:    externalTime,
		resolution:      ResolutionPending,
		createdAt:       time.Now(),
	}
}

// ID returns the conflict's unique identifier.
func (c *Conflict) ID() uuid.UUID {
	return c.id
}

// UserID returns the user ID associated with this conflict.
func (c *Conflict) UserID() uuid.UUID {
	return c.userID
}

// ConflictType returns the type of conflict.
func (c *Conflict) ConflictType() ConflictType {
	return c.conflictType
}

// OrbitaBlockID returns the Orbita block involved in the conflict.
func (c *Conflict) OrbitaBlockID() uuid.UUID {
	return c.orbitaBlockID
}

// OrbitaBlockTime returns the time range of the Orbita block.
func (c *Conflict) OrbitaBlockTime() TimeRange {
	return c.orbitaBlockTime
}

// ExternalEventID returns the external event ID involved in the conflict.
func (c *Conflict) ExternalEventID() string {
	return c.externalEventID
}

// ExternalTime returns the time range of the external event.
func (c *Conflict) ExternalTime() TimeRange {
	return c.externalTime
}

// Resolution returns the current resolution status.
func (c *Conflict) Resolution() ConflictResolution {
	return c.resolution
}

// ResolvedAt returns when the conflict was resolved.
func (c *Conflict) ResolvedAt() *time.Time {
	return c.resolvedAt
}

// CreatedAt returns when the conflict was created.
func (c *Conflict) CreatedAt() time.Time {
	return c.createdAt
}

// IsPending returns true if the conflict is not yet resolved.
func (c *Conflict) IsPending() bool {
	return c.resolution == ResolutionPending
}

// Resolve marks the conflict as resolved with the given resolution.
func (c *Conflict) Resolve(resolution ConflictResolution) {
	c.resolution = resolution
	now := time.Now()
	c.resolvedAt = &now
}

// MarkRescheduled marks the conflict as resolved by rescheduling.
func (c *Conflict) MarkRescheduled() {
	c.Resolve(ResolutionRescheduled)
}

// MarkKept marks the conflict as kept (both items remain).
func (c *Conflict) MarkKept() {
	c.Resolve(ResolutionKept)
}

// MarkRemoved marks the conflict as resolved by removing one item.
func (c *Conflict) MarkRemoved() {
	c.Resolve(ResolutionRemoved)
}

// ConflictRepository defines persistence operations for conflicts.
type ConflictRepository interface {
	// Save persists a conflict.
	Save(c *Conflict) error

	// FindByID retrieves a conflict by its ID.
	FindByID(id uuid.UUID) (*Conflict, error)

	// FindByUser retrieves all conflicts for a user.
	FindByUser(userID uuid.UUID) ([]*Conflict, error)

	// FindPending retrieves all pending conflicts for a user.
	FindPending(userID uuid.UUID) ([]*Conflict, error)

	// Delete removes a conflict.
	Delete(id uuid.UUID) error
}

// DetectOverlap checks if two time ranges have a conflict and returns the conflict type.
func DetectOverlap(block TimeRange, event TimeRange) (bool, ConflictType) {
	if block.Overlaps(event) {
		return true, ConflictTypeOverlap
	}
	return false, ""
}
