package policy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

const (
	testCapabilityExecutionID = "exec-capability-1"
	testCapabilityName        = "external.mutate"
)

func TestCapabilityApprovalGovernor_RequiresApprovalContext(t *testing.T) {
	db := setupPolicyTestDB(t)
	svc := NewApprovalService(db, audit.NewAuditService(db))
	governor := NewCapabilityApprovalGovernor(svc)
	descriptor := tool.CapabilityDescriptor{
		Name:            testCapabilityName,
		Version:         "1",
		Operation:       "apply",
		SideEffectClass: tool.SideEffectIrreversible,
	}

	err := governor.CheckCapabilityExecution(context.Background(), descriptor)
	if !errors.Is(err, ErrCapabilityApprovalRequired) {
		t.Fatalf("expected ErrCapabilityApprovalRequired, got %v", err)
	}
}

func TestCapabilityApprovalGovernor_AcceptsMatchingApprovedExecution(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, emptyJSONPayload)
	approverID := seedUserInWorkspace(t, db, workspaceID)
	svc := NewApprovalService(db, audit.NewAuditService(db))
	governor := NewCapabilityApprovalGovernor(svc)
	descriptor := tool.CapabilityDescriptor{
		Name:             testCapabilityName,
		Version:          "1",
		Operation:        "apply",
		SideEffectClass:  tool.SideEffectMutate,
		ApprovalRequired: true,
	}

	resourceType := tool.CapabilityApprovalResourceType
	resourceID := testCapabilityExecutionID
	req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID:  workspaceID,
		RequestedBy:  requesterID,
		ApproverID:   approverID,
		Action:       tool.CapabilityApprovalAction(descriptor),
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		ExpiresAt:    time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateApprovalRequest error: %v", err)
	}
	if err := svc.DecideApprovalRequest(context.Background(), req.ID, decisionApprove, approverID); err != nil {
		t.Fatalf("DecideApprovalRequest error: %v", err)
	}

	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, workspaceID)
	ctx = context.WithValue(ctx, ctxkeys.ExecutionID, testCapabilityExecutionID)
	ctx = context.WithValue(ctx, ctxkeys.ApprovalID, req.ID)
	if err := governor.CheckCapabilityExecution(ctx, descriptor); err != nil {
		t.Fatalf("CheckCapabilityExecution error: %v", err)
	}
}

func TestCapabilityApprovalGovernor_RejectsApprovalForDifferentExecution(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, requesterID := seedWorkspaceUserRole(t, db, emptyJSONPayload)
	approverID := seedUserInWorkspace(t, db, workspaceID)
	svc := NewApprovalService(db, audit.NewAuditService(db))
	governor := NewCapabilityApprovalGovernor(svc)
	descriptor := tool.CapabilityDescriptor{
		Name:            testCapabilityName,
		Version:         "1",
		Operation:       "apply",
		SideEffectClass: tool.SideEffectIrreversible,
	}

	resourceType := tool.CapabilityApprovalResourceType
	approvedExecutionID := "exec-other"
	req, err := svc.CreateApprovalRequest(context.Background(), CreateApprovalRequestInput{
		WorkspaceID:  workspaceID,
		RequestedBy:  requesterID,
		ApproverID:   approverID,
		Action:       tool.CapabilityApprovalAction(descriptor),
		ResourceType: &resourceType,
		ResourceID:   &approvedExecutionID,
		ExpiresAt:    time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateApprovalRequest error: %v", err)
	}
	if err := svc.DecideApprovalRequest(context.Background(), req.ID, decisionApprove, approverID); err != nil {
		t.Fatalf("DecideApprovalRequest error: %v", err)
	}

	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, workspaceID)
	ctx = context.WithValue(ctx, ctxkeys.ExecutionID, testCapabilityExecutionID)
	ctx = context.WithValue(ctx, ctxkeys.ApprovalID, req.ID)
	err = governor.CheckCapabilityExecution(ctx, descriptor)
	if !errors.Is(err, ErrCapabilityApprovalMismatch) {
		t.Fatalf("expected ErrCapabilityApprovalMismatch, got %v", err)
	}
}
