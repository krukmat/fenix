package eval_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

func TestBenchmarkRegistry_CreateBenchmarkCase(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "benchmark-create")
	runner := eval.NewRunnerService(db)
	service := eval.NewBenchmarkRegistryService(db, runner)

	benchmarkCase, err := service.Create(context.Background(), eval.CreateBenchmarkCaseInput{
		WorkspaceID:     wsID,
		Slug:            "password-reset",
		Name:            "Password Reset",
		Domain:          "support",
		Version:         2,
		InputPayload:    json.RawMessage(`{"prompt":"reset password"}`),
		ExpectedOutcome: json.RawMessage(`{"status":"success"}`),
		Tags:            []string{"support", "deterministic"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if benchmarkCase.ID == "" {
		t.Fatal("expected benchmark case ID to be populated")
	}
	if benchmarkCase.Version != 2 {
		t.Fatalf("Version = %d; want 2", benchmarkCase.Version)
	}
	if benchmarkCase.Slug != "password-reset" {
		t.Fatalf("Slug = %q; want %q", benchmarkCase.Slug, "password-reset")
	}
}

func TestBenchmarkRegistry_RunBenchmarkCase(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "benchmark-run")
	mustInsertEvalSuite(t, db, wsID, "suite-1")

	runner := eval.NewRunnerService(db)
	service := eval.NewBenchmarkRegistryService(db, runner)
	benchmarkCase, err := service.Create(context.Background(), eval.CreateBenchmarkCaseInput{
		WorkspaceID:     wsID,
		Slug:            "password-reset",
		Name:            "Password Reset",
		Domain:          "support",
		Version:         1,
		InputPayload:    json.RawMessage(`{"prompt":"reset password"}`),
		ExpectedOutcome: json.RawMessage(`{"status":"success"}`),
		Tags:            []string{"support"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	run, err := service.RunBenchmarkCase(context.Background(), benchmarkCase.ID, eval.RunBenchmarkCaseInput{
		WorkspaceID: wsID,
		EvalSuiteID: "suite-1",
	})
	if err != nil {
		t.Fatalf("RunBenchmarkCase: %v", err)
	}
	if run.Provenance == nil {
		t.Fatal("expected provenance to be present")
	}
	if run.Provenance.Mode != eval.ReplayModeBenchmark {
		t.Fatalf("Mode = %q; want %q", run.Provenance.Mode, eval.ReplayModeBenchmark)
	}
	if got := deref(run.Provenance.BenchmarkCaseID); got != benchmarkCase.ID {
		t.Fatalf("BenchmarkCaseID = %q; want %q", got, benchmarkCase.ID)
	}

	assertBenchmarkCasePersisted(t, db, run.ID, benchmarkCase.ID)
}

func TestBenchmarkRegistry_Create_WithSyntheticOrg(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "benchmark-synthetic")
	runner := eval.NewRunnerService(db)
	orgSvc := eval.NewSyntheticOrgService(db)
	service := eval.NewBenchmarkRegistryService(db, runner)

	org, err := orgSvc.Create(context.Background(), eval.CreateSyntheticOrgInput{
		WorkspaceID: wsID,
		Slug:        "seeded-acme",
		Name:        "Seeded Acme",
		Version:     1,
		Seed:        11,
		FixtureData: json.RawMessage(`{"accounts":[{"id":"acc-1"}]}`),
	})
	if err != nil {
		t.Fatalf("Create synthetic org: %v", err)
	}

	benchmarkCase, err := service.Create(context.Background(), eval.CreateBenchmarkCaseInput{
		WorkspaceID:     wsID,
		SyntheticOrgID:  &org.ID,
		Slug:            "password-reset",
		Name:            "Password Reset",
		Domain:          "support",
		Version:         1,
		InputPayload:    json.RawMessage(`{"prompt":"reset password"}`),
		ExpectedOutcome: json.RawMessage(`{"status":"success"}`),
		Tags:            []string{"support"},
	})
	if err != nil {
		t.Fatalf("Create benchmark with synthetic org: %v", err)
	}
	if got := deref(benchmarkCase.SyntheticOrgID); got != org.ID {
		t.Fatalf("SyntheticOrgID = %q; want %q", got, org.ID)
	}
}

func TestBenchmarkRegistry_Create_WithSyntheticOrgOutsideWorkspace_Fails(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsA := mustCreateWorkspace(t, db, "benchmark-synthetic-a")
	wsB := mustCreateWorkspace(t, db, "benchmark-synthetic-b")
	runner := eval.NewRunnerService(db)
	orgSvc := eval.NewSyntheticOrgService(db)
	service := eval.NewBenchmarkRegistryService(db, runner)

	org, err := orgSvc.Create(context.Background(), eval.CreateSyntheticOrgInput{
		WorkspaceID: wsA,
		Slug:        "seeded-acme",
		Name:        "Seeded Acme",
		Version:     1,
		Seed:        22,
		FixtureData: json.RawMessage(`{"accounts":[{"id":"acc-1"}]}`),
	})
	if err != nil {
		t.Fatalf("Create synthetic org: %v", err)
	}

	_, err = service.Create(context.Background(), eval.CreateBenchmarkCaseInput{
		WorkspaceID:     wsB,
		SyntheticOrgID:  &org.ID,
		Slug:            "password-reset",
		Name:            "Password Reset",
		Domain:          "support",
		Version:         1,
		InputPayload:    json.RawMessage(`{"prompt":"reset password"}`),
		ExpectedOutcome: json.RawMessage(`{"status":"success"}`),
		Tags:            []string{"support"},
	})
	if err == nil {
		t.Fatal("expected synthetic org workspace validation error")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows wrapped, got %v", err)
	}
}

func assertBenchmarkCasePersisted(t *testing.T, db *sql.DB, runID, benchmarkCaseID string) {
	t.Helper()

	var persistedBenchmarkID string
	var replayMode string
	err := db.QueryRow(`SELECT benchmark_case_id, replay_mode FROM eval_run WHERE id = ?`, runID).Scan(&persistedBenchmarkID, &replayMode)
	if err != nil {
		t.Fatalf("read eval_run: %v", err)
	}
	if persistedBenchmarkID != benchmarkCaseID {
		t.Fatalf("persisted benchmark_case_id = %q; want %q", persistedBenchmarkID, benchmarkCaseID)
	}
	if replayMode != string(eval.ReplayModeBenchmark) {
		t.Fatalf("persisted replay_mode = %q; want %q", replayMode, eval.ReplayModeBenchmark)
	}
}
