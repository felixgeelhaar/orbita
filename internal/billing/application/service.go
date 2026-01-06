package application

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/google/uuid"
)

// Service provides billing and entitlement access.
type Service struct {
	entitlements  domain.EntitlementRepository
	subscriptions domain.SubscriptionRepository
}

// NewService creates a new billing service.
func NewService(entitlements domain.EntitlementRepository, subscriptions domain.SubscriptionRepository) *Service {
	return &Service{entitlements: entitlements, subscriptions: subscriptions}
}

// GetSubscription returns the user's subscription, if any.
func (s *Service) GetSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	if s == nil || s.subscriptions == nil {
		return nil, nil
	}
	return s.subscriptions.FindByUserID(ctx, userID)
}

// ListEntitlements returns all entitlements for the user.
func (s *Service) ListEntitlements(ctx context.Context, userID uuid.UUID) ([]domain.Entitlement, error) {
	if s == nil || s.entitlements == nil {
		return nil, nil
	}
	return s.entitlements.List(ctx, userID)
}

// SetEntitlement updates a module entitlement.
func (s *Service) SetEntitlement(ctx context.Context, userID uuid.UUID, module string, active bool, source string) error {
	if s == nil || s.entitlements == nil {
		return nil
	}
	if source == "" {
		source = "manual"
	}
	return s.entitlements.Set(ctx, userID, module, active, source)
}

// HasEntitlement reports whether the user can access the module.
func (s *Service) HasEntitlement(ctx context.Context, userID uuid.UUID, module string) (bool, error) {
	if s == nil || s.entitlements == nil {
		return true, nil
	}
	list, err := s.entitlements.List(ctx, userID)
	if err != nil {
		return false, err
	}
	if len(list) == 0 {
		return true, nil
	}
	return s.entitlements.IsActive(ctx, userID, module)
}
