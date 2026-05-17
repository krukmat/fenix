// Tests for MemoryStore — shared key-value store over agent_memory table (Task A.3, ADR-100).
package blackboard_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// setupMemoryDB creates an isolated in-memory SQLite DB with migrations applied
// and a workspace + cognitive_workspace row seeded, ready for MemoryStore tests.
// Returns the DB and the cognitive_workspace ID to use as scope.
func setupMemoryDB(t *testing.T, cwID, wsID string) (*sql.DB, string) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}

	_, err = db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, wsID, "Test WS "+wsID, "ws-"+wsID)
	if err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err = db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES (?, ?, 'active', datetime('now'))`, cwID, wsID)
	if err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, cwID
}

func TestMemoryStore_SetAndGet(t *testing.T) {
	t.Parallel()

	db, cwID := setupMemoryDB(t, "cw-ms-setget", "ws-ms-setget")
	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()

	entry := blackboard.AgentMemory{
		ID:                   "am-setget-0000-0000-000000000001",
		CognitiveWorkspaceID: cwID,
		Key:                  "agent.current_intent",
		Value:                []byte(`{"intent":"resolve_ticket"}`),
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := store.Set(ctx, entry); err != nil {
		t.Fatalf("Set() error = %v; want nil", err)
	}

	got, err := store.Get(ctx, cwID, entry.Key)
	if err != nil {
		t.Fatalf("Get() error = %v; want nil", err)
	}
	if string(got.Value) != string(entry.Value) {
		t.Errorf("Get() value = %q; want %q", got.Value, entry.Value)
	}
	if got.Key != entry.Key {
		t.Errorf("Get() key = %q; want %q", got.Key, entry.Key)
	}
	if got.Scope != entry.Scope {
		t.Errorf("Get() scope = %q; want %q", got.Scope, entry.Scope)
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	t.Parallel()

	db, cwID := setupMemoryDB(t, "cw-ms-notfound", "ws-ms-notfound")
	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, cwID, "nonexistent.key")
	if !errors.Is(err, blackboard.ErrMemoryNotFound) {
		t.Errorf("Get() error = %v; want ErrMemoryNotFound", err)
	}
}

func TestMemoryStore_Get_Expired(t *testing.T) {
	t.Parallel()

	db, cwID := setupMemoryDB(t, "cw-ms-expired", "ws-ms-expired")
	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()

	past := time.Now().UTC().Add(-1 * time.Hour)
	entry := blackboard.AgentMemory{
		ID:                   "am-expired-0000-0000-000000000001",
		CognitiveWorkspaceID: cwID,
		Key:                  "agent.expired_key",
		Value:                []byte(`{"data":"old"}`),
		Scope:                blackboard.MemoryScopeSession,
		ExpiresAt:            &past,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := store.Set(ctx, entry); err != nil {
		t.Fatalf("Set() error = %v; want nil", err)
	}

	_, err := store.Get(ctx, cwID, entry.Key)
	if !errors.Is(err, blackboard.ErrMemoryExpired) {
		t.Errorf("Get() error = %v; want ErrMemoryExpired", err)
	}

	// Lazy TTL: row must be deleted after Get returns ErrMemoryExpired
	var count int
	if dbErr := db.QueryRow(
		"SELECT COUNT(*) FROM agent_memory WHERE cognitive_workspace_id = ? AND key = ?",
		cwID, entry.Key,
	).Scan(&count); dbErr != nil {
		t.Fatalf("count query: %v", dbErr)
	}
	if count != 0 {
		t.Errorf("expired row still in DB after Get; want 0 rows, got %d", count)
	}
}

func TestMemoryStore_Get_NotExpired(t *testing.T) {
	t.Parallel()

	db, cwID := setupMemoryDB(t, "cw-ms-notexp", "ws-ms-notexp")
	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()

	future := time.Now().UTC().Add(1 * time.Hour)
	entry := blackboard.AgentMemory{
		ID:                   "am-notexp-0000-0000-000000000001",
		CognitiveWorkspaceID: cwID,
		Key:                  "agent.future_ttl",
		Value:                []byte(`{"data":"fresh"}`),
		Scope:                blackboard.MemoryScopeSession,
		ExpiresAt:            &future,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := store.Set(ctx, entry); err != nil {
		t.Fatalf("Set() error = %v; want nil", err)
	}

	got, err := store.Get(ctx, cwID, entry.Key)
	if err != nil {
		t.Fatalf("Get() with future TTL error = %v; want nil", err)
	}
	if string(got.Value) != string(entry.Value) {
		t.Errorf("Get() value = %q; want %q", got.Value, entry.Value)
	}
}

func TestMemoryStore_Set_Upsert(t *testing.T) {
	t.Parallel()

	db, cwID := setupMemoryDB(t, "cw-ms-upsert", "ws-ms-upsert")
	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()

	first := blackboard.AgentMemory{
		ID:                   "am-upsert-0000-0000-000000000001",
		CognitiveWorkspaceID: cwID,
		Key:                  "agent.state",
		Value:                []byte(`{"state":"initial"}`),
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := store.Set(ctx, first); err != nil {
		t.Fatalf("first Set() error = %v; want nil", err)
	}

	second := blackboard.AgentMemory{
		ID:                   "am-upsert-0000-0000-000000000002",
		CognitiveWorkspaceID: cwID,
		Key:                  "agent.state", // same key
		Value:                []byte(`{"state":"updated"}`),
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC().Add(1 * time.Second),
	}

	if err := store.Set(ctx, second); err != nil {
		t.Fatalf("second Set() (upsert) error = %v; want nil", err)
	}

	got, err := store.Get(ctx, cwID, "agent.state")
	if err != nil {
		t.Fatalf("Get() after upsert error = %v; want nil", err)
	}
	if string(got.Value) != string(second.Value) {
		t.Errorf("Get() after upsert value = %q; want %q", got.Value, second.Value)
	}

	// Only one row must exist (upsert, not double insert)
	var count int
	if dbErr := db.QueryRow(
		"SELECT COUNT(*) FROM agent_memory WHERE cognitive_workspace_id = ? AND key = ?",
		cwID, "agent.state",
	).Scan(&count); dbErr != nil {
		t.Fatalf("count query: %v", dbErr)
	}
	if count != 1 {
		t.Errorf("row count after upsert = %d; want 1", count)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	t.Parallel()

	db, cwID := setupMemoryDB(t, "cw-ms-delete", "ws-ms-delete")
	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()

	entry := blackboard.AgentMemory{
		ID:                   "am-delete-0000-0000-000000000001",
		CognitiveWorkspaceID: cwID,
		Key:                  "agent.to_delete",
		Value:                []byte(`{}`),
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := store.Set(ctx, entry); err != nil {
		t.Fatalf("Set() error = %v; want nil", err)
	}

	if err := store.Delete(ctx, cwID, entry.Key); err != nil {
		t.Fatalf("Delete() error = %v; want nil", err)
	}

	_, err := store.Get(ctx, cwID, entry.Key)
	if !errors.Is(err, blackboard.ErrMemoryNotFound) {
		t.Errorf("Get() after Delete error = %v; want ErrMemoryNotFound", err)
	}
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	t.Parallel()

	db, cwID := setupMemoryDB(t, "cw-ms-delnf", "ws-ms-delnf")
	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()

	// Delete of non-existent key must be idempotent — no error
	if err := store.Delete(ctx, cwID, "nonexistent.key"); err != nil {
		t.Errorf("Delete() on nonexistent key error = %v; want nil (idempotent)", err)
	}
}

func TestMemoryStore_ClearSession(t *testing.T) {
	t.Parallel()

	db, cwID := setupMemoryDB(t, "cw-ms-clear", "ws-ms-clear")
	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()

	sessionEntry := blackboard.AgentMemory{
		ID:                   "am-clear-sess-0000-000000000001",
		CognitiveWorkspaceID: cwID,
		Key:                  "session.key",
		Value:                []byte(`{"temp":true}`),
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
	persistentEntry := blackboard.AgentMemory{
		ID:                   "am-clear-pers-0000-000000000002",
		CognitiveWorkspaceID: cwID,
		Key:                  "persistent.key",
		Value:                []byte(`{"durable":true}`),
		Scope:                blackboard.MemoryScopePersistent,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := store.Set(ctx, sessionEntry); err != nil {
		t.Fatalf("Set(session) error = %v", err)
	}
	if err := store.Set(ctx, persistentEntry); err != nil {
		t.Fatalf("Set(persistent) error = %v", err)
	}

	if err := store.ClearSession(ctx, cwID); err != nil {
		t.Fatalf("ClearSession() error = %v; want nil", err)
	}

	// session key must be gone
	_, err := store.Get(ctx, cwID, sessionEntry.Key)
	if !errors.Is(err, blackboard.ErrMemoryNotFound) {
		t.Errorf("Get(session key) after ClearSession = %v; want ErrMemoryNotFound", err)
	}

	// persistent key must survive
	got, err := store.Get(ctx, cwID, persistentEntry.Key)
	if err != nil {
		t.Fatalf("Get(persistent key) after ClearSession error = %v; want nil", err)
	}
	if string(got.Value) != string(persistentEntry.Value) {
		t.Errorf("Get(persistent key) value = %q; want %q", got.Value, persistentEntry.Value)
	}
}

func TestMemoryStore_ScopeIsolation(t *testing.T) {
	t.Parallel()

	// Two separate cognitive workspaces, same DB
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}

	for _, id := range []string{"ws-iso-a", "ws-iso-b"} {
		if _, err := db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
			VALUES (?, ?, ?, datetime('now'), datetime('now'))`, id, "WS "+id, id); err != nil {
			t.Fatalf("workspace insert %s: %v", id, err)
		}
	}
	for i, pair := range [][2]string{{"cw-iso-a", "ws-iso-a"}, {"cw-iso-b", "ws-iso-b"}} {
		if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
			VALUES (?, ?, 'active', datetime('now'))`, pair[0], pair[1]); err != nil {
			t.Fatalf("cognitive_workspace insert %d: %v", i, err)
		}
	}

	store := blackboard.NewMemoryStore(db)
	ctx := context.Background()
	sharedKey := "shared.key"

	entryA := blackboard.AgentMemory{
		ID:                   "am-iso-a-0000-0000-000000000001",
		CognitiveWorkspaceID: "cw-iso-a",
		Key:                  sharedKey,
		Value:                []byte(`{"owner":"A"}`),
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
	entryB := blackboard.AgentMemory{
		ID:                   "am-iso-b-0000-0000-000000000002",
		CognitiveWorkspaceID: "cw-iso-b",
		Key:                  sharedKey,
		Value:                []byte(`{"owner":"B"}`),
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	if err := store.Set(ctx, entryA); err != nil {
		t.Fatalf("Set(A) error = %v", err)
	}
	if err := store.Set(ctx, entryB); err != nil {
		t.Fatalf("Set(B) error = %v", err)
	}

	gotA, err := store.Get(ctx, "cw-iso-a", sharedKey)
	if err != nil {
		t.Fatalf("Get(cw-a) error = %v", err)
	}
	if string(gotA.Value) != string(entryA.Value) {
		t.Errorf("Get(cw-a) value = %q; want %q", gotA.Value, entryA.Value)
	}

	gotB, err := store.Get(ctx, "cw-iso-b", sharedKey)
	if err != nil {
		t.Fatalf("Get(cw-b) error = %v", err)
	}
	if string(gotB.Value) != string(entryB.Value) {
		t.Errorf("Get(cw-b) value = %q; want %q", gotB.Value, entryB.Value)
	}
}
