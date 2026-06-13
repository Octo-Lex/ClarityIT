package wsx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clarityit/api/internal/iam"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

func contextWithClaims(r *http.Request, claims *iam.TokenClaims) context.Context {
	return context.WithValue(r.Context(), "claims", claims)
}

func TestPhase5_WebSocket(t *testing.T) {
	jwtSecret := "test-secret"

	issueToken := func(userID, teamID, role string) string {
		claims := iam.TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			UserID:  userID,
			TeamID:  teamID,
			TeamRole: role,
		}
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
		return tok
	}

	hub := NewHub()

	// Create test server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate middleware: parse JWT and set claims in context
		auth := r.Header.Get("Authorization")
		if auth == "" {
			// Check query param for WebSocket
			auth = "Bearer " + r.URL.Query().Get("token")
		}
		if strings.HasPrefix(auth, "Bearer ") {
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			if claims, err := iam.ParseAccessToken(jwtSecret, tokenStr); err == nil {
				ctx := contextWithClaims(r, claims)
				r = r.WithContext(ctx)
			}
		}
		hub.HandleWS(w, r)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	t.Run("Unauthenticated_WebSocket_Rejected", func(t *testing.T) {
		_, resp, err := websocket.DefaultDialer.Dial(wsURL+"/api/ws", nil)
		if err == nil {
			t.Error("Should reject unauthenticated WebSocket")
		}
		if resp != nil && resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Authenticated_WebSocket_Accepted", func(t *testing.T) {
		token := issueToken("user-1", "team-1", "owner")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/api/ws?token="+token, nil)
		if err != nil {
			t.Fatalf("Should accept authenticated WebSocket: %v", err)
		}
		conn.Close()
	})

	t.Run("Team_Scoped_Event_Delivered", func(t *testing.T) {
		token := issueToken("user-2", "team-2", "owner")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/api/ws?token="+token, nil)
		if err != nil {
			t.Fatalf("Connect: %v", err)
		}
		defer conn.Close()

		// Give the server a moment to register
		time.Sleep(50 * time.Millisecond)

		// Broadcast to team-2
		hub.BroadcastToTeam("team-2", map[string]string{"type": "test", "msg": "hello"})

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Read: %v", err)
		}

		var parsed map[string]any
		json.Unmarshal(msg, &parsed)
		if parsed["type"] != "test" {
			t.Errorf("Unexpected message: %s", string(msg))
		}
	})

	t.Run("Cross_Team_Event_Not_Delivered", func(t *testing.T) {
		token := issueToken("user-3", "team-3", "owner")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/api/ws?token="+token, nil)
		if err != nil {
			t.Fatalf("Connect: %v", err)
		}
		defer conn.Close()

		time.Sleep(50 * time.Millisecond)

		// Broadcast to team-4 (not team-3)
		hub.BroadcastToTeam("team-4", map[string]string{"type": "wrong_team"})

		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, _, err = conn.ReadMessage()
		if err == nil {
			t.Error("Should not receive cross-team event")
		}
	})
}
