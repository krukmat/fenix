package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

func TestCartaIntegration(t *testing.T) {
	t.Run("scenario A success", func(t *testing.T) {
		t.Parallel()

		db := setupDSLRunnerDB(t)
		mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('agent_carta_success', 'ws_dsl', 'dsl success', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
		mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, spec_source, version, status, created_at, updated_at)
		VALUES ('wf_carta_success', 'ws_dsl', 'agent_carta_success', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"', 'CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 1
    min_confidence: medium', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

		executor := &stubRuntimeExecutor{output: map[string]any{"status": "updated"}}
		runner := NewDSLRunnerWithDependencies(workflowdomain.NewService(db), NewDSLRuntime(), executor)
		orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
		groundsValidator := NewGroundsValidator(stubGroundsEvidenceBuilder{
			pack: &knowledge.EvidencePack{
				Sources:    []knowledge.Evidence{{ID: "ev-1", CreatedAt: time.Now()}},
				Confidence: knowledge.ConfidenceHigh,
			},
		})

		run, err := runner.Run(context.Background(), &RunContext{
			Orchestrator:     orch,
			GroundsValidator: groundsValidator,
			DB:               db,
		}, TriggerAgentInput{
			AgentID:        "agent_carta_success",
			WorkspaceID:    "ws_dsl",
			TriggerType:    TriggerTypeEvent,
			TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if run.Status != StatusSuccess {
			t.Fatalf("status = %s, want %s", run.Status, StatusSuccess)
		}
		if len(executor.ops) != 1 {
			t.Fatalf("executed ops = %d, want 1", len(executor.ops))
		}
	})

	t.Run("scenario D abstained and DSL runtime not called", func(t *testing.T) {
		t.Parallel()

		db := setupDSLRunnerDB(t)
		mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('agent_carta_abstain', 'ws_dsl', 'dsl abstain', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
		mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, spec_source, version, status, created_at, updated_at)
		VALUES ('wf_carta_abstain', 'ws_dsl', 'agent_carta_abstain', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"', 'CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
    min_confidence: medium', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

		executor := &stubRuntimeExecutor{output: map[string]any{"status": "updated"}}
		runner := NewDSLRunnerWithDependencies(workflowdomain.NewService(db), NewDSLRuntime(), executor)
		orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
		groundsValidator := NewGroundsValidator(stubGroundsEvidenceBuilder{
			pack: &knowledge.EvidencePack{
				Sources:    []knowledge.Evidence{{ID: "ev-1", CreatedAt: time.Now()}},
				Confidence: knowledge.ConfidenceLow,
			},
		})

		run, err := runner.Run(context.Background(), &RunContext{
			Orchestrator:     orch,
			GroundsValidator: groundsValidator,
			DB:               db,
		}, TriggerAgentInput{
			AgentID:        "agent_carta_abstain",
			WorkspaceID:    "ws_dsl",
			TriggerType:    TriggerTypeEvent,
			TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if run.Status != StatusAbstained {
			t.Fatalf("status = %s, want %s", run.Status, StatusAbstained)
		}
		if len(executor.ops) != 0 {
			t.Fatalf("executed ops = %d, want 0", len(executor.ops))
		}
	})

	t.Run("scenario E delegated with zero token usage", func(t *testing.T) {
		t.Parallel()

		db := setupDSLRunnerDB(t)
		mustExecDSLRunner(t, db, `INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES ('user_delegate', 'ws_dsl', 'delegate@example.com', 'Delegate Owner', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
		mustExecDSLRunner(t, db, `INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at)
		VALUES ('case-1', 'ws_dsl', 'user_delegate', 'Delegate Case', 'high', 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
		mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('agent_carta_delegate', 'ws_dsl', 'dsl delegate', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
		mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, spec_source, version, status, created_at, updated_at)
		VALUES ('wf_carta_delegate', 'ws_dsl', 'agent_carta_delegate', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"', 'CARTA resolve_support_case
AGENT search_knowledge
  DELEGATE TO HUMAN
    when: case.tier == "enterprise"
    reason: "Enterprise review required"
    package: [evidence_ids, case_summary]', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

		executor := &stubRuntimeExecutor{output: map[string]any{"status": "updated"}}
		runner := NewDSLRunnerWithDependencies(workflowdomain.NewService(db), NewDSLRuntime(), executor)
		orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())

		run, err := runner.Run(context.Background(), &RunContext{
			Orchestrator: orch,
			DB:           db,
		}, TriggerAgentInput{
			AgentID:        "agent_carta_delegate",
			WorkspaceID:    "ws_dsl",
			TriggerType:    TriggerTypeEvent,
			TriggerContext: json.RawMessage(`{"case":{"id":"case-1","tier":"enterprise"}}`),
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if run.Status != StatusDelegated {
			t.Fatalf("status = %s, want %s", run.Status, StatusDelegated)
		}
		if len(executor.ops) != 0 {
			t.Fatalf("executed ops = %d, want 0", len(executor.ops))
		}
		if run.TotalTokens != nil && *run.TotalTokens != 0 {
			t.Fatalf("TotalTokens = %v, want nil or 0", run.TotalTokens)
		}
	})
}
