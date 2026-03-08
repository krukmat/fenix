// Traces: FR-061
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
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
)

func TestApprovalHandler_ListPendingApprovals_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	requesterID := createUser(t, db, wsID)
	approverID := createUser(t, db, wsID)

	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	_, err := svc.CreateApprovalRequest(context.Background(), policy.CreateApprovalRequestInput{
		WorkspaceID: wsID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("seed approval request: %v", err)
	}

	h := NewApprovalHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil)
	req = req.WithContext(contextWithUserID(req.Context(), approverID))

	rr := httptest.NewRecorder()
	h.ListPendingApprovals(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatalf("data type=%T, want []any", resp["data"])
	}
	if len(data) != 1 {
		t.Fatalf("data len=%d want=1", len(data))
	}
}

func TestApprovalHandler_ListPendingApprovals_MissingUser_Returns401(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewApprovalHandler(policy.NewApprovalService(db, audit.NewAuditService(db)))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil)
	rr := httptest.NewRecorder()

	h.ListPendingApprovals(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}
}

func TestApprovalHandler_DecideApproval_SuccessNoContent(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	requesterID := createUser(t, db, wsID)
	approverID := createUser(t, db, wsID)

	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	approval, err := svc.CreateApprovalRequest(context.Background(), policy.CreateApprovalRequestInput{
		WorkspaceID: wsID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("seed approval request: %v", err)
	}

	h := NewApprovalHandler(svc)
	body, _ := json.Marshal(map[string]any{"decision": "approve"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/"+approval.ID, bytes.NewReader(body))
	req = req.WithContext(contextWithUserID(req.Context(), approverID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", approval.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
}

func TestApprovalHandler_DecideApproval_InvalidDecision_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	requesterID := createUser(t, db, wsID)
	approverID := createUser(t, db, wsID)

	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	approval, err := svc.CreateApprovalRequest(context.Background(), policy.CreateApprovalRequestInput{
		WorkspaceID: wsID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("seed approval request: %v", err)
	}

	h := NewApprovalHandler(svc)
	body, _ := json.Marshal(map[string]any{"decision": "maybe"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/"+approval.ID, bytes.NewReader(body))
	req = req.WithContext(contextWithUserID(req.Context(), approverID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", approval.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}

// TestApprovalHandler_DecideApproval_MissingUser returns 401 without user context.
func TestApprovalHandler_DecideApproval_MissingUser(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	h := NewApprovalHandler(svc)

	body, _ := json.Marshal(map[string]any{"decision": "approve"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/some-id", bytes.NewReader(body))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "some-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

// TestApprovalHandler_DecideApproval_InvalidBody returns 400 on bad JSON.
func TestApprovalHandler_DecideApproval_InvalidBody(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	h := NewApprovalHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/some-id", bytes.NewBufferString("not-json"))
	req = req.WithContext(contextWithUserID(req.Context(), "user-id"))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "some-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

// TestApprovalHandler_DecideApproval_MissingID returns 400 without ID param.
func TestApprovalHandler_DecideApproval_MissingID(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	h := NewApprovalHandler(svc)

	body, _ := json.Marshal(map[string]any{"decision": "approve"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/", bytes.NewReader(body))
	req = req.WithContext(contextWithUserID(req.Context(), "user-id"))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

// TestApprovalHandler_DecideApproval_NotFound returns 404 for unknown approval ID.
func TestApprovalHandler_DecideApproval_NotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	h := NewApprovalHandler(svc)

	body, _ := json.Marshal(map[string]any{"decision": "approve"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/nonexistent", bytes.NewReader(body))
	req = req.WithContext(contextWithUserID(req.Context(), "some-user-id"))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
}

// TestApprovalHandler_DecideApproval_Forbidden returns 403 when user is not the assigned approver.
func TestApprovalHandler_DecideApproval_Forbidden(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	requesterID := createUser(t, db, wsID)
	approverID := createUser(t, db, wsID)
	otherUserID := createUser(t, db, wsID)

	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	approval, err := svc.CreateApprovalRequest(context.Background(), policy.CreateApprovalRequestInput{
		WorkspaceID: wsID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("seed approval request: %v", err)
	}

	h := NewApprovalHandler(svc)
	body, _ := json.Marshal(map[string]any{"decision": "approve"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/"+approval.ID, bytes.NewReader(body))
	req = req.WithContext(contextWithUserID(req.Context(), otherUserID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", approval.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
}

// TestApprovalHandler_DecideApproval_Expired returns 409 for an expired approval request.
func TestApprovalHandler_DecideApproval_Expired(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	requesterID := createUser(t, db, wsID)
	approverID := createUser(t, db, wsID)

	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	approval, err := svc.CreateApprovalRequest(context.Background(), policy.CreateApprovalRequestInput{
		WorkspaceID: wsID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(-1 * time.Minute), // already expired
	})
	if err != nil {
		t.Fatalf("seed approval request: %v", err)
	}

	h := NewApprovalHandler(svc)
	body, _ := json.Marshal(map[string]any{"decision": "approve"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/"+approval.ID, bytes.NewReader(body))
	req = req.WithContext(contextWithUserID(req.Context(), approverID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", approval.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
	}
}

// TestApprovalHandler_DecideApproval_AlreadyClosed returns 409 when approval is already decided.
func TestApprovalHandler_DecideApproval_AlreadyClosed(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	requesterID := createUser(t, db, wsID)
	approverID := createUser(t, db, wsID)

	svc := policy.NewApprovalService(db, audit.NewAuditService(db))
	approval, err := svc.CreateApprovalRequest(context.Background(), policy.CreateApprovalRequestInput{
		WorkspaceID: wsID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("seed approval request: %v", err)
	}

	// Decide it once successfully
	if err := svc.DecideApprovalRequest(context.Background(), approval.ID, "approve", approverID); err != nil {
		t.Fatalf("first decision: %v", err)
	}

	h := NewApprovalHandler(svc)
	body, _ := json.Marshal(map[string]any{"decision": "approve"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/approvals/"+approval.ID, bytes.NewReader(body))
	req = req.WithContext(contextWithUserID(req.Context(), approverID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", approval.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DecideApproval(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
	}
}

func contextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxkeys.UserID, userID)
}
