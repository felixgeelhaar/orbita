package domain

import (
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

const (
	AggregateType = "User"

	RoutingKeyUserCreated = "identity.user.created"
	RoutingKeyUserUpdated = "identity.user.updated"
)

// UserCreated is emitted when a new user is created.
type UserCreated struct {
	sharedDomain.BaseEvent
	Email string `json:"email"`
	Name  string `json:"name"`
}

// NewUserCreated creates a UserCreated event.
func NewUserCreated(userID uuid.UUID, email, name string) UserCreated {
	return UserCreated{
		BaseEvent: sharedDomain.NewBaseEvent(userID, AggregateType, RoutingKeyUserCreated),
		Email:     email,
		Name:      name,
	}
}

// UserUpdated is emitted when a user profile is updated.
type UserUpdated struct {
	sharedDomain.BaseEvent
	Name string `json:"name"`
}

// NewUserUpdated creates a UserUpdated event.
func NewUserUpdated(userID uuid.UUID, name string) UserUpdated {
	return UserUpdated{
		BaseEvent: sharedDomain.NewBaseEvent(userID, AggregateType, RoutingKeyUserUpdated),
		Name:      name,
	}
}
