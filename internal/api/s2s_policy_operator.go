package api

import (
	"context"
	"database/sql"

	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

// S2SPolicyPermissionEngine is satisfied by Fenix policy.PolicyEngine.
// B2.B4 reuses Fenix's unified RBAC/ABAC decisions; it does not create
// another permission authority or ToolRegistry executor.
type S2SPolicyPermissionEngine interface {
	CheckActionPermission(context.Context, string, string, string, map[string]string) (bool, error)
}

// S2SPolicyOperatorAuthorizer checks BOTH authenticated Fenix user workspace
// and the existing policy engine's explicit s2s_capabilities:manage action.
// No new role grant is provisioned by this code.
type S2SPolicyOperatorAuthorizer struct {
	db *sql.DB
	policy S2SPolicyPermissionEngine
}

func NewS2SPolicyOperatorAuthorizer(
	db *sql.DB, policy S2SPolicyPermissionEngine,
) *S2SPolicyOperatorAuthorizer {
	return &S2SPolicyOperatorAuthorizer{db: db, policy: policy}
}

func (a *S2SPolicyOperatorAuthorizer) AuthorizeS2SPolicyChange(
	ctx context.Context, operatorID, workspaceID string,
) error {
	if a == nil || a.db == nil || a.policy == nil ||
		operatorID == "" || workspaceID == "" {
		return integration.ErrCapabilityConfiguration
	}
	var actualWorkspace string
	err := a.db.QueryRowContext(ctx,
		"SELECT workspace_id FROM user_account WHERE id = ? AND status = 'active' LIMIT 1",
		operatorID).Scan(&actualWorkspace)
	if err != nil || actualWorkspace != workspaceID {
		return integration.ErrCapabilityConfiguration
	}
	allowed, err := a.policy.CheckActionPermission(ctx, operatorID,
		"s2s_capabilities", "manage", map[string]string{"workspace_id": workspaceID})
	if err != nil || !allowed {
		return integration.ErrCapabilityConfiguration
	}
	return nil
}

// NewDormantS2SCapabilityOperatorStore exposes a management-side store ONLY
// to explicitly trusted internal operator code. The S2S admission HTTP
// handler never uses this constructor. Missing DB/policy remains fail-closed.
func NewDormantS2SCapabilityOperatorStore(
	db *sql.DB, policy S2SPolicyPermissionEngine,
) *integration.S2SCapabilityPolicyStore {
	return integration.NewS2SCapabilityPolicyStoreWithOperator(
		db, NewS2SPolicyOperatorAuthorizer(db, policy),
	)
}
