package mfa

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Crypto provides AES-256-GCM encryption/decryption for TOTP secrets.
// The encryption key is derived from MFA_KEY using HKDF-SHA256
// to ensure the AES key has proper entropy even if MFA_KEY is reused.
type Crypto struct {
	aead cipher.AEAD
}

// NewCrypto creates a Crypto instance from the MFA_KEY environment variable.
// The key is derived using HKDF-SHA256 to produce a 32-byte AES-256 key.
func NewCrypto(mfaKey string) (*Crypto, error) {
	if len(mfaKey) < 16 {
		return nil, fmt.Errorf("MFA_KEY must be at least 16 characters")
	}

	// Derive 32-byte AES key from MFA_KEY using HKDF-SHA256
	kdf := hkdf.New(sha256.New, []byte(mfaKey), []byte("clarityit-mfa-v1"), []byte("aes-256-gcm"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(kdf, aesKey); err != nil {
		return nil, fmt.Errorf("derive AES key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	// Zero out the derived key from memory
	for i := range aesKey {
		aesKey[i] = 0
	}

	return &Crypto{aead: aead}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns nonce || ciphertext.
func (c *Crypto) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts nonce || ciphertext using AES-256-GCM.
func (c *Crypto) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ct := ciphertext[nonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}
