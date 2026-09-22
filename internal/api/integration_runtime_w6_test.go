package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
	"github.com/matiasleandrokruk/fenix/internal/domain/flowinterop"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	evidenceoutbox "github.com/matiasleandrokruk/fenix/internal/infra/integration/evidenceoutbox"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

const w6BlackboardOnlyMarker = "RAW_BLACKBOARD_REASONING_SHOULD_NOT_LEAVE"

type w6AuditStub struct {
	details []map[string]any
}

func (s *w6AuditStub) LogWithDetails(
	_ context.Context,
	_, _ string,
	_ audit.ActorType,
	_ string,
	_, _ *string,
	details *audit.EventDetails,
	_ audit.Outcome,
) error {
	if details == nil {
		return nil
	}
	if meta, ok := details.Metadata.(map[string]any); ok {
		s.details = append(s.details, meta)
	}
	return nil
}

type w6EvidenceRecorder struct {
	requests []tool.CapabilityEvidenceRequest
}

func (r *w6EvidenceRecorder) RecordCapabilityEvidence(
	_ context.Context,
	request tool.CapabilityEvidenceRequest,
) (tool.CapabilityEvidenceResult, error) {
	r.requests = append(r.requests, request)
	return tool.CapabilityEvidenceResult{
		State: "verified",
		AuditMetadata: map[string]any{
			"event_id":      "event-w6",
			"checkpoint_id": "checkpoint-w6",
		},
	}, nil
}

type w6BoundaryEvidenceRecorder struct {
	requests  []tool.CapabilityEvidenceRequest
	envelopes []evidence.Envelope
}

func (r *w6BoundaryEvidenceRecorder) RecordCapabilityEvidence(
	_ context.Context,
	request tool.CapabilityEvidenceRequest,
) (tool.CapabilityEvidenceResult, error) {
	envelope, err := evidence.BuildRuntimeEnvelope(request)
	if err != nil {
		return tool.CapabilityEvidenceResult{State: "indeterminate"}, err
	}
	r.requests = append(r.requests, request)
	r.envelopes = append(r.envelopes, envelope)
	return tool.CapabilityEvidenceResult{
		State: "recorded",
		AuditMetadata: map[string]any{
			"execution_id": envelope.ExecutionID,
			"stream_id":    envelope.StreamID,
		},
	}, nil
}

type w6ResilienceEvidenceRecorder struct {
	result   tool.CapabilityEvidenceResult
	err      error
	requests []tool.CapabilityEvidenceRequest
}

func (r *w6ResilienceEvidenceRecorder) RecordCapabilityEvidence(
	_ context.Context,
	request tool.CapabilityEvidenceRequest,
) (tool.CapabilityEvidenceResult, error) {
	r.requests = append(r.requests, request)
	return r.result, r.err
}

type w6RestartLifecycleSink struct {
	recordErr         error
	lookupRef         *evidence.ProofReference
	progress          evidence.VerificationProgress
	recordCalls       int
	lookupCalls       int
	verificationCalls int
	envelopes         []evidence.Envelope
}

func (s *w6RestartLifecycleSink) RecordEvidence(
	_ context.Context,
	envelope evidence.Envelope,
) (evidence.ProofReference, error) {
	s.recordCalls++
	s.envelopes = append(s.envelopes, envelope)
	return evidence.ProofReference{}, s.recordErr
}

func (s *w6RestartLifecycleSink) LookupEvidence(
	_ context.Context,
	_, _ string,
) (*evidence.ProofReference, error) {
	s.lookupCalls++
	return s.lookupRef, nil
}

func (s *w6RestartLifecycleSink) CheckpointAndVerify(
	_ context.Context,
	_ string,
) (evidence.VerificationProgress, error) {
	s.verificationCalls++
	return s.progress, nil
}

type w6ProviderCall struct {
	authorization string
	traceID       string
	executionID   string
	request       flowinterop.Request
}

func TestW6A_BlackboardExecutesGovernedM2SFWithOptionalEvidence(t *testing.T) {
	for _, tc := range []struct {
		name            string
		evidenceEnabled bool
		wantEvidence    bool
	}{
		{name: "evidence off", evidenceEnabled: false, wantEvidence: false},
		{name: "evidence on", evidenceEnabled: true, wantEvidence: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			providerCalls := make(chan w6ProviderCall, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				var semanticRequest flowinterop.Request
				if err := json.NewDecoder(request.Body).Decode(&semanticRequest); err != nil {
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				providerCalls <- w6ProviderCall{
					authorization: request.Header.Get("Authorization"),
					traceID:       request.Header.Get("X-Fenix-Trace-ID"),
					executionID:   request.Header.Get("X-Fenix-Execution-ID"),
					request:       semanticRequest,
				}
				_ = json.NewEncoder(w).Encode(flowinterop.Result{
					ContractVersion: flowinterop.ContractVersion,
					Operation:       flowinterop.OperationExport,
					Status:          flowinterop.ResultSucceeded,
					Artifacts: []flowinterop.Artifact{{
						Format:  flowinterop.FormatSalesforceFlowXML,
						Name:    "Demo",
						Content: "<Flow />",
					}},
					Fidelity: flowinterop.FidelityReport{
						ContractVersion: flowinterop.ContractVersion,
						Level:           flowinterop.FidelityGuaranteed,
						FlowFamily:      flowinterop.FlowFamilyAutolaunched,
					},
					SemanticMetadata: flowinterop.SemanticMetadata{
						FlowFamily:  flowinterop.FlowFamilyAutolaunched,
						FlowAPIName: "Demo",
					},
					ProviderReference: "m2sf-w6",
				})
			}))
			defer server.Close()

			db, workspace := setupW6FunctionalWorkspace(t)
			auditStub := &w6AuditStub{}
			evidenceRecorder := &w6EvidenceRecorder{}
			registry := tool.NewToolRegistryWithRuntime(db, nil, auditStub)

			settings := crossPlatformRuntimeSettings{
				M2SF: providerRuntimeSettings{
					BaseURL: server.URL,
					Token:   "test-token",
					Enabled: true,
				},
				Timeout:              2 * time.Second,
				M2SFOptionalEvidence: tc.evidenceEnabled,
			}
			if err := configureCrossPlatformRuntime(registry, settings); err != nil {
				t.Fatalf("configureCrossPlatformRuntime: %v", err)
			}
			governancePlanner, err := newCrossPlatformGovernancePlanner(
				crossPlatformPolicySelector{optionalM2SFEvidence: tc.evidenceEnabled},
			)
			if err != nil {
				t.Fatalf("newCrossPlatformGovernancePlanner: %v", err)
			}
			registry.SetCapabilityGovernancePlanner(governancePlanner)
			registry.SetCapabilityEvidenceRecorder(evidenceRecorder)

			executor := blackboard.NewPlannerExecutor(db, nil, nil, registry, nil)
			plan := w6ExportPlan(workspace.ID)
			traceID := "trace-w6-" + strings.ReplaceAll(tc.name, " ", "-")
			ctx := ctxkeys.WithValue(context.Background(), ctxkeys.TraceID, traceID)
			ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, "agent-w6")
			ctx = ctxkeys.WithValue(ctx, ctxkeys.RunID, "run-w6")

			outcome, err := executor.Execute(ctx, workspace, plan)
			if err != nil {
				t.Fatalf("PlannerExecutor.Execute: %v", err)
			}
			if len(outcome.Executed) != 1 || outcome.Executed[0].Error != "" {
				t.Fatalf("unexpected execution outcome: %#v", outcome)
			}

			var result flowinterop.Result
			if err := json.Unmarshal(outcome.Executed[0].Result, &result); err != nil {
				t.Fatalf("decode M2SF result: %v", err)
			}
			if result.ProviderReference != "m2sf-w6" ||
				result.Status != flowinterop.ResultSucceeded ||
				result.Fidelity.Level != flowinterop.FidelityGuaranteed {
				t.Fatalf("unexpected M2SF result: %#v", result)
			}

			call := <-providerCalls
			if call.authorization != "Bearer test-token" {
				t.Fatalf("authorization = %q", call.authorization)
			}
			if call.traceID != traceID || strings.TrimSpace(call.executionID) == "" {
				t.Fatalf("provider correlation trace=%q execution=%q", call.traceID, call.executionID)
			}
			if call.request.Operation != flowinterop.OperationExport {
				t.Fatalf("provider operation = %q", call.request.Operation)
			}

			if got := len(evidenceRecorder.requests); got != boolCount(tc.wantEvidence) {
				t.Fatalf("evidence calls = %d, want %d", got, boolCount(tc.wantEvidence))
			}
			if tc.wantEvidence {
				request := evidenceRecorder.requests[0]
				if request.TraceID != call.traceID || request.ExecutionID != call.executionID {
					t.Fatalf(
						"evidence correlation trace=%q execution=%q, provider trace=%q execution=%q",
						request.TraceID,
						request.ExecutionID,
						call.traceID,
						call.executionID,
					)
				}
				if request.Descriptor.Name != string(flowinterop.OperationExport) {
					t.Fatalf("evidence capability = %q", request.Descriptor.Name)
				}
			}

			assertW6BlackboardProjection(t, db, workspace.ID)
			assertW6ToolAudit(t, auditStub, call.executionID, tc.wantEvidence)
		})
	}
}

func setupW6FunctionalWorkspace(t *testing.T) (*sql.DB, blackboard.CognitiveWorkspace) {
	t.Helper()
	db, err := isqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("sqlite.NewDB: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := isqlite.MigrateUp(db); err != nil {
		_ = db.Close()
		t.Fatalf("sqlite.MigrateUp: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	workspace := blackboard.CognitiveWorkspace{
		ID:          "cw-w6",
		WorkspaceID: "ws-w6",
	}
	if _, err := db.Exec(
		`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		 VALUES (?, ?, ?, datetime('now'), datetime('now'))`,
		workspace.WorkspaceID,
		"W6 Workspace",
		"w6-workspace",
	); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		 VALUES (?, ?, 'active', datetime('now'))`,
		workspace.ID,
		workspace.WorkspaceID,
	); err != nil {
		t.Fatalf("insert cognitive workspace: %v", err)
	}
	return db, workspace
}

func w6ExportPlan(cognitiveWorkspaceID string) *blackboard.CollaborativePlanningResult {
	params := json.RawMessage(`{
		"contract_version":"1",
		"operation":"salesforce.flow.export",
		"input":{
			"format":"mermaid",
			"name":"Demo",
			"content":"flowchart TD\\nStart --> End"
		}
	}`)
	proposal := blackboard.CollaborativePlanProposal{
		ProposalID:   "proposal-w6",
		HypothesisID: "hypothesis-w6",
		Summary:      "Export an agent-selected Salesforce Flow through the governed M2SF capability.",
		State:        blackboard.PlanningStateReady,
		Steps: []blackboard.ToolSequenceStep{{
			Sequence: 1,
			ToolName: string(flowinterop.OperationExport),
			Reason:   "Produce Salesforce Flow XML from the shared Flow artifact.",
			Params:   params,
		}},
	}
	return &blackboard.CollaborativePlanningResult{
		CognitiveWorkspaceID: cognitiveWorkspaceID,
		GeneratedAt:          time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC),
		State:                blackboard.PlanningStateReady,
		SelectedProposal:     &proposal,
		Proposals:            []blackboard.CollaborativePlanProposal{proposal},
	}
}

func assertW6BlackboardProjection(t *testing.T, db *sql.DB, cognitiveWorkspaceID string) {
	t.Helper()
	memory := blackboard.NewMemoryStore(db)
	entry, err := memory.Get(
		context.Background(),
		cognitiveWorkspaceID,
		blackboard.DefaultPlannerExecutionResultMemoryKey,
	)
	if err != nil {
		t.Fatalf("load Blackboard execution result: %v", err)
	}
	raw := string(entry.Value)
	if !strings.Contains(raw, "m2sf-w6") {
		t.Fatalf("Blackboard result is missing provider reference: %s", raw)
	}
	for _, forbidden := range []string{"verification_bundle", "portable_bundle", "reasoning_trace"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("Blackboard leaked forbidden field %q: %s", forbidden, raw)
		}
	}

	timeline := blackboard.NewReasoningTimeline(db)
	events, err := timeline.List(
		context.Background(),
		cognitiveWorkspaceID,
		blackboard.TimelineFilter{EventType: blackboard.EventTypeObservation},
	)
	if err != nil {
		t.Fatalf("load Blackboard observations: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("Blackboard observations = %d, want 1", len(events))
	}
}

func assertW6ToolAudit(
	t *testing.T,
	auditStub *w6AuditStub,
	executionID string,
	evidenceExpected bool,
) {
	t.Helper()
	for _, meta := range auditStub.details {
		if meta["capability_name"] != string(flowinterop.OperationExport) {
			continue
		}
		if meta["execution_id"] != executionID {
			t.Fatalf("audit execution_id = %#v, want %q", meta["execution_id"], executionID)
		}
		evidenceMeta, ok := meta["evidence"].(map[string]any)
		if !ok {
			t.Fatalf("audit evidence metadata = %#v", meta["evidence"])
		}
		if evidenceExpected {
			if evidenceMeta["state"] != "verified" || evidenceMeta["event_id"] != "event-w6" {
				t.Fatalf("audit evidence metadata = %#v", evidenceMeta)
			}
		} else if evidenceMeta["state"] != "not_planned" {
			t.Fatalf("audit evidence metadata = %#v", evidenceMeta)
		}
		return
	}
	t.Fatal("governed M2SF audit entry not found")
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestW6B_CollaborativePlanExecutesDistinctGovernedStepsWithStableRetryIdentity(t *testing.T) {
	db, workspace := setupW6FunctionalWorkspace(t)
	seedW6CollaborativePlanningState(t, db, workspace.ID, true)

	var mu sync.Mutex
	providerCalls := make([]w6ProviderCall, 0, 3)
	exportAttempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var semanticRequest flowinterop.Request
		if err := json.NewDecoder(request.Body).Decode(&semanticRequest); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		call := w6ProviderCall{
			authorization: request.Header.Get("Authorization"),
			traceID:       request.Header.Get("X-Fenix-Trace-ID"),
			executionID:   request.Header.Get("X-Fenix-Execution-ID"),
			request:       semanticRequest,
		}
		mu.Lock()
		providerCalls = append(providerCalls, call)
		if semanticRequest.Operation == flowinterop.OperationExport {
			exportAttempts++
			attempt := exportAttempts
			mu.Unlock()
			if attempt == 1 {
				http.Error(w, "temporary provider failure", http.StatusServiceUnavailable)
				return
			}
		} else {
			mu.Unlock()
		}

		_ = json.NewEncoder(w).Encode(w6ProviderResult(semanticRequest.Operation))
	}))
	defer server.Close()

	registry := tool.NewToolRegistry(db)
	settings := crossPlatformRuntimeSettings{
		M2SF: providerRuntimeSettings{
			BaseURL: server.URL,
			Token:   "test-token",
			Enabled: true,
		},
		Timeout: 2 * time.Second,
	}
	if err := configureCrossPlatformRuntime(registry, settings); err != nil {
		t.Fatalf("configureCrossPlatformRuntime: %v", err)
	}
	boundaryRecorder := &w6BoundaryEvidenceRecorder{}
	governancePlanner, err := newCrossPlatformGovernancePlanner(
		crossPlatformPolicySelector{optionalM2SFEvidence: true},
	)
	if err != nil {
		t.Fatalf("newCrossPlatformGovernancePlanner: %v", err)
	}
	registry.SetCapabilityGovernancePlanner(governancePlanner)
	registry.SetCapabilityEvidenceRecorder(boundaryRecorder)

	plan, err := blackboard.NewPlanner(db).BuildWorkspacePlan(
		context.Background(),
		workspace.ID,
		blackboard.PlanningConfig{
			Now: time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC),
			ActionSteps: []blackboard.ToolSequenceStep{
				{
					ToolName: string(flowinterop.OperationExport),
					Reason:   "Export the collaboratively selected Flow.",
					Params:   w6FlowRequest(flowinterop.OperationExport),
				},
				{
					ToolName: string(flowinterop.OperationValidate),
					Reason:   "Validate the Flow after export.",
					Params:   w6FlowRequest(flowinterop.OperationValidate),
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("BuildWorkspacePlan: %v", err)
	}
	if plan.State != blackboard.PlanningStateReady || plan.SelectedProposal == nil {
		t.Fatalf("collaborative plan = %#v", plan)
	}
	if len(plan.SelectedProposal.Contributors) != 2 {
		t.Fatalf("contributors = %#v, want signal+evidence agents", plan.SelectedProposal.Contributors)
	}
	if len(plan.SelectedProposal.Steps) != 2 {
		t.Fatalf("bound action steps = %#v", plan.SelectedProposal.Steps)
	}

	traceID := "trace-w6-b"
	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.TraceID, traceID)
	ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, "agent-w6")
	ctx = ctxkeys.WithValue(ctx, ctxkeys.RunID, "run-w6-b")

	executor := blackboard.NewPlannerExecutor(db, nil, nil, registry, nil)
	outcome, err := executor.Execute(ctx, workspace, plan)
	if err != nil {
		t.Fatalf("PlannerExecutor.Execute: %v", err)
	}
	if len(outcome.Executed) != 2 {
		t.Fatalf("executed steps = %#v", outcome.Executed)
	}
	if outcome.Executed[0].Step.ToolName != string(flowinterop.OperationExport) ||
		outcome.Executed[1].Step.ToolName != string(flowinterop.OperationValidate) {
		t.Fatalf("executed tool order = %#v", outcome.Executed)
	}

	mu.Lock()
	calls := append([]w6ProviderCall(nil), providerCalls...)
	mu.Unlock()
	if len(calls) != 3 {
		t.Fatalf("provider calls = %d, want export retry + export success + validate success", len(calls))
	}
	if calls[0].request.Operation != flowinterop.OperationExport ||
		calls[1].request.Operation != flowinterop.OperationExport ||
		calls[2].request.Operation != flowinterop.OperationValidate {
		t.Fatalf("provider operation order = %#v", []flowinterop.Operation{
			calls[0].request.Operation,
			calls[1].request.Operation,
			calls[2].request.Operation,
		})
	}
	for _, call := range calls {
		if call.traceID != traceID || strings.TrimSpace(call.executionID) == "" {
			t.Fatalf("provider correlation trace=%q execution=%q", call.traceID, call.executionID)
		}
		if call.authorization != "Bearer test-token" {
			t.Fatalf("authorization = %q", call.authorization)
		}
	}

	exportExecutionID := calls[0].executionID
	if calls[1].executionID != exportExecutionID {
		t.Fatalf(
			"export retry changed execution identity: first=%q retry=%q",
			exportExecutionID,
			calls[1].executionID,
		)
	}
	if calls[2].executionID == exportExecutionID {
		t.Fatalf(
			"distinct external steps reused execution identity %q",
			exportExecutionID,
		)
	}

	if len(boundaryRecorder.envelopes) != 1 || len(boundaryRecorder.requests) != 1 {
		t.Fatalf(
			"evidence records = %d envelopes / %d requests, want one export evidence record",
			len(boundaryRecorder.envelopes),
			len(boundaryRecorder.requests),
		)
	}
	envelope := boundaryRecorder.envelopes[0]
	if envelope.ExecutionID != exportExecutionID {
		t.Fatalf("evidence execution_id = %q, want %q", envelope.ExecutionID, exportExecutionID)
	}
	if envelope.InputDigest == nil || envelope.OutputDigest == nil {
		t.Fatalf("evidence envelope must contain input/output digests: %#v", envelope)
	}
	rawEnvelope, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal evidence envelope: %v", err)
	}
	if strings.Contains(string(rawEnvelope), w6BlackboardOnlyMarker) {
		t.Fatalf("Blackboard-only collaboration state leaked into evidence envelope: %s", rawEnvelope)
	}
}

func seedW6CollaborativePlanningState(
	t *testing.T,
	db *sql.DB,
	cognitiveWorkspaceID string,
	includeEvidence bool,
) {
	t.Helper()
	now := time.Date(2026, 9, 22, 10, 55, 0, 0, time.UTC)
	store := blackboard.NewMemoryStore(db)

	persist := func(id, key string, value any) {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s: %v", key, err)
		}
		if err := store.Set(context.Background(), blackboard.AgentMemory{
			ID:                   id,
			CognitiveWorkspaceID: cognitiveWorkspaceID,
			Key:                  key,
			Value:                raw,
			Scope:                blackboard.MemoryScopeSession,
			CreatedAt:            now,
			UpdatedAt:            now,
		}); err != nil {
			t.Fatalf("persist %s: %v", key, err)
		}
	}

	persist("w6-b-arbitration", blackboard.DefaultArbitrationMemoryKey, blackboard.ArbitrationResult{
		CognitiveWorkspaceID: cognitiveWorkspaceID,
		GeneratedAt:          now,
		Ranked: []blackboard.RankedHypothesis{{
			Rank:  1,
			Score: 0.94,
			Hypothesis: blackboard.SignalHypothesis{
				ID:                   "hyp-w6-b",
				CognitiveWorkspaceID: cognitiveWorkspaceID,
				Content:              "export and validate the selected Salesforce Flow",
				Confidence:           0.94,
				Status:               blackboard.HypothesisStatusOpen,
				CreatedAt:            now.Add(-5 * time.Minute),
			},
		}},
	})
	persist(
		"w6-b-signal",
		"specialized_agents/blackboard-signal-agent/last_artifact",
		map[string]any{
			"contributor":   "blackboard-signal-agent",
			"artifact_type": "signal_hypothesis",
			"summary":       "Signal agent selected the Flow interoperability action. " + w6BlackboardOnlyMarker,
		},
	)
	if includeEvidence {
		persist(
			"w6-b-evidence",
			"specialized_agents/blackboard-evidence-agent/last_artifact",
			map[string]any{
				"contributor":   "blackboard-evidence-agent",
				"artifact_type": "evidence_finding",
				"summary":       "Evidence agent confirmed the source Flow artifact.",
			},
		)
	}
}

func w6FlowRequest(operation flowinterop.Operation) json.RawMessage {
	request := flowinterop.Request{
		ContractVersion: flowinterop.ContractVersion,
		Operation:       operation,
		Input: flowinterop.Artifact{
			Format:  flowinterop.FormatMermaid,
			Name:    "Demo",
			Content: "flowchart TD\nStart --> End",
		},
	}
	raw, _ := json.Marshal(request)
	return raw
}

func w6ProviderResult(operation flowinterop.Operation) flowinterop.Result {
	result := flowinterop.Result{
		ContractVersion: flowinterop.ContractVersion,
		Operation:       operation,
		Status:          flowinterop.ResultSucceeded,
		Fidelity: flowinterop.FidelityReport{
			ContractVersion: flowinterop.ContractVersion,
			Level:           flowinterop.FidelityGuaranteed,
			FlowFamily:      flowinterop.FlowFamilyAutolaunched,
		},
		SemanticMetadata: flowinterop.SemanticMetadata{
			FlowFamily:  flowinterop.FlowFamilyAutolaunched,
			FlowAPIName: "Demo",
		},
		ProviderReference: "m2sf-w6-b-" + string(operation),
	}
	if operation == flowinterop.OperationExport {
		result.Artifacts = []flowinterop.Artifact{{
			Format:  flowinterop.FormatSalesforceFlowXML,
			Name:    "Demo",
			Content: "<Flow />",
		}}
	}
	return result
}

func TestW6B_NotReadyCollaborationDefersBeforeProviderInvocation(t *testing.T) {
	db, workspace := setupW6FunctionalWorkspace(t)
	seedW6CollaborativePlanningState(t, db, workspace.ID, false)

	var providerMu sync.Mutex
	providerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		providerMu.Lock()
		providerCalls++
		providerMu.Unlock()
		_ = json.NewEncoder(w).Encode(w6ProviderResult(flowinterop.OperationExport))
	}))
	defer server.Close()

	registry := tool.NewToolRegistry(db)
	settings := crossPlatformRuntimeSettings{
		M2SF: providerRuntimeSettings{
			BaseURL: server.URL,
			Token:   "test-token",
			Enabled: true,
		},
		Timeout: 2 * time.Second,
	}
	if err := configureCrossPlatformRuntime(registry, settings); err != nil {
		t.Fatalf("configureCrossPlatformRuntime: %v", err)
	}
	governancePlanner, err := newCrossPlatformGovernancePlanner(
		crossPlatformPolicySelector{optionalM2SFEvidence: false},
	)
	if err != nil {
		t.Fatalf("newCrossPlatformGovernancePlanner: %v", err)
	}
	registry.SetCapabilityGovernancePlanner(governancePlanner)

	plan, err := blackboard.NewPlanner(db).BuildWorkspacePlan(
		context.Background(),
		workspace.ID,
		blackboard.PlanningConfig{
			Now: time.Date(2026, 9, 22, 11, 15, 0, 0, time.UTC),
			ActionSteps: []blackboard.ToolSequenceStep{{
				ToolName: string(flowinterop.OperationExport),
				Reason:   "Export only after collaboration is ready.",
				Params:   w6FlowRequest(flowinterop.OperationExport),
			}},
		},
	)
	if err != nil {
		t.Fatalf("BuildWorkspacePlan: %v", err)
	}
	if plan.State != blackboard.PlanningStateAwaitingEvidence {
		t.Fatalf("plan state = %q, want awaiting_evidence", plan.State)
	}

	traceID := "trace-w6-b-not-ready"
	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.TraceID, traceID)
	ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, "agent-w6")
	executor := blackboard.NewPlannerExecutor(db, nil, nil, registry, nil)
	outcome, err := executor.Execute(ctx, workspace, plan)
	if err != nil {
		t.Fatalf("PlannerExecutor.Execute: %v", err)
	}
	if outcome.DeferralReason != "awaiting_evidence" || len(outcome.Executed) != 0 {
		t.Fatalf("deferred outcome = %#v", outcome)
	}
	providerMu.Lock()
	calls := providerCalls
	providerMu.Unlock()
	if calls != 0 {
		t.Fatalf("provider calls = %d, want 0 while collaboration is not ready", calls)
	}
}

func TestW6C_M2SFUnavailableFailsExplicitlyWithoutInventedResult(t *testing.T) {
	db, workspace := setupW6FunctionalWorkspace(t)

	var mu sync.Mutex
	calls := make([]w6ProviderCall, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var semanticRequest flowinterop.Request
		if err := json.NewDecoder(request.Body).Decode(&semanticRequest); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		calls = append(calls, w6ProviderCall{
			traceID:     request.Header.Get("X-Fenix-Trace-ID"),
			executionID: request.Header.Get("X-Fenix-Execution-ID"),
			request:     semanticRequest,
		})
		mu.Unlock()
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	registry := tool.NewToolRegistry(db)
	settings := crossPlatformRuntimeSettings{
		M2SF: providerRuntimeSettings{
			BaseURL: server.URL,
			Token:   "test-token",
			Enabled: true,
		},
		Timeout: 2 * time.Second,
	}
	if err := configureCrossPlatformRuntime(registry, settings); err != nil {
		t.Fatalf("configureCrossPlatformRuntime: %v", err)
	}
	governancePlanner, err := newCrossPlatformGovernancePlanner(
		crossPlatformPolicySelector{optionalM2SFEvidence: false},
	)
	if err != nil {
		t.Fatalf("newCrossPlatformGovernancePlanner: %v", err)
	}
	registry.SetCapabilityGovernancePlanner(governancePlanner)

	traceID := "trace-w6-c-m2sf-down"
	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.TraceID, traceID)
	ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, "agent-w6")
	executor := blackboard.NewPlannerExecutor(db, nil, nil, registry, nil)

	outcome, err := executor.Execute(ctx, workspace, w6ExportPlan(workspace.ID))
	if err != nil {
		t.Fatalf("PlannerExecutor.Execute: %v", err)
	}
	if outcome.DeferralReason != "tool_failure" || len(outcome.Executed) != 1 {
		t.Fatalf("outcome = %#v", outcome)
	}
	if len(outcome.Executed[0].Result) != 0 {
		t.Fatalf("M2SF failure produced invented result: %s", outcome.Executed[0].Result)
	}
	if !strings.Contains(outcome.Executed[0].Error, string(tool.ToolErrorCapabilityFailed)) {
		t.Fatalf("tool error = %q", outcome.Executed[0].Error)
	}

	mu.Lock()
	recorded := append([]w6ProviderCall(nil), calls...)
	mu.Unlock()
	if len(recorded) != 2 {
		t.Fatalf("M2SF calls = %d, want 2 bounded attempts", len(recorded))
	}
	if recorded[0].executionID == "" || recorded[1].executionID != recorded[0].executionID {
		t.Fatalf("execution identity across failed retries = %#v", recorded)
	}
	if recorded[0].traceID != traceID || recorded[1].traceID != traceID {
		t.Fatalf("trace identity across failed retries = %#v", recorded)
	}
}

func TestW6C_OptionalVELUnavailablePreservesBusinessResultAsIndeterminateEvidence(t *testing.T) {
	recorder := &w6ResilienceEvidenceRecorder{
		result: tool.CapabilityEvidenceResult{State: "indeterminate"},
		err:    errors.New("evidence provider unavailable"),
	}
	outcome, providerCalls, projection := executeW6CExportWithEvidenceRecorder(
		t,
		"trace-w6-c-vel-down",
		recorder,
	)

	if providerCalls != 1 {
		t.Fatalf("provider calls = %d, want 1", providerCalls)
	}
	if len(recorder.requests) != 1 {
		t.Fatalf("evidence calls = %d, want 1", len(recorder.requests))
	}
	if outcome.DeferralReason != "tool_failure" || len(outcome.Executed) != 1 {
		t.Fatalf("outcome = %#v", outcome)
	}
	step := outcome.Executed[0]
	if len(step.Result) == 0 {
		t.Fatal("business result was lost when optional evidence became unavailable")
	}
	if !strings.Contains(step.Error, string(tool.ToolErrorEvidenceIndeterminate)) {
		t.Fatalf("tool error = %q", step.Error)
	}

	var result flowinterop.Result
	if err := json.Unmarshal(step.Result, &result); err != nil {
		t.Fatalf("decode preserved provider result: %v", err)
	}
	if result.Status != flowinterop.ResultSucceeded || result.ProviderReference != "m2sf-w6-c" {
		t.Fatalf("preserved provider result = %#v", result)
	}
	assertW6CAgentSafeFailureProjection(
		t,
		projection,
		string(tool.ToolErrorEvidenceIndeterminate),
	)
}

func TestW6C_VerificationFailureCannotBePresentedAsVerified(t *testing.T) {
	recorder := &w6ResilienceEvidenceRecorder{
		result: tool.CapabilityEvidenceResult{
			State: "verification_failed",
			AuditMetadata: map[string]any{
				"verification_status": "failed",
				"issue_count":         1,
			},
		},
	}
	outcome, providerCalls, projection := executeW6CExportWithEvidenceRecorder(
		t,
		"trace-w6-c-verification-failed",
		recorder,
	)

	if providerCalls != 1 || len(recorder.requests) != 1 {
		t.Fatalf("provider=%d evidence=%d, want 1/1", providerCalls, len(recorder.requests))
	}
	if outcome.DeferralReason != "tool_failure" || len(outcome.Executed) != 1 {
		t.Fatalf("outcome = %#v", outcome)
	}
	step := outcome.Executed[0]
	if len(step.Result) == 0 {
		t.Fatal("business result was lost after evidence verification failure")
	}
	if !strings.Contains(step.Error, string(tool.ToolErrorEvidenceVerificationFailed)) {
		t.Fatalf("tool error = %q", step.Error)
	}
	assertW6CAgentSafeFailureProjection(
		t,
		projection,
		string(tool.ToolErrorEvidenceVerificationFailed),
	)
}

func executeW6CExportWithEvidenceRecorder(
	t *testing.T,
	traceID string,
	recorder tool.CapabilityEvidenceRecorder,
) (*blackboard.ExecutionOutcome, int, string) {
	t.Helper()

	db, workspace := setupW6FunctionalWorkspace(t)
	var mu sync.Mutex
	providerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		mu.Lock()
		providerCalls++
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(flowinterop.Result{
			ContractVersion: flowinterop.ContractVersion,
			Operation:       flowinterop.OperationExport,
			Status:          flowinterop.ResultSucceeded,
			Artifacts: []flowinterop.Artifact{{
				Format:  flowinterop.FormatSalesforceFlowXML,
				Name:    "Demo",
				Content: "<Flow />",
			}},
			Fidelity: flowinterop.FidelityReport{
				ContractVersion: flowinterop.ContractVersion,
				Level:           flowinterop.FidelityGuaranteed,
				FlowFamily:      flowinterop.FlowFamilyAutolaunched,
			},
			SemanticMetadata: flowinterop.SemanticMetadata{
				FlowFamily:  flowinterop.FlowFamilyAutolaunched,
				FlowAPIName: "Demo",
			},
			ProviderReference: "m2sf-w6-c",
		})
	}))
	defer server.Close()

	registry := tool.NewToolRegistry(db)
	settings := crossPlatformRuntimeSettings{
		M2SF: providerRuntimeSettings{
			BaseURL: server.URL,
			Token:   "test-token",
			Enabled: true,
		},
		Timeout:              2 * time.Second,
		M2SFOptionalEvidence: true,
	}
	if err := configureCrossPlatformRuntime(registry, settings); err != nil {
		t.Fatalf("configureCrossPlatformRuntime: %v", err)
	}
	governancePlanner, err := newCrossPlatformGovernancePlanner(
		crossPlatformPolicySelector{optionalM2SFEvidence: true},
	)
	if err != nil {
		t.Fatalf("newCrossPlatformGovernancePlanner: %v", err)
	}
	registry.SetCapabilityGovernancePlanner(governancePlanner)
	registry.SetCapabilityEvidenceRecorder(recorder)

	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.TraceID, traceID)
	ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, "agent-w6")
	ctx = ctxkeys.WithValue(ctx, ctxkeys.RunID, "run-w6-c")
	executor := blackboard.NewPlannerExecutor(db, nil, nil, registry, nil)
	outcome, err := executor.Execute(ctx, workspace, w6ExportPlan(workspace.ID))
	if err != nil {
		t.Fatalf("PlannerExecutor.Execute: %v", err)
	}

	entry, err := blackboard.NewMemoryStore(db).Get(
		context.Background(),
		workspace.ID,
		blackboard.DefaultPlannerExecutionResultMemoryKey,
	)
	if err != nil {
		t.Fatalf("load persisted Blackboard failure projection: %v", err)
	}

	mu.Lock()
	calls := providerCalls
	mu.Unlock()
	return outcome, calls, string(entry.Value)
}

func assertW6CAgentSafeFailureProjection(
	t *testing.T,
	projection string,
	expectedErrorCode string,
) {
	t.Helper()

	if !strings.Contains(projection, "m2sf-w6-c") {
		t.Fatalf("Blackboard failure projection lost provider result: %s", projection)
	}
	if !strings.Contains(projection, expectedErrorCode) {
		t.Fatalf(
			"Blackboard failure projection missing %q: %s",
			expectedErrorCode,
			projection,
		)
	}
	for _, forbidden := range []string{
		"verification_bundle",
		"portable_bundle",
		"event_hash",
		"merkle_root",
		"signature_ref",
		"reasoning_trace",
	} {
		if strings.Contains(projection, forbidden) {
			t.Fatalf("Blackboard failure projection leaked %q: %s", forbidden, projection)
		}
	}
}

func TestW6C_RestartReconcilesEvidenceWithoutReplayingM2SF(t *testing.T) {
	db, workspace := setupW6FunctionalWorkspace(t)

	var providerMu sync.Mutex
	providerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		providerMu.Lock()
		providerCalls++
		providerMu.Unlock()
		_ = json.NewEncoder(w).Encode(flowinterop.Result{
			ContractVersion: flowinterop.ContractVersion,
			Operation:       flowinterop.OperationExport,
			Status:          flowinterop.ResultSucceeded,
			Artifacts: []flowinterop.Artifact{{
				Format:  flowinterop.FormatSalesforceFlowXML,
				Name:    "Demo",
				Content: "<Flow />",
			}},
			Fidelity: flowinterop.FidelityReport{
				ContractVersion: flowinterop.ContractVersion,
				Level:           flowinterop.FidelityGuaranteed,
				FlowFamily:      flowinterop.FlowFamilyAutolaunched,
			},
			SemanticMetadata: flowinterop.SemanticMetadata{
				FlowFamily:  flowinterop.FlowFamilyAutolaunched,
				FlowAPIName: "Demo",
			},
			ProviderReference: "m2sf-w6-c-restart",
		})
	}))
	defer server.Close()

	firstSink := &w6RestartLifecycleSink{
		recordErr: evidence.NewIndeterminateRecordError(errors.New("lost append response")),
	}
	firstRecorder, err := evidenceoutbox.NewRecorder(db, firstSink)
	if err != nil {
		t.Fatalf("create first durable recorder: %v", err)
	}

	registry := tool.NewToolRegistry(db)
	settings := crossPlatformRuntimeSettings{
		M2SF: providerRuntimeSettings{
			BaseURL: server.URL,
			Token:   "test-token",
			Enabled: true,
		},
		Timeout:              2 * time.Second,
		M2SFOptionalEvidence: true,
	}
	if err := configureCrossPlatformRuntime(registry, settings); err != nil {
		t.Fatalf("configureCrossPlatformRuntime: %v", err)
	}
	governancePlanner, err := newCrossPlatformGovernancePlanner(
		crossPlatformPolicySelector{optionalM2SFEvidence: true},
	)
	if err != nil {
		t.Fatalf("newCrossPlatformGovernancePlanner: %v", err)
	}
	registry.SetCapabilityGovernancePlanner(governancePlanner)
	registry.SetCapabilityEvidenceRecorder(firstRecorder)

	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.TraceID, "trace-w6-c-restart")
	ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, "agent-w6")
	ctx = ctxkeys.WithValue(ctx, ctxkeys.RunID, "run-w6-c-restart")
	executor := blackboard.NewPlannerExecutor(db, nil, nil, registry, nil)
	outcome, err := executor.Execute(ctx, workspace, w6ExportPlan(workspace.ID))
	if err != nil {
		t.Fatalf("PlannerExecutor.Execute: %v", err)
	}
	if outcome.DeferralReason != "" || len(outcome.Executed) != 1 ||
		len(outcome.Executed[0].Result) == 0 || outcome.Executed[0].Error != "" {
		t.Fatalf("initial business outcome = %#v", outcome)
	}
	if firstSink.recordCalls != 1 || firstSink.lookupCalls != 1 ||
		len(firstSink.envelopes) != 1 {
		t.Fatalf(
			"initial evidence lifecycle record=%d lookup=%d envelopes=%d",
			firstSink.recordCalls,
			firstSink.lookupCalls,
			len(firstSink.envelopes),
		)
	}

	providerMu.Lock()
	initialProviderCalls := providerCalls
	providerMu.Unlock()
	if initialProviderCalls != 1 {
		t.Fatalf("initial M2SF calls = %d, want 1", initialProviderCalls)
	}

	envelope := firstSink.envelopes[0]
	if _, err := db.Exec(
		"UPDATE evidence_delivery SET next_attempt_at_ms = 0 WHERE execution_id = ?",
		envelope.ExecutionID,
	); err != nil {
		t.Fatalf("make durable evidence due for restart: %v", err)
	}

	ref := evidence.ProofReference{
		SchemaVersion:      evidence.SchemaVersion,
		Provider:           "vel-http",
		ExecutionID:        envelope.ExecutionID,
		StreamID:           envelope.StreamID,
		EventID:            "event-w6-c-restart",
		EventHash:          strings.Repeat("a", 64),
		KeyID:              "key-w6-c",
		SignatureRef:       "vel://events/event-w6-c-restart#signature",
		Sequence:           1,
		VerificationStatus: evidence.VerificationRecorded,
	}
	secondSink := &w6RestartLifecycleSink{
		lookupRef: &ref,
		progress: evidence.VerificationProgress{
			Checkpoint: evidence.CheckpointReference{
				CheckpointID:   "checkpoint-w6-c-restart",
				CheckpointHash: strings.Repeat("b", 64),
				MerkleRoot:     strings.Repeat("c", 64),
				TreeSize:       1,
			},
			Status: evidence.VerificationVerified,
		},
	}
	secondRecorder, err := evidenceoutbox.NewRecorder(db, secondSink)
	if err != nil {
		t.Fatalf("create restarted durable recorder: %v", err)
	}
	if err := secondRecorder.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("restart ReconcileOnce: %v", err)
	}

	if secondSink.recordCalls != 0 {
		t.Fatalf("restart replayed VEL append %d times", secondSink.recordCalls)
	}
	if secondSink.lookupCalls != 1 || secondSink.verificationCalls != 1 {
		t.Fatalf(
			"restart lifecycle lookup=%d verification=%d, want 1/1",
			secondSink.lookupCalls,
			secondSink.verificationCalls,
		)
	}

	providerMu.Lock()
	finalProviderCalls := providerCalls
	providerMu.Unlock()
	if finalProviderCalls != 1 {
		t.Fatalf(
			"evidence reconciliation replayed M2SF: calls=%d, want 1",
			finalProviderCalls,
		)
	}

	var deliveryState string
	var proofRaw string
	if err := db.QueryRow(
		"SELECT delivery_state, proof_json FROM evidence_delivery WHERE execution_id = ?",
		envelope.ExecutionID,
	).Scan(&deliveryState, &proofRaw); err != nil {
		t.Fatalf("load reconciled durable evidence: %v", err)
	}
	if deliveryState != string(evidence.DeliveryVerified) {
		t.Fatalf("delivery state = %q, want verified", deliveryState)
	}
	if !strings.Contains(proofRaw, "\"verification_status\":\"verified\"") {
		t.Fatalf("proof was not verified after restart: %s", proofRaw)
	}

	entry, err := blackboard.NewMemoryStore(db).Get(
		context.Background(),
		workspace.ID,
		blackboard.DefaultPlannerExecutionResultMemoryKey,
	)
	if err != nil {
		t.Fatalf("load Blackboard business result after evidence restart: %v", err)
	}
	projection := string(entry.Value)
	if !strings.Contains(projection, "m2sf-w6-c-restart") {
		t.Fatalf("Blackboard lost business result after evidence recovery: %s", projection)
	}
	for _, forbidden := range []string{
		"verification_bundle",
		"portable_bundle",
		"merkle_root",
		"signature_ref",
	} {
		if strings.Contains(projection, forbidden) {
			t.Fatalf("Blackboard absorbed evidence-provider internals %q: %s", forbidden, projection)
		}
	}
}
