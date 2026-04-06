package agent

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

type stubRunner struct{}

func (stubRunner) Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error) {
	_ = ctx
	_ = rc
	return &Run{
		WorkspaceID:  input.WorkspaceID,
		DefinitionID: input.AgentID,
	}, nil
}

var _ Runner = stubRunner{}

func TestAgentRunnerContract(t *testing.T) {
	runner := stubRunner{}

	run, err := runner.Run(context.Background(), &RunContext{}, TriggerAgentInput{
		AgentID:     "support-agent",
		WorkspaceID: "ws-1",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.DefinitionID != "support-agent" {
		t.Fatalf("DefinitionID = %q, want %q", run.DefinitionID, "support-agent")
	}
	if run.WorkspaceID != "ws-1" {
		t.Fatalf("WorkspaceID = %q, want %q", run.WorkspaceID, "ws-1")
	}
}

func TestRunContextClonePreservesDepsAndCopiesChain(t *testing.T) {
	bus := eventbus.New()
	db := &sql.DB{}
	orch := &Orchestrator{db: db}
	policyEngine := &policy.PolicyEngine{}
	approvalService := &policy.ApprovalService{}
	auditService := &audit.AuditService{}
	groundsValidator := &GroundsValidator{}

	original := &RunContext{
		Orchestrator:     orch,
		PolicyEngine:     policyEngine,
		ApprovalService:  approvalService,
		AuditService:     auditService,
		EventBus:         bus,
		RunnerRegistry:   NewRunnerRegistry(),
		GroundsValidator: groundsValidator,
		DB:               db,
		CallDepth:        1,
		CallChain:        []string{"support-agent"},
	}

	clone := original.Clone()
	if clone == original {
		t.Fatal("Clone() returned same pointer")
	}
	if clone.Orchestrator != orch || clone.EventBus != bus || clone.DB != db {
		t.Fatal("Clone() did not preserve runtime dependencies")
	}
	if clone.RunnerRegistry != original.RunnerRegistry {
		t.Fatal("Clone() did not preserve runner registry")
	}
	if clone.GroundsValidator != groundsValidator {
		t.Fatal("Clone() did not preserve grounds validator")
	}

	clone.CallChain[0] = "changed"
	if original.CallChain[0] != "support-agent" {
		t.Fatal("Clone() did not copy call chain")
	}
}

func TestRunContextCloneNilReceiverReturnsEmpty(t *testing.T) {
	var rc *RunContext
	clone := rc.Clone()
	if clone == nil {
		t.Fatal("Clone() on nil rc should return non-nil RunContext")
	}
}

func TestRunContextWithCallExtendsChain(t *testing.T) {
	original := &RunContext{
		CallDepth: 1,
		CallChain: []string{"support-agent"},
	}

	next := original.WithCall("kb-agent")
	if next.CallDepth != 2 {
		t.Fatalf("CallDepth = %d, want 2", next.CallDepth)
	}
	if len(next.CallChain) != 2 || next.CallChain[1] != "kb-agent" {
		t.Fatalf("CallChain = %#v, want appended agent", next.CallChain)
	}
	if len(original.CallChain) != 1 {
		t.Fatalf("original CallChain mutated: %#v", original.CallChain)
	}
}
