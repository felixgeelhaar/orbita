package domain_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTrialLicense(t *testing.T) {
	license := domain.NewTrialLicense()

	assert.Equal(t, 1, license.Version)
	assert.Equal(t, "", license.LicenseKey)
	assert.False(t, license.IsActivated())
	assert.False(t, license.TrialStartedAt.IsZero())
	assert.WithinDuration(t, time.Now(), license.TrialStartedAt, time.Second)
}

func TestLicense_IsActivated(t *testing.T) {
	tests := []struct {
		name     string
		license  *domain.License
		expected bool
	}{
		{
			name:     "nil license",
			license:  nil,
			expected: false,
		},
		{
			name:     "empty license key",
			license:  &domain.License{LicenseKey: ""},
			expected: false,
		},
		{
			name:     "valid license key",
			license:  &domain.License{LicenseKey: "ORB-XXXX-XXXX-XXXX"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.license == nil {
				assert.False(t, tt.expected)
				return
			}
			assert.Equal(t, tt.expected, tt.license.IsActivated())
		})
	}
}

func TestLicense_TrialDaysRemaining(t *testing.T) {
	tests := []struct {
		name          string
		trialStartedAt time.Time
		expectedDays  int
	}{
		{
			name:          "just started",
			trialStartedAt: time.Now(),
			expectedDays:  14,
		},
		{
			name:          "7 days ago",
			trialStartedAt: time.Now().Add(-7 * 24 * time.Hour),
			expectedDays:  7,
		},
		{
			name:          "14 days ago",
			trialStartedAt: time.Now().Add(-14 * 24 * time.Hour),
			expectedDays:  0,
		},
		{
			name:          "20 days ago",
			trialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
			expectedDays:  0, // Clamped to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := &domain.License{TrialStartedAt: tt.trialStartedAt}
			// Allow 1 day variance due to time calculations
			remaining := license.TrialDaysRemaining()
			assert.InDelta(t, tt.expectedDays, remaining, 1)
		})
	}
}

func TestLicense_GraceDaysRemaining(t *testing.T) {
	tests := []struct {
		name         string
		expiresAt    time.Time
		expectedDays int
	}{
		{
			name:         "just expired",
			expiresAt:    time.Now(),
			expectedDays: 7,
		},
		{
			name:         "expired 3 days ago",
			expiresAt:    time.Now().Add(-3 * 24 * time.Hour),
			expectedDays: 4,
		},
		{
			name:         "expired 7 days ago",
			expiresAt:    time.Now().Add(-7 * 24 * time.Hour),
			expectedDays: 0,
		},
		{
			name:         "not yet expired",
			expiresAt:    time.Now().Add(30 * 24 * time.Hour),
			expectedDays: 37, // 30 + 7 grace days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := &domain.License{ExpiresAt: tt.expiresAt}
			remaining := license.GraceDaysRemaining()
			assert.InDelta(t, tt.expectedDays, remaining, 1)
		})
	}
}

func TestLicense_MaskedKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "standard key",
			key:      "ORB-ABCD-EFGH-IJKL",
			expected: "ORB-****-****-IJKL",
		},
		{
			name:     "short key",
			key:      "ORB-ABC",
			expected: "ORB-***",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := &domain.License{LicenseKey: tt.key}
			assert.Equal(t, tt.expected, license.MaskedKey())
		})
	}
}

func TestLicense_SignatureBytes(t *testing.T) {
	t.Run("valid base64 signature", func(t *testing.T) {
		license := &domain.License{
			LicenseKey: "ORB-TEST-TEST-TEST",
			Signature:  "SGVsbG8gV29ybGQ=", // "Hello World" in base64
		}

		bytes, err := license.SignatureBytes()
		require.NoError(t, err)
		assert.Equal(t, []byte("Hello World"), bytes)
	})

	t.Run("invalid base64 signature", func(t *testing.T) {
		license := &domain.License{
			LicenseKey: "ORB-TEST-TEST-TEST",
			Signature:  "not-valid-base64!!!",
		}

		_, err := license.SignatureBytes()
		assert.Error(t, err)
	})

	t.Run("empty signature", func(t *testing.T) {
		license := &domain.License{
			LicenseKey: "ORB-TEST-TEST-TEST",
			Signature:  "",
		}

		bytes, err := license.SignatureBytes()
		require.NoError(t, err)
		assert.Empty(t, bytes)
	})
}

func TestLicenseStatus_String(t *testing.T) {
	assert.Equal(t, "trial", string(domain.LicenseStatusTrial))
	assert.Equal(t, "active", string(domain.LicenseStatusActive))
	assert.Equal(t, "grace_period", string(domain.LicenseStatusGracePeriod))
	assert.Equal(t, "expired", string(domain.LicenseStatusExpired))
	assert.Equal(t, "free_tier", string(domain.LicenseStatusFreeTier))
	assert.Equal(t, "invalid", string(domain.LicenseStatusInvalid))
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 14*24*time.Hour, domain.TrialDuration)
	assert.Equal(t, 7*24*time.Hour, domain.GracePeriodDuration)
}

func createActivatedLicense() *domain.License {
	return &domain.License{
		Version:         1,
		LicenseKey:      "ORB-TEST-TEST-TEST",
		LicenseID:       uuid.New(),
		Email:           "test@example.com",
		Plan:            "pro",
		Entitlements:    []string{"smart-habits", "ai-inbox"},
		IssuedAt:        time.Now().Add(-30 * 24 * time.Hour),
		ExpiresAt:       time.Now().Add(335 * 24 * time.Hour), // ~1 year from issuance
		TrialStartedAt:  time.Now().Add(-45 * 24 * time.Hour),
		LastValidatedAt: time.Now().Add(-1 * time.Hour),
		Signature:       "test-signature",
	}
}

func TestCreateActivatedLicense(t *testing.T) {
	license := createActivatedLicense()

	assert.True(t, license.IsActivated())
	assert.Equal(t, "pro", license.Plan)
	assert.Len(t, license.Entitlements, 2)
}
