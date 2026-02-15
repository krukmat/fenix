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

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

type noopExecutor struct{}

func (noopExecutor) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
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
