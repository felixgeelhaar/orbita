package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateValidKey generates a valid 32-byte base64-encoded key for testing.
func generateValidKey() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestNewAESGCMFromBase64Key(t *testing.T) {
	t.Run("creates encrypter with valid 32-byte key", func(t *testing.T) {
		key := generateValidKey()

		encrypter, err := NewAESGCMFromBase64Key(key)

		require.NoError(t, err)
		assert.NotNil(t, encrypter)
	})

	t.Run("returns error for empty key", func(t *testing.T) {
		encrypter, err := NewAESGCMFromBase64Key("")

		assert.Error(t, err)
		assert.Nil(t, encrypter)
		assert.Contains(t, err.Error(), "encryption key is empty")
	})

	t.Run("returns error for invalid base64", func(t *testing.T) {
		encrypter, err := NewAESGCMFromBase64Key("not-valid-base64!!!")

		assert.Error(t, err)
		assert.Nil(t, encrypter)
	})

	t.Run("returns error for key shorter than 32 bytes", func(t *testing.T) {
		shortKey := base64.StdEncoding.EncodeToString([]byte("short"))

		encrypter, err := NewAESGCMFromBase64Key(shortKey)

		assert.Error(t, err)
		assert.Nil(t, encrypter)
		assert.Contains(t, err.Error(), "encryption key must be 32 bytes")
	})

	t.Run("returns error for key longer than 32 bytes", func(t *testing.T) {
		longKey := make([]byte, 64)
		for i := range longKey {
			longKey[i] = byte(i)
		}
		encodedLongKey := base64.StdEncoding.EncodeToString(longKey)

		encrypter, err := NewAESGCMFromBase64Key(encodedLongKey)

		assert.Error(t, err)
		assert.Nil(t, encrypter)
		assert.Contains(t, err.Error(), "encryption key must be 32 bytes")
	})
}

func TestAESEncrypter_Encrypt(t *testing.T) {
	t.Run("encrypts plaintext successfully", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		plaintext := []byte("Hello, World!")

		ciphertext, err := encrypter.Encrypt(plaintext)

		require.NoError(t, err)
		assert.NotNil(t, ciphertext)
		// Ciphertext should be longer than plaintext (includes nonce and auth tag)
		assert.Greater(t, len(ciphertext), len(plaintext))
		// Ciphertext should not equal plaintext
		assert.NotEqual(t, plaintext, ciphertext)
	})

	t.Run("encrypts empty plaintext", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		plaintext := []byte{}

		ciphertext, err := encrypter.Encrypt(plaintext)

		require.NoError(t, err)
		assert.NotNil(t, ciphertext)
		// Should still have nonce and auth tag
		assert.Greater(t, len(ciphertext), 0)
	})

	t.Run("produces different ciphertext for same plaintext", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		plaintext := []byte("Same message")

		ciphertext1, err := encrypter.Encrypt(plaintext)
		require.NoError(t, err)

		ciphertext2, err := encrypter.Encrypt(plaintext)
		require.NoError(t, err)

		// Each encryption should produce different ciphertext due to random nonce
		assert.NotEqual(t, ciphertext1, ciphertext2)
	})
}

func TestAESEncrypter_Decrypt(t *testing.T) {
	t.Run("decrypts ciphertext successfully", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		originalPlaintext := []byte("Secret message")

		ciphertext, err := encrypter.Encrypt(originalPlaintext)
		require.NoError(t, err)

		decryptedPlaintext, err := encrypter.Decrypt(ciphertext)

		require.NoError(t, err)
		assert.Equal(t, originalPlaintext, decryptedPlaintext)
	})

	t.Run("decrypts empty plaintext", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		originalPlaintext := []byte{}

		ciphertext, err := encrypter.Encrypt(originalPlaintext)
		require.NoError(t, err)

		decryptedPlaintext, err := encrypter.Decrypt(ciphertext)

		require.NoError(t, err)
		// Empty slice and nil slice are semantically equivalent
		assert.Empty(t, decryptedPlaintext)
	})

	t.Run("returns error for ciphertext too short", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		shortCiphertext := []byte("short")

		decrypted, err := encrypter.Decrypt(shortCiphertext)

		assert.Error(t, err)
		assert.Nil(t, decrypted)
		assert.Contains(t, err.Error(), "ciphertext too short")
	})

	t.Run("returns error for tampered ciphertext", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		plaintext := []byte("Original message")
		ciphertext, err := encrypter.Encrypt(plaintext)
		require.NoError(t, err)

		// Tamper with the ciphertext
		ciphertext[len(ciphertext)-1] ^= 0xFF

		decrypted, err := encrypter.Decrypt(ciphertext)

		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})

	t.Run("returns error for wrong key", func(t *testing.T) {
		key1 := generateValidKey()
		encrypter1, err := NewAESGCMFromBase64Key(key1)
		require.NoError(t, err)

		// Create a different key
		differentKey := make([]byte, 32)
		for i := range differentKey {
			differentKey[i] = byte(i + 100)
		}
		key2 := base64.StdEncoding.EncodeToString(differentKey)
		encrypter2, err := NewAESGCMFromBase64Key(key2)
		require.NoError(t, err)

		plaintext := []byte("Secret")
		ciphertext, err := encrypter1.Encrypt(plaintext)
		require.NoError(t, err)

		// Try to decrypt with different key
		decrypted, err := encrypter2.Decrypt(ciphertext)

		assert.Error(t, err)
		assert.Nil(t, decrypted)
	})
}

func TestAESEncrypter_RoundTrip(t *testing.T) {
	t.Run("encrypts and decrypts various message sizes", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		testCases := []struct {
			name      string
			plaintext []byte
		}{
			{"empty", []byte{}},
			{"single byte", []byte{0x42}},
			{"short string", []byte("Hello")},
			{"medium string", []byte("This is a medium length message for testing")},
			{"long string", make([]byte, 1024)},
			{"binary data", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ciphertext, err := encrypter.Encrypt(tc.plaintext)
				require.NoError(t, err)

				decrypted, err := encrypter.Decrypt(ciphertext)
				require.NoError(t, err)

				// Handle empty slice vs nil slice equivalence
				if len(tc.plaintext) == 0 {
					assert.Empty(t, decrypted)
				} else {
					assert.Equal(t, tc.plaintext, decrypted)
				}
			})
		}
	})
}

func TestEncrypterInterface(t *testing.T) {
	t.Run("AESEncrypter implements Encrypter interface", func(t *testing.T) {
		key := generateValidKey()
		encrypter, err := NewAESGCMFromBase64Key(key)
		require.NoError(t, err)

		// Verify it implements the interface
		var _ Encrypter = encrypter
	})
}
