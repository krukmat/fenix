// Task 4.7 — FR-242: Eval Service Basic handler
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	domaineval "github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

// EvalHandler wraps eval domain services for HTTP API.
// Task 4.7: FR-242
type EvalHandler struct {
	suites     *domaineval.SuiteService
	runner     *domaineval.RunnerService
	benchmarks *domaineval.BenchmarkRegistryService
	authz      ActionAuthorizer
}

// NewEvalHandler constructs an EvalHandler.
// Task 4.7: FR-242
func NewEvalHandler(suites *domaineval.SuiteService, runner *domaineval.RunnerService) *EvalHandler {
	return &EvalHandler{suites: suites, runner: runner}
}

func NewEvalHandlerWithBenchmarkRegistry(
	suites *domaineval.SuiteService,
	runner *domaineval.RunnerService,
	benchmarks *domaineval.BenchmarkRegistryService,
) *EvalHandler {
	return &EvalHandler{suites: suites, runner: runner, benchmarks: benchmarks}
}

func NewEvalHandlerWithAuthorizer(
	suites *domaineval.SuiteService,
	runner *domaineval.RunnerService,
	benchmarks *domaineval.BenchmarkRegistryService,
	authz ActionAuthorizer,
) *EvalHandler {
	return &EvalHandler{suites: suites, runner: runner, benchmarks: benchmarks, authz: authz}
}

// CreateSuiteRequest — request body for POST /admin/eval/suites
type CreateSuiteRequest struct {
	Name       string                `json:"name"`
	Domain     string                `json:"domain"`
	TestCases  []domaineval.TestCase `json:"test_cases"`
	Thresholds domaineval.Thresholds `json:"thresholds"`
}

func isCreateSuiteRequestValid(req CreateSuiteRequest) bool {
	return req.Name != "" && req.Domain != "" && len(req.TestCases) > 0
}

// CreateSuite — POST /api/v1/admin/eval/suites
// Task 4.7: FR-242
func (h *EvalHandler) CreateSuite(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.eval.suites.create") {
		return
	}

	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	var req CreateSuiteRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	if !isCreateSuiteRequestValid(req) {
		writeError(w, http.StatusBadRequest, errEvalSuiteNameRequired)
		return
	}
	suite, err := h.suites.Create(r.Context(), domaineval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        req.Name,
		Domain:      req.Domain,
		TestCases:   req.TestCases,
		Thresholds:  req.Thresholds,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create eval suite: %v", err))
		return
	}
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = writeJSONOr500(w, suite)
}

// ListSuites — GET /api/v1/admin/eval/suites
// Task 4.7: FR-242
func (h *EvalHandler) ListSuites(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.eval.suites.list") {
		return
	}

	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	suites, err := h.suites.List(r.Context(), wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list eval suites: %v", err))
		return
	}
	_ = writeJSONOr500(w, map[string]any{"data": suites})
}

// GetSuite — GET /api/v1/admin/eval/suites/{id}
// Task 4.7: FR-242
func (h *EvalHandler) GetSuite(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.eval.suites.get") {
		return
	}

	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	suite, err := h.suites.GetByID(r.Context(), wsID, id)
	if handleGetError(w, err, errEvalSuiteNotFound, "failed to get eval suite: %v") {
		return
	}
	_ = writeJSONOr500(w, suite)
}

// RunEvalRequest — request body for POST /admin/eval/run
type RunEvalRequest struct {
	EvalSuiteID                string  `json:"eval_suite_id"`
	PromptVersionID            *string `json:"prompt_version_id,omitempty"`
	BenchmarkCaseID            *string `json:"benchmark_case_id,omitempty"`
	SyntheticOrgID             *string `json:"synthetic_org_id,omitempty"`
	SourceAgentRunID           *string `json:"source_agent_run_id,omitempty"`
	SourceCognitiveWorkspaceID *string `json:"source_cognitive_workspace_id,omitempty"`
	SourceTraceID              *string `json:"source_trace_id,omitempty"`
	ReplayMode                 *string `json:"replay_mode,omitempty"`
}

// CreateBenchmarkRequest is the request body for POST /admin/eval/benchmarks.
type CreateBenchmarkRequest struct {
	SyntheticOrgID  *string         `json:"synthetic_org_id,omitempty"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Domain          string          `json:"domain"`
	Version         int             `json:"version"`
	InputPayload    json.RawMessage `json:"input_payload"`
	ExpectedOutcome json.RawMessage `json:"expected_outcome"`
	Tags            []string        `json:"tags"`
}

func isRunEvalRequestValid(req RunEvalRequest) bool {
	return req.EvalSuiteID != ""
}

func isCreateBenchmarkRequestValid(req CreateBenchmarkRequest) bool {
	return req.Slug != "" && req.Name != "" && req.Domain != ""
}

func (req RunEvalRequest) provenance() *domaineval.ReplayProvenance {
	if !req.hasProvenanceFields() {
		return nil
	}
	return req.buildProvenance()
}

func (req RunEvalRequest) hasProvenanceFields() bool {
	return req.BenchmarkCaseID != nil ||
		req.SyntheticOrgID != nil ||
		req.SourceAgentRunID != nil ||
		req.SourceCognitiveWorkspaceID != nil ||
		req.SourceTraceID != nil ||
		req.ReplayMode != nil
}

func (req RunEvalRequest) buildProvenance() *domaineval.ReplayProvenance {
	provenance := &domaineval.ReplayProvenance{
		BenchmarkCaseID:            req.BenchmarkCaseID,
		SyntheticOrgID:             req.SyntheticOrgID,
		SourceAgentRunID:           req.SourceAgentRunID,
		SourceCognitiveWorkspaceID: req.SourceCognitiveWorkspaceID,
		SourceTraceID:              req.SourceTraceID,
	}
	if req.ReplayMode != nil {
		provenance.Mode = domaineval.ReplayMode(*req.ReplayMode)
	}
	return provenance
}

// CreateBenchmark — POST /api/v1/admin/eval/benchmarks
func (h *EvalHandler) CreateBenchmark(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.eval.benchmarks.create") {
		return
	}
	if h.benchmarks == nil {
		writeError(w, http.StatusInternalServerError, "benchmark registry unavailable")
		return
	}

	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	var req CreateBenchmarkRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	if !isCreateBenchmarkRequestValid(req) {
		writeError(w, http.StatusBadRequest, "slug, name, and domain are required")
		return
	}

	benchmarkCase, err := h.benchmarks.Create(r.Context(), domaineval.CreateBenchmarkCaseInput{
		WorkspaceID:     wsID,
		SyntheticOrgID:  req.SyntheticOrgID,
		Slug:            req.Slug,
		Name:            req.Name,
		Domain:          req.Domain,
		Version:         req.Version,
		InputPayload:    req.InputPayload,
		ExpectedOutcome: req.ExpectedOutcome,
		Tags:            req.Tags,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create benchmark case: %v", err))
		return
	}
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = writeJSONOr500(w, benchmarkCase)
}

// RunEval — POST /api/v1/admin/eval/run
// Task 4.7: FR-242
func (h *EvalHandler) RunEval(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.eval.run") {
		return
	}

	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	var req RunEvalRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	if !isRunEvalRequestValid(req) {
		writeError(w, http.StatusBadRequest, errEvalSuiteIDRequired)
		return
	}
	run, err := h.runEvalRequest(r.Context(), wsID, req)
	if err != nil {
		writeRunEvalError(w, err)
		return
	}
	_ = writeJSONOr500(w, run)
}

func (h *EvalHandler) runEvalRequest(
	ctx context.Context,
	workspaceID string,
	req RunEvalRequest,
) (*domaineval.Run, error) {
	if req.BenchmarkCaseID != nil {
		if h.benchmarks == nil {
			return nil, fmt.Errorf("benchmark registry unavailable")
		}
		run, runErr := h.benchmarks.RunBenchmarkCase(ctx, *req.BenchmarkCaseID, domaineval.RunBenchmarkCaseInput{
			WorkspaceID:     workspaceID,
			EvalSuiteID:     req.EvalSuiteID,
			PromptVersionID: req.PromptVersionID,
		})
		if runErr != nil {
			return nil, fmt.Errorf("run benchmark case: %w", runErr)
		}
		return run, nil
	}
	run, runErr := h.runner.Run(ctx, domaineval.RunInput{
		WorkspaceID:     workspaceID,
		EvalSuiteID:     req.EvalSuiteID,
		PromptVersionID: req.PromptVersionID,
		Provenance:      req.provenance(),
	})
	if runErr != nil {
		return nil, fmt.Errorf("run eval: %w", runErr)
	}
	return run, nil
}

// writeRunEvalError maps runner errors to HTTP status codes. (task-C2.4)
func writeRunEvalError(w http.ResponseWriter, err error) {
	var replayErr *domaineval.ReplaySourceError
	if errors.As(err, &replayErr) {
		writeError(w, http.StatusUnprocessableEntity, replayErr.Error())
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		if strings.Contains(err.Error(), "benchmark case") {
			writeError(w, http.StatusNotFound, errEvalBenchmarkNotFound)
			return
		}
		writeError(w, http.StatusNotFound, errEvalSuiteNotFound)
		return
	}
	writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to run eval: %v", err))
}

// ListRuns — GET /api/v1/admin/eval/runs
// Task 4.7: FR-242
func (h *EvalHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.eval.runs.list") {
		return
	}

	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	page := parsePaginationParams(r)
	runs, err := h.runner.ListRuns(r.Context(), wsID, page.Limit, page.Offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list eval runs: %v", err))
		return
	}
	_ = writeJSONOr500(w, map[string]any{"data": runs})
}

// GetRun — GET /api/v1/admin/eval/runs/{id}
// Task 4.7: FR-242
func (h *EvalHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.eval.runs.get") {
		return
	}

	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	run, err := h.runner.GetRun(r.Context(), wsID, id)
	if handleGetError(w, err, errEvalRunNotFound, "failed to get eval run: %v") {
		return
	}
	_ = writeJSONOr500(w, run)
}
