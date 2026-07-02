// Package agents provides concrete agent implementations.
// Task 3.7: Support Agent UC-C1 - tests
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/domain/usage"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

type mockKnowledgeSearch struct {
	results *knowledge.SearchResults
	err     error
}

type supportUsageStub struct {
	inputs []usage.RecordEventInput
}

func (s *supportUsageStub) RecordEvent(_ context.Context, input usage.RecordEventInput) (*usage.Event, error) {
	s.inputs = append(s.inputs, input)
	return &usage.Event{}, nil
}

func (m *mockKnowledgeSearch) BuildEvidencePack(_ context.Context, input knowledge.BuildEvidencePackInput) (*knowledge.EvidencePack, error) {
	if m.err != nil {
		return nil, m.err
	}
	return searchResultsToEvidencePack(input.Query, m.results), nil
}

func (m *mockKnowledgeSearch) HybridSearch(_ context.Context, _ knowledge.SearchInput) (*knowledge.SearchResults, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.results == nil {
		return emptyResults(), nil
	}
	return m.results, nil
}

func setupAgentTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// insertSupportAgentDefinition seeds a single workspace's agent_definition
// row using the literal id "support-agent". Since AGENTDEF-BOOTSTRAP-IMPL-B2-001,
// triggerSupportRun resolves the id by (workspace_id, agent_type) rather than
// assuming this literal, so this helper remains valid for single-workspace
// test setups; it just no longer represents the only id a workspace can have
// (real bootstrap-provisioned rows get a UUID id via AGENTDEF-BOOTSTRAP-IMPL-A-001).
func insertSupportAgentDefinition(t *testing.T, db *sql.DB, workspaceID string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('support-agent', ?, 'Support Agent', 'support', 'active')`,
		workspaceID,
	)
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

func newTestSupportAgent(t *testing.T, db *sql.DB, search SupportEvidenceBuilder) *SupportAgent {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	if err := tool.RegisterBuiltInExecutors(registry, tool.BuiltinServices{
		DB:   db,
		Case: crm.NewCaseService(db),
	}); err != nil {
		t.Fatalf("register builtins: %v", err)
	}
	if err := registry.EnsureBuiltInToolDefinitionsForAllWorkspaces(context.Background()); err != nil {
		t.Fatalf("ensure builtins: %v", err)
	}
	return NewSupportAgentWithDB(orch, registry, search, db)
}

func TestSupportAgent_AllowedTools(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	tools := sa.AllowedTools()
	required := []string{"update_case", "send_reply", "create_task", "search_knowledge", "get_case", "get_contact"}
	if len(tools) != len(required) {
		t.Fatalf("expected %d tools, got %d", len(required), len(tools))
	}
	seen := make(map[string]bool, len(tools))
	for _, item := range tools {
		seen[item] = true
	}
	for _, item := range required {
		if !seen[item] {
			t.Fatalf("missing tool %s", item)
		}
	}
}

func TestDetermineAction_NoEvidence_Escalates(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	action := sa.determineAction(
		SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help", Priority: "high"},
		&CaseContext{ID: "case-1", WorkspaceID: "ws-1", Priority: "high"},
		searchResultsToEvidencePack("help", emptyResults()),
	)
	if action.Type != supportActionEscalate {
		t.Fatalf("expected escalate, got %s", action.Type)
	}
}

func TestDetermineAction_HighScore_Resolves(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	action := sa.determineAction(
		SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help", Priority: "medium"},
		&CaseContext{ID: "case-1", WorkspaceID: "ws-1", Priority: "medium"},
		searchResultsToEvidencePack("help", &knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: supportRawHybridScore(0.95)}}}),
	)
	if action.Type != supportActionUpdateCase {
		t.Fatalf("expected update_case, got %s", action.Type)
	}
}

func TestDetermineAction_HighScoreHighPriorityRequiresApproval(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	action := sa.determineAction(
		SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help"},
		&CaseContext{ID: "case-1", WorkspaceID: "ws-1", Priority: "high"},
		searchResultsToEvidencePack("help", &knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: supportRawHybridScore(0.95)}}}),
	)
	if !actionRequiresApproval(action) {
		t.Fatalf("expected action to require approval, metadata = %q", action.Metadata)
	}

	var metadata map[string]string
	if err := json.Unmarshal([]byte(action.Metadata), &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata["sensitivity"] != sensitivityHigh {
		t.Fatalf("sensitivity = %q want %q", metadata["sensitivity"], sensitivityHigh)
	}
	if metadata["approval_reason"] != sensitivityHighReason {
		t.Fatalf("approval_reason = %q want %q", metadata["approval_reason"], sensitivityHighReason)
	}
	if metadata["source"] != "support-agent" {
		t.Fatalf("source = %q want %q", metadata["source"], "support-agent")
	}
}

func TestDetermineAction_MediumScore_Abstains(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	action := sa.determineAction(
		SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help", Priority: "medium"},
		&CaseContext{ID: "case-1", WorkspaceID: "ws-1", Priority: "medium"},
		searchResultsToEvidencePack("help", &knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: supportRawHybridScore(0.7)}}}),
	)
	if action.Type != supportActionAbstain {
		t.Fatalf("expected abstain, got %s", action.Type)
	}
}

func TestSupportAgent_Run_MissingCaseID(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	_, err := sa.Run(context.Background(), SupportAgentConfig{WorkspaceID: "ws-1", CustomerQuery: "help"})
	if err != ErrCaseIDRequired {
		t.Fatalf("expected ErrCaseIDRequired, got %v", err)
	}
}

func TestSupportAgent_Run_MissingWorkspaceID(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	_, err := sa.Run(context.Background(), SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help"})
	if err != ErrWorkspaceIDRequired {
		t.Fatalf("expected ErrWorkspaceIDRequired, got %v", err)
	}
}

func TestSupportAgent_Run_EscalatesWhenNoKnowledge(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "high")
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "I need help",
		Priority:      "high",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if stored.Status != agent.StatusEscalated {
		t.Fatalf("expected escalated, got %s", stored.Status)
	}

	caseTicket, err := crm.NewCaseService(db).Get(context.Background(), wsID, caseID)
	if err != nil {
		t.Fatalf("get case: %v", err)
	}
	if caseTicket.Status != agent.StatusEscalated {
		t.Fatalf("expected escalated case, got %s", caseTicket.Status)
	}
}

func TestSupportAgent_Run_ResolvesWhenHighConfidence(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{
		results: &knowledge.SearchResults{
			Items: []knowledge.SearchResult{{KnowledgeItemID: "ki-1", Score: supportRawHybridScore(0.9), Snippet: "restart the service"}},
		},
	})

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "service is down",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if stored.Status != agent.StatusSuccess {
		t.Fatalf("expected success, got %s", stored.Status)
	}
	if got := string(stored.RetrievalQueries); got != `["service is down"]` {
		t.Fatalf("expected retrieval query trace, got %s", got)
	}
	if got := string(stored.RetrievedEvidenceIDs); got == "" || got == "[]" {
		t.Fatalf("expected evidence ids, got %s", got)
	}
	if stored.TriggeredByUserID == nil || *stored.TriggeredByUserID != ownerID {
		t.Fatalf("expected triggered_by to be preserved, got %#v", stored.TriggeredByUserID)
	}

	caseTicket, err := crm.NewCaseService(db).Get(context.Background(), wsID, caseID)
	if err != nil {
		t.Fatalf("get case: %v", err)
	}
	if caseTicket.Status != "resolved" {
		t.Fatalf("expected resolved case, got %s", caseTicket.Status)
	}

	events, err := audit.NewAuditService(db).ListByAction(context.Background(), wsID, "agent.support.run.completed", 10, 0)
	if err != nil {
		t.Fatalf("ListByAction() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected support run audit event")
	}
}

func TestSupportAgent_Run_HighConfidenceHighPriorityCreatesApprovalRequest(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	approverID := seedSupportUser(t, db, wsID)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "high")
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{
		results: &knowledge.SearchResults{
			Items: []knowledge.SearchResult{{KnowledgeItemID: "ki-1", Score: supportRawHybridScore(0.9), Snippet: "apply privileged remediation"}},
		},
	})

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "service is down for all enterprise users",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if stored.Status != agent.StatusEscalated {
		t.Fatalf("expected escalated, got %s", stored.Status)
	}

	var output map[string]any
	if err := json.Unmarshal(stored.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got, _ := output["Type"].(string); got != supportPendingApprovalAction {
		t.Fatalf("output type = %q want %q", got, supportPendingApprovalAction)
	}
	approvalID, _ := output["ApprovalID"].(string)
	if approvalID == "" {
		t.Fatal("expected approval id in stored output")
	}

	var action, reason, status, storedApproverID string
	if err := db.QueryRowContext(context.Background(), `
		SELECT action, reason, status, approver_id
		FROM approval_request
		WHERE id = ?
	`, approvalID).Scan(&action, &reason, &status, &storedApproverID); err != nil {
		t.Fatalf("query approval_request: %v", err)
	}
	if action != "support.case.update" {
		t.Fatalf("approval action = %q want %q", action, "support.case.update")
	}
	if reason != sensitivityHighReason {
		t.Fatalf("approval reason = %q want %q", reason, sensitivityHighReason)
	}
	if status != "pending" {
		t.Fatalf("approval status = %q want pending", status)
	}
	if storedApproverID != approverID {
		t.Fatalf("approval approver_id = %q want %q", storedApproverID, approverID)
	}

	caseTicket, err := crm.NewCaseService(db).Get(context.Background(), wsID, caseID)
	if err != nil {
		t.Fatalf("get case: %v", err)
	}
	if caseTicket.Status != "open" {
		t.Fatalf("expected case to remain open pending approval, got %s", caseTicket.Status)
	}
}

func TestSupportAgent_Run_AbstainsWhenConfidenceIsMedium(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{
		results: &knowledge.SearchResults{
			Items: []knowledge.SearchResult{{Score: supportRawHybridScore(0.7), Snippet: "possible workaround"}},
		},
	})

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "service is unstable",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if stored.Status != agent.StatusAbstained {
		t.Fatalf("expected abstained, got %s", stored.Status)
	}

	caseTicket, err := crm.NewCaseService(db).Get(context.Background(), wsID, caseID)
	if err != nil {
		t.Fatalf("get case: %v", err)
	}
	if caseTicket.Status != "open" {
		t.Fatalf("expected open case, got %s", caseTicket.Status)
	}
}

func emptyResults() *knowledge.SearchResults {
	return &knowledge.SearchResults{Items: []knowledge.SearchResult{}}
}

func searchResultsToEvidencePack(query string, results *knowledge.SearchResults) *knowledge.EvidencePack {
	if results == nil {
		results = emptyResults()
	}
	sources := make([]knowledge.Evidence, 0, len(results.Items))
	methods := make([]knowledge.EvidenceMethod, 0, len(results.Items))
	confidence := knowledge.ConfidenceLow
	if len(results.Items) > 0 {
		normalizedScore := knowledge.NormalizeHybridSearchScore(results.Items[0].Score)
		switch {
		case normalizedScore >= 0.85:
			confidence = knowledge.ConfidenceHigh
		case normalizedScore >= 0.55:
			confidence = knowledge.ConfidenceMedium
		}
	}
	for i, item := range results.Items {
		method := item.Method
		if method == "" {
			method = knowledge.EvidenceMethodHybrid
		}
		snippet := item.Snippet
		var snippetPtr *string
		if snippet != "" {
			snippetPtr = &snippet
		}
		evidenceID := item.KnowledgeItemID
		if evidenceID == "" {
			evidenceID = fmt.Sprintf("ev-test-%d", i)
		}
		sources = append(sources, knowledge.Evidence{
			ID:              evidenceID,
			KnowledgeItemID: item.KnowledgeItemID,
			Method:          method,
			Score:           item.Score,
			Snippet:         snippetPtr,
		})
		methods = append(methods, method)
	}
	return &knowledge.EvidencePack{
		SchemaVersion:        knowledge.EvidencePackSchemaVersion,
		Query:                query,
		Sources:              sources,
		SourceCount:          len(sources),
		DedupCount:           0,
		Confidence:           confidence,
		FilteredCount:        0,
		Warnings:             []string{},
		RetrievalMethodsUsed: methods,
		BuiltAt:              time.Now().UTC(),
	}
}

func supportRawHybridScore(normalized float64) float64 {
	return normalized * knowledge.MaxHybridSearchScore()
}

func seedSupportWorkspace(t *testing.T, db *sql.DB) (string, string) {
	t.Helper()
	suffix := time.Now().UTC().Format("150405.000000000")
	wsID := "ws-support-" + suffix
	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, 'Support Workspace', ?, datetime('now'), datetime('now'))
	`, wsID, "support-"+suffix)
	if err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	ownerID := "user-support-" + suffix
	_, err = db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, 'Support Owner', 'active', datetime('now'), datetime('now'))
	`, ownerID, wsID, ownerID+"@example.com")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return wsID, ownerID
}

func seedSupportUser(t *testing.T, db *sql.DB, wsID string) string {
	t.Helper()
	suffix := time.Now().UTC().Format("150405.000000000")
	userID := "user-support-extra-" + suffix
	_, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, 'Support Approver', 'active', datetime('now'), datetime('now'))
	`, userID, wsID, userID+"@example.com")
	if err != nil {
		t.Fatalf("insert support user: %v", err)
	}
	return userID
}

func seedSupportCase(t *testing.T, db *sql.DB, wsID, ownerID, priority string) string {
	t.Helper()
	contact, err := crm.NewContactService(db).Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		FirstName:   "Ana",
		LastName:    "Cliente",
		Email:       "ana@example.com",
		Status:      "active",
		OwnerID:     ownerID,
	})
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}
	ticket, err := crm.NewCaseService(db).Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		ContactID:   contact.ID,
		OwnerID:     ownerID,
		Subject:     "Service issue",
		Description: "Customer cannot access the service",
		Priority:    priority,
		Status:      "open",
	})
	if err != nil {
		t.Fatalf("create case: %v", err)
	}
	return ticket.ID
}

func supportRunContext(ctx context.Context, workspaceID, ownerID string) context.Context {
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, workspaceID)
	return context.WithValue(ctx, ctxkeys.UserID, ownerID)
}

func TestSupportAgent_NewSupportAgent_Constructor(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	sa := NewSupportAgent(orch, registry, &mockKnowledgeSearch{results: emptyResults()})
	if sa == nil {
		t.Fatal("NewSupportAgent returned nil")
	}
}

func TestSupportAgent_Objective_ReturnsJSON(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	obj := sa.Objective()
	if len(obj) == 0 {
		t.Fatal("Objective() returned empty")
	}
}

func TestSupportError_Error_ReturnsMessage(t *testing.T) {
	err := ErrSupportDBNotConfigured
	if err.Error() == "" {
		t.Fatal("SupportError.Error() should not be empty")
	}
}

func TestSupportAgent_RequestSupportApprovalAndBuildApprovalEscalationResult(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	approverID := seedSupportUser(t, db, wsID)
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	ctx := supportRunContext(context.Background(), wsID, ownerID)
	caseContext := &CaseContext{
		ID:          "case-approval-1",
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Sensitive support issue",
		Priority:    "high",
	}
	action := &Action{
		Type:       supportActionUpdateCase,
		CaseID:     caseContext.ID,
		Status:     "resolved",
		Details:    "Apply sensitive remediation",
		Confidence: 90,
	}

	approvalID, err := sa.requestSupportApproval(ctx, caseContext, action)
	if err != nil {
		t.Fatalf("requestSupportApproval() error = %v", err)
	}
	if approvalID == "" {
		t.Fatal("expected non-empty approval id")
	}

	var storedAction, storedApproverID string
	if err := db.QueryRowContext(context.Background(), `
		SELECT action, approver_id
		FROM approval_request
		WHERE id = ?
	`, approvalID).Scan(&storedAction, &storedApproverID); err != nil {
		t.Fatalf("query approval_request: %v", err)
	}
	if storedAction != "support.case.update" {
		t.Fatalf("approval action = %q want %q", storedAction, "support.case.update")
	}
	if storedApproverID != approverID {
		t.Fatalf("approval approver_id = %q want %q", storedApproverID, approverID)
	}

	tokens := int64(0)
	cost := 0.0
	result, err := sa.buildApprovalEscalationResult(
		ctx,
		time.Now().Add(-time.Second),
		SupportAgentConfig{WorkspaceID: wsID, CaseID: caseContext.ID, CustomerQuery: "please help"},
		caseContext,
		emptySupportEvidencePack("please help"),
		action,
		&tokens,
		&cost,
	)
	if err != nil {
		t.Fatalf("buildApprovalEscalationResult() error = %v", err)
	}
	if result.Status != agent.StatusEscalated {
		t.Fatalf("status = %q want %q", result.Status, agent.StatusEscalated)
	}

	var output map[string]any
	if err := json.Unmarshal(result.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got, _ := output["Type"].(string); got != "pending_approval" {
		t.Fatalf("action = %q want pending_approval", got)
	}
	if got, _ := output["ApprovalID"].(string); got == "" {
		t.Fatal("expected approval_id in output")
	}
}

func TestSupportAgent_RequestSupportApproval_FailsClosedWithoutDistinctApprover(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	ctx := supportRunContext(context.Background(), wsID, ownerID)
	caseContext := &CaseContext{
		ID:          "case-approval-blocked-1",
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Sensitive support issue",
		Priority:    "high",
	}
	action := &Action{
		Type:       supportActionUpdateCase,
		CaseID:     caseContext.ID,
		Status:     "resolved",
		Details:    "Apply sensitive remediation",
		Confidence: 90,
	}

	_, err := sa.requestSupportApproval(ctx, caseContext, action)
	if !errors.Is(err, ErrSupportApprovalApproverNotFound) {
		t.Fatalf("requestSupportApproval() error = %v want %v", err, ErrSupportApprovalApproverNotFound)
	}
}

func TestSupportAgent_FailSupportRunReturnsCauseAndMarksFailed(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, _ := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	run, err := agent.NewOrchestrator(db).TriggerAgent(context.Background(), agent.TriggerAgentInput{
		AgentID:     "support-agent",
		WorkspaceID: wsID,
		TriggerType: agent.TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent() error = %v", err)
	}

	cause := errors.New("boom")
	if err := sa.failSupportRun(context.Background(), run, cause); !errors.Is(err, cause) {
		t.Fatalf("failSupportRun() error = %v want %v", err, cause)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if stored.Status != agent.StatusFailed {
		t.Fatalf("status = %q want %q", stored.Status, agent.StatusFailed)
	}

	events, err := audit.NewAuditService(db).ListByAction(context.Background(), wsID, "agent.support.run.failed", 10, 0)
	if err != nil {
		t.Fatalf("ListByAction() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected failed support audit event")
	}
}

func TestSupportAgent_Run_RecordsUsageForCompletedRun(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	registry := tool.NewToolRegistry(db)
	if err := tool.RegisterBuiltInExecutors(registry, tool.BuiltinServices{
		DB:   db,
		Case: crm.NewCaseService(db),
	}); err != nil {
		t.Fatalf("register builtins: %v", err)
	}
	if err := registry.EnsureBuiltInToolDefinitionsForAllWorkspaces(context.Background()); err != nil {
		t.Fatalf("ensure builtins: %v", err)
	}
	usageStub := &supportUsageStub{}
	sa := NewSupportAgentWithDBAndUsage(agent.NewOrchestrator(db), registry, &mockKnowledgeSearch{
		results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{KnowledgeItemID: "ki-1", Score: supportRawHybridScore(0.9), Snippet: "restart the service"}}},
	}, db, usageStub)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "service is down",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var supportEvent *usage.RecordEventInput
	for i := range usageStub.inputs {
		event := usageStub.inputs[i]
		if event.RunID != nil && *event.RunID == run.ID && event.ToolName == nil {
			supportEvent = &usageStub.inputs[i]
			break
		}
	}
	if supportEvent == nil {
		t.Fatalf("expected support runtime usage event, got %#v", usageStub.inputs)
	}
	if supportEvent.WorkspaceID != wsID || supportEvent.ActorID != ownerID {
		t.Fatalf("unexpected usage attribution: %#v", supportEvent)
	}
	if supportEvent.ActorType != "user" {
		t.Fatalf("unexpected actor type: %q", supportEvent.ActorType)
	}
	if supportEvent.OutputUnits != 0 {
		t.Fatalf("expected zero output units with current runtime totals, got %d", supportEvent.OutputUnits)
	}
	if supportEvent.LatencyMs == nil {
		t.Fatal("expected latency")
	}
}

// Tests for triggerSupportRun's workspace-scoped AgentID resolution
// (AGENTDEF-BOOTSTRAP-IMPL-B1-001 pinned the pre-fix behavior;
// AGENTDEF-BOOTSTRAP-IMPL-B2-001 replaced the hardcoded literal id lookup
// with resolveSupportAgentID, which resolves each workspace's own
// support-typed agent_definition row instead of a globally-shared literal.

// TestSupportAgent_TriggerSupportRun_SucceedsForWorkspaceBootstrappedByRegister
// is the end-to-end proof that AGENTDEF-BOOTSTRAP-IMPL-A-001 (Register's
// per-workspace agent_definition bootstrap) and AGENTDEF-BOOTSTRAP-IMPL-B2-001
// (workspace-scoped id resolution) together close the original gap: a freshly
// registered workspace can trigger the support agent with no manual
// agent_definition insert anywhere in the test.
func TestSupportAgent_TriggerSupportRun_SucceedsForWorkspaceBootstrappedByRegister(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	// domainauth.Register issues a JWT as part of registration; pkgauth.GenerateJWT
	// panics if JWT_SECRET is unset (this package has no TestMain that sets it).
	t.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!")

	authSvc := domainauth.NewAuthService(db)
	result, err := authSvc.Register(context.Background(), domainauth.RegisterInput{
		Email:         "bootstrap@example.com",
		Password:      "SecurePass123!",
		DisplayName:   "Bootstrap Owner",
		WorkspaceName: "Bootstrap Workspace",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	run, err := sa.triggerSupportRun(context.Background(), SupportAgentConfig{
		WorkspaceID:   result.WorkspaceID,
		CaseID:        "case-1",
		CustomerQuery: "help",
	})
	if err != nil {
		t.Fatalf("triggerSupportRun() error = %v; want nil (workspace was bootstrapped by Register)", err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
	if run.WorkspaceID != result.WorkspaceID {
		t.Fatalf("run.WorkspaceID = %q want %q", run.WorkspaceID, result.WorkspaceID)
	}
	if run.DefinitionID == "" {
		t.Fatal("expected non-empty run.DefinitionID")
	}
}

func TestSupportAgent_TriggerSupportRun_SucceedsWithLiteralIDInSingleWorkspace(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, _ := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	run, err := sa.triggerSupportRun(context.Background(), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        "case-1",
		CustomerQuery: "help",
	})
	if err != nil {
		t.Fatalf("triggerSupportRun() error = %v", err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
	if run.WorkspaceID != wsID {
		t.Fatalf("run.WorkspaceID = %q want %q", run.WorkspaceID, wsID)
	}
	if run.DefinitionID != "support-agent" {
		t.Fatalf("run.DefinitionID = %q want %q", run.DefinitionID, "support-agent")
	}
}

func TestSupportAgent_TriggerSupportRun_FailsWhenNoAgentDefinitionRowExists(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, _ := seedSupportWorkspace(t, db)
	// Deliberately do not provision any agent_definition row: this
	// simulates a workspace that predates the AGENTDEF-BOOTSTRAP-IMPL-A-001
	// bootstrap default and was never provisioned.
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	run, err := sa.triggerSupportRun(context.Background(), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        "case-1",
		CustomerQuery: "help",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if run != nil {
		t.Fatalf("expected nil run, got %#v", run)
	}
	if !errors.Is(err, ErrSupportAgentNotProvisioned) {
		t.Fatalf("triggerSupportRun() error = %v, want %v", err, ErrSupportAgentNotProvisioned)
	}
}

func TestSupportAgent_TriggerSupportRun_ResolvesOwnRowForDifferentWorkspaceWithSameLiteralID(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsA, _ := seedSupportWorkspace(t, db)
	wsB, _ := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsA)
	// wsB gets its own row with a distinct id, mirroring what
	// AGENTDEF-BOOTSTRAP-IMPL-A-001's bootstrap now does per-workspace (a
	// UUID id, not the literal "support-agent"). The lookup is scoped by
	// (workspace_id, agent_type), so wsB must resolve its own row and
	// succeed even though wsA independently has a row of agent_type
	// "support" too.
	wsBAgentID := "agent-def-ws-b"
	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES (?, ?, 'Support Agent', 'support', 'active')`,
		wsBAgentID, wsB,
	); err != nil {
		t.Fatalf("insert agent_definition for wsB: %v", err)
	}
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	run, err := sa.triggerSupportRun(context.Background(), SupportAgentConfig{
		WorkspaceID:   wsB,
		CaseID:        "case-1",
		CustomerQuery: "help",
	})
	if err != nil {
		t.Fatalf("triggerSupportRun() error = %v", err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
	if run.WorkspaceID != wsB {
		t.Fatalf("run.WorkspaceID = %q want %q", run.WorkspaceID, wsB)
	}
	if run.DefinitionID != wsBAgentID {
		t.Fatalf("run.DefinitionID = %q want %q (wsB's own row, not wsA's)", run.DefinitionID, wsBAgentID)
	}
}

func TestSupportAgent_InsertSupportAgentDefinition_CollidesAcrossWorkspacesOnGlobalPrimaryKey(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsA, _ := seedSupportWorkspace(t, db)
	wsB, _ := seedSupportWorkspace(t, db)

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('support-agent', ?, 'Support Agent', 'support', 'active')`,
		wsA,
	)
	if err != nil {
		t.Fatalf("insert agent_definition for wsA: %v", err)
	}

	_, err = db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('support-agent', ?, 'Support Agent', 'support', 'active')`,
		wsB,
	)
	if err == nil {
		t.Fatal("expected PRIMARY KEY constraint violation inserting duplicate id for a second workspace, got nil error")
	}
	if !strings.Contains(err.Error(), "UNIQUE constraint failed") && !strings.Contains(err.Error(), "PRIMARY KEY") {
		t.Fatalf("expected UNIQUE/PRIMARY KEY constraint violation, got: %v", err)
	}
}
