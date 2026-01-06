package domain

import (
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// User represents a user account in the system.
type User struct {
	sharedDomain.BaseAggregateRoot
	email Email
	name  Name
}

// NewUser creates a new user with the given email and name.
func NewUser(email Email, name Name) *User {
	u := &User{
		BaseAggregateRoot: sharedDomain.NewBaseAggregateRoot(),
		email:             email,
		name:              name,
	}

	u.AddDomainEvent(NewUserCreated(u.ID(), email.String(), name.String()))

	return u
}

// NewUserWithID creates a user with a specific ID (for rehydration).
func NewUserWithID(id uuid.UUID, email Email, name Name) *User {
	return &User{
		BaseAggregateRoot: sharedDomain.NewBaseAggregateRootWithID(id),
		email:             email,
		name:              name,
	}
}

// Getters
func (u *User) Email() Email { return u.email }
func (u *User) Name() Name   { return u.name }

// UpdateName changes the user's name.
func (u *User) UpdateName(name Name) {
	if u.name.Equals(name) {
		return
	}

	u.name = name
	u.Touch()

	u.AddDomainEvent(NewUserUpdated(u.ID(), name.String()))
}
