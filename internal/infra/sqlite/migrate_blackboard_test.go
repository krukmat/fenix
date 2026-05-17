// Tests for migration 031: cognitive workspace schema (Task A.1)
package sqlite_test

import (
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func TestMigrate_CognitiveWorkspaceTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "cognitive_workspace")
}

func TestMigrate_ReasoningEventTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "reasoning_event")
}

func TestMigrate_SignalHypothesisTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "signal_hypothesis")
}

func TestMigrate_AgentMemoryTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "agent_memory")
}

func TestMigrate_AgentRunHasCognitiveWorkspaceColumn(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	// Insert a workspace and agent_definition required for agent_run FK
	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-bb-col', 'BB Col WS', 'bb-col-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, allowed_tools, limits, trigger_config, created_at, updated_at)
		VALUES ('ad-bb-col', 'ws-bb-col', 'test-agent', 'support', '[]', '{}', '{}', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("agent_definition insert: %v", err)
	}

	// Insert agent_run with NULL cognitive_workspace_id — must succeed (column is nullable)
	_, err := db.Exec(`
		INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, created_at, updated_at)
		VALUES ('run-bb-col', 'ws-bb-col', 'ad-bb-col', 'manual', 'running', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("agent_run insert without cognitive_workspace_id failed: %v; want success (nullable column)", err)
	}
}

func TestMigrate_AgentRunCognitiveWorkspaceForeignKey(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-bb-fk', 'BB FK WS', 'bb-fk-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, allowed_tools, limits, trigger_config, created_at, updated_at)
		VALUES ('ad-bb-fk', 'ws-bb-fk', 'test-agent', 'support', '[]', '{}', '{}', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("agent_definition insert: %v", err)
	}

	// Insert agent_run with a non-existent cognitive_workspace_id — must fail FK
	_, err := db.Exec(`
		INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, cognitive_workspace_id, created_at, updated_at)
		VALUES ('run-bb-fk', 'ws-bb-fk', 'ad-bb-fk', 'manual', 'running', 'nonexistent-cw-id', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("agent_run insert with invalid cognitive_workspace_id succeeded; want FK constraint error")
	}
}

func TestMigrate_CognitiveWorkspaceStatusCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-cw-status', 'CW Status WS', 'cw-status-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-status', 'ws-cw-status', 'invalid_status', datetime('now'))
	`)
	if err == nil {
		t.Error("cognitive_workspace insert with invalid status succeeded; want CHECK constraint error")
	}
}

func TestMigrate_ReasoningEventTypeCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-re-type', 'RE Type WS', 're-type-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-re-type', 'ws-re-type', 'active', datetime('now'))
	`); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO reasoning_event (id, cognitive_workspace_id, event_type, payload, created_at)
		VALUES ('re-type', 'cw-re-type', 'invalid_type', '{}', datetime('now'))
	`)
	if err == nil {
		t.Error("reasoning_event insert with invalid event_type succeeded; want CHECK constraint error")
	}
}

func TestMigrate_SignalHypothesisConfidenceCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-sh-conf', 'SH Conf WS', 'sh-conf-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-sh-conf', 'ws-sh-conf', 'active', datetime('now'))
	`); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO signal_hypothesis (id, cognitive_workspace_id, content, confidence, status, created_at)
		VALUES ('sh-conf', 'cw-sh-conf', 'test hypothesis', 1.5, 'open', datetime('now'))
	`)
	if err == nil {
		t.Error("signal_hypothesis insert with confidence > 1.0 succeeded; want CHECK constraint error")
	}
}

func TestMigrate_AgentMemoryScopeCheck(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-am-scope', 'AM Scope WS', 'am-scope-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-am-scope', 'ws-am-scope', 'active', datetime('now'))
	`); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO agent_memory (id, cognitive_workspace_id, key, value, scope, created_at, updated_at)
		VALUES ('am-scope', 'cw-am-scope', 'test-key', '{}', 'invalid_scope', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("agent_memory insert with invalid scope succeeded; want CHECK constraint error")
	}
}

func TestMigrate_CognitiveWorkspaceInsertValid(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-cw-valid', 'CW Valid WS', 'cw-valid-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	// Full valid insert with all optional fields null
	_, err := db.Exec(`
		INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-valid', 'ws-cw-valid', 'active', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("valid cognitive_workspace insert failed: %v", err)
	}
}

func TestMigrate_ReasoningEventAppendOnly(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-re-ao', 'RE AO WS', 're-ao-ws', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-re-ao', 'ws-re-ao', 'active', datetime('now'))
	`); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO reasoning_event (id, cognitive_workspace_id, event_type, payload, created_at)
		VALUES ('re-ao', 'cw-re-ao', 'observation', '{"note":"test"}', datetime('now'))
	`); err != nil {
		t.Fatalf("reasoning_event insert failed: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM reasoning_event WHERE id = 're-ao'").Scan(&count); err != nil {
		t.Fatalf("query reasoning_event: %v", err)
	}
	if count != 1 {
		t.Errorf("reasoning_event count = %d; want 1", count)
	}
}
