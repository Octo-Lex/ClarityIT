package approval

import (
	"time"

	"github.com/clarityit/api/internal/iam"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func issueTokenHelper(secret, userID, email, name, teamID, teamRole string) (string, error) {
	now := time.Now()
	claims := &iam.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "clarityit",
			Subject:   userID,
			ID:        uuid.New().String(),
		},
		UserID:       userID,
		Email:        email,
		Name:         name,
		TeamID:       teamID,
		TeamRole:     teamRole,
		IsOwner:      teamRole == "owner",
		TokenVersion: 1,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
