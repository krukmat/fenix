// Traces: FR-070, NFR-031
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

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
