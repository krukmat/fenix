package agent

import (
	"context"
	"database/sql"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
	signaldomain "github.com/matiasleandrokruk/fenix/internal/domain/signal"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

// RunContext carries shared runtime dependencies for any agent runner.
//
// The context is passed per execution instead of being captured in a runner
// constructor so current Go agents and future declarative runners can share the
// same execution contract.
type RunContext struct {
	Orchestrator    *Orchestrator
	ToolRegistry    *tool.ToolRegistry
	PolicyEngine    *policy.PolicyEngine
	ApprovalService *policy.ApprovalService
	Scheduler       schedulerdomain.Scheduler
	SignalService   *signaldomain.Service
	AuditService    *audit.AuditService
	EventBus        eventbus.EventBus
	RunnerRegistry  *RunnerRegistry
	DB              *sql.DB

	// Call metadata is used by future nested executions and delegation flow.
	CallDepth int
	CallChain []string
}

// Runner is the common execution contract for any runnable agent.
//
// Concrete Go agents are adapted to this interface in F1.5. Future declarative
// runners, such as DSLRunner, can implement the same contract without changing
// orchestrator-facing execution semantics.
type Runner interface {
	Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error)
}

// Clone returns a shallow copy of the run context with an independent call
// chain slice so nested runners can extend it safely.
func (rc *RunContext) Clone() *RunContext {
	if rc == nil {
		return &RunContext{}
	}

	clone := *rc
	if len(rc.CallChain) > 0 {
		clone.CallChain = append([]string(nil), rc.CallChain...)
	}
	return &clone
}

// WithCall returns a cloned runtime context extended with one additional call
// record. It is forward-compatible with internal delegation and loop detection.
func (rc *RunContext) WithCall(agentID string) *RunContext {
	next := rc.Clone()
	next.CallDepth++
	if agentID != "" {
		next.CallChain = append(next.CallChain, agentID)
	}
	return next
}
