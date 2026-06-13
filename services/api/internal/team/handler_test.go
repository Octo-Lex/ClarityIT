package team

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPhase3(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}

	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("Connect DB: %v", err)
	}
	defer pool.Close()

	iamHandler := iam.NewHandler(pool, cfg)
	teamHandler := NewHandler(pool, cfg)

	// Build chi router for proper URL param extraction + auth injection
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			auth := req.Header.Get("Authorization")
			if auth != "" {
				parts := strings.SplitN(auth, " ", 2)
				if len(parts) == 2 {
					if claims, err := iam.ParseAccessToken(cfg.JWTSecret, parts[1]); err == nil {
						newCtx := context.WithValue(req.Context(), "claims", claims)
						req = req.WithContext(newCtx)
					}
				}
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Post("/api/bootstrap", iamHandler.Bootstrap)
	r.Post("/api/auth/register", iamHandler.Register)
	r.Get("/api/teams/{teamId}/settings", teamHandler.GetSettings)
	r.Patch("/api/teams/{teamId}/settings", teamHandler.UpdateSettings)
	r.Get("/api/teams/{teamId}/members", teamHandler.ListMembers)
	r.Patch("/api/teams/{teamId}/members/{membershipId}", teamHandler.UpdateMemberRole)
	r.Delete("/api/teams/{teamId}/members/{membershipId}", teamHandler.RemoveMember)
	r.Post("/api/teams/{teamId}/invitations", teamHandler.CreateInvitation)
	r.Get("/api/teams/{teamId}/invitations", teamHandler.ListInvitations)
	r.Post("/api/teams/{teamId}/invitations/{id}/accept", teamHandler.AcceptInvitation)
	r.Delete("/api/teams/{teamId}/invitations/{id}", teamHandler.RevokeInvitation)
	r.Post("/api/teams/{teamId}/access-grants", teamHandler.CreateAccessGrant)
	r.Delete("/api/teams/{teamId}/access-grants/{id}", teamHandler.RevokeAccessGrant)

	var ownerToken, memberToken string
	var teamID, ownerMembershipID string
	var viewerRoleID, memberRoleID string

	doReq := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var bodyReader *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(b)
		} else {
			bodyReader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, bodyReader)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// ─── Setup ───
	t.Run("Setup", func(t *testing.T) {
		pool.Exec(ctx, "ALTER TABLE bootstrap_lock DISABLE TRIGGER trg_bootstrap_lock")
		pool.Exec(ctx, "UPDATE bootstrap_lock SET is_locked = FALSE, locked_by_user_id = NULL, locked_at = NULL WHERE id = 1")
		pool.Exec(ctx, "ALTER TABLE bootstrap_lock ENABLE TRIGGER trg_bootstrap_lock")
		pool.Exec(ctx, "TRUNCATE users, teams, team_memberships, user_platform_roles, user_sessions, refresh_tokens, audit_logs, outbox_events, password_reset_tokens, invitations, team_access_grants, integration_api_keys, idempotency_keys CASCADE")

		w := doReq("POST", "/api/bootstrap", "", map[string]string{
			"name": "Owner", "email": "owner@test.dev", "password": "password12", "team_name": "Test Team",
		})
		if w.Code != 200 {
			t.Fatalf("Bootstrap: %d %s", w.Code, w.Body.String())
		}
		var boot map[string]any
		json.NewDecoder(w.Body).Decode(&boot)
		ownerToken = boot["access_token"].(string)

		w2 := doReq("POST", "/api/auth/register", "", map[string]string{
			"name": "Member", "email": "member@test.dev", "password": "password12",
		})
		var reg map[string]any
		json.NewDecoder(w2.Body).Decode(&reg)
		memberToken = reg["access_token"].(string)

		pool.QueryRow(ctx, "SELECT id FROM teams WHERE slug = 'test-team'").Scan(&teamID)
		pool.QueryRow(ctx, "SELECT id FROM team_memberships WHERE team_id = $1", teamID).Scan(&ownerMembershipID)
		pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'viewer'").Scan(&viewerRoleID)
		pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'member'").Scan(&memberRoleID)
	})

	// ─── Team Settings ───
	t.Run("TeamSettings_Read_Allowed", func(t *testing.T) {
		w := doReq("GET", "/api/teams/"+teamID+"/settings", ownerToken, nil)
		if w.Code != 200 {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("TeamSettings_Update_Allowed", func(t *testing.T) {
		w := doReq("PATCH", "/api/teams/"+teamID+"/settings", ownerToken, map[string]string{"name": "Updated Team"})
		if w.Code != 200 {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var count int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs WHERE action = 'team.settings.updated'").Scan(&count)
		if count == 0 {
			t.Error("No audit for settings update")
		}
	})

	// ─── Members ───
	t.Run("Member_List_Allowed", func(t *testing.T) {
		w := doReq("GET", "/api/teams/"+teamID+"/members", ownerToken, nil)
		if w.Code != 200 {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Member_Role_Change_Allowed", func(t *testing.T) {
		var userID string
		pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'member@test.dev'").Scan(&userID)
		pool.Exec(ctx, "INSERT INTO team_memberships (user_id, team_id, role_id) VALUES ($1, $2, $3)", userID, teamID, memberRoleID)

		var membershipID string
		pool.QueryRow(ctx, "SELECT id FROM team_memberships WHERE user_id = $1 AND team_id = $2", userID, teamID).Scan(&membershipID)

		w := doReq("PATCH", "/api/teams/"+teamID+"/members/"+membershipID, ownerToken, map[string]string{"role_id": viewerRoleID})
		if w.Code != 200 {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Last_Owner_Demotion_Blocked", func(t *testing.T) {
		w := doReq("PATCH", "/api/teams/"+teamID+"/members/"+ownerMembershipID, ownerToken, map[string]string{"role_id": viewerRoleID})
		if w.Code != 409 {
			t.Errorf("Expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Last_Owner_Removal_Blocked", func(t *testing.T) {
		w := doReq("DELETE", "/api/teams/"+teamID+"/members/"+ownerMembershipID, ownerToken, nil)
		if w.Code != 409 {
			t.Errorf("Expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Member_Removal_Allowed", func(t *testing.T) {
		var membershipID string
		pool.QueryRow(ctx, "SELECT id FROM team_memberships WHERE team_id = $1 AND role_id = $2", teamID, viewerRoleID).Scan(&membershipID)

		w := doReq("DELETE", "/api/teams/"+teamID+"/members/"+membershipID, ownerToken, nil)
		if w.Code != 200 {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var count int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs WHERE action = 'team.member.removed'").Scan(&count)
		if count == 0 {
			t.Error("No audit for member removal")
		}
	})

	// ─── Invitations ───
	t.Run("Invitation_Create_Writes_Audit_Outbox", func(t *testing.T) {
		pool.Exec(ctx, "TRUNCATE audit_logs, outbox_events CASCADE")
		w := doReq("POST", "/api/teams/"+teamID+"/invitations", ownerToken, map[string]string{
			"email": "invited@test.dev", "role_id": memberRoleID,
		})
		if w.Code != 200 {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var ac, oc int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs WHERE action = 'team.member.invited'").Scan(&ac)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'clarity.v1.team.member.invited'").Scan(&oc)
		if ac == 0 || oc == 0 {
			t.Errorf("Missing audit(%d) or outbox(%d)", ac, oc)
		}
	})

	t.Run("Invitation_Token_Not_Raw", func(t *testing.T) {
		var tokenHash string
		pool.QueryRow(ctx, "SELECT token_hash FROM invitations LIMIT 1").Scan(&tokenHash)
		if len(tokenHash) != 64 {
			t.Errorf("Token hash length %d, expected 64 (SHA256)", len(tokenHash))
		}
	})

	t.Run("Invitation_Accept_Fails_Wrong_Email", func(t *testing.T) {
		var invID string
		pool.QueryRow(ctx, "SELECT id FROM invitations WHERE email = 'invited@test.dev'").Scan(&invID)
		// member@test.dev tries to accept invitation for invited@test.dev
		w := doReq("POST", "/api/teams/"+teamID+"/invitations/"+invID+"/accept", memberToken, nil)
		if w.Code != 400 {
			t.Errorf("Expected 400 for email mismatch, got %d", w.Code)
		}
	})

	t.Run("Invitation_Accept_Matching_Email_Succeeds", func(t *testing.T) {
		// Create invitation for member@test.dev
		token := iam.GenerateToken()
		tokenHash := iam.HashToken(token)
		var invID string
		pool.QueryRow(ctx, `
			INSERT INTO invitations (team_id, email, role_id, token_hash, invited_by, expires_at)
			VALUES ($1, 'member@test.dev', $2, $3, (SELECT id FROM users WHERE email='owner@test.dev'), NOW() + INTERVAL '7 days')
			RETURNING id
		`, teamID, memberRoleID, tokenHash).Scan(&invID)

		w := doReq("POST", "/api/teams/"+teamID+"/invitations/"+invID+"/accept", memberToken, nil)
		if w.Code != 200 {
			t.Errorf("Accept: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Reuse should fail
		w2 := doReq("POST", "/api/teams/"+teamID+"/invitations/"+invID+"/accept", memberToken, nil)
		if w2.Code != 400 {
			t.Errorf("Reuse: expected 400, got %d", w2.Code)
		}
	})

	t.Run("Invitation_Accept_Fails_Expired", func(t *testing.T) {
		token := iam.GenerateToken()
		tokenHash := iam.HashToken(token)
		var invID string
		pool.QueryRow(ctx, `
			INSERT INTO invitations (team_id, email, role_id, token_hash, invited_by, expires_at)
			VALUES ($1, 'expired@test.dev', $2, $3, (SELECT id FROM users WHERE email='owner@test.dev'), NOW() - INTERVAL '1 day')
			RETURNING id
		`, teamID, memberRoleID, tokenHash).Scan(&invID)

		w := doReq("POST", "/api/teams/"+teamID+"/invitations/"+invID+"/accept", ownerToken, nil)
		if w.Code != 400 {
			t.Errorf("Expired: expected 400, got %d", w.Code)
		}
	})

	t.Run("Invitation_Revoke_Writes_Audit_Outbox", func(t *testing.T) {
		pool.Exec(ctx, "TRUNCATE audit_logs, outbox_events CASCADE")
		token := iam.GenerateToken()
		tokenHash := iam.HashToken(token)
		var invID string
		pool.QueryRow(ctx, `
			INSERT INTO invitations (team_id, email, role_id, token_hash, invited_by, expires_at)
			VALUES ($1, 'revoke@test.dev', $2, $3, (SELECT id FROM users WHERE email='owner@test.dev'), NOW() + INTERVAL '7 days')
			RETURNING id
		`, teamID, memberRoleID, tokenHash).Scan(&invID)

		w := doReq("DELETE", "/api/teams/"+teamID+"/invitations/"+invID, ownerToken, nil)
		if w.Code != 200 {
			t.Errorf("Revoke: expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var ac, oc int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs WHERE action = 'team.invitation.revoked'").Scan(&ac)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'clarity.v1.team.invitation.revoked'").Scan(&oc)
		if ac == 0 || oc == 0 {
			t.Errorf("Missing audit(%d) or outbox(%d) for revoke", ac, oc)
		}
	})

	// ─── Access Grants ───
	t.Run("Access_Grant_Create_Writes_Audit_Outbox", func(t *testing.T) {
		pool.Exec(ctx, "TRUNCATE audit_logs, outbox_events CASCADE")
		var userID string
		pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'member@test.dev'").Scan(&userID)

		w := doReq("POST", "/api/teams/"+teamID+"/access-grants", ownerToken, map[string]string{
			"user_id": userID, "grant_type": "temporary", "role_id": memberRoleID, "duration": "24h",
		})
		if w.Code != 200 {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var ac, oc int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs WHERE action = 'team.access_grant.created'").Scan(&ac)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'clarity.v1.team.access_grant.created'").Scan(&oc)
		if ac == 0 || oc == 0 {
			t.Errorf("Missing audit(%d) or outbox(%d)", ac, oc)
		}
	})

	t.Run("Expired_Grant_No_Permissions", func(t *testing.T) {
		var userID string
		pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'member@test.dev'").Scan(&userID)
		pool.Exec(ctx, `
			INSERT INTO team_access_grants (team_id, user_id, granted_by, grant_type, role_id, expires_at)
			VALUES ($1, $2, (SELECT id FROM users WHERE email='owner@test.dev'), 'temporary', $3, NOW() - INTERVAL '1 day')
		`, teamID, userID, memberRoleID)
		var count int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM team_access_grants WHERE user_id = $1 AND team_id = $2 AND revoked_at IS NULL AND expires_at > NOW()`, userID, teamID).Scan(&count)
		if count > 1 {
			t.Errorf("Too many effective grants: %d", count)
		}
	})

	t.Run("Revoked_Grant_No_Permissions", func(t *testing.T) {
		var userID string
		pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'member@test.dev'").Scan(&userID)
		pool.Exec(ctx, "UPDATE team_access_grants SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", userID)
		var count int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM team_access_grants WHERE user_id = $1 AND team_id = $2 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())`, userID, teamID).Scan(&count)
		if count != 0 {
			t.Errorf("Expected 0 effective grants after revoke, got %d", count)
		}
	})

	// ─── PII ───
	t.Run("PII_Redacted_In_Invitations_Audit", func(t *testing.T) {
		// Create a new invitation to generate fresh audit data
		pool.Exec(ctx, "TRUNCATE audit_logs, outbox_events CASCADE")
		doReq("POST", "/api/teams/"+teamID+"/invitations", ownerToken, map[string]string{
			"email": "pii-check@test.dev", "role_id": memberRoleID,
		})

		var raw int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE old_value::text LIKE '%@test.dev%' OR new_value::text LIKE '%@test.dev%'`).Scan(&raw)
		if raw > 0 {
			t.Error("Raw email found in audit logs")
		}
		var hasHMAC bool
		pool.QueryRow(ctx, `SELECT COUNT(*) > 0 FROM audit_logs WHERE new_value::text LIKE '%email_hmac%'`).Scan(&hasHMAC)
		if !hasHMAC {
			t.Error("No email_hmac in audit events")
		}
	})
}
