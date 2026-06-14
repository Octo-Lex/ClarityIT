package mfa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WebAuthnHandler handles WebAuthn/FIDO2 registration and authentication.
// It is isolated from the TOTP handler and does not change any existing MFA semantics.
type WebAuthnHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
	wa   *webauthn.WebAuthn

	// In-memory session store for WebAuthn challenges (short-lived)
	sessions   map[uuid.UUID]*webauthn.SessionData
	sessionsMu sync.Mutex
}

// webauthnUser implements the webauthn.User interface.
type webauthnUser struct {
	id          uuid.UUID
	email       string
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte {
	return u.id[:]
}

func (u *webauthnUser) WebAuthnName() string {
	return u.email
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	return u.email
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u *webauthnUser) WebAuthnIcon() string {
	return ""
}

func NewWebAuthnHandler(pool *pgxpool.Pool, cfg *config.Config) (*WebAuthnHandler, error) {
	h := &WebAuthnHandler{
		pool:     pool,
		cfg:      cfg,
		sessions: make(map[uuid.UUID]*webauthn.SessionData),
	}

	if cfg.WebAuthnEnabled {
		wa, err := webauthn.New(&webauthn.Config{
			RPDisplayName: cfg.WebAuthnRPDisplayName,
			RPID:          cfg.WebAuthnRPID,
			RPOrigins:     []string{cfg.WebAuthnRPOrigin},
			Timeouts: webauthn.TimeoutsConfig{
				Login:        webauthn.TimeoutConfig{Enforce: true, Timeout: 300000, TimeoutUVD: 300000},
				Registration: webauthn.TimeoutConfig{Enforce: true, Timeout: 300000, TimeoutUVD: 300000},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create WebAuthn instance: %w", err)
		}
		h.wa = wa
	}

	return h, nil
}

// Routes returns the chi router for WebAuthn endpoints.
func (h *WebAuthnHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/webauthn/register/start", h.RegisterStart)
	r.Post("/webauthn/register/finish", h.RegisterFinish)
	r.Post("/webauthn/authenticate/start", h.AuthenticateStart)
	r.Post("/webauthn/authenticate/finish", h.AuthenticateFinish)
	r.Get("/webauthn/credentials", h.ListCredentials)
	r.With(requireIdempotencyWA).
		Delete("/webauthn/credentials/{credentialId}", h.DisableCredential)
	return r
}

func requireIdempotencyWA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") == "" {
			writeErr(w, 400, "Idempotency-Key header required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─── Registration Start ───
// POST /api/auth/mfa/webauthn/register/start
func (h *WebAuthnHandler) RegisterStart(w http.ResponseWriter, r *http.Request) {
	if h.wa == nil {
		writeErr(w, 503, "WebAuthn is not enabled")
		return
	}

	ctx := r.Context()
	userIDStr := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userIDStr)

	var req struct {
		Label string `json:"label"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if len(req.Label) > 100 {
		req.Label = req.Label[:100]
	}

	// Load user
	waUser, err := h.loadUser(ctx, uid)
	if err != nil {
		writeErr(w, 500, "Failed to load user")
		return
	}

	options, session, err := h.wa.BeginRegistration(waUser)
	if err != nil {
		writeErr(w, 500, "Failed to begin registration")
		return
	}

	// Store session
	h.storeSession(uid, session)

	// Store label in session response for later use
	writeJSON(w, 200, map[string]any{
		"options": options.Response,
		"label":   req.Label,
	})
}

// ─── Registration Finish ───
// POST /api/auth/mfa/webauthn/register/finish
func (h *WebAuthnHandler) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	if h.wa == nil {
		writeErr(w, 503, "WebAuthn is not enabled")
		return
	}

	ctx := r.Context()
	userIDStr := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userIDStr)

	session := h.popSession(uid)
	if session == nil {
		writeErr(w, 400, "No active registration session. Start registration first.")
		return
	}

	// Parse the credential creation response from the browser
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		writeErr(w, 400, "Failed to parse credential response")
		return
	}

	waUser, err := h.loadUser(ctx, uid)
	if err != nil {
		writeErr(w, 500, "Failed to load user")
		return
	}

	credential, err := h.wa.CreateCredential(waUser, *session, parsedResponse)
	if err != nil {
		writeErr(w, 400, fmt.Sprintf("Credential verification failed: %v", err))
		return
	}

	// Extract label from query
	label := r.URL.Query().Get("label")
	if label == "" {
		label = "Security Key"
	}

	credIDHash := sha256Hex(credential.ID)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	credUUID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO user_webauthn_credentials
			(id, user_id, credential_id_hash, credential_id_bytes, public_key,
			 sign_count, device_type, backup_eligible, backup_state, label, aaguid,
			 transports, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false, false, $8, $9, $10, 'active', NOW())
	`, credUUID, uid, credIDHash, credential.ID, credential.PublicKey,
		uint32(credential.Authenticator.SignCount),
		credential.Authenticator.Attachment,
		label,
		credential.Authenticator.AAGUID,
		transportsToStrings(credential.Transport),
	)
	if err != nil {
		writeErr(w, 500, "Failed to store credential")
		return
	}

	// Audit — no credential_id, no public_key
	auditMeta, _ := json.Marshal(map[string]any{
		"credential_id": credUUID.String(),
		"label":         label,
		"aaguid":        credential.Authenticator.AAGUID,
	})
	audit.Write(ctx, tx, audit.Event{
		ActorID:    uid,
		ActorType:  "user",
		Action:     "mfa.webauthn.registered",
		EntityType: "webauthn_credential",
		EntityID:   credUUID,
		Summary:    fmt.Sprintf("WebAuthn credential registered: %s", label),
		NewValue:   auditMeta,
	})

	teamIDStr := ""
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.identity.mfa.webauthn.registered",
		AggregateType: "webauthn_credential",
		AggregateID:   credUUID.String(),
		Payload: json.RawMessage(fmt.Sprintf(
			`{"credential_id":"%s","user_id":"%s","label":"%s"}`,
			credUUID, userIDStr, label)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 201, map[string]any{
		"message":       "WebAuthn credential registered",
		"credential_id": credUUID,
		"label":         label,
	})
}

// ─── Authentication Start ───
// POST /api/auth/mfa/webauthn/authenticate/start
func (h *WebAuthnHandler) AuthenticateStart(w http.ResponseWriter, r *http.Request) {
	if h.wa == nil {
		writeErr(w, 503, "WebAuthn is not enabled")
		return
	}

	ctx := r.Context()
	userIDStr := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userIDStr)

	waUser, err := h.loadUser(ctx, uid)
	if err != nil {
		writeErr(w, 500, "Failed to load user")
		return
	}

	if len(waUser.credentials) == 0 {
		writeErr(w, 404, "No WebAuthn credentials registered")
		return
	}

	options, session, err := h.wa.BeginLogin(waUser)
	if err != nil {
		writeErr(w, 500, "Failed to begin authentication")
		return
	}

	h.storeSession(uid, session)

	writeJSON(w, 200, map[string]any{
		"options": options.Response,
	})
}

// ─── Authentication Finish ───
// POST /api/auth/mfa/webauthn/authenticate/finish
func (h *WebAuthnHandler) AuthenticateFinish(w http.ResponseWriter, r *http.Request) {
	if h.wa == nil {
		writeErr(w, 503, "WebAuthn is not enabled")
		return
	}

	ctx := r.Context()
	userIDStr := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userIDStr)

	session := h.popSession(uid)
	if session == nil {
		writeErr(w, 400, "No active authentication session. Start authentication first.")
		return
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		writeErr(w, 400, "Failed to parse authentication response")
		return
	}

	waUser, err := h.loadUser(ctx, uid)
	if err != nil {
		writeErr(w, 500, "Failed to load user")
		return
	}

	credential, err := h.wa.ValidateLogin(waUser, *session, parsedResponse)
	if err != nil {
		writeErr(w, 401, fmt.Sprintf("Authentication failed: %v", err))
		return
	}

	// Update sign count and last_used_at
	credIDHash := sha256Hex(credential.ID)
	h.pool.Exec(ctx, `
		UPDATE user_webauthn_credentials
		SET sign_count = $1, last_used_at = NOW()
		WHERE credential_id_hash = $2 AND user_id = $3
	`, uint32(credential.Authenticator.SignCount), credIDHash, uid)

	// Set recent_mfa_at on ALL active sessions (same as TOTP)
	_, err = h.pool.Exec(ctx, `UPDATE user_sessions SET recent_mfa_at=NOW() WHERE user_id=$1 AND revoked_at IS NULL`, uid)
	if err != nil {
		writeErr(w, 500, "Failed to update session")
		return
	}

	// Audit
	var credUUID uuid.UUID
		h.pool.QueryRow(ctx, `
			SELECT id FROM user_webauthn_credentials WHERE credential_id_hash=$1 AND user_id=$2
		`, credIDHash, uid).Scan(&credUUID)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	audit.Write(ctx, tx, audit.Event{
		ActorID:    uid,
		ActorType:  "user",
		Action:     "mfa.webauthn.verified",
		EntityType: "webauthn_credential",
		EntityID:   credUUID,
		Summary:    "WebAuthn verification successful",
	})

	teamIDStr := ""
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.identity.mfa.webauthn.verified",
		AggregateType: "webauthn_credential",
		AggregateID:   credUUID.String(),
		Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s"}`, userIDStr)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{
		"message":     "WebAuthn verified",
		"verified_at": time.Now(),
		"valid_until": time.Now().Add(mfaWindow),
	})
}

// ─── List Credentials ───
// GET /api/auth/mfa/webauthn/credentials
func (h *WebAuthnHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDStr := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userIDStr)

	rows, err := h.pool.Query(ctx, `
		SELECT id::text, label, aaguid, device_type, status, created_at, last_used_at
		FROM user_webauthn_credentials
		WHERE user_id=$1 ORDER BY created_at DESC
	`, uid)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer rows.Close()

	credentials := []map[string]any{}
	for rows.Next() {
		var id, label, aaguid, deviceType, status string
		var createdAt time.Time
		var lastUsedAt *time.Time
		rows.Scan(&id, &label, &aaguid, &deviceType, &status, &createdAt, &lastUsedAt)
		credentials = append(credentials, map[string]any{
			"id":          id,
			"label":       label,
			"aaguid":      aaguid,
			"device_type": deviceType,
			"status":      status,
			"created_at":  createdAt,
			"last_used_at": lastUsedAt,
		})
	}

	writeJSON(w, 200, credentials)
}

// ─── Disable Credential ───
// DELETE /api/auth/mfa/webauthn/credentials/{credentialId}
func (h *WebAuthnHandler) DisableCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDStr := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userIDStr)
	credID := chi.URLParam(r, "credentialId")
	credUUID, err := uuid.Parse(credID)
	if err != nil {
		writeErr(w, 400, "Invalid credential ID")
		return
	}

	// Require recent MFA (TOTP or WebAuthn both set recent_mfa_at)
	if !h.hasRecentMFA(ctx, uid) {
		writeErr(w, 403, "Recent MFA verification required to disable a security key")
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE user_webauthn_credentials SET status='disabled', disabled_at=NOW()
		WHERE id=$1 AND user_id=$2 AND status='active'
	`, credUUID, uid)
	if err != nil || tag.RowsAffected() == 0 {
		writeErr(w, 404, "Credential not found or not active")
		return
	}

	audit.Write(ctx, tx, audit.Event{
		ActorID:    uid,
		ActorType:  "user",
		Action:     "mfa.webauthn.disabled",
		EntityType: "webauthn_credential",
		EntityID:   credUUID,
		Summary:    "WebAuthn credential disabled",
	})

	teamIDStr := ""
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.identity.mfa.webauthn.disabled",
		AggregateType: "webauthn_credential",
		AggregateID:   credUUID.String(),
		Payload:       json.RawMessage(fmt.Sprintf(`{"credential_id":"%s","user_id":"%s"}`, credUUID, userIDStr)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{
		"message": "WebAuthn credential disabled",
	})
}

// ─── Helpers ───

// loadUser loads a webauthnUser with their existing credentials.
func (h *WebAuthnHandler) loadUser(ctx context.Context, uid uuid.UUID) (*webauthnUser, error) {
	var email string
	err := h.pool.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, uid).Scan(&email)
	if err != nil {
		return nil, err
	}

	// Load existing active credentials
	rows, err := h.pool.Query(ctx, `
		SELECT credential_id_bytes, public_key, sign_count
		FROM user_webauthn_credentials
		WHERE user_id=$1 AND status='active'
	`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []webauthn.Credential
	for rows.Next() {
		var credIDBytes, pubKey []byte
		var signCount int64
		rows.Scan(&credIDBytes, &pubKey, &signCount)
		creds = append(creds, webauthn.Credential{
			ID:              credIDBytes,
			PublicKey:       pubKey,
			Authenticator:   webauthn.Authenticator{SignCount: uint32(signCount)},
		})
	}

	return &webauthnUser{
		id:          uid,
		email:       email,
		credentials: creds,
	}, nil
}

// storeSession stores a WebAuthn session with automatic expiry cleanup.
func (h *WebAuthnHandler) storeSession(uid uuid.UUID, session *webauthn.SessionData) {
	h.sessionsMu.Lock()
	defer h.sessionsMu.Unlock()

	// Clean expired sessions
	now := time.Now()
	for id, s := range h.sessions {
		if s.Expires.Before(now) {
			delete(h.sessions, id)
		}
	}

	h.sessions[uid] = session
}

// popSession retrieves and removes a WebAuthn session.
func (h *WebAuthnHandler) popSession(uid uuid.UUID) *webauthn.SessionData {
	h.sessionsMu.Lock()
	defer h.sessionsMu.Unlock()

	session := h.sessions[uid]
	delete(h.sessions, uid)

	if session != nil && session.Expires.Before(time.Now()) {
		return nil
	}

	return session
}

// hasRecentMFA checks if the user has a recent MFA verification.
func (h *WebAuthnHandler) hasRecentMFA(ctx context.Context, userID uuid.UUID) bool {
	var recentMFA *time.Time
	h.pool.QueryRow(ctx, `
		SELECT MAX(recent_mfa_at) FROM user_sessions
		WHERE user_id=$1 AND revoked_at IS NULL
	`, userID).Scan(&recentMFA)
	if recentMFA == nil {
		return false
	}
	return time.Since(*recentMFA) < mfaWindow
}

// sha256Hex returns hex-encoded SHA-256 hash.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// transportsToStrings converts protocol.URIs to string slice.
func transportsToStrings(t []protocol.AuthenticatorTransport) []string {
	result := make([]string, len(t))
	for i, v := range t {
		result[i] = string(v)
	}
	return result
}

// IsWebAuthnEnabled returns whether WebAuthn is available.
func (h *WebAuthnHandler) IsWebAuthnEnabled() bool {
	return h.wa != nil
}

// ensure strings import
var _ = strings.TrimSpace
