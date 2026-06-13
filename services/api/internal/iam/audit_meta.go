package iam

import (
	"context"
	"encoding/json"

	"github.com/clarityit/api/internal/authz"
	"github.com/google/uuid"
)

// AuditMeta returns raw JSON with authz bypass info, or nil if no bypass.
func AuditMeta(ctx context.Context) json.RawMessage {
	bypass := authz.GetBypass(ctx)
	if bypass != nil {
		meta, _ := json.Marshal(map[string]string{
			"authorization_path":  bypass.Path,
			"permission_checked":  bypass.PermissionChecked,
			"team_id":            bypass.TeamID,
		})
		return meta
	}
	return nil
}

// AuditTeamMeta returns JSON with team_id and optional authz bypass info.
func AuditTeamMeta(ctx context.Context, teamID uuid.UUID) json.RawMessage {
	bypass := authz.GetBypass(ctx)
	if bypass != nil {
		meta, _ := json.Marshal(map[string]string{
			"authorization_path":  bypass.Path,
			"permission_checked":  bypass.PermissionChecked,
			"team_id":            teamID.String(),
		})
		return meta
	}
	return json.RawMessage(`{"team_id":"` + teamID.String() + `"}`)
}

// MergeAuditMeta merges authz bypass metadata into an existing value map.
// If bypass metadata exists, the authorization_path, permission_checked, and team_id
// are added to the map. Returns the merged JSON.
func MergeAuditMeta(ctx context.Context, existing map[string]any) json.RawMessage {
	bypass := authz.GetBypass(ctx)
	if bypass != nil {
		existing["authorization_path"] = bypass.Path
		existing["permission_checked"] = bypass.PermissionChecked
		if bypass.TeamID != "" {
			existing["team_id"] = bypass.TeamID
		}
	}
	merged, _ := json.Marshal(existing)
	return merged
}
