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
)

type mockPromptVersionService struct {
	versions    map[string]*agent.PromptVersion
	promoteErr  error
	rollbackErr error
}

func newMockPromptVersionService() *mockPromptVersionService {
	return &mockPromptVersionService{
		versions: make(map[string]*agent.PromptVersion),
	}
}

func (m *mockPromptVersionService) CreatePromptVersion(_ context.Context, input agent.CreatePromptVersionInput) (*agent.PromptVersion, error) {
	pv := &agent.PromptVersion{
		ID:                input.AgentDefinitionID + "_v1",
		WorkspaceID:       input.WorkspaceID,
		AgentDefinitionID: input.AgentDefinitionID,
		VersionNumber:     1,
		SystemPrompt:      input.SystemPrompt,
		Status:            agent.PromptStatusDraft,
		CreatedAt:         time.Now(),
	}
	m.versions[pv.ID] = pv
	return pv, nil
}

func (m *mockPromptVersionService) GetActivePrompt(_ context.Context, _, agentID string) (*agent.PromptVersion, error) {
	for _, version := range m.versions {
		if version.AgentDefinitionID == agentID && version.Status == agent.PromptStatusActive {
			return version, nil
		}
	}
	return nil, nil
}

func (m *mockPromptVersionService) ListPromptVersions(_ context.Context, _, agentID string) ([]*agent.PromptVersion, error) {
	var versions []*agent.PromptVersion
	for _, version := range m.versions {
		if version.AgentDefinitionID == agentID {
			versions = append(versions, version)
		}
	}
	return versions, nil
}

func (m *mockPromptVersionService) GetPromptVersionByID(_ context.Context, _, promptVersionID string) (*agent.PromptVersion, error) {
	version, ok := m.versions[promptVersionID]
	if !ok {
		return nil, agent.ErrPromptVersionNotFound
	}
	return version, nil
}

func (m *mockPromptVersionService) PromotePrompt(_ context.Context, _, promptVersionID string) error {
	if m.promoteErr != nil {
		return m.promoteErr
	}
	if version, ok := m.versions[promptVersionID]; ok {
		version.Status = agent.PromptStatusActive
	}
	return nil
}

func (m *mockPromptVersionService) RollbackPrompt(_ context.Context, _, promptVersionID string) error {
	if m.rollbackErr != nil {
		return m.rollbackErr
	}
	if version, ok := m.versions[promptVersionID]; ok {
		version.Status = agent.PromptStatusActive
	}
	return nil
}

type mockPromptExperimentService struct {
	experiments map[string]*agent.PromptExperiment
}

func newMockExperimentService() *mockPromptExperimentService {
	return &mockPromptExperimentService{experiments: make(map[string]*agent.PromptExperiment)}
}

func (m *mockPromptExperimentService) StartPromptExperiment(_ context.Context, input agent.StartPromptExperimentInput) (*agent.PromptExperiment, error) {
	experiment := &agent.PromptExperiment{
		ID:                       "exp_1",
		WorkspaceID:              input.WorkspaceID,
		AgentDefinitionID:        "agent_support",
		ControlPromptVersionID:   input.ControlPromptVersionID,
		CandidatePromptVersionID: input.CandidatePromptVersionID,
		ControlTrafficPercent:    input.ControlTrafficPercent,
		CandidateTrafficPercent:  input.CandidateTrafficPercent,
		Status:                   agent.PromptExperimentStatusRunning,
	}
	m.experiments[experiment.ID] = experiment
	return experiment, nil
}

func (m *mockPromptExperimentService) ListPromptExperiments(_ context.Context, _, _ string) ([]*agent.PromptExperiment, error) {
	var experiments []*agent.PromptExperiment
	for _, experiment := range m.experiments {
		experiments = append(experiments, experiment)
	}
	return experiments, nil
}

func (m *mockPromptExperimentService) StopPromptExperiment(_ context.Context, input agent.StopPromptExperimentInput) (*agent.PromptExperiment, error) {
	experiment, ok := m.experiments[input.ExperimentID]
	if !ok {
		return nil, agent.ErrPromptExperimentNotFound
	}
	experiment.Status = agent.PromptExperimentStatusCompleted
	experiment.WinnerPromptVersionID = input.WinnerPromptVersionID
	return experiment, nil
}

func TestListPromptsHandler_Returns200(t *testing.T) {
	mock := newMockPromptVersionService()
	mock.versions["pv_1"] = &agent.PromptVersion{
		ID:                "pv_1",
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		VersionNumber:     1,
		SystemPrompt:      "test",
		Status:            agent.PromptStatusDraft,
		CreatedAt:         time.Now(),
	}
	handler := NewPromptHandler(mock, newMockExperimentService())

	r := chi.NewRouter()
	r.Get("/admin/prompts", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/admin/prompts?agent_id=agent_support", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestCreatePromptHandler_Returns201(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), newMockExperimentService())

	r := chi.NewRouter()
	r.Post("/admin/prompts", handler.Create)

	body, _ := json.Marshal(CreatePromptVersionRequest{
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "You are a support agent.",
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/prompts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withPromptContext(req.Context()))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestCreatePromptHandler_ReturnsBadRequestOnInvalidBody(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), newMockExperimentService())

	r := chi.NewRouter()
	r.Post("/admin/prompts", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/admin/prompts", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withPromptContext(req.Context()))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreatePromptHandler_ReturnsBadRequestOnMissingFields(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), newMockExperimentService())

	r := chi.NewRouter()
	r.Post("/admin/prompts", handler.Create)

	body, _ := json.Marshal(CreatePromptVersionRequest{AgentDefinitionID: "agent_support"})
	req := httptest.NewRequest(http.MethodPost, "/admin/prompts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withPromptContext(req.Context()))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPromotePromptHandler_ReturnsConflictOnMissingEval(t *testing.T) {
	mock := newMockPromptVersionService()
	mock.versions["pv_123"] = &agent.PromptVersion{
		ID:                "pv_123",
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		VersionNumber:     1,
		SystemPrompt:      "test",
		Status:            agent.PromptStatusDraft,
		CreatedAt:         time.Now(),
	}
	mock.promoteErr = agent.ErrPromptPromotionEvalMissing

	handler := NewPromptHandler(mock, newMockExperimentService())
	r := chi.NewRouter()
	r.Put("/admin/prompts/{id}/promote", handler.Promote)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/pv_123/promote", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestPromotePromptHandler_ReturnsUnauthorizedWithoutWorkspace(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), newMockExperimentService())
	r := chi.NewRouter()
	r.Put("/admin/prompts/{id}/promote", handler.Promote)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/pv_123/promote", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestPromotePromptHandler_ReturnsBadRequestWithoutID(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), newMockExperimentService())
	r := chi.NewRouter()
	r.Put("/admin/prompts/", handler.Promote)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPromotePromptHandler_ReturnsNotFound(t *testing.T) {
	mock := newMockPromptVersionService()
	mock.promoteErr = agent.ErrPromptVersionNotFound
	handler := NewPromptHandler(mock, newMockExperimentService())
	r := chi.NewRouter()
	r.Put("/admin/prompts/{id}/promote", handler.Promote)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/pv_missing/promote", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestPromotePromptHandler_ReturnsInternalError(t *testing.T) {
	mock := newMockPromptVersionService()
	mock.promoteErr = errors.New("boom")
	handler := NewPromptHandler(mock, newMockExperimentService())
	r := chi.NewRouter()
	r.Put("/admin/prompts/{id}/promote", handler.Promote)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/pv_123/promote", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestRollbackPromptHandler_UsesPromptVersionID(t *testing.T) {
	mock := newMockPromptVersionService()
	mock.versions["pv_archived"] = &agent.PromptVersion{
		ID:                "pv_archived",
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		VersionNumber:     1,
		SystemPrompt:      "archived",
		Status:            agent.PromptStatusArchived,
		CreatedAt:         time.Now(),
	}
	handler := NewPromptHandler(mock, newMockExperimentService())

	r := chi.NewRouter()
	r.Put("/admin/prompts/{id}/rollback", handler.Rollback)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/pv_archived/rollback", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestRollbackPromptHandler_ReturnsConflictForInvalidRollback(t *testing.T) {
	mock := newMockPromptVersionService()
	mock.rollbackErr = agent.ErrPromptRollbackInvalid
	handler := NewPromptHandler(mock, newMockExperimentService())

	r := chi.NewRouter()
	r.Put("/admin/prompts/{id}/rollback", handler.Rollback)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/pv_123/rollback", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestRollbackPromptHandler_ReturnsNotFound(t *testing.T) {
	mock := newMockPromptVersionService()
	mock.rollbackErr = agent.ErrPromptVersionNotFound
	handler := NewPromptHandler(mock, newMockExperimentService())

	r := chi.NewRouter()
	r.Put("/admin/prompts/{id}/rollback", handler.Rollback)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/pv_missing/rollback", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestRollbackPromptHandler_ReturnsInternalError(t *testing.T) {
	mock := newMockPromptVersionService()
	mock.rollbackErr = errors.New("boom")
	handler := NewPromptHandler(mock, newMockExperimentService())

	r := chi.NewRouter()
	r.Put("/admin/prompts/{id}/rollback", handler.Rollback)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/pv_123/rollback", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestPromptExperimentHandlers_StartListStop(t *testing.T) {
	experimentMock := newMockExperimentService()
	handler := NewPromptHandler(newMockPromptVersionService(), experimentMock)

	r := chi.NewRouter()
	r.Get("/admin/prompts/experiments", handler.ListExperiments)
	r.Post("/admin/prompts/experiments", handler.StartExperiment)
	r.Put("/admin/prompts/experiments/{id}/stop", handler.StopExperiment)

	startBody, _ := json.Marshal(StartPromptExperimentRequest{
		ControlPromptVersionID:   "pv_control",
		CandidatePromptVersionID: "pv_candidate",
		ControlTrafficPercent:    50,
		CandidateTrafficPercent:  50,
	})
	startReq := httptest.NewRequest(http.MethodPost, "/admin/prompts/experiments", bytes.NewReader(startBody))
	startReq.Header.Set("Content-Type", "application/json")
	startReq = startReq.WithContext(withPromptContext(startReq.Context()))
	startRR := httptest.NewRecorder()
	r.ServeHTTP(startRR, startReq)
	if startRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", startRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/prompts/experiments?agent_id=agent_support", nil)
	listReq = listReq.WithContext(withPromptContext(listReq.Context()))
	listRR := httptest.NewRecorder()
	r.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRR.Code)
	}

	stopBody, _ := json.Marshal(StopPromptExperimentRequest{WinnerPromptVersionID: stringPtr("pv_control")})
	stopReq := httptest.NewRequest(http.MethodPut, "/admin/prompts/experiments/exp_1/stop", bytes.NewReader(stopBody))
	stopReq.Header.Set("Content-Type", "application/json")
	stopReq = stopReq.WithContext(withPromptContext(stopReq.Context()))
	stopRR := httptest.NewRecorder()
	r.ServeHTTP(stopRR, stopReq)
	if stopRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", stopRR.Code)
	}
}

func TestPromptExperimentHandler_ReturnsBadRequestOnInvalidSplit(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), &promptExperimentErrorService{err: agent.ErrPromptExperimentInvalidSplit})

	r := chi.NewRouter()
	r.Post("/admin/prompts/experiments", handler.StartExperiment)

	body, _ := json.Marshal(StartPromptExperimentRequest{
		ControlPromptVersionID:   "pv_control",
		CandidatePromptVersionID: "pv_candidate",
		ControlTrafficPercent:    70,
		CandidateTrafficPercent:  20,
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/prompts/experiments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPromptExperimentHandlers_ReturnBadRequestOnInvalidBodies(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), newMockExperimentService())

	r := chi.NewRouter()
	r.Post("/admin/prompts/experiments", handler.StartExperiment)
	r.Put("/admin/prompts/experiments/{id}/stop", handler.StopExperiment)

	startReq := httptest.NewRequest(http.MethodPost, "/admin/prompts/experiments", strings.NewReader("{"))
	startReq.Header.Set("Content-Type", "application/json")
	startReq = startReq.WithContext(withPromptContext(startReq.Context()))
	startRR := httptest.NewRecorder()
	r.ServeHTTP(startRR, startReq)
	if startRR.Code != http.StatusBadRequest {
		t.Fatalf("expected start 400, got %d", startRR.Code)
	}

	stopReq := httptest.NewRequest(http.MethodPut, "/admin/prompts/experiments/exp_1/stop", strings.NewReader("{"))
	stopReq.Header.Set("Content-Type", "application/json")
	stopReq = stopReq.WithContext(withPromptContext(stopReq.Context()))
	stopRR := httptest.NewRecorder()
	r.ServeHTTP(stopRR, stopReq)
	if stopRR.Code != http.StatusBadRequest {
		t.Fatalf("expected stop 400, got %d", stopRR.Code)
	}
}

func TestListExperimentsHandler_ReturnsBadRequestWithoutAgentID(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), newMockExperimentService())
	r := chi.NewRouter()
	r.Get("/admin/prompts/experiments", handler.ListExperiments)

	req := httptest.NewRequest(http.MethodGet, "/admin/prompts/experiments", nil)
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPromptExperimentHandlers_ReturnNotFoundOnStopMissingExperiment(t *testing.T) {
	handler := NewPromptHandler(newMockPromptVersionService(), newMockExperimentService())
	r := chi.NewRouter()
	r.Put("/admin/prompts/experiments/{id}/stop", handler.StopExperiment)

	req := httptest.NewRequest(http.MethodPut, "/admin/prompts/experiments/exp_missing/stop", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withPromptContext(req.Context()))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestResolvePromptConfig_NilReturnsEmptyObject(t *testing.T) {
	if got := resolvePromptConfig(nil); got != errEmptyJSON {
		t.Fatalf("expected %s, got %s", errEmptyJSON, got)
	}
}

func TestToPromptVersionResponse_Nil(t *testing.T) {
	if got := toPromptVersionResponse(nil); got != nil {
		t.Fatalf("expected nil response, got %+v", got)
	}
}

func TestIsPromptNotFoundError(t *testing.T) {
	if !isPromptNotFoundError(sql.ErrNoRows) {
		t.Fatal("expected sql.ErrNoRows to be treated as not found")
	}
	if !isPromptNotFoundError(agent.ErrPromptVersionNotFound) {
		t.Fatal("expected ErrPromptVersionNotFound to be treated as not found")
	}
	if isPromptNotFoundError(errors.New("boom")) {
		t.Fatal("expected generic error not to be treated as not found")
	}
}

type promptExperimentErrorService struct {
	err error
}

func (s *promptExperimentErrorService) StartPromptExperiment(_ context.Context, _ agent.StartPromptExperimentInput) (*agent.PromptExperiment, error) {
	return nil, s.err
}

func (s *promptExperimentErrorService) ListPromptExperiments(_ context.Context, _, _ string) ([]*agent.PromptExperiment, error) {
	return nil, nil
}

func (s *promptExperimentErrorService) StopPromptExperiment(_ context.Context, _ agent.StopPromptExperimentInput) (*agent.PromptExperiment, error) {
	return nil, nil
}

func withPromptContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	return ctx
}

func stringPtr(value string) *string {
	return &value
}
