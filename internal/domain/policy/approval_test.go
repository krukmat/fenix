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

func TestApprovalService_Decide_ApproveAndDeny(t *testing.T) {
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

	t.Run("deny", func(t *testing.T) {
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
		if stored.Status != ApprovalStatusDenied {
			t.Fatalf("status = %s; want denied", stored.Status)
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
