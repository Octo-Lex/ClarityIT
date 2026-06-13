package admin

import (
	"encoding/json"
	"net/http"
)

// SetupStatus returns a checklist of admin setup items.
// Shows status only — never raw config values or secrets.
func (h *Handler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status := map[string]any{}

	// Bootstrap
	var bootstrapped bool
	h.pool.QueryRow(ctx, "SELECT is_locked FROM bootstrap_lock WHERE id = 1").Scan(&bootstrapped)
	status["bootstrap_complete"] = bootstrapped

	// Teams
	var teamCount int
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM teams WHERE deleted_at IS NULL").Scan(&teamCount)
	status["first_team_exists"] = teamCount > 0

	// Users
	var userCount int
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&userCount)
	status["users_exist"] = userCount > 0

	// Integration keys
	var keyCount int
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM integration_api_keys WHERE revoked_at IS NULL").Scan(&keyCount)
	status["integration_key_created"] = keyCount > 0

	// Email configured (from env, not from DB)
	status["email_configured"] = h.cfg.SMTPHost != ""
	status["email_mode"] = h.cfg.EmailMode

	// Proxmox mode
	status["proxmox_mode"] = "fake"
	if h.cfg.ProxmoxEnabled {
		status["proxmox_mode"] = "real"
	}

	// Webhook signing enforced
	status["webhook_signing_enforced"] = h.cfg.Env == "production"

	// Worker profile status
	status["agent_profile_required"] = "Agent profile optional — use 'docker compose --profile agent up -d' when WORKER_TOKEN is set"

	// Next actions
	var nextActions []string
	if !bootstrapped {
		nextActions = append(nextActions, "Bootstrap the platform via POST /api/bootstrap")
	}
	if bootstrapped && keyCount == 0 {
		nextActions = append(nextActions, "Create an integration key for webhook ingestion")
	}
	if h.cfg.SMTPHost == "" && h.cfg.EmailMode == "smtp" {
		nextActions = append(nextActions, "Configure SMTP_HOST for email delivery")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "All setup items complete. Review docs/ops/ for operational guides.")
	}
	status["next_actions"] = nextActions

	writeJSON(w, http.StatusOK, status)
}

var _ = json.Marshal
