package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type Attachment struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	EntityType  string    `json:"entityType"`
	EntityID    string    `json:"entityId"`
	UploaderID  string    `json:"uploaderId"`
	Filename    string    `json:"filename"`
	ContentType *string   `json:"contentType,omitempty"`
	SizeBytes   *int64    `json:"sizeBytes,omitempty"`
	StoragePath string    `json:"storagePath"`
	Sensitivity *string   `json:"sensitivity,omitempty"`
	Metadata    *string   `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CreateAttachmentInput struct {
	WorkspaceID string
	EntityType  string
	EntityID    string
	UploaderID  string
	Filename    string
	ContentType string
	SizeBytes   *int64
	StoragePath string
	Sensitivity string
	Metadata    string
}

type ListAttachmentsInput struct {
	Limit  int
	Offset int
}

type AttachmentService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

func NewAttachmentService(db *sql.DB) *AttachmentService {
	return &AttachmentService{db: db, querier: sqlcgen.New(db)}
}

func (s *AttachmentService) Create(ctx context.Context, input CreateAttachmentInput) (*Attachment, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.querier.CreateAttachment(ctx, sqlcgen.CreateAttachmentParams{
		ID:          id,
		WorkspaceID: input.WorkspaceID,
		EntityType:  input.EntityType,
		EntityID:    input.EntityID,
		UploaderID:  input.UploaderID,
		Filename:    input.Filename,
		ContentType: nullString(input.ContentType),
		SizeBytes:   input.SizeBytes,
		StoragePath: input.StoragePath,
		Sensitivity: nullString(input.Sensitivity),
		Metadata:    nullString(input.Metadata),
		CreatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("create attachment: %w", err)
	}
	if err := createTimelineEvent(ctx, s.querier, input.WorkspaceID, input.EntityType, input.EntityID, input.UploaderID, "attachment_created"); err != nil {
		return nil, fmt.Errorf("create attachment timeline: %w", err)
	}
	return s.Get(ctx, input.WorkspaceID, id)
}

func (s *AttachmentService) Get(ctx context.Context, workspaceID, attachmentID string) (*Attachment, error) {
	row, err := s.querier.GetAttachmentByID(ctx, sqlcgen.GetAttachmentByIDParams{ID: attachmentID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	return rowToAttachment(row), nil
}

func (s *AttachmentService) List(ctx context.Context, workspaceID string, input ListAttachmentsInput) ([]*Attachment, int, error) {
	total, err := s.querier.CountAttachmentsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count attachments: %w", err)
	}
	rows, err := s.querier.ListAttachmentsByWorkspace(ctx, sqlcgen.ListAttachmentsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list attachments: %w", err)
	}
	out := make([]*Attachment, len(rows))
	for i := range rows {
		out[i] = rowToAttachment(rows[i])
	}
	return out, int(total), nil
}

func (s *AttachmentService) Delete(ctx context.Context, workspaceID, attachmentID string) error {
	existing, _ := s.Get(ctx, workspaceID, attachmentID)
	err := s.querier.DeleteAttachment(ctx, sqlcgen.DeleteAttachmentParams{ID: attachmentID, WorkspaceID: workspaceID})
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	if existing != nil {
		if err := createTimelineEvent(ctx, s.querier, workspaceID, existing.EntityType, existing.EntityID, existing.UploaderID, "attachment_deleted"); err != nil {
			return fmt.Errorf("delete attachment timeline: %w", err)
		}
	}
	return nil
}

func rowToAttachment(row sqlcgen.Attachment) *Attachment {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	return &Attachment{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		EntityType:  row.EntityType,
		EntityID:    row.EntityID,
		UploaderID:  row.UploaderID,
		Filename:    row.Filename,
		ContentType: row.ContentType,
		SizeBytes:   row.SizeBytes,
		StoragePath: row.StoragePath,
		Sensitivity: row.Sensitivity,
		Metadata:    row.Metadata,
		CreatedAt:   createdAt,
	}
}
