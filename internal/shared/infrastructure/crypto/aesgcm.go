package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encrypter encrypts and decrypts data.
type Encrypter interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// AESEncrypter uses AES-GCM for encryption.
type AESEncrypter struct {
	aead cipher.AEAD
}

// NewAESGCMFromBase64Key creates an AESEncrypter from a base64-encoded 32-byte key.
func NewAESGCMFromBase64Key(encodedKey string) (*AESEncrypter, error) {
	if encodedKey == "" {
		return nil, errors.New("encryption key is empty")
	}
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &AESEncrypter{aead: aead}, nil
}

// Encrypt encrypts plaintext and prepends the nonce.
func (e *AESEncrypter) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := e.aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// Decrypt decrypts ciphertext with a nonce prefix.
func (e *AESEncrypter) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]
	return e.aead.Open(nil, nonce, data, nil)
}
