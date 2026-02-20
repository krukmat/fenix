package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var ErrBuiltinExecutionFailed = errors.New("builtin tool execution failed")

const errInvalidParams = "%w: invalid params"

const errDBNotConfigured = "%w: db not configured"

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
	in, err := parseCreateTaskParams(params)
	if err != nil {
		return nil, err
	}
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	taskID, createdAt, err := e.insertTaskActivity(ctx, workspaceID, in)
	if err != nil {
		return nil, err
	}
	return marshalTaskCreated(taskID, createdAt), nil
}

func parseCreateTaskParams(params json.RawMessage) (createTaskParams, error) {
	var in createTaskParams
	if err := json.Unmarshal(params, &in); err != nil {
		return createTaskParams{}, fmt.Errorf(errInvalidParams, ErrBuiltinExecutionFailed)
	}
	if in.OwnerID == "" || in.Title == "" || in.EntityType == "" || in.EntityID == "" {
		return createTaskParams{}, fmt.Errorf("%w: owner_id, title, entity_type and entity_id are required", ErrBuiltinExecutionFailed)
	}
	return in, nil
}

func (e *CreateTaskExecutor) insertTaskActivity(ctx context.Context, workspaceID string, in createTaskParams) (string, string, error) {
	if e.db == nil {
		return "", "", fmt.Errorf(errDBNotConfigured, ErrBuiltinExecutionFailed)
	}
	taskID := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO activity (
			id, workspace_id, activity_type, entity_type, entity_id,
			owner_id, subject, status, due_at, created_at, updated_at
		) VALUES (?, ?, 'task', ?, ?, ?, ?, 'pending', ?, ?, ?)
	`, taskID, workspaceID, in.EntityType, in.EntityID, in.OwnerID, in.Title, nullableString(in.DueDate), now, now)
	if err != nil {
		return "", "", fmt.Errorf("%w: create activity: %v", ErrBuiltinExecutionFailed, err)
	}
	return taskID, now, nil
}

func marshalTaskCreated(taskID, createdAt string) json.RawMessage {
	out, _ := json.Marshal(map[string]any{"task_id": taskID, "created_at": createdAt})
	return out
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
	in, err := parseUpdateCaseParams(params)
	if err != nil {
		return nil, err
	}
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := e.updateCase(ctx, workspaceID, in)
	if err != nil {
		return nil, err
	}
	return marshalCaseUpdated(updated), nil
}

func parseUpdateCaseParams(params json.RawMessage) (updateCaseParams, error) {
	var in updateCaseParams
	if err := json.Unmarshal(params, &in); err != nil {
		return updateCaseParams{}, fmt.Errorf(errInvalidParams, ErrBuiltinExecutionFailed)
	}
	if in.CaseID == "" {
		return updateCaseParams{}, fmt.Errorf("%w: case_id is required", ErrBuiltinExecutionFailed)
	}
	return in, nil
}

func (e *UpdateCaseExecutor) updateCase(ctx context.Context, workspaceID string, in updateCaseParams) (*crm.CaseTicket, error) {
	if e.cases == nil {
		return nil, fmt.Errorf("%w: case service not configured", ErrBuiltinExecutionFailed)
	}
	existing, err := e.cases.Get(ctx, workspaceID, in.CaseID)
	if err != nil {
		return nil, fmt.Errorf("%w: case not found", ErrBuiltinExecutionFailed)
	}
	updated, err := e.cases.Update(ctx, workspaceID, in.CaseID, buildUpdateCaseInput(existing, in))
	if err != nil {
		return nil, fmt.Errorf("%w: update case: %v", ErrBuiltinExecutionFailed, err)
	}
	return updated, nil
}

func buildUpdateCaseInput(existing *crm.CaseTicket, in updateCaseParams) crm.UpdateCaseInput {
	return crm.UpdateCaseInput{
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
		SLAConfig:   derefString(existing.SLAConfig),
		SLADeadline: derefString(existing.SLADeadline),
		Metadata:    firstNonEmpty(metadataFromTags(in.Tags), derefString(existing.Metadata)),
	}
}

func metadataFromTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	raw, _ := json.Marshal(map[string]any{"tags": tags})
	return string(raw)
}

func marshalCaseUpdated(updated *crm.CaseTicket) json.RawMessage {
	out, _ := json.Marshal(map[string]any{"case_id": updated.ID, "updated_at": updated.UpdatedAt.Format(time.RFC3339)})
	return out
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
	in, err := parseSendReplyParams(params)
	if err != nil {
		return nil, err
	}
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	noteID, createdAt, err := e.insertReplyNote(ctx, workspaceID, in)
	if err != nil {
		return nil, err
	}
	return marshalReplyCreated(noteID, createdAt), nil
}

func parseSendReplyParams(params json.RawMessage) (sendReplyParams, error) {
	var in sendReplyParams
	if err := json.Unmarshal(params, &in); err != nil {
		return sendReplyParams{}, fmt.Errorf(errInvalidParams, ErrBuiltinExecutionFailed)
	}
	if in.CaseID == "" || in.Body == "" {
		return sendReplyParams{}, fmt.Errorf("%w: case_id and body are required", ErrBuiltinExecutionFailed)
	}
	return in, nil
}

func (e *SendReplyExecutor) insertReplyNote(ctx context.Context, workspaceID string, in sendReplyParams) (string, string, error) {
	if e.cases == nil || e.db == nil {
		return "", "", fmt.Errorf("%w: case service or db not configured", ErrBuiltinExecutionFailed)
	}
	caseTicket, err := e.cases.Get(ctx, workspaceID, in.CaseID)
	if err != nil {
		return "", "", fmt.Errorf("%w: case not found", ErrBuiltinExecutionFailed)
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
		return "", "", fmt.Errorf("%w: create note: %v", ErrBuiltinExecutionFailed, err)
	}
	return noteID, now, nil
}

func marshalReplyCreated(noteID, createdAt string) json.RawMessage {
	out, _ := json.Marshal(map[string]any{"note_id": noteID, "created_at": createdAt})
	return out
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

// Task 4.5a — GetLeadExecutor
type GetLeadExecutor struct{ leads *crm.LeadService }

func NewGetLeadExecutor(leads *crm.LeadService) ToolExecutor {
	return &GetLeadExecutor{leads: leads}
}

type getLeadParams struct {
	LeadID string `json:"lead_id"`
}

func (e *GetLeadExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	var in getLeadParams
	if err := json.Unmarshal(params, &in); err != nil {
		return nil, fmt.Errorf(errInvalidParams, ErrBuiltinExecutionFailed)
	}
	if in.LeadID == "" {
		return nil, fmt.Errorf("%w: lead_id is required", ErrBuiltinExecutionFailed)
	}
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if e.leads == nil {
		return nil, fmt.Errorf("%w: lead service not configured", ErrBuiltinExecutionFailed)
	}
	lead, err := e.leads.Get(ctx, workspaceID, in.LeadID)
	if err != nil {
		return nil, fmt.Errorf("%w: lead not found", ErrBuiltinExecutionFailed)
	}
	out, _ := json.Marshal(map[string]any{"lead": lead})
	return out, nil
}

// Task 4.5a — GetAccountExecutor
type GetAccountExecutor struct{ accounts *crm.AccountService }

func NewGetAccountExecutor(accounts *crm.AccountService) ToolExecutor {
	return &GetAccountExecutor{accounts: accounts}
}

type getAccountParams struct {
	AccountID string `json:"account_id"`
}

func (e *GetAccountExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	var in getAccountParams
	if err := json.Unmarshal(params, &in); err != nil {
		return nil, fmt.Errorf(errInvalidParams, ErrBuiltinExecutionFailed)
	}
	if in.AccountID == "" {
		return nil, fmt.Errorf("%w: account_id is required", ErrBuiltinExecutionFailed)
	}
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if e.accounts == nil {
		return nil, fmt.Errorf("%w: account service not configured", ErrBuiltinExecutionFailed)
	}
	account, err := e.accounts.Get(ctx, workspaceID, in.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: account not found", ErrBuiltinExecutionFailed)
	}
	out, _ := json.Marshal(map[string]any{"account": account})
	return out, nil
}

// Task 4.5a — CreateKnowledgeItemExecutor
type CreateKnowledgeItemExecutor struct{ ingest knowledgeIngestor }

func NewCreateKnowledgeItemExecutor(ingest knowledgeIngestor) ToolExecutor {
	return &CreateKnowledgeItemExecutor{ingest: ingest}
}

type createKnowledgeItemParams struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	SourceType  string `json:"source_type"`
	WorkspaceID string `json:"workspace_id"`
}

func (e *CreateKnowledgeItemExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	in, err := parseCreateKnowledgeItemParams(params)
	if err != nil {
		return nil, err
	}
	workspaceID, err := resolveWorkspaceID(ctx, in.WorkspaceID)
	if err != nil {
		return nil, err
	}
	item, err := e.createKnowledgeItem(ctx, workspaceID, in)
	if err != nil {
		return nil, err
	}
	out, _ := json.Marshal(map[string]any{"knowledge_item_id": item.ID, "created_at": item.CreatedAt.Format(time.RFC3339)})
	return out, nil
}

func parseCreateKnowledgeItemParams(params json.RawMessage) (createKnowledgeItemParams, error) {
	var in createKnowledgeItemParams
	if err := json.Unmarshal(params, &in); err != nil {
		return createKnowledgeItemParams{}, fmt.Errorf(errInvalidParams, ErrBuiltinExecutionFailed)
	}
	if in.Title == "" || in.Content == "" || in.SourceType == "" {
		return createKnowledgeItemParams{}, fmt.Errorf("%w: title, content and source_type are required", ErrBuiltinExecutionFailed)
	}
	return in, nil
}

func (e *CreateKnowledgeItemExecutor) createKnowledgeItem(
	ctx context.Context,
	workspaceID string,
	in createKnowledgeItemParams,
) (*knowledge.KnowledgeItem, error) {
	if e.ingest == nil {
		return nil, fmt.Errorf("%w: ingest service not configured", ErrBuiltinExecutionFailed)
	}
	item, err := e.ingest.Ingest(ctx, knowledge.CreateKnowledgeItemInput{
		WorkspaceID: workspaceID,
		SourceType:  knowledge.SourceType(in.SourceType),
		Title:       in.Title,
		RawContent:  in.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: create knowledge item: %v", ErrBuiltinExecutionFailed, err)
	}
	return item, nil
}

// Task 4.5a — UpdateKnowledgeItemExecutor
type UpdateKnowledgeItemExecutor struct{ db *sql.DB }

func NewUpdateKnowledgeItemExecutor(db *sql.DB) ToolExecutor {
	return &UpdateKnowledgeItemExecutor{db: db}
}

type updateKnowledgeItemParams struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (e *UpdateKnowledgeItemExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	in, err := parseUpdateKnowledgeItemParams(params)
	if err != nil {
		return nil, err
	}
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	updateErr := e.updateKnowledgeItem(ctx, workspaceID, in)
	if updateErr != nil {
		return nil, updateErr
	}
	out, _ := json.Marshal(map[string]any{"knowledge_item_id": in.ID, "updated_at": time.Now().UTC().Format(time.RFC3339)})
	return out, nil
}

func parseUpdateKnowledgeItemParams(params json.RawMessage) (updateKnowledgeItemParams, error) {
	var in updateKnowledgeItemParams
	if err := json.Unmarshal(params, &in); err != nil {
		return updateKnowledgeItemParams{}, fmt.Errorf(errInvalidParams, ErrBuiltinExecutionFailed)
	}
	if in.ID == "" {
		return updateKnowledgeItemParams{}, fmt.Errorf("%w: id is required", ErrBuiltinExecutionFailed)
	}
	if in.Title == "" && in.Content == "" {
		return updateKnowledgeItemParams{}, fmt.Errorf("%w: title or content is required", ErrBuiltinExecutionFailed)
	}
	return in, nil
}

func (e *UpdateKnowledgeItemExecutor) updateKnowledgeItem(ctx context.Context, workspaceID string, in updateKnowledgeItemParams) error {
	if e.db == nil {
		return fmt.Errorf(errDBNotConfigured, ErrBuiltinExecutionFailed)
	}
	res, err := e.db.ExecContext(ctx, `
		UPDATE knowledge_item
		SET title = COALESCE(NULLIF(?, ''), title),
		    raw_content = COALESCE(NULLIF(?, ''), raw_content),
		    normalized_content = COALESCE(NULLIF(?, ''), normalized_content),
		    updated_at = ?
		WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL
	`, in.Title, in.Content, strings.TrimSpace(in.Content), time.Now().UTC(), in.ID, workspaceID)
	if err != nil {
		return fmt.Errorf("%w: update knowledge item: %v", ErrBuiltinExecutionFailed, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("%w: knowledge item not found", ErrBuiltinExecutionFailed)
	}
	return nil
}

// Task 4.5a — QueryMetricsExecutor
type QueryMetricsExecutor struct{ db *sql.DB }

func NewQueryMetricsExecutor(db *sql.DB) ToolExecutor {
	return &QueryMetricsExecutor{db: db}
}

type queryMetricsParams struct {
	Metric      string `json:"metric"`
	WorkspaceID string `json:"workspace_id"`
	From        string `json:"from"`
	To          string `json:"to"`
}

func (e *QueryMetricsExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	in, err := parseQueryMetricsParams(params)
	if err != nil {
		return nil, err
	}
	workspaceID, err := resolveWorkspaceID(ctx, in.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if e.db == nil {
		return nil, fmt.Errorf(errDBNotConfigured, ErrBuiltinExecutionFailed)
	}
	data, err := e.queryMetric(ctx, workspaceID, in)
	if err != nil {
		return nil, err
	}
	out, _ := json.Marshal(map[string]any{
		"metric": in.Metric,
		"data":   data,
	})
	return out, nil
}

func parseQueryMetricsParams(params json.RawMessage) (queryMetricsParams, error) {
	var in queryMetricsParams
	if err := json.Unmarshal(params, &in); err != nil {
		return queryMetricsParams{}, fmt.Errorf(errInvalidParams, ErrBuiltinExecutionFailed)
	}
	if in.Metric == "" {
		return queryMetricsParams{}, fmt.Errorf("%w: metric is required", ErrBuiltinExecutionFailed)
	}
	return in, nil
}

func resolveWorkspaceID(ctx context.Context, payloadWorkspaceID string) (string, error) {
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	if payloadWorkspaceID != "" && payloadWorkspaceID != workspaceID {
		return "", fmt.Errorf("%w: workspace_id mismatch", ErrBuiltinExecutionFailed)
	}
	return workspaceID, nil
}

func (e *QueryMetricsExecutor) queryMetric(ctx context.Context, workspaceID string, in queryMetricsParams) ([]map[string]any, error) {
	from, to := in.From, in.To
	switch in.Metric {
	case "sales_funnel":
		return e.queryRowsAsMaps(ctx, `
			SELECT d.stage_id, COUNT(*) AS deal_count, COALESCE(SUM(d.amount), 0) AS total_value
			FROM deal d
			WHERE d.workspace_id = ?
			  AND d.deleted_at IS NULL
			  AND (? = '' OR d.created_at >= ?)
			  AND (? = '' OR d.created_at <= ?)
			GROUP BY d.stage_id
			ORDER BY deal_count DESC
		`, workspaceID, from, from, to, to)
	case "deal_aging":
		return e.queryRowsAsMaps(ctx, `
			SELECT d.stage_id, AVG(julianday('now') - julianday(d.created_at)) AS avg_days
			FROM deal d
			WHERE d.workspace_id = ?
			  AND d.deleted_at IS NULL
			  AND d.status = 'open'
			  AND (? = '' OR d.created_at >= ?)
			  AND (? = '' OR d.created_at <= ?)
			GROUP BY d.stage_id
		`, workspaceID, from, from, to, to)
	case "case_volume":
		return e.queryRowsAsMaps(ctx, `
			SELECT c.priority, c.status, COUNT(*) AS total
			FROM case_ticket c
			WHERE c.workspace_id = ?
			  AND c.deleted_at IS NULL
			  AND (? = '' OR c.created_at >= ?)
			  AND (? = '' OR c.created_at <= ?)
			GROUP BY c.priority, c.status
			ORDER BY total DESC
		`, workspaceID, from, from, to, to)
	case "case_backlog":
		return e.queryRowsAsMaps(ctx, `
			SELECT c.status, COUNT(*) AS total
			FROM case_ticket c
			WHERE c.workspace_id = ?
			  AND c.deleted_at IS NULL
			  AND c.status IN ('open', 'in_progress', 'waiting')
			  AND (julianday('now') - julianday(c.created_at)) > 30
			GROUP BY c.status
			ORDER BY total DESC
		`, workspaceID)
	case "mttr":
		return e.queryRowsAsMaps(ctx, `
			SELECT c.priority, AVG(julianday(c.updated_at) - julianday(c.created_at)) AS avg_days_to_resolve
			FROM case_ticket c
			WHERE c.workspace_id = ?
			  AND c.deleted_at IS NULL
			  AND c.status IN ('resolved', 'closed')
			  AND (? = '' OR c.updated_at >= ?)
			  AND (? = '' OR c.updated_at <= ?)
			GROUP BY c.priority
		`, workspaceID, from, from, to, to)
	default:
		return nil, fmt.Errorf("%w: unsupported metric %q", ErrBuiltinExecutionFailed, in.Metric)
	}
}

func (e *QueryMetricsExecutor) queryRowsAsMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: query metrics: %v", ErrBuiltinExecutionFailed, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("%w: read columns: %v", ErrBuiltinExecutionFailed, err)
	}

	out := make([]map[string]any, 0)
	for rows.Next() {
		item, scanErr := scanMetricRow(rows, cols)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	iterErr := rows.Err()
	if iterErr != nil {
		return nil, fmt.Errorf("%w: iterate metrics rows: %v", ErrBuiltinExecutionFailed, iterErr)
	}
	return out, nil
}

func scanMetricRow(rows *sql.Rows, cols []string) (map[string]any, error) {
	vals := make([]any, len(cols))
	scanTargets := make([]any, len(cols))
	for i := range vals {
		scanTargets[i] = &vals[i]
	}
	if err := rows.Scan(scanTargets...); err != nil {
		return nil, fmt.Errorf("%w: scan metrics row: %v", ErrBuiltinExecutionFailed, err)
	}
	item := make(map[string]any, len(cols))
	for i, c := range cols {
		item[c] = normalizeDBValue(vals[i])
	}
	return item, nil
}

func normalizeDBValue(v any) any {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return x
	}
}
