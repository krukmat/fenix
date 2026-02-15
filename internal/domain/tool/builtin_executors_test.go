package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
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
	if len(items) != 3 {
		t.Fatalf("expected 3 built-in definitions, got %d", len(items))
	}
}

func TestRegisterBuiltInExecutors(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	r := NewToolRegistry(db)

	if err := RegisterBuiltInExecutors(r, BuiltinServices{DB: db, Case: crm.NewCaseService(db)}); err != nil {
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
