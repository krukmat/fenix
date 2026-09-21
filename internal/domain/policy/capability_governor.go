package policy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

var (
	ErrCapabilityApprovalRequired = errors.New("capability approval is required")
	ErrCapabilityApprovalMismatch = errors.New("capability approval does not match execution")
	ErrCapabilityApprovalPending  = errors.New("capability approval is not approved")
)

// CapabilityApprovalGovernor binds risky capability execution to the existing
// ApprovalService lifecycle without creating a second authorization authority.
type CapabilityApprovalGovernor struct {
	approvals *ApprovalService
}

// NewCapabilityApprovalGovernor creates the approval-backed capability governor.
func NewCapabilityApprovalGovernor(approvals *ApprovalService) *CapabilityApprovalGovernor {
	return &CapabilityApprovalGovernor{approvals: approvals}
}

// CheckCapabilityExecution enforces approval only when the descriptor requires it.
// Mutating capabilities may opt in explicitly; irreversible capabilities always require approval.
func (g *CapabilityApprovalGovernor) CheckCapabilityExecution(ctx context.Context, descriptor tool.CapabilityDescriptor) error {
	if !descriptor.RequiresApproval() {
		return nil
	}
	if g == nil || g.approvals == nil {
		return ErrCapabilityApprovalRequired
	}

	approvalID := capabilityContextValue(ctx, ctxkeys.ApprovalID)
	workspaceID := capabilityContextValue(ctx, ctxkeys.WorkspaceID)
	executionID := capabilityContextValue(ctx, ctxkeys.ExecutionID)
	if approvalID == "" || workspaceID == "" || executionID == "" {
		return ErrCapabilityApprovalRequired
	}

	req, err := g.approvals.getApprovalByID(ctx, approvalID)
	if err != nil {
		return fmt.Errorf("load capability approval: %w", err)
	}
	if req.Status != ApprovalStatusApproved {
		return ErrCapabilityApprovalPending
	}
	if req.WorkspaceID != workspaceID ||
		req.Action != tool.CapabilityApprovalAction(descriptor) ||
		!matchesApprovalResource(req, executionID) {
		return ErrCapabilityApprovalMismatch
	}
	return nil
}

func matchesApprovalResource(req *ApprovalRequest, executionID string) bool {
	return req.ResourceType != nil &&
		req.ResourceID != nil &&
		*req.ResourceType == tool.CapabilityApprovalResourceType &&
		*req.ResourceID == executionID
}

func capabilityContextValue(ctx context.Context, key ctxkeys.Key) string {
	value, _ := ctx.Value(key).(string)
	return strings.TrimSpace(value)
}
