package crypto

import (
	"crypto/ed25519"
	_ "embed"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
)

//go:embed license_public_key.pem
var embeddedPublicKeyPEM []byte

// Verifier handles Ed25519 signature verification for licenses.
type Verifier struct {
	publicKey ed25519.PublicKey
}

// NewVerifier creates a new verifier using the embedded public key.
func NewVerifier() (*Verifier, error) {
	publicKey, err := parsePublicKey(embeddedPublicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded public key: %w", err)
	}
	return &Verifier{publicKey: publicKey}, nil
}

// NewVerifierWithKey creates a verifier with a custom public key (for testing).
func NewVerifierWithKey(publicKey ed25519.PublicKey) *Verifier {
	return &Verifier{publicKey: publicKey}
}

// Verify checks if the license signature is valid.
func (v *Verifier) Verify(license *domain.License) bool {
	if license == nil || !license.IsActivated() {
		return false
	}

	signatureBytes, err := license.SignatureBytes()
	if err != nil {
		return false
	}

	signedData := v.buildSignedData(license)
	return ed25519.Verify(v.publicKey, []byte(signedData), signatureBytes)
}

// buildSignedData creates the canonical string that was signed.
// Format: "license_id|plan|entitlements|expires_at"
func (v *Verifier) buildSignedData(license *domain.License) string {
	entitlements := strings.Join(license.Entitlements, ",")
	expiresAt := license.ExpiresAt.Format(time.RFC3339)

	return fmt.Sprintf("%s|%s|%s|%s",
		license.LicenseID.String(),
		license.Plan,
		entitlements,
		expiresAt,
	)
}

// parsePublicKey parses a PEM-encoded Ed25519 public key.
func parsePublicKey(pemData []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Ed25519 public keys are 32 bytes
	// The PEM may contain a SubjectPublicKeyInfo wrapper or just the raw key
	keyData := block.Bytes

	// If it's a SubjectPublicKeyInfo (common format), extract the key
	// SubjectPublicKeyInfo for Ed25519 has a 12-byte header
	if len(keyData) == 44 {
		// Skip the ASN.1 header (12 bytes for Ed25519)
		keyData = keyData[12:]
	}

	if len(keyData) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(keyData), ed25519.PublicKeySize)
	}

	return ed25519.PublicKey(keyData), nil
}
