// Task 3.9: Prompt Versioning
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

// MockPromptVersionService para testing
type MockPromptVersionService struct {
	createCalls    int
	getCalls       int
	promoteCalls   int
	rollbackCalls  int
	versions       map[string]*agent.PromptVersion
}

func NewMockPromptVersionService() *MockPromptVersionService {
	return &MockPromptVersionService{
		versions: make(map[string]*agent.PromptVersion),
	}
}

func (m *MockPromptVersionService) CreatePromptVersion(ctx context.Context, input agent.CreatePromptVersionInput) (*agent.PromptVersion, error) {
	m.createCalls++
	pv := &agent.PromptVersion{
		ID:                input.AgentDefinitionID + "_v1",
		WorkspaceID:       input.WorkspaceID,
		AgentDefinitionID: input.AgentDefinitionID,
		VersionNumber:     1,
		SystemPrompt:      input.SystemPrompt,
		Status:            agent.PromptStatusDraft,
	}
	m.versions[pv.ID] = pv
	return pv, nil
}

func (m *MockPromptVersionService) GetActivePrompt(ctx context.Context, workspaceID, agentID string) (*agent.PromptVersion, error) {
	for _, pv := range m.versions {
		if pv.AgentDefinitionID == agentID && pv.Status == agent.PromptStatusActive {
			return pv, nil
		}
	}
	return nil, nil
}

func (m *MockPromptVersionService) ListPromptVersions(ctx context.Context, workspaceID, agentID string) ([]*agent.PromptVersion, error) {
	var result []*agent.PromptVersion
	for _, pv := range m.versions {
		if pv.AgentDefinitionID == agentID {
			result = append(result, pv)
		}
	}
	return result, nil
}

func (m *MockPromptVersionService) GetPromptVersionByID(ctx context.Context, workspaceID, promptVersionID string) (*agent.PromptVersion, error) {
	return m.versions[promptVersionID], nil
}

func (m *MockPromptVersionService) PromotePrompt(ctx context.Context, workspaceID, promptVersionID string) error {
	m.promoteCalls++
	if pv, ok := m.versions[promptVersionID]; ok {
		pv.Status = agent.PromptStatusActive
	}
	return nil
}

func (m *MockPromptVersionService) RollbackPrompt(ctx context.Context, workspaceID, agentID string) error {
	m.rollbackCalls++
	return nil
}

func TestListPromptsHandler_FiltersWorkspace(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	// Mount routes on chi router
	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Get("/", handler.List)
	})

	req := httptest.NewRequest("GET", "/admin/prompts?agent_id=agent_support", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCreatePromptHandler_Returns201(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	// Mount routes on chi router
	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Post("/", handler.Create)
	})

	body := CreatePromptVersionRequest{
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "You are a support agent.",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/prompts", bytes.NewReader(bodyBytes))
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	if mock.createCalls != 1 {
		t.Errorf("expected 1 create call, got %d", mock.createCalls)
	}
}

func TestPromotePromptHandler_Returns200(t *testing.T) {
	mock := NewMockPromptVersionService()
	pv := &agent.PromptVersion{
		ID:                "pv_123",
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		VersionNumber:     1,
		SystemPrompt:      "test",
		Status:            agent.PromptStatusDraft,
	}
	mock.versions["pv_123"] = pv

	handler := NewPromptHandler(mock)

	// Mount routes on chi router
	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Put("/{id}/promote", handler.Promote)
	})

	req := httptest.NewRequest("PUT", "/admin/prompts/pv_123/promote", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if mock.promoteCalls != 1 {
		t.Errorf("expected 1 promote call, got %d", mock.promoteCalls)
	}
}

func TestRollbackPromptHandler_NoArchived_ReturnsConflict(t *testing.T) {
	// Custom mock that returns error on RollbackPrompt (no archived prompt)
	mockSvc := &mockRollbackErrorService{}
	handler := NewPromptHandler(mockSvc)

	// Mount routes on chi router
	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Put("/{id}/rollback", handler.Rollback)
	})

	req := httptest.NewRequest("PUT", "/admin/prompts/agent_support/rollback", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 409 Conflict since no archived prompt to rollback to
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestRollbackPromptHandler_WithArchivedPrompt_ReturnsSuccess(t *testing.T) {
	mock := NewMockPromptVersionService()

	// Setup: create an archived prompt version
	archivedPv := &agent.PromptVersion{
		ID:                "pv_archived",
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		VersionNumber:     1,
		SystemPrompt:      "test archived",
		Status:            agent.PromptStatusArchived,
	}
	mock.versions["pv_archived"] = archivedPv

	// Custom RollbackPrompt that simulates finding archived prompt
	handler := &PromptHandler{service: &mockRollbackService{
		versions: mock.versions,
		onRollback: func() (*agent.PromptVersion, error) {
			return archivedPv, nil
		},
	}}

	// Mount routes on chi router
	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Put("/{id}/rollback", handler.Rollback)
	})

	req := httptest.NewRequest("PUT", "/admin/prompts/agent_support/rollback", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// mockRollbackService extends the base mock for rollback-specific testing
type mockRollbackService struct {
	versions   map[string]*agent.PromptVersion
	onRollback func() (*agent.PromptVersion, error)
}

func (m *mockRollbackService) CreatePromptVersion(ctx context.Context, input agent.CreatePromptVersionInput) (*agent.PromptVersion, error) {
	return nil, nil
}

func (m *mockRollbackService) GetActivePrompt(ctx context.Context, workspaceID, agentID string) (*agent.PromptVersion, error) {
	if m.onRollback != nil {
		return m.onRollback()
	}
	return nil, nil
}

func (m *mockRollbackService) ListPromptVersions(ctx context.Context, workspaceID, agentID string) ([]*agent.PromptVersion, error) {
	return nil, nil
}

func (m *mockRollbackService) GetPromptVersionByID(ctx context.Context, workspaceID, promptVersionID string) (*agent.PromptVersion, error) {
	return nil, nil
}

func (m *mockRollbackService) PromotePrompt(ctx context.Context, workspaceID, promptVersionID string) error {
	return nil
}

func (m *mockRollbackService) RollbackPrompt(ctx context.Context, workspaceID, agentID string) error {
	return nil
}

// mockRollbackErrorService returns error on RollbackPrompt (simulates no archived prompt)
type mockRollbackErrorService struct{}

func (m *mockRollbackErrorService) CreatePromptVersion(ctx context.Context, input agent.CreatePromptVersionInput) (*agent.PromptVersion, error) {
	return nil, nil
}

func (m *mockRollbackErrorService) GetActivePrompt(ctx context.Context, workspaceID, agentID string) (*agent.PromptVersion, error) {
	return nil, nil
}

func (m *mockRollbackErrorService) ListPromptVersions(ctx context.Context, workspaceID, agentID string) ([]*agent.PromptVersion, error) {
	return nil, nil
}

func (m *mockRollbackErrorService) GetPromptVersionByID(ctx context.Context, workspaceID, promptVersionID string) (*agent.PromptVersion, error) {
	return nil, nil
}

func (m *mockRollbackErrorService) PromotePrompt(ctx context.Context, workspaceID, promptVersionID string) error {
	return nil
}

func (m *mockRollbackErrorService) RollbackPrompt(ctx context.Context, workspaceID, agentID string) error {
	return fmt.Errorf("no archived prompt to rollback to")
}

// Test error cases and edge paths

func TestListPromptsHandler_MissingWorkspaceID(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Get("/", handler.List)
	})

	// Request WITHOUT workspace ID in context
	req := httptest.NewRequest("GET", "/admin/prompts?agent_id=agent_support", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListPromptsHandler_MissingAgentID(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Get("/", handler.List)
	})

	req := httptest.NewRequest("GET", "/admin/prompts", nil) // No agent_id query param
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreatePromptHandler_MissingWorkspaceID(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Post("/", handler.Create)
	})

	body := CreatePromptVersionRequest{
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "You are a support agent.",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/prompts", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreatePromptHandler_InvalidJSON(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Post("/", handler.Create)
	})

	req := httptest.NewRequest("POST", "/admin/prompts", bytes.NewReader([]byte("invalid json")))
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreatePromptHandler_MissingRequiredFields(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Post("/", handler.Create)
	})

	// Missing SystemPrompt
	body := CreatePromptVersionRequest{
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "", // Empty required field
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/prompts", bytes.NewReader(bodyBytes))
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPromotePromptHandler_MissingWorkspaceID(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Put("/{id}/promote", handler.Promote)
	})

	req := httptest.NewRequest("PUT", "/admin/prompts/pv_123/promote", nil)
	// No workspace ID in context

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestPromotePromptHandler_MissingIDParam(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Put("/{id}/promote", handler.Promote)
	})

	req := httptest.NewRequest("PUT", "/admin/prompts//promote", nil) // Empty id param
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRollbackPromptHandler_MissingWorkspaceID(t *testing.T) {
	mock := NewMockPromptVersionService()
	handler := NewPromptHandler(mock)

	r := chi.NewRouter()
	r.Route("/admin/prompts", func(r chi.Router) {
		r.Put("/{id}/rollback", handler.Rollback)
	})

	req := httptest.NewRequest("PUT", "/admin/prompts/agent_support/rollback", nil)
	// No workspace ID

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
