package domain

import (
	"context"

	"github.com/google/uuid"
)

// BillingService defines the interface for billing and entitlement operations.
// This interface is implemented by both the database-backed service (server mode)
// and the license-based service (local mode).
type BillingService interface {
	// GetSubscription returns the user's subscription, if any.
	GetSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error)

	// ListEntitlements returns all entitlements for the user.
	ListEntitlements(ctx context.Context, userID uuid.UUID) ([]Entitlement, error)

	// SetEntitlement updates a module entitlement.
	SetEntitlement(ctx context.Context, userID uuid.UUID, module string, active bool, source string) error

	// HasEntitlement reports whether the user can access the module.
	HasEntitlement(ctx context.Context, userID uuid.UUID, module string) (bool, error)
}
