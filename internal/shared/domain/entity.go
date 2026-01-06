package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entity represents a domain entity with identity.
type Entity interface {
	ID() uuid.UUID
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Equals(other Entity) bool
}

// BaseEntity provides common entity functionality.
type BaseEntity struct {
	id        uuid.UUID
	createdAt time.Time
	updatedAt time.Time
}

// NewBaseEntity creates a new entity with generated ID and current timestamps.
func NewBaseEntity() BaseEntity {
	now := time.Now().UTC()
	return BaseEntity{
		id:        uuid.New(),
		createdAt: now,
		updatedAt: now,
	}
}

// NewBaseEntityWithID creates a new entity with a specific ID.
func NewBaseEntityWithID(id uuid.UUID) BaseEntity {
	now := time.Now().UTC()
	return BaseEntity{
		id:        id,
		createdAt: now,
		updatedAt: now,
	}
}

// RehydrateBaseEntity recreates an entity from persisted state.
func RehydrateBaseEntity(id uuid.UUID, createdAt, updatedAt time.Time) BaseEntity {
	return BaseEntity{
		id:        id,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (e BaseEntity) ID() uuid.UUID        { return e.id }
func (e BaseEntity) CreatedAt() time.Time { return e.createdAt }
func (e BaseEntity) UpdatedAt() time.Time { return e.updatedAt }

// Touch updates the updatedAt timestamp.
func (e *BaseEntity) Touch() {
	e.updatedAt = time.Now().UTC()
}

// Equals checks if two entities have the same identity.
func (e BaseEntity) Equals(other Entity) bool {
	if other == nil {
		return false
	}
	return e.id == other.ID()
}
