package application

import (
	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

type metadataSetter interface {
	SetMetadata(metadata domain.EventMetadata)
}

// NewEventMetadata creates command-scoped metadata for domain events.
func NewEventMetadata(userID uuid.UUID) domain.EventMetadata {
	return domain.EventMetadata{
		CorrelationID: uuid.New(),
		CausationID:   uuid.New(),
		UserID:        userID,
	}
}

// ApplyEventMetadata sets metadata on all events that support it.
func ApplyEventMetadata(events []domain.DomainEvent, metadata domain.EventMetadata) {
	for _, event := range events {
		if setter, ok := event.(metadataSetter); ok {
			setter.SetMetadata(metadata)
		}
	}
}
