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

func TestHasEntitlement_NilService(t *testing.T) {
	var svc *Service
	allowed, err := svc.HasEntitlement(context.Background(), uuid.New(), domain.ModuleSmartMeetings)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestGetSubscription_Success(t *testing.T) {
	userID := uuid.New()
	sub := &domain.Subscription{
		UserID: userID,
		Status: "active",
	}
	repo := &fakeSubscriptionRepoWithSub{sub: sub}
	svc := NewService(fakeEntitlementRepo{}, repo)

	result, err := svc.GetSubscription(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, sub, result)
}

func TestGetSubscription_NilService(t *testing.T) {
	var svc *Service
	result, err := svc.GetSubscription(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestGetSubscription_NilRepo(t *testing.T) {
	svc := &Service{}
	result, err := svc.GetSubscription(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestListEntitlements_Success(t *testing.T) {
	userID := uuid.New()
	entitlements := []domain.Entitlement{
		{UserID: userID, Module: domain.ModuleSmartMeetings, Active: true, Source: "stripe"},
		{UserID: userID, Module: domain.ModuleAdaptiveFrequency, Active: false, Source: "manual"},
	}
	svc := NewService(fakeEntitlementRepo{list: entitlements}, fakeSubscriptionRepo{})

	result, err := svc.ListEntitlements(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, entitlements, result)
}

func TestListEntitlements_NilService(t *testing.T) {
	var svc *Service
	result, err := svc.ListEntitlements(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestListEntitlements_NilRepo(t *testing.T) {
	svc := &Service{}
	result, err := svc.ListEntitlements(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestSetEntitlement_Success(t *testing.T) {
	svc := NewService(fakeEntitlementRepo{}, fakeSubscriptionRepo{})
	err := svc.SetEntitlement(context.Background(), uuid.New(), domain.ModuleSmartMeetings, true, "stripe")
	require.NoError(t, err)
}

func TestSetEntitlement_DefaultSource(t *testing.T) {
	svc := NewService(fakeEntitlementRepo{}, fakeSubscriptionRepo{})
	err := svc.SetEntitlement(context.Background(), uuid.New(), domain.ModuleSmartMeetings, true, "")
	require.NoError(t, err)
}

func TestSetEntitlement_NilService(t *testing.T) {
	var svc *Service
	err := svc.SetEntitlement(context.Background(), uuid.New(), domain.ModuleSmartMeetings, true, "manual")
	require.NoError(t, err)
}

func TestSetEntitlement_NilRepo(t *testing.T) {
	svc := &Service{}
	err := svc.SetEntitlement(context.Background(), uuid.New(), domain.ModuleSmartMeetings, true, "manual")
	require.NoError(t, err)
}

// fakeSubscriptionRepoWithSub is a fake subscription repo that returns a subscription
type fakeSubscriptionRepoWithSub struct {
	sub *domain.Subscription
}

func (f *fakeSubscriptionRepoWithSub) Upsert(ctx context.Context, subscription *domain.Subscription) error {
	return nil
}

func (f *fakeSubscriptionRepoWithSub) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return f.sub, nil
}
