package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusDenied   ApprovalStatus = "denied"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

var (
	ErrApprovalNotFound      = errors.New("approval request not found")
	ErrApprovalForbidden     = errors.New("approval request does not belong to approver")
	ErrApprovalAlreadyClosed = errors.New("approval request is already decided")
	ErrApprovalExpired       = errors.New("approval request is expired")
	ErrInvalidDecision       = errors.New("invalid approval decision")
)

type ApprovalRequest struct {
	ID           string
	WorkspaceID  string
	RequestedBy  string
	ApproverID   string
	DecidedBy    *string
	Action       string
	ResourceType *string
	ResourceID   *string
	Payload      json.RawMessage
	Reason       *string
	Status       ApprovalStatus
	ExpiresAt    time.Time
	DecidedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateApprovalRequestInput struct {
	WorkspaceID  string
	RequestedBy  string
	ApproverID   string
	Action       string
	ResourceType *string
	ResourceID   *string
	Payload      json.RawMessage
	Reason       *string
	ExpiresAt    time.Time
}

type ApprovalService struct {
	db    *sql.DB
	audit *audit.AuditService
}

func NewApprovalService(db *sql.DB, auditService *audit.AuditService) *ApprovalService {
	if auditService == nil {
		auditService = audit.NewAuditService(db)
	}
	return &ApprovalService{db: db, audit: auditService}
}

func (s *ApprovalService) CreateApprovalRequest(ctx context.Context, input CreateApprovalRequestInput) (*ApprovalRequest, error) {
	if len(input.Payload) == 0 {
		input.Payload = json.RawMessage(`{}`)
	}

	now := time.Now().UTC()
	approval := &ApprovalRequest{
		ID:           uuid.NewV7().String(),
		WorkspaceID:  input.WorkspaceID,
		RequestedBy:  input.RequestedBy,
		ApproverID:   input.ApproverID,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		Payload:      input.Payload,
		Reason:       input.Reason,
		Status:       ApprovalStatusPending,
		ExpiresAt:    input.ExpiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO approval_request (
			id, workspace_id, requested_by, approver_id, decided_by,
			action, resource_type, resource_id, payload, reason,
			status, expires_at, decided_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)
	`,
		approval.ID,
		approval.WorkspaceID,
		approval.RequestedBy,
		approval.ApproverID,
		approval.Action,
		approval.ResourceType,
		approval.ResourceID,
		[]byte(approval.Payload),
		approval.Reason,
		string(approval.Status),
		approval.ExpiresAt,
		approval.CreatedAt,
		approval.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	_ = s.audit.LogWithDetails(
		ctx,
		approval.WorkspaceID,
		approval.RequestedBy,
		audit.ActorTypeUser,
		"approval.requested",
		approval.ResourceType,
		approval.ResourceID,
		&audit.EventDetails{Metadata: map[string]any{"approval_id": approval.ID, "action": approval.Action}},
		audit.OutcomeSuccess,
	)

	return approval, nil
}

func (s *ApprovalService) DecideApprovalRequest(ctx context.Context, id, decision, decidedBy string) error {
	status := decisionToStatus(decision)
	if status == "" {
		return ErrInvalidDecision
	}

	req, err := s.getApprovalByID(ctx, id)
	if err != nil {
		return err
	}

	if err := validateApprovalDecision(req, decidedBy); err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := s.expireIfNeeded(ctx, req, id, decidedBy, now); err != nil {
		return err
	}

	return s.applyDecision(ctx, req, id, decidedBy, status, now)
}

func (s *ApprovalService) GetPendingApprovals(ctx context.Context, userID string) ([]*ApprovalRequest, error) {
	now := time.Now().UTC()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, requested_by, approver_id, decided_by,
		       action, resource_type, resource_id, payload, reason,
		       status, expires_at, decided_at, created_at, updated_at
		FROM approval_request
		WHERE approver_id = ? AND status = ?
		ORDER BY created_at ASC
	`, userID, string(ApprovalStatusPending))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, expiredIDs, err := collectPendingApprovals(rows, now)
	if err != nil {
		return nil, err
	}

	if err := s.markApprovalsExpired(ctx, expiredIDs, now); err != nil {
		return nil, err
	}

	return items, nil
}

func validateApprovalDecision(req *ApprovalRequest, decidedBy string) error {
	if req.ApproverID != decidedBy {
		return ErrApprovalForbidden
	}
	if req.Status == ApprovalStatusExpired {
		return ErrApprovalExpired
	}
	if req.Status != ApprovalStatusPending {
		return ErrApprovalAlreadyClosed
	}
	return nil
}

func (s *ApprovalService) expireIfNeeded(ctx context.Context, req *ApprovalRequest, id, decidedBy string, now time.Time) error {
	if req.ExpiresAt.After(now) {
		return nil
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE approval_request
		SET status = ?, decided_at = ?, updated_at = ?, decided_by = ?
		WHERE id = ?
	`, string(ApprovalStatusExpired), now, now, decidedBy, id); err != nil {
		return err
	}

	_ = s.audit.LogWithDetails(
		ctx,
		req.WorkspaceID,
		decidedBy,
		audit.ActorTypeUser,
		"approval.expired",
		req.ResourceType,
		req.ResourceID,
		&audit.EventDetails{Metadata: map[string]any{"approval_id": id}},
		audit.OutcomeSuccess,
	)

	return ErrApprovalExpired
}

func (s *ApprovalService) applyDecision(ctx context.Context, req *ApprovalRequest, id, decidedBy string, status ApprovalStatus, now time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE approval_request
		SET status = ?, decided_by = ?, decided_at = ?, updated_at = ?
		WHERE id = ? AND status = ? AND approver_id = ?
	`, string(status), decidedBy, now, now, id, string(ApprovalStatusPending), decidedBy)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrApprovalAlreadyClosed
	}

	action := "approval.denied"
	if status == ApprovalStatusApproved {
		action = "approval.approved"
	}

	_ = s.audit.LogWithDetails(
		ctx,
		req.WorkspaceID,
		decidedBy,
		audit.ActorTypeUser,
		action,
		req.ResourceType,
		req.ResourceID,
		&audit.EventDetails{Metadata: map[string]any{"approval_id": id}},
		audit.OutcomeSuccess,
	)

	return nil
}

func collectPendingApprovals(rows *sql.Rows, now time.Time) ([]*ApprovalRequest, []string, error) {
	items := make([]*ApprovalRequest, 0)
	expiredIDs := make([]string, 0)
	for rows.Next() {
		item, err := scanApprovalRequest(rows)
		if err != nil {
			return nil, nil, err
		}
		if !item.ExpiresAt.After(now) {
			expiredIDs = append(expiredIDs, item.ID)
			continue
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, expiredIDs, nil
}

func (s *ApprovalService) markApprovalsExpired(ctx context.Context, expiredIDs []string, now time.Time) error {
	for _, id := range expiredIDs {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE approval_request
			SET status = ?, updated_at = ?
			WHERE id = ?
		`, string(ApprovalStatusExpired), now, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *ApprovalService) getApprovalByID(ctx context.Context, id string) (*ApprovalRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, requested_by, approver_id, decided_by,
		       action, resource_type, resource_id, payload, reason,
		       status, expires_at, decided_at, created_at, updated_at
		FROM approval_request
		WHERE id = ?
	`, id)

	item, err := scanApprovalRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrApprovalNotFound
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

type approvalScanner interface {
	Scan(dest ...any) error
}

func scanApprovalRequest(scan approvalScanner) (*ApprovalRequest, error) {
	var (
		item         ApprovalRequest
		decidedByRaw sql.NullString
		resourceType sql.NullString
		resourceID   sql.NullString
		reason       sql.NullString
		payload      []byte
		decidedAtRaw sql.NullTime
	)

	if err := scan.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.RequestedBy,
		&item.ApproverID,
		&decidedByRaw,
		&item.Action,
		&resourceType,
		&resourceID,
		&payload,
		&reason,
		&item.Status,
		&item.ExpiresAt,
		&decidedAtRaw,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.Payload = payload
	if decidedByRaw.Valid {
		v := decidedByRaw.String
		item.DecidedBy = &v
	}
	if resourceType.Valid {
		v := resourceType.String
		item.ResourceType = &v
	}
	if resourceID.Valid {
		v := resourceID.String
		item.ResourceID = &v
	}
	if reason.Valid {
		v := reason.String
		item.Reason = &v
	}
	if decidedAtRaw.Valid {
		v := decidedAtRaw.Time
		item.DecidedAt = &v
	}

	return &item, nil
}

func decisionToStatus(decision string) ApprovalStatus {
	switch decision {
	case "approve", "approved":
		return ApprovalStatusApproved
	case "deny", "denied":
		return ApprovalStatusDenied
	default:
		return ""
	}
}
