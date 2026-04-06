// Traces: FR-061
package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

func TestApprovalService_CreateApprovalRequest(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)
	approverID := seedUserInWorkspace(t, db, workspaceID)

	svc := NewApprovalService(db, audit.NewAuditService(db))
	req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID: workspaceID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateApprovalRequest error: %v", err)
	}
	if req.Status != ApprovalStatusPending {
		t.Fatalf("status = %s; want pending", req.Status)
	}
}

func TestApprovalService_Decide_ApproveAndReject(t *testing.T) {
	t.Run("approve", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		workspaceID, requesterID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)
		approverID := seedUserInWorkspace(t, db, workspaceID)
		svc := NewApprovalService(db, audit.NewAuditService(db))

		req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
			WorkspaceID: workspaceID,
			RequestedBy: requesterID,
			ApproverID:  approverID,
			Action:      "tool.execute",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("create error: %v", err)
		}

		if err := svc.DecideApprovalRequest(context.Background(), req.ID, "approve", approverID); err != nil {
			t.Fatalf("approve error: %v", err)
		}

		stored, err := svc.getApprovalByID(context.Background(), req.ID)
		if err != nil {
			t.Fatalf("getApprovalByID error: %v", err)
		}
		if stored.Status != ApprovalStatusApproved {
			t.Fatalf("status = %s; want approved", stored.Status)
		}
	})

	t.Run("reject via legacy deny alias", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		workspaceID, requesterID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)
		approverID := seedUserInWorkspace(t, db, workspaceID)
		svc := NewApprovalService(db, audit.NewAuditService(db))

		req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
			WorkspaceID: workspaceID,
			RequestedBy: requesterID,
			ApproverID:  approverID,
			Action:      "tool.execute",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("create error: %v", err)
		}

		if err := svc.DecideApprovalRequest(context.Background(), req.ID, "deny", approverID); err != nil {
			t.Fatalf("deny error: %v", err)
		}

		stored, err := svc.getApprovalByID(context.Background(), req.ID)
		if err != nil {
			t.Fatalf("getApprovalByID error: %v", err)
		}
		if stored.Status != ApprovalStatusRejected {
			t.Fatalf("status = %s; want rejected", stored.Status)
		}
	})
}

func TestApprovalService_CancelPendingRequest(t *testing.T) {
	t.Run("requester can cancel", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		workspaceID, requesterID := seedWorkspaceUserRole(t, db, emptyJSONPayload)
		approverID := seedUserInWorkspace(t, db, workspaceID)
		svc := NewApprovalService(db, audit.NewAuditService(db))

		req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
			WorkspaceID: workspaceID,
			RequestedBy: requesterID,
			ApproverID:  approverID,
			Action:      "tool.execute",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("create error: %v", err)
		}

		if err := svc.DecideApprovalRequest(context.Background(), req.ID, "cancel", requesterID); err != nil {
			t.Fatalf("cancel error: %v", err)
		}

		stored, err := svc.getApprovalByID(context.Background(), req.ID)
		if err != nil {
			t.Fatalf("getApprovalByID error: %v", err)
		}
		if stored.Status != ApprovalStatusCancelled {
			t.Fatalf("status = %s; want cancelled", stored.Status)
		}
		if stored.DecidedBy == nil || *stored.DecidedBy != requesterID {
			t.Fatalf("decidedBy = %v; want requester %q", stored.DecidedBy, requesterID)
		}
	})

	t.Run("unrelated actor cannot cancel", func(t *testing.T) {
		db := setupPolicyTestDB(t)
		workspaceID, requesterID := seedWorkspaceUserRole(t, db, emptyJSONPayload)
		approverID := seedUserInWorkspace(t, db, workspaceID)
		intruderID := seedUserInWorkspace(t, db, workspaceID)
		svc := NewApprovalService(db, audit.NewAuditService(db))

		req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
			WorkspaceID: workspaceID,
			RequestedBy: requesterID,
			ApproverID:  approverID,
			Action:      "tool.execute",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("create error: %v", err)
		}

		err = svc.DecideApprovalRequest(context.Background(), req.ID, "cancel", intruderID)
		if !errors.Is(err, ErrApprovalForbidden) {
			t.Fatalf("expected ErrApprovalForbidden, got %v", err)
		}
	})
}

func TestApprovalService_ExpiredAndPending(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)
	approverID := seedUserInWorkspace(t, db, workspaceID)
	svc := NewApprovalService(db, audit.NewAuditService(db))

	_, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID: workspaceID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(-2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create expired error: %v", err)
	}

	pending, err := svc.GetPendingApprovals(context.Background(), approverID)
	if err != nil {
		t.Fatalf("GetPendingApprovals error: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending len = %d; want 0", len(pending))
	}
}

func TestApprovalService_ForbiddenApprover(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, `{"tools":["update_case"]}`)
	approverID := seedUserInWorkspace(t, db, workspaceID)
	intruderID := seedUserInWorkspace(t, db, workspaceID)
	svc := NewApprovalService(db, audit.NewAuditService(db))

	req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID: workspaceID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		Payload:     json.RawMessage(`{"case_id":"x"}`),
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	err = svc.DecideApprovalRequest(context.Background(), req.ID, "approve", intruderID)
	if !errors.Is(err, ErrApprovalForbidden) {
		t.Fatalf("expected ErrApprovalForbidden, got %v", err)
	}
}

func TestApprovalService_DecideExpiredRequest_ReturnsExpiredError(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, emptyJSONPayload)
	approverID := seedUserInWorkspace(t, db, workspaceID)
	svc := NewApprovalService(db, audit.NewAuditService(db))

	req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID: workspaceID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(-5 * time.Minute), // already expired
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	err = svc.DecideApprovalRequest(context.Background(), req.ID, "approve", approverID)
	if !errors.Is(err, ErrApprovalExpired) {
		t.Fatalf("expected ErrApprovalExpired, got %v", err)
	}
}

func TestApprovalService_DecideAlreadyClosed_ReturnsAlreadyClosedError(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, emptyJSONPayload)
	approverID := seedUserInWorkspace(t, db, workspaceID)
	svc := NewApprovalService(db, audit.NewAuditService(db))

	req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID: workspaceID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	if err := svc.DecideApprovalRequest(context.Background(), req.ID, "approve", approverID); err != nil {
		t.Fatalf("first approve error: %v", err)
	}

	// Second decision on already-closed request
	err = svc.DecideApprovalRequest(context.Background(), req.ID, "reject", approverID)
	if !errors.Is(err, ErrApprovalAlreadyClosed) {
		t.Fatalf("expected ErrApprovalAlreadyClosed, got %v", err)
	}
}

func TestApprovalService_GetPendingApprovals_ReturnsPending(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, emptyJSONPayload)
	approverID := seedUserInWorkspace(t, db, workspaceID)
	svc := NewApprovalService(db, audit.NewAuditService(db))

	_, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID: workspaceID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	pending, err := svc.GetPendingApprovals(context.Background(), approverID)
	if err != nil {
		t.Fatalf("GetPendingApprovals error: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending approval, got %d", len(pending))
	}
	if pending[0].Status != ApprovalStatusPending {
		t.Fatalf("expected pending status, got %s", pending[0].Status)
	}
}

func TestApprovalService_GetApprovalByID_NotFound(t *testing.T) {
	db := setupPolicyTestDB(t)
	svc := NewApprovalService(db, audit.NewAuditService(db))

	_, err := svc.getApprovalByID(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for non-existent approval ID")
	}
}

func TestApprovalService_DecideApprovalRequest_InvalidDecision(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, emptyJSONPayload)
	approverID := seedUserInWorkspace(t, db, workspaceID)
	svc := NewApprovalService(db, audit.NewAuditService(db))

	req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID: workspaceID,
		RequestedBy: requesterID,
		ApproverID:  approverID,
		Action:      "tool.execute",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	err = svc.DecideApprovalRequest(context.Background(), req.ID, "invalid_decision", approverID)
	if err == nil {
		t.Fatal("expected error for invalid decision value")
	}
}

func TestDecisionToStatus(t *testing.T) {
	tests := []struct {
		input string
		want  ApprovalStatus
	}{
		{"approve", ApprovalStatusApproved},
		{"approved", ApprovalStatusApproved},
		{"deny", ApprovalStatusRejected},
		{"denied", ApprovalStatusRejected},
		{"reject", ApprovalStatusRejected},
		{"rejected", ApprovalStatusRejected},
		{"cancel", ApprovalStatusCancelled},
		{"cancelled", ApprovalStatusCancelled},
		{"unknown", ""},
		{"", ""},
	}
	for _, tc := range tests {
		got := decisionToStatus(tc.input)
		if got != tc.want {
			t.Errorf("decisionToStatus(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestScanApprovalRequest_NormalizesLegacyDenied(t *testing.T) {
	now := time.Now().UTC()
	stored, err := scanApprovalRequest(approvalScannerStub{
		values: []any{
			"apr-1",
			"ws-1",
			"req-1",
			"app-1",
			sql.NullString{String: "app-1", Valid: true},
			"tool.execute",
			sql.NullString{},
			sql.NullString{},
			[]byte(`{}`),
			sql.NullString{},
			approvalStatusLegacyDenied,
			now.Add(time.Hour),
			sql.NullTime{Time: now, Valid: true},
			now,
			now,
		},
	})
	if err != nil {
		t.Fatalf("scanApprovalRequest error: %v", err)
	}
	if stored.Status != ApprovalStatusRejected {
		t.Fatalf("status = %s; want rejected", stored.Status)
	}
}

type approvalScannerStub struct {
	values []any
}

func (s approvalScannerStub) Scan(dest ...any) error {
	for i := range dest {
		switch ptr := dest[i].(type) {
		case *string:
			*ptr = s.values[i].(string)
		case *json.RawMessage:
			*ptr = s.values[i].(json.RawMessage)
		case *[]byte:
			*ptr = s.values[i].([]byte)
		case *ApprovalStatus:
			*ptr = s.values[i].(ApprovalStatus)
		case *time.Time:
			*ptr = s.values[i].(time.Time)
		case *sql.NullString:
			*ptr = s.values[i].(sql.NullString)
		case *sql.NullTime:
			*ptr = s.values[i].(sql.NullTime)
		default:
			return fmt.Errorf("unsupported dest type %T", dest[i])
		}
	}
	return nil
}

func seedUserInWorkspace(t *testing.T, db *sql.DB, workspaceID string) string {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	userID := uuid.NewV7().String()

	if _, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?)
	`, userID, workspaceID, fmt.Sprintf("%s@example.com", userID), "Approver", now, now); err != nil {
		t.Fatalf("insert user_account in workspace: %v", err)
	}

	return userID
}
