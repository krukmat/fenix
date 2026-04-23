package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

type mockWorkflowService struct {
	items          map[string]*workflowdomain.Workflow
	updateCalls    int
	lastUpdate     workflowdomain.UpdateWorkflowInput
	createErr      error
	getErr         error
	listErr        error
	updateErr      error
	markTestingErr error
	markActiveErr  error
	newVersionErr  error
	rollbackErr    error
	deleteErr      error
}

func newMockWorkflowService() *mockWorkflowService {
	return &mockWorkflowService{items: make(map[string]*workflowdomain.Workflow)}
}

func (m *mockWorkflowService) Create(_ context.Context, input workflowdomain.CreateWorkflowInput) (*workflowdomain.Workflow, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	now := time.Now().UTC()
	item := &workflowdomain.Workflow{
		ID:          "wf_1",
		WorkspaceID: input.WorkspaceID,
		Name:        input.Name,
		Description: stringPtrToOptional(input.Description),
		DSLSource:   input.DSLSource,
		SpecSource:  stringPtrToOptional(input.SpecSource),
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.items[item.ID] = item
	return item, nil
}

func (m *mockWorkflowService) Get(_ context.Context, _, workflowID string) (*workflowdomain.Workflow, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	item, ok := m.items[workflowID]
	if !ok {
		return nil, workflowdomain.ErrWorkflowNotFound
	}
	return item, nil
}

func (m *mockWorkflowService) List(_ context.Context, _, _ string) ([]*workflowdomain.Workflow, error) {
	panic("unused")
}

func (m *mockWorkflowService) List2(_ context.Context, _ string, _ workflowdomain.ListWorkflowsInput) ([]*workflowdomain.Workflow, error) {
	panic("unused")
}

func (m *mockWorkflowService) Update(_ context.Context, _, workflowID string, input workflowdomain.UpdateWorkflowInput) (*workflowdomain.Workflow, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.updateCalls++
	m.lastUpdate = input
	item, ok := m.items[workflowID]
	if !ok {
		return nil, workflowdomain.ErrWorkflowNotFound
	}
	item.Description = stringPtrToOptional(input.Description)
	item.DSLSource = input.DSLSource
	item.SpecSource = stringPtrToOptional(input.SpecSource)
	item.UpdatedAt = time.Now().UTC()
	return item, nil
}

func (m *mockWorkflowService) ListVersions(_ context.Context, _, workflowID string) ([]*workflowdomain.Workflow, error) {
	item, ok := m.items[workflowID]
	if !ok {
		return nil, workflowdomain.ErrWorkflowNotFound
	}
	out := make([]*workflowdomain.Workflow, 0, len(m.items))
	for _, candidate := range m.items {
		if candidate.Name == item.Name {
			out = append(out, candidate)
		}
	}
	return out, nil
}

func (m *mockWorkflowService) MarkTesting(_ context.Context, _, workflowID string) (*workflowdomain.Workflow, error) {
	if m.markTestingErr != nil {
		return nil, m.markTestingErr
	}
	item, ok := m.items[workflowID]
	if !ok {
		return nil, workflowdomain.ErrWorkflowNotFound
	}
	item.Status = workflowdomain.StatusTesting
	item.UpdatedAt = time.Now().UTC()
	return item, nil
}

func (m *mockWorkflowService) MarkActive(_ context.Context, _, workflowID string) (*workflowdomain.Workflow, error) {
	if m.markActiveErr != nil {
		return nil, m.markActiveErr
	}
	item, ok := m.items[workflowID]
	if !ok {
		return nil, workflowdomain.ErrWorkflowNotFound
	}
	item.Status = workflowdomain.StatusActive
	item.UpdatedAt = time.Now().UTC()
	return item, nil
}

func (m *mockWorkflowService) Activate(_ context.Context, _, workflowID string) (*workflowdomain.Workflow, error) {
	return m.MarkActive(context.Background(), "", workflowID)
}

func (m *mockWorkflowService) NewVersion(_ context.Context, _, workflowID string) (*workflowdomain.Workflow, error) {
	if m.newVersionErr != nil {
		return nil, m.newVersionErr
	}
	item, ok := m.items[workflowID]
	if !ok {
		return nil, workflowdomain.ErrWorkflowNotFound
	}
	now := time.Now().UTC()
	parentID := item.ID
	nextID := workflowID + "_vnext"
	next := &workflowdomain.Workflow{
		ID:              nextID,
		WorkspaceID:     item.WorkspaceID,
		ParentVersionID: &parentID,
		Name:            item.Name,
		Description:     item.Description,
		DSLSource:       item.DSLSource,
		SpecSource:      item.SpecSource,
		Version:         item.Version + 1,
		Status:          workflowdomain.StatusDraft,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	m.items[nextID] = next
	return next, nil
}

func (m *mockWorkflowService) Rollback(_ context.Context, _, workflowID string) (*workflowdomain.Workflow, error) {
	if m.rollbackErr != nil {
		return nil, m.rollbackErr
	}
	item, ok := m.items[workflowID]
	if !ok {
		return nil, workflowdomain.ErrWorkflowNotFound
	}
	item.Status = workflowdomain.StatusActive
	item.UpdatedAt = time.Now().UTC()
	return item, nil
}

func (m *mockWorkflowService) DeleteDraft(_ context.Context, _, workflowID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.items[workflowID]; !ok {
		return workflowdomain.ErrWorkflowNotFound
	}
	delete(m.items, workflowID)
	return nil
}

func (m *mockWorkflowService) ListWorkflows(_ context.Context, _ string, _ workflowdomain.ListWorkflowsInput) ([]*workflowdomain.Workflow, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]*workflowdomain.Workflow, 0, len(m.items))
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}

func TestWorkflowHandler_Create_Returns201(t *testing.T) {
	mock := newMockWorkflowService()
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows", handler.Create)

	body, _ := json.Marshal(CreateWorkflowRequest{
		Name:      "qualify_lead",
		DSLSource: "ON lead.created",
	})
	req := httptest.NewRequest(http.MethodPost, "/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestWorkflowHandler_Create_Returns422(t *testing.T) {
	mock := newMockWorkflowService()
	mock.createErr = workflowdomain.ErrInvalidWorkflowInput
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows", handler.Create)

	body, _ := json.Marshal(CreateWorkflowRequest{Name: "qualify_lead", DSLSource: ""})
	req := httptest.NewRequest(http.MethodPost, "/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestWorkflowHandler_Get_Returns404(t *testing.T) {
	mock := newMockWorkflowService()
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/workflows/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/workflows/missing", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestWorkflowHandler_List_Returns200(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_1"] = &workflowdomain.Workflow{
		ID:          "wf_1",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/workflows", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestWorkflowHandler_Update_Returns409(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_1"] = &workflowdomain.Workflow{
		ID:          "wf_1",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      workflowdomain.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	mock.updateErr = workflowdomain.ErrWorkflowNotEditable
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Put("/workflows/{id}", handler.Update)

	body, _ := json.Marshal(UpdateWorkflowRequest{DSLSource: "ON lead.updated"})
	req := httptest.NewRequest(http.MethodPut, "/workflows/wf_1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestWorkflowHandler_Delete_Returns204(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_1"] = &workflowdomain.Workflow{
		ID:          "wf_1",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Delete("/workflows/{id}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/workflows/wf_1", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestWorkflowHandler_List_InvalidStatus_Returns400(t *testing.T) {
	mock := newMockWorkflowService()
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/workflows", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/workflows?status=broken", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestWorkflowHandler_Diff_ReturnsLayoutOnlyForWhitespaceChanges(t *testing.T) {
	t.Parallel()

	handler := NewWorkflowHandler(workflowServiceAdapter{newMockWorkflowService()})
	r := chi.NewRouter()
	r.Post("/workflows/diff", handler.Diff)

	body, _ := json.Marshal(WorkflowDiffRequest{
		Before: WorkflowDiffSource{DSLSource: "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"triaged\""},
		After:  WorkflowDiffSource{DSLSource: "\nWORKFLOW resolve_support_case\nON case.created\n\nSET case.status = \"triaged\""},
	})
	req := httptest.NewRequest(http.MethodPost, "/workflows/diff", bytes.NewReader(body))
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data WorkflowDiffResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.Diff.HasSemanticChanges {
		t.Fatalf("HasSemanticChanges = true, diff = %#v", payload.Data.Diff)
	}
	if !payload.Data.Diff.LayoutOnly {
		t.Fatalf("LayoutOnly = false, diff = %#v", payload.Data.Diff)
	}
}

func TestWorkflowHandler_Diff_ReturnsSemanticChanges(t *testing.T) {
	t.Parallel()

	handler := NewWorkflowHandler(workflowServiceAdapter{newMockWorkflowService()})
	r := chi.NewRouter()
	r.Post("/workflows/diff", handler.Diff)

	body, _ := json.Marshal(WorkflowDiffRequest{
		Before: WorkflowDiffSource{DSLSource: "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"triaged\""},
		After:  WorkflowDiffSource{DSLSource: "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\""},
	})
	req := httptest.NewRequest(http.MethodPost, "/workflows/diff", bytes.NewReader(body))
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data WorkflowDiffResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.Data.Diff.HasSemanticChanges {
		t.Fatalf("HasSemanticChanges = false, diff = %#v", payload.Data.Diff)
	}
	if payload.Data.Diff.LayoutOnly {
		t.Fatalf("LayoutOnly = true, diff = %#v", payload.Data.Diff)
	}
	if len(payload.Data.Diff.NodeChanges) == 0 {
		t.Fatalf("NodeChanges = empty, diff = %#v", payload.Data.Diff)
	}
}

func TestWorkflowHandler_Diff_Returns422ForInvalidSource(t *testing.T) {
	t.Parallel()

	handler := NewWorkflowHandler(workflowServiceAdapter{newMockWorkflowService()})
	r := chi.NewRouter()
	r.Post("/workflows/diff", handler.Diff)

	body, _ := json.Marshal(WorkflowDiffRequest{
		Before: WorkflowDiffSource{DSLSource: "ON case.created"},
		After:  WorkflowDiffSource{DSLSource: "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"triaged\""},
	})
	req := httptest.NewRequest(http.MethodPost, "/workflows/diff", bytes.NewReader(body))
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkflowHandler_ListVersions_Returns200(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_1"] = &workflowdomain.Workflow{
		ID:          "wf_1",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      workflowdomain.StatusArchived,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	mock.items["wf_2"] = &workflowdomain.Workflow{
		ID:          "wf_2",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.updated",
		Version:     2,
		Status:      workflowdomain.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/workflows/{id}/versions", handler.ListVersions)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf_1/versions", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkflowHandler_NewVersion_Returns200(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_1"] = &workflowdomain.Workflow{
		ID:          "wf_1",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      workflowdomain.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows/{id}/new-version", handler.NewVersion)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_1/new-version", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkflowHandler_Rollback_Returns200(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_1"] = &workflowdomain.Workflow{
		ID:          "wf_1",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      workflowdomain.StatusArchived,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	invalidator := &workflowCacheInvalidatorStub{}
	handler := NewWorkflowHandlerWithRuntime(workflowServiceAdapter{mock}, nil, nil, nil, nil, nil, nil, nil, invalidator)

	r := chi.NewRouter()
	r.Put("/workflows/{id}/rollback", handler.Rollback)

	req := httptest.NewRequest(http.MethodPut, "/workflows/wf_1/rollback", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(invalidator.invalidated) != 1 || invalidator.invalidated[0] != "wf_1" {
		t.Fatalf("unexpected invalidated workflows = %#v", invalidator.invalidated)
	}
}

func TestWorkflowHandler_Execute_Returns200(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	insertDSLWorkflowAgent(t, db, wsID, "dsl-agent-exec")
	insertExecutableWorkflow(t, db, wsID, "wf_exec_1", "dsl-agent-exec")

	toolRegistry := setupWorkflowToolRegistry(t, db, wsID)
	orch := agent.NewOrchestratorWithRegistry(db, agent.NewRunnerRegistry())
	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, orch, toolRegistry, nil, nil, nil, nil)

	r := chi.NewRouter()
	r.Post("/workflows/{id}/execute", handler.Execute)

	body, _ := json.Marshal(ExecuteWorkflowRequest{
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_exec_1/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := contextWithWorkspaceID(req.Context(), wsID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data["run"] == nil {
		t.Fatalf("expected run in response: %s", rr.Body.String())
	}
}

func TestWorkflowHandler_Verify_Returns200WithJudgeResult(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	spec := "CONTEXT\n  system = crm\nACTORS\n  admin\nBEHAVIOR resolve_support_case\n  GIVEN a workflow\nCONSTRAINTS\n  one active per name"
	mock.items["wf_verify_1"] = &workflowdomain.Workflow{
		ID:          "wf_verify_1",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource:  &spec,
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows/{id}/verify", handler.Verify)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_verify_1/verify", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data agent.JudgeResult `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.Data.Passed {
		t.Fatalf("expected Passed=true, got %+v", payload.Data)
	}
	if mock.items["wf_verify_1"].Status != workflowdomain.StatusTesting {
		t.Fatalf("status = %s, want %s", mock.items["wf_verify_1"].Status, workflowdomain.StatusTesting)
	}
}

func TestWorkflowHandler_Verify_Returns200WhenJudgeFindsViolations(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_verify_bad"] = &workflowdomain.Workflow{
		ID:          "wf_verify_bad",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "ON case.created\nSET case.status = \"resolved\"",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows/{id}/verify", handler.Verify)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_verify_bad/verify", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data agent.JudgeResult `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.Passed {
		t.Fatalf("expected Passed=false, got %+v", payload.Data)
	}
	if len(payload.Data.Violations) == 0 {
		t.Fatalf("expected violations, got %+v", payload.Data)
	}
	if mock.items["wf_verify_bad"].Status != workflowdomain.StatusDraft {
		t.Fatalf("status = %s, want %s", mock.items["wf_verify_bad"].Status, workflowdomain.StatusDraft)
	}
}

func TestWorkflowHandler_Verify_Returns404WhenWorkflowMissing(t *testing.T) {
	mock := newMockWorkflowService()
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows/{id}/verify", handler.Verify)

	req := httptest.NewRequest(http.MethodPost, "/workflows/missing/verify", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkflowHandler_Graph_Returns200WithGraphAndConformance(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT update_case`
	mock.items["wf_graph_1"] = &workflowdomain.Workflow{
		ID:          "wf_graph_1",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource:  &spec,
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/workflows/{id}/graph", handler.Graph)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf_graph_1/graph", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data WorkflowGraphResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.WorkflowID != "wf_graph_1" {
		t.Fatalf("WorkflowID = %q, want wf_graph_1", payload.Data.WorkflowID)
	}
	if payload.Data.Conformance.Profile != agent.ConformanceProfileSafe {
		t.Fatalf("Conformance.Profile = %q, want safe", payload.Data.Conformance.Profile)
	}
	if payload.Data.Conformance.Graph != nil {
		t.Fatalf("Conformance.Graph = %#v, want nil in graph endpoint response", payload.Data.Conformance.Graph)
	}
	if payload.Data.SemanticGraph == nil || len(payload.Data.SemanticGraph.Nodes) == 0 {
		t.Fatalf("SemanticGraph = %#v, want nodes", payload.Data.SemanticGraph)
	}
}

func TestWorkflowHandler_Graph_ReturnsVisualProjectionWhenFormatVisual(t *testing.T) { // CLSF-61
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_visual_1"] = &workflowdomain.Workflow{
		ID:          "wf_visual_1",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/workflows/{id}/graph", handler.Graph)

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf_visual_1/graph?format=visual", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data agent.WorkflowVisualProjection `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.WorkflowName == "" {
		t.Fatal("WorkflowName is empty, want non-empty")
	}
	if len(payload.Data.Nodes) == 0 {
		t.Fatal("Nodes is empty, want visual nodes")
	}
	if payload.Data.Nodes[0].Color == "" {
		t.Fatal("first node Color is empty, want a color")
	}
	if payload.Data.Conformance.Profile == "" {
		t.Fatal("Conformance.Profile is empty")
	}
}

func TestWorkflowHandler_Graph_Returns404WhenWorkflowMissing(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/workflows/{id}/graph", handler.Graph)

	req := httptest.NewRequest(http.MethodGet, "/workflows/missing/graph", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkflowHandler_Validate_Returns200WhenWorkflowPasses(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT update_case`
	mock.items["wf_validate_1"] = &workflowdomain.Workflow{
		ID:          "wf_validate_1",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource:  &spec,
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows/{id}/validate", handler.Validate)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_validate_1/validate", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data WorkflowValidateResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.Data.Passed {
		t.Fatalf("Passed = false, response = %+v", payload.Data)
	}
	if payload.Data.Conformance.Profile != agent.ConformanceProfileSafe {
		t.Fatalf("Conformance.Profile = %q, want safe", payload.Data.Conformance.Profile)
	}
	if payload.Data.Conformance.Graph != nil {
		t.Fatalf("Conformance.Graph = %#v, want nil in validate endpoint response", payload.Data.Conformance.Graph)
	}
}

func TestWorkflowHandler_Validate_Returns422WithDiagnosticsWhenJudgeFails(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT send_reply`
	mock.items["wf_validate_bad"] = &workflowdomain.Workflow{
		ID:          "wf_validate_bad",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"",
		SpecSource:  &spec,
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows/{id}/validate", handler.Validate)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_validate_bad/validate", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data WorkflowValidateResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.Passed {
		t.Fatalf("Passed = true, response = %+v", payload.Data)
	}
	if !workflowValidationHasViolation(payload.Data, "tool_not_permitted") {
		t.Fatalf("Violations = %#v, want tool_not_permitted", payload.Data.Diagnostics.Violations)
	}
	if payload.Data.Conformance.Profile != agent.ConformanceProfileSafe {
		t.Fatalf("Conformance.Profile = %q, want safe", payload.Data.Conformance.Profile)
	}
}

func TestWorkflowHandler_Validate_Returns404WhenWorkflowMissing(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows/{id}/validate", handler.Validate)

	req := httptest.NewRequest(http.MethodPost, "/workflows/missing/validate", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkflowHandler_Preview_ReturnsVisualProjectionFromDraftSource(t *testing.T) {
	t.Parallel()

	handler := NewWorkflowHandler(workflowServiceAdapter{newMockWorkflowService()})
	r := chi.NewRouter()
	r.Post("/workflows/preview", handler.Preview)

	body := strings.NewReader(`{"dsl_source":"WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"","spec_source":"CARTA resolve_support_case\nAGENT search_knowledge\n  PERMIT update_case"}`)
	req := httptest.NewRequest(http.MethodPost, "/workflows/preview", body)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data WorkflowPreviewResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.Data.Passed {
		t.Fatalf("Passed = false, response = %+v", payload.Data)
	}
	if payload.Data.VisualGraph.WorkflowName != "resolve_support_case" {
		t.Fatalf("VisualGraph.WorkflowName = %q, want resolve_support_case", payload.Data.VisualGraph.WorkflowName)
	}
	if len(payload.Data.VisualGraph.Nodes) == 0 {
		t.Fatal("VisualGraph.Nodes is empty, want renderable nodes")
	}
	if payload.Data.Conformance.Graph != nil || payload.Data.VisualGraph.Conformance.Graph != nil {
		t.Fatal("preview response should not embed semantic graph inside conformance")
	}
}

func TestWorkflowHandler_Preview_Returns422WithDiagnosticsForInvalidSource(t *testing.T) {
	t.Parallel()

	handler := NewWorkflowHandler(workflowServiceAdapter{newMockWorkflowService()})
	r := chi.NewRouter()
	r.Post("/workflows/preview", handler.Preview)

	req := httptest.NewRequest(http.MethodPost, "/workflows/preview", strings.NewReader(`{"dsl_source":"WORKFLOW"}`))
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data WorkflowPreviewResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.Passed {
		t.Fatalf("Passed = true, response = %+v", payload.Data)
	}
	if len(payload.Data.Diagnostics.Violations) == 0 {
		t.Fatalf("Violations is empty, response = %+v", payload.Data)
	}
	if payload.Data.Conformance.Profile != agent.ConformanceProfileInvalid {
		t.Fatalf("Conformance.Profile = %q, want invalid", payload.Data.Conformance.Profile)
	}
}

func TestWorkflowHandler_VisualAuthoring_PersistsGeneratedSourcesWhenValidationPasses(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_visual"] = &workflowdomain.Workflow{
		ID:          "wf_visual",
		WorkspaceID: "ws_test",
		Name:        "old_workflow",
		Description: stringPtrToOptional("keep description"),
		DSLSource:   "WORKFLOW old_workflow\nON lead.created\nSET status = \"open\"",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})
	r := chi.NewRouter()
	r.Post("/workflows/{id}/visual-authoring", handler.VisualAuthoring)

	body, _ := json.Marshal(WorkflowVisualAuthoringRequest{Graph: validWorkflowVisualAuthoringGraph("sales_followup")})
	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_visual/visual-authoring", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.updateCalls != 1 {
		t.Fatalf("Update calls = %d, want 1", mock.updateCalls)
	}
	if !strings.Contains(mock.lastUpdate.DSLSource, "WORKFLOW sales_followup") {
		t.Fatalf("DSLSource = %q, want generated workflow source", mock.lastUpdate.DSLSource)
	}
	if mock.lastUpdate.SpecSource != "" {
		t.Fatalf("SpecSource = %q, want empty spec for graph without governance", mock.lastUpdate.SpecSource)
	}

	var payload struct {
		Data WorkflowResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.ID != "wf_visual" || payload.Data.DSLSource != mock.lastUpdate.DSLSource {
		t.Fatalf("unexpected response data = %+v, update = %+v", payload.Data, mock.lastUpdate)
	}
}

func TestWorkflowHandler_VisualAuthoring_Returns422AndDoesNotPersistInvalidVisualGraph(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_visual"] = &workflowdomain.Workflow{
		ID:          "wf_visual",
		WorkspaceID: "ws_test",
		Name:        "sales_followup",
		DSLSource:   "WORKFLOW sales_followup\nON lead.created\nSET lead.status = \"new\"",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})
	r := chi.NewRouter()
	r.Post("/workflows/{id}/visual-authoring", handler.VisualAuthoring)

	graph := validWorkflowVisualAuthoringGraph("sales_followup")
	graph.Nodes = graph.Nodes[:1]
	body, _ := json.Marshal(WorkflowVisualAuthoringRequest{Graph: graph})
	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_visual/visual-authoring", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.updateCalls != 0 {
		t.Fatalf("Update calls = %d, want 0", mock.updateCalls)
	}

	var payload struct {
		Data WorkflowValidateResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.Passed {
		t.Fatalf("Passed = true, response = %+v", payload.Data)
	}
	if !workflowValidationHasViolation(payload.Data, "visual_trigger_missing") {
		t.Fatalf("Violations = %#v, want visual_trigger_missing", payload.Data.Diagnostics.Violations)
	}
}

func TestWorkflowHandler_VisualAuthoring_Returns422AndDoesNotPersistWhenGeneratedSourceFails(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_visual"] = &workflowdomain.Workflow{
		ID:          "wf_visual",
		WorkspaceID: "ws_test",
		Name:        "branching_workflow",
		DSLSource:   "WORKFLOW branching_workflow\nON lead.created\nSET lead.status = \"new\"",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})
	r := chi.NewRouter()
	r.Post("/workflows/{id}/visual-authoring", handler.VisualAuthoring)

	graph := validWorkflowVisualAuthoringGraph("branching_workflow")
	graph.Nodes = append(graph.Nodes, agent.NewVisualAuthoringNode("decision", agent.SemanticNodeDecision, "deal.value > 1000", agent.WorkflowVisualPosition{Y: 320}))
	body, _ := json.Marshal(WorkflowVisualAuthoringRequest{Graph: graph})
	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_visual/visual-authoring", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.updateCalls != 0 {
		t.Fatalf("Update calls = %d, want 0", mock.updateCalls)
	}
}

func TestWorkflowHandler_Verify_DoesNotPromoteNonDraftWorkflow(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	spec := "CONTEXT\n  system = crm\nACTORS\n  admin\nBEHAVIOR resolve_support_case\n  GIVEN a workflow\nCONSTRAINTS\n  one active per name"
	mock.items["wf_verify_testing"] = &workflowdomain.Workflow{
		ID:          "wf_verify_testing",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource:  &spec,
		Version:     1,
		Status:      workflowdomain.StatusTesting,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Post("/workflows/{id}/verify", handler.Verify)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_verify_testing/verify", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.items["wf_verify_testing"].Status != workflowdomain.StatusTesting {
		t.Fatalf("status = %s, want %s", mock.items["wf_verify_testing"].Status, workflowdomain.StatusTesting)
	}
}

func TestWorkflowHandler_Activate_Returns200AndPromotesTestingWorkflow(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	spec := "CONTEXT\n  system = crm\nACTORS\n  admin\nBEHAVIOR resolve_support_case\n  GIVEN a workflow\nCONSTRAINTS\n  one active per name"
	mock.items["wf_activate_1"] = &workflowdomain.Workflow{
		ID:          "wf_activate_1",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource:  &spec,
		Version:     1,
		Status:      workflowdomain.StatusTesting,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	invalidator := &workflowCacheInvalidatorStub{}
	handler := NewWorkflowHandlerWithRuntime(workflowServiceAdapter{mock}, nil, nil, nil, nil, nil, nil, nil, invalidator)

	r := chi.NewRouter()
	r.Put("/workflows/{id}/activate", handler.Activate)

	req := httptest.NewRequest(http.MethodPut, "/workflows/wf_activate_1/activate", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.items["wf_activate_1"].Status != workflowdomain.StatusActive {
		t.Fatalf("status = %s, want %s", mock.items["wf_activate_1"].Status, workflowdomain.StatusActive)
	}
	if len(invalidator.invalidated) != 1 || invalidator.invalidated[0] != "wf_activate_1" {
		t.Fatalf("unexpected invalidated workflows = %#v", invalidator.invalidated)
	}
}

func TestWorkflowHandler_Activate_ReverifyBlocksInvalidWorkflow(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_activate_invalid"] = &workflowdomain.Workflow{
		ID:          "wf_activate_invalid",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "ON case.created\nSET case.status = \"resolved\"",
		Version:     1,
		Status:      workflowdomain.StatusTesting,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	invalidator := &workflowCacheInvalidatorStub{}
	handler := NewWorkflowHandlerWithRuntime(workflowServiceAdapter{mock}, nil, nil, nil, nil, nil, nil, nil, invalidator)

	r := chi.NewRouter()
	r.Put("/workflows/{id}/activate", handler.Activate)

	req := httptest.NewRequest(http.MethodPut, "/workflows/wf_activate_invalid/activate", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data agent.JudgeResult `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.Passed {
		t.Fatalf("expected Passed=false, got %+v", payload.Data)
	}
	if mock.items["wf_activate_invalid"].Status != workflowdomain.StatusTesting {
		t.Fatalf("status = %s, want %s", mock.items["wf_activate_invalid"].Status, workflowdomain.StatusTesting)
	}
	if len(invalidator.invalidated) != 0 {
		t.Fatalf("expected no invalidation, got %#v", invalidator.invalidated)
	}
}

func TestWorkflowHandler_Activate_RejectsNonTestingWorkflow(t *testing.T) {
	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_activate_draft"] = &workflowdomain.Workflow{
		ID:          "wf_activate_draft",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Put("/workflows/{id}/activate", handler.Activate)

	req := httptest.NewRequest(http.MethodPut, "/workflows/wf_activate_draft/activate", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.items["wf_activate_draft"].Status != workflowdomain.StatusDraft {
		t.Fatalf("status = %s, want %s", mock.items["wf_activate_draft"].Status, workflowdomain.StatusDraft)
	}
}

func TestWorkflowHandler_Execute_RejectsWorkflowWithoutAgentDefinition(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO workflow (id, workspace_id, name, dsl_source, version, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, 'draft', ?, ?)
	`, "wf_no_agent", wsID, "orphan_workflow", "WORKFLOW orphan_workflow\nON case.created\nSET case.status = \"resolved\"", now, now); err != nil {
		t.Fatalf("insert workflow: %v", err)
	}

	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, agent.NewOrchestratorWithRegistry(db, agent.NewRunnerRegistry()), nil, nil, nil, nil, nil)
	r := chi.NewRouter()
	r.Post("/workflows/{id}/execute", handler.Execute)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_no_agent/execute", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkflowHandler_Execute_RejectsNonActiveWorkflow(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	insertDSLWorkflowAgent(t, db, wsID, "dsl-agent-non-active")
	if _, err := db.Exec(`
		INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
		VALUES (?, ?, ?, 'resolve_support_case', ?, 1, 'draft', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, "wf_exec_draft", wsID, "dsl-agent-non-active", "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\""); err != nil {
		t.Fatalf("insert workflow: %v", err)
	}

	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, agent.NewOrchestratorWithRegistry(db, agent.NewRunnerRegistry()), nil, nil, nil, nil, nil)
	r := chi.NewRouter()
	r.Post("/workflows/{id}/execute", handler.Execute)

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_exec_draft/execute", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkflowHandler_Update_InvalidatesDSLRunnerCache(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO workflow (id, workspace_id, name, dsl_source, version, status, created_at, updated_at)
		VALUES (?, ?, 'qualify_lead', ?, 1, 'draft', ?, ?)
	`, "wf_cache_invalidate", wsID, "WORKFLOW qualify_lead\nON lead.created\nSET case.status = \"open\"", now, now); err != nil {
		t.Fatalf("insert workflow: %v", err)
	}

	invalidator := &workflowCacheInvalidatorStub{}
	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, nil, nil, nil, nil, nil, invalidator)
	r := chi.NewRouter()
	r.Put("/workflows/{id}", handler.Update)

	body, _ := json.Marshal(UpdateWorkflowRequest{
		DSLSource: "WORKFLOW qualify_lead\nON lead.created\nNOTIFY contact WITH \"updated\"",
	})
	req := httptest.NewRequest(http.MethodPut, "/workflows/wf_cache_invalidate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(invalidator.invalidated) != 1 || invalidator.invalidated[0] != "wf_cache_invalidate" {
		t.Fatalf("unexpected invalidated workflows = %#v", invalidator.invalidated)
	}
}

func withWorkflowContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	return ctx
}

func stringPtrToOptional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func workflowValidationHasViolation(response WorkflowValidateResponse, code string) bool {
	for _, violation := range response.Diagnostics.Violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}

func validWorkflowVisualAuthoringGraph(name string) *agent.VisualAuthoringGraph {
	graph := agent.NewVisualAuthoringGraph(name)
	graph.AddNode(agent.NewVisualAuthoringNode("workflow", agent.SemanticNodeWorkflow, name, agent.WorkflowVisualPosition{}))
	trigger := agent.NewVisualAuthoringNode("trigger", agent.SemanticNodeTrigger, "lead.created", agent.WorkflowVisualPosition{X: 260})
	trigger.Data.Event = "lead.created"
	graph.AddNode(trigger)
	action := agent.NewVisualAuthoringNode("set-status", agent.SemanticNodeAction, "lead.status", agent.WorkflowVisualPosition{Y: 160})
	action.Data.Target = "lead.status"
	action.Data.Value = "qualified"
	graph.AddNode(action)
	graph.AddEdge(agent.NewVisualAuthoringEdge("workflow-trigger", "workflow", "trigger", agent.SemanticEdgeContains))
	graph.AddEdge(agent.NewVisualAuthoringEdge("trigger-action", "trigger", "set-status", agent.SemanticEdgeNext))
	return graph
}

type workflowServiceAdapter struct{ *mockWorkflowService }

func (a workflowServiceAdapter) List(ctx context.Context, workspaceID string, input workflowdomain.ListWorkflowsInput) ([]*workflowdomain.Workflow, error) {
	return a.mockWorkflowService.ListWorkflows(ctx, workspaceID, input)
}

type workflowStubToolExecutor struct {
	result json.RawMessage
}

type workflowCacheInvalidatorStub struct {
	invalidated []string
}

func (s *workflowCacheInvalidatorStub) InvalidateCache(workflowID string) {
	s.invalidated = append(s.invalidated, workflowID)
}

func (s workflowStubToolExecutor) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	return s.result, nil
}

func setupWorkflowToolRegistry(t testing.TB, db *sql.DB, wsID string) *tool.ToolRegistry {
	t.Helper()

	registry := tool.NewToolRegistry(db)
	if _, err := registry.CreateToolDefinition(context.Background(), tool.CreateToolDefinitionInput{
		WorkspaceID: wsID,
		Name:        tool.BuiltinUpdateCase,
		InputSchema: json.RawMessage(`{"type":"object","required":["case_id"],"properties":{"case_id":{"type":"string"},"status":{"type":"string"}},"additionalProperties":false}`),
	}); err != nil {
		t.Fatalf("CreateToolDefinition(update_case): %v", err)
	}
	if err := registry.Register(tool.BuiltinUpdateCase, workflowStubToolExecutor{result: json.RawMessage(`{"status":"updated"}`)}); err != nil {
		t.Fatalf("Register(update_case): %v", err)
	}
	return registry
}

func insertDSLWorkflowAgent(t testing.TB, db *sql.DB, wsID, agentID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES (?, ?, ?, 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, agentID, wsID, agentID); err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

func insertExecutableWorkflow(t testing.TB, db *sql.DB, wsID, workflowID, agentID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
		VALUES (?, ?, ?, 'resolve_support_case', ?, 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, workflowID, wsID, agentID, "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\""); err != nil {
		t.Fatalf("insert workflow: %v", err)
	}
}

func TestFormatOptionalWorkflowTime_NonNil(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	got := formatOptionalWorkflowTime(&ts)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if *got == "" {
		t.Fatal("expected non-empty formatted time")
	}
}

func TestFormatOptionalWorkflowTime_Nil(t *testing.T) {
	t.Parallel()

	if formatOptionalWorkflowTime(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestNewWorkflowHandlerWithAuthorizer_NotNil(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	h := NewWorkflowHandlerWithAuthorizer(workflowServiceAdapter{mock}, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestWriteWorkflowExecuteError_StatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err        error
		wantStatus int
	}{
		{workflowdomain.ErrWorkflowNotFound, http.StatusNotFound},
		{agent.ErrDSLWorkflowNotFound, http.StatusNotFound},
		{workflowdomain.ErrInvalidWorkflowInput, http.StatusUnprocessableEntity},
		{agent.ErrInvalidTriggerType, http.StatusUnprocessableEntity},
	}

	for _, tc := range tests {
		w := httptest.NewRecorder()
		writeWorkflowExecuteError(w, tc.err)
		if w.Code != tc.wantStatus {
			t.Errorf("writeWorkflowExecuteError(%v): status = %d, want %d", tc.err, w.Code, tc.wantStatus)
		}
	}
}

func TestDecodeWorkflowListInput_ParsesNameAndStatus(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/workflows?name=pilot&status=active", nil)
	input, err := decodeWorkflowListInput(req)
	if err != nil {
		t.Fatalf("decodeWorkflowListInput() error = %v", err)
	}
	if input.Name != "pilot" {
		t.Fatalf("Name = %q, want pilot", input.Name)
	}
	if input.Status == nil || *input.Status != workflowdomain.StatusActive {
		t.Fatalf("Status = %#v, want active", input.Status)
	}
}

func TestDecodeWorkflowListInput_InvalidStatus(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/workflows?status=weird", nil)
	if _, err := decodeWorkflowListInput(req); err == nil {
		t.Fatal("decodeWorkflowListInput() expected error")
	}
}

func TestDecodeOptionalWorkflowExecuteBodyAndNormalize(t *testing.T) {
	t.Parallel()

	var reqBody ExecuteWorkflowRequest
	req := httptest.NewRequest(http.MethodPost, "/workflows/wf/execute", nil)
	rr := httptest.NewRecorder()
	if !decodeOptionalWorkflowExecuteBody(rr, req, &reqBody) {
		t.Fatal("decodeOptionalWorkflowExecuteBody(nil body) expected true")
	}
	if got := normalizeOptionalJSONObject(nil); string(got) != errEmptyJSON {
		t.Fatalf("normalizeOptionalJSONObject(nil) = %s", string(got))
	}
	if got := normalizeOptionalJSONObject(json.RawMessage(`{"x":1}`)); string(got) != `{"x":1}` {
		t.Fatalf("normalizeOptionalJSONObject(raw) = %s", string(got))
	}
}

func TestStaticWorkflowResolverAndWorkflowToResponse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	agentID := "agent-1"
	parentID := "parent-1"
	createdBy := "user-1"
	item := &workflowdomain.Workflow{
		ID:                "wf-1",
		WorkspaceID:       "ws-1",
		AgentDefinitionID: &agentID,
		ParentVersionID:   &parentID,
		Name:              "qualify_lead",
		Description:       testStringPtr("desc"),
		DSLSource:         "WORKFLOW qualify_lead\nON lead.created\nSET case.status = \"open\"",
		SpecSource:        testStringPtr("spec"),
		Version:           2,
		Status:            workflowdomain.StatusActive,
		CreatedByUserID:   &createdBy,
		ArchivedAt:        &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	resolver := staticWorkflowResolver{workflow: item}
	got, err := resolver.Get(context.Background(), "ws-1", "wf-1")
	if err != nil || got.ID != "wf-1" {
		t.Fatalf("Get() = %#v, %v", got, err)
	}
	got, err = resolver.GetActiveByAgentDefinition(context.Background(), "ws-1", "agent-1")
	if err != nil || got.ID != "wf-1" {
		t.Fatalf("GetActiveByAgentDefinition() = %#v, %v", got, err)
	}
	if _, err := resolver.Get(context.Background(), "ws-2", "wf-1"); !errors.Is(err, workflowdomain.ErrWorkflowNotFound) {
		t.Fatalf("Get(mismatch) err = %v", err)
	}

	resp := workflowToResponse(item)
	if resp == nil || resp.ID != "wf-1" || resp.Status != string(workflowdomain.StatusActive) {
		t.Fatalf("workflowToResponse() = %#v", resp)
	}
	if workflowToResponse(nil) != nil {
		t.Fatal("workflowToResponse(nil) expected nil")
	}
}

func TestWriteWorkflowErrorAndValidateExecution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err        error
		wantStatus int
	}{
		{workflowdomain.ErrWorkflowNotFound, http.StatusNotFound},
		{workflowdomain.ErrInvalidWorkflowInput, http.StatusUnprocessableEntity},
		{workflowdomain.ErrWorkflowNameConflict, http.StatusConflict},
		{errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		rr := httptest.NewRecorder()
		writeWorkflowError(rr, tc.err)
		if rr.Code != tc.wantStatus {
			t.Fatalf("writeWorkflowError(%v) = %d, want %d", tc.err, rr.Code, tc.wantStatus)
		}
	}

	if err := validateWorkflowForExecution(&workflowdomain.Workflow{Status: workflowdomain.StatusActive}); err == nil {
		t.Fatal("validateWorkflowForExecution() expected missing agent definition error")
	}
	agentID := "agent-1"
	if err := validateWorkflowForExecution(&workflowdomain.Workflow{AgentDefinitionID: &agentID, Status: workflowdomain.StatusDraft}); err == nil {
		t.Fatal("validateWorkflowForExecution() expected active status error")
	}
	if err := validateWorkflowForExecution(&workflowdomain.Workflow{AgentDefinitionID: &agentID, Status: workflowdomain.StatusActive}); err != nil {
		t.Fatalf("validateWorkflowForExecution(valid) error = %v", err)
	}
}

func TestWorkflowHandlerJudgeHelpers(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_judge"] = &workflowdomain.Workflow{
		ID:          "wf_judge",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		Version:     1,
		Status:      workflowdomain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/workflows/{id}", func(w http.ResponseWriter, r *http.Request) {
		workspaceID, id, item, ok := handler.loadWorkflowForJudge(w, r)
		if !ok {
			return
		}
		result, ok := handler.verifyWorkflowForJudge(w, r, item)
		if !ok {
			return
		}
		if !handler.promoteVerifiedDraftWorkflow(w, r, workspaceID, id, item, result) {
			return
		}
		handler.writeJudgeResult(w, result)
	})

	req := httptest.NewRequest(http.MethodGet, "/workflows/wf_judge", nil)
	req = req.WithContext(withWorkflowContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.items["wf_judge"].Status != workflowdomain.StatusTesting {
		t.Fatalf("status = %s, want testing", mock.items["wf_judge"].Status)
	}
}

func TestWorkflowHandlerActivationAndResponseHelpers(t *testing.T) {
	t.Parallel()

	mock := newMockWorkflowService()
	now := time.Now().UTC()
	mock.items["wf_activate_helper"] = &workflowdomain.Workflow{
		ID:          "wf_activate_helper",
		WorkspaceID: "ws_test",
		Name:        "resolve_support_case",
		DSLSource:   "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		Version:     1,
		Status:      workflowdomain.StatusTesting,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	invalidator := &workflowCacheInvalidatorStub{}
	handler := NewWorkflowHandlerWithRuntime(workflowServiceAdapter{mock}, nil, nil, nil, nil, nil, nil, nil, invalidator)

	if !validateWorkflowForActivation(httptest.NewRecorder(), mock.items["wf_activate_helper"]) {
		t.Fatal("validateWorkflowForActivation(testing) expected true")
	}

	rr := httptest.NewRecorder()
	out, ok := handler.activateVerifiedWorkflow(rr, httptest.NewRequest(http.MethodPut, "/", nil), "ws_test", "wf_activate_helper")
	if !ok || out == nil || out.Status != workflowdomain.StatusActive {
		t.Fatalf("activateVerifiedWorkflow() = %#v, %v", out, ok)
	}
	if len(invalidator.invalidated) != 1 || invalidator.invalidated[0] != "wf_activate_helper" {
		t.Fatalf("unexpected invalidation = %#v", invalidator.invalidated)
	}

	rr = httptest.NewRecorder()
	handler.writeWorkflowResponse(rr, out)
	if rr.Code != http.StatusOK {
		t.Fatalf("writeWorkflowResponse() status = %d", rr.Code)
	}

	conflict := httptest.NewRecorder()
	handler.writeJudgeConflict(conflict, &agent.JudgeResult{Passed: false})
	if conflict.Code != http.StatusConflict {
		t.Fatalf("writeJudgeConflict() status = %d", conflict.Code)
	}
}

func TestWorkflowHandlerRuntimeChecksAndExecuteHelpers(t *testing.T) {
	t.Parallel()

	if NewWorkflowHandler(workflowServiceAdapter{newMockWorkflowService()}).isRuntimeConfigured() {
		t.Fatal("isRuntimeConfigured() expected false without runtime")
	}

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	if _, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES ('user_test', ?, 'user_test@example.com', 'User Test', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, wsID); err != nil {
		t.Fatalf("insert user_account: %v", err)
	}
	insertDSLWorkflowAgent(t, db, wsID, "dsl-agent-exec")
	insertExecutableWorkflow(t, db, wsID, "wf_exec_helper", "dsl-agent-exec")
	if _, err := db.Exec(`
		INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at)
		VALUES ('case-1', ?, 'owner-1', 'helper case', 'medium', 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, wsID); err != nil {
		t.Fatalf("insert case_ticket: %v", err)
	}
	registry := agent.NewRunnerRegistry()
	orch := agent.NewOrchestratorWithRegistry(db, registry)
	toolRegistry := setupWorkflowToolRegistry(t, db, wsID)
	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, orch, toolRegistry, nil, nil, nil, nil)

	if !handler.isRuntimeConfigured() {
		t.Fatal("isRuntimeConfigured() expected true")
	}

	req := httptest.NewRequest(http.MethodPost, "/workflows/wf_exec_helper/execute", bytes.NewReader([]byte(`{}`)))
	req = req.WithContext(contextWithWorkspaceID(context.WithValue(req.Context(), ctxkeys.UserID, "user_test"), wsID))
	run, err := handler.executeDSLWorkflow(req, wsID, &workflowdomain.Workflow{
		ID:                "wf_exec_helper",
		WorkspaceID:       wsID,
		AgentDefinitionID: testStringPtr("dsl-agent-exec"),
		Name:              "resolve_support_case",
		DSLSource:         "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		Version:           1,
		Status:            workflowdomain.StatusActive,
	}, ExecuteWorkflowRequest{})
	if err != nil || run == nil {
		t.Fatalf("executeDSLWorkflow() = %#v, %v", run, err)
	}
}
