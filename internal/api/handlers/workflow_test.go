package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

type mockWorkflowService struct {
	items     map[string]*workflowdomain.Workflow
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
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
