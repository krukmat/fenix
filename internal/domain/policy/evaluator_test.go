// Traces: FR-060, FR-071
package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

func setupPolicyTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("sqlite.NewDB failed: %v", err)
	}
	// IMPORTANT: sqlite :memory: is per-connection.
	// Force a single connection so migrations and queries share same DB.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("sqlite.MigrateUp failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedWorkspaceUserRole(t *testing.T, db *sql.DB, permsJSON string) (workspaceID, userID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)

	workspaceID = uuid.NewV7().String()
	userID = uuid.NewV7().String()
	roleID := uuid.NewV7().String()
	userRoleID := uuid.NewV7().String()

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, workspaceID, "Policy WS", "policy-ws-"+workspaceID, now, now); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?)
	`, userID, workspaceID, fmt.Sprintf("%s@example.com", userID), "Policy User", now, now); err != nil {
		t.Fatalf("insert user_account: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO role (id, workspace_id, name, permissions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, roleID, workspaceID, "policy-role", permsJSON, now, now); err != nil {
		t.Fatalf("insert role: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO user_role (id, user_id, role_id, created_at)
		VALUES (?, ?, ?, ?)
	`, userRoleID, userID, roleID, now); err != nil {
		t.Fatalf("insert user_role: %v", err)
	}

	return workspaceID, userID
}

func seedActivePolicyVersion(t *testing.T, db *sql.DB, workspaceID string, version int, policyJSON string) (string, string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	setID := uuid.NewV7().String()
	verID := uuid.NewV7().String()

	if _, err := db.Exec(`
		INSERT INTO policy_set (id, workspace_id, name, description, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, ?, ?)
	`, setID, workspaceID, "default-policy", "test set", now, now); err != nil {
		t.Fatalf("insert policy_set: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO policy_version (id, policy_set_id, workspace_id, version_number, policy_json, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?)
	`, verID, setID, workspaceID, version, policyJSON, now); err != nil {
		t.Fatalf("insert policy_version: %v", err)
	}

	return setID, verID
}

func TestBuildPermissionFilter(t *testing.T) {
	t.Run("read_all role -> workspace-only filter", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		workspaceID, userID := seedWorkspaceUserRole(t, db, `{"global":["read_all"]}`)

		engine := NewPolicyEngine(db, nil, nil)
		filter, err := engine.BuildPermissionFilter(context.Background(), userID)
		if err != nil {
			t.Fatalf("BuildPermissionFilter error: %v", err)
		}

		if filter.Where != "workspace_id = ?" {
			t.Fatalf("unexpected where: %q", filter.Where)
		}
		if len(filter.Args) != 1 || filter.Args[0] != workspaceID {
			t.Fatalf("unexpected args: %#v", filter.Args)
		}
	})

	t.Run("without read_all -> workspace+owner filter", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		workspaceID, userID := seedWorkspaceUserRole(t, db, `{"records":["read_own"]}`)

		engine := NewPolicyEngine(db, nil, nil)
		filter, err := engine.BuildPermissionFilter(context.Background(), userID)
		if err != nil {
			t.Fatalf("BuildPermissionFilter error: %v", err)
		}

		if filter.Where != "workspace_id = ? AND owner_id = ?" {
			t.Fatalf("unexpected where: %q", filter.Where)
		}
		if len(filter.Args) != 2 || filter.Args[0] != workspaceID || filter.Args[1] != userID {
			t.Fatalf("unexpected args: %#v", filter.Args)
		}
	})
}

func TestRedactPII(t *testing.T) {
	db := setupPolicyTestDB(t)
	engine := NewPolicyEngine(db, nil, nil)

	text := "Contactar a john.doe@example.com o al +34 600-123-456. SSN: 123-45-6789"
	evidence := []knowledge.Evidence{{Snippet: &text}}

	redacted, err := engine.RedactPII(context.Background(), evidence)
	if err != nil {
		t.Fatalf("RedactPII error: %v", err)
	}
	if len(redacted) != 1 || redacted[0].Snippet == nil {
		t.Fatalf("unexpected redacted result: %#v", redacted)
	}

	out := *redacted[0].Snippet
	if out == text {
		t.Fatalf("expected snippet to change after PII redaction")
	}
	if !redacted[0].PiiRedacted {
		t.Fatalf("expected PiiRedacted=true")
	}
	if !(containsToken(out, "[EMAIL_") && containsToken(out, "[PHONE_") && containsToken(out, "[SSN_")) {
		t.Fatalf("expected EMAIL/PHONE/SSN tokens in output, got: %q", out)
	}
}

func containsToken(s, tokenPrefix string) bool {
	return len(s) > 0 && (len(tokenPrefix) > 0) && (indexOf(s, tokenPrefix) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestCheckToolPermission(t *testing.T) {
	t.Run("granted", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		_, userID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)

		engine := NewPolicyEngine(db, nil, nil)
		ok, err := engine.CheckToolPermission(context.Background(), userID, "update_case")
		if err != nil {
			t.Fatalf("CheckToolPermission error: %v", err)
		}
		if !ok {
			t.Fatalf("expected permission granted")
		}
	})

	t.Run("denied", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		_, userID := seedWorkspaceUserRole(t, db, `{"tools":["create_task"]}`)

		engine := NewPolicyEngine(db, nil, nil)
		ok, err := engine.CheckToolPermission(context.Background(), userID, "update_case")
		if err != nil {
			t.Fatalf("CheckToolPermission error: %v", err)
		}
		if ok {
			t.Fatalf("expected permission denied")
		}
	})
}

func TestLogAuditEvent(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, userID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)

	auditService := audit.NewAuditService(db)
	engine := NewPolicyEngine(db, nil, auditService)

	entityType := "case"
	entityID := uuid.NewV7().String()
	err := engine.LogAuditEvent(context.Background(), AuditLogEvent{
		WorkspaceID: workspaceID,
		ActorID:     userID,
		ActorType:   audit.ActorTypeUser,
		Action:      "tool.execute",
		EntityType:  &entityType,
		EntityID:    &entityID,
		Outcome:     audit.OutcomeSuccess,
		Details:     map[string]any{"tool": "update_case"},
	})
	if err != nil {
		t.Fatalf("LogAuditEvent error: %v", err)
	}

	items, total, err := auditService.ListByWorkspace(context.Background(), workspaceID, 10, 0)
	if err != nil {
		t.Fatalf("ListByWorkspace error: %v", err)
	}
	if total < 1 || len(items) < 1 {
		t.Fatalf("expected at least one audit event, total=%d len=%d", total, len(items))
	}
	if items[0].Action != "tool.execute" {
		t.Fatalf("unexpected action in audit event: %q", items[0].Action)
	}
}

func TestEvaluatePolicyDecision_DeterministicPrecedenceAndTrace(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, userID := seedWorkspaceUserRole(t, db, `{"tools":["read_own"]}`)

	policyJSON := `{
	  "rules": [
	    {"id":"allow_tools_wildcard","resource":"tools","action":"*","effect":"allow","priority":1},
	    {"id":"deny_update_case","resource":"tools","action":"update_case","effect":"deny","priority":10}
	  ]
	}`
	setID, verID := seedActivePolicyVersion(t, db, workspaceID, 1, policyJSON)

	auditService := audit.NewAuditService(db)
	engine := NewPolicyEngine(db, nil, auditService)

	decision, err := engine.EvaluatePolicyDecision(context.Background(), workspaceID, userID, "tools", "update_case", nil)
	if err != nil {
		t.Fatalf("EvaluatePolicyDecision error: %v", err)
	}
	if decision.Allow {
		t.Fatalf("expected deny by precedence")
	}
	if decision.Trace == nil {
		t.Fatalf("expected non-nil trace")
	}
	if decision.Trace.PolicySetID != setID || decision.Trace.PolicyVersionID != verID {
		t.Fatalf("unexpected policy identifiers in trace: %#v", decision.Trace)
	}
	if decision.Trace.MatchedRuleID != "deny_update_case" || decision.Trace.MatchedEffect != "deny" {
		t.Fatalf("unexpected matched rule/effect: %#v", decision.Trace)
	}

	events, err := auditService.ListByAction(context.Background(), workspaceID, "policy.evaluated", 20, 0)
	if err != nil {
		t.Fatalf("ListByAction(policy.evaluated) error: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("expected at least one policy.evaluated event")
	}

	var details map[string]any
	if err := json.Unmarshal(events[0].Details, &details); err != nil {
		t.Fatalf("unmarshal details error: %v", err)
	}
	meta, ok := details["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata map in details: %#v", details)
	}
	if got, _ := meta["matched_rule_id"].(string); got != "deny_update_case" {
		t.Fatalf("unexpected matched_rule_id in audit details: %v", meta["matched_rule_id"])
	}
}

func TestCheckToolPermission_UsesActivePolicyVersionWhenAvailable(t *testing.T) {
	t.Run("policy deny overrides role allow", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		workspaceID, userID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)
		policyJSON := `{"rules":[{"id":"deny_update_case","resource":"tools","action":"update_case","effect":"deny","priority":100}]}`
		seedActivePolicyVersion(t, db, workspaceID, 1, policyJSON)

		engine := NewPolicyEngine(db, nil, nil)
		ok, err := engine.CheckToolPermission(context.Background(), userID, "update_case")
		if err != nil {
			t.Fatalf("CheckToolPermission error: %v", err)
		}
		if ok {
			t.Fatalf("expected denied by active policy")
		}
	})

	t.Run("fallback to role permissions when no active policy", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		_, userID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)

		engine := NewPolicyEngine(db, nil, nil)
		ok, err := engine.CheckToolPermission(context.Background(), userID, "update_case")
		if err != nil {
			t.Fatalf("CheckToolPermission error: %v", err)
		}
		if !ok {
			t.Fatalf("expected granted by fallback role permissions")
		}
	})
}

func TestCheckActionPermission(t *testing.T) {
	t.Run("role fallback allows api admin action", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		_, userID := seedWorkspaceUserRole(t, db, `{"api":["admin"]}`)

		engine := NewPolicyEngine(db, nil, nil)
		ok, err := engine.CheckActionPermission(context.Background(), userID, "api", "admin.tools.create", nil)
		if err != nil {
			t.Fatalf("CheckActionPermission error: %v", err)
		}
		if !ok {
			t.Fatalf("expected granted by role fallback")
		}
	})

	t.Run("active policy deny overrides role fallback", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		workspaceID, userID := seedWorkspaceUserRole(t, db, `{"api":["admin"]}`)
		policyJSON := `{"rules":[{"id":"deny_admin_tools_create","resource":"api","action":"admin.tools.create","effect":"deny","priority":100}]}`
		seedActivePolicyVersion(t, db, workspaceID, 1, policyJSON)

		engine := NewPolicyEngine(db, nil, nil)
		ok, err := engine.CheckActionPermission(context.Background(), userID, "api", "admin.tools.create", nil)
		if err != nil {
			t.Fatalf("CheckActionPermission error: %v", err)
		}
		if ok {
			t.Fatalf("expected denied by active policy")
		}
	})
}
