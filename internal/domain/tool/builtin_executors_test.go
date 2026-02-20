package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

func TestCreateTaskExecutor_Execute_CreatesActivity(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	ownerID := createToolUser(t, db, wsID)

	exec := NewCreateTaskExecutor(db)
	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, wsID)

	params := json.RawMessage(`{"owner_id":"` + ownerID + `","title":"Follow up","entity_type":"case","entity_id":"case-1"}`)
	out, err := exec.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var decoded map[string]any
	_ = json.Unmarshal(out, &decoded)
	taskID, _ := decoded["task_id"].(string)
	if taskID == "" {
		t.Fatalf("expected task_id in response, got %s", string(out))
	}

	svc := crm.NewActivityService(db)
	activity, err := svc.Get(context.Background(), wsID, taskID)
	if err != nil {
		t.Fatalf("expected activity to exist, err = %v", err)
	}
	if activity.Subject != "Follow up" {
		t.Fatalf("expected subject Follow up, got %q", activity.Subject)
	}
}

func TestUpdateCaseExecutor_Execute_UpdatesCase(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	ownerID := createToolUser(t, db, wsID)
	caseSvc := crm.NewCaseService(db)

	created, err := caseSvc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Case from tool",
		Status:      "open",
		Priority:    "medium",
	})
	if err != nil {
		t.Fatalf("Create case error = %v", err)
	}

	exec := NewUpdateCaseExecutor(caseSvc)
	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, wsID)
	params := json.RawMessage(`{"case_id":"` + created.ID + `","status":"in_progress","priority":"high","tags":["vip"]}`)

	if _, err := exec.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	updated, err := caseSvc.Get(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("Get case error = %v", err)
	}
	if updated.Status != "in_progress" || updated.Priority != "high" {
		t.Fatalf("expected status/priority updated, got status=%q priority=%q", updated.Status, updated.Priority)
	}
}

func TestSendReplyExecutor_Execute_CreatesNote(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	ownerID := createToolUser(t, db, wsID)
	caseSvc := crm.NewCaseService(db)

	created, err := caseSvc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Need reply",
	})
	if err != nil {
		t.Fatalf("Create case error = %v", err)
	}

	exec := NewSendReplyExecutor(db, caseSvc)
	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, wsID)
	ctx = context.WithValue(ctx, ctxkeys.UserID, ownerID)

	out, err := exec.Execute(ctx, json.RawMessage(`{"case_id":"`+created.ID+`","body":"Reply body","is_internal":true}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var decoded map[string]any
	_ = json.Unmarshal(out, &decoded)
	noteID, _ := decoded["note_id"].(string)
	if noteID == "" {
		t.Fatalf("expected note_id in response, got %s", string(out))
	}

	noteSvc := crm.NewNoteService(db)
	note, err := noteSvc.Get(context.Background(), wsID, noteID)
	if err != nil {
		t.Fatalf("expected note to exist, err = %v", err)
	}
	if note.Content != "Reply body" {
		t.Fatalf("expected note content Reply body, got %q", note.Content)
	}
}

func TestEnsureBuiltInToolDefinitionsForAllWorkspaces_Idempotent(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	r := NewToolRegistry(db)

	if err := r.EnsureBuiltInToolDefinitionsForAllWorkspaces(context.Background()); err != nil {
		t.Fatalf("first EnsureBuiltInToolDefinitionsForAllWorkspaces error = %v", err)
	}
	if err := r.EnsureBuiltInToolDefinitionsForAllWorkspaces(context.Background()); err != nil {
		t.Fatalf("second EnsureBuiltInToolDefinitionsForAllWorkspaces error = %v", err)
	}

	items, err := r.ListToolDefinitions(context.Background(), wsID)
	if err != nil {
		t.Fatalf("ListToolDefinitions error = %v", err)
	}
	if len(items) != 8 {
		t.Fatalf("expected 8 built-in definitions, got %d", len(items))
	}
}

func TestRegisterBuiltInExecutors(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	r := NewToolRegistry(db)

	if err := RegisterBuiltInExecutors(r, BuiltinServices{
		DB:      db,
		Case:    crm.NewCaseService(db),
		Lead:    crm.NewLeadService(db),
		Account: crm.NewAccountService(db),
		Ingest:  knowledge.NewIngestService(db, eventbus.New()),
	}); err != nil {
		t.Fatalf("RegisterBuiltInExecutors error = %v", err)
	}

	if _, err := r.Get(BuiltinCreateTask); err != nil {
		t.Fatalf("expected create_task executor registered, err = %v", err)
	}
	if _, err := r.Get(BuiltinUpdateCase); err != nil {
		t.Fatalf("expected update_case executor registered, err = %v", err)
	}
	if _, err := r.Get(BuiltinSendReply); err != nil {
		t.Fatalf("expected send_reply executor registered, err = %v", err)
	}
	if _, err := r.Get(BuiltinGetLead); err != nil {
		t.Fatalf("expected get_lead executor registered, err = %v", err)
	}
	if _, err := r.Get(BuiltinGetAccount); err != nil {
		t.Fatalf("expected get_account executor registered, err = %v", err)
	}
	if _, err := r.Get(BuiltinCreateKnowledgeItem); err != nil {
		t.Fatalf("expected create_knowledge_item executor registered, err = %v", err)
	}
	if _, err := r.Get(BuiltinUpdateKnowledgeItem); err != nil {
		t.Fatalf("expected update_knowledge_item executor registered, err = %v", err)
	}
	if _, err := r.Get(BuiltinQueryMetrics); err != nil {
		t.Fatalf("expected query_metrics executor registered, err = %v", err)
	}
}

func TestGetLeadExecutor_SuccessAndNotFound(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	ownerID := createToolUser(t, db, wsID)

	lead, err := crm.NewLeadService(db).Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Status:      "new",
	})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}

	exec := NewGetLeadExecutor(crm.NewLeadService(db))
	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, wsID)

	out, err := exec.Execute(ctx, json.RawMessage(`{"lead_id":"`+lead.ID+`"}`))
	if err != nil {
		t.Fatalf("Execute success error = %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}

	_, err = exec.Execute(ctx, json.RawMessage(`{"lead_id":"missing"}`))
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestGetAccountExecutor_SuccessAndNotFound(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	ownerID := createToolUser(t, db, wsID)

	acc, err := crm.NewAccountService(db).Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "ACME",
		OwnerID:     ownerID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	exec := NewGetAccountExecutor(crm.NewAccountService(db))
	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, wsID)

	out, err := exec.Execute(ctx, json.RawMessage(`{"account_id":"`+acc.ID+`"}`))
	if err != nil {
		t.Fatalf("Execute success error = %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}

	_, err = exec.Execute(ctx, json.RawMessage(`{"account_id":"missing"}`))
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestCreateKnowledgeItemExecutor_SuccessAndWorkspaceMismatch(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	exec := NewCreateKnowledgeItemExecutor(knowledge.NewIngestService(db, eventbus.New()))
	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, wsID)

	out, err := exec.Execute(ctx, json.RawMessage(`{"title":"KB1","content":"contenido","source_type":"document","workspace_id":"`+wsID+`"}`))
	if err != nil {
		t.Fatalf("Execute success error = %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}

	_, err = exec.Execute(ctx, json.RawMessage(`{"title":"KB2","content":"contenido","source_type":"document","workspace_id":"otro"}`))
	if err == nil {
		t.Fatal("expected workspace mismatch error")
	}
}

func TestUpdateKnowledgeItemExecutor_SuccessAndNotFound(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	ingest := knowledge.NewIngestService(db, eventbus.New())
	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, wsID)

	item, err := ingest.Ingest(ctx, knowledge.CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  knowledge.SourceTypeDocument,
		Title:       "Old title",
		RawContent:  "old content",
	})
	if err != nil {
		t.Fatalf("ingest item: %v", err)
	}

	exec := NewUpdateKnowledgeItemExecutor(db)
	out, err := exec.Execute(ctx, json.RawMessage(`{"id":"`+item.ID+`","title":"New title","content":"new content"}`))
	if err != nil {
		t.Fatalf("Execute success error = %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}

	_, err = exec.Execute(ctx, json.RawMessage(`{"id":"missing","title":"x"}`))
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestQueryMetricsExecutor_SalesFunnelAndInvalidMetric(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	ownerID := createToolUser(t, db, wsID)
	pipelineID, stageID := createPipelineStageForToolTest(t, db, wsID)
	createAccountForMetrics(t, db, wsID, ownerID)
	createDealForMetrics(t, db, wsID, ownerID, pipelineID, stageID, "open", 100)

	exec := NewQueryMetricsExecutor(db)
	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, wsID)

	out, err := exec.Execute(ctx, json.RawMessage(`{"metric":"sales_funnel","workspace_id":"`+wsID+`"}`))
	if err != nil {
		t.Fatalf("Execute sales_funnel error = %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}

	_, err = exec.Execute(ctx, json.RawMessage(`{"metric":"unknown","workspace_id":"`+wsID+`"}`))
	if err == nil {
		t.Fatal("expected invalid metric error")
	}
}

func createPipelineStageForToolTest(t *testing.T, db *sql.DB, workspaceID string) (string, string) {
	t.Helper()
	pipelineID := "pipeline-tool-" + randID()
	stageID := "stage-tool-" + randID()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at)
		VALUES (?, ?, ?, 'deal', ?, ?)
	`, pipelineID, workspaceID, "Sales", now, now)
	if err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at)
		VALUES (?, ?, ?, 1, ?, ?)
	`, stageID, pipelineID, "Qualified", now, now)
	if err != nil {
		t.Fatalf("create stage: %v", err)
	}
	return pipelineID, stageID
}

func createAccountForMetrics(t *testing.T, db *sql.DB, workspaceID, ownerID string) string {
	t.Helper()
	id := "account-tool-" + randID()
	now := time.Now().UTC().Format(time.RFC3339)
	name := "Metrics Co " + randID()
	_, err := db.Exec(`
		INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, workspaceID, name, ownerID, now, now)
	if err != nil {
		t.Fatalf("create account for metrics: %v", err)
	}
	return id
}

func createDealForMetrics(t *testing.T, db *sql.DB, workspaceID, ownerID, pipelineID, stageID, status string, amount float64) {
	t.Helper()
	accountID := createAccountForMetrics(t, db, workspaceID, ownerID)
	id := "deal-tool-" + randID()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO deal (
			id, workspace_id, account_id, pipeline_id, stage_id, owner_id, title,
			amount, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, workspaceID, accountID, pipelineID, stageID, ownerID, "Deal Metrics", amount, status, now, now)
	if err != nil {
		t.Fatalf("create deal for metrics: %v", err)
	}
}

func createToolUser(t *testing.T, db *sql.DB, workspaceID string) string {
	t.Helper()
	id := "user-tool-" + randID()
	_, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, password_hash, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', datetime('now'), datetime('now'))
	`, id, workspaceID, id+"@example.com", "hash", "Tool User")
	if err != nil {
		t.Fatalf("createToolUser error = %v", err)
	}
	return id
}
