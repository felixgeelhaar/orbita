package application_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/licensing/application"
	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/felixgeelhaar/orbita/internal/licensing/infrastructure/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository is an in-memory repository for testing.
type mockRepository struct {
	license *domain.License
}

func newMockRepository() *mockRepository {
	return &mockRepository{}
}

func (r *mockRepository) Load(ctx context.Context) (*domain.License, error) {
	return r.license, nil
}

func (r *mockRepository) Save(ctx context.Context, license *domain.License) error {
	r.license = license
	return nil
}

func (r *mockRepository) Delete(ctx context.Context) error {
	r.license = nil
	return nil
}

func (r *mockRepository) Exists(ctx context.Context) bool {
	return r.license != nil
}

// createTestVerifier creates a verifier with a test keypair.
func createTestVerifier(t *testing.T) (*crypto.Verifier, ed25519.PrivateKey) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	return crypto.NewVerifierWithKey(publicKey), privateKey
}

// signLicense signs a license with the given private key.
func signLicense(license *domain.License, privateKey ed25519.PrivateKey) {
	entitlements := strings.Join(license.Entitlements, ",")
	expiresAt := license.ExpiresAt.Format(time.RFC3339)
	signedData := fmt.Sprintf("%s|%s|%s|%s",
		license.LicenseID.String(),
		license.Plan,
		entitlements,
		expiresAt,
	)
	signature := ed25519.Sign(privateKey, []byte(signedData))
	license.Signature = base64.StdEncoding.EncodeToString(signature)
}

func TestService_GetCurrent_NewTrial(t *testing.T) {
	repo := newMockRepository()
	verifier, _ := createTestVerifier(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service := application.NewService(repo, verifier, logger)

	ctx := context.Background()
	license, err := service.GetCurrent(ctx)

	require.NoError(t, err)
	require.NotNil(t, license)
	assert.False(t, license.IsActivated())
	assert.False(t, license.TrialStartedAt.IsZero())
	assert.WithinDuration(t, time.Now(), license.TrialStartedAt, time.Second)
}

func TestService_GetCurrent_ExistingLicense(t *testing.T) {
	repo := newMockRepository()
	existingLicense := &domain.License{
		Version:        1,
		LicenseKey:     "ORB-TEST-XXXX-YYYY",
		TrialStartedAt: time.Now().Add(-7 * 24 * time.Hour),
	}
	repo.license = existingLicense

	verifier, _ := createTestVerifier(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	service := application.NewService(repo, verifier, logger)

	ctx := context.Background()
	license, err := service.GetCurrent(ctx)

	require.NoError(t, err)
	require.NotNil(t, license)
	assert.Equal(t, "ORB-TEST-XXXX-YYYY", license.LicenseKey)
}

func TestService_GetStatus_Trial(t *testing.T) {
	repo := newMockRepository()
	verifier, _ := createTestVerifier(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

	tests := []struct {
		name     string
		license  *domain.License
		expected domain.LicenseStatus
	}{
		{
			name:     "nil license",
			license:  nil,
			expected: domain.LicenseStatusTrial,
		},
		{
			name: "new trial",
			license: &domain.License{
				TrialStartedAt: time.Now(),
			},
			expected: domain.LicenseStatusTrial,
		},
		{
			name: "mid trial",
			license: &domain.License{
				TrialStartedAt: time.Now().Add(-7 * 24 * time.Hour),
			},
			expected: domain.LicenseStatusTrial,
		},
		{
			name: "trial expired",
			license: &domain.License{
				TrialStartedAt: time.Now().Add(-15 * 24 * time.Hour),
			},
			expected: domain.LicenseStatusFreeTier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := service.GetStatus(tt.license)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestService_GetStatus_Active(t *testing.T) {
	verifier, privateKey := createTestVerifier(t)
	repo := newMockRepository()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

	license := &domain.License{
		Version:        1,
		LicenseKey:     "ORB-TEST-XXXX-YYYY",
		LicenseID:      uuid.New(),
		Plan:           "pro",
		Entitlements:   []string{"smart-habits"},
		ExpiresAt:      time.Now().Add(30 * 24 * time.Hour),
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
	}
	signLicense(license, privateKey)

	status := service.GetStatus(license)
	assert.Equal(t, domain.LicenseStatusActive, status)
}

func TestService_GetStatus_GracePeriod(t *testing.T) {
	verifier, privateKey := createTestVerifier(t)
	repo := newMockRepository()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

	license := &domain.License{
		Version:        1,
		LicenseKey:     "ORB-TEST-XXXX-YYYY",
		LicenseID:      uuid.New(),
		Plan:           "pro",
		Entitlements:   []string{"smart-habits"},
		ExpiresAt:      time.Now().Add(-3 * 24 * time.Hour), // Expired 3 days ago
		TrialStartedAt: time.Now().Add(-400 * 24 * time.Hour),
	}
	signLicense(license, privateKey)

	status := service.GetStatus(license)
	assert.Equal(t, domain.LicenseStatusGracePeriod, status)
}

func TestService_GetStatus_Expired(t *testing.T) {
	verifier, privateKey := createTestVerifier(t)
	repo := newMockRepository()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

	license := &domain.License{
		Version:        1,
		LicenseKey:     "ORB-TEST-XXXX-YYYY",
		LicenseID:      uuid.New(),
		Plan:           "pro",
		Entitlements:   []string{"smart-habits"},
		ExpiresAt:      time.Now().Add(-10 * 24 * time.Hour), // Expired 10 days ago (past grace period)
		TrialStartedAt: time.Now().Add(-400 * 24 * time.Hour),
	}
	signLicense(license, privateKey)

	status := service.GetStatus(license)
	assert.Equal(t, domain.LicenseStatusExpired, status)
}

func TestService_GetStatus_Invalid(t *testing.T) {
	verifier, _ := createTestVerifier(t)
	repo := newMockRepository()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

	// License with invalid signature
	license := &domain.License{
		Version:        1,
		LicenseKey:     "ORB-TEST-XXXX-YYYY",
		LicenseID:      uuid.New(),
		Plan:           "pro",
		Entitlements:   []string{"smart-habits"},
		ExpiresAt:      time.Now().Add(30 * 24 * time.Hour),
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
		Signature:      "invalid-signature",
	}

	status := service.GetStatus(license)
	assert.Equal(t, domain.LicenseStatusInvalid, status)
}

func TestService_Activate(t *testing.T) {
	repo := newMockRepository()
	verifier, privateKey := createTestVerifier(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

	// Set up existing trial
	repo.license = &domain.License{
		TrialStartedAt: time.Now().Add(-5 * 24 * time.Hour),
	}

	// Activate new license
	newLicense := &domain.License{
		Version:      1,
		LicenseKey:   "ORB-NEW1-KEY2-HERE",
		LicenseID:    uuid.New(),
		Plan:         "pro",
		Entitlements: []string{"smart-habits", "ai-inbox"},
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
	}
	signLicense(newLicense, privateKey)

	ctx := context.Background()
	err := service.Activate(ctx, newLicense)

	require.NoError(t, err)
	assert.NotNil(t, repo.license)
	assert.Equal(t, "ORB-NEW1-KEY2-HERE", repo.license.LicenseKey)
	// Should preserve trial start time
	assert.WithinDuration(t, time.Now().Add(-5*24*time.Hour), repo.license.TrialStartedAt, time.Hour)
}

func TestService_Deactivate(t *testing.T) {
	repo := newMockRepository()
	verifier, privateKey := createTestVerifier(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

	// Set up active license
	activeLicense := &domain.License{
		Version:        1,
		LicenseKey:     "ORB-ACTV-LICE-ENSE",
		LicenseID:      uuid.New(),
		Plan:           "pro",
		Entitlements:   []string{"smart-habits"},
		ExpiresAt:      time.Now().Add(300 * 24 * time.Hour),
		TrialStartedAt: time.Now().Add(-20 * 24 * time.Hour),
	}
	signLicense(activeLicense, privateKey)
	repo.license = activeLicense

	ctx := context.Background()
	err := service.Deactivate(ctx)

	require.NoError(t, err)
	assert.NotNil(t, repo.license)
	assert.Empty(t, repo.license.LicenseKey)
	// Should preserve trial start time
	assert.WithinDuration(t, time.Now().Add(-20*24*time.Hour), repo.license.TrialStartedAt, time.Hour)
}

func TestService_NeedsValidation(t *testing.T) {
	verifier, _ := createTestVerifier(t)
	repo := newMockRepository()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

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
			name: "not activated",
			license: &domain.License{
				TrialStartedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "recently validated",
			license: &domain.License{
				LicenseKey:      "ORB-TEST-XXXX-YYYY",
				LastValidatedAt: time.Now().Add(-1 * 24 * time.Hour),
			},
			expected: false,
		},
		{
			name: "needs validation (old)",
			license: &domain.License{
				LicenseKey:      "ORB-TEST-XXXX-YYYY",
				LastValidatedAt: time.Now().Add(-31 * 24 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needs := service.NeedsValidation(tt.license)
			assert.Equal(t, tt.expected, needs)
		})
	}
}

func TestService_CacheBehavior(t *testing.T) {
	repo := newMockRepository()
	verifier, _ := createTestVerifier(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := application.NewService(repo, verifier, logger)

	ctx := context.Background()

	// First call creates new trial
	license1, err := service.GetCurrent(ctx)
	require.NoError(t, err)

	// Second call should return cached version
	license2, err := service.GetCurrent(ctx)
	require.NoError(t, err)
	assert.Same(t, license1, license2)

	// Clear cache and get again
	service.ClearCache()
	license3, err := service.GetCurrent(ctx)
	require.NoError(t, err)
	// Should load from repo (same data but different pointer)
	assert.Equal(t, license1.TrialStartedAt, license3.TrialStartedAt)
}
