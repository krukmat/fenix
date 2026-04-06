package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	signaldomain "github.com/matiasleandrokruk/fenix/internal/domain/signal"
)

type mockSignalService struct {
	items        map[string]*signaldomain.Signal
	listErr      error
	getEntityErr error
	dismissErr   error
	dismissedID  string
	dismissedBy  string
}

func newMockSignalService() *mockSignalService {
	return &mockSignalService{items: make(map[string]*signaldomain.Signal)}
}

func (m *mockSignalService) List(_ context.Context, _, _ string) ([]*signaldomain.Signal, error) {
	panic("unused")
}

func (m *mockSignalService) List2(_ context.Context, _ string, _ signaldomain.Filters) ([]*signaldomain.Signal, error) {
	panic("unused")
}

func (m *mockSignalService) ListSignals(_ context.Context, _ string, _ signaldomain.Filters) ([]*signaldomain.Signal, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]*signaldomain.Signal, 0, len(m.items))
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}

func (m *mockSignalService) GetByEntity(_ context.Context, _, entityType, entityID string) ([]*signaldomain.Signal, error) {
	if m.getEntityErr != nil {
		return nil, m.getEntityErr
	}
	out := make([]*signaldomain.Signal, 0, len(m.items))
	for _, item := range m.items {
		if item.EntityType == entityType && item.EntityID == entityID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *mockSignalService) Dismiss(_ context.Context, _, signalID, actorID string) error {
	if m.dismissErr != nil {
		return m.dismissErr
	}
	if _, ok := m.items[signalID]; !ok {
		return signaldomain.ErrSignalNotFound
	}
	m.dismissedID = signalID
	m.dismissedBy = actorID
	return nil
}

func TestSignalHandler_List_Returns200(t *testing.T) {
	mock := newMockSignalService()
	now := time.Now().UTC()
	mock.items["sig_1"] = &signaldomain.Signal{
		ID:          "sig_1",
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.9,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf_1",
		Status:      signaldomain.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewSignalHandler(signalServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/signals", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/signals", nil)
	req = req.WithContext(withSignalContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestSignalHandler_List_ByEntity_Returns200(t *testing.T) {
	mock := newMockSignalService()
	now := time.Now().UTC()
	mock.items["sig_1"] = &signaldomain.Signal{
		ID:          "sig_1",
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.9,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf_1",
		Status:      signaldomain.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewSignalHandler(signalServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/signals", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/signals?entity_type=lead&entity_id=lead_1", nil)
	req = req.WithContext(withSignalContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestSignalHandler_List_InvalidStatus_Returns400(t *testing.T) {
	handler := NewSignalHandler(signalServiceAdapter{newMockSignalService()})

	r := chi.NewRouter()
	r.Get("/signals", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/signals?status=broken", nil)
	req = req.WithContext(withSignalContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestSignalHandler_Dismiss_Returns204(t *testing.T) {
	mock := newMockSignalService()
	mock.items["sig_1"] = &signaldomain.Signal{ID: "sig_1"}
	handler := NewSignalHandler(signalServiceAdapter{mock})

	r := chi.NewRouter()
	r.Put("/signals/{id}/dismiss", handler.Dismiss)

	req := httptest.NewRequest(http.MethodPut, "/signals/sig_1/dismiss", nil)
	req = req.WithContext(withSignalContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if mock.dismissedBy != "user_test" {
		t.Fatalf("dismissedBy = %s, want user_test", mock.dismissedBy)
	}
}

func TestSignalHandler_Dismiss_Returns404(t *testing.T) {
	mock := newMockSignalService()
	handler := NewSignalHandler(signalServiceAdapter{mock})

	r := chi.NewRouter()
	r.Put("/signals/{id}/dismiss", handler.Dismiss)

	req := httptest.NewRequest(http.MethodPut, "/signals/missing/dismiss", nil)
	req = req.WithContext(withSignalContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestSignalHandler_List_ResponseBody(t *testing.T) {
	mock := newMockSignalService()
	now := time.Now().UTC()
	mock.items["sig_1"] = &signaldomain.Signal{
		ID:          "sig_1",
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.9,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf_1",
		Metadata:    json.RawMessage(`{"reason":"score"}`),
		Status:      signaldomain.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	handler := NewSignalHandler(signalServiceAdapter{mock})

	r := chi.NewRouter()
	r.Get("/signals", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/signals", nil)
	req = req.WithContext(withSignalContext(req.Context()))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	var payload struct {
		Data []SignalResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(payload.Data))
	}
	if payload.Data[0].Metadata["reason"] != "score" {
		t.Fatalf("metadata reason mismatch: %#v", payload.Data[0].Metadata)
	}
}

func withSignalContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, "ws_test")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "user_test")
	return ctx
}

func TestNewSignalHandlerWithAuthorizer_NotNil(t *testing.T) {
	t.Parallel()

	mock := newMockSignalService()
	h := NewSignalHandlerWithAuthorizer(signalServiceAdapter{mock}, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestWriteSignalError_StatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err        error
		wantStatus int
	}{
		{signaldomain.ErrSignalNotFound, http.StatusNotFound},
		{signaldomain.ErrInvalidSignalInput, http.StatusUnprocessableEntity},
		{signaldomain.ErrSignalDismissInvalid, http.StatusConflict},
	}

	for _, tc := range tests {
		w := httptest.NewRecorder()
		writeSignalError(w, tc.err)
		if w.Code != tc.wantStatus {
			t.Errorf("writeSignalError(%v): status = %d, want %d", tc.err, w.Code, tc.wantStatus)
		}
	}
}

func TestFormatOptionalSignalTime_NonNil(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	got := formatOptionalSignalTime(&ts)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if *got == "" {
		t.Fatal("expected non-empty formatted time")
	}
}

func TestSignalToResponse_Nil(t *testing.T) {
	t.Parallel()

	if signalToResponse(nil) != nil {
		t.Fatal("expected nil response for nil input")
	}
}

type signalServiceAdapter struct{ *mockSignalService }

func (a signalServiceAdapter) List(ctx context.Context, workspaceID string, filters signaldomain.Filters) ([]*signaldomain.Signal, error) {
	return a.mockSignalService.ListSignals(ctx, workspaceID, filters)
}
