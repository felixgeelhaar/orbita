package application_test

import (
	"context"
	"crypto/ed25519"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/felixgeelhaar/orbita/internal/licensing/application"
	licensingDomain "github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/felixgeelhaar/orbita/internal/licensing/infrastructure/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createLocalBillingService(t *testing.T, license *licensingDomain.License) (*application.LocalBillingService, *mockRepository, ed25519.PrivateKey) {
	repo := newMockRepository()
	repo.license = license

	verifier, privateKey := createTestVerifier(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	licenseService := application.NewService(repo, verifier, logger)

	return application.NewLocalBillingService(licenseService), repo, privateKey
}

func TestLocalBillingService_HasEntitlement_Trial(t *testing.T) {
	// Trial license - all features enabled
	trialLicense := &licensingDomain.License{
		Version:        1,
		TrialStartedAt: time.Now(), // Active trial
	}

	service, _, _ := createLocalBillingService(t, trialLicense)
	ctx := context.Background()
	userID := uuid.New()

	// All modules should be available during trial
	modules := []string{
		domain.ModuleSmartHabits,
		domain.ModuleSmartMeetings,
		domain.ModuleAutoRescheduler,
		domain.ModuleAIInbox,
		domain.ModulePriorityEngine,
		domain.ModuleAdaptiveFrequency,
	}

	for _, module := range modules {
		t.Run(module, func(t *testing.T) {
			has, err := service.HasEntitlement(ctx, userID, module)
			require.NoError(t, err)
			assert.True(t, has, "trial should have access to %s", module)
		})
	}
}

func TestLocalBillingService_HasEntitlement_Active(t *testing.T) {
	// Create active license with specific entitlements
	activeLicense := &licensingDomain.License{
		Version:        1,
		LicenseKey:     "ORB-TEST-XXXX-YYYY",
		LicenseID:      uuid.New(),
		Plan:           "pro",
		Entitlements:   []string{domain.ModuleSmartHabits, domain.ModuleAIInbox},
		ExpiresAt:      time.Now().Add(30 * 24 * time.Hour),
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
	}

	service, _, privateKey := createLocalBillingService(t, activeLicense)
	signLicense(activeLicense, privateKey)

	ctx := context.Background()
	userID := uuid.New()

	// Should have entitled modules
	has, err := service.HasEntitlement(ctx, userID, domain.ModuleSmartHabits)
	require.NoError(t, err)
	assert.True(t, has)

	has, err = service.HasEntitlement(ctx, userID, domain.ModuleAIInbox)
	require.NoError(t, err)
	assert.True(t, has)

	// Should not have non-entitled modules
	has, err = service.HasEntitlement(ctx, userID, domain.ModuleSmartMeetings)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestLocalBillingService_HasEntitlement_FreeTier(t *testing.T) {
	// Expired trial = free tier
	freeTierLicense := &licensingDomain.License{
		Version:        1,
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour), // Trial expired
	}

	service, _, _ := createLocalBillingService(t, freeTierLicense)
	ctx := context.Background()
	userID := uuid.New()

	// No modules should be available on free tier
	modules := []string{
		domain.ModuleSmartHabits,
		domain.ModuleSmartMeetings,
		domain.ModuleAutoRescheduler,
		domain.ModuleAIInbox,
		domain.ModulePriorityEngine,
		domain.ModuleAdaptiveFrequency,
	}

	for _, module := range modules {
		t.Run(module, func(t *testing.T) {
			has, err := service.HasEntitlement(ctx, userID, module)
			require.NoError(t, err)
			assert.False(t, has, "free tier should not have access to %s", module)
		})
	}
}

func TestLocalBillingService_GetSubscription_Trial(t *testing.T) {
	trialLicense := &licensingDomain.License{
		Version:        1,
		TrialStartedAt: time.Now().Add(-3 * 24 * time.Hour),
	}

	service, _, _ := createLocalBillingService(t, trialLicense)
	ctx := context.Background()
	userID := uuid.New()

	sub, err := service.GetSubscription(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, "trial", sub.Plan)
	assert.Equal(t, domain.SubscriptionTrialing, sub.Status)
	assert.NotNil(t, sub.CurrentPeriodEnd)
}

func TestLocalBillingService_GetSubscription_Active(t *testing.T) {
	activeLicense := &licensingDomain.License{
		Version:        1,
		LicenseKey:     "ORB-TEST-XXXX-YYYY",
		LicenseID:      uuid.New(),
		Plan:           "pro",
		Entitlements:   []string{domain.ModuleSmartHabits},
		ExpiresAt:      time.Now().Add(30 * 24 * time.Hour),
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
	}

	service, _, privateKey := createLocalBillingService(t, activeLicense)
	signLicense(activeLicense, privateKey)

	ctx := context.Background()
	userID := uuid.New()

	sub, err := service.GetSubscription(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, "pro", sub.Plan)
	assert.Equal(t, domain.SubscriptionActive, sub.Status)
	assert.NotNil(t, sub.CurrentPeriodEnd)
}

func TestLocalBillingService_GetSubscription_FreeTier(t *testing.T) {
	freeTierLicense := &licensingDomain.License{
		Version:        1,
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
	}

	service, _, _ := createLocalBillingService(t, freeTierLicense)
	ctx := context.Background()
	userID := uuid.New()

	sub, err := service.GetSubscription(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, "free", sub.Plan)
	assert.Equal(t, domain.SubscriptionActive, sub.Status)
}

func TestLocalBillingService_ListEntitlements_Trial(t *testing.T) {
	trialLicense := &licensingDomain.License{
		Version:        1,
		TrialStartedAt: time.Now(),
	}

	service, _, _ := createLocalBillingService(t, trialLicense)
	ctx := context.Background()
	userID := uuid.New()

	entitlements, err := service.ListEntitlements(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, entitlements, 6) // All Pro modules
	for _, ent := range entitlements {
		assert.True(t, ent.Active)
		assert.Equal(t, "trial", ent.Source)
	}
}

func TestLocalBillingService_ListEntitlements_Active(t *testing.T) {
	activeLicense := &licensingDomain.License{
		Version:        1,
		LicenseKey:     "ORB-TEST-XXXX-YYYY",
		LicenseID:      uuid.New(),
		Plan:           "pro",
		Entitlements:   []string{domain.ModuleSmartHabits, domain.ModuleAIInbox},
		ExpiresAt:      time.Now().Add(30 * 24 * time.Hour),
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
	}

	service, _, privateKey := createLocalBillingService(t, activeLicense)
	signLicense(activeLicense, privateKey)

	ctx := context.Background()
	userID := uuid.New()

	entitlements, err := service.ListEntitlements(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, entitlements, 2)
	for _, ent := range entitlements {
		assert.True(t, ent.Active)
		assert.Equal(t, "license", ent.Source)
	}
}

func TestLocalBillingService_ListEntitlements_FreeTier(t *testing.T) {
	freeTierLicense := &licensingDomain.License{
		Version:        1,
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
	}

	service, _, _ := createLocalBillingService(t, freeTierLicense)
	ctx := context.Background()
	userID := uuid.New()

	entitlements, err := service.ListEntitlements(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, entitlements)
}

func TestLocalBillingService_SetEntitlement_NoOp(t *testing.T) {
	trialLicense := &licensingDomain.License{
		Version:        1,
		TrialStartedAt: time.Now(),
	}

	service, _, _ := createLocalBillingService(t, trialLicense)
	ctx := context.Background()
	userID := uuid.New()

	// SetEntitlement is a no-op in local mode
	err := service.SetEntitlement(ctx, userID, domain.ModuleSmartHabits, false, "test")
	require.NoError(t, err)
}

func TestLocalBillingService_GetLicenseStatus(t *testing.T) {
	tests := []struct {
		name           string
		license        *licensingDomain.License
		expectedStatus licensingDomain.LicenseStatus
	}{
		{
			name: "trial",
			license: &licensingDomain.License{
				Version:        1,
				TrialStartedAt: time.Now(),
			},
			expectedStatus: licensingDomain.LicenseStatusTrial,
		},
		{
			name: "free tier",
			license: &licensingDomain.License{
				Version:        1,
				TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
			},
			expectedStatus: licensingDomain.LicenseStatusFreeTier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _, _ := createLocalBillingService(t, tt.license)
			ctx := context.Background()

			status, err := service.GetLicenseStatus(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestLocalBillingService_VerifiesInterface(t *testing.T) {
	trialLicense := &licensingDomain.License{
		Version:        1,
		TrialStartedAt: time.Now(),
	}

	service, _, _ := createLocalBillingService(t, trialLicense)

	// Verify that LocalBillingService implements domain.BillingService
	var _ domain.BillingService = service
}

// createTestVerifier creates a verifier for testing.
func createTestVerifier2(t *testing.T) (*crypto.Verifier, ed25519.PrivateKey) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	return crypto.NewVerifierWithKey(publicKey), privateKey
}
