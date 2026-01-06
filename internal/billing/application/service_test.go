package application

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeEntitlementRepo struct {
	list   []domain.Entitlement
	active map[string]bool
}

func (f fakeEntitlementRepo) Set(ctx context.Context, userID uuid.UUID, module string, active bool, source string) error {
	return nil
}

func (f fakeEntitlementRepo) List(ctx context.Context, userID uuid.UUID) ([]domain.Entitlement, error) {
	return f.list, nil
}

func (f fakeEntitlementRepo) IsActive(ctx context.Context, userID uuid.UUID, module string) (bool, error) {
	if f.active == nil {
		return false, nil
	}
	return f.active[module], nil
}

type fakeSubscriptionRepo struct{}

func (f fakeSubscriptionRepo) Upsert(ctx context.Context, subscription *domain.Subscription) error {
	return nil
}

func (f fakeSubscriptionRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return nil, nil
}

func TestHasEntitlement_DefaultAllowWhenEmpty(t *testing.T) {
	svc := NewService(fakeEntitlementRepo{list: nil}, fakeSubscriptionRepo{})
	allowed, err := svc.HasEntitlement(context.Background(), uuid.New(), domain.ModuleAdaptiveFrequency)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestHasEntitlement_RespectsExplicitEntries(t *testing.T) {
	svc := NewService(fakeEntitlementRepo{
		list: []domain.Entitlement{{UserID: uuid.New(), Module: domain.ModuleSmartMeetings, Active: true, Source: "manual"}},
		active: map[string]bool{
			domain.ModuleSmartMeetings:     true,
			domain.ModuleAdaptiveFrequency: false,
		},
	}, fakeSubscriptionRepo{})

	allowed, err := svc.HasEntitlement(context.Background(), uuid.New(), domain.ModuleAdaptiveFrequency)
	require.NoError(t, err)
	require.False(t, allowed)
}
