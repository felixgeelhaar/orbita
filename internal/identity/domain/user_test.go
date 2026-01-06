package domain_test

import (
	"testing"

	"github.com/felixgeelhaar/orbita/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestUser(t *testing.T) *domain.User {
	email, err := domain.NewEmail("test@example.com")
	require.NoError(t, err)
	name, err := domain.NewName("Test User")
	require.NoError(t, err)
	return domain.NewUser(email, name)
}

func TestNewUser(t *testing.T) {
	email, _ := domain.NewEmail("user@example.com")
	name, _ := domain.NewName("John Doe")

	user := domain.NewUser(email, name)

	assert.NotEqual(t, uuid.Nil, user.ID())
	assert.Equal(t, "user@example.com", user.Email().String())
	assert.Equal(t, "John Doe", user.Name().String())
}

func TestNewUser_EmitsCreatedEvent(t *testing.T) {
	email, _ := domain.NewEmail("user@example.com")
	name, _ := domain.NewName("John Doe")

	user := domain.NewUser(email, name)

	events := user.DomainEvents()
	require.Len(t, events, 1)

	createdEvent, ok := events[0].(domain.UserCreated)
	require.True(t, ok)
	assert.Equal(t, user.ID(), createdEvent.AggregateID())
	assert.Equal(t, domain.RoutingKeyUserCreated, createdEvent.RoutingKey())
	assert.Equal(t, "user@example.com", createdEvent.Email)
	assert.Equal(t, "John Doe", createdEvent.Name)
}

func TestNewUserWithID(t *testing.T) {
	id := uuid.New()
	email, _ := domain.NewEmail("user@example.com")
	name, _ := domain.NewName("John Doe")

	user := domain.NewUserWithID(id, email, name)

	assert.Equal(t, id, user.ID())
	assert.Empty(t, user.DomainEvents()) // No events for rehydration
}

func TestUser_UpdateName(t *testing.T) {
	user := createTestUser(t)
	user.ClearDomainEvents()

	newName, _ := domain.NewName("Updated Name")
	user.UpdateName(newName)

	assert.Equal(t, "Updated Name", user.Name().String())

	events := user.DomainEvents()
	require.Len(t, events, 1)

	updatedEvent, ok := events[0].(domain.UserUpdated)
	require.True(t, ok)
	assert.Equal(t, "Updated Name", updatedEvent.Name)
}

func TestUser_UpdateName_SameName(t *testing.T) {
	user := createTestUser(t)
	user.ClearDomainEvents()

	sameName, _ := domain.NewName("Test User")
	user.UpdateName(sameName)

	// No event emitted if name is the same
	assert.Empty(t, user.DomainEvents())
}
