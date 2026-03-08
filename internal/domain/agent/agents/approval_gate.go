package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
)

type approvalGateInput struct {
	WorkspaceID  string
	RequestedBy  string
	ApproverID   string
	Action       string
	ResourceType string
	ResourceID   string
	Reason       string
	Payload      map[string]any
	TTL          time.Duration
}

func createApprovalGateRequest(ctx context.Context, db *sql.DB, in approvalGateInput) (string, error) {
	if db == nil {
		return "", errors.New("db is required")
	}
	if in.TTL <= 0 {
		in.TTL = 24 * time.Hour
	}
	payload, _ := json.Marshal(in.Payload)
	resourceType := in.ResourceType
	resourceID := in.ResourceID
	reason := in.Reason

	svc := policy.NewApprovalService(db, nil)
	req, err := svc.CreateApprovalRequest(ctx, policy.CreateApprovalRequestInput{
		WorkspaceID:  in.WorkspaceID,
		RequestedBy:  in.RequestedBy,
		ApproverID:   in.ApproverID,
		Action:       in.Action,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Payload:      payload,
		Reason:       &reason,
		ExpiresAt:    time.Now().UTC().Add(in.TTL),
	})
	if err != nil {
		return "", err
	}
	return req.ID, nil
}

func requesterFromCtxOrDefault(ctx context.Context, fallback string) string {
	if userID, _ := ctx.Value(ctxkeys.UserID).(string); userID != "" {
		return userID
	}
	return fallback
}

func isHighSensitivityMetadata(metadata *string) bool {
	if metadata == nil {
		return false
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(*metadata), &m); err != nil {
		return false
	}
	return m["sensitivity"] == sensitivityHigh
}
