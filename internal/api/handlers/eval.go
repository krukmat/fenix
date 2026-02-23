// Task 4.7 — FR-242: Eval Service Basic handler
package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	domaineval "github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

// EvalHandler wraps eval domain services for HTTP API.
// Task 4.7: FR-242
type EvalHandler struct {
	suites *domaineval.SuiteService
	runner *domaineval.RunnerService
}

// NewEvalHandler constructs an EvalHandler.
// Task 4.7: FR-242
func NewEvalHandler(suites *domaineval.SuiteService, runner *domaineval.RunnerService) *EvalHandler {
	return &EvalHandler{suites: suites, runner: runner}
}

// CreateSuiteRequest — request body for POST /admin/eval/suites
type CreateSuiteRequest struct {
	Name       string                `json:"name"`
	Domain     string                `json:"domain"`
	TestCases  []domaineval.TestCase `json:"test_cases"`
	Thresholds domaineval.Thresholds `json:"thresholds"`
}

func isCreateSuiteRequestValid(req CreateSuiteRequest) bool {
	return req.Name != "" && req.Domain != ""
}

// CreateSuite — POST /api/v1/admin/eval/suites
// Task 4.7: FR-242
func (h *EvalHandler) CreateSuite(w http.ResponseWriter, r *http.Request) {
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
	w.WriteHeader(http.StatusCreated)
	_ = writeJSONOr500(w, suite)
}

// ListSuites — GET /api/v1/admin/eval/suites
// Task 4.7: FR-242
func (h *EvalHandler) ListSuites(w http.ResponseWriter, r *http.Request) {
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
	EvalSuiteID     string  `json:"eval_suite_id"`
	PromptVersionID *string `json:"prompt_version_id,omitempty"`
}

func isRunEvalRequestValid(req RunEvalRequest) bool {
	return req.EvalSuiteID != ""
}

// RunEval — POST /api/v1/admin/eval/run
// Task 4.7: FR-242
func (h *EvalHandler) RunEval(w http.ResponseWriter, r *http.Request) {
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
	run, err := h.runner.Run(r.Context(), domaineval.RunInput{
		WorkspaceID:     wsID,
		EvalSuiteID:     req.EvalSuiteID,
		PromptVersionID: req.PromptVersionID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to run eval: %v", err))
		return
	}
	_ = writeJSONOr500(w, run)
}

// ListRuns — GET /api/v1/admin/eval/runs
// Task 4.7: FR-242
func (h *EvalHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
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