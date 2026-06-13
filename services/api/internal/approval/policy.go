package approval

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Policy represents a resolved approval policy for a risk level.
type Policy struct {
	ID              uuid.UUID
	TeamID          uuid.UUID
	Name            string
	RiskLevel       string
	RequiresMFA     bool
	RequiresApproval bool
	AutoApprove     bool
	TimeoutSeconds  int
	MinApprovers    int
	AllowSelfApprove bool
}

// ResolvePolicy finds the default policy for a given team + risk level.
// Falls back to built-in defaults if no policy exists.
func ResolvePolicy(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID, riskLevel string) (*Policy, error) {
	p := &Policy{TeamID: teamID, RiskLevel: riskLevel}

	// Try DB lookup
	err := pool.QueryRow(ctx, `
		SELECT id, name, risk_level, requires_mfa, requires_approval, auto_approve,
		       timeout_seconds, min_approvers, allow_self_approve
		FROM approval_policies
		WHERE team_id = $1 AND risk_level = $2 AND is_default = true
		LIMIT 1
	`, teamID, riskLevel).Scan(
		&p.ID, &p.Name, &p.RiskLevel, &p.RequiresMFA, &p.RequiresApproval,
		&p.AutoApprove, &p.TimeoutSeconds, &p.MinApprovers, &p.AllowSelfApprove,
	)

	if err != nil {
		// Fall back to built-in defaults
		return builtinPolicy(teamID, riskLevel), nil
	}
	return p, nil
}

// builtinPolicy returns hardcoded safe defaults when no DB policy exists.
func builtinPolicy(teamID uuid.UUID, riskLevel string) *Policy {
	switch riskLevel {
	case "low":
		return &Policy{TeamID: teamID, RiskLevel: "low", RequiresMFA: false, RequiresApproval: false, AutoApprove: true, TimeoutSeconds: 3600, MinApprovers: 1, AllowSelfApprove: true}
	case "medium":
		return &Policy{TeamID: teamID, RiskLevel: "medium", RequiresMFA: false, RequiresApproval: true, AutoApprove: false, TimeoutSeconds: 3600, MinApprovers: 1, AllowSelfApprove: false}
	case "high":
		return &Policy{TeamID: teamID, RiskLevel: "high", RequiresMFA: true, RequiresApproval: true, AutoApprove: false, TimeoutSeconds: 1800, MinApprovers: 1, AllowSelfApprove: false}
	case "critical":
		return &Policy{TeamID: teamID, RiskLevel: "critical", RequiresMFA: true, RequiresApproval: true, AutoApprove: false, TimeoutSeconds: 900, MinApprovers: 2, AllowSelfApprove: false}
	default:
		return &Policy{TeamID: teamID, RiskLevel: "medium", RequiresMFA: true, RequiresApproval: true, AutoApprove: false, TimeoutSeconds: 1800, MinApprovers: 1, AllowSelfApprove: false}
	}
}

// ExpiresAt computes when an approval request expires.
func (p *Policy) ExpiresAt() time.Time {
	return time.Now().Add(time.Duration(p.TimeoutSeconds) * time.Second)
}

// CountApprovals counts approved decisions for a request.
func CountApprovals(ctx context.Context, pool *pgxpool.Pool, approvalID uuid.UUID) (int, error) {
	var count int
	err := pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM approval_decisions WHERE approval_id=$1 AND decision='approved'",
		approvalID).Scan(&count)
	return count, err
}

// CountRejections counts rejected decisions for a request.
func CountRejections(ctx context.Context, pool *pgxpool.Pool, approvalID uuid.UUID) (int, error) {
	var count int
	err := pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM approval_decisions WHERE approval_id=$1 AND decision='rejected'",
		approvalID).Scan(&count)
	return count, err
}

// HasUserDecided checks if a user has already made a decision on a request.
func HasUserDecided(ctx context.Context, pool *pgxpool.Pool, approvalID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM approval_decisions WHERE approval_id=$1 AND decided_by=$2)",
		approvalID, userID).Scan(&exists)
	return exists, err
}

// ValidateRiskLevel checks the risk level is one of the allowed values.
func ValidateRiskLevel(level string) error {
	switch level {
	case "low", "medium", "high", "critical":
		return nil
	default:
		return fmt.Errorf("invalid risk level: %s (must be low, medium, high, or critical)", level)
	}
}

// IsTerminalState returns true if the approval cannot transition further.
func IsTerminalState(status string) bool {
	switch status {
	case "approved", "rejected", "cancelled", "expired", "executed", "failed":
		return true
	default:
		return false
	}
}
