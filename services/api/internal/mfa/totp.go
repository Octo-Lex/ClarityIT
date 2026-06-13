package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"time"
)

const (
	totpStep     = 30 // seconds
	totpDigits   = 6
	totpWindow   = 1  // ±1 step for clock skew
)

// GenerateTOTPSecret generates a random 20-byte TOTP secret and returns
// the raw bytes and the base32-encoded string.
func GenerateTOTPSecret() ([]byte, string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", fmt.Errorf("generate secret: %w", err)
	}
	// Use standard base32 encoding (no padding) as per RFC 6238
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encoded := encoder.EncodeToString(secret)
	return secret, encoded, nil
}

// GenerateTOTP computes the TOTP value for a secret at a given time.
func GenerateTOTP(secret []byte, t time.Time) string {
	counter := uint64(t.Unix()) / totpStep
	return computeTOTP(secret, int64(counter))
}

// ValidateTOTP checks a code against the secret within the allowed window.
// Returns true if the code is valid for the current, previous, or next step.
func ValidateTOTP(secret []byte, code string, t time.Time) bool {
	counter := uint64(t.Unix()) / totpStep

	for i := -totpWindow; i <= totpWindow; i++ {
		expected := computeTOTP(secret, int64(counter)+int64(i))
		if hmacEqual([]byte(expected), []byte(code)) {
			return true
		}
	}
	return false
}

// ProvisioningURI generates an otpauth:// URI for QR code rendering.
// Format: otpauth://totp/Label?secret=SECRET&issuer=ISSUER&algorithm=SHA1&digits=6&period=30
func ProvisioningURI(account, issuer, secretB32 string) string {
	label := url.PathEscape(fmt.Sprintf("%s:%s", issuer, account))
	q := url.Values{}
	q.Set("secret", secretB32)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", totpDigits))
	q.Set("period", fmt.Sprintf("%d", totpStep))
	return fmt.Sprintf("otpauth://totp/%s?%s", label, q.Encode())
}

// computeTOTP implements the core HOTP/TOTP algorithm (RFC 4226).
func computeTOTP(secret []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := int(hash[len(hash)-1] & 0x0f)
	binary := (uint32(hash[offset])&0x7f)<<24 |
		uint32(hash[offset+1])<<16 |
		uint32(hash[offset+2])<<8 |
		uint32(hash[offset+3])

	otp := binary % uint32(math.Pow10(totpDigits))
	return fmt.Sprintf("%0*d", totpDigits, otp)
}

// hmacEqual is a constant-time comparison.
func hmacEqual(a, b []byte) bool {
	return hmac.Equal(a, b)
}

// GenerateRecoveryCodes generates n random recovery codes.
// Returns the raw codes (to show once) and their HMAC hashes (to store).
func GenerateRecoveryCodes(n int, hmacKey string) ([]string, []string) {
	codes := make([]string, n)
	hashes := make([]string, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 8)
		rand.Read(buf)
		code := fmt.Sprintf("%x", buf)
		codes[i] = code
		hashes[i] = hashRecoveryCode(hmacKey, code)
	}
	return codes, hashes
}

// hashRecoveryCode hashes a recovery code using HMAC-SHA256.
func hashRecoveryCode(hmacKey, code string) string {
	mac := hmac.New(sha1.New, []byte(hmacKey))
	mac.Write([]byte(code))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
