package eval_test

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

func TestEvalDomainTypes_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	synthetic := eval.SyntheticOrg{
		ID:          "org-1",
		WorkspaceID: "ws-1",
		Slug:        "acme-support",
		Name:        "Acme Support",
		Version:     2,
		Seed:        42,
		FixtureData: json.RawMessage(`{"accounts":[{"id":"acc-1"}]}`),
	}
	benchmark := eval.BenchmarkCase{
		ID:              "bench-1",
		WorkspaceID:     "ws-1",
		SyntheticOrgID:  stringPtr("org-1"),
		Slug:            "password-reset",
		Name:            "Password Reset",
		Domain:          "support",
		Version:         3,
		InputPayload:    json.RawMessage(`{"prompt":"reset my password"}`),
		ExpectedOutcome: json.RawMessage(`{"should_abstain":false}`),
		Tags:            []string{"support", "deterministic"},
	}
	provenance := eval.ReplayProvenance{
		Mode:                       eval.ReplayModeReplay,
		BenchmarkCaseID:            stringPtr("bench-1"),
		SyntheticOrgID:             stringPtr("org-1"),
		SourceAgentRunID:           stringPtr("agent-run-1"),
		SourceCognitiveWorkspaceID: stringPtr("cw-1"),
		SourceTraceID:              stringPtr("trace-1"),
	}

	payload := struct {
		Synthetic  eval.SyntheticOrg     `json:"synthetic"`
		Benchmark  eval.BenchmarkCase    `json:"benchmark"`
		Provenance eval.ReplayProvenance `json:"provenance"`
	}{Synthetic: synthetic, Benchmark: benchmark, Provenance: provenance}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTrip struct {
		Synthetic  eval.SyntheticOrg     `json:"synthetic"`
		Benchmark  eval.BenchmarkCase    `json:"benchmark"`
		Provenance eval.ReplayProvenance `json:"provenance"`
	}
	if err := json.Unmarshal(raw, &roundTrip); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTrip.Synthetic.Seed != synthetic.Seed {
		t.Fatalf("Seed mismatch: got %d want %d", roundTrip.Synthetic.Seed, synthetic.Seed)
	}
	if string(roundTrip.Benchmark.InputPayload) != string(benchmark.InputPayload) {
		t.Fatalf("InputPayload mismatch: got %s want %s", roundTrip.Benchmark.InputPayload, benchmark.InputPayload)
	}
	if roundTrip.Provenance.Mode != eval.ReplayModeReplay {
		t.Fatalf("Mode mismatch: got %s", roundTrip.Provenance.Mode)
	}
	if got := deref(roundTrip.Provenance.SourceTraceID); got != "trace-1" {
		t.Fatalf("SourceTraceID mismatch: got %q", got)
	}
}

func TestMigrate_EvalFrameworkExtensionTablesCreated(t *testing.T) {
	db := mustOpenDB(t)

	assertTableExists(t, db, "synthetic_org")
	assertTableExists(t, db, "benchmark_case")
	assertColumnExists(t, db, "eval_run", "benchmark_case_id")
	assertColumnExists(t, db, "eval_run", "synthetic_org_id")
	assertColumnExists(t, db, "eval_run", "source_agent_run_id")
	assertColumnExists(t, db, "eval_run", "source_cognitive_workspace_id")
	assertColumnExists(t, db, "eval_run", "source_trace_id")
	assertColumnExists(t, db, "eval_run", "replay_mode")
}

func TestMigrate_EvalFrameworkExtensionConstraints(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "constraints")

	if _, err := db.Exec(`
		INSERT INTO synthetic_org (id, workspace_id, slug, name, version, seed, fixture_data)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"org-1", wsID, "seeded-org", "Seeded Org", 1, 7, `{"seed":7}`,
	); err != nil {
		t.Fatalf("insert synthetic_org: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO benchmark_case (id, workspace_id, synthetic_org_id, slug, name, domain, version, input_payload, expected_outcome, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"bench-1", wsID, "org-1", "case-1", "Case 1", "support", 1, `{}`, `{}`, `[]`,
	); err != nil {
		t.Fatalf("insert benchmark_case: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO benchmark_case (id, workspace_id, synthetic_org_id, slug, name, domain, version, input_payload, expected_outcome, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"bench-duplicate", wsID, "org-1", "case-1", "Case 1 v1", "support", 1, `{}`, `{}`, `[]`,
	); err == nil {
		t.Fatal("expected unique workspace+slug+version constraint to fail")
	}

	if _, err := db.Exec(`
		INSERT INTO eval_run (
			id, workspace_id, eval_suite_id, status, scores, details, benchmark_case_id, synthetic_org_id, replay_mode, started_at, created_at
		)
		VALUES (?, ?, ?, 'running', '{}', '[]', ?, ?, 'benchmark', datetime('now'), datetime('now'))`,
		"run-invalid", wsID, "missing-suite", "bench-1", "org-1",
	); err == nil {
		t.Fatal("expected eval_run FK constraint to fail for missing suite")
	}
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var name string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
		t.Fatalf("table %s missing: %v", table, err)
	}
}

func assertColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()

	var (
		cid      int
		name     string
		kind     string
		notNull  int
		defaultV sql.NullString
		pk       int
	)
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultV, &pk); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		if name == column {
			return
		}
	}
	t.Fatalf("column %s missing from %s", column, table)
}

func stringPtr(v string) *string { return &v }

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
