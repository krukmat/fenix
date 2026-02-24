package eval_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// mustOpenDB opens an in-memory SQLite database with migrations applied.
func mustOpenDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { db.Close() })
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	return db
}

// mustCreateWorkspace creates a workspace for testing.
// Task 4.7: FR-242
func mustCreateWorkspace(t *testing.T, db *sql.DB, suffix string) string {
	t.Helper()
	id := "ws-" + t.Name() + "-" + suffix
	slug := "slug-" + id
	_, err := db.Exec(
		`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		 VALUES (?, ?, ?, datetime('now'), datetime('now'))`,
		id, "Test", slug,
	)
	if err != nil {
		t.Fatalf("createWorkspace: %v", err)
	}
	return id
}

// TestSuiteService_Create_Success verifies suite creation.
func TestSuiteService_Create_Success(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "a")
	svc := eval.NewSuiteService(db)

	suite, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Support Suite",
		Domain:      "support",
		TestCases: []eval.TestCase{
			{Input: "How do I reset my password?", ExpectedKeywords: []string{"password", "reset"}, ShouldAbstain: false},
		},
		Thresholds: eval.Thresholds{Groundedness: 0.8, Exactitude: 0.85, Abstention: 0.95, Policy: 1.0},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if suite.ID == "" {
		t.Error("ID should not be empty")
	}
	if suite.WorkspaceID != wsID {
		t.Errorf("WorkspaceID mismatch: got %s, want %s", suite.WorkspaceID, wsID)
	}
	if suite.Name != "Support Suite" {
		t.Errorf("Name mismatch: got %s, want Support Suite", suite.Name)
	}
}

// TestSuiteService_GetByID_Success verifies fetching a suite by ID.
func TestSuiteService_GetByID_Success(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "a")
	svc := eval.NewSuiteService(db)

	created, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Test Suite",
		Domain:      "general",
		TestCases:   []eval.TestCase{{Input: "test", ExpectedKeywords: []string{"test"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.8, Exactitude: 0.85, Abstention: 0.95, Policy: 1.0},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	fetched, err := svc.GetByID(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", fetched.ID, created.ID)
	}
	if fetched.Name != created.Name {
		t.Errorf("Name mismatch: got %s, want %s", fetched.Name, created.Name)
	}
}

// TestSuiteService_GetByID_NotFound verifies error when suite doesn't exist.
func TestSuiteService_GetByID_NotFound(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "a")
	svc := eval.NewSuiteService(db)

	_, err := svc.GetByID(context.Background(), wsID, "nonexistent-id")
	if err == nil {
		t.Error("expected error for non-existent suite")
	}
}

// TestSuiteService_List_ReturnsOnlyWorkspace verifies suites are isolated by workspace.
func TestSuiteService_List_ReturnsOnlyWorkspace(t *testing.T) {
	db := mustOpenDB(t)
	wsA := mustCreateWorkspace(t, db, "a")
	wsB := mustCreateWorkspace(t, db, "b")
	svc := eval.NewSuiteService(db)

	// Create 2 suites in workspace A
	for i := 0; i < 2; i++ {
		_, err := svc.Create(context.Background(), eval.CreateSuiteInput{
			WorkspaceID: wsA,
			Name:        fmt.Sprintf("Suite A %d", i),
			Domain:      "general",
			TestCases:   []eval.TestCase{},
			Thresholds:  eval.Thresholds{},
		})
		if err != nil {
			t.Fatalf("Create wsA: %v", err)
		}
	}

	// Create 1 suite in workspace B
	_, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsB,
		Name:        "Suite B",
		Domain:      "general",
		TestCases:   []eval.TestCase{},
		Thresholds:  eval.Thresholds{},
	})
	if err != nil {
		t.Fatalf("Create wsB: %v", err)
	}

	listA, err := svc.List(context.Background(), wsA)
	if err != nil {
		t.Fatalf("List wsA: %v", err)
	}
	if len(listA) != 2 {
		t.Errorf("expected 2 suites in wsA, got %d", len(listA))
	}

	listB, err := svc.List(context.Background(), wsB)
	if err != nil {
		t.Fatalf("List wsB: %v", err)
	}
	if len(listB) != 1 {
		t.Errorf("expected 1 suite in wsB, got %d", len(listB))
	}
}

// TestSuiteService_Update_Success verifies updating a suite.
func TestSuiteService_Update_Success(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "a")
	svc := eval.NewSuiteService(db)

	created, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Original Name",
		Domain:      "general",
		TestCases:   []eval.TestCase{},
		Thresholds:  eval.Thresholds{},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = svc.Update(context.Background(), eval.UpdateSuiteInput{
		ID:          created.ID,
		WorkspaceID: wsID,
		Name:        "Updated Name",
		Domain:      "support",
		TestCases:   []eval.TestCase{},
		Thresholds:  eval.Thresholds{},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := svc.GetByID(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Name not updated: got %s", updated.Name)
	}
}

// TestSuiteService_Delete_Success verifies deleting a suite.
func TestSuiteService_Delete_Success(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "a")
	svc := eval.NewSuiteService(db)

	created, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "To Delete",
		Domain:      "general",
		TestCases:   []eval.TestCase{},
		Thresholds:  eval.Thresholds{},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	err = svc.Delete(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = svc.GetByID(context.Background(), wsID, created.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

// TestRunnerService_Run_PassedSuite verifies a passing eval run.
func TestRunnerService_Run_PassedSuite(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "a")
	suiteSvc := eval.NewSuiteService(db)
	runnerSvc := eval.NewRunnerService(db)

	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Passing Suite",
		Domain:      "support",
		TestCases: []eval.TestCase{
			{Input: "hello world", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false},
		},
		Thresholds: eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("Create suite: %v", err)
	}

	run, err := runnerSvc.Run(context.Background(), eval.RunInput{
		WorkspaceID: wsID,
		EvalSuiteID: suite.ID,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != "passed" {
		t.Errorf("expected status passed, got %s", run.Status)
	}
	if run.Scores.Exactitude <= 0 {
		t.Error("expected positive exactitude score")
	}
}

// TestRunnerService_Run_FailedSuite verifies a failing eval run.
func TestRunnerService_Run_FailedSuite(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "a")
	suiteSvc := eval.NewSuiteService(db)
	runnerSvc := eval.NewRunnerService(db)

	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Failing Suite",
		Domain:      "support",
		TestCases: []eval.TestCase{
			{Input: "completely different content", ExpectedKeywords: []string{"nonexistent_keyword_xyz"}, ShouldAbstain: false},
		},
		Thresholds: eval.Thresholds{Groundedness: 0.5, Exactitude: 1.0, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("Create suite: %v", err)
	}

	run, err := runnerSvc.Run(context.Background(), eval.RunInput{
		WorkspaceID: wsID,
		EvalSuiteID: suite.ID,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != "failed" {
		t.Errorf("expected status failed, got %s", run.Status)
	}
}

// TestRunnerService_ListRuns_ReturnsPaginated verifies paginated listing of runs.
func TestRunnerService_ListRuns_ReturnsPaginated(t *testing.T) {
	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "a")
	suiteSvc := eval.NewSuiteService(db)
	runnerSvc := eval.NewRunnerService(db)

	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Test Suite",
		Domain:      "general",
		TestCases:   []eval.TestCase{{Input: "test", ExpectedKeywords: []string{"test"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{},
	})
	if err != nil {
		t.Fatalf("Create suite: %v", err)
	}

	// Create 3 runs
	for i := 0; i < 3; i++ {
		_, err := runnerSvc.Run(context.Background(), eval.RunInput{
			WorkspaceID: wsID,
			EvalSuiteID: suite.ID,
		})
		if err != nil {
			t.Fatalf("Run %d: %v", i, err)
		}
	}

	runs, err := runnerSvc.ListRuns(context.Background(), wsID, 2, 0)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}
