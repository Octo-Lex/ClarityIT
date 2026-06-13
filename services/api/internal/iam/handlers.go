package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/email"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	pool  *pgxpool.Pool
	cfg   *config.Config
	email *email.Service
}

func NewHandler(pool *pgxpool.Pool, cfg *config.Config) *Handler {
	return &Handler{pool: pool, cfg: cfg, email: nil}
}

func NewHandlerWithEmail(pool *pgxpool.Pool, cfg *config.Config, em *email.Service) *Handler {
	return &Handler{pool: pool, cfg: cfg, email: em}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/bootstrap", h.Bootstrap)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
	r.Get("/me", h.Me)
	r.Post("/switch-team", h.SwitchTeam)
	r.Get("/permissions", h.Permissions)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
	r.Post("/change-password", h.ChangePassword)
	r.Get("/sessions", h.ListSessions)
	r.Delete("/sessions/{id}", h.RevokeSession)

	return r
}

// ─── Bootstrap ───

type BootstrapRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	TeamName string `json:"team_name"`
}

func (h *Handler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check bootstrap lock
	var isLocked bool
	if err := h.pool.QueryRow(ctx, "SELECT is_locked FROM bootstrap_lock WHERE id = 1").Scan(&isLocked); err != nil {
		writeErr(w, http.StatusInternalServerError, "bootstrap check failed")
		return
	}
	if isLocked {
		writeErr(w, http.StatusConflict, "Platform already bootstrapped")
		return
	}

	var req BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "Name, email, and password (min 8 chars) required")
		return
	}
	if req.TeamName == "" {
		req.TeamName = "Platform"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Password hashing failed")
		return
	}

	userID := uuid.New()
	teamID := uuid.New()
	ipHMAC := HMACString(h.cfg.HMACKey, r.RemoteAddr)

	// Transaction: create user, team, memberships, platform role, lock bootstrap, audit, outbox
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Create user
		_, err := tx.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, name, is_active)
			VALUES ($1, $2, $3, $4, TRUE)
		`, userID, req.Email, string(passwordHash), req.Name)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		// Create team
		slug := strings.ToLower(strings.ReplaceAll(req.TeamName, " ", "-"))
		_, err = tx.Exec(ctx, `
			INSERT INTO teams (id, name, slug)
			VALUES ($1, $2, $3)
		`, teamID, req.TeamName, slug)
		if err != nil {
			return fmt.Errorf("create team: %w", err)
		}

		// Get owner role ID
		var ownerRoleID uuid.UUID
		if err := tx.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'owner'").Scan(&ownerRoleID); err != nil {
			return fmt.Errorf("get owner role: %w", err)
		}

		// Add user as team owner
		_, err = tx.Exec(ctx, `
			INSERT INTO team_memberships (user_id, team_id, role_id)
			VALUES ($1, $2, $3)
		`, userID, teamID, ownerRoleID)
		if err != nil {
			return fmt.Errorf("add membership: %w", err)
		}

		// Get platform_owner role ID
		var platformOwnerID uuid.UUID
		if err := tx.QueryRow(ctx, "SELECT id FROM platform_roles WHERE name = 'platform_owner'").Scan(&platformOwnerID); err != nil {
			return fmt.Errorf("get platform_owner role: %w", err)
		}

		// Assign platform_owner
		_, err = tx.Exec(ctx, `
			INSERT INTO user_platform_roles (user_id, platform_role_id, granted_by)
			VALUES ($1, $2, $1)
		`, userID, platformOwnerID)
		if err != nil {
			return fmt.Errorf("assign platform role: %w", err)
		}

		// Audit
		audit.Write(ctx, tx, audit.Event{
			ActorID:   userID,
			ActorType: "system",
			Action:    "platform.bootstrapped",
			EntityType: "user",
			Summary:   "Platform bootstrapped",
			IPHMAC:    ipHMAC,
		})

		// Outbox
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.identity.user.bootstrapped",
			AggregateType: "user",
			AggregateID:   userID.String(),
			Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s","team_id":"%s"}`, userID, teamID)),
		})

		// Lock bootstrap
		_, err = tx.Exec(ctx, "UPDATE bootstrap_lock SET is_locked = TRUE, locked_by_user_id = $1, locked_at = NOW() WHERE id = 1", userID)
		if err != nil {
			return fmt.Errorf("lock bootstrap: %w", err)
		}

		return nil
	})

	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Issue tokens
	accessToken, err := IssueAccessToken(h.cfg.JWTSecret, userID.String(), req.Email, req.Name,
		strPtr(teamID.String()), strPtr("owner"), true, 1, h.cfg.AccessTokenTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Token generation failed")
		return
	}

	refreshToken := GenerateToken()
	refreshHash := HashToken(refreshToken)
	familyID := uuid.New()

	_, err = h.pool.Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, ip_hmac, expires_at)
		VALUES ($1, $2, $3, NOW() + $4)
	`, familyID, userID, ipHMAC, h.cfg.RefreshTokenTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Session creation failed")
		return
	}

	sessionID := familyID // Use same ID for first session
	_, err = h.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, session_id, token_hash, family_id, expires_at)
		VALUES ($1, $2, $3, $4, NOW() + $5)
	`, userID, sessionID, refreshHash, familyID, h.cfg.RefreshTokenTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Refresh token creation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": map[string]any{
			"id":    userID,
			"email": req.Email,
			"name":  req.Name,
		},
		"team": map[string]any{
			"id":   teamID,
			"name": req.TeamName,
			"role": "owner",
		},
	})
}

// ─── Register ───

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify bootstrap is locked
	var isLocked bool
	if err := h.pool.QueryRow(ctx, "SELECT is_locked FROM bootstrap_lock WHERE id = 1").Scan(&isLocked); err != nil || !isLocked {
		writeErr(w, http.StatusConflict, "Platform not bootstrapped")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "Name, email, and password (min 8 chars) required")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Password hashing failed")
		return
	}

	userID := uuid.New()
	ipHMAC := HMACString(h.cfg.HMACKey, r.RemoteAddr)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Check email uniqueness
		var exists bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)", req.Email).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Email already registered")
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, name, is_active)
			VALUES ($1, $2, $3, $4, TRUE)
		`, userID, req.Email, string(passwordHash), req.Name)
		if err != nil {
			return err
		}

		audit.Write(ctx, tx, audit.Event{
			ActorID:   userID,
			ActorType: "user",
			Action:    "identity.user.registered",
			EntityType: "user",
			Summary:   "User registered",
			IPHMAC:    ipHMAC,
		})

		outbox.Write(ctx, tx, nil, outbox.Event{
			EventType:     "clarity.v1.identity.user.registered",
			AggregateType: "user",
			AggregateID:   userID.String(),
			Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s"}`, userID)),
		})

		return nil
	})

	if err != nil {
		if err.Error() == "Email already registered" {
			writeErr(w, http.StatusConflict, err.Error())
		} else {
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Create session and tokens
	accessToken, refreshToken, err := h.createSession(ctx, userID, req.Email, req.Name, nil, nil, false, 1, ipHMAC)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// ─── Login ───

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ipHMAC := HMACString(h.cfg.HMACKey, r.RemoteAddr)
	uaHMAC := HMACString(h.cfg.HMACKey, r.UserAgent())

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	emailHMAC := HMACString(h.cfg.HMACKey, req.Email)

	var userID uuid.UUID
	var email, name, passwordHash string
	var tokenVersion int
	err := h.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, token_version
		FROM users WHERE email = $1 AND is_active = TRUE AND deleted_at IS NULL
	`, req.Email).Scan(&userID, &email, &name, &passwordHash, &tokenVersion)
	if err != nil {
		// Sanitized security audit for failed login (user not found)
		h.writeSecurityAudit(ctx, "identity.login.failed", "session",
			uuid.Nil, emailHMAC, ipHMAC, uaHMAC,
			json.RawMessage(`{"failure_reason":"invalid_credentials"}`))
		writeErr(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		// Sanitized security audit for failed login (wrong password)
		h.writeSecurityAudit(ctx, "identity.login.failed", "session",
			uuid.Nil, emailHMAC, ipHMAC, uaHMAC,
			json.RawMessage(`{"failure_reason":"invalid_credentials"}`))
		writeErr(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Get team context
	teamID, teamRole, isPlatformOwner := h.getUserTeamContext(ctx, userID)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Update last login
		_, err := tx.Exec(ctx, "UPDATE users SET last_login_at = NOW() WHERE id = $1", userID)
		if err != nil {
			return err
		}

		audit.Write(ctx, tx, audit.Event{
			ActorID:   userID,
			ActorType: "user",
			Action:    "identity.session.created",
			EntityType: "session",
			EntityID:  userID,
			Summary:   "User logged in",
			IPHMAC:    ipHMAC,
		})

		outbox.Write(ctx, tx, nil, outbox.Event{
			EventType:     "clarity.v1.identity.session.created",
			AggregateType: "session",
			AggregateID:   userID.String(),
			Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s"}`, userID)),
		})

		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	var teamIDStr, teamRoleStr *string
	if teamID != nil {
		teamIDStr = strPtr(teamID.String())
	}
	if teamRole != nil {
		teamRoleStr = strPtr(*teamRole)
	}

	accessToken, refreshToken, err := h.createSession(ctx, userID, email, name, teamIDStr, teamRoleStr, isPlatformOwner, tokenVersion, ipHMAC)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// ─── Refresh (with token family rotation) ───

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tokenHash := HashToken(body.RefreshToken)

	// Look up token
	var tokenID, userID, familyID, sessionID uuid.UUID
	var replacedBy *uuid.UUID
	var reuseDetected, revoked *time.Time
	var expiresAt time.Time
	var email, name string
	var tokenVersion int

	err := h.pool.QueryRow(ctx, `
		SELECT rt.id, rt.user_id, rt.family_id, rt.session_id,
		       rt.replaced_by_token_id, rt.reuse_detected_at, rt.revoked_at,
		       rt.expires_at,
		       u.email, u.name, u.token_version
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id AND u.is_active = TRUE AND u.deleted_at IS NULL
		WHERE rt.token_hash = $1
	`, tokenHash).Scan(&tokenID, &userID, &familyID, &sessionID,
		&replacedBy, &reuseDetected, &revoked,
		&expiresAt, &email, &name, &tokenVersion)

	if err != nil {
		writeErr(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	// Check if revoked
	if revoked != nil {
		writeErr(w, http.StatusUnauthorized, "Token revoked")
		return
	}

	// Check expiry
	if time.Now().After(expiresAt) {
		writeErr(w, http.StatusUnauthorized, "Token expired")
		return
	}

	// REUSE DETECTION: if token was already rotated (replaced_by_token_id is set)
	if replacedBy != nil {
		// Token reuse detected — revoke entire family
		_, _ = h.pool.Exec(ctx, `
			UPDATE refresh_tokens
			SET reuse_detected_at = NOW()
			WHERE id = $1
		`, tokenID)

		// Revoke all tokens in the family
		_, _ = h.pool.Exec(ctx, `
			UPDATE refresh_tokens
			SET revoked_at = NOW()
			WHERE family_id = $1 AND revoked_at IS NULL
		`, familyID)

		// Revoke the session
		_, _ = h.pool.Exec(ctx, `
			UPDATE user_sessions
			SET revoked_at = NOW()
			WHERE id = $1
		`, sessionID)

		// Audit reuse detection
		database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
			audit.Write(ctx, tx, audit.Event{
				ActorID:       userID,
				ActorType:     "user",
				Action:        "identity.session.reuse_detected",
				EntityType: "session",
				Summary:       "Refresh token reuse detected, family revoked",
			})
			return nil
		})

		writeErr(w, http.StatusUnauthorized, "Token reuse detected, session revoked")
		return
	}

	// NORMAL ROTATION: create new token in same family
	newRefreshToken := GenerateToken()
	newTokenHash := HashToken(newRefreshToken)
	newTokenID := uuid.New()

	// Insert new token first (so FK on old token's replaced_by_token_id is valid)
	_, err = h.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, session_id, token_hash, family_id, expires_at)
		VALUES ($1, $2, $3, $4, $5, NOW() + $6)
	`, newTokenID, userID, sessionID, newTokenHash, familyID, h.cfg.RefreshTokenTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "New token creation failed")
		return
	}

	// Now update old token to point to the new one
	_, err = h.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET replaced_by_token_id = $1, rotated_at = NOW()
		WHERE id = $2
	`, newTokenID, tokenID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Token rotation failed")
		return
	}

	// Issue new access token
	teamID, teamRole, isPlatformOwner := h.getUserTeamContext(ctx, userID)
	var teamIDStr, teamRoleStr *string
	if teamID != nil {
		teamIDStr = strPtr(teamID.String())
	}
	if teamRole != nil {
		teamRoleStr = strPtr(*teamRole)
	}

	accessToken, err := IssueAccessToken(h.cfg.JWTSecret, userID.String(), email, name,
		teamIDStr, teamRoleStr, isPlatformOwner, tokenVersion, h.cfg.AccessTokenTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Access token failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

// ─── Logout ───

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	userID, _ := uuid.Parse(claims.UserID)
	ipHMAC := HMACString(h.cfg.HMACKey, r.RemoteAddr)

	// Revoke all refresh tokens for user's active sessions
	_, err := h.pool.Exec(ctx, `
		UPDATE refresh_tokens rt
		SET revoked_at = NOW()
		FROM user_sessions us
		WHERE rt.session_id = us.id
		  AND us.user_id = $1
		  AND us.revoked_at IS NULL
		  AND rt.revoked_at IS NULL
	`, userID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Logout failed")
		return
	}

	// Revoke sessions
	_, _ = h.pool.Exec(ctx, "UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", userID)

	// Audit
	database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		audit.Write(ctx, tx, audit.Event{
			ActorID:       userID,
			ActorType:     "user",
			Action:        "identity.session.revoked",
			EntityType: "session",
			Summary:       "User logged out",
			IPHMAC:        ipHMAC,
		})
		return nil
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out"})
}

// ─── Me ───

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	userID, _ := uuid.Parse(claims.UserID)

	// Get user
	var email, name string
	var isActive bool
	var avatarURL *string
	err := h.pool.QueryRow(ctx, `
		SELECT email, name, is_active, avatar_url FROM users WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&email, &name, &isActive, &avatarURL)
	if err != nil {
		writeErr(w, http.StatusNotFound, "User not found")
		return
	}

	// Get teams with roles
	rows, err := h.pool.Query(ctx, `
		SELECT t.id, t.name, t.slug, t.icon, r.name
		FROM team_memberships tm
		JOIN teams t ON t.id = tm.team_id AND t.deleted_at IS NULL
		JOIN roles r ON r.id = tm.role_id
		WHERE tm.user_id = $1
	`, userID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load teams")
		return
	}
	defer rows.Close()

	type TeamInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
		Icon string `json:"icon"`
		Role string `json:"role"`
	}
	var teams []TeamInfo
	for rows.Next() {
		var t TeamInfo
		rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Icon, &t.Role)
		teams = append(teams, t)
	}

	// Get platform roles
	prows, err := h.pool.Query(ctx, `
		SELECT pr.name FROM user_platform_roles upr
		JOIN platform_roles pr ON pr.id = upr.platform_role_id
		WHERE upr.user_id = $1 AND upr.revoked_at IS NULL
	`, userID)
	if err == nil {
		defer prows.Close()
		var platformRoles []string
		for prows.Next() {
			var pr string
			prows.Scan(&pr)
			platformRoles = append(platformRoles, pr)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":       userID,
		"email":    email,
		"name":     name,
		"active":   isActive,
		"teams":    teams,
	})
}

// ─── Switch Team ───

func (h *Handler) SwitchTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var body struct {
		TeamID string `json:"team_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	teamID, err := uuid.Parse(body.TeamID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	userID, _ := uuid.Parse(claims.UserID)

	// Check membership and get role
	var roleName string
	err = h.pool.QueryRow(ctx, `
		SELECT r.name FROM team_memberships tm
		JOIN roles r ON r.id = tm.role_id
		WHERE tm.user_id = $1 AND tm.team_id = $2
	`, userID, teamID).Scan(&roleName)
	if err != nil {
		writeErr(w, http.StatusForbidden, "Not a member of this team")
		return
	}

	// Issue new access token with team context
	accessToken, err := IssueAccessToken(h.cfg.JWTSecret, claims.UserID, claims.Email, claims.Name,
		strPtr(teamID.String()), strPtr(roleName), claims.IsOwner, claims.TokenVersion, h.cfg.AccessTokenTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Token generation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"team_id":      teamID,
		"role":         roleName,
	})
}

// ─── Permissions ───

func (h *Handler) Permissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	if claims.TeamID == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"role":        nil,
			"team_id":     nil,
			"permissions": []string{},
		})
		return
	}

	userID, _ := uuid.Parse(claims.UserID)
	teamID, _ := uuid.Parse(claims.TeamID)

	rows, err := h.pool.Query(ctx, `
		SELECT p.name FROM team_memberships tm
		JOIN role_permissions rp ON rp.role_id = tm.role_id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE tm.user_id = $1 AND tm.team_id = $2
	`, userID, teamID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load permissions")
		return
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		rows.Scan(&p)
		perms = append(perms, p)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"role":        claims.TeamRole,
		"team_id":     claims.TeamID,
		"permissions": perms,
	})
}

// ─── Forgot Password ───

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ipHMAC := HMACString(h.cfg.HMACKey, r.RemoteAddr)
	uaHMAC := HMACString(h.cfg.HMACKey, r.UserAgent())

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	emailHMAC := HMACString(h.cfg.HMACKey, body.Email)

	var userID uuid.UUID
	err := h.pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1 AND is_active = TRUE AND deleted_at IS NULL", body.Email).Scan(&userID)
	if err != nil {
		// Don't reveal whether email exists — still return success
		writeJSON(w, http.StatusOK, map[string]string{"message": "If that email exists, a reset link has been sent"})
		return
	}

	token := GenerateToken()
	tokenHash := HashToken(token)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
			VALUES ($1, $2, NOW() + INTERVAL '1 hour')
		`, userID, tokenHash)
		if err != nil {
			return err
		}

		audit.Write(ctx, tx, audit.Event{
			ActorID:    userID,
			ActorType:  "user",
			Action:     "identity.password_reset.requested",
			EntityType: "password_reset",
			EntityID:   userID,
			Summary:    "Password reset requested",
			IPHMAC:     ipHMAC,
			UserAgentHMAC: uaHMAC,
			NewValue:   json.RawMessage(fmt.Sprintf(`{"email_hmac":"%s"}`, emailHMAC)),
		})

		outbox.Write(ctx, tx, nil, outbox.Event{
			EventType:     "clarity.v1.identity.password_reset.requested",
			AggregateType: "user",
			AggregateID:   userID.String(),
			Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s","email_hmac":"%s"}`, userID, emailHMAC)),
		})

		return nil
	})
	if err != nil {
		// Don't reveal error to client
		_ = err
	}

	// Send reset email or dev preview
	if h.email != nil && h.cfg.EmailMode == "smtp" {
		resetURL := fmt.Sprintf("%s/reset-password?token=%s", h.getWebBaseURL(), token)
		_ = h.email.Send(body.Email, "ClarityIT Password Reset",
			fmt.Sprintf("Reset your password: %s", resetURL))
	} else if h.cfg.EmailMode == "dev" {
		// Dev preview — safe hash-based URL
		writeJSON(w, http.StatusOK, map[string]any{
			"message":     "If that email exists, a reset link has been sent",
			"dev_preview": fmt.Sprintf("/reset-password?token=%s", token),
			"_dev_notice": "DEV MODE ONLY — token returned for local testing. Not available in production.",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "If that email exists, a reset link has been sent"})
}

// ─── Reset Password ───

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(body.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	tokenHash := HashToken(body.Token)

	var userID uuid.UUID
	var expiresAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT user_id, expires_at FROM password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL
	`, tokenHash).Scan(&userID, &expiresAt)
	if err != nil || time.Now().After(expiresAt) {
		writeErr(w, http.StatusBadRequest, "Invalid or expired reset token")
		return
	}

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), 12)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE users SET password_hash = $1, token_version = token_version + 1 WHERE id = $2", string(passwordHash), userID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1", tokenHash)
		if err != nil {
			return err
		}
		// Revoke all sessions
		_, err = tx.Exec(ctx, "UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", userID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", userID)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Password reset failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password reset successfully"})
}

// ─── Change Password ───

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(body.NewPassword) < 8 {
		writeErr(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	userID, _ := uuid.Parse(claims.UserID)
	var currentHash string
	err := h.pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		writeErr(w, http.StatusNotFound, "User not found")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(body.CurrentPassword)) != nil {
		writeErr(w, http.StatusBadRequest, "Current password is incorrect")
		return
	}

	newHash, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), 12)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE users SET password_hash = $1, token_version = token_version + 1 WHERE id = $2", string(newHash), userID)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Password change failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// ─── Sessions ───

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	userID, _ := uuid.Parse(claims.UserID)

	rows, err := h.pool.Query(ctx, `
		SELECT id, ip_hmac, created_at, expires_at
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load sessions")
		return
	}
	defer rows.Close()

	type Session struct {
		ID        string `json:"id"`
		IPHMAC    string `json:"ip_hmac"`
		CreatedAt string `json:"created_at"`
		ExpiresAt string `json:"expires_at"`
	}
	var sessions []Session
	for rows.Next() {
		var s Session
		rows.Scan(&s.ID, &s.IPHMAC, &s.CreatedAt, &s.ExpiresAt)
		sessions = append(sessions, s)
	}

	writeJSON(w, http.StatusOK, sessions)
}

func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	userID, _ := uuid.Parse(claims.UserID)
	sessionID := chi.URLParam(r, "id")

	sid, err := uuid.Parse(sessionID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	// Verify session belongs to user
	var count int
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_sessions WHERE id = $1 AND user_id = $2", sid, userID).Scan(&count)
	if count == 0 {
		writeErr(w, http.StatusNotFound, "Session not found")
		return
	}

	// Revoke session and all its refresh tokens
	database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		tx.Exec(ctx, "UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1", sid)
		tx.Exec(ctx, "UPDATE refresh_tokens SET revoked_at = NOW() WHERE session_id = $1", sid)
		return nil
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Session revoked"})
}

// ─── Helpers ───

func (h *Handler) createSession(ctx context.Context, userID uuid.UUID, email, name string, teamID, teamRole *string, isPlatformOwner bool, tokenVersion int, ipHMAC string) (string, string, error) {
	accessToken, err := IssueAccessToken(h.cfg.JWTSecret, userID.String(), email, name, teamID, teamRole, isPlatformOwner, tokenVersion, h.cfg.AccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	refreshToken := GenerateToken()
	refreshHash := HashToken(refreshToken)
	familyID := uuid.New()

	_, err = h.pool.Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, ip_hmac, expires_at)
		VALUES ($1, $2, $3, NOW() + $4)
	`, familyID, userID, ipHMAC, h.cfg.RefreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	_, err = h.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, session_id, token_hash, family_id, expires_at)
		VALUES ($1, $2, $3, $4, NOW() + $5)
	`, userID, familyID, refreshHash, familyID, h.cfg.RefreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (h *Handler) getUserTeamContext(ctx context.Context, userID uuid.UUID) (*uuid.UUID, *string, bool) {
	var isPlatformOwner bool
	h.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_platform_roles upr
			JOIN platform_roles pr ON pr.id = upr.platform_role_id
			WHERE upr.user_id = $1 AND pr.name = 'platform_owner' AND upr.revoked_at IS NULL
		)
	`, userID).Scan(&isPlatformOwner)

	var teamID uuid.UUID
	var roleName string
	err := h.pool.QueryRow(ctx, `
		SELECT t.id, r.name
		FROM team_memberships tm
		JOIN teams t ON t.id = tm.team_id AND t.deleted_at IS NULL
		JOIN roles r ON r.id = tm.role_id
		WHERE tm.user_id = $1
		ORDER BY tm.joined_at ASC
		LIMIT 1
	`, userID).Scan(&teamID, &roleName)
	if err != nil {
		return nil, nil, isPlatformOwner
	}

	return &teamID, &roleName, isPlatformOwner
}

// GetClaims extracts TokenClaims from request context.
// Exported for use by team/admin handler packages.
func GetClaims(r *http.Request) (*TokenClaims, bool) {
	claims, ok := r.Context().Value("claims").(*TokenClaims)
	return claims, ok
}

func getClaims(r *http.Request) (*TokenClaims, bool) {
	return GetClaims(r)
}

func strPtr(s string) *string { return &s }

func userUUID(u uuid.UUID) uuid.UUID { return u }

// writeSecurityAudit writes a sanitized security audit event outside a transaction.
// Used for events like failed login where no domain transaction exists.
// Uses HMAC'd values only — no raw PII.
func (h *Handler) writeSecurityAudit(ctx context.Context, action, entityType string, entityID uuid.UUID, emailHMAC, ipHMAC, uaHMAC string, metadata json.RawMessage) {
	eventID := uuid.New().String()
	summary := action
	if metadata != nil {
		var m map[string]any
		json.Unmarshal(metadata, &m)
		if reason, ok := m["failure_reason"]; ok {
			summary = fmt.Sprintf("%s: %s", action, reason)
		}
	}
	h.pool.Exec(ctx, `
		INSERT INTO audit_logs (
			event_id, actor_id, actor_type, action, entity_type, entity_id,
			old_value, new_value, change_summary, ip_hmac, user_agent_hmac
		) VALUES ($1, $2, 'system', $3, $4, $5, '{}', $6, $7, $8, $9)
	`, eventID, entityID, action, entityType, entityID,
		metadata, summary, ipHMAC, uaHMAC)
}

func (h *Handler) getWebBaseURL() string {
	return "http://localhost:3000"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"detail": msg})
}
