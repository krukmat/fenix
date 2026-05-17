package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// mustOpenDBWithMigrationsEval opens an in-memory DB with migrations.
// Task 4.7: FR-242
func mustOpenDBWithMigrationsEval(t *testing.T) *sql.DB {
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

// setupWorkspaceAndOwnerEval creates workspace and owner for testing.
// Task 4.7: FR-242
func setupWorkspaceAndOwnerEval(t *testing.T, db *sql.DB) (workspaceID, ownerID string) {
	t.Helper()
	wsID := "ws-eval-" + t.Name()
	ownerID = "user-eval-" + t.Name()
	_, err := db.Exec(
		`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		 VALUES (?, ?, ?, datetime('now'), datetime('now'))`,
		wsID, "Test Workspace", "slug-"+wsID,
	)
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'active', datetime('now'), datetime('now'))`,
		ownerID, wsID, "test@example.com", "Test User",
	)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return wsID, ownerID
}

// contextWithWorkspaceIDEval creates context with workspace ID.
// Task 4.7: FR-242
func contextWithWorkspaceIDEval(ctx context.Context, wsID string) context.Context {
	return context.WithValue(ctx, ctxkeys.WorkspaceID, wsID)
}

func contextWithUserIDEval(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxkeys.UserID, userID)
}

// TestEvalHandler_CreateSuite_201 verifies creating an eval suite.
func TestEvalHandler_CreateSuite_201(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))

	body := `{"name":"Test Suite","domain":"support","test_cases":[{"input":"hello","expected_keywords":["hello"],"should_abstain":false}],"thresholds":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/suites", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSuite(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["id"] == nil {
		t.Error("expected id in response")
	}
}

// TestEvalHandler_CreateSuite_400_MissingName verifies validation error.
func TestEvalHandler_CreateSuite_400_MissingName(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))

	body := `{"domain":"support"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/suites", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSuite(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestEvalHandler_CreateSuite_400_EmptyTestCases(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))

	body := `{"name":"Test Suite","domain":"support","test_cases":[],"thresholds":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/suites", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSuite(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestEvalHandler_ListSuites_200 verifies listing suites.
func TestEvalHandler_ListSuites_200(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	suiteSvc := eval.NewSuiteService(db)
	h := NewEvalHandler(suiteSvc, eval.NewRunnerService(db))

	// Create 2 suites
	for i := 0; i < 2; i++ {
		_, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
			WorkspaceID: wsID,
			Name:        "Suite " + string(rune('A'+i)),
			Domain:      "general",
			TestCases:   []eval.TestCase{},
			Thresholds:  eval.Thresholds{},
		})
		if err != nil {
			t.Fatalf("create suite: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/suites", nil)
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListSuites(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Errorf("expected 2 suites, got %d", len(data))
	}
}

// TestEvalHandler_GetSuite_200 verifies getting a suite.
func TestEvalHandler_GetSuite_200(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	suiteSvc := eval.NewSuiteService(db)
	h := NewEvalHandler(suiteSvc, eval.NewRunnerService(db))

	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Test Suite",
		Domain:      "general",
		TestCases:   []eval.TestCase{},
		Thresholds:  eval.Thresholds{},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	// Verify the suite was created by fetching directly
	fetched, fetchErr := suiteSvc.GetByID(context.Background(), wsID, suite.ID)
	if fetchErr != nil {
		t.Fatalf("direct fetch failed: %v, wsID: %s, suiteID: %s", fetchErr, wsID, suite.ID)
	}
	t.Logf("Direct fetch succeeded: %+v", fetched)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/suites/"+suite.ID, nil)
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", suite.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSuite(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s, wsID: %s, suiteID: %s", rr.Code, rr.Body.String(), wsID, suite.ID)
	}
}

// TestEvalHandler_GetSuite_404 verifies not found.
func TestEvalHandler_GetSuite_404(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/suites/nonexistent", nil)
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSuite(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestEvalHandler_RunEval_200 verifies running an eval.
func TestEvalHandler_RunEval_200(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	suiteSvc := eval.NewSuiteService(db)
	runner := eval.NewRunnerService(db)
	h := NewEvalHandlerWithBenchmarkRegistry(suiteSvc, runner, eval.NewBenchmarkRegistryService(db, runner))

	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Test Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	body := `{"eval_suite_id":"` + suite.ID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RunEval(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status"] != "passed" && resp["status"] != "failed" {
		t.Errorf("expected status passed or failed, got %v", resp["status"])
	}
}

// TestEvalHandler_ListRuns_200 verifies listing runs.
func TestEvalHandler_ListRuns_200(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	suiteSvc := eval.NewSuiteService(db)
	runnerSvc := eval.NewRunnerService(db)
	h := NewEvalHandler(suiteSvc, runnerSvc)

	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Test Suite",
		Domain:      "general",
		TestCases:   []eval.TestCase{{Input: "test", ExpectedKeywords: []string{"test"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	// Trigger run via service
	_, err = runnerSvc.Run(context.Background(), eval.RunInput{
		WorkspaceID: wsID,
		EvalSuiteID: suite.ID,
	})
	if err != nil {
		t.Fatalf("run eval: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/runs", nil)
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListRuns(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 run, got %d", len(data))
	}
}

// TestEvalHandler_GetRun_200 verifies getting a specific run.
func TestEvalHandler_GetRun_200(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	suiteSvc := eval.NewSuiteService(db)
	runnerSvc := eval.NewRunnerService(db)
	h := NewEvalHandler(suiteSvc, runnerSvc)

	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Test Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	run, err := runnerSvc.Run(context.Background(), eval.RunInput{
		WorkspaceID: wsID,
		EvalSuiteID: suite.ID,
	})
	if err != nil {
		t.Fatalf("run eval: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/runs/"+run.ID, nil)
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", run.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetRun(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["id"] != run.ID {
		t.Errorf("expected id %s, got %v", run.ID, resp["id"])
	}
}

// TestEvalHandler_GetRun_404 verifies not found for run.
func TestEvalHandler_GetRun_404(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/runs/nonexistent", nil)
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetRun(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestEvalHandler_MissingWorkspaceID_400 verifies workspace requirement.
func TestEvalHandler_MissingWorkspaceID_400(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/suites", nil)
	rr := httptest.NewRecorder()

	h.ListSuites(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestEvalHandler_CreateSuite_ForbiddenByAuthorizer(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, ownerID := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandlerWithAuthorizer(
		eval.NewSuiteService(db),
		eval.NewRunnerService(db),
		nil,
		&toolAuthzStub{allow: false},
	)

	body := `{"name":"Test Suite","domain":"support","test_cases":[],"thresholds":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/suites", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req = req.WithContext(contextWithUserIDEval(req.Context(), ownerID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSuite(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}
}

func TestEvalHandler_ListSuites_MissingUserIDWithAuthorizer_401(t *testing.T) {
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandlerWithAuthorizer(
		eval.NewSuiteService(db),
		eval.NewRunnerService(db),
		nil,
		&toolAuthzStub{allow: true},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/suites", nil)
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListSuites(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestEvalHandler_RunEval_MissingWorkspace_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(`{"eval_suite_id":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RunEval(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestEvalHandler_RunEval_MissingSuiteID_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(`{}`))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RunEval(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestEvalHandler_RunEval_SuiteNotFound_404(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(`{"eval_suite_id":"missing-suite"}`))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RunEval(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestEvalHandler_CreateBenchmark_201(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	runner := eval.NewRunnerService(db)
	h := NewEvalHandlerWithBenchmarkRegistry(eval.NewSuiteService(db), runner, eval.NewBenchmarkRegistryService(db, runner))

	body := `{"slug":"password-reset","name":"Password Reset","domain":"support","version":1,"input_payload":{"prompt":"reset"},"expected_outcome":{"status":"success"},"tags":["support","deterministic"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/benchmarks", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateBenchmark(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["slug"] != "password-reset" {
		t.Fatalf("expected slug password-reset, got %v", resp["slug"])
	}
}

func TestEvalHandler_RunEval_ServiceError_500(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))
	db.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(`{"eval_suite_id":"nonexistent"}`))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RunEval(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestEvalHandler_RunEval_BenchmarkCase_UsesRegistry(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	suiteSvc := eval.NewSuiteService(db)
	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Benchmark Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	runner := eval.NewRunnerService(db)
	benchmarkSvc := eval.NewBenchmarkRegistryService(db, runner)
	benchmark, err := benchmarkSvc.Create(context.Background(), eval.CreateBenchmarkCaseInput{
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
		t.Fatalf("create benchmark: %v", err)
	}

	h := NewEvalHandlerWithBenchmarkRegistry(suiteSvc, runner, benchmarkSvc)
	body := `{"eval_suite_id":"` + suite.ID + `","benchmark_case_id":"` + benchmark.ID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RunEval(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	provenance, ok := resp["provenance"].(map[string]any)
	if !ok {
		t.Fatalf("expected provenance object, got %#v", resp["provenance"])
	}
	if provenance["benchmarkCaseId"] != benchmark.ID {
		t.Fatalf("expected benchmarkCaseId %s, got %v", benchmark.ID, provenance["benchmarkCaseId"])
	}
}

func TestEvalHandler_RunEval_BenchmarkCaseNotFound_404(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	suiteSvc := eval.NewSuiteService(db)
	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Benchmark Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	runner := eval.NewRunnerService(db)
	h := NewEvalHandlerWithBenchmarkRegistry(suiteSvc, runner, eval.NewBenchmarkRegistryService(db, runner))
	body := `{"eval_suite_id":"` + suite.ID + `","benchmark_case_id":"missing-benchmark"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RunEval(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestEvalHandler_ListRuns_MissingWorkspace_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/runs", nil)
	rr := httptest.NewRecorder()
	h.ListRuns(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// TestEvalHandler_RunEval_ReplaySourceErrorMapped asserts that a replay source
// error from the runner is mapped to HTTP 422. (task-C2.4)
//
// Setup: agent_run exists so provenance FK passes, but reasoning_event/audit_event
// rows are absent — replay engine finds no trace, returning ReplaySourceError.
func TestEvalHandler_RunEval_ReplaySourceErrorMapped(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)

	// Insert agent_definition and agent_run so FK on agent_run is satisfied.
	mustInsertAgentDefinitionEval(t, db, wsID, "agent-def-1")
	mustInsertAgentRunWithTraceEval(t, db, wsID, "agent-run-1", "agent-def-1", "trace-missing")

	suiteSvc := eval.NewSuiteService(db)
	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Replay Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	runner := eval.NewRunnerServiceWithReplay(db, eval.NewSQLiteReplayEngine(db))
	h := NewEvalHandler(suiteSvc, runner)

	body := `{"eval_suite_id":"` + suite.ID + `","source_agent_run_id":"agent-run-1","source_trace_id":"trace-missing"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RunEval(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestEvalHandler_RunEval_BackwardCompatibleWithoutProvenance asserts that the
// handler returns 200 when no provenance fields are present. (task-C2.4)
func TestEvalHandler_RunEval_BackwardCompatibleWithoutProvenance(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	suiteSvc := eval.NewSuiteService(db)
	suite, err := suiteSvc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Legacy Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	h := NewEvalHandler(suiteSvc, eval.NewRunnerService(db))
	body := `{"eval_suite_id":"` + suite.ID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/eval/run", bytes.NewBufferString(body))
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RunEval(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["replay_artifact"] != nil {
		t.Fatal("expected replay_artifact to be absent on legacy path")
	}
}

func TestEvalHandler_ListRuns_ServiceError_500(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrationsEval(t)
	wsID, _ := setupWorkspaceAndOwnerEval(t, db)
	h := NewEvalHandler(eval.NewSuiteService(db), eval.NewRunnerService(db))
	db.Close()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/eval/runs", nil)
	req = req.WithContext(contextWithWorkspaceIDEval(req.Context(), wsID))
	rr := httptest.NewRecorder()
	h.ListRuns(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func mustInsertAgentDefinitionEval(t *testing.T, db *sql.DB, wsID, agentID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, allowed_tools, limits, trigger_config, status, created_at, updated_at)
		VALUES (?, ?, ?, 'support', '[]', '{}', '{}', 'active', datetime('now'), datetime('now'))`,
		agentID, wsID, "Agent "+agentID,
	); err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

func mustInsertAgentRunWithTraceEval(t *testing.T, db *sql.DB, wsID, runID, agentDefID, traceID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, trigger_type, status,
			trigger_context, inputs, retrieval_queries, retrieved_evidence_ids,
			reasoning_trace, tool_calls, output, trace_id,
			started_at, completed_at, created_at, updated_at, cognitive_workspace_id
		) VALUES (?, ?, ?, 'manual', 'success', ?, ?, ?, ?, ?, ?, ?, ?,
		          datetime('now'), datetime('now'), datetime('now'), datetime('now'), NULL)`,
		runID, wsID, agentDefID,
		`{"type":"case.updated"}`,
		`{"caseId":"case-1"}`,
		`["search cases"]`,
		`["ev-1"]`,
		`{"steps":["observe"]}`,
		`[{"tool":"crm.case.get","status":"executed","params":{"id":"case-1"}}]`,
		`{"summary":"done"}`,
		traceID,
	); err != nil {
		t.Fatalf("insert agent_run: %v", err)
	}
}
