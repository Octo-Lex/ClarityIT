package iam

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Password hashing with bcrypt cost 12
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Token generation
func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// HMAC for PII
func HMACString(key, value string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// JWT
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	TeamID    string `json:"team_id,omitempty"`
	TeamRole  string `json:"team_role,omitempty"`
	IsOwner   bool   `json:"is_platform_owner,omitempty"`
	TokenVersion int `json:"tv"`
}

func IssueAccessToken(secret string, userID, email, name string, teamID, teamRole *string, isPlatformOwner bool, tokenVersion int, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "clarityit",
			Subject:   userID,
			ID:        uuid.New().String(),
		},
		UserID:       userID,
		Email:        email,
		Name:         name,
		IsOwner:      isPlatformOwner,
		TokenVersion: tokenVersion,
	}
	if teamID != nil {
		claims.TeamID = *teamID
	}
	if teamRole != nil {
		claims.TeamRole = *teamRole
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseAccessToken(secret, tokenStr string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// Audit helpers
type AuditEvent struct {
	EventID       string
	EventType     string
	ActorID       string
	ActorType     string
	TeamID        string
	AggregateType string
	AggregateID   string
	RequestID     string
	Action        string
	EntityType    string
	EntityID      string
	OldValue      json.RawMessage
	NewValue      json.RawMessage
	Summary       string
	IPHMAC        string
	UserAgentHMAC string
	IdempotencyKey string
}

// Outbox helpers
type OutboxEvent struct {
	ID            string
	EventType     string
	EventVersion  int
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
	Status        string
}

// Generate API key
func GenerateAPIKey() string {
	prefix := make([]byte, 4)
	rand.Read(prefix)
	secret := make([]byte, 28)
	rand.Read(secret)
	return "clt_" + hex.EncodeToString(prefix) + hex.EncodeToString(secret)
}

func GetKeyPrefix(key string) string {
	if len(key) > 12 {
		return key[:12]
	}
	return key
}
