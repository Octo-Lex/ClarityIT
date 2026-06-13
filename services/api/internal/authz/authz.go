package authz

import "context"

// Bypass records that a permission check was bypassed due to platform-owner status.
type Bypass struct {
	Path             string `json:"authorization_path"`
	PermissionChecked string `json:"permission_checked"`
	TeamID           string `json:"team_id"`
}

type contextKey string

const bypassKey contextKey = "authz_bypass"

// WithBypass stores an authz bypass in context.
func WithBypass(ctx context.Context, b *Bypass) context.Context {
	return context.WithValue(ctx, bypassKey, b)
}

// GetBypass retrieves authz bypass metadata from context.
func GetBypass(ctx context.Context) *Bypass {
	b, _ := ctx.Value(bypassKey).(*Bypass)
	return b
}
