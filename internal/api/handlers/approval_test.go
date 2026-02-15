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

func contextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxkeys.UserID, userID)
}
