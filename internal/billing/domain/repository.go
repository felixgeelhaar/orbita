package domain

import (
	"context"

	"github.com/google/uuid"
)

// EntitlementRepository defines access for entitlement persistence.
type EntitlementRepository interface {
	Set(ctx context.Context, userID uuid.UUID, module string, active bool, source string) error
	List(ctx context.Context, userID uuid.UUID) ([]Entitlement, error)
	IsActive(ctx context.Context, userID uuid.UUID, module string) (bool, error)
}

// SubscriptionRepository defines access for subscription persistence.
type SubscriptionRepository interface {
	Upsert(ctx context.Context, subscription *Subscription) error
	FindByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error)
}
