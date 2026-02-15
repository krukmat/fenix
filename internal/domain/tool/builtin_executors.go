package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var ErrBuiltinExecutionFailed = errors.New("builtin tool execution failed")

type CreateTaskExecutor struct{ db *sql.DB }

func NewCreateTaskExecutor(db *sql.DB) ToolExecutor {
	return &CreateTaskExecutor{db: db}
}

type createTaskParams struct {
	OwnerID    string `json:"owner_id"`
	Title      string `json:"title"`
	DueDate    string `json:"due_date"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

func (e *CreateTaskExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	if e.db == nil {
		return nil, fmt.Errorf("%w: db not configured", ErrBuiltinExecutionFailed)
	}

	var in createTaskParams
	if err := json.Unmarshal(params, &in); err != nil {
		return nil, fmt.Errorf("%w: invalid params", ErrBuiltinExecutionFailed)
	}
	if in.OwnerID == "" || in.Title == "" || in.EntityType == "" || in.EntityID == "" {
		return nil, fmt.Errorf("%w: owner_id, title, entity_type and entity_id are required", ErrBuiltinExecutionFailed)
	}

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	taskID := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = e.db.ExecContext(ctx, `
		INSERT INTO activity (
			id, workspace_id, activity_type, entity_type, entity_id,
			owner_id, subject, status, due_at, created_at, updated_at
		) VALUES (?, ?, 'task', ?, ?, ?, ?, 'pending', ?, ?, ?)
	`, taskID, workspaceID, in.EntityType, in.EntityID, in.OwnerID, in.Title, nullableString(in.DueDate), now, now)
	if err != nil {
		return nil, fmt.Errorf("%w: create activity: %v", ErrBuiltinExecutionFailed, err)
	}

	out, _ := json.Marshal(map[string]any{
		"task_id":    taskID,
		"created_at": now,
	})
	return out, nil
}

type UpdateCaseExecutor struct{ cases *crm.CaseService }

func NewUpdateCaseExecutor(cases *crm.CaseService) ToolExecutor {
	return &UpdateCaseExecutor{cases: cases}
}

type updateCaseParams struct {
	CaseID   string   `json:"case_id"`
	Status   string   `json:"status"`
	Priority string   `json:"priority"`
	Tags     []string `json:"tags"`
}

func (e *UpdateCaseExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	if e.cases == nil {
		return nil, fmt.Errorf("%w: case service not configured", ErrBuiltinExecutionFailed)
	}

	var in updateCaseParams
	if err := json.Unmarshal(params, &in); err != nil {
		return nil, fmt.Errorf("%w: invalid params", ErrBuiltinExecutionFailed)
	}
	if in.CaseID == "" {
		return nil, fmt.Errorf("%w: case_id is required", ErrBuiltinExecutionFailed)
	}

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := e.cases.Get(ctx, workspaceID, in.CaseID)
	if err != nil {
		return nil, fmt.Errorf("%w: case not found", ErrBuiltinExecutionFailed)
	}

	metadata := ""
	if len(in.Tags) > 0 {
		raw, _ := json.Marshal(map[string]any{"tags": in.Tags})
		metadata = string(raw)
	}

	updated, err := e.cases.Update(ctx, workspaceID, in.CaseID, crm.UpdateCaseInput{
		AccountID:   derefString(existing.AccountID),
		ContactID:   derefString(existing.ContactID),
		PipelineID:  derefString(existing.PipelineID),
		StageID:     derefString(existing.StageID),
		OwnerID:     existing.OwnerID,
		Subject:     existing.Subject,
		Description: derefString(existing.Description),
		Priority:    firstNonEmpty(in.Priority, existing.Priority),
		Status:      firstNonEmpty(in.Status, existing.Status),
		Channel:     derefString(existing.Channel),
		SlaConfig:   derefString(existing.SlaConfig),
		SlaDeadline: derefString(existing.SlaDeadline),
		Metadata:    firstNonEmpty(metadata, derefString(existing.Metadata)),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: update case: %v", ErrBuiltinExecutionFailed, err)
	}

	out, _ := json.Marshal(map[string]any{
		"case_id":    updated.ID,
		"updated_at": updated.UpdatedAt.Format(time.RFC3339),
	})
	return out, nil
}

type SendReplyExecutor struct {
	db    *sql.DB
	cases *crm.CaseService
}

func NewSendReplyExecutor(db *sql.DB, cases *crm.CaseService) ToolExecutor {
	return &SendReplyExecutor{db: db, cases: cases}
}

type sendReplyParams struct {
	CaseID     string `json:"case_id"`
	Body       string `json:"body"`
	IsInternal bool   `json:"is_internal"`
}

func (e *SendReplyExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	if e.cases == nil || e.db == nil {
		return nil, fmt.Errorf("%w: case service or db not configured", ErrBuiltinExecutionFailed)
	}

	var in sendReplyParams
	if err := json.Unmarshal(params, &in); err != nil {
		return nil, fmt.Errorf("%w: invalid params", ErrBuiltinExecutionFailed)
	}
	if in.CaseID == "" || in.Body == "" {
		return nil, fmt.Errorf("%w: case_id and body are required", ErrBuiltinExecutionFailed)
	}

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	caseTicket, err := e.cases.Get(ctx, workspaceID, in.CaseID)
	if err != nil {
		return nil, fmt.Errorf("%w: case not found", ErrBuiltinExecutionFailed)
	}

	authorID := firstNonEmpty(userIDFromContext(ctx), caseTicket.OwnerID)

	noteID := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = e.db.ExecContext(ctx, `
		INSERT INTO note (
			id, workspace_id, entity_type, entity_id, author_id,
			content, is_internal, created_at, updated_at
		) VALUES (?, ?, 'case', ?, ?, ?, ?, ?, ?)
	`, noteID, workspaceID, in.CaseID, authorID, in.Body, in.IsInternal, now, now)
	if err != nil {
		return nil, fmt.Errorf("%w: create note: %v", ErrBuiltinExecutionFailed, err)
	}

	out, _ := json.Marshal(map[string]any{
		"note_id":    noteID,
		"created_at": now,
	})
	return out, nil
}

func workspaceIDFromContext(ctx context.Context) (string, error) {
	workspaceID, ok := ctx.Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		return "", fmt.Errorf("%w: missing workspace_id in context", ErrBuiltinExecutionFailed)
	}
	return workspaceID, nil
}

func userIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(ctxkeys.UserID).(string)
	return userID
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
