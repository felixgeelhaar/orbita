package domain

import "github.com/google/uuid"

// AggregateRoot is a domain entity that is the root of an aggregate.
type AggregateRoot interface {
	Entity
	DomainEvents() []DomainEvent
	ClearDomainEvents()
	AddDomainEvent(event DomainEvent)
	Version() int
}

// BaseAggregateRoot provides common aggregate functionality.
type BaseAggregateRoot struct {
	BaseEntity
	domainEvents []DomainEvent
	version      int
}

// NewBaseAggregateRoot creates a new aggregate root.
func NewBaseAggregateRoot() BaseAggregateRoot {
	return BaseAggregateRoot{
		BaseEntity:   NewBaseEntity(),
		domainEvents: make([]DomainEvent, 0),
		version:      0,
	}
}

// NewBaseAggregateRootWithID creates a new aggregate root with a specific ID.
func NewBaseAggregateRootWithID(id uuid.UUID) BaseAggregateRoot {
	return BaseAggregateRoot{
		BaseEntity:   NewBaseEntityWithID(id),
		domainEvents: make([]DomainEvent, 0),
		version:      0,
	}
}

// DomainEvents returns all uncommitted domain events.
func (a *BaseAggregateRoot) DomainEvents() []DomainEvent {
	return a.domainEvents
}

// ClearDomainEvents removes all uncommitted domain events.
func (a *BaseAggregateRoot) ClearDomainEvents() {
	a.domainEvents = make([]DomainEvent, 0)
}

// AddDomainEvent adds a domain event to the aggregate.
func (a *BaseAggregateRoot) AddDomainEvent(event DomainEvent) {
	a.domainEvents = append(a.domainEvents, event)
}

// Version returns the aggregate version for optimistic concurrency.
func (a *BaseAggregateRoot) Version() int {
	return a.version
}

// IncrementVersion increments the aggregate version.
func (a *BaseAggregateRoot) IncrementVersion() {
	a.version++
}

// SetVersion sets the aggregate version (used when rehydrating from storage).
func (a *BaseAggregateRoot) SetVersion(version int) {
	a.version = version
}

// RehydrateBaseAggregateRoot recreates an aggregate from persisted state.
func RehydrateBaseAggregateRoot(entity BaseEntity, version int) BaseAggregateRoot {
	return BaseAggregateRoot{
		BaseEntity:   entity,
		domainEvents: make([]DomainEvent, 0),
		version:      version,
	}
}
