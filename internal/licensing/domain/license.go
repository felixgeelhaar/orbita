package domain

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LicenseStatus represents the current state of a license.
type LicenseStatus string

const (
	// LicenseStatusTrial indicates the user is in a free trial period.
	LicenseStatusTrial LicenseStatus = "trial"
	// LicenseStatusActive indicates the license is valid and active.
	LicenseStatusActive LicenseStatus = "active"
	// LicenseStatusGracePeriod indicates the license expired but is in grace period.
	LicenseStatusGracePeriod LicenseStatus = "grace_period"
	// LicenseStatusExpired indicates the license has fully expired.
	LicenseStatusExpired LicenseStatus = "expired"
	// LicenseStatusFreeTier indicates trial ended with no license activated.
	LicenseStatusFreeTier LicenseStatus = "free_tier"
	// LicenseStatusInvalid indicates the license signature is invalid.
	LicenseStatusInvalid LicenseStatus = "invalid"
)

// TrialDuration is the length of the free trial period.
const TrialDuration = 14 * 24 * time.Hour

// GracePeriodDuration is the length of the grace period after license expiry.
const GracePeriodDuration = 7 * 24 * time.Hour

// License represents a software license for Orbita Pro features.
type License struct {
	Version         int       `json:"version"`
	LicenseKey      string    `json:"license_key,omitempty"`
	LicenseID       uuid.UUID `json:"license_id,omitempty"`
	Email           string    `json:"email,omitempty"`
	Plan            string    `json:"plan,omitempty"`
	Entitlements    []string  `json:"entitlements,omitempty"`
	IssuedAt        time.Time `json:"issued_at,omitempty"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	Signature       string    `json:"signature,omitempty"`
	LastValidatedAt time.Time `json:"last_validated_at,omitempty"`
	TrialStartedAt  time.Time `json:"trial_started_at,omitempty"`
}

// NewTrialLicense creates a new license file with trial started.
func NewTrialLicense() *License {
	return &License{
		Version:        1,
		TrialStartedAt: time.Now(),
	}
}

// NewLicense creates a new activated license.
func NewLicense(
	licenseKey string,
	licenseID uuid.UUID,
	email string,
	plan string,
	entitlements []string,
	issuedAt time.Time,
	expiresAt time.Time,
	signature string,
) *License {
	return &License{
		Version:         1,
		LicenseKey:      licenseKey,
		LicenseID:       licenseID,
		Email:           email,
		Plan:            plan,
		Entitlements:    entitlements,
		IssuedAt:        issuedAt,
		ExpiresAt:       expiresAt,
		Signature:       signature,
		LastValidatedAt: time.Now(),
	}
}

// IsActivated returns true if a license key has been activated.
func (l *License) IsActivated() bool {
	return l != nil && l.LicenseKey != ""
}

// HasEntitlement checks if the license includes a specific module entitlement.
func (l *License) HasEntitlement(module string) bool {
	if l == nil {
		return false
	}
	for _, ent := range l.Entitlements {
		if ent == module {
			return true
		}
	}
	return false
}

// SignatureBytes returns the decoded signature bytes.
func (l *License) SignatureBytes() ([]byte, error) {
	if l == nil || l.Signature == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(l.Signature)
}

// TrialDaysRemaining returns the number of days remaining in the trial period.
// Returns 0 if not in trial or trial has expired.
func (l *License) TrialDaysRemaining() int {
	if l == nil || l.TrialStartedAt.IsZero() {
		return 0
	}
	trialEnd := l.TrialStartedAt.Add(TrialDuration)
	remaining := time.Until(trialEnd)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Hours()/24) + 1 // Round up
}

// GraceDaysRemaining returns the number of days remaining in the grace period.
// Returns 0 if not in grace period.
func (l *License) GraceDaysRemaining() int {
	if l == nil || l.ExpiresAt.IsZero() {
		return 0
	}
	gracePeriodEnd := l.ExpiresAt.Add(GracePeriodDuration)
	remaining := time.Until(gracePeriodEnd)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Hours()/24) + 1 // Round up
}

// MaskedKey returns the license key with middle sections masked for display.
// e.g., "ORB-****-****-IJKL"
func (l *License) MaskedKey() string {
	if l == nil || l.LicenseKey == "" {
		return ""
	}
	key := l.LicenseKey
	if len(key) <= 4 {
		return key
	}
	// For short keys, show first 4 chars and mask the rest
	if len(key) < 12 {
		return key[:4] + strings.Repeat("*", len(key)-4)
	}
	// For standard keys (ORB-XXXX-XXXX-XXXX), show prefix, mask middle, show last segment
	// Preserve the dash structure
	return key[:4] + "****-****-" + key[len(key)-4:]
}
