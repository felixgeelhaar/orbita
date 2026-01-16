package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/felixgeelhaar/orbita/internal/licensing/infrastructure/crypto"
)

// Service handles license management operations.
type Service struct {
	repo     domain.Repository
	verifier *crypto.Verifier
	logger   *slog.Logger
	cache    *domain.License // In-memory cache
}

// NewService creates a new license service.
func NewService(repo domain.Repository, verifier *crypto.Verifier, logger *slog.Logger) *Service {
	return &Service{
		repo:     repo,
		verifier: verifier,
		logger:   logger,
	}
}

// GetCurrent retrieves the current license from storage.
// If no license exists, creates and returns a trial license.
func (s *Service) GetCurrent(ctx context.Context) (*domain.License, error) {
	// Use cached license if available
	if s.cache != nil {
		return s.cache, nil
	}

	license, err := s.repo.Load(ctx)
	if err != nil {
		return nil, err
	}

	// No license file exists - start trial
	if license == nil {
		license = domain.NewTrialLicense()
		if err := s.repo.Save(ctx, license); err != nil {
			s.logger.Warn("failed to save trial license", "error", err)
			// Continue with in-memory trial
		}
	}

	s.cache = license
	return license, nil
}

// GetStatus determines the current status of a license.
func (s *Service) GetStatus(license *domain.License) domain.LicenseStatus {
	if license == nil {
		return domain.LicenseStatusTrial // First run
	}

	// If license is activated, verify and check expiry
	if license.IsActivated() {
		return s.checkActivatedLicenseStatus(license)
	}

	// Not activated - check trial status
	return s.checkTrialStatus(license)
}

// checkActivatedLicenseStatus checks the status of an activated license.
func (s *Service) checkActivatedLicenseStatus(license *domain.License) domain.LicenseStatus {
	// Verify signature
	if !s.verifier.Verify(license) {
		return domain.LicenseStatusInvalid
	}

	now := time.Now()

	// Check if license is still valid
	if now.Before(license.ExpiresAt) {
		return domain.LicenseStatusActive
	}

	// Check grace period
	gracePeriodEnd := license.ExpiresAt.Add(domain.GracePeriodDuration)
	if now.Before(gracePeriodEnd) {
		return domain.LicenseStatusGracePeriod
	}

	return domain.LicenseStatusExpired
}

// checkTrialStatus checks the status of a trial period.
func (s *Service) checkTrialStatus(license *domain.License) domain.LicenseStatus {
	// No trial start time means first run
	if license.TrialStartedAt.IsZero() {
		return domain.LicenseStatusTrial
	}

	trialEnd := license.TrialStartedAt.Add(domain.TrialDuration)
	if time.Now().Before(trialEnd) {
		return domain.LicenseStatusTrial
	}

	return domain.LicenseStatusFreeTier
}

// Activate activates a license key.
func (s *Service) Activate(ctx context.Context, license *domain.License) error {
	// Preserve trial start time if it exists
	existing, _ := s.repo.Load(ctx)
	if existing != nil && !existing.TrialStartedAt.IsZero() {
		license.TrialStartedAt = existing.TrialStartedAt
	}

	if err := s.repo.Save(ctx, license); err != nil {
		return err
	}

	s.cache = license
	s.logger.Info("license activated", "plan", license.Plan, "expires", license.ExpiresAt)
	return nil
}

// Deactivate removes the current license (reverts to trial/free tier).
func (s *Service) Deactivate(ctx context.Context) error {
	existing, _ := s.repo.Load(ctx)

	// Create a new license that only preserves trial start
	newLicense := domain.NewTrialLicense()
	if existing != nil && !existing.TrialStartedAt.IsZero() {
		newLicense.TrialStartedAt = existing.TrialStartedAt
	}

	if err := s.repo.Save(ctx, newLicense); err != nil {
		return err
	}

	s.cache = newLicense
	s.logger.Info("license deactivated")
	return nil
}

// UpdateLastValidated updates the last validation timestamp.
func (s *Service) UpdateLastValidated(ctx context.Context) error {
	license, err := s.GetCurrent(ctx)
	if err != nil {
		return err
	}

	license.LastValidatedAt = time.Now()
	if err := s.repo.Save(ctx, license); err != nil {
		return err
	}

	s.cache = license
	return nil
}

// NeedsValidation checks if the license should be re-validated with the server.
func (s *Service) NeedsValidation(license *domain.License) bool {
	if license == nil || !license.IsActivated() {
		return false
	}

	// Default validation interval: 30 days
	validationInterval := 30 * 24 * time.Hour

	return time.Since(license.LastValidatedAt) > validationInterval
}

// ClearCache clears the in-memory license cache.
func (s *Service) ClearCache() {
	s.cache = nil
}
