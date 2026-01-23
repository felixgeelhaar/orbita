package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVerifier(t *testing.T) {
	// This uses the embedded public key
	verifier, err := NewVerifier()
	require.NoError(t, err)
	assert.NotNil(t, verifier)
}

func TestNewVerifierWithKey(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	verifier := NewVerifierWithKey(pub)
	assert.NotNil(t, verifier)
}

func TestVerifier_Verify_NilLicense(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	verifier := NewVerifierWithKey(pub)
	assert.False(t, verifier.Verify(nil))
}

func TestVerifier_Verify_NotActivated(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	verifier := NewVerifierWithKey(pub)
	license := &domain.License{} // No license key = not activated

	assert.False(t, verifier.Verify(license))
}

func TestVerifier_Verify_InvalidSignature(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	verifier := NewVerifierWithKey(pub)
	license := &domain.License{
		LicenseKey:   "ORB-TEST-1234-5678",
		LicenseID:    uuid.New(),
		Plan:         "pro",
		Entitlements: []string{"smart-habits"},
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
		Signature:    base64.StdEncoding.EncodeToString([]byte("invalid-signature")),
	}

	assert.False(t, verifier.Verify(license))
}

func TestVerifier_Verify_ValidSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	licenseID := uuid.New()
	plan := "pro"
	entitlements := []string{"smart-habits", "ai-inbox"}
	expiresAt := time.Now().Add(365 * 24 * time.Hour)

	// Build signed data
	signedData := fmt.Sprintf("%s|%s|%s|%s",
		licenseID.String(),
		plan,
		"smart-habits,ai-inbox",
		expiresAt.Format(time.RFC3339),
	)

	signature := ed25519.Sign(priv, []byte(signedData))
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	license := &domain.License{
		LicenseKey:   "ORB-TEST-1234-5678",
		LicenseID:    licenseID,
		Plan:         plan,
		Entitlements: entitlements,
		ExpiresAt:    expiresAt,
		Signature:    signatureB64,
	}

	verifier := NewVerifierWithKey(pub)
	assert.True(t, verifier.Verify(license))
}

func TestVerifier_Verify_TamperedData(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	licenseID := uuid.New()
	plan := "pro"
	originalEntitlements := "smart-habits"
	expiresAt := time.Now().Add(365 * 24 * time.Hour)

	// Build signed data with original entitlements
	signedData := fmt.Sprintf("%s|%s|%s|%s",
		licenseID.String(),
		plan,
		originalEntitlements,
		expiresAt.Format(time.RFC3339),
	)

	signature := ed25519.Sign(priv, []byte(signedData))
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	// Create license with different entitlements (tampered)
	license := &domain.License{
		LicenseKey:   "ORB-TEST-1234-5678",
		LicenseID:    licenseID,
		Plan:         plan,
		Entitlements: []string{"smart-habits", "ai-inbox"}, // Added extra entitlement
		ExpiresAt:    expiresAt,
		Signature:    signatureB64,
	}

	verifier := NewVerifierWithKey(pub)
	assert.False(t, verifier.Verify(license)) // Should fail due to tampered entitlements
}

func TestVerifier_Verify_MalformedSignature(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	verifier := NewVerifierWithKey(pub)
	license := &domain.License{
		LicenseKey:   "ORB-TEST-1234-5678",
		LicenseID:    uuid.New(),
		Plan:         "pro",
		Entitlements: []string{"smart-habits"},
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
		Signature:    "not-valid-base64!!!",
	}

	assert.False(t, verifier.Verify(license))
}

func TestVerifier_BuildSignedData(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	verifier := NewVerifierWithKey(pub)

	licenseID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	expiresAt := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	license := &domain.License{
		LicenseKey:   "ORB-TEST-1234-5678",
		LicenseID:    licenseID,
		Plan:         "pro",
		Entitlements: []string{"smart-habits", "ai-inbox"},
		ExpiresAt:    expiresAt,
	}

	expected := "550e8400-e29b-41d4-a716-446655440000|pro|smart-habits,ai-inbox|2025-12-31T23:59:59Z"
	result := verifier.buildSignedData(license)

	assert.Equal(t, expected, result)
}

func TestParsePublicKey_InvalidPEM(t *testing.T) {
	_, err := parsePublicKey([]byte("not a PEM block"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode PEM block")
}

func TestParsePublicKey_InvalidKeySize(t *testing.T) {
	// Create a PEM with wrong key size
	pemData := []byte(`-----BEGIN PUBLIC KEY-----
dG9vIHNob3J0
-----END PUBLIC KEY-----`)

	_, err := parsePublicKey(pemData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid public key size")
}

func TestParsePublicKey_ValidRawKey(t *testing.T) {
	// Generate a real key pair
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create PEM with raw 32-byte key
	pemData := fmt.Sprintf(`-----BEGIN PUBLIC KEY-----
%s
-----END PUBLIC KEY-----`, base64.StdEncoding.EncodeToString(pub))

	parsedKey, err := parsePublicKey([]byte(pemData))
	require.NoError(t, err)
	assert.Equal(t, []byte(pub), []byte(parsedKey))
}
