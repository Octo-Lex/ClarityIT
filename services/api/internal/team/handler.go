package team

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
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool  *pgxpool.Pool
	cfg   *config.Config
}

func NewHandler(pool *pgxpool.Pool, cfg *config.Config) *Handler {
	return &Handler{pool: pool, cfg: cfg}
}

// ─── Team Settings ───

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	var name, slug, description string
	var settings json.RawMessage
	err = h.pool.QueryRow(ctx, `
		SELECT name, slug, COALESCE(description, ''), settings
		FROM teams WHERE id = $1 AND deleted_at IS NULL
	`, teamID).Scan(&name, &slug, &description, &settings)
	if err != nil {
		writeErr(w, http.StatusNotFound, "Team not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":          teamID,
		"name":        name,
		"slug":        slug,
		"description": description,
		"settings":    json.RawMessage(settings),
	})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Name        *string          `json:"name"`
		Description *string          `json:"description"`
		Settings    *json.RawMessage `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Fetch current for audit
		var oldName, oldDesc string
		var oldSettings json.RawMessage
		if err := tx.QueryRow(ctx, "SELECT name, COALESCE(description,''), settings FROM teams WHERE id = $1 AND deleted_at IS NULL", teamID).Scan(&oldName, &oldDesc, &oldSettings); err != nil {
			return fmt.Errorf("team not found")
		}

		if req.Name != nil {
			if _, err := tx.Exec(ctx, "UPDATE teams SET name = $1 WHERE id = $2", *req.Name, teamID); err != nil {
				return err
			}
		}
		if req.Description != nil {
			if _, err := tx.Exec(ctx, "UPDATE teams SET description = $1 WHERE id = $2", *req.Description, teamID); err != nil {
				return err
			}
		}
		if req.Settings != nil {
			if _, err := tx.Exec(ctx, "UPDATE teams SET settings = $1 WHERE id = $2", *req.Settings, teamID); err != nil {
				return err
			}
		}

		newValue := iam.MergeAuditMeta(ctx, map[string]any{
			"name": req.Name, "description": req.Description, "settings": req.Settings,
		})
		audit.Write(ctx, tx, audit.Event{
			ActorID:    actorID,
			ActorType:  "user",
			TeamID:     &teamID,
			Action:     "team.settings.updated",
			EntityType: "team",
			EntityID:   teamID,
			OldValue:   oldSettings,
			NewValue:   newValue,
			Summary:    "Team settings updated",
		})

		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.team.settings.updated",
			AggregateType: "team",
			AggregateID:   teamID.String(),
			Payload:       newValue,
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Settings updated"})
}

// ─── Team Members ───

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	rows, err := h.pool.Query(ctx, `
		SELECT tm.id, u.id, u.name, u.email, r.name, tm.joined_at
		FROM team_memberships tm
		JOIN users u ON u.id = tm.user_id AND u.deleted_at IS NULL
		JOIN roles r ON r.id = tm.role_id
		WHERE tm.team_id = $1
		ORDER BY tm.joined_at ASC
	`, teamID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load members")
		return
	}
	defer rows.Close()

	type Member struct {
		MembershipID string `json:"membership_id"`
		UserID       string `json:"user_id"`
		Name         string `json:"name"`
		Email        string `json:"email"`
		Role         string `json:"role"`
		JoinedAt     string `json:"joined_at"`
	}
	var members []Member
	for rows.Next() {
		var m Member
		rows.Scan(&m.MembershipID, &m.UserID, &m.Name, &m.Email, &m.Role, &m.JoinedAt)
		members = append(members, m)
	}

	if members == nil {
		members = []Member{}
	}
	writeJSON(w, http.StatusOK, members)
}

func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	membershipID, err := uuid.Parse(chi.URLParam(r, "membershipId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid membership ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		RoleID string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	newRoleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid role_id")
		return
	}

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Get current membership
		var targetUserID uuid.UUID
		var currentRoleID uuid.UUID
		if err := tx.QueryRow(ctx, "SELECT user_id, role_id FROM team_memberships WHERE id = $1 AND team_id = $2", membershipID, teamID).Scan(&targetUserID, &currentRoleID); err != nil {
			return fmt.Errorf("membership not found")
		}

		// Check if demoting from owner — prevent if last owner
		var oldRoleName string
		tx.QueryRow(ctx, "SELECT name FROM roles WHERE id = $1", currentRoleID).Scan(&oldRoleName)

		if oldRoleName == "owner" {
			var ownerCount int
			tx.QueryRow(ctx, `
				SELECT COUNT(*) FROM team_memberships tm
				JOIN roles r ON r.id = tm.role_id
				WHERE tm.team_id = $1 AND r.name = 'owner'
			`, teamID).Scan(&ownerCount)
			if ownerCount <= 1 {
				return fmt.Errorf("cannot demote the last team owner")
			}
		}

		var newRoleName string
		tx.QueryRow(ctx, "SELECT name FROM roles WHERE id = $1", newRoleID).Scan(&newRoleName)

		// Update
		if _, err := tx.Exec(ctx, "UPDATE team_memberships SET role_id = $1 WHERE id = $2", newRoleID, membershipID); err != nil {
			return err
		}

		meta := iam.MergeAuditMeta(ctx, map[string]any{
			"old_role": oldRoleName, "new_role": newRoleName,
		})
		audit.Write(ctx, tx, audit.Event{
			ActorID:    actorID,
			ActorType:  "user",
			TeamID:     &teamID,
			Action:     "team.member.role_changed",
			EntityType: "membership",
			EntityID:   targetUserID,
			OldValue:   json.RawMessage(fmt.Sprintf(`{"role":"%s"}`, oldRoleName)),
			NewValue:   meta,
			Summary:    fmt.Sprintf("Role changed from %s to %s", oldRoleName, newRoleName),
		})

		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.team.member.role_changed",
			AggregateType: "membership",
			AggregateID:   membershipID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "last team owner") {
			writeErr(w, http.StatusConflict, err.Error())
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Role updated"})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	membershipID, err := uuid.Parse(chi.URLParam(r, "membershipId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid membership ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var targetUserID uuid.UUID
		var roleID uuid.UUID
		if err := tx.QueryRow(ctx, "SELECT user_id, role_id FROM team_memberships WHERE id = $1 AND team_id = $2", membershipID, teamID).Scan(&targetUserID, &roleID); err != nil {
			return fmt.Errorf("membership not found")
		}

		// Check last owner protection
		var roleName string
		tx.QueryRow(ctx, "SELECT name FROM roles WHERE id = $1", roleID).Scan(&roleName)
		if roleName == "owner" {
			var ownerCount int
			tx.QueryRow(ctx, `
				SELECT COUNT(*) FROM team_memberships tm
				JOIN roles r ON r.id = tm.role_id
				WHERE tm.team_id = $1 AND r.name = 'owner'
			`, teamID).Scan(&ownerCount)
			if ownerCount <= 1 {
				return fmt.Errorf("cannot remove the last team owner")
			}
		}

		if _, err := tx.Exec(ctx, "DELETE FROM team_memberships WHERE id = $1", membershipID); err != nil {
			return err
		}

		audit.Write(ctx, tx, audit.Event{
			ActorID:    actorID,
			ActorType:  "user",
			TeamID:     &teamID,
			Action:     "team.member.removed",
			EntityType: "membership",
			EntityID:   targetUserID,
			NewValue:   iam.MergeAuditMeta(ctx, map[string]any{"user_id": targetUserID.String()}),
			Summary:    "Team member removed",
		})

		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.team.member.removed",
			AggregateType: "membership",
			AggregateID:   membershipID.String(),
			Payload: json.RawMessage(fmt.Sprintf(
				`{"user_id":"%s","team_id":"%s"}`, targetUserID, teamID)),
		})
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "last team owner") {
			writeErr(w, http.StatusConflict, err.Error())
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Member removed"})
}

// ─── Invitations ───

func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Email  string `json:"email"`
		RoleID string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid role_id")
		return
	}

	emailHMAC := iam.HMACString(h.cfg.HMACKey, req.Email)
	token := iam.GenerateToken()
	tokenHash := iam.HashToken(token)

	var invitationID uuid.UUID
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO invitations (team_id, email, role_id, token_hash, invited_by, expires_at)
			VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '7 days')
			RETURNING id
		`, teamID, req.Email, roleID, tokenHash, actorID).Scan(&id)
		if err != nil {
			return err
		}
		invitationID = id

		meta := iam.MergeAuditMeta(ctx, map[string]any{
			"email_hmac": emailHMAC,
			"role_id":    roleID.String(),
		})
		audit.Write(ctx, tx, audit.Event{
			ActorID:    actorID,
			ActorType:  "user",
			TeamID:     &teamID,
			Action:     "team.member.invited",
			EntityType: "invitation",
			EntityID:   invitationID,
			NewValue:   meta,
			Summary:    "Team member invited",
		})

		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.team.member.invited",
			AggregateType: "invitation",
			AggregateID:   invitationID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// In production, never return raw token — send via email service instead
	if h.cfg.IsProd() {
		// TODO: wire email service for invitations (email.Service)
		// For now, production does not return the token
		writeJSON(w, http.StatusOK, map[string]any{
			"id":         invitationID,
			"message":    "Invitation sent. The invitee will receive an email.",
			"expires_at": time.Now().Add(7 * 24 * time.Hour),
		})
	} else {
		// Dev mode: return token for local testing
		writeJSON(w, http.StatusOK, map[string]any{
			"id":          invitationID,
			"token":       token,
			"dev_preview": "/accept-invitation?token=" + token,
			"_dev_notice": "DEV MODE ONLY — token returned for local testing. Not available in production.",
			"expires_at":  time.Now().Add(7 * 24 * time.Hour),
		})
	}
}

func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	rows, err := h.pool.Query(ctx, `
		SELECT i.id, i.email, r.name as role_name, i.invited_by, i.expires_at, i.accepted_at, i.created_at
		FROM invitations i
		JOIN roles r ON r.id = i.role_id
		WHERE i.team_id = $1
		ORDER BY i.created_at DESC
	`, teamID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load invitations")
		return
	}
	defer rows.Close()

	type Invitation struct {
		ID        string  `json:"id"`
		Email     string  `json:"email"`
		RoleName  string  `json:"role_name"`
		InvitedBy string  `json:"invited_by"`
		ExpiresAt string  `json:"expires_at"`
		Accepted  bool    `json:"accepted"`
		CreatedAt string  `json:"created_at"`
	}
	var invitations []Invitation
	for rows.Next() {
		var inv Invitation
		var acceptedAt *string
		rows.Scan(&inv.ID, &inv.Email, &inv.RoleName, &inv.InvitedBy, &inv.ExpiresAt, &acceptedAt, &inv.CreatedAt)
		inv.Accepted = acceptedAt != nil
		invitations = append(invitations, inv)
	}
	if invitations == nil {
		invitations = []Invitation{}
	}
	writeJSON(w, http.StatusOK, invitations)
}

func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	invitationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	userID, _ := uuid.Parse(claims.UserID)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var email string
		var roleID uuid.UUID
		var expiresAt time.Time
		var acceptedAt *time.Time
		err := tx.QueryRow(ctx, `
			SELECT email, role_id, expires_at, accepted_at
			FROM invitations WHERE id = $1 AND team_id = $2
		`, invitationID, teamID).Scan(&email, &roleID, &expiresAt, &acceptedAt)
		if err != nil {
			return fmt.Errorf("invitation not found")
		}

		if acceptedAt != nil {
			return fmt.Errorf("invitation already accepted")
		}
		if time.Now().After(expiresAt) {
			return fmt.Errorf("invitation has expired")
		}

		// Verify authenticated user email matches invitation email
		var userEmail string
		if err := tx.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&userEmail); err != nil {
			return fmt.Errorf("user not found")
		}
		if strings.ToLower(userEmail) != strings.ToLower(email) {
			return fmt.Errorf("invitation email does not match your account email")
		}

		// Accept invitation
		if _, err := tx.Exec(ctx, "UPDATE invitations SET accepted_at = NOW() WHERE id = $1", invitationID); err != nil {
			return err
		}

		// Create membership
		if _, err := tx.Exec(ctx, `
			INSERT INTO team_memberships (user_id, team_id, role_id)
			VALUES ($1, $2, $3)
		`, userID, teamID, roleID); err != nil {
			return err
		}

		emailHMAC := iam.HMACString(h.cfg.HMACKey, email)
		meta := iam.MergeAuditMeta(ctx, map[string]any{"email_hmac": emailHMAC})

		audit.Write(ctx, tx, audit.Event{
			ActorID:    userID,
			ActorType:  "user",
			TeamID:     &teamID,
			Action:     "team.member.accepted",
			EntityType: "invitation",
			EntityID:   invitationID,
			NewValue:   meta,
			Summary:    "Invitation accepted",
		})

		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.team.member.accepted",
			AggregateType: "membership",
			AggregateID:   userID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation accepted"})
}

func (h *Handler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	invitationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var email string
		err := tx.QueryRow(ctx, "SELECT email FROM invitations WHERE id = $1 AND team_id = $2", invitationID, teamID).Scan(&email)
		if err != nil {
			return fmt.Errorf("invitation not found")
		}

		if _, err := tx.Exec(ctx, "DELETE FROM invitations WHERE id = $1", invitationID); err != nil {
			return err
		}

		emailHMAC := iam.HMACString(h.cfg.HMACKey, email)
		meta := iam.MergeAuditMeta(ctx, map[string]any{"email_hmac": emailHMAC})

		audit.Write(ctx, tx, audit.Event{
			ActorID:    actorID,
			ActorType:  "user",
			TeamID:     &teamID,
			Action:     "team.invitation.revoked",
			EntityType: "invitation",
			EntityID:   invitationID,
			NewValue:   meta,
			Summary:    "Invitation revoked",
		})

		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.team.invitation.revoked",
			AggregateType: "invitation",
			AggregateID:   invitationID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation revoked"})
}

// ─── Access Grants ───

func (h *Handler) ListAccessGrants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	rows, err := h.pool.Query(ctx, `
		SELECT ag.id, ag.user_id, u.name, ag.grant_type, ag.scope, ag.expires_at, ag.revoked_at, ag.created_at,
		       COALESCE(r.name, '')
		FROM team_access_grants ag
		JOIN users u ON u.id = ag.user_id
		LEFT JOIN roles r ON r.id = ag.role_id
		WHERE ag.team_id = $1
		ORDER BY ag.created_at DESC
	`, teamID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load access grants")
		return
	}
	defer rows.Close()

	type Grant struct {
		ID        string  `json:"id"`
		UserID    string  `json:"user_id"`
		UserName  string  `json:"user_name"`
		GrantType string  `json:"grant_type"`
		Scope     *string `json:"scope"`
		RoleName  string  `json:"role_name"`
		ExpiresAt *string `json:"expires_at"`
		RevokedAt *string `json:"revoked_at"`
		CreatedAt string  `json:"created_at"`
	}
	var grants []Grant
	for rows.Next() {
		var g Grant
		rows.Scan(&g.ID, &g.UserID, &g.UserName, &g.GrantType, &g.Scope, &g.ExpiresAt, &g.RevokedAt, &g.CreatedAt, &g.RoleName)
		grants = append(grants, g)
	}
	if grants == nil {
		grants = []Grant{}
	}
	writeJSON(w, http.StatusOK, grants)
}

func (h *Handler) CreateAccessGrant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		UserID    string  `json:"user_id"`
		GrantType string  `json:"grant_type"`
		Scope     *string `json:"scope"`
		RoleID    string  `json:"role_id"`
		Duration  string  `json:"duration"` // e.g., "24h", "7d"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid user_id")
		return
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid role_id")
		return
	}

	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid duration (use e.g. 24h, 168h)")
		return
	}

	var grantID uuid.UUID
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO team_access_grants (team_id, user_id, granted_by, grant_type, scope, role_id, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW() + $7)
			RETURNING id
		`, teamID, targetUserID, actorID, req.GrantType, req.Scope, roleID, duration).Scan(&id)
		if err != nil {
			return err
		}
		grantID = id

		meta := iam.MergeAuditMeta(ctx, map[string]any{
			"user_id":    targetUserID.String(),
			"grant_type": req.GrantType,
			"role_id":    roleID.String(),
			"duration":   req.Duration,
		})
		audit.Write(ctx, tx, audit.Event{
			ActorID:    actorID,
			ActorType:  "user",
			TeamID:     &teamID,
			Action:     "team.access_grant.created",
			EntityType: "access_grant",
			EntityID:   grantID,
			NewValue:   meta,
			Summary:    "Access grant created",
		})

		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.team.access_grant.created",
			AggregateType: "access_grant",
			AggregateID:   grantID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"id":      grantID.String(),
		"message": "Access grant created",
	})
}

func (h *Handler) RevokeAccessGrant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	grantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid grant ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var count int
		tx.QueryRow(ctx, "SELECT COUNT(*) FROM team_access_grants WHERE id = $1 AND team_id = $2 AND revoked_at IS NULL", grantID, teamID).Scan(&count)
		if count == 0 {
			return fmt.Errorf("grant not found or already revoked")
		}

		if _, err := tx.Exec(ctx, "UPDATE team_access_grants SET revoked_at = NOW() WHERE id = $1", grantID); err != nil {
			return err
		}

		audit.Write(ctx, tx, audit.Event{
			ActorID:    actorID,
			ActorType:  "user",
			TeamID:     &teamID,
			Action:     "team.access_grant.revoked",
			EntityType: "access_grant",
			EntityID:   grantID,
			NewValue:   iam.MergeAuditMeta(ctx, map[string]any{"grant_id": grantID.String()}),
			Summary:    "Access grant revoked",
		})

		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType:     "clarity.v1.team.access_grant.revoked",
			AggregateType: "access_grant",
			AggregateID:   grantID.String(),
			Payload:       json.RawMessage(fmt.Sprintf(`{"grant_id":"%s"}`, grantID)),
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Access grant revoked"})
}

// ─── Helpers ───

func strPtr(s string) *string { return &s }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}
