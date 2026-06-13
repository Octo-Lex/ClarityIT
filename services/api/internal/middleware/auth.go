package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/clarityit/api/internal/authz"
	"github.com/clarityit/api/internal/iam"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ClaimsKey   = "claims"
	UserIDKey   = "user_id"
	TeamIDKey   = "team_id"
	TeamRoleKey = "team_role"
)

// ResolveAuth extracts JWT from Authorization header and populates context.
func ResolveAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := iam.ParseAccessToken(jwtSecret, parts[1])
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, TeamIDKey, claims.TeamID)
			ctx = context.WithValue(ctx, TeamRoleKey, claims.TeamRole)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth rejects requests without valid claims.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := getClaims(r)
		if claims == nil {
			writeErr(w, http.StatusUnauthorized, "Missing authorization")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePermission checks that the authenticated user has a specific permission
// in their current team context. Queries the database via role_permissions.
// Platform-owner bypass is recorded in context for audit metadata.
func RequirePermission(pool *pgxpool.Pool, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := getClaims(r)
			if claims == nil {
				writeErr(w, http.StatusUnauthorized, "Missing authorization")
				return
			}

			// Platform owners bypass team permission checks
			if claims.IsOwner {
				bypass := &authz.Bypass{
					Path:              "platform_owner_bypass",
					PermissionChecked: permission,
					TeamID:            claims.TeamID,
				}
				ctx := authz.WithBypass(r.Context(), bypass)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if claims.TeamID == "" {
				writeErr(w, http.StatusForbidden, "No team context")
				return
			}

			userID, _ := uuid.Parse(claims.UserID)
			teamID, _ := uuid.Parse(claims.TeamID)

			// Query: does this user's role in this team grant this permission?
			var hasPerm bool
			err := pool.QueryRow(r.Context(), `
				SELECT EXISTS(
					SELECT 1 FROM team_memberships tm
					JOIN role_permissions rp ON rp.role_id = tm.role_id
					JOIN permissions p ON p.id = rp.permission_id
					WHERE tm.user_id = $1 AND tm.team_id = $2 AND p.name = $3
				)
			`, userID, teamID, permission).Scan(&hasPerm)

			if err != nil || !hasPerm {
				// Write permission-denied audit event
				ipHMAC := r.Header.Get("X-Request-ID") // best available
				writePermissionDeniedAudit(r.Context(), pool, userID, permission, claims.TeamID, ipHMAC)
				writeErr(w, http.StatusForbidden, "Permission denied: "+permission)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePlatformRole checks that the user has a specific platform role.
func RequirePlatformRole(pool *pgxpool.Pool, roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := getClaims(r)
			if claims == nil {
				writeErr(w, http.StatusUnauthorized, "Missing authorization")
				return
			}

			userID, _ := uuid.Parse(claims.UserID)

			var hasRole bool
			err := pool.QueryRow(r.Context(), `
				SELECT EXISTS(
					SELECT 1 FROM user_platform_roles upr
					JOIN platform_roles pr ON pr.id = upr.platform_role_id
					WHERE upr.user_id = $1 AND pr.name = $2 AND upr.revoked_at IS NULL
				)
			`, userID, roleName).Scan(&hasRole)

			if err != nil || !hasRole {
				writeErr(w, http.StatusForbidden, "Requires platform role: "+roleName)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAuthzBypass returns the platform-owner bypass metadata from context, if any.
func GetAuthzBypass(ctx context.Context) *authz.Bypass {
	return authz.GetBypass(ctx)
}

// writePermissionDeniedAudit writes a sanitized audit event for denied permission checks.
func writePermissionDeniedAudit(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, permission, teamID, ipHMAC string) {
	eventID := uuid.New().String()
	metadata, _ := json.Marshal(map[string]string{
		"permission_checked": permission,
		"team_id":           teamID,
		"denial_reason":     "insufficient_role_permissions",
	})
	summary := fmt.Sprintf("Permission denied: %s", permission)
	pool.Exec(ctx, `
		INSERT INTO audit_logs (
			event_id, actor_id, actor_type, action, entity_type, entity_id,
			old_value, new_value, change_summary, ip_hmac
		) VALUES ($1, $2, 'user', 'identity.permission.denied', 'permission', $2, '{}', $3, $4, $5)
	`, eventID, userID, metadata, summary, ipHMAC)
}

func getClaims(r *http.Request) *iam.TokenClaims {
	claims, _ := r.Context().Value(ClaimsKey).(*iam.TokenClaims)
	return claims
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"detail":"` + msg + `"}`))
}
