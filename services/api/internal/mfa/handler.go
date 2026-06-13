package mfa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	mfaWindow          = 5 * time.Minute  // recent MFA validity
	challengeTTL       = 5 * time.Minute  // challenge expiry
	maxFailedAttempts  = 5                // before lockout
	lockoutDuration    = 15 * time.Minute // lockout period
	recoveryCodeCount  = 10
)

type Handler struct {
	pool   *pgxpool.Pool
	crypto *Crypto
	cfg    *config.Config
}

func NewHandler(pool *pgxpool.Pool, cfg *config.Config) (*Handler, error) {
	c, err := NewCrypto(cfg.MFAKey)
	if err != nil {
		return nil, err
	}
	return &Handler{pool: pool, crypto: c, cfg: cfg}, nil
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/totp/enroll", h.EnrollTOTP)
	r.Post("/totp/verify-enrollment", h.VerifyEnrollment)
	r.Post("/challenge", h.CreateChallenge)
	r.Post("/verify", h.VerifyChallenge)
	r.Post("/recovery-codes/regenerate", h.RegenerateRecoveryCodes)
	r.Delete("/factors/{factorId}", h.DisableFactor)
	r.Get("/factors", h.ListFactors)
	r.Get("/status", h.MFAStatus)
	return r
}

// ─── Enroll TOTP ───
// POST /api/auth/mfa/totp/enroll
// Starts enrollment. Returns provisioning URI + secret (base32) once.
// Secret is stored encrypted; raw secret never returned again.
func (h *Handler) EnrollTOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)

	// Check if user already has an active factor
	var existingCount int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_mfa_factors WHERE user_id=$1 AND status='active'`, userID).Scan(&existingCount)
	if existingCount > 0 {
		writeErr(w, 409, "MFA already enrolled. Disable existing factor first.")
		return
	}

	// Remove any pending enrollment
	h.pool.Exec(ctx, `DELETE FROM user_mfa_factors WHERE user_id=$1 AND status='pending'`, userID)

	// Generate TOTP secret
	secretBytes, secretB32, err := GenerateTOTPSecret()
	if err != nil {
		writeErr(w, 500, "Failed to generate secret")
		return
	}

	// Encrypt secret
	encrypted, err := h.crypto.Encrypt(secretBytes)
	if err != nil {
		writeErr(w, 500, "Failed to encrypt secret")
		return
	}

	uid, _ := uuid.Parse(userID)
	factorID := uuid.New()

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO user_mfa_factors (id, user_id, factor_type, secret, status)
		VALUES ($1, $2, 'totp', $3, 'pending')
	`, factorID, uid, encrypted)
	if err != nil {
		writeErr(w, 500, "Failed to create factor")
		return
	}

	// Audit — no raw secret
	audit.Write(ctx, tx, audit.Event{
		ActorID:    uid,
		ActorType:  "user",
		Action:     "mfa.enrollment.started",
		EntityType: "mfa_factor",
		EntityID:   factorID,
		Summary:    "MFA TOTP enrollment started",
	})

	outbox.Write(ctx, tx, nil, outbox.Event{
		EventType:     "clarity.v1.identity.mfa.enrollment.started",
		AggregateType: "mfa_factor",
		AggregateID:   factorID.String(),
		Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s","factor_id":"%s"}`, userID, factorID)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	// Return provisioning URI + raw secret ONCE
	issuer := "ClarityIT"
	var userName string
	h.pool.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, uid).Scan(&userName)
	uri := ProvisioningURI(userName, issuer, secretB32)

	writeJSON(w, 201, map[string]any{
		"factor_id":         factorID,
		"provisioning_uri":  uri,
		"secret":            secretB32,
		"_notice":           "Store the provisioning URI securely. The raw secret will not be shown again.",
		"algorithm":         "SHA1",
		"digits":            6,
		"period":            30,
	})
}

// ─── Verify Enrollment ───
// POST /api/auth/mfa/totp/verify-enrollment
// Verifies the first TOTP code from the authenticator app.
// Activates the factor and generates recovery codes.
func (h *Handler) VerifyEnrollment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid request body")
		return
	}
	if len(req.Code) != 6 {
		writeErr(w, 400, "Code must be 6 digits")
		return
	}

	uid, _ := uuid.Parse(userID)

	// Get pending factor
	var factorID uuid.UUID
	var encryptedSecret []byte
	err := h.pool.QueryRow(ctx, `
		SELECT id, secret FROM user_mfa_factors
		WHERE user_id=$1 AND status='pending'
		ORDER BY created_at DESC LIMIT 1
	`, uid).Scan(&factorID, &encryptedSecret)
	if err != nil {
		writeErr(w, 404, "No pending enrollment found")
		return
	}

	// Decrypt secret
	secret, err := h.crypto.Decrypt(encryptedSecret)
	if err != nil {
		writeErr(w, 500, "Failed to decrypt secret")
		return
	}

	// Validate TOTP
	if !ValidateTOTP(secret, req.Code, time.Now()) {
		writeErr(w, 401, "Invalid TOTP code")
		return
	}

	// Generate recovery codes
	rawCodes, codeHashes := GenerateRecoveryCodes(recoveryCodeCount, h.cfg.HMACKey)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	// Activate factor
	_, err = tx.Exec(ctx, `
		UPDATE user_mfa_factors SET status='active', verified_at=NOW(), failed_attempts=0
		WHERE id=$1
	`, factorID)
	if err != nil {
		writeErr(w, 500, "Failed to activate factor")
		return
	}

	// Store recovery code hashes
	for _, hash := range codeHashes {
		tx.Exec(ctx, `INSERT INTO mfa_recovery_codes (user_id, code_hash) VALUES ($1, $2)`, uid, hash)
	}

	// Audit
	audit.Write(ctx, tx, audit.Event{
		ActorID:    uid,
		ActorType:  "user",
		Action:     "mfa.enrollment.completed",
		EntityType: "mfa_factor",
		EntityID:   factorID,
		Summary:    "MFA TOTP enrollment completed",
	})

	outbox.Write(ctx, tx, nil, outbox.Event{
		EventType:     "clarity.v1.identity.mfa.enrollment.completed",
		AggregateType: "mfa_factor",
		AggregateID:   factorID.String(),
		Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s","factor_id":"%s"}`, userID, factorID)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{
		"message":        "MFA enrollment complete",
		"factor_id":      factorID,
		"recovery_codes": rawCodes,
		"_notice":        "Save recovery codes securely. They will not be shown again.",
	})
}

// ─── Create Challenge ───
// POST /api/auth/mfa/challenge
// Creates a short-lived MFA challenge for the current user.
func (h *Handler) CreateChallenge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userID)

	// Get active factor
	var factorID uuid.UUID
	var failedAttempts int
	var lockedUntil *time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, failed_attempts, locked_until FROM user_mfa_factors
		WHERE user_id=$1 AND status='active'
	`, uid).Scan(&factorID, &failedAttempts, &lockedUntil)
	if err != nil {
		writeErr(w, 404, "No active MFA factor. Enroll first.")
		return
	}

	// Check lockout
	if lockedUntil != nil && lockedUntil.After(time.Now()) {
		writeErr(w, 429, "Too many failed attempts. Try again later.")
		return
	}

	// Create challenge
	challengeID := uuid.New()
	challengeNonce := uuid.New().String()
	expiresAt := time.Now().Add(challengeTTL)

	_, err = h.pool.Exec(ctx, `
		INSERT INTO mfa_challenges (id, user_id, factor_id, challenge, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, challengeID, uid, factorID, challengeNonce, expiresAt)
	if err != nil {
		writeErr(w, 500, "Failed to create challenge")
		return
	}

	writeJSON(w, 200, map[string]any{
		"challenge_id": challengeID,
		"expires_at":   expiresAt,
	})
}

// ─── Verify Challenge ───
// POST /api/auth/mfa/verify
// Verifies a TOTP code or recovery code against the challenge.
// Sets recent_mfa_at on the user's active session.
func (h *Handler) VerifyChallenge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userID)

	var req struct {
		ChallengeID   string `json:"challenge_id"`
		Code          string `json:"code"`
		RecoveryCode  string `json:"recovery_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid request body")
		return
	}

	// Validate challenge
	var factorID uuid.UUID
	var expiresAt time.Time
	var verified bool
	err := h.pool.QueryRow(ctx, `
		SELECT factor_id, expires_at, verified FROM mfa_challenges
		WHERE id=$1 AND user_id=$2
	`, req.ChallengeID, uid).Scan(&factorID, &expiresAt, &verified)
	if err != nil {
		writeErr(w, 404, "Challenge not found")
		return
	}
	if verified {
		writeErr(w, 409, "Challenge already used")
		return
	}
	if time.Now().After(expiresAt) {
		writeErr(w, 401, "Challenge expired")
		return
	}

	authenticated := false

	if req.RecoveryCode != "" {
		// Recovery code authentication
		var codeID uuid.UUID
		err := h.pool.QueryRow(ctx, `
			SELECT id FROM mfa_recovery_codes
			WHERE user_id=$1 AND code_hash=$2 AND used_at IS NULL
		`, uid, hashRecoveryCode(h.cfg.HMACKey, req.RecoveryCode)).Scan(&codeID)
		if err == nil {
			// Mark recovery code as used
			h.pool.Exec(ctx, `UPDATE mfa_recovery_codes SET used_at=NOW() WHERE id=$1`, codeID)
			authenticated = true
		}
	} else if req.Code != "" {
		// TOTP code authentication
		var encryptedSecret []byte
		var failedAttempts int
		var lockedUntil *time.Time
		h.pool.QueryRow(ctx, `SELECT secret, failed_attempts, locked_until FROM user_mfa_factors WHERE id=$1`, factorID).
			Scan(&encryptedSecret, &failedAttempts, &lockedUntil)

		if lockedUntil != nil && lockedUntil.After(time.Now()) {
			writeErr(w, 429, "Too many failed attempts. Try again later.")
			return
		}

		secret, err := h.crypto.Decrypt(encryptedSecret)
		if err == nil && ValidateTOTP(secret, req.Code, time.Now()) {
			authenticated = true
			// Reset failed attempts
			h.pool.Exec(ctx, `UPDATE user_mfa_factors SET failed_attempts=0, locked_until=NULL WHERE id=$1`, factorID)
		} else {
			// Increment failed attempts
			newAttempts := failedAttempts + 1
			if newAttempts >= maxFailedAttempts {
				h.pool.Exec(ctx, `UPDATE user_mfa_factors SET failed_attempts=$1, locked_until=NOW() + $2::interval WHERE id=$3`,
					newAttempts, fmt.Sprintf("%d minutes", int(lockoutDuration.Minutes())), factorID)
			} else {
				h.pool.Exec(ctx, `UPDATE user_mfa_factors SET failed_attempts=$1 WHERE id=$2`, newAttempts, factorID)
			}
		}
	}

	if !authenticated {
		writeErr(w, 401, "Invalid MFA code")
		return
	}

	// Mark challenge as verified
	h.pool.Exec(ctx, `UPDATE mfa_challenges SET verified=true WHERE id=$1`, req.ChallengeID)

	// Set recent_mfa_at on ALL active sessions for this user
	_, err = h.pool.Exec(ctx, `UPDATE user_sessions SET recent_mfa_at=NOW() WHERE user_id=$1 AND revoked_at IS NULL`, uid)
	if err != nil {
		writeErr(w, 500, "Failed to update session")
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	audit.Write(ctx, tx, audit.Event{
		ActorID:    uid,
		ActorType:  "user",
		Action:     "mfa.verified",
		EntityType: "mfa_factor",
		EntityID:   factorID,
		Summary:    "MFA verification successful",
	})

	outbox.Write(ctx, tx, nil, outbox.Event{
		EventType:     "clarity.v1.identity.mfa.verified",
		AggregateType: "mfa_factor",
		AggregateID:   factorID.String(),
		Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s"}`, userID)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{
		"message":       "MFA verified",
		"verified_at":   time.Now(),
		"valid_until":   time.Now().Add(mfaWindow),
	})
}

// ─── Regenerate Recovery Codes ───
// POST /api/auth/mfa/recovery-codes/regenerate
// Requires recent MFA verification.
func (h *Handler) RegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userID)

	// Require recent MFA
	if !h.hasRecentMFA(ctx, uid) {
		writeErr(w, 403, "Recent MFA verification required")
		return
	}

	// Invalidate old recovery codes
	h.pool.Exec(ctx, `UPDATE mfa_recovery_codes SET used_at=NOW() WHERE user_id=$1 AND used_at IS NULL`, uid)

	// Generate new codes
	rawCodes, codeHashes := GenerateRecoveryCodes(recoveryCodeCount, h.cfg.HMACKey)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	for _, hash := range codeHashes {
		tx.Exec(ctx, `INSERT INTO mfa_recovery_codes (user_id, code_hash) VALUES ($1, $2)`, uid, hash)
	}

	audit.Write(ctx, tx, audit.Event{
		ActorID:    uid,
		ActorType:  "user",
		Action:     "mfa.recovery_codes.regenerated",
		EntityType: "mfa_factor",
		Summary:    "MFA recovery codes regenerated",
	})

	outbox.Write(ctx, tx, nil, outbox.Event{
		EventType:     "clarity.v1.identity.mfa.recovery_codes.regenerated",
		AggregateType: "mfa_factor",
		AggregateID:   userID,
		Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s"}`, userID)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{
		"recovery_codes": rawCodes,
		"_notice":        "Save recovery codes securely. They will not be shown again.",
	})
}

// ─── Disable Factor ───
// DELETE /api/auth/mfa/factors/{factorId}
// Requires recent MFA verification.
func (h *Handler) DisableFactor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userID)
	factorID := chi.URLParam(r, "factorId")
	factorUUID, err := uuid.Parse(factorID)
	if err != nil {
		writeErr(w, 400, "Invalid factor ID")
		return
	}

	// Require recent MFA
	if !h.hasRecentMFA(ctx, uid) {
		writeErr(w, 403, "Recent MFA verification required to disable MFA")
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE user_mfa_factors SET status='disabled', disabled_at=NOW()
		WHERE id=$1 AND user_id=$2 AND status='active'
	`, factorUUID, uid)
	if err != nil || tag.RowsAffected() == 0 {
		writeErr(w, 404, "Factor not found or not active")
		return
	}

	// Invalidate all recovery codes
	tx.Exec(ctx, `UPDATE mfa_recovery_codes SET used_at=NOW() WHERE user_id=$1 AND used_at IS NULL`, uid)

	// Clear recent_mfa_at on all sessions
	tx.Exec(ctx, `UPDATE user_sessions SET recent_mfa_at=NULL WHERE user_id=$1`, uid)

	audit.Write(ctx, tx, audit.Event{
		ActorID:    uid,
		ActorType:  "user",
		Action:     "mfa.factor.disabled",
		EntityType: "mfa_factor",
		EntityID:   factorUUID,
		Summary:    "MFA factor disabled",
	})

	outbox.Write(ctx, tx, nil, outbox.Event{
		EventType:     "clarity.v1.identity.mfa.factor.disabled",
		AggregateType: "mfa_factor",
		AggregateID:   factorID,
		Payload:       json.RawMessage(fmt.Sprintf(`{"user_id":"%s","factor_id":"%s"}`, userID, factorID)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{
		"message": "MFA factor disabled",
	})
}

// ─── List Factors ───
// GET /api/auth/mfa/factors
// Returns the user's MFA factors (without secrets).
func (h *Handler) ListFactors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userID)

	rows, err := h.pool.Query(ctx, `
		SELECT id::text, factor_type, status, created_at, verified_at, disabled_at
		FROM user_mfa_factors WHERE user_id=$1 ORDER BY created_at DESC
	`, uid)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer rows.Close()

	factors := []map[string]any{}
	for rows.Next() {
		var id, factorType, status string
		var createdAt, verifiedAt, disabledAt *time.Time
		rows.Scan(&id, &factorType, &status, &createdAt, &verifiedAt, &disabledAt)
		factors = append(factors, map[string]any{
			"id":          id,
			"factor_type": factorType,
			"status":      status,
			"created_at":  createdAt,
			"verified_at": verifiedAt,
			"disabled_at": disabledAt,
		})
	}

	writeJSON(w, 200, factors)
}

// ─── MFA Status ───
// GET /api/auth/mfa/status
// Returns the user's MFA enrollment status and recent_mfa_at.
func (h *Handler) MFAStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)
	uid, _ := uuid.Parse(userID)

	var activeFactor int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_mfa_factors WHERE user_id=$1 AND status='active'`, uid).Scan(&activeFactor)

	var recentMFA *time.Time
	h.pool.QueryRow(ctx, `
		SELECT MAX(recent_mfa_at) FROM user_sessions
		WHERE user_id=$1 AND revoked_at IS NULL AND recent_mfa_at IS NOT NULL
	`, uid).Scan(&recentMFA)

	status := map[string]any{
		"enrolled": activeFactor > 0,
	}
	if recentMFA != nil {
		status["recent_mfa_at"] = recentMFA
		status["mfa_valid"] = time.Since(*recentMFA) < mfaWindow
	} else {
		status["recent_mfa_at"] = nil
		status["mfa_valid"] = false
	}

	writeJSON(w, 200, status)
}

// ─── Helpers ───

// hasRecentMFA checks if the user has a recent MFA verification on any active session.
func (h *Handler) hasRecentMFA(ctx context.Context, userID uuid.UUID) bool {
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

// RequireRecentMFA is a middleware that checks for recent MFA on the current session.
// Can be used to gate high-risk actions.
func RequireRecentMFA(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID, ok := ctx.Value("user_id").(string)
			if !ok {
				writeErr(w, 401, "Unauthorized")
				return
			}

			uid, _ := uuid.Parse(userID)
			var recentMFA *time.Time
			pool.QueryRow(ctx, `
				SELECT MAX(recent_mfa_at) FROM user_sessions
				WHERE user_id=$1 AND revoked_at IS NULL
			`, uid).Scan(&recentMFA)

			if recentMFA == nil || time.Since(*recentMFA) > mfaWindow {
				writeErr(w, 403, "Recent MFA verification required for this action")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}

// unused but needed for pgx import
var _ = pgx.Tx(nil)
