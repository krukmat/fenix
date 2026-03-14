package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	createErr      error
	getErr         error
	listErr        error
	updateErr      error
	markTestingErr error
	markActiveErr  error
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

func TestWorkflowHandler_Execute_Returns200(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	insertDSLWorkflowAgent(t, db, wsID, "dsl-agent-exec")
	insertExecutableWorkflow(t, db, wsID, "wf_exec_1", "dsl-agent-exec")

	toolRegistry := setupWorkflowToolRegistry(t, db, wsID)
	orch := agent.NewOrchestratorWithRegistry(db, agent.NewRunnerRegistry())
	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, orch, toolRegistry, nil, nil, nil)

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
	handler := NewWorkflowHandlerWithRuntime(workflowServiceAdapter{mock}, nil, nil, nil, nil, nil, nil, invalidator)

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
	handler := NewWorkflowHandlerWithRuntime(workflowServiceAdapter{mock}, nil, nil, nil, nil, nil, nil, invalidator)

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

	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, agent.NewOrchestratorWithRegistry(db, agent.NewRunnerRegistry()), nil, nil, nil, nil)
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

	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, agent.NewOrchestratorWithRegistry(db, agent.NewRunnerRegistry()), nil, nil, nil, nil)
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
	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, nil, nil, nil, nil, invalidator)
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
	handler := NewWorkflowHandlerWithRuntime(workflowServiceAdapter{mock}, nil, nil, nil, nil, nil, nil, invalidator)

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
	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, orch, toolRegistry, nil, nil, nil)

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
