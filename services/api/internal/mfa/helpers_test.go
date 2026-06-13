package mfa

import (
	"encoding/base32"
	"time"

	"github.com/clarityit/api/internal/iam"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// base32Decode decodes a base32 (no padding) string.
func base32Decode(s string) ([]byte, error) {
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	return encoder.DecodeString(s)
}

// issueTokenHelper issues a JWT token for test purposes.
func issueTokenHelper(secret, userID, email, name, teamID string) (string, error) {
	now := time.Now()
	uid, _ := uuid.Parse(userID)
	tid, _ := uuid.Parse(teamID)
	claims := &iam.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "clarityit",
			Subject:   userID,
			ID:        uuid.New().String(),
		},
		UserID:    userID,
		Email:     email,
		Name:      name,
		TeamID:    teamID,
		TeamRole:  "owner",
		IsOwner:   true,
		TokenVersion: 1,
	}
	_ = uid
	_ = tid
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
