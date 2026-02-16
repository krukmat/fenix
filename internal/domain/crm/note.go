package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type Note struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	EntityType  string    `json:"entityType"`
	EntityID    string    `json:"entityId"`
	AuthorID    string    `json:"authorId"`
	Content     string    `json:"content"`
	IsInternal  bool      `json:"isInternal"`
	Metadata    *string   `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateNoteInput struct {
	WorkspaceID string
	EntityType  string
	EntityID    string
	AuthorID    string
	Content     string
	IsInternal  bool
	Metadata    string
}

type UpdateNoteInput struct {
	Content    string
	IsInternal bool
	Metadata   string
}

type ListNotesInput struct {
	Limit  int
	Offset int
}

type NoteService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

func NewNoteService(db *sql.DB) *NoteService {
	return &NoteService{db: db, querier: sqlcgen.New(db)}
}

func (s *NoteService) Create(ctx context.Context, input CreateNoteInput) (*Note, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.querier.CreateNote(ctx, sqlcgen.CreateNoteParams{
		ID:          id,
		WorkspaceID: input.WorkspaceID,
		EntityType:  input.EntityType,
		EntityID:    input.EntityID,
		AuthorID:    input.AuthorID,
		Content:     input.Content,
		IsInternal:  input.IsInternal,
		Metadata:    nullString(input.Metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, input.WorkspaceID, input.EntityType, input.EntityID, input.AuthorID, "note_created"); timelineErr != nil {
		return nil, fmt.Errorf("create note timeline: %w", timelineErr)
	}
	return s.Get(ctx, input.WorkspaceID, id)
}

func (s *NoteService) Get(ctx context.Context, workspaceID, noteID string) (*Note, error) {
	row, err := s.querier.GetNoteByID(ctx, sqlcgen.GetNoteByIDParams{ID: noteID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	return rowToNote(row), nil
}

func (s *NoteService) List(ctx context.Context, workspaceID string, input ListNotesInput) ([]*Note, int, error) {
	total, err := s.querier.CountNotesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count notes: %w", err)
	}
	rows, err := s.querier.ListNotesByWorkspace(ctx, sqlcgen.ListNotesByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list notes: %w", err)
	}
	out := make([]*Note, len(rows))
	for i := range rows {
		out[i] = rowToNote(rows[i])
	}
	return out, int(total), nil
}

func (s *NoteService) Update(ctx context.Context, workspaceID, noteID string, input UpdateNoteInput) (*Note, error) {
	err := s.querier.UpdateNote(ctx, sqlcgen.UpdateNoteParams{
		Content:     input.Content,
		IsInternal:  input.IsInternal,
		Metadata:    nullString(input.Metadata),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		ID:          noteID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	existing, getErr := s.Get(ctx, workspaceID, noteID)
	if getErr == nil {
		if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, existing.EntityType, existing.EntityID, existing.AuthorID, "note_updated"); timelineErr != nil {
			return nil, fmt.Errorf("update note timeline: %w", timelineErr)
		}
	}
	return s.Get(ctx, workspaceID, noteID)
}

func (s *NoteService) Delete(ctx context.Context, workspaceID, noteID string) error {
	existing, _ := s.Get(ctx, workspaceID, noteID)
	err := s.querier.DeleteNote(ctx, sqlcgen.DeleteNoteParams{ID: noteID, WorkspaceID: workspaceID})
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if existing != nil {
		if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, existing.EntityType, existing.EntityID, existing.AuthorID, "note_deleted"); timelineErr != nil {
			return fmt.Errorf("delete note timeline: %w", timelineErr)
		}
	}
	return nil
}

func rowToNote(row sqlcgen.Note) *Note {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)
	return &Note{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		EntityType:  row.EntityType,
		EntityID:    row.EntityID,
		AuthorID:    row.AuthorID,
		Content:     row.Content,
		IsInternal:  row.IsInternal,
		Metadata:    row.Metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
