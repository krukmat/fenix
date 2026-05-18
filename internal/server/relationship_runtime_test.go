package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

type runtimeTestLLM struct{}

func (runtimeTestLLM) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: `{"summary":"customer interaction","sentiment":"positive"}`}, nil
}

func (runtimeTestLLM) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{0.1, 0.2, 0.3}}}, nil
}

func (runtimeTestLLM) ModelInfo() llm.ModelMeta { return llm.ModelMeta{ID: "test"} }

func (runtimeTestLLM) HealthCheck(_ context.Context) error { return nil }

func openRuntimeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.NewDB(filepath.Join(dir, "server-runtime.db"))
	if err != nil {
		t.Fatalf("sqlite.NewDB: %v", err)
	}
	// Serialize all writes through a single connection to avoid SQLITE_BUSY
	// when multiple background goroutines (Summarizer, MemoryEmbedder, etc.)
	// compete for the write lock in the same process.
	db.SetMaxOpenConns(1)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("sqlite.MigrateUp: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedWorkspace(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, id, "Workspace "+id, "slug-"+id); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
}

func seedUser(t *testing.T, db *sql.DB, workspaceID, userID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', datetime('now'), datetime('now'))
	`, userID, workspaceID, userID+"@example.com", userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func seedDealDependencies(t *testing.T, db *sql.DB, workspaceID, ownerID string) (accountID, pipelineID, stageID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	accountID = "acc-runtime"
	pipelineID = "pl-runtime"
	stageID = "st-runtime"

	if _, err := db.Exec(`
		INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, accountID, workspaceID, "Runtime Account", ownerID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at)
		VALUES (?, ?, ?, 'deal', ?, ?)
	`, pipelineID, workspaceID, "Runtime Pipeline", now, now); err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at)
		VALUES (?, ?, ?, 1, ?, ?)
	`, stageID, pipelineID, "Discovery", now, now); err != nil {
		t.Fatalf("seed stage: %v", err)
	}
	return accountID, pipelineID, stageID
}

func waitForDB(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := db.QueryRow(query, args...).Scan(&count); err == nil && count > 0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("condition not met for query: %s", query)
}

func newRuntimeServer(db *sql.DB) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{db: db, bgCtx: ctx, cancel: cancel}
}

func TestRelationshipRuntime_ActivityCreatedBuildsDerivedArtifacts(t *testing.T) {
	db := openRuntimeTestDB(t)
	workspaceID := "ws-runtime-activity"
	ownerID := "user-runtime-activity"
	seedWorkspace(t, db, workspaceID)
	seedUser(t, db, workspaceID, ownerID)
	accountID, _, _ := seedDealDependencies(t, db, workspaceID, ownerID)

	s := newRuntimeServer(db)
	defer s.cancel()
	bus := eventbus.New()
	provider := runtimeTestLLM{}
	s.startRelationshipRuntime(bus, provider, provider)

	activitySvc := crm.NewActivityServiceWithBus(db, bus)
	if _, err := activitySvc.Create(context.Background(), crm.CreateActivityInput{
		WorkspaceID:  workspaceID,
		ActivityType: "call",
		EntityType:   "account",
		EntityID:     accountID,
		OwnerID:      ownerID,
		Subject:      "Customer called",
		Body:         "Discussed renewal",
	}); err != nil {
		t.Fatalf("create activity: %v", err)
	}

	waitForDB(t, db, `SELECT COUNT(*) FROM relationship_memory WHERE workspace_id = ? AND entity_type = 'account' AND entity_id = ?`, workspaceID, accountID)
	waitForDB(t, db, `SELECT COUNT(*) FROM interaction_signal s JOIN relationship_memory m ON m.id = s.relationship_memory_id WHERE m.workspace_id = ? AND m.entity_id = ?`, workspaceID, accountID)
	waitForDB(t, db, `SELECT COUNT(*) FROM trust_score ts JOIN relationship_memory m ON m.id = ts.relationship_memory_id WHERE m.workspace_id = ? AND m.entity_id = ?`, workspaceID, accountID)
	waitForDB(t, db, `SELECT COUNT(*) FROM interaction_signal_embedding WHERE workspace_id = ?`, workspaceID)

	var signalType string
	if err := db.QueryRow(`
		SELECT s.signal_type
		FROM interaction_signal s
		JOIN relationship_memory m ON m.id = s.relationship_memory_id
		WHERE m.workspace_id = ? AND m.entity_id = ?
		LIMIT 1
	`, workspaceID, accountID).Scan(&signalType); err != nil {
		t.Fatalf("select signal type: %v", err)
	}
	if signalType != "call" {
		t.Fatalf("signal_type = %q; want %q", signalType, "call")
	}
}

func TestRelationshipRuntime_ApprovalDecisionBuildsStakeholderGraph(t *testing.T) {
	db := openRuntimeTestDB(t)
	workspaceID := "ws-runtime-approval"
	requesterID := "user-requester"
	approverID := "user-approver"
	seedWorkspace(t, db, workspaceID)
	seedUser(t, db, workspaceID, requesterID)
	seedUser(t, db, workspaceID, approverID)
	accountID, pipelineID, stageID := seedDealDependencies(t, db, workspaceID, requesterID)

	deal, err := crm.NewDealService(db).Create(context.Background(), crm.CreateDealInput{
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		PipelineID:  pipelineID,
		StageID:     stageID,
		OwnerID:     requesterID,
		Title:       "Approval Deal",
	})
	if err != nil {
		t.Fatalf("create deal: %v", err)
	}

	s := newRuntimeServer(db)
	defer s.cancel()
	bus := eventbus.New()
	provider := runtimeTestLLM{}
	s.startRelationshipRuntime(bus, provider, provider)

	approvalSvc := policy.NewApprovalServiceWithBus(db, domainaudit.NewAuditService(db), bus)
	resourceType := "deal"
	resourceID := deal.ID
	req, err := approvalSvc.CreateApprovalRequest(context.Background(), policy.CreateApprovalRequestInput{
		WorkspaceID:  workspaceID,
		RequestedBy:  requesterID,
		ApproverID:   approverID,
		Action:       "approve_deal",
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Payload:      mustJSON(t, map[string]any{"deal_id": deal.ID}),
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create approval request: %v", err)
	}
	if err := approvalSvc.DecideApprovalRequest(context.Background(), req.ID, "approve", approverID); err != nil {
		t.Fatalf("decide approval request: %v", err)
	}

	// Wait for the approval.decided edge (strength=0.9), not just the approval.requested one (strength=0.6).
	waitForDB(t, db, `SELECT COUNT(*) FROM stakeholder_graph WHERE workspace_id = ? AND from_entity_id = ? AND to_entity_id = ? AND strength = 0.9`, workspaceID, approverID, deal.ID)

	var (
		strength      float64
		influenceType string
	)
	if err := db.QueryRow(`
		SELECT strength, influence_type
		FROM stakeholder_graph
		WHERE workspace_id = ? AND from_entity_id = ? AND to_entity_id = ?
		LIMIT 1
	`, workspaceID, approverID, deal.ID).Scan(&strength, &influenceType); err != nil {
		t.Fatalf("select stakeholder_graph: %v", err)
	}
	if influenceType != "approves" {
		t.Fatalf("influence_type = %q; want %q", influenceType, "approves")
	}
	if strength != 0.9 {
		t.Fatalf("strength = %v; want %v", strength, 0.9)
	}
}

func TestRelationshipRuntime_ShutdownStopsWorkers(t *testing.T) {
	db := openRuntimeTestDB(t)
	s := newRuntimeServer(db)
	bus := eventbus.New()
	provider := runtimeTestLLM{}
	s.startRelationshipRuntime(bus, provider, provider)

	s.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := s.waitBackground(ctx); err != nil {
		t.Fatalf("waitBackground error = %v", err)
	}
}

func mustJSON(t *testing.T, value map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return raw
}

func TestServer_StartBackgroundWaitsForWorker(t *testing.T) {
	s := newRuntimeServer(nil)
	started := make(chan struct{})
	s.startBackground(func() {
		close(started)
		<-s.bgCtx.Done()
	})
	<-started
	s.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := s.waitBackground(ctx); err != nil {
		t.Fatalf("waitBackground error = %v", err)
	}
}

func TestServer_WaitBackgroundTimeout(t *testing.T) {
	s := newRuntimeServer(nil)
	s.startBackground(func() {
		time.Sleep(200 * time.Millisecond)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := s.waitBackground(ctx); err == nil {
		t.Fatal("waitBackground returned nil error; want context deadline exceeded")
	}
	s.cancel()
}

func TestApprovalRuntimePayloadUsesApproverAsActor(t *testing.T) {
	db := openRuntimeTestDB(t)
	workspaceID := "ws-runtime-payload"
	requesterID := "user-requester-payload"
	approverID := "user-approver-payload"
	seedWorkspace(t, db, workspaceID)
	seedUser(t, db, workspaceID, requesterID)
	seedUser(t, db, workspaceID, approverID)

	bus := eventbus.New()
	ch := bus.Subscribe("approval.requested")
	svc := policy.NewApprovalServiceWithBus(db, domainaudit.NewAuditService(db), bus)
	resourceType := "deal"
	resourceID := "deal-payload"

	if _, err := svc.CreateApprovalRequest(context.Background(), policy.CreateApprovalRequestInput{
		WorkspaceID:  workspaceID,
		RequestedBy:  requesterID,
		ApproverID:   approverID,
		Action:       "approve_deal",
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create approval request: %v", err)
	}

	select {
	case ev := <-ch:
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload type = %T", ev.Payload)
		}
		if payload["actor_id"] != approverID {
			t.Fatalf("actor_id = %v; want %q", payload["actor_id"], approverID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected approval.requested event")
	}
}
