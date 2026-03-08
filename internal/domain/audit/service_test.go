// Traces: FR-070, NFR-031
package audit

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!")
	code := m.Run()
	os.Exit(code)
}

// setupTestDB creates an in-memory database with migrations for testing
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Important for SQLite in-memory DBs: force a single shared connection.
	// Otherwise, additional pooled connections may see a different empty
	// in-memory database (causing "no such table" in async/goroutine paths).
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func createWorkspaceForTest(t *testing.T, db *sql.DB, workspaceID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, workspaceID, "Test Workspace", fmt.Sprintf("ws-%s", workspaceID), now, now)
	if err != nil {
		t.Fatalf("failed to insert workspace fixture: %v", err)
	}
}

// TestCreateAuditEvent_Success verifies that Log creates an audit event correctly
func TestCreateAuditEvent_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	actorID := uuid.NewV7().String()

	details := &EventDetails{
		NewValue: map[string]interface{}{"name": "Test Account"},
		Metadata: map[string]interface{}{"source": "api"},
	}

	event := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		ActorID:     actorID,
		ActorType:   ActorTypeUser,
		Action:      "create_account",
		EntityType:  strPtr("account"),
		EntityID:    strPtr(uuid.NewV7().String()),
		Details:     mustJSON(details),
		Outcome:     OutcomeSuccess,
		IPAddress:   strPtr("127.0.0.1"),
		UserAgent:   strPtr("test-agent"),
		CreatedAt:   time.Now(),
	}

	err := service.Log(ctx, event)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify event can be retrieved
	retrieved, err := service.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != event.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, event.ID)
	}
	if retrieved.WorkspaceID != workspaceID {
		t.Errorf("WorkspaceID mismatch: got %s, want %s", retrieved.WorkspaceID, workspaceID)
	}
	if retrieved.ActorID != actorID {
		t.Errorf("ActorID mismatch: got %s, want %s", retrieved.ActorID, actorID)
	}
	if retrieved.ActorType != ActorTypeUser {
		t.Errorf("ActorType mismatch: got %s, want %s", retrieved.ActorType, ActorTypeUser)
	}
	if retrieved.Action != "create_account" {
		t.Errorf("Action mismatch: got %s, want create_account", retrieved.Action)
	}
	if retrieved.Outcome != OutcomeSuccess {
		t.Errorf("Outcome mismatch: got %s, want %s", retrieved.Outcome, OutcomeSuccess)
	}
}

func TestAuditEvent_AppendOnly_UpdateAndDeleteBlocked(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)

	event := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		ActorID:     uuid.NewV7().String(),
		ActorType:   ActorTypeUser,
		Action:      "tool.executed",
		Outcome:     OutcomeSuccess,
		CreatedAt:   time.Now(),
	}
	if err := service.Log(ctx, event); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	if _, err := db.ExecContext(ctx, `UPDATE audit_event SET action = 'mutated' WHERE id = ?`, event.ID); err == nil {
		t.Fatal("expected UPDATE on audit_event to fail")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM audit_event WHERE id = ?`, event.ID); err == nil {
		t.Fatal("expected DELETE on audit_event to fail")
	}

	stored, err := service.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if stored.Action != "tool.executed" {
		t.Fatalf("unexpected action after blocked mutation: %s", stored.Action)
	}
}

// TestLogWithDetails_Success tests the helper method
func TestLogWithDetails_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	actorID := uuid.NewV7().String()
	entityID := uuid.NewV7().String()

	details := &EventDetails{
		NewValue: map[string]string{"name": "Test Account", "status": "active"},
		Changes: []Change{
			{Field: "name", OldValue: nil, NewValue: "Test Account"},
		},
	}

	err := service.LogWithDetails(
		ctx,
		workspaceID,
		actorID,
		ActorTypeUser,
		"create_account",
		strPtr("account"),
		&entityID,
		details,
		OutcomeSuccess,
	)
	if err != nil {
		t.Fatalf("LogWithDetails failed: %v", err)
	}

	// Verify by listing events
	events, total, err := service.ListByWorkspace(ctx, workspaceID, 10, 0)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 event, got %d", total)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event in list, got %d", len(events))
	}

	event := events[0]
	if event.Action != "create_account" {
		t.Errorf("Action mismatch: got %s", event.Action)
	}
	if event.Outcome != OutcomeSuccess {
		t.Errorf("Outcome mismatch: got %s", event.Outcome)
	}
	if event.EntityType == nil || *event.EntityType != "account" {
		t.Errorf("EntityType mismatch")
	}
}

// TestListAuditEventsByWorkspace tests pagination
func TestListAuditEventsByWorkspace(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	actorID := uuid.NewV7().String()

	// Create 5 events
	for i := 0; i < 5; i++ {
		event := &AuditEvent{
			ID:          uuid.NewV7().String(),
			WorkspaceID: workspaceID,
			ActorID:     actorID,
			ActorType:   ActorTypeUser,
			Action:      "test_action",
			Outcome:     OutcomeSuccess,
			CreatedAt:   time.Now(),
		}
		if err := service.Log(ctx, event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Test pagination - page 1
	events, total, err := service.ListByWorkspace(ctx, workspaceID, 3, 0)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events on first page, got %d", len(events))
	}

	// Test pagination - page 2
	events, _, err = service.ListByWorkspace(ctx, workspaceID, 3, 3)
	if err != nil {
		t.Fatalf("ListByWorkspace page 2 failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events on second page, got %d", len(events))
	}
}

// TestListAuditEventsByActor tests filtering by actor
func TestListAuditEventsByActor(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	actorID1 := uuid.NewV7().String()
	actorID2 := uuid.NewV7().String()

	// Create 2 events for actor1
	for i := 0; i < 2; i++ {
		event := &AuditEvent{
			ID:          uuid.NewV7().String(),
			WorkspaceID: workspaceID,
			ActorID:     actorID1,
			ActorType:   ActorTypeUser,
			Action:      "test_action",
			Outcome:     OutcomeSuccess,
			CreatedAt:   time.Now(),
		}
		if err := service.Log(ctx, event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Create 1 event for actor2
	event := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		ActorID:     actorID2,
		ActorType:   ActorTypeUser,
		Action:      "test_action",
		Outcome:     OutcomeSuccess,
		CreatedAt:   time.Now(),
	}
	if err := service.Log(ctx, event); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// List by actor1
	events, err := service.ListByActor(ctx, actorID1, 10)
	if err != nil {
		t.Fatalf("ListByActor failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events for actor1, got %d", len(events))
	}

	// Verify all events belong to actor1
	for _, e := range events {
		if e.ActorID != actorID1 {
			t.Errorf("event has wrong actor_id: %s", e.ActorID)
		}
	}
}

// TestListAuditEventsByEntity tests filtering by entity
func TestListAuditEventsByEntity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	actorID := uuid.NewV7().String()
	entityID := uuid.NewV7().String()

	// Create events for entity
	for i := 0; i < 3; i++ {
		event := &AuditEvent{
			ID:          uuid.NewV7().String(),
			WorkspaceID: workspaceID,
			ActorID:     actorID,
			ActorType:   ActorTypeUser,
			Action:      "update_account",
			EntityType:  strPtr("account"),
			EntityID:    strPtr(entityID),
			Outcome:     OutcomeSuccess,
			CreatedAt:   time.Now(),
		}
		if err := service.Log(ctx, event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	// Create event for different entity
	event := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		ActorID:     actorID,
		ActorType:   ActorTypeUser,
		Action:      "update_account",
		EntityType:  strPtr("account"),
		EntityID:    strPtr(uuid.NewV7().String()),
		Outcome:     OutcomeSuccess,
		CreatedAt:   time.Now(),
	}
	if err := service.Log(ctx, event); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// List by entity
	events, err := service.ListByEntity(ctx, "account", entityID, 10)
	if err != nil {
		t.Fatalf("ListByEntity failed: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("expected 3 events for entity, got %d", len(events))
	}

	// Verify all events are for correct entity
	for _, e := range events {
		if e.EntityType == nil || *e.EntityType != "account" {
			t.Errorf("wrong entity_type")
		}
		if e.EntityID == nil || *e.EntityID != entityID {
			t.Errorf("wrong entity_id")
		}
	}
}

// TestAuditTenantIsolation verifies workspace isolation
func TestAuditTenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID1 := uuid.NewV7().String()
	workspaceID2 := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID1)
	createWorkspaceForTest(t, db, workspaceID2)
	actorID := uuid.NewV7().String()

	// Create event in workspace1
	event1 := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID1,
		ActorID:     actorID,
		ActorType:   ActorTypeUser,
		Action:      "test_action",
		Outcome:     OutcomeSuccess,
		CreatedAt:   time.Now(),
	}
	if err := service.Log(ctx, event1); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Create event in workspace2
	event2 := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID2,
		ActorID:     actorID,
		ActorType:   ActorTypeUser,
		Action:      "test_action",
		Outcome:     OutcomeSuccess,
		CreatedAt:   time.Now(),
	}
	if err := service.Log(ctx, event2); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// List workspace1 - should only see event1
	events, total, err := service.ListByWorkspace(ctx, workspaceID1, 10, 0)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}

	if total != 1 {
		t.Errorf("workspace1 should have 1 event, got %d", total)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != event1.ID {
		t.Errorf("wrong event returned for workspace1")
	}

	// List workspace2 - should only see event2
	events, total, err = service.ListByWorkspace(ctx, workspaceID2, 10, 0)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}

	if total != 1 {
		t.Errorf("workspace2 should have 1 event, got %d", total)
	}
	if events[0].ID != event2.ID {
		t.Errorf("wrong event returned for workspace2")
	}
}

// TestDifferentActorTypes tests user, agent, and system actor types
func TestDifferentActorTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	testCases := []struct {
		actorType ActorType
		action    string
	}{
		{ActorTypeUser, "user_action"},
		{ActorTypeAgent, "agent_action"},
		{ActorTypeSystem, "system_cleanup"},
	}

	for _, tc := range testCases {
		event := &AuditEvent{
			ID:          uuid.NewV7().String(),
			WorkspaceID: workspaceID,
			ActorID:     uuid.NewV7().String(),
			ActorType:   tc.actorType,
			Action:      tc.action,
			Outcome:     OutcomeSuccess,
			CreatedAt:   time.Now(),
		}
		if err := service.Log(ctx, event); err != nil {
			t.Fatalf("Log failed for %s: %v", tc.actorType, err)
		}

		// Verify
		retrieved, err := service.GetByID(ctx, event.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if retrieved.ActorType != tc.actorType {
			t.Errorf("ActorType mismatch for %s: got %s", tc.actorType, retrieved.ActorType)
		}
	}
}

// TestDifferentOutcomes tests success, denied, and error outcomes
func TestDifferentOutcomes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	testCases := []struct {
		outcome Outcome
		action  string
	}{
		{OutcomeSuccess, "successful_action"},
		{OutcomeDenied, "denied_action"},
		{OutcomeError, "error_action"},
	}

	for _, tc := range testCases {
		event := &AuditEvent{
			ID:          uuid.NewV7().String(),
			WorkspaceID: workspaceID,
			ActorID:     uuid.NewV7().String(),
			ActorType:   ActorTypeUser,
			Action:      tc.action,
			Outcome:     tc.outcome,
			CreatedAt:   time.Now(),
		}
		if err := service.Log(ctx, event); err != nil {
			t.Fatalf("Log failed for %s: %v", tc.outcome, err)
		}

		// Verify
		retrieved, err := service.GetByID(ctx, event.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if retrieved.Outcome != tc.outcome {
			t.Errorf("Outcome mismatch for %s: got %s", tc.outcome, retrieved.Outcome)
		}
	}
}

// TestGetByID_NotFound tests retrieving non-existent event
func TestGetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	_, err := service.GetByID(ctx, uuid.NewV7().String())
	if err == nil {
		t.Error("expected error for non-existent event, got nil")
	}
}

// TestEventOrdering tests that events are returned in descending time order
func TestEventOrdering(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	actorID := uuid.NewV7().String()

	// Create events with small delay
	ids := make([]string, 3)
	for i := 0; i < 3; i++ {
		event := &AuditEvent{
			ID:          uuid.NewV7().String(),
			WorkspaceID: workspaceID,
			ActorID:     actorID,
			ActorType:   ActorTypeUser,
			Action:      "test_action",
			Outcome:     OutcomeSuccess,
			CreatedAt:   time.Now(),
		}
		if err := service.Log(ctx, event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		ids[i] = event.ID
		time.Sleep(10 * time.Millisecond) // Small delay for ordering
	}

	// List events - should be in descending order (newest first)
	events, _, err := service.ListByWorkspace(ctx, workspaceID, 10, 0)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Verify descending order (newest first)
	for i := 0; i < len(events)-1; i++ {
		if events[i].CreatedAt.Before(events[i+1].CreatedAt) {
			t.Error("events not in descending order")
		}
	}
}

// TestJSONDetails tests that JSON details are stored and retrieved correctly
func TestJSONDetails(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	actorID := uuid.NewV7().String()

	// Create complex details
	details := &EventDetails{
		OldValue: map[string]interface{}{
			"name":   "Old Name",
			"status": "inactive",
			"nested": map[string]interface{}{
				"field1": "value1",
				"field2": 123,
			},
		},
		NewValue: map[string]interface{}{
			"name":   "New Name",
			"status": "active",
			"nested": map[string]interface{}{
				"field1": "updated_value",
				"field2": 456,
			},
		},
		Changes: []Change{
			{Field: "name", OldValue: "Old Name", NewValue: "New Name"},
			{Field: "status", OldValue: "inactive", NewValue: "active"},
		},
		Metadata: map[string]interface{}{
			"source":    "api",
			"ip":        "192.168.1.1",
			"requestId": "req-123",
		},
	}

	event := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		ActorID:     actorID,
		ActorType:   ActorTypeUser,
		Action:      "update_account",
		EntityType:  strPtr("account"),
		EntityID:    strPtr(uuid.NewV7().String()),
		Details:     mustJSON(details),
		Outcome:     OutcomeSuccess,
		CreatedAt:   time.Now(),
	}

	if err := service.Log(ctx, event); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Retrieve and verify JSON integrity
	retrieved, err := service.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	var retrievedDetails EventDetails
	if err := json.Unmarshal(retrieved.Details, &retrievedDetails); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}

	if len(retrievedDetails.Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(retrievedDetails.Changes))
	}
}

// TestNilEntityFields tests events without entity (e.g., auth actions)
func TestNilEntityFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	actorID := uuid.NewV7().String()

	// Create auth event without entity
	event := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		ActorID:     actorID,
		ActorType:   ActorTypeUser,
		Action:      "login",
		EntityType:  nil, // No entity for login
		EntityID:    nil,
		Details:     nil,
		Outcome:     OutcomeSuccess,
		IPAddress:   strPtr("127.0.0.1"),
		CreatedAt:   time.Now(),
	}

	if err := service.Log(ctx, event); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify
	retrieved, err := service.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.EntityType != nil {
		t.Error("expected nil EntityType")
	}
	if retrieved.EntityID != nil {
		t.Error("expected nil EntityID")
	}
}

func TestListByOutcome_And_ListByAction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewAuditService(db)
	ctx := context.Background()

	workspaceID := uuid.NewV7().String()
	otherWorkspaceID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, workspaceID)
	createWorkspaceForTest(t, db, otherWorkspaceID)
	actorID := uuid.NewV7().String()

	fixtures := []struct {
		ws      string
		action  string
		outcome Outcome
	}{
		{workspaceID, "create_account", OutcomeSuccess},
		{workspaceID, "create_account", OutcomeDenied},
		{workspaceID, "delete_account", OutcomeDenied},
		{workspaceID, "delete_account", OutcomeError},
		{otherWorkspaceID, "create_account", OutcomeDenied},
	}

	for _, fx := range fixtures {
		event := &AuditEvent{
			ID:          uuid.NewV7().String(),
			WorkspaceID: fx.ws,
			ActorID:     actorID,
			ActorType:   ActorTypeUser,
			Action:      fx.action,
			Outcome:     fx.outcome,
			CreatedAt:   time.Now(),
		}
		if err := service.Log(ctx, event); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	byOutcome, err := service.ListByOutcome(ctx, workspaceID, OutcomeDenied, 10, 0)
	if err != nil {
		t.Fatalf("ListByOutcome failed: %v", err)
	}
	if len(byOutcome) != 2 {
		t.Fatalf("expected 2 denied events in workspace, got %d", len(byOutcome))
	}
	for _, e := range byOutcome {
		if e.WorkspaceID != workspaceID {
			t.Fatalf("unexpected workspace in outcome filter: %s", e.WorkspaceID)
		}
		if e.Outcome != OutcomeDenied {
			t.Fatalf("unexpected outcome in result: %s", e.Outcome)
		}
	}

	byAction, err := service.ListByAction(ctx, workspaceID, "delete_account", 10, 0)
	if err != nil {
		t.Fatalf("ListByAction failed: %v", err)
	}
	if len(byAction) != 2 {
		t.Fatalf("expected 2 delete_account events in workspace, got %d", len(byAction))
	}
	for _, e := range byAction {
		if e.WorkspaceID != workspaceID {
			t.Fatalf("unexpected workspace in action filter: %s", e.WorkspaceID)
		}
		if e.Action != "delete_account" {
			t.Fatalf("unexpected action in result: %s", e.Action)
		}
	}
}

func TestQuery_NoFilters_ReturnAllWorkspaceEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewAuditService(db)
	ctx := context.Background()

	wsID := uuid.NewV7().String()
	otherWS := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)
	createWorkspaceForTest(t, db, otherWS)

	for i := 0; i < 3; i++ {
		mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "a", OutcomeSuccess, time.Now().Add(time.Duration(i)*time.Second))
	}
	mustLogEvent(t, svc, otherWS, uuid.NewV7().String(), "a", OutcomeSuccess, time.Now())

	items, err := svc.Query(ctx, QueryInput{WorkspaceID: wsID, Limit: 50})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestQuery_FilterByAction_ReturnsOnlyMatching(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewAuditService(db)
	ctx := context.Background()
	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)

	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "tool.executed", OutcomeSuccess, time.Now())
	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "approval.decided", OutcomeSuccess, time.Now())

	items, err := svc.Query(ctx, QueryInput{WorkspaceID: wsID, Action: "tool.executed", Limit: 20})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 1 || items[0].Action != "tool.executed" {
		t.Fatalf("unexpected query result: %+v", items)
	}
}

func TestQuery_FilterByOutcome_ReturnsOnlyMatching(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewAuditService(db)
	ctx := context.Background()
	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)

	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "x", OutcomeDenied, time.Now())
	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "x", OutcomeSuccess, time.Now())

	items, err := svc.Query(ctx, QueryInput{WorkspaceID: wsID, Outcome: string(OutcomeDenied), Limit: 20})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 1 || items[0].Outcome != OutcomeDenied {
		t.Fatalf("unexpected outcome filter result: %+v", items)
	}
}

func TestQuery_FilterByDateRange_ReturnsInRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewAuditService(db)
	ctx := context.Background()
	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)

	start := time.Now().UTC().Add(-2 * time.Hour)
	inRange := time.Now().UTC()
	end := time.Now().UTC().Add(2 * time.Hour)

	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "x", OutcomeSuccess, start.Add(-time.Hour))
	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "x", OutcomeSuccess, inRange)

	items, err := svc.Query(ctx, QueryInput{
		WorkspaceID: wsID,
		DateFrom:    start.Format(time.RFC3339),
		DateTo:      end.Format(time.RFC3339),
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 in-range event, got %d", len(items))
	}
}

func TestQuery_FilterByActorID_ReturnsOnlyMatching(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewAuditService(db)
	ctx := context.Background()
	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)
	actorID := uuid.NewV7().String()

	mustLogEvent(t, svc, wsID, actorID, "x", OutcomeSuccess, time.Now())
	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "x", OutcomeSuccess, time.Now())

	items, err := svc.Query(ctx, QueryInput{WorkspaceID: wsID, ActorID: actorID, Limit: 20})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 1 || items[0].ActorID != actorID {
		t.Fatalf("unexpected actor filter result: %+v", items)
	}
}

func TestQuery_MultipleFilters_CombinesCorrectly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewAuditService(db)
	ctx := context.Background()
	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)
	actorID := uuid.NewV7().String()

	mustLogEvent(t, svc, wsID, actorID, "tool.executed", OutcomeSuccess, time.Now())
	mustLogEvent(t, svc, wsID, actorID, "tool.executed", OutcomeDenied, time.Now())
	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "tool.executed", OutcomeSuccess, time.Now())

	items, err := svc.Query(ctx, QueryInput{
		WorkspaceID: wsID,
		ActorID:     actorID,
		Action:      "tool.executed",
		Outcome:     string(OutcomeSuccess),
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 combined result, got %d", len(items))
	}
}

func TestExportCSV_Returns1000Rows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewAuditService(db)
	ctx := context.Background()
	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)

	for i := 0; i < 1000; i++ {
		mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "bulk", OutcomeSuccess, time.Now().Add(time.Duration(i)*time.Millisecond))
	}

	r, err := svc.Export(ctx, ExportInput{WorkspaceID: wsID})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read export failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 1001 { // header + 1000 rows
		t.Fatalf("expected 1001 csv lines, got %d", len(lines))
	}
}

func TestExportCSV_ContainsHeaderRow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewAuditService(db)
	ctx := context.Background()
	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)
	mustLogEvent(t, svc, wsID, uuid.NewV7().String(), "x", OutcomeSuccess, time.Now())

	r, err := svc.Export(ctx, ExportInput{WorkspaceID: wsID})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	br := bufio.NewReader(r)
	header, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read header failed: %v", err)
	}
	if !strings.Contains(header, "id,workspace_id,actor_id,actor_type,action") {
		t.Fatalf("unexpected csv header: %s", header)
	}
}

func mustLogEvent(t *testing.T, svc *AuditService, wsID, actorID, action string, outcome Outcome, createdAt time.Time) {
	t.Helper()
	e := &AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: wsID,
		ActorID:     actorID,
		ActorType:   ActorTypeUser,
		Action:      action,
		Outcome:     outcome,
		CreatedAt:   createdAt,
	}
	if err := svc.Log(context.Background(), e); err != nil {
		t.Fatalf("log event failed: %v", err)
	}
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

func mustJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(b)
}

func TestRegisterEventSubscribers_ConsumesEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewAuditService(db)
	ctx := context.Background()

	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)
	actorID := uuid.NewV7().String()
	entityID := uuid.NewV7().String()

	bus := eventbus.New()
	svc.RegisterEventSubscribers(bus)

	// Missing workspace/actor: must be ignored by consumeEvents.
	bus.Publish("agent.run.started", map[string]any{"workspace_id": "", "actor_id": ""})

	bus.Publish("agent.run.failed", map[string]any{
		"workspace_id": wsID,
		"actor_id":     actorID,
		"entity_type":  "deal",
		"entity_id":    entityID,
	})
	bus.Publish("tool.executed", map[string]any{
		"workspace_id": wsID,
		"actor_id":     actorID,
	})

	var events []*AuditEvent
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		items, _, err := svc.ListByWorkspace(ctx, wsID, 20, 0)
		if err != nil {
			t.Fatalf("ListByWorkspace failed: %v", err)
		}
		events = items
		if len(events) >= 2 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 audit events from bus, got %d", len(events))
	}

	seenFailed := false
	seenTool := false
	for _, ev := range events {
		switch ev.Action {
		case "agent.run.failed":
			seenFailed = true
			if ev.ActorType != ActorTypeAgent {
				t.Fatalf("agent.run.failed ActorType = %s; want %s", ev.ActorType, ActorTypeAgent)
			}
			if ev.Outcome != OutcomeDenied {
				t.Fatalf("agent.run.failed Outcome = %s; want %s", ev.Outcome, OutcomeDenied)
			}
			if ev.EntityType == nil || *ev.EntityType != "deal" {
				t.Fatalf("agent.run.failed EntityType mismatch: %+v", ev.EntityType)
			}
			if ev.EntityID == nil || *ev.EntityID != entityID {
				t.Fatalf("agent.run.failed EntityID mismatch: %+v", ev.EntityID)
			}
		case "tool.executed":
			seenTool = true
			if ev.ActorType != ActorTypeSystem {
				t.Fatalf("tool.executed ActorType = %s; want %s", ev.ActorType, ActorTypeSystem)
			}
			if ev.Outcome != OutcomeSuccess {
				t.Fatalf("tool.executed Outcome = %s; want %s", ev.Outcome, OutcomeSuccess)
			}
		}
	}

	if !seenFailed {
		t.Fatal("expected to find agent.run.failed audit event")
	}
	if !seenTool {
		t.Fatal("expected to find tool.executed audit event")
	}
}

type typedAuditPayload struct {
	WorkspaceID string
	ActorID     string
	EntityType  string
	EntityID    string
	Status      string
}

func TestRegisterEventSubscribers_ConsumesTypedPayloadsAndNormalizesActions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewAuditService(db)
	ctx := context.Background()

	wsID := uuid.NewV7().String()
	createWorkspaceForTest(t, db, wsID)
	actorID := uuid.NewV7().String()

	bus := eventbus.New()
	svc.RegisterEventSubscribers(bus)

	bus.Publish("approval.decided", typedAuditPayload{
		WorkspaceID: wsID,
		ActorID:     actorID,
		EntityType:  "approval_request",
		EntityID:    uuid.NewV7().String(),
		Status:      "approved",
	})
	bus.Publish("tool.executed", typedAuditPayload{
		WorkspaceID: wsID,
		ActorID:     actorID,
		EntityType:  "tool",
		EntityID:    "create_task",
	})

	var events []*AuditEvent
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		items, _, err := svc.ListByWorkspace(ctx, wsID, 20, 0)
		if err != nil {
			t.Fatalf("ListByWorkspace failed: %v", err)
		}
		events = items
		if len(events) >= 2 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	if len(events) < 2 {
		t.Fatalf("expected typed payload audit events, got %d", len(events))
	}

	foundApproved := false
	for _, ev := range events {
		if ev.Action != "approval.approved" {
			continue
		}
		foundApproved = true
		if ev.ActorType != ActorTypeSystem {
			t.Fatalf("approval.approved ActorType = %s; want %s", ev.ActorType, ActorTypeSystem)
		}
		if ev.Outcome != OutcomeSuccess {
			t.Fatalf("approval.approved Outcome = %s; want %s", ev.Outcome, OutcomeSuccess)
		}
	}

	if !foundApproved {
		t.Fatal("expected normalized approval.approved audit event")
	}
}

func TestServiceHelpers_UtilityFunctions(t *testing.T) {
	if got := resolveQueryLimit(0); got != 25 {
		t.Fatalf("resolveQueryLimit(0) = %d; want 25", got)
	}
	if got := resolveQueryLimit(10); got != 10 {
		t.Fatalf("resolveQueryLimit(10) = %d; want 10", got)
	}

	if got := normalizeDateArg(""); got != "" {
		t.Fatalf("normalizeDateArg(empty) = %#v; want empty string", got)
	}

	in := "2026-02-23T00:35:15Z"
	got := normalizeDateArg(in)
	if got != "2026-02-23 00:35:15" {
		t.Fatalf("normalizeDateArg(valid) = %#v; want %q", got, "2026-02-23 00:35:15")
	}

	bad := "not-a-date"
	if got := normalizeDateArg(bad); got != bad {
		t.Fatalf("normalizeDateArg(invalid) = %#v; want original", got)
	}

	if got := derefString(nil); got != "" {
		t.Fatalf("derefString(nil) = %q; want empty", got)
	}
	v := "x"
	if got := derefString(&v); got != "x" {
		t.Fatalf("derefString(ptr) = %q; want x", got)
	}

	if resolveActorType("agent.run.started") != ActorTypeAgent {
		t.Fatal("resolveActorType(agent.*) must return ActorTypeAgent")
	}
	if resolveActorType("tool.executed") != ActorTypeSystem {
		t.Fatal("resolveActorType(non-agent) must return ActorTypeSystem")
	}

	if resolveOutcome("agent.run.failed") != OutcomeDenied {
		t.Fatal("resolveOutcome(agent.run.failed) must return denied")
	}
	if resolveOutcome("tool.denied") != OutcomeDenied {
		t.Fatal("resolveOutcome(tool.denied) must return denied")
	}
	if resolveOutcome("tool.executed") != OutcomeSuccess {
		t.Fatal("resolveOutcome(default) must return success")
	}

	m := map[string]any{"entity_type": "account", "entity_id": "a-1"}
	if et := optionalString(m, "entity_type"); et == nil || *et != "account" {
		t.Fatalf("optionalString(entity_type) mismatch: %+v", et)
	}
	m["empty"] = ""
	if got := optionalString(m, "empty"); got != nil {
		t.Fatalf("optionalString(empty) = %+v; want nil", got)
	}
	if got := optionalString(m, "missing"); got != nil {
		t.Fatalf("optionalString(missing) = %+v; want nil", got)
	}

	ws, actor, et, eid := extractEventContext("not-a-map")
	if ws != "" || actor != "" || et != nil || eid != nil {
		t.Fatalf("extractEventContext(non-map) unexpected values: ws=%q actor=%q et=%v eid=%v", ws, actor, et, eid)
	}

	payload := map[string]any{
		"workspace_id": "ws-1",
		"actor_id":     "u-1",
		"entity_type":  "case",
		"entity_id":    "c-1",
	}
	ws, actor, et, eid = extractEventContext(payload)
	if ws != "ws-1" || actor != "u-1" {
		t.Fatalf("extractEventContext ids mismatch: ws=%q actor=%q", ws, actor)
	}
	if et == nil || *et != "case" {
		t.Fatalf("extractEventContext entity_type mismatch: %v", et)
	}
	if eid == nil || *eid != "c-1" {
		t.Fatalf("extractEventContext entity_id mismatch: %v", eid)
	}

	typed := typedAuditPayload{
		WorkspaceID: "ws-typed",
		ActorID:     "u-typed",
		EntityType:  "approval_request",
		EntityID:    "apr-1",
		Status:      "denied",
	}
	ws, actor, et, eid = extractEventContext(typed)
	if ws != "ws-typed" || actor != "u-typed" {
		t.Fatalf("extractEventContext(typed) ids mismatch: ws=%q actor=%q", ws, actor)
	}
	if et == nil || *et != "approval_request" {
		t.Fatalf("extractEventContext(typed) entity_type mismatch: %v", et)
	}
	if eid == nil || *eid != "apr-1" {
		t.Fatalf("extractEventContext(typed) entity_id mismatch: %v", eid)
	}
	if got := resolveAuditAction("approval.decided", typed); got != "approval.denied" {
		t.Fatalf("resolveAuditAction(typed approval) = %q; want approval.denied", got)
	}
	if got := resolveAuditAction("tool.executed", typed); got != "tool.executed" {
		t.Fatalf("resolveAuditAction(non-approval) = %q; want tool.executed", got)
	}
}
