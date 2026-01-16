package application

import (
	"context"
	"slices"

	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	licensingDomain "github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/google/uuid"
)

// LocalBillingService implements billingDomain.BillingService using local license files.
// This is used in local mode instead of database-backed entitlements.
type LocalBillingService struct {
	licenseService *Service
}

// NewLocalBillingService creates a new local billing service.
func NewLocalBillingService(licenseService *Service) *LocalBillingService {
	return &LocalBillingService{
		licenseService: licenseService,
	}
}

// GetSubscription returns subscription info based on the license.
// In local mode, subscription info is derived from the license.
func (s *LocalBillingService) GetSubscription(ctx context.Context, userID uuid.UUID) (*billingDomain.Subscription, error) {
	license, err := s.licenseService.GetCurrent(ctx)
	if err != nil {
		return nil, err
	}

	status := s.licenseService.GetStatus(license)

	// Map license status to subscription
	switch status {
	case licensingDomain.LicenseStatusActive, licensingDomain.LicenseStatusGracePeriod:
		return &billingDomain.Subscription{
			UserID:           userID,
			Plan:             license.Plan,
			Status:           billingDomain.SubscriptionActive,
			CurrentPeriodEnd: &license.ExpiresAt,
		}, nil

	case licensingDomain.LicenseStatusTrial:
		trialEnd := license.TrialStartedAt.Add(licensingDomain.TrialDuration)
		return &billingDomain.Subscription{
			UserID:           userID,
			Plan:             "trial",
			Status:           billingDomain.SubscriptionTrialing,
			CurrentPeriodEnd: &trialEnd,
		}, nil

	default:
		return &billingDomain.Subscription{
			UserID: userID,
			Plan:   "free",
			Status: billingDomain.SubscriptionActive,
		}, nil
	}
}

// ListEntitlements returns all entitlements based on the license.
func (s *LocalBillingService) ListEntitlements(ctx context.Context, userID uuid.UUID) ([]billingDomain.Entitlement, error) {
	license, err := s.licenseService.GetCurrent(ctx)
	if err != nil {
		return nil, err
	}

	status := s.licenseService.GetStatus(license)

	// In trial or active license, return all entitlements
	switch status {
	case licensingDomain.LicenseStatusTrial:
		// Trial gets all Pro features
		return allProEntitlements(userID, "trial"), nil

	case licensingDomain.LicenseStatusActive, licensingDomain.LicenseStatusGracePeriod:
		// Return license entitlements
		entitlements := make([]billingDomain.Entitlement, 0, len(license.Entitlements))
		for _, module := range license.Entitlements {
			entitlements = append(entitlements, billingDomain.Entitlement{
				UserID: userID,
				Module: module,
				Active: true,
				Source: "license",
			})
		}
		return entitlements, nil

	default:
		// Free tier - no entitlements
		return nil, nil
	}
}

// SetEntitlement is not supported in local mode (licenses are immutable).
func (s *LocalBillingService) SetEntitlement(ctx context.Context, userID uuid.UUID, module string, active bool, source string) error {
	// In local mode, entitlements come from the license and cannot be modified
	return nil
}

// HasEntitlement reports whether the user can access the module based on their license.
func (s *LocalBillingService) HasEntitlement(ctx context.Context, userID uuid.UUID, module string) (bool, error) {
	license, err := s.licenseService.GetCurrent(ctx)
	if err != nil {
		return false, err
	}

	status := s.licenseService.GetStatus(license)

	switch status {
	case licensingDomain.LicenseStatusTrial:
		// Trial gets all Pro features
		return true, nil

	case licensingDomain.LicenseStatusActive, licensingDomain.LicenseStatusGracePeriod:
		// Check if module is in license entitlements
		return slices.Contains(license.Entitlements, module), nil

	case licensingDomain.LicenseStatusFreeTier, licensingDomain.LicenseStatusExpired:
		// Free tier - no Pro features
		return false, nil

	case licensingDomain.LicenseStatusInvalid:
		// Invalid license - no Pro features
		return false, nil

	default:
		return false, nil
	}
}

// GetLicenseStatus returns the current license status for display.
func (s *LocalBillingService) GetLicenseStatus(ctx context.Context) (licensingDomain.LicenseStatus, error) {
	license, err := s.licenseService.GetCurrent(ctx)
	if err != nil {
		return "", err
	}
	return s.licenseService.GetStatus(license), nil
}

// GetLicense returns the current license for display.
func (s *LocalBillingService) GetLicense(ctx context.Context) (*licensingDomain.License, error) {
	return s.licenseService.GetCurrent(ctx)
}

// allProEntitlements returns all Pro module entitlements.
func allProEntitlements(userID uuid.UUID, source string) []billingDomain.Entitlement {
	modules := []string{
		billingDomain.ModuleSmartHabits,
		billingDomain.ModuleSmartMeetings,
		billingDomain.ModuleAutoRescheduler,
		billingDomain.ModuleAIInbox,
		billingDomain.ModulePriorityEngine,
		billingDomain.ModuleAdaptiveFrequency,
	}

	entitlements := make([]billingDomain.Entitlement, 0, len(modules))
	for _, module := range modules {
		entitlements = append(entitlements, billingDomain.Entitlement{
			UserID: userID,
			Module: module,
			Active: true,
			Source: source,
		})
	}
	return entitlements
}
