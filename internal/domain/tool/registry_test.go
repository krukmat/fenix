// Traces: FR-202
package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

type noopExecutor struct{}

func (noopExecutor) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
}

type toolPermStub struct {
	allow bool
	err   error
}

func (s toolPermStub) CheckToolPermission(_ context.Context, _, _ string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.allow, nil
}

type toolAuditStub struct {
	actions  []string
	outcomes []audit.Outcome
	details  []map[string]any
}

func (s *toolAuditStub) LogWithDetails(
	_ context.Context,
	_, _ string,
	_ audit.ActorType,
	action string,
	_, _ *string,
	details *audit.EventDetails,
	outcome audit.Outcome,
) error {
	s.actions = append(s.actions, action)
	s.outcomes = append(s.outcomes, outcome)
	if meta, ok := details.Metadata.(map[string]any); ok {
		s.details = append(s.details, meta)
	}
	return nil
}

func openToolTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("sqlite.NewDB failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("sqlite.MigrateUp failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestToolRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	r := NewToolRegistry(db)

	if err := r.Register("update_case", noopExecutor{}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if _, err := r.Get("update_case"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

func TestToolRegistry_ValidateParams_InvalidJSON_ReturnsError(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	r := NewToolRegistry(db)

	_, err := r.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID: wsID,
		Name:        "create_task",
		InputSchema: json.RawMessage(`{"type":"object","required":["title"],"properties":{"title":{"type":"string"}},"additionalProperties":false}`),
	})
	if err != nil {
		t.Fatalf("CreateToolDefinition returned error: %v", err)
	}

	err = r.ValidateParams(context.Background(), wsID, "create_task", json.RawMessage(`{"owner_id":"u1"`))
	if err == nil {
		t.Fatalf("expected validation error for invalid JSON")
	}
	if !errors.Is(err, ErrToolValidationFailed) {
		t.Fatalf("expected ErrToolValidationFailed, got: %v", err)
	}
}

func TestToolRegistry_CreateToolDefinition_RejectsWeakSchema(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	r := NewToolRegistry(db)

	_, err := r.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID: wsID,
		Name:        "create_task",
		InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
	})
	if !errors.Is(err, ErrToolDefinitionInvalid) {
		t.Fatalf("expected ErrToolDefinitionInvalid, got %v", err)
	}
}

func TestToolRegistry_ListToolDefinitions_DeserializesSchemaAndPerms(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	r := NewToolRegistry(db)

	schema := json.RawMessage(`{"type":"object","required":["case_id"],"properties":{"case_id":{"type":"string"}},"additionalProperties":false}`)
	_, err := r.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID:         wsID,
		Name:                "update_case",
		InputSchema:         schema,
		RequiredPermissions: []string{"tools:update_case"},
	})
	if err != nil {
		t.Fatalf("CreateToolDefinition returned error: %v", err)
	}

	items, err := r.ListToolDefinitions(context.Background(), wsID)
	if err != nil {
		t.Fatalf("ListToolDefinitions returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(items))
	}
	if string(items[0].InputSchema) == "" {
		t.Fatalf("expected input schema to be loaded from DB")
	}
	if len(items[0].RequiredPermissions) != 1 || items[0].RequiredPermissions[0] != "tools:update_case" {
		t.Fatalf("unexpected required permissions: %#v", items[0].RequiredPermissions)
	}
}

func TestToolRegistry_UpdateActivateDeactivateDeleteLifecycle(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	r := NewToolRegistry(db)

	created, err := r.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID:         wsID,
		Name:                "update_case",
		InputSchema:         json.RawMessage(`{"type":"object","required":["case_id"],"properties":{"case_id":{"type":"string"}},"additionalProperties":false}`),
		RequiredPermissions: []string{"tools:update_case"},
	})
	if err != nil {
		t.Fatalf("CreateToolDefinition returned error: %v", err)
	}

	updated, err := r.UpdateToolDefinition(context.Background(), UpdateToolDefinitionInput{
		ID:                  created.ID,
		WorkspaceID:         wsID,
		Name:                "update_case_v2",
		Description:         ptrString("updated"),
		InputSchema:         json.RawMessage(`{"type":"object","required":["case_id","status"],"properties":{"case_id":{"type":"string"},"status":{"type":"string"}},"additionalProperties":false}`),
		RequiredPermissions: []string{"tools:update_case", "tools:update_case_v2"},
	})
	if err != nil {
		t.Fatalf("UpdateToolDefinition returned error: %v", err)
	}
	if updated.Name != "update_case_v2" {
		t.Fatalf("updated.Name=%s want=update_case_v2", updated.Name)
	}

	inactive, err := r.SetToolDefinitionActive(context.Background(), wsID, created.ID, false)
	if err != nil {
		t.Fatalf("SetToolDefinitionActive(false) returned error: %v", err)
	}
	if inactive.IsActive {
		t.Fatal("expected tool to be inactive")
	}

	active, err := r.SetToolDefinitionActive(context.Background(), wsID, created.ID, true)
	if err != nil {
		t.Fatalf("SetToolDefinitionActive(true) returned error: %v", err)
	}
	if !active.IsActive {
		t.Fatal("expected tool to be active")
	}

	if err := r.DeleteToolDefinition(context.Background(), wsID, created.ID); err != nil {
		t.Fatalf("DeleteToolDefinition returned error: %v", err)
	}
	if _, err := r.GetToolDefinitionByID(context.Background(), wsID, created.ID); !errors.Is(err, ErrToolDefinitionNotFound) {
		t.Fatalf("expected ErrToolDefinitionNotFound after delete, got %v", err)
	}
}

func TestToolRegistry_Execute_EnforcesActiveValidationAndPermissions(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	r := NewToolRegistryWithAuthorizer(db, toolPermStub{allow: true})
	if err := r.Register("create_task", noopExecutor{}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	created, err := r.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID:         wsID,
		Name:                "create_task",
		InputSchema:         json.RawMessage(`{"type":"object","required":["title"],"properties":{"title":{"type":"string"}},"additionalProperties":false}`),
		RequiredPermissions: []string{"tools:create_task"},
	})
	if err != nil {
		t.Fatalf("CreateToolDefinition returned error: %v", err)
	}

	ctx := context.WithValue(context.Background(), ctxkeys.UserID, "user-1")

	if _, err := r.Execute(ctx, wsID, "create_task", json.RawMessage(`{"title":"x"}`)); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if _, err := r.Execute(ctx, wsID, "create_task", json.RawMessage(`{"unexpected":true}`)); !errors.Is(err, ErrToolValidationFailed) {
		t.Fatalf("expected ErrToolValidationFailed, got %v", err)
	}

	if _, err := r.SetToolDefinitionActive(context.Background(), wsID, created.ID, false); err != nil {
		t.Fatalf("SetToolDefinitionActive returned error: %v", err)
	}
	if _, err := r.Execute(ctx, wsID, "create_task", json.RawMessage(`{"title":"x"}`)); !errors.Is(err, ErrToolInactive) {
		t.Fatalf("expected ErrToolInactive, got %v", err)
	}

	denied := NewToolRegistryWithAuthorizer(db, toolPermStub{allow: false})
	if err := denied.Register("create_task", noopExecutor{}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if _, err := denied.SetToolDefinitionActive(context.Background(), wsID, created.ID, true); err != nil {
		t.Fatalf("SetToolDefinitionActive returned error: %v", err)
	}
	if _, err := denied.Execute(ctx, wsID, "create_task", json.RawMessage(`{"title":"x"}`)); !errors.Is(err, ErrToolPermissionDenied) {
		t.Fatalf("expected ErrToolPermissionDenied, got %v", err)
	}
}

func TestToolRegistry_Execute_BuiltinAuditAndErrorContract(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	auditStub := &toolAuditStub{}
	r := NewToolRegistryWithRuntime(db, toolPermStub{allow: true}, auditStub)
	if err := r.Register(BuiltinCreateTask, noopExecutor{}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	_, err := r.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID:         wsID,
		Name:                BuiltinCreateTask,
		InputSchema:         json.RawMessage(`{"type":"object","required":["title"],"properties":{"title":{"type":"string"}},"additionalProperties":false}`),
		RequiredPermissions: []string{"tools:create_task"},
	})
	if err != nil {
		t.Fatalf("CreateToolDefinition returned error: %v", err)
	}

	ctx := context.WithValue(context.Background(), ctxkeys.UserID, "user-1")
	if _, err := r.Execute(ctx, wsID, BuiltinCreateTask, json.RawMessage(`{"title":"x"}`)); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(auditStub.actions) != 1 || auditStub.actions[0] != "tool.executed" {
		t.Fatalf("unexpected audit actions: %#v", auditStub.actions)
	}
	if auditStub.outcomes[0] != audit.OutcomeSuccess {
		t.Fatalf("unexpected audit outcome: %v", auditStub.outcomes[0])
	}

	deniedAudit := &toolAuditStub{}
	denied := NewToolRegistryWithRuntime(db, toolPermStub{allow: false}, deniedAudit)
	if err := denied.Register(BuiltinCreateTask, noopExecutor{}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	err = executeExpectError(t, denied, ctx, wsID, BuiltinCreateTask, json.RawMessage(`{"title":"x"}`))
	if !IsToolExecutionErrorCode(err, ToolErrorPermissionDenied) {
		t.Fatalf("expected ToolErrorPermissionDenied, got %v", err)
	}
	if len(deniedAudit.actions) != 1 || deniedAudit.actions[0] != "tool.denied" {
		t.Fatalf("unexpected denied audit actions: %#v", deniedAudit.actions)
	}
}

func TestValidateAgainstMinimalSchema(t *testing.T) {
	t.Parallel()

	t.Run("missing required field", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"name": "alice"}
		schema := map[string]any{
			"required": []any{"name", "email"},
		}

		err := validateAgainstMinimalSchema(input, schema)
		if !errors.Is(err, ErrToolValidationFailed) {
			t.Fatalf("expected ErrToolValidationFailed, got %v", err)
		}
	})

	t.Run("unknown field rejected when additional properties false", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"name": "alice", "unexpected": true}
		schema := map[string]any{
			"additionalProperties": false,
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		}

		err := validateAgainstMinimalSchema(input, schema)
		if !errors.Is(err, ErrToolValidationFailed) {
			t.Fatalf("expected ErrToolValidationFailed, got %v", err)
		}
	})

	t.Run("unknown field allowed when additional properties true", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"name": "alice", "unexpected": true}
		schema := map[string]any{
			"additionalProperties": true,
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		}

		if err := validateAgainstMinimalSchema(input, schema); err != nil {
			t.Fatalf("validateAgainstMinimalSchema returned error: %v", err)
		}
	})

	t.Run("default additional properties true", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"unknown": true}
		schema := map[string]any{}

		if err := validateAgainstMinimalSchema(input, schema); err != nil {
			t.Fatalf("validateAgainstMinimalSchema returned error: %v", err)
		}
	})
}

func TestExtractStringSlice(t *testing.T) {
	t.Parallel()

	out := extractStringSlice([]any{"name", "", "  ", 123, "email"})
	if len(out) != 2 || out[0] != "name" || out[1] != "email" {
		t.Fatalf("unexpected slice: %#v", out)
	}

	out = extractStringSlice("not-array")
	if out != nil {
		t.Fatalf("expected nil for non-array input, got %#v", out)
	}
}

func TestValidateUpdateInput_Errors(t *testing.T) {
	t.Parallel()

	validSchema := json.RawMessage(`{"type":"object","required":["x"],"properties":{"x":{"type":"string"}},"additionalProperties":false}`)

	t.Run("empty ID returns error", func(t *testing.T) {
		t.Parallel()
		err := validateUpdateInput(UpdateToolDefinitionInput{ID: "", Name: "n", InputSchema: validSchema})
		if err == nil {
			t.Fatal("expected error for empty ID")
		}
	})

	t.Run("empty Name returns error", func(t *testing.T) {
		t.Parallel()
		err := validateUpdateInput(UpdateToolDefinitionInput{ID: "x", Name: "  ", InputSchema: validSchema})
		if err == nil {
			t.Fatal("expected error for empty Name")
		}
	})
}

func TestIsUniqueConstraintError(t *testing.T) {
	t.Parallel()

	if isUniqueConstraintError(nil) {
		t.Fatal("expected false for nil error")
	}
	if isUniqueConstraintError(sql.ErrNoRows) {
		t.Fatal("expected false for sql.ErrNoRows")
	}
	if !isUniqueConstraintError(fmt.Errorf("UNIQUE constraint failed: foo.bar")) {
		t.Fatal("expected true for UNIQUE constraint error")
	}
	if isUniqueConstraintError(fmt.Errorf("some other error")) {
		t.Fatal("expected false for unrelated error")
	}
}

func TestToolRegistry_Execute_MissingUserContext(t *testing.T) {
	t.Parallel()

	db := openToolTestDB(t)
	wsID := createWorkspace(t, db)
	r := NewToolRegistryWithAuthorizer(db, toolPermStub{allow: true})
	if err := r.Register("create_task", noopExecutor{}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	_, err := r.CreateToolDefinition(context.Background(), CreateToolDefinitionInput{
		WorkspaceID: wsID,
		Name:        "create_task",
		InputSchema: json.RawMessage(`{"type":"object","required":["title"],"properties":{"title":{"type":"string"}},"additionalProperties":false}`),
	})
	if err != nil {
		t.Fatalf("CreateToolDefinition returned error: %v", err)
	}

	// No user ID in context — enforceToolPermission should return ErrToolUserContextMissing.
	_, err = r.Execute(context.Background(), wsID, "create_task", json.RawMessage(`{"title":"x"}`))
	if !errors.Is(err, ErrToolUserContextMissing) {
		t.Fatalf("expected ErrToolUserContextMissing, got %v", err)
	}
}

func ptrString(v string) *string {
	return &v
}

func executeExpectError(t *testing.T, r *ToolRegistry, ctx context.Context, wsID, toolName string, params json.RawMessage) error {
	t.Helper()
	_, err := r.Execute(ctx, wsID, toolName, params)
	if err == nil {
		t.Fatal("expected execution error")
	}
	return err
}

var toolRandCounter int64

func createWorkspace(t *testing.T, db *sql.DB) string {
	t.Helper()
	id := "ws-tool-" + randID()
	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, id, "Tool WS", "tool-"+randID())
	if err != nil {
		t.Fatalf("createWorkspace error = %v", err)
	}
	return id
}

func randID() string {
	n := atomic.AddInt64(&toolRandCounter, 1)
	return time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", n)
}

func TestExecutionError_Error_Format(t *testing.T) {
	underlying := errors.New("underlying")
	err := &ExecutionError{
		ToolName: "my_tool",
		Code:     ToolErrorInternal,
		Err:      underlying,
	}
	msg := err.Error()
	if msg == "" {
		t.Fatal("Error() should not be empty")
	}
	if err.Unwrap() != underlying {
		t.Fatal("Unwrap() should return underlying error")
	}
}
