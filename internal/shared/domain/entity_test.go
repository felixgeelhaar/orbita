package domain_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseEntity(t *testing.T) {
	before := time.Now().UTC()
	entity := domain.NewBaseEntity()
	after := time.Now().UTC()

	assert.NotEqual(t, uuid.Nil, entity.ID())
	require.False(t, entity.CreatedAt().Before(before))
	require.False(t, entity.CreatedAt().After(after))
	assert.Equal(t, entity.CreatedAt(), entity.UpdatedAt())
}

func TestNewBaseEntityWithID(t *testing.T) {
	id := uuid.New()
	entity := domain.NewBaseEntityWithID(id)

	assert.Equal(t, id, entity.ID())
}

func TestBaseEntity_Touch(t *testing.T) {
	entity := domain.NewBaseEntity()
	originalUpdatedAt := entity.UpdatedAt()

	time.Sleep(time.Millisecond)
	entity.Touch()

	assert.True(t, entity.UpdatedAt().After(originalUpdatedAt))
	assert.Equal(t, entity.CreatedAt(), entity.CreatedAt()) // CreatedAt unchanged
}

func TestBaseEntity_Equals(t *testing.T) {
	id := uuid.New()
	entity1 := domain.NewBaseEntityWithID(id)
	entity2 := domain.NewBaseEntityWithID(id)
	entity3 := domain.NewBaseEntity()

	assert.True(t, entity1.Equals(&entity2))
	assert.False(t, entity1.Equals(&entity3))
}
