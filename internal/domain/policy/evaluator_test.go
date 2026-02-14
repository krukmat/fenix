package policy

import (
	"context"
	"database/sql"
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
